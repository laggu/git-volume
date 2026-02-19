package gitvolume

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindWorktreeRoot(t *testing.T) {
	// 1. Regular git repo
	tmpDir := t.TempDir()
	cmd := exec.Command("git", "init", tmpDir)
	require.NoError(t, cmd.Run())

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))

	// Find root from root
	root, err := FindWorktreeRoot(tmpDir)
	require.NoError(t, err)

	// Evaluate symlinks in case /var vs /private/var on Mac
	evalTmpDir, _ := filepath.EvalSymlinks(tmpDir)
	evalRoot, _ := filepath.EvalSymlinks(root)
	assert.Equal(t, evalTmpDir, evalRoot)

	// Find root from subdir
	root, err = FindWorktreeRoot(subDir)
	require.NoError(t, err)
	evalRoot, _ = filepath.EvalSymlinks(root)
	assert.Equal(t, evalTmpDir, evalRoot)

	// 2. Not a git repo
	nonGitDir := t.TempDir()
	_, err = FindWorktreeRoot(nonGitDir)
	assert.Error(t, err)
}

func TestFindCommonDir(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Regular repo
	repoDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.Mkdir(repoDir, 0755))
	cmd := exec.Command("git", "init", repoDir)
	require.NoError(t, cmd.Run())

	commonDir, err := findCommonDir(repoDir)
	require.NoError(t, err)

	evalRepoDir, _ := filepath.EvalSymlinks(repoDir)
	evalCommonDir, _ := filepath.EvalSymlinks(commonDir)
	assert.Equal(t, evalRepoDir, evalCommonDir)

	// 2. Worktree (standard layout) is harder to set up without bare repo or commits
	// but findCommonDir should return main repo root for the main repo itself.
}

func TestIsBareRepository(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Regular repo
	repoDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.Mkdir(repoDir, 0755))
	cmd := exec.Command("git", "init", repoDir)
	require.NoError(t, cmd.Run())

	isBare, err := isBareRepository(repoDir)
	require.NoError(t, err)
	assert.False(t, isBare)

	// 2. Bare repo
	bareDir := filepath.Join(tmpDir, "bare.git")
	cmd = exec.Command("git", "init", "--bare", bareDir)
	require.NoError(t, cmd.Run())

	isBare, err = isBareRepository(bareDir)
	require.NoError(t, err)
	assert.True(t, isBare)
}
