// Package github provides a thin GitHub REST API client with no external
// dependencies.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.github.com"

// Client is a minimal GitHub REST API client.
type Client struct {
	http    *http.Client
	token   string
	BaseURL string // exported so tests can point at an httptest server
}

// NewClient creates a GitHub API client.
// Provide a personal-access token for authenticated requests (5,000 req/hr);
// pass an empty string for unauthenticated access (60 req/hr).
func NewClient(token string) *Client {
	return &Client{
		http:    &http.Client{Timeout: 15 * time.Second},
		token:   token,
		BaseURL: defaultBaseURL,
	}
}

// Repo represents a single GitHub repository list entry.
type Repo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Archived bool `json:"archived"`
}

// PullRequest represents an open pull request.
type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Draft     bool      `json:"draft"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

// Review represents a single pull-request review.
type Review struct {
	State string `json:"state"`
	User  struct {
		Login string `json:"login"`
	} `json:"user"`
}

// CommunityProfile summarises the community health files of a repository.
type CommunityProfile struct {
	Files struct {
		Readme         *struct{ URL string `json:"url"` }  `json:"readme"`
		License        *struct{ Key string `json:"key"` }  `json:"license"`
		Contributing   *struct{ URL string `json:"url"` }  `json:"contributing"`
		SecurityPolicy *struct{ URL string `json:"url"` }  `json:"security"`
	} `json:"files"`
}

// ListOrgRepos returns all repositories belonging to org (paginates automatically).
func (c *Client) ListOrgRepos(ctx context.Context, org string) ([]Repo, error) {
	return c.listRepos(ctx, fmt.Sprintf("%s/orgs/%s/repos", c.BaseURL, org))
}

// ListUserRepos returns all repositories belonging to user (paginates automatically).
func (c *Client) ListUserRepos(ctx context.Context, user string) ([]Repo, error) {
	return c.listRepos(ctx, fmt.Sprintf("%s/users/%s/repos", c.BaseURL, user))
}

// ListOpenPRs returns the first page (up to 100) of open pull requests for
// owner/repo.
func (c *Client) ListOpenPRs(ctx context.Context, owner, repo string) ([]PullRequest, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls?state=open&per_page=100", c.BaseURL, owner, repo)
	var prs []PullRequest
	if err := c.getJSON(ctx, url, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

// ListPRReviews returns all reviews for the given pull request number.
func (c *Client) ListPRReviews(ctx context.Context, owner, repo string, prNum int) ([]Review, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews", c.BaseURL, owner, repo, prNum)
	var reviews []Review
	if err := c.getJSON(ctx, url, &reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}

// GetCommunityProfile returns the community health profile for owner/repo.
func (c *Client) GetCommunityProfile(ctx context.Context, owner, repo string) (*CommunityProfile, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/community/profile", c.BaseURL, owner, repo)
	var profile CommunityProfile
	if err := c.getJSON(ctx, url, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// HasWorkflows reports whether .github/workflows/ contains at least one
// .yml or .yaml file, querying the GitHub Contents API.
func (c *Client) HasWorkflows(ctx context.Context, owner, repo string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/.github/workflows", c.BaseURL, owner, repo)
	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := c.getJSON(ctx, url, &entries); err != nil {
		// A 404 means the directory does not exist – that is not a client error.
		return false, nil
	}
	for _, e := range entries {
		if e.Type == "file" &&
			(strings.HasSuffix(e.Name, ".yml") || strings.HasSuffix(e.Name, ".yaml")) {
			return true, nil
		}
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// internal helpers

// listRepos paginates through all pages of a repos list endpoint.
func (c *Client) listRepos(ctx context.Context, baseURL string) ([]Repo, error) {
	var all []Repo
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s?per_page=100&page=%d", baseURL, page)
		var repos []Repo
		if err := c.getJSON(ctx, url, &repos); err != nil {
			return nil, err
		}
		all = append(all, repos...)
		if len(repos) < 100 {
			break
		}
	}
	return all, nil
}

// getJSON performs an authenticated GET and decodes the JSON body into dst.
func (c *Client) getJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		msg := string(body)
		if readErr != nil {
			msg = fmt.Sprintf("(could not read body: %v)", readErr)
		}
		return fmt.Errorf("GitHub API %s: %s", resp.Status, msg)
	}
	if readErr != nil {
		return fmt.Errorf("reading GitHub API response: %w", readErr)
	}
	return json.Unmarshal(body, dst)
}
