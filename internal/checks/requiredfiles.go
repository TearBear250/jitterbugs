// Package checks implements the individual QAQC scan checks.
package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/backbiten/jitterbugs/internal/core"
)

// recommendedFiles lists files whose absence triggers a warning (not a failure).
var recommendedFiles = []string{"SECURITY.md", "CONTRIBUTING.md"}

type requiredFilesCheck struct {
	extraRequired []string
}

// NewRequiredFilesCheck returns a check that verifies expected files exist.
// When cfg supplies a non-empty RequiredFiles list those files are checked in
// addition to the built-in defaults.
func NewRequiredFilesCheck(cfg *core.Config) core.Check {
	c := &requiredFilesCheck{}
	if cfg != nil {
		c.extraRequired = cfg.RequiredFiles
	}
	return c
}

func (c *requiredFilesCheck) Name() string { return "required_files" }

func (c *requiredFilesCheck) Run(repoPath string) core.CheckResult {
	result := core.CheckResult{
		Name:   "Required Files",
		Status: core.SeverityPass,
	}

	var failMissing []string
	var warnMissing []string

	// README and LICENSE are mandatory – absence causes a fail.
	if !globExists(repoPath, "README*") {
		failMissing = append(failMissing, "README")
	}
	if !globExists(repoPath, "LICENSE*") {
		failMissing = append(failMissing, "LICENSE")
	}

	// SECURITY.md and CONTRIBUTING.md are recommended – absence is a warning.
	for _, name := range recommendedFiles {
		if !fileExists(filepath.Join(repoPath, name)) {
			warnMissing = append(warnMissing, name)
		}
	}

	// Extra files from config are treated as mandatory.
	for _, name := range c.extraRequired {
		if !fileExists(filepath.Join(repoPath, name)) {
			failMissing = append(failMissing, name)
		}
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

// fileExists reports whether path exists on disk.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// globExists reports whether any file matching pattern exists in dir.
func globExists(dir, pattern string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	return err == nil && len(matches) > 0
}
