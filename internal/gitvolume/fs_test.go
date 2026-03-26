package gitvolume

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	// 1. Normal copy
	err := os.WriteFile(src, []byte("hello"), 0644)
	require.NoError(t, err)

	err = copyFile(src, dst)
	require.NoError(t, err)

	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	info, err := os.Stat(dst)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	}

	// 2. Source missing
	err = copyFile(filepath.Join(tmpDir, "missing"), dst)
	assert.Error(t, err)

	// 3. Dest read-only (directory)
	// Create a directory where the file should be to trigger error
	err = os.Mkdir(filepath.Join(tmpDir, "readonly"), 0755)
	require.NoError(t, err)
	err = copyFile(src, filepath.Join(tmpDir, "readonly"))
	assert.Error(t, err)
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")

	// Setup source structure
	require.NoError(t, os.MkdirAll(filepath.Join(src, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "file1.txt"), []byte("file1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "subdir", "file2.txt"), []byte("file2"), 0644))

	// 1. Normal recursive copy
	err := copyDir(src, dst)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(dst, "file1.txt"))
	assert.FileExists(t, filepath.Join(dst, "subdir", "file2.txt"))

	// 2. Symlink in source (should fail)
	symLinkSrc := filepath.Join(tmpDir, "symsrc")
	require.NoError(t, os.Mkdir(symLinkSrc, 0755))
	require.NoError(t, os.Symlink(dst, filepath.Join(symLinkSrc, "link")))

	err = copyDir(symLinkSrc, filepath.Join(tmpDir, "symdst"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlink, which is not allowed")

	// 3. Target exists and is not a directory
	isFile := filepath.Join(tmpDir, "isFile")
	require.NoError(t, os.WriteFile(isFile, []byte("data"), 0644))
	err = copyDir(src, isFile)
	assert.Error(t, err)
	// Error message differs by OS for MkdirAll on existing file
	// Linux/Mac: "not a directory"
}

func TestCopyDir_PreservesDestinationRootAndReplacesConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")

	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "app.env"), []byte("SRC_APP"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "child.txt"), []byte("SRC_CHILD"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(dst, "keep"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "keep", "local.txt"), []byte("LOCAL"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "app.env"), []byte("OLD_APP"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "nested"), []byte("conflicting file"), 0644))

	err := copyDir(src, dst)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dst, "app.env"))
	require.NoError(t, err)
	assert.Equal(t, "SRC_APP", string(content))

	content, err = os.ReadFile(filepath.Join(dst, "nested", "child.txt"))
	require.NoError(t, err)
	assert.Equal(t, "SRC_CHILD", string(content))

	content, err = os.ReadFile(filepath.Join(dst, "keep", "local.txt"))
	require.NoError(t, err)
	assert.Equal(t, "LOCAL", string(content), "unrelated files in destination root should be preserved")
}

func TestCopyDir_FileDoesNotReplaceExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")

	require.NoError(t, os.MkdirAll(src, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "keep"), []byte("SRC_FILE"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(dst, "keep"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "keep", "local.txt"), []byte("LOCAL"), 0644))

	err := copyDir(src, dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target exists and is a directory")

	content, readErr := os.ReadFile(filepath.Join(dst, "keep", "local.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "LOCAL", string(content), "existing directory subtree should be preserved on conflict")
}

func TestHashAndVerify(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	require.NoError(t, os.WriteFile(file1, []byte("content"), 0644))
	require.NoError(t, os.WriteFile(file2, []byte("content"), 0644))

	// 1. hashFile
	h1, err := hashFile(file1)
	require.NoError(t, err)
	h2, err := hashFile(file2)
	require.NoError(t, err)
	assert.Equal(t, h1, h2)

	// 2. verifyHash matches
	match, err := verifyHash(file1, file2)
	require.NoError(t, err)
	assert.True(t, match)

	// 3. verifyHash mismatch
	require.NoError(t, os.WriteFile(file2, []byte("diff"), 0644))
	match, err = verifyHash(file1, file2)
	require.NoError(t, err)
	assert.False(t, match)

	// 4. Missing file
	_, err = hashFile(filepath.Join(tmpDir, "missing"))
	assert.Error(t, err)
}

func TestCleanEmptyParents(t *testing.T) {
	tmpDir := t.TempDir()
	nested := filepath.Join(tmpDir, "a", "b", "c")
	require.NoError(t, os.MkdirAll(nested, 0755))

	// 1. Clean up empty dirs logic
	// Remove 'c', then cleanEmptyParents should remove 'b' and 'a' but stop at tmpDir
	err := os.Remove(nested)
	require.NoError(t, err)

	cleanEmptyParents(filepath.Join(tmpDir, "a", "b"), tmpDir)

	assert.NoDirExists(t, filepath.Join(tmpDir, "a", "b"))
	assert.NoDirExists(t, filepath.Join(tmpDir, "a"))
	assert.DirExists(t, tmpDir)

	// 2. Stop if not empty
	require.NoError(t, os.MkdirAll(nested, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a", "file.txt"), []byte("keep"), 0644))

	err = os.Remove(nested)
	require.NoError(t, err)

	cleanEmptyParents(filepath.Join(tmpDir, "a", "b"), tmpDir)

	assert.NoDirExists(t, filepath.Join(tmpDir, "a", "b"))
	assert.DirExists(t, filepath.Join(tmpDir, "a")) // Should exist because of file.txt
}
