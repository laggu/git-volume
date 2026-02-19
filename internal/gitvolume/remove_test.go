package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalRemove(t *testing.T) {
	_, _, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	// 1. Setup - Create additional files for testing
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "file1.txt"), []byte("content"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "dir1", "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "dir1", "file2.txt"), []byte("content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "dir1", "subdir", "file3.txt"), []byte("content"), 0644))

	gv := createTestGitVolume("", "", globalDir, nil)

	// Test 1: success - remove file
	err := gv.GlobalRemove([]string{"file1.txt"})
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(globalDir, "file1.txt"))

	// Test 2: success - remove directory
	// Removing 'subdir' should remove file3.txt
	err = gv.GlobalRemove([]string{"dir1/subdir"})
	require.NoError(t, err)
	assert.NoDirExists(t, filepath.Join(globalDir, "dir1", "subdir"))
	assert.FileExists(t, filepath.Join(globalDir, "dir1", "file2.txt")) // Sibling preserved

	// Test 3: cleanup - remove last file in dir1, dir1 should be removed
	err = gv.GlobalRemove([]string{"dir1/file2.txt"})
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(globalDir, "dir1", "file2.txt"))
	assert.NoDirExists(t, filepath.Join(globalDir, "dir1")) // Cleanup happened

	// Test 4: missing file
	err = gv.GlobalRemove([]string{"missing.txt"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test 5: path traversal
	err = gv.GlobalRemove([]string{"../outside.txt"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")

	// Test 6: multiple files with error aggregation
	// Create one valid file
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "valid.txt"), []byte("content"), 0644))

	err = gv.GlobalRemove([]string{"valid.txt", "missing.txt"})
	assert.Error(t, err)                                          // Should return error because one failed
	assert.NoFileExists(t, filepath.Join(globalDir, "valid.txt")) // Valid one should still be removed
}

func TestGlobalRemove_NotInitialized(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-volume-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Point to a non-existent directory
	globalDir := filepath.Join(tmpDir, "non_existent_global")
	gv := createTestGitVolume("", "", globalDir, nil)

	err = gv.GlobalRemove([]string{"file.txt"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global storage not initialized")
}
