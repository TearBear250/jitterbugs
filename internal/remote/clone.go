// Package remote provides helpers for working with remote git repositories.
package remote

import (
	"fmt"
	"os"
	"os/exec"
)

// Clone performs a shallow git clone of the repository at url into a new
// temporary directory. It returns the path to the cloned directory and a
// cleanup function that removes it. The caller must invoke cleanup() when
// the directory is no longer needed, regardless of whether an error occurred
// in subsequent operations.
//
// A shallow clone (--depth 1) is used so that only the latest commit is
// fetched, keeping bandwidth and disk usage minimal.
func Clone(url string) (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "qaqc-clone-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("creating temp directory: %w", err)
	}
	cleanup = func() { os.RemoveAll(dir) } //nolint:errcheck

	cmd := exec.Command("git", "clone", "--depth", "1", "--", url, dir)
	if out, cloneErr := cmd.CombinedOutput(); cloneErr != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("git clone %q: %w\n%s", url, cloneErr, out)
	}

	return dir, cleanup, nil
}
