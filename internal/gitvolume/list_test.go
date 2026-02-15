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

func TestBuildGlobalTree(t *testing.T) {
	globalDir, cleanup := setupGlobalTestEnv(t)
	defer cleanup()

	// 1. Setup files
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "file_root_z.txt"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "file_root_a.txt"), []byte(""), 0644))

	subDir := filepath.Join(globalDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file_sub.txt"), []byte(""), 0644))

	// 2. Build Tree
	gv := createTestGitVolumeWithGlobal(globalDir)
	root, err := gv.buildGlobalTree(globalDir)
	require.NoError(t, err)

	// 3. Verify Structure
	// Root should have 3 children: subdir (dir), file_root_a.txt, file_root_z.txt
	// Sorting should put directories first, then files alphabetically.
	require.Len(t, root.children, 3)

	// Child 0: subdir
	assert.Equal(t, "subdir", root.children[0].name)
	assert.True(t, root.children[0].isDir)
	assert.Len(t, root.children[0].children, 1)
	assert.Equal(t, "file_sub.txt", root.children[0].children[0].name)

	// Child 1: file_root_a.txt
	assert.Equal(t, "file_root_a.txt", root.children[1].name)
	assert.False(t, root.children[1].isDir)

	// Child 2: file_root_z.txt
	assert.Equal(t, "file_root_z.txt", root.children[2].name)
	assert.False(t, root.children[2].isDir)
}

func TestSortTree(t *testing.T) {
	// Manually construct an unsorted tree
	root := &treeNode{name: "root", isDir: true}
	fileB := &treeNode{name: "b.txt", isDir: false}
	fileA := &treeNode{name: "a.txt", isDir: false}
	dirC := &treeNode{name: "c_dir", isDir: true}

	root.children = []*treeNode{fileB, dirC, fileA}

	// helper to verify order
	verifyOrder := func() {
		require.Len(t, root.children, 3)
		assert.Equal(t, "c_dir", root.children[0].name, "Directories should come first")
		assert.Equal(t, "a.txt", root.children[1].name, "Files should be alphabetical")
		assert.Equal(t, "b.txt", root.children[2].name, "Files should be alphabetical")
	}

	// Sort
	sortTree(root)
	verifyOrder()
}
