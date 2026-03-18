package checks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/backbiten/jitterbugs/internal/core"
	"github.com/backbiten/jitterbugs/internal/github"
)

// stalePRDays is the number of days without an update before a PR is flagged
// as stale (warning).
const stalePRDays = 30

// expiredPRDays is the number of days without an update before a PR is flagged
// as expired (fail).
const expiredPRDays = 90

// PRClient is the subset of github.Client used by pullRequestsCheck.
// Using an interface keeps the check unit-testable without a real HTTP server.
type PRClient interface {
	ListOpenPRs(ctx context.Context, owner, repo string) ([]github.PullRequest, error)
	ListPRReviews(ctx context.Context, owner, repo string, prNum int) ([]github.Review, error)
}

type pullRequestsCheck struct {
	client PRClient
	owner  string
	repo   string
}

// NewPullRequestsCheck returns a check that audits the open pull requests of a
// GitHub repository using the GitHub API.
func NewPullRequestsCheck(client PRClient, owner, repo string) core.Check {
	return &pullRequestsCheck{client: client, owner: owner, repo: repo}
}

func (c *pullRequestsCheck) Name() string { return "pull_requests" }

// Run satisfies the core.Check interface; the repoPath argument is unused
// because the check operates against the GitHub API.
func (c *pullRequestsCheck) Run(_ string) core.CheckResult {
	result := core.CheckResult{Name: "Pull Requests"}

	ctx := context.Background()
	prs, err := c.client.ListOpenPRs(ctx, c.owner, c.repo)
	if err != nil {
		result.Status = core.SeverityWarning
		result.Message = fmt.Sprintf("Could not fetch pull requests: %v", err)
		return result
	}

	if len(prs) == 0 {
		result.Status = core.SeverityPass
		result.Message = "No open pull requests"
		return result
	}

	now := time.Now().UTC()
	var findings []core.Finding
	hasExpired := false

	for _, pr := range prs {
		ageDays := int(now.Sub(pr.UpdatedAt).Hours() / 24)

		// Expired PR (>expiredPRDays days idle) – escalates to fail.
		if ageDays > expiredPRDays {
			hasExpired = true
			findings = append(findings, core.Finding{
				File:    fmt.Sprintf("PR #%d", pr.Number),
				Pattern: "expired-pr",
				Match:   fmt.Sprintf("%q – stale for %d days", pr.Title, ageDays),
			})
			continue // already the worst case; skip further checks for this PR
		}

		// Stale PR (>stalePRDays days without update).
		if ageDays > stalePRDays {
			findings = append(findings, core.Finding{
				File:    fmt.Sprintf("PR #%d", pr.Number),
				Pattern: "stale-pr",
				Match:   fmt.Sprintf("%q – no activity for %d days", pr.Title, ageDays),
			})
		}

		// PR without a body/description.
		if strings.TrimSpace(pr.Body) == "" {
			findings = append(findings, core.Finding{
				File:    fmt.Sprintf("PR #%d", pr.Number),
				Pattern: "missing-description",
				Match:   fmt.Sprintf("%q – no description provided", pr.Title),
			})
		}

		// Non-draft PR without any review.
		if !pr.Draft {
			reviews, reviewErr := c.client.ListPRReviews(ctx, c.owner, c.repo, pr.Number)
			if reviewErr == nil && len(reviews) == 0 {
				findings = append(findings, core.Finding{
					File:    fmt.Sprintf("PR #%d", pr.Number),
					Pattern: "no-review",
					Match:   fmt.Sprintf("%q – no reviews submitted", pr.Title),
				})
			}
		}
	}

	result.Findings = findings

	switch {
	case hasExpired:
		result.Status = core.SeverityFail
		result.Message = fmt.Sprintf(
			"%d open PR(s) including expired PR(s) with no activity for more than %d days",
			len(prs), expiredPRDays,
		)
	case len(findings) > 0:
		result.Status = core.SeverityWarning
		result.Message = fmt.Sprintf("%d open PR(s) with quality issues found", len(prs))
	default:
		result.Status = core.SeverityPass
		result.Message = fmt.Sprintf("%d open PR(s) – all look healthy", len(prs))
	}

	return result
}
