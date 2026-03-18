package checks

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/backbiten/jitterbugs/internal/core"
)

// conflictMarkerPrefixes are the three line-start prefixes that git inserts
// when a merge conflict is left unresolved. The actual marker lines may contain
// additional text after the prefix (e.g. "<<<<<<< HEAD" or ">>>>>>> branch"),
// so HasPrefix is the correct comparison — not a full-string equality check.
var conflictMarkerPrefixes = []string{"<<<<<<<", "=======", ">>>>>>>"}

type conflictMarkersCheck struct{}

// NewConflictMarkersCheck returns a check that scans tracked files for
// unresolved Git merge conflict markers.
func NewConflictMarkersCheck() core.Check {
	return &conflictMarkersCheck{}
}

func (c *conflictMarkersCheck) Name() string { return "conflict_markers" }

func (c *conflictMarkersCheck) Run(repoPath string) core.CheckResult {
	result := core.CheckResult{Name: "Conflict Markers"}

	files, err := trackedFiles(repoPath)
	if err != nil {
		files, err = allFiles(repoPath)
		if err != nil {
			result.Status = core.SeverityWarning
			result.Message = "Could not enumerate repository files for conflict marker scan"
			return result
		}
	}

	var findings []core.Finding
	for _, rel := range files {
		ff := scanConflictMarkers(filepath.Join(repoPath, rel), rel)
		findings = append(findings, ff...)
	}

	result.Findings = findings

	if len(findings) > 0 {
		result.Status = core.SeverityFail
		result.Message = fmt.Sprintf("Found %d unresolved merge conflict marker(s) in %d file(s)",
			len(findings), countDistinctFiles(findings))
	} else {
		result.Status = core.SeverityPass
		result.Message = "No unresolved merge conflict markers detected"
	}

	return result
}

// scanConflictMarkers reads path and returns a Finding for each line that
// contains a Git merge conflict marker prefix. relPath is stored in findings.
func scanConflictMarkers(path, relPath string) []core.Finding {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	buf := make([]byte, maxScanBytes)
	n, _ := f.Read(buf)
	if n == 0 {
		return nil
	}
	data := buf[:n]

	// Skip binary files (NUL bytes in first 8 KiB).
	probe := data
	if len(probe) > 8192 {
		probe = probe[:8192]
	}
	if bytes.IndexByte(probe, 0) >= 0 {
		return nil
	}

	var findings []core.Finding
	sc := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		for _, prefix := range conflictMarkerPrefixes {
			if strings.HasPrefix(line, prefix) {
				findings = append(findings, core.Finding{
					File:    relPath,
					Line:    lineNum,
					Pattern: "conflict_marker",
					Match:   line[:min(len(line), 40)],
				})
				break
			}
		}
	}
	return findings
}

// countDistinctFiles returns the number of unique file paths across findings.
func countDistinctFiles(findings []core.Finding) int {
	seen := make(map[string]struct{}, len(findings))
	for _, f := range findings {
		seen[f.File] = struct{}{}
	}
	return len(seen)
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
