package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/backbiten/jitterbugs/internal/checks"
	"github.com/backbiten/jitterbugs/internal/core"
	"github.com/backbiten/jitterbugs/internal/github"
	"github.com/backbiten/jitterbugs/internal/report"
)

// multiReport aggregates per-repository scan results from a GitHub audit.
type multiReport struct {
	Owner     string         `json:"owner"`
	Timestamp time.Time      `json:"timestamp"`
	Repos     []*core.Report `json:"repos"`
}

// overallStatus returns the worst status across all repo reports.
func (m *multiReport) overallStatus() core.Severity {
	status := core.SeverityPass
	for _, r := range m.Repos {
		switch r.OverallStatus {
		case core.SeverityFail:
			return core.SeverityFail
		case core.SeverityWarning:
			status = core.SeverityWarning
		}
	}
	return status
}

// exitCode returns the POSIX exit code for the multi-repo audit.
func (m *multiReport) exitCode() int {
	switch m.overallStatus() {
	case core.SeverityFail:
		return 2
	case core.SeverityWarning:
		return 1
	default:
		return 0
	}
}

const auditGitHubUsage = `audit-github – audit all GitHub repositories for an organisation or user

Usage:
  qaqc audit-github [flags]

Flags:
  --org         GitHub organisation to audit (mutually exclusive with --user)
  --user        GitHub user account to audit  (mutually exclusive with --org)
  --token       GitHub personal access token (default: $GITHUB_TOKEN)
  --html        Write an HTML report to this file in addition to JSON stdout
  --concurrency Number of repositories to scan concurrently (default: 5)

Checks run per repository:
  - Required community-health files (README, LICENSE, CONTRIBUTING, SECURITY)
  - CI/CD workflow configuration (.github/workflows/)
  - Pull-request health (stale PRs, expired PRs, missing reviews/descriptions)

Exit codes match qaqc scan: 0 = pass, 1 = warnings, 2 = fail, 3 = error.
`

func runAuditGitHub(args []string) {
	fs := flag.NewFlagSet("audit-github", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, auditGitHubUsage) }

	org := fs.String("org", "", "GitHub organisation to audit")
	user := fs.String("user", "", "GitHub user account to audit")
	token := fs.String("token", "", "GitHub personal access token (default: $GITHUB_TOKEN)")
	htmlOut := fs.String("html", "", "Optional path to write an HTML report file")
	concurrency := fs.Int("concurrency", 5, "Number of repositories to scan concurrently")
	_ = fs.Parse(args)

	if *org == "" && *user == "" {
		fmt.Fprintln(os.Stderr, "error: --org or --user is required")
		fs.Usage()
		os.Exit(1)
	}
	if *org != "" && *user != "" {
		fmt.Fprintln(os.Stderr, "error: specify only one of --org or --user")
		os.Exit(1)
	}

	tok := *token
	if tok == "" {
		tok = os.Getenv("GITHUB_TOKEN")
	}

	owner := *org
	if owner == "" {
		owner = *user
	}

	client := github.NewClient(tok)
	ctx := context.Background()

	// List all repositories for the organisation / user.
	var repos []github.Repo
	var listErr error
	if *org != "" {
		repos, listErr = client.ListOrgRepos(ctx, owner)
	} else {
		repos, listErr = client.ListUserRepos(ctx, owner)
	}
	if listErr != nil {
		fmt.Fprintf(os.Stderr, "error listing repositories for %q: %v\n", owner, listErr)
		os.Exit(3)
	}
	if len(repos) == 0 {
		fmt.Fprintf(os.Stderr, "no repositories found for %q\n", owner)
		os.Exit(0)
	}

	// Audit all repos in parallel, bounded by --concurrency.
	multi := &multiReport{
		Owner:     owner,
		Timestamp: time.Now().UTC(),
		Repos:     make([]*core.Report, len(repos)),
	}

	sem := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup

	for i, r := range repos {
		wg.Add(1)
		go func(idx int, repo github.Repo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			multi.Repos[idx] = auditGitHubRepo(ctx, client, owner, repo.Name)
		}(i, r)
	}
	wg.Wait()

	// Render JSON to stdout.
	data, err := json.MarshalIndent(multi, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error rendering JSON report: %v\n", err)
		os.Exit(3)
	}
	fmt.Println(string(data))

	// Optionally write HTML.
	if *htmlOut != "" {
		if err := report.WriteMultiHTML(owner, multi.Timestamp, multi.Repos, *htmlOut); err != nil {
			fmt.Fprintf(os.Stderr, "error writing HTML report: %v\n", err)
		}
	}

	os.Exit(multi.exitCode())
}

// auditGitHubRepo runs all GitHub-API–based checks for a single repository
// and returns its report.
func auditGitHubRepo(ctx context.Context, client *github.Client, owner, repo string) *core.Report {
	runner := core.NewRunner(owner+"/"+repo, &core.Config{})
	runner.AddCheck(newGHRequiredFilesCheck(ctx, client, owner, repo))
	runner.AddCheck(newGHCICheck(ctx, client, owner, repo))
	runner.AddCheck(checks.NewPullRequestsCheck(client, owner, repo))
	return runner.Run()
}

// ---------------------------------------------------------------------------
// API-backed Required Files check

type ghRequiredFilesCheck struct {
	ctx    context.Context
	client *github.Client
	owner  string
	repo   string
}

func newGHRequiredFilesCheck(ctx context.Context, client *github.Client, owner, repo string) core.Check {
	return &ghRequiredFilesCheck{ctx: ctx, client: client, owner: owner, repo: repo}
}

func (c *ghRequiredFilesCheck) Name() string { return "required_files" }

func (c *ghRequiredFilesCheck) Run(_ string) core.CheckResult {
	result := core.CheckResult{Name: "Required Files", Status: core.SeverityPass}

	profile, err := c.client.GetCommunityProfile(c.ctx, c.owner, c.repo)
	if err != nil {
		result.Status = core.SeverityWarning
		result.Message = fmt.Sprintf("Could not fetch community profile: %v", err)
		return result
	}

	var failMissing []string
	var warnMissing []string

	if profile.Files.Readme == nil {
		failMissing = append(failMissing, "README")
	}
	if profile.Files.License == nil {
		failMissing = append(failMissing, "LICENSE")
	}
	if profile.Files.Contributing == nil {
		warnMissing = append(warnMissing, "CONTRIBUTING.md")
	}
	if profile.Files.SecurityPolicy == nil {
		warnMissing = append(warnMissing, "SECURITY.md")
	}

	switch {
	case len(failMissing) > 0:
		result.Status = core.SeverityFail
		all := append(failMissing, warnMissing...)
		result.Message = fmt.Sprintf("Missing required files: %s", strings.Join(all, ", "))
	case len(warnMissing) > 0:
		result.Status = core.SeverityWarning
		result.Message = fmt.Sprintf("Missing recommended files: %s", strings.Join(warnMissing, ", "))
	default:
		result.Message = "All required files present"
	}

	return result
}

// ---------------------------------------------------------------------------
// API-backed CI Detection check

type ghCICheck struct {
	ctx    context.Context
	client *github.Client
	owner  string
	repo   string
}

func newGHCICheck(ctx context.Context, client *github.Client, owner, repo string) core.Check {
	return &ghCICheck{ctx: ctx, client: client, owner: owner, repo: repo}
}

func (c *ghCICheck) Name() string { return "ci" }

func (c *ghCICheck) Run(_ string) core.CheckResult {
	result := core.CheckResult{Name: "CI Configuration"}

	ok, err := c.client.HasWorkflows(c.ctx, c.owner, c.repo)
	if err != nil {
		result.Status = core.SeverityWarning
		result.Message = fmt.Sprintf("Could not check CI workflows: %v", err)
		return result
	}

	if ok {
		result.Status = core.SeverityPass
		result.Message = "GitHub Actions workflow file(s) found"
	} else {
		result.Status = core.SeverityWarning
		result.Message = "No GitHub Actions workflow files found in .github/workflows/"
	}
	return result
}
