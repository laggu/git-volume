package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalAdd(t *testing.T) {
	// Setup temporary directory for global storage and source files
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global")
	srcDir := filepath.Join(tmpDir, "src")

	err := os.MkdirAll(globalDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Create a dummy GitVolume instance with mocked context
	gv := &GitVolume{
		ctx: &Context{
			GlobalDir: globalDir,
		},
		quiet: true,
	}

	// Create test files
	file1 := filepath.Join(srcDir, "file1.txt")
	err = os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(srcDir, "file2.txt")
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	t.Run("Add single file", func(t *testing.T) {
		err := gv.GlobalAdd([]string{file1}, AddOptions{})
		assert.NoError(t, err)

		destPath := filepath.Join(globalDir, "file1.txt")
		assert.FileExists(t, destPath)
		content, _ := os.ReadFile(destPath)
		assert.Equal(t, "content1", string(content))
	})

	t.Run("Add multiple files", func(t *testing.T) {
		err := gv.GlobalAdd([]string{file1, file2}, AddOptions{Force: true}) // Use force because file1 exists
		assert.NoError(t, err)

		assert.FileExists(t, filepath.Join(globalDir, "file1.txt"))
		assert.FileExists(t, filepath.Join(globalDir, "file2.txt"))
	})

	t.Run("Add safely (no overwrite)", func(t *testing.T) {
		// file1 already exists in global
		err := gv.GlobalAdd([]string{file1}, AddOptions{Force: false})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("Add with --as (rename)", func(t *testing.T) {
		err := gv.GlobalAdd([]string{file1}, AddOptions{As: "renamed.txt"})
		assert.NoError(t, err)

		destPath := filepath.Join(globalDir, "renamed.txt")
		assert.FileExists(t, destPath)
		content, _ := os.ReadFile(destPath)
		assert.Equal(t, "content1", string(content))
	})

	t.Run("Add with --path (subdirectory)", func(t *testing.T) {
		err := gv.GlobalAdd([]string{file2}, AddOptions{Path: "subdir"})
		assert.NoError(t, err)

		destPath := filepath.Join(globalDir, "subdir", "file2.txt")
		assert.FileExists(t, destPath)
		content, _ := os.ReadFile(destPath)
		assert.Equal(t, "content2", string(content))
	})

	t.Run("Add with both --as and multiple files (error)", func(t *testing.T) {
		err := gv.GlobalAdd([]string{file1, file2}, AddOptions{As: "renamed.txt"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--as can only be used with a single file")
	})

	t.Run("Add non-existent file", func(t *testing.T) {
		err := gv.GlobalAdd([]string{filepath.Join(srcDir, "missing.txt")}, AddOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source does not exist")
	})
}
