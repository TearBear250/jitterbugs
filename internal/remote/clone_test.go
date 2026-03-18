package remote

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// makeLocalRepo initialises a git repository at src with a single commit,
// so that Clone can be tested without any network access.
func makeLocalRepo(t *testing.T) string {
	t.Helper()

	src := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		c := exec.Command(args[0], args[1:]...) //nolint:gosec
		c.Dir = src
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "Test")
	must(t, os.WriteFile(filepath.Join(src, "README.md"), []byte("# test\n"), 0o644))
	run("git", "add", ".")
	run("git", "commit", "-m", "init")

	return src
}

func TestClone_Success(t *testing.T) {
	src := makeLocalRepo(t)

	dir, cleanup, err := Clone(src)
	if err != nil {
		t.Fatalf("Clone returned unexpected error: %v", err)
	}
	defer cleanup()

	if dir == "" {
		t.Fatal("expected non-empty dir path")
	}

	// The cloned repo must contain the file we committed.
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Errorf("expected README.md in cloned repo: %v", err)
	}
}

func TestClone_InvalidURL(t *testing.T) {
	_, cleanup, err := Clone("/this/path/does/not/exist/at/all")
	cleanup() // must be callable even on error
	if err == nil {
		t.Fatal("expected an error cloning a non-existent path, got nil")
	}
}

func TestClone_CleanupRemovesDir(t *testing.T) {
	src := makeLocalRepo(t)

	dir, cleanup, err := Clone(src)
	if err != nil {
		t.Fatalf("Clone returned unexpected error: %v", err)
	}

	cleanup()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected temp dir %q to be removed after cleanup, but it still exists", dir)
	}
}

// must is a test-helper that calls t.Fatal on error.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
