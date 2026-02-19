package gitvolume

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindCommonDir_Repro(t *testing.T) {
	// Create a temporary directory for our test environment
	tmpDir, err := os.MkdirTemp("", "git-volume-repro-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 1. Test Case: Bare Repository + Worktree
	t.Run("Bare Repository with Worktree", func(t *testing.T) {
		bareRepo := filepath.Join(tmpDir, "bare.git")
		worktree := filepath.Join(tmpDir, "worktree")

		// Create a non-bare origin repo with an initial commit,
		// then clone it as bare.

		origin := filepath.Join(tmpDir, "origin")
		require.NoError(t, os.MkdirAll(origin, 0755))
		cmd := exec.Command("git", "init", origin)
		require.NoError(t, cmd.Run())

		// config user
		cmd = exec.Command("git", "-C", origin, "config", "user.email", "test@example.com")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", origin, "config", "user.name", "Test User")
		require.NoError(t, cmd.Run())

		// commit
		require.NoError(t, os.WriteFile(filepath.Join(origin, "README.md"), []byte("test"), 0644))
		cmd = exec.Command("git", "-C", origin, "add", ".")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", origin, "commit", "-m", "initial")
		require.NoError(t, cmd.Run())

		// Clone as bare (this creates the bareRepo directory)
		cmd = exec.Command("git", "clone", "--bare", origin, bareRepo)
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "-C", bareRepo, "worktree", "add", "-b", "bare-worktree", worktree)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git worktree add failed: %s, output: %s", err, out)
		}

		// Resolve symlinks for accurate comparison
		bareRepo, err = filepath.EvalSymlinks(bareRepo)
		require.NoError(t, err)

		// Run findCommonDir from worktree root
		commonDir, err := findCommonDir(worktree)
		require.NoError(t, err)

		// For bare repo, common dir should be the bare repo path itself
		assert.Equal(t, bareRepo, commonDir, "Should identify bare repo root as common dir")

		// Run findCommonDir from subdirectory of worktree
		subDir := filepath.Join(worktree, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		commonDirSub, err := findCommonDir(subDir)
		require.NoError(t, err)
		assert.Equal(t, bareRepo, commonDirSub, "Should identify bare repo root from subdirectory")
	})

	// 2. Test Case: Regular Repository + Worktree
	t.Run("Regular Repository with Worktree", func(t *testing.T) {
		mainRepo := filepath.Join(tmpDir, "main")
		worktree := filepath.Join(tmpDir, "main-worktree")

		// Initialize main repo
		require.NoError(t, os.MkdirAll(mainRepo, 0755))
		cmd := exec.Command("git", "init", mainRepo)
		require.NoError(t, cmd.Run())

		// config user
		cmd = exec.Command("git", "-C", mainRepo, "config", "user.email", "test@example.com")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", mainRepo, "config", "user.name", "Test User")
		require.NoError(t, cmd.Run())

		// commit
		require.NoError(t, os.WriteFile(filepath.Join(mainRepo, "README.md"), []byte("test"), 0644))
		cmd = exec.Command("git", "-C", mainRepo, "add", ".")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", mainRepo, "commit", "-m", "initial")
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "-C", mainRepo, "worktree", "add", "-b", "main-worktree-branch", worktree)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git worktree add failed: %s, output: %s", err, out)
		}

		// Resolve symlinks
		mainRepo, err = filepath.EvalSymlinks(mainRepo)
		require.NoError(t, err)

		// Run findCommonDir from worktree root
		commonDir, err := findCommonDir(worktree)
		require.NoError(t, err)

		// For regular repo, common dir is .git dir, so root is parent of .git
		// BUT findCommonDir returns the ROOT of the main repo, not the .git dir.
		assert.Equal(t, mainRepo, commonDir, "Should identify main repo root")
	})
}
