// Package core defines the shared data types and interfaces for the QAQC scanner.
package core

import "time"

// Severity represents the result level of a check.
type Severity string

const (
	SeverityPass    Severity = "pass"
	SeverityWarning Severity = "warning"
	SeverityFail    Severity = "fail"
)

// Finding represents a single detected issue within a file.
type Finding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Pattern string `json:"pattern"`
	Match   string `json:"match"`
}

// CheckResult holds the outcome of a single check.
type CheckResult struct {
	Name     string    `json:"name"`
	Status   Severity  `json:"status"`
	Message  string    `json:"message"`
	Findings []Finding `json:"findings,omitempty"`
}

// Report is the top-level scan result returned to callers.
type Report struct {
	RepoPath      string        `json:"repo_path"`
	Timestamp     time.Time     `json:"timestamp"`
	OverallStatus Severity      `json:"overall_status"`
	Results       []CheckResult `json:"results"`
}

// ExitCode maps the overall status to a POSIX exit code:
//
//	0 = pass, 1 = warnings only, 2 = fail.
func (r *Report) ExitCode() int {
	switch r.OverallStatus {
	case SeverityFail:
		return 2
	case SeverityWarning:
		return 1
	default:
		return 0
	}
}

// ChecksConfig controls which checks are enabled.
type ChecksConfig struct {
	RequiredFiles *bool `json:"required_files"`
	CI            *bool `json:"ci"`
	Secrets       *bool `json:"secrets"`
	PullRequests  *bool `json:"pull_requests"`
}

// Config holds the optional per-repo .qaqc.json configuration.
type Config struct {
	// RequiredFiles overrides the default list of required file names to verify.
	RequiredFiles []string     `json:"required_files"`
	Checks        ChecksConfig `json:"checks"`
}

// CheckEnabled returns true when the named check should run.
// All checks are enabled by default when no config is present.
func (c *Config) CheckEnabled(name string) bool {
	if c == nil {
		return true
	}
	var flag *bool
	switch name {
	case "required_files":
		flag = c.Checks.RequiredFiles
	case "ci":
		flag = c.Checks.CI
	case "secrets":
		flag = c.Checks.Secrets
	case "pull_requests":
		flag = c.Checks.PullRequests
	}
	return flag == nil || *flag
}
