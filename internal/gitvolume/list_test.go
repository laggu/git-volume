package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGlobalTestEnv(t *testing.T) (globalDir string, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "git-volume-global-test")
	require.NoError(t, err)

	globalDir = filepath.Join(tmpDir, "global")
	cleanup = func() { _ = os.RemoveAll(tmpDir) }
	return
}

func createTestGitVolumeWithGlobal(globalDir string) *GitVolume {
	return &GitVolume{
		ctx: &Context{
			GlobalDir: globalDir,
		},
		quiet: true, // suppress stdout in tests
	}
}

func TestBuildGlobalTree_NonExistentDir(t *testing.T) {
	globalDir, cleanup := setupGlobalTestEnv(t)
	defer cleanup()

	gv := createTestGitVolumeWithGlobal(globalDir)
	root, err := gv.buildGlobalTree(globalDir)
	require.NoError(t, err)
	assert.Empty(t, root.children, "should have no children if dir doesn't exist")
}

func TestBuildGlobalTree_EmptyDir(t *testing.T) {
	globalDir, cleanup := setupGlobalTestEnv(t)
	defer cleanup()

	require.NoError(t, os.MkdirAll(globalDir, 0755))

	gv := createTestGitVolumeWithGlobal(globalDir)
	root, err := gv.buildGlobalTree(globalDir)
	require.NoError(t, err)
	assert.Empty(t, root.children, "should have no children if dir is empty")
}

func TestBuildGlobalTree_WithFiles(t *testing.T) {
	globalDir, cleanup := setupGlobalTestEnv(t)
	defer cleanup()

	require.NoError(t, os.MkdirAll(globalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "file1.txt"), []byte("c"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "file2.txt"), []byte("c"), 0644))

	subDir := filepath.Join(globalDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file3.txt"), []byte("c"), 0644))

	gv := createTestGitVolumeWithGlobal(globalDir)
	root, err := gv.buildGlobalTree(globalDir)
	require.NoError(t, err)

	// Root should have: subdir (dir, sorted first), file1.txt, file2.txt
	assert.Len(t, root.children, 3)
	assert.Equal(t, "subdir", root.children[0].name)
	assert.True(t, root.children[0].isDir)
	assert.Len(t, root.children[0].children, 1)
	assert.Equal(t, "file3.txt", root.children[0].children[0].name)
	assert.Equal(t, "file1.txt", root.children[1].name)
	assert.Equal(t, "file2.txt", root.children[2].name)
}
