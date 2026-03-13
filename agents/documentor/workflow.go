package documentor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ensureClone clones repoURL into workDir if it has not been cloned yet.
func ensureClone(repoURL, ref, workDir string) error {
	if _, err := os.Stat(filepath.Join(workDir, ".git")); err == nil {
		return nil // already cloned
	}

	args := []string{"clone", "--depth=1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, repoURL, ".")

	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
	}
	return nil
}
