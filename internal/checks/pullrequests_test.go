package checks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/backbiten/jitterbugs/internal/core"
	"github.com/backbiten/jitterbugs/internal/github"
)

// mockPRClient implements PRClient for testing.
type mockPRClient struct {
	prs     []github.PullRequest
	reviews map[int][]github.Review
	listErr error
}

func (m *mockPRClient) ListOpenPRs(_ context.Context, _, _ string) ([]github.PullRequest, error) {
	return m.prs, m.listErr
}

func (m *mockPRClient) ListPRReviews(_ context.Context, _, _ string, prNum int) ([]github.Review, error) {
	return m.reviews[prNum], nil
}

// helper – a PR updated n days ago.
func prUpdatedDaysAgo(num int, title, body string, days int, draft bool) github.PullRequest {
	return github.PullRequest{
		Number:    num,
		Title:     title,
		Body:      body,
		Draft:     draft,
		UpdatedAt: time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour),
	}
}

func TestPRCheck_NoPRs(t *testing.T) {
	c := NewPullRequestsCheck(&mockPRClient{}, "owner", "repo")
	r := c.Run("ignored")
	if r.Status != core.SeverityPass {
		t.Errorf("expected pass with no open PRs, got %s", r.Status)
	}
}

func TestPRCheck_HealthyPR(t *testing.T) {
	pr := prUpdatedDaysAgo(1, "feat: thing", "This PR does something useful.", 0, false)
	reviews := map[int][]github.Review{1: {{State: "APPROVED"}}}
	c := NewPullRequestsCheck(&mockPRClient{prs: []github.PullRequest{pr}, reviews: reviews}, "owner", "repo")
	r := c.Run("ignored")
	if r.Status != core.SeverityPass {
		t.Errorf("expected pass for healthy PR, got %s: %s", r.Status, r.Message)
	}
}

func TestPRCheck_NoReview(t *testing.T) {
	pr := prUpdatedDaysAgo(1, "feat: thing", "Has a description.", 0, false)
	c := NewPullRequestsCheck(&mockPRClient{prs: []github.PullRequest{pr}, reviews: map[int][]github.Review{}}, "owner", "repo")
	r := c.Run("ignored")
	if r.Status != core.SeverityWarning {
		t.Errorf("expected warning when PR has no reviews, got %s", r.Status)
	}
	if !containsPattern(r.Findings, "no-review") {
		t.Error("expected finding with pattern 'no-review'")
	}
}

func TestPRCheck_MissingDescription(t *testing.T) {
	pr := prUpdatedDaysAgo(2, "fix: bug", "", 0, false)
	reviews := map[int][]github.Review{2: {{State: "APPROVED"}}}
	c := NewPullRequestsCheck(&mockPRClient{prs: []github.PullRequest{pr}, reviews: reviews}, "owner", "repo")
	r := c.Run("ignored")
	if r.Status != core.SeverityWarning {
		t.Errorf("expected warning for missing description, got %s", r.Status)
	}
	if !containsPattern(r.Findings, "missing-description") {
		t.Error("expected finding with pattern 'missing-description'")
	}
}

func TestPRCheck_StalePR(t *testing.T) {
	pr := prUpdatedDaysAgo(3, "chore: old stuff", "Some description.", stalePRDays+1, false)
	reviews := map[int][]github.Review{3: {{State: "CHANGES_REQUESTED"}}}
	c := NewPullRequestsCheck(&mockPRClient{prs: []github.PullRequest{pr}, reviews: reviews}, "owner", "repo")
	r := c.Run("ignored")
	if r.Status != core.SeverityWarning {
		t.Errorf("expected warning for stale PR, got %s", r.Status)
	}
	if !containsPattern(r.Findings, "stale-pr") {
		t.Error("expected finding with pattern 'stale-pr'")
	}
}

func TestPRCheck_ExpiredPR(t *testing.T) {
	pr := prUpdatedDaysAgo(4, "wip: ancient", "Old PR.", expiredPRDays+1, false)
	c := NewPullRequestsCheck(&mockPRClient{prs: []github.PullRequest{pr}}, "owner", "repo")
	r := c.Run("ignored")
	if r.Status != core.SeverityFail {
		t.Errorf("expected fail for expired PR, got %s", r.Status)
	}
	if !containsPattern(r.Findings, "expired-pr") {
		t.Error("expected finding with pattern 'expired-pr'")
	}
}

func TestPRCheck_DraftPRSkipsReviewCheck(t *testing.T) {
	// Draft PRs should not be flagged for missing reviews.
	pr := prUpdatedDaysAgo(5, "wip: draft", "Draft PR.", 0, true)
	c := NewPullRequestsCheck(&mockPRClient{prs: []github.PullRequest{pr}, reviews: map[int][]github.Review{}}, "owner", "repo")
	r := c.Run("ignored")
	// May still warn for other reasons, but not for no-review.
	if containsPattern(r.Findings, "no-review") {
		t.Error("draft PRs should not be flagged for missing reviews")
	}
}

func TestPRCheck_APIError(t *testing.T) {
	c := NewPullRequestsCheck(&mockPRClient{listErr: errors.New("network failure")}, "owner", "repo")
	r := c.Run("ignored")
	if r.Status != core.SeverityWarning {
		t.Errorf("expected warning on API error, got %s", r.Status)
	}
}

// containsPattern returns true if any finding has the given pattern.
func containsPattern(findings []core.Finding, pattern string) bool {
	for _, f := range findings {
		if f.Pattern == pattern {
			return true
		}
	}
	return false
}
