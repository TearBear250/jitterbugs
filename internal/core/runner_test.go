package core

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Runner tests ---

type stubCheck struct {
	name   string
	result CheckResult
}

func (s *stubCheck) Name() string             { return s.name }
func (s *stubCheck) Run(_ string) CheckResult { return s.result }

func TestRunner_OverallPass(t *testing.T) {
	r := NewRunner(t.TempDir(), &Config{})
	r.AddCheck(&stubCheck{"a", CheckResult{Status: SeverityPass}})
	r.AddCheck(&stubCheck{"b", CheckResult{Status: SeverityPass}})
	rpt := r.Run()
	if rpt.OverallStatus != SeverityPass {
		t.Errorf("expected pass, got %s", rpt.OverallStatus)
	}
	if rpt.ExitCode() != 0 {
		t.Errorf("expected exit code 0, got %d", rpt.ExitCode())
	}
}

func TestRunner_OverallWarning(t *testing.T) {
	r := NewRunner(t.TempDir(), &Config{})
	r.AddCheck(&stubCheck{"a", CheckResult{Status: SeverityPass}})
	r.AddCheck(&stubCheck{"b", CheckResult{Status: SeverityWarning}})
	rpt := r.Run()
	if rpt.OverallStatus != SeverityWarning {
		t.Errorf("expected warning, got %s", rpt.OverallStatus)
	}
	if rpt.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", rpt.ExitCode())
	}
}

func TestRunner_OverallFail(t *testing.T) {
	r := NewRunner(t.TempDir(), &Config{})
	r.AddCheck(&stubCheck{"a", CheckResult{Status: SeverityWarning}})
	r.AddCheck(&stubCheck{"b", CheckResult{Status: SeverityFail}})
	rpt := r.Run()
	if rpt.OverallStatus != SeverityFail {
		t.Errorf("expected fail, got %s", rpt.OverallStatus)
	}
	if rpt.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d", rpt.ExitCode())
	}
}

func TestRunner_FailOverridesWarning(t *testing.T) {
	// Fail should win even if it comes before warning.
	r := NewRunner(t.TempDir(), &Config{})
	r.AddCheck(&stubCheck{"a", CheckResult{Status: SeverityFail}})
	r.AddCheck(&stubCheck{"b", CheckResult{Status: SeverityWarning}})
	rpt := r.Run()
	if rpt.OverallStatus != SeverityFail {
		t.Errorf("expected fail to dominate warning, got %s", rpt.OverallStatus)
	}
}

func TestRunner_SkipsDisabledCheck(t *testing.T) {
	f := false
	cfg := &Config{Checks: ChecksConfig{CI: &f}}
	r := NewRunner(t.TempDir(), cfg)
	r.AddCheck(&stubCheck{"ci", CheckResult{Status: SeverityFail}})
	rpt := r.Run()
	if len(rpt.Results) != 0 {
		t.Errorf("expected disabled check to be skipped, got %d results", len(rpt.Results))
	}
	if rpt.OverallStatus != SeverityPass {
		t.Errorf("expected pass when all checks disabled, got %s", rpt.OverallStatus)
	}
}

// --- LoadConfig tests ---

func TestLoadConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	cfg := LoadConfig(dir)
	if cfg == nil {
		t.Fatal("expected non-nil config even when file absent")
	}
}

func TestLoadConfig_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	json := `{"required_files":["CHANGELOG.md"],"checks":{"secrets":false}}`
	if err := os.WriteFile(filepath.Join(dir, ".qaqc.json"), []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadConfig(dir)
	if len(cfg.RequiredFiles) != 1 || cfg.RequiredFiles[0] != "CHANGELOG.md" {
		t.Errorf("unexpected required_files: %v", cfg.RequiredFiles)
	}
	if cfg.Checks.Secrets == nil || *cfg.Checks.Secrets {
		t.Error("expected secrets check to be disabled")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".qaqc.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := LoadConfig(dir)
	// Should not panic and return default config.
	if cfg == nil {
		t.Fatal("expected non-nil config on parse error")
	}
}

// --- CheckEnabled tests ---

func TestConfig_CheckEnabled_Defaults(t *testing.T) {
	cfg := &Config{}
	for _, name := range []string{"required_files", "ci", "secrets", "conflict_markers"} {
		if !cfg.CheckEnabled(name) {
			t.Errorf("check %q should be enabled by default", name)
		}
	}
}

func TestConfig_CheckEnabled_NilConfig(t *testing.T) {
	var cfg *Config
	if !cfg.CheckEnabled("ci") {
		t.Error("nil config should enable all checks")
	}
}
