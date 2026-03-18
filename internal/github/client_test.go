package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// serve registers handler under path on a test server and returns a Client
// pointed at that server.
func serve(t *testing.T, path string, body any) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("")
	c.BaseURL = srv.URL
	return c
}

func TestListOrgRepos(t *testing.T) {
	repos := []Repo{{Name: "alpha", FullName: "org/alpha"}, {Name: "beta", FullName: "org/beta"}}
	c := serve(t, "/orgs/org/repos", repos)
	got, err := c.ListOrgRepos(context.Background(), "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 repos, got %d", len(got))
	}
	if got[0].Name != "alpha" {
		t.Errorf("want name 'alpha', got %q", got[0].Name)
	}
}

func TestListUserRepos(t *testing.T) {
	repos := []Repo{{Name: "myrepo", FullName: "user/myrepo"}}
	c := serve(t, "/users/user/repos", repos)
	got, err := c.ListUserRepos(context.Background(), "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "myrepo" {
		t.Errorf("unexpected repos: %v", got)
	}
}

func TestListOpenPRs(t *testing.T) {
	prs := []PullRequest{
		{Number: 1, Title: "feat: add thing", UpdatedAt: time.Now()},
		{Number: 2, Title: "fix: a bug", UpdatedAt: time.Now()},
	}
	c := serve(t, "/repos/owner/repo/pulls", prs)
	got, err := c.ListOpenPRs(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 PRs, got %d", len(got))
	}
}

func TestListPRReviews(t *testing.T) {
	reviews := []Review{{State: "APPROVED"}}
	c := serve(t, "/repos/owner/repo/pulls/42/reviews", reviews)
	got, err := c.ListPRReviews(context.Background(), "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].State != "APPROVED" {
		t.Errorf("unexpected reviews: %v", got)
	}
}

func TestGetCommunityProfile(t *testing.T) {
	profile := CommunityProfile{}
	profile.Files.Readme = &struct {
		URL string `json:"url"`
	}{URL: "https://github.com/owner/repo/README.md"}
	c := serve(t, "/repos/owner/repo/community/profile", profile)
	got, err := c.GetCommunityProfile(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Files.Readme == nil {
		t.Error("expected Readme to be non-nil")
	}
}

func TestHasWorkflows_Found(t *testing.T) {
	entries := []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}{{Name: "ci.yml", Type: "file"}}
	c := serve(t, "/repos/owner/repo/contents/.github/workflows", entries)
	ok, err := c.HasWorkflows(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected HasWorkflows to return true")
	}
}

func TestHasWorkflows_Missing(t *testing.T) {
	// 404 from serve (path mismatch) should mean no workflows.
	c := serve(t, "/repos/owner/repo/contents/.github/workflows", nil)
	// Override handler to return 404 for all paths.
	srv := httptest.NewServer(http.NotFoundHandler())
	c.BaseURL = srv.URL
	defer srv.Close()

	ok, err := c.HasWorkflows(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected HasWorkflows to return false on 404")
	}
}

func TestGetJSON_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient("")
	c.BaseURL = srv.URL
	var result any
	err := c.getJSON(context.Background(), srv.URL+"/anything", &result)
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}
