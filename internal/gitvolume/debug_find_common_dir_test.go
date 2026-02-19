package gitvolume

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugFindCommonDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-volume-debug-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 1. Regular Repository
	t.Run("Regular Repository", func(t *testing.T) {
		repoDir := filepath.Join(tmpDir, "regular")
		require.NoError(t, os.MkdirAll(repoDir, 0755))

		cmd := exec.Command("git", "init", repoDir)
		require.NoError(t, cmd.Run())

		// Set identity
		cmd = exec.Command("git", "-C", repoDir, "config", "user.email", "test@test.com")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", repoDir, "config", "user.name", "Test")
		require.NoError(t, cmd.Run())

		// Create a commit so we can create a worktree
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("init"), 0644))
		cmd = exec.Command("git", "-C", repoDir, "add", ".")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", repoDir, "commit", "-m", "Initial commit")
		require.NoError(t, cmd.Run())

		commonDir, err := findCommonDir(repoDir)
		require.NoError(t, err)

		realRepoDir, _ := filepath.EvalSymlinks(repoDir)
		assert.Equal(t, realRepoDir, commonDir, "Common dir of regular repo root should be itself")

		// Worktree from Regular Repo
		wtDir := filepath.Join(tmpDir, "regular-worktree")
		cmd = exec.Command("git", "-C", repoDir, "worktree", "add", wtDir)
		require.NoError(t, cmd.Run())

		commonDirWT, err := findCommonDir(wtDir)
		require.NoError(t, err)

		commonDirWT, err = filepath.EvalSymlinks(commonDirWT)
		require.NoError(t, err)

		assert.Equal(t, realRepoDir, commonDirWT, "Worktree from regular repo should point back to main repo root")
	})

	// 2. Bare Repository
	t.Run("Bare Repository", func(t *testing.T) {
		bareRepoDir := filepath.Join(tmpDir, "bare.git")
		cmd := exec.Command("git", "init", "--bare", bareRepoDir)
		require.NoError(t, cmd.Run())

		realBareDir, err := filepath.EvalSymlinks(bareRepoDir)
		require.NoError(t, err)

		commonDir, err := findCommonDir(bareRepoDir)
		require.NoError(t, err)

		commonDir, err = filepath.EvalSymlinks(commonDir)
		require.NoError(t, err)

		assert.Equal(t, realBareDir, commonDir, "Common dir of bare repo root should be itself")

		// Create a worktree from bare repo
		// Need a commit first? Bare repos don't have commits unless pushed or created from existing.
		// Let's create a regular repo first, then clone as bare to have commits.
		srcRepo := filepath.Join(tmpDir, "src")
		require.NoError(t, os.MkdirAll(srcRepo, 0755))
		cmd = exec.Command("git", "init", srcRepo)
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "-C", srcRepo, "config", "user.email", "test@test.com")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", srcRepo, "config", "user.name", "Test")
		require.NoError(t, cmd.Run())

		require.NoError(t, os.WriteFile(filepath.Join(srcRepo, "README.md"), []byte("init"), 0644))
		cmd = exec.Command("git", "-C", srcRepo, "add", ".")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", srcRepo, "commit", "-m", "Initial commit")
		require.NoError(t, cmd.Run())

		// Clone as bare
		bareCloned := filepath.Join(tmpDir, "bare-cloned.git")
		cmd = exec.Command("git", "clone", "--bare", srcRepo, bareCloned)
		require.NoError(t, cmd.Run())

		realBareCloned, err := filepath.EvalSymlinks(bareCloned)
		require.NoError(t, err)

		// Create worktree
		wtDir := filepath.Join(tmpDir, "bare-worktree")
		cmd = exec.Command("git", "-C", bareCloned, "worktree", "add", wtDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Git output: %s", string(out))
			require.NoError(t, err)
		}

		commonDirWT, err := findCommonDir(wtDir)
		require.NoError(t, err)
		commonDirWT, err = filepath.EvalSymlinks(commonDirWT)
		require.NoError(t, err)

		// DEBUGGING: Check what git rev-parse --git-common-dir returns here
		cmd = exec.Command("git", "-C", wtDir, "rev-parse", "--git-common-dir")
		out, _ := cmd.Output()
		gitCommonDir := strings.TrimSpace(string(out))
		t.Logf("git rev-parse --git-common-dir in worktree: %s", gitCommonDir)

		// Check isBareRepository on that dir
		isBare, err := isBareRepository(gitCommonDir)
		t.Logf("isBareRepository(%s) = %v, err=%v", gitCommonDir, isBare, err)

		assert.Equal(t, realBareCloned, commonDirWT, "Worktree from bare repo should point back to bare repo root")
	})
}
