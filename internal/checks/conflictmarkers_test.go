package checks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backbiten/jitterbugs/internal/core"
)

func TestConflictMarkersCheck_NoMarkers(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	result := NewConflictMarkersCheck().Run(dir)
	if result.Status != core.SeverityPass {
		t.Errorf("expected pass when no conflict markers, got %s: %s", result.Status, result.Message)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected no findings, got %d", len(result.Findings))
	}
}

func TestConflictMarkersCheck_ConflictInFile(t *testing.T) {
	dir := t.TempDir()
	content := "line one\n<<<<<<< HEAD\nour change\n=======\ntheir change\n>>>>>>> feature-branch\nline after\n"
	must(t, os.WriteFile(filepath.Join(dir, "conflict.go"), []byte(content), 0o644))

	result := NewConflictMarkersCheck().Run(dir)
	if result.Status != core.SeverityFail {
		t.Errorf("expected fail when conflict markers present, got %s: %s", result.Status, result.Message)
	}
	// Expect three findings: <<<<<<<, =======, >>>>>>>
	if len(result.Findings) != 3 {
		t.Errorf("expected 3 findings (one per marker type), got %d", len(result.Findings))
	}
}

func TestConflictMarkersCheck_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	content := "<<<<<<< HEAD\nA\n=======\nB\n>>>>>>> other\n"
	must(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte(content), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "file2.txt"), []byte(content), 0o644))

	result := NewConflictMarkersCheck().Run(dir)
	if result.Status != core.SeverityFail {
		t.Errorf("expected fail, got %s", result.Status)
	}
	if len(result.Findings) != 6 {
		t.Errorf("expected 6 findings (3 per file × 2 files), got %d", len(result.Findings))
	}
}

func TestConflictMarkersCheck_FindingsIncludeFileAndLine(t *testing.T) {
	dir := t.TempDir()
	content := "before\n<<<<<<< HEAD\nA\n=======\nB\n>>>>>>> other\nafter\n"
	must(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0o644))

	result := NewConflictMarkersCheck().Run(dir)
	if result.Status != core.SeverityFail {
		t.Fatalf("expected fail, got %s", result.Status)
	}

	for _, f := range result.Findings {
		if f.File == "" {
			t.Error("finding should have a non-empty file path")
		}
		if f.Line == 0 {
			t.Error("finding should have a non-zero line number")
		}
		if f.Pattern != "conflict_marker" {
			t.Errorf("unexpected pattern %q", f.Pattern)
		}
	}

	// <<<<<<< is on line 2
	if result.Findings[0].Line != 2 {
		t.Errorf("expected first marker on line 2, got %d", result.Findings[0].Line)
	}
}

func TestConflictMarkersCheck_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := NewConflictMarkersCheck().Run(dir)
	// No files → pass.
	if result.Status != core.SeverityPass {
		t.Errorf("expected pass for empty directory, got %s: %s", result.Status, result.Message)
	}
}

func TestConflictMarkersCheck_SkipBinaryFile(t *testing.T) {
	dir := t.TempDir()
	// Binary content (NUL bytes) that happens to contain a conflict marker prefix.
	content := []byte("<<<<<<< HEAD\x00\x00\x00binary")
	must(t, os.WriteFile(filepath.Join(dir, "archive.bin"), content, 0o644))

	result := NewConflictMarkersCheck().Run(dir)
	if result.Status == core.SeverityFail {
		t.Errorf("binary files should be skipped, got fail: %s", result.Message)
	}
}

func TestConflictMarkersCheck_MarkerNotAtLineStart(t *testing.T) {
	dir := t.TempDir()
	// These are NOT conflict markers because they don't start the line.
	content := "prefix <<<<<<< HEAD\nsome = ======= text\n"
	must(t, os.WriteFile(filepath.Join(dir, "code.go"), []byte(content), 0o644))

	result := NewConflictMarkersCheck().Run(dir)
	if result.Status != core.SeverityPass {
		t.Errorf("markers embedded mid-line should not be flagged, got %s: %s", result.Status, result.Message)
	}
}
