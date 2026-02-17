package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitVolume_Add_SymlinkSourceRejected(t *testing.T) {
	_, _, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "git-volume-add-symlink-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	realFile := filepath.Join(tmpDir, "real.txt")
	require.NoError(t, os.WriteFile(realFile, []byte("secret"), 0644))

	symlinkFile := filepath.Join(tmpDir, "link-to-real.txt")
	require.NoError(t, os.Symlink(realFile, symlinkFile))

	gv := createTestGitVolume("", "", globalDir, nil)

	err = gv.GlobalAdd([]string{symlinkFile}, AddOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
	assert.Contains(t, err.Error(), "not allowed")

	entries, _ := os.ReadDir(globalDir)
	for _, e := range entries {
		assert.NotEqual(t, "link-to-real.txt", e.Name(), "symlink source should not have been copied to global dir")
	}
}

func TestGitVolume_Add_RegularFile(t *testing.T) {
	_, _, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "git-volume-add-regular-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	realFile := filepath.Join(tmpDir, "config.txt")
	require.NoError(t, os.WriteFile(realFile, []byte("config-content"), 0644))

	gv := createTestGitVolume("", "", globalDir, nil)

	err = gv.GlobalAdd([]string{realFile}, AddOptions{})
	require.NoError(t, err)

	dstPath := filepath.Join(globalDir, "config.txt")
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "config-content", string(data))
}

func TestGitVolume_Add_SymlinkDirectoryRejected(t *testing.T) {
	_, _, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "git-volume-add-symdir-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	realDir := filepath.Join(tmpDir, "real-dir")
	require.NoError(t, os.MkdirAll(realDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(realDir, "file.txt"), []byte("inside"), 0644))

	symlinkDir := filepath.Join(tmpDir, "link-to-dir")
	require.NoError(t, os.Symlink(realDir, symlinkDir))

	gv := createTestGitVolume("", "", globalDir, nil)

	err = gv.GlobalAdd([]string{symlinkDir}, AddOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}

func TestGitVolume_Add_NonexistentSource(t *testing.T) {
	_, _, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	gv := createTestGitVolume("", "", globalDir, nil)

	err := gv.GlobalAdd([]string{"/nonexistent/path/file.txt"}, AddOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source does not exist")
}

func TestGitVolume_Add_DestinationSymlinkDetected(t *testing.T) {
	_, _, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "git-volume-add-dst-symlink-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	srcFile := filepath.Join(tmpDir, "new.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("new-content"), 0644))

	externalDir := filepath.Join(tmpDir, "external")
	require.NoError(t, os.MkdirAll(externalDir, 0755))
	externalFile := filepath.Join(externalDir, "target.txt")
	require.NoError(t, os.WriteFile(externalFile, []byte("external"), 0644))

	dstSymlink := filepath.Join(globalDir, "new.txt")
	require.NoError(t, os.Symlink(externalFile, dstSymlink))

	gv := createTestGitVolume("", "", globalDir, nil)

	err = gv.GlobalAdd([]string{srcFile}, AddOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGitVolume_Add_DirectoryContainingSymlinkRejected(t *testing.T) {
	_, _, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "git-volume-add-dir-symlink-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	srcDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "app.env"), []byte("A=1"), 0644))

	outside := filepath.Join(tmpDir, "outside.txt")
	require.NoError(t, os.WriteFile(outside, []byte("outside"), 0644))
	require.NoError(t, os.Symlink(outside, filepath.Join(srcDir, "secret-link")))

	gv := createTestGitVolume("", "", globalDir, nil)

	err = gv.GlobalAdd([]string{srcDir}, AddOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contains a symlink")

	_, statErr := os.Lstat(filepath.Join(globalDir, "config", "secret-link"))
	assert.True(t, os.IsNotExist(statErr), "symlink entry should never be copied")
}

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
		file3 := filepath.Join(srcDir, "file3.txt")
		err = os.WriteFile(file3, []byte("content3"), 0644)
		require.NoError(t, err)
		file4 := filepath.Join(srcDir, "file4.txt")
		err = os.WriteFile(file4, []byte("content4"), 0644)
		require.NoError(t, err)

		err := gv.GlobalAdd([]string{file3, file4}, AddOptions{})
		assert.NoError(t, err)

		assert.FileExists(t, filepath.Join(globalDir, "file3.txt"))
		assert.FileExists(t, filepath.Join(globalDir, "file4.txt"))
	})

	t.Run("Add existing file (error)", func(t *testing.T) {
		err := gv.GlobalAdd([]string{file1}, AddOptions{})
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

	t.Run("Security: Block symlink traversal", func(t *testing.T) {
		outsideDir := t.TempDir()
		symlinkDir := filepath.Join(globalDir, "symlink_dir")
		err := os.Symlink(outsideDir, symlinkDir)
		require.NoError(t, err)

		err = gv.GlobalAdd([]string{file1}, AddOptions{Path: "symlink_dir"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security error")
		assert.Contains(t, err.Error(), "path escapes base directory")
	})
}
