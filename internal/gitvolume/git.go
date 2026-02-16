package gitvolume

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindWorktreeRoot returns the root directory of the current git worktree.
func FindWorktreeRoot(startDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = startDir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git rev-parse failed (%s): %w", strings.TrimSpace(string(exitErr.Stderr)), exitErr)
		}
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// findCommonDir finds the main worktree root directory.
// For regular repos, returns the parent of .git directory.
// For bare repos, returns the common directory itself.
func findCommonDir(startDir string) (string, error) {
	// git rev-parse --git-common-dir returns the path to the .git dir of the main repo
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = startDir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git rev-parse --git-common-dir failed (%s): %w", strings.TrimSpace(string(exitErr.Stderr)), exitErr)
		}
		return "", fmt.Errorf("git rev-parse --git-common-dir failed: %w", err)
	}
	commonDir := strings.TrimSpace(string(out))

	var absCommonDir string
	if filepath.IsAbs(commonDir) {
		absCommonDir = commonDir
	} else {
		absCommonDir = filepath.Join(startDir, commonDir)
	}

	absCommonDir, err = filepath.Abs(absCommonDir)
	if err != nil {
		return "", err
	}

	// Check if this is a bare repository
	isBare, err := isBareRepository(absCommonDir)
	if err != nil {
		// If bare check fails, assume not bare (safer default)
		isBare = false
	}

	var mainRoot string
	if isBare {
		// For bare repos, the common dir itself is the repo root
		// Config files should be placed directly in the bare repo directory
		mainRoot = absCommonDir
	} else {
		// For regular repos, the root is the parent of .git
		mainRoot = filepath.Dir(absCommonDir)

		// Verify by checking if this looks like a valid git repo root
		verifyCmd := exec.Command("git", "rev-parse", "--show-toplevel")
		verifyCmd.Dir = mainRoot
		verifyOut, verifyErr := verifyCmd.Output()
		if verifyErr == nil {
			mainRoot = strings.TrimSpace(string(verifyOut))
		}
	}

	return mainRoot, nil
}

// isBareRepository checks if the given directory is a bare git repository.
func isBareRepository(dir string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}
