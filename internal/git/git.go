// Package git implements Git integration for SLB.
// Handles pre-commit hooks, branch detection, and repository context.
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRepo checks if the given path is inside a Git repository.
func IsRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetRoot returns the repository root for the given path.
func GetRoot(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetBranch returns the current branch name.
func GetBranch(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// InstallHook installs the SLB pre-commit hook.
func InstallHook(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}

	hookPath := filepath.Join(absPath, ".git", "hooks", "pre-commit")

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		return os.ErrExist
	}

	hookContent := `#!/bin/sh
# SLB pre-commit hook - validates pending approvals
exec slb hook pre-commit "$@"
`
	return os.WriteFile(hookPath, []byte(hookContent), 0755)
}
