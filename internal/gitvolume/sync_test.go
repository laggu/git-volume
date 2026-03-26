package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitVolume_Sync_Link(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	require.NoError(t, gv.Sync(SyncOptions{}))

	linkPath := filepath.Join(targetDir, "link1.txt")
	info, err := os.Lstat(linkPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "link1.txt should be valid symlink")

	target, _ := os.Readlink(linkPath)
	expected := filepath.Join(sourceDir, "source1.txt")
	assert.Equal(t, expected, target)
}

func TestGitVolume_Sync_Copy(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source2.txt", Target: "copy2.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	require.NoError(t, gv.Sync(SyncOptions{}), "Sync failed")

	copyPath := filepath.Join(targetDir, "copy2.txt")
	info, err := os.Stat(copyPath)
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular(), "copy2.txt should be regular file")

	data, _ := os.ReadFile(copyPath)
	assert.Equal(t, "content2", string(data), "copy content mismatch")
}

func TestGitVolume_Sync_CopyDirectory(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.env"), []byte("A=1"), 0644))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	require.NoError(t, gv.Sync(SyncOptions{}), "Sync failed")

	dstFile := filepath.Join(targetDir, "config", "app.env")
	data, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, "A=1", string(data))
}

func TestGitVolume_Sync_CopyDirectory_ExistingTargetOverwrites(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.env"), []byte("A=1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "new.env"), []byte("NEW"), 0644))

	targetConfigDir := filepath.Join(targetDir, "config")
	require.NoError(t, os.MkdirAll(targetConfigDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(targetConfigDir, "app.env"), []byte("OLD"), 0644))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Should succeed and overwrite because we always overwrite now
	err := gv.Sync(SyncOptions{})
	require.NoError(t, err)

	// Verify content was updated
	data, err := os.ReadFile(filepath.Join(targetDir, "config", "app.env"))
	require.NoError(t, err)
	assert.Equal(t, "A=1", string(data))
}

func TestGitVolume_Sync_CopyDirectory_PreservesExistingTargetDirectoryContents(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "nested"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.env"), []byte("A=1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "nested", "source.txt"), []byte("SRC"), 0644))

	targetConfigDir := filepath.Join(targetDir, "config")
	require.NoError(t, os.MkdirAll(filepath.Join(targetConfigDir, "keep"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(targetConfigDir, "keep", "local.txt"), []byte("LOCAL"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(targetConfigDir, "app.env"), []byte("OLD"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(targetConfigDir, "nested"), []byte("conflicting file"), 0644))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	require.NoError(t, gv.Sync(SyncOptions{}))

	data, err := os.ReadFile(filepath.Join(targetConfigDir, "app.env"))
	require.NoError(t, err)
	assert.Equal(t, "A=1", string(data))

	data, err = os.ReadFile(filepath.Join(targetConfigDir, "nested", "source.txt"))
	require.NoError(t, err)
	assert.Equal(t, "SRC", string(data))

	data, err = os.ReadFile(filepath.Join(targetConfigDir, "keep", "local.txt"))
	require.NoError(t, err)
	assert.Equal(t, "LOCAL", string(data), "existing unrelated files in target directory should be preserved")
}

func TestGitVolume_Sync_Link_ExistingTargetOverwrites(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create existing file at target
	targetPath := filepath.Join(targetDir, "link1.txt")
	require.NoError(t, os.WriteFile(targetPath, []byte("existing content"), 0644))

	// Try sync - should succeed (rename/remove logic)
	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	err := gv.Sync(SyncOptions{})
	assert.NoError(t, err)

	// Verify it's now a symlink
	info, err := os.Lstat(targetPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "target should be replaced with symlink")
}

func TestGitVolume_Sync_Link_ExistingDirectoryOverwrites(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create existing directory at target
	targetPath := filepath.Join(targetDir, "link1.txt")
	require.NoError(t, os.Mkdir(targetPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(targetPath, "subfile.txt"), []byte("sub"), 0644))

	// Sync - should succeed (remove directory and symlink)
	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	err := gv.Sync(SyncOptions{})
	assert.NoError(t, err)

	// Verify it's now a symlink
	info, err := os.Lstat(targetPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "target directory should be replaced with symlink")
}

func TestGitVolume_Sync_Transition_LinkToCopy(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// 1. Sync as Link
	volumes := []Volume{
		{Source: "source1.txt", Target: "file.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Verify symlink
	info, err := os.Lstat(filepath.Join(targetDir, "file.txt"))
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)

	// 2. Sync as Copy
	volumes[0].Mode = ModeCopy
	gv = createTestGitVolume(sourceDir, targetDir, "", volumes)
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Verify file (not symlink)
	info, err = os.Lstat(filepath.Join(targetDir, "file.txt"))
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink == 0, "Symlink should be replaced by regular file")
	assert.True(t, info.Mode().IsRegular())
}

func TestGitVolume_Sync_DryRun(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
		{Source: "source2.txt", Target: "copy2.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync with dry-run
	require.NoError(t, gv.Sync(SyncOptions{DryRun: true}))

	// Verify nothing was created
	_, err := os.Stat(filepath.Join(targetDir, "link1.txt"))
	assert.True(t, os.IsNotExist(err), "link1.txt should not exist in dry-run mode")

	_, err = os.Stat(filepath.Join(targetDir, "copy2.txt"))
	assert.True(t, os.IsNotExist(err), "copy2.txt should not exist in dry-run mode")
}

func TestGitVolume_Sync_RelativeLinks(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync with relative links
	require.NoError(t, gv.Sync(SyncOptions{RelativeLinks: true}), "Sync with relative links failed")

	linkPath := filepath.Join(targetDir, "link1.txt")
	target, err := os.Readlink(linkPath)
	require.NoError(t, err, "Failed to read link")

	// Relative link should not be absolute path
	assert.False(t, filepath.IsAbs(target), "link target should be relative")

	// Verify the link still works (can read through it)
	content, err := os.ReadFile(linkPath)
	require.NoError(t, err, "Failed to read through relative symlink")
	assert.Equal(t, "content1", string(content), "content mismatch through relative symlink")
}

func TestGitVolume_Sync_NestedTarget(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "deep/nested/dir/link.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	require.NoError(t, gv.Sync(SyncOptions{}))

	linkPath := filepath.Join(targetDir, "deep/nested/dir/link.txt")
	info, err := os.Lstat(linkPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "nested target should be symlink")
}

func TestGitVolume_Sync_GlobalSource_Link(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "secrets/prod.key", Target: "config/key", Mode: ModeLink, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	require.NoError(t, gv.Sync(SyncOptions{}), "Sync failed")

	linkPath := filepath.Join(targetDir, "config/key")
	info, err := os.Lstat(linkPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "config/key should be valid symlink")

	// Verify link points to global directory
	target, _ := os.Readlink(linkPath)
	expected := filepath.Join(globalDir, "secrets", "prod.key")
	assert.Equal(t, expected, target)

	// Verify content is readable
	content, err := os.ReadFile(linkPath)
	require.NoError(t, err, "Failed to read through symlink")
	assert.Equal(t, "global-secret", string(content))
}

func TestGitVolume_Sync_GlobalSource_Copy(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "global.txt", Target: "copied.txt", Mode: ModeCopy, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	require.NoError(t, gv.Sync(SyncOptions{}), "Sync failed")

	copyPath := filepath.Join(targetDir, "copied.txt")
	info, err := os.Stat(copyPath)
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular(), "copied.txt should be regular file")

	data, _ := os.ReadFile(copyPath)
	assert.Equal(t, "global-content", string(data))
}

func TestGitVolume_Sync_MixedLocalAndGlobal(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "local.txt", Mode: ModeLink, IsGlobal: false},
		{Source: "global.txt", Target: "global.txt", Mode: ModeLink, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	require.NoError(t, gv.Sync(SyncOptions{}), "Sync failed")

	// Verify local link points to source directory
	localPath := filepath.Join(targetDir, "local.txt")
	localTarget, _ := os.Readlink(localPath)
	assert.Equal(t, filepath.Join(sourceDir, "source1.txt"), localTarget)

	// Verify global link points to global directory
	globalPath := filepath.Join(targetDir, "global.txt")
	globalTarget, _ := os.Readlink(globalPath)
	assert.Equal(t, filepath.Join(globalDir, "global.txt"), globalTarget)
}

func TestGitVolume_Sync_GlobalSource_NotFound(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "nonexistent.txt", Target: "target.txt", Mode: ModeLink, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	err := gv.Sync(SyncOptions{})
	assert.Error(t, err, "expected error for non-existent global source")
}

func TestGitVolume_Sync_GlobalSource_EmptyGlobalBase(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// GlobalDir is empty string
	volumes := []Volume{
		{Source: "secrets/key", Target: "config/key", Mode: ModeLink, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	err := gv.Sync(SyncOptions{})
	assert.Error(t, err, "expected error when GlobalDir is empty")
	assert.Contains(t, err.Error(), "global directory not configured")
}

func TestGitVolume_Sync_SourceIsSymlink(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create symlink source
	// We need absolute path for symlink source to be valid in most OSs or relative
	targetFile := filepath.Join(sourceDir, "target.txt")
	require.NoError(t, os.WriteFile(targetFile, []byte("target"), 0644))

	symlinkSource := filepath.Join(sourceDir, "symlink-source")
	require.NoError(t, os.Symlink(targetFile, symlinkSource))

	volumes := []Volume{
		{Source: "symlink-source", Target: "dest.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	err := gv.Sync(SyncOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source file is a symlink")
}

func TestGitVolume_Sync_PathTraversal(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// 1. Source traversal
	volumes := []Volume{
		{Source: "../outside.txt", Target: "dest.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)
	err := gv.Sync(SyncOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security error for source")

	// 2. Target traversal
	volumes = []Volume{
		{Source: "source1.txt", Target: "../outside.txt", Mode: ModeCopy},
	}
	gv = createTestGitVolume(sourceDir, targetDir, "", volumes)
	err = gv.Sync(SyncOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security error for target")
}

func TestGitVolume_Sync_Verbose(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link.txt", Mode: ModeLink},
		{Source: "source2.txt", Target: "copy.txt", Mode: ModeCopy},
		{Source: "global.txt", Target: "global.txt", Mode: ModeLink, IsGlobal: true},
	}
	ctx := &Context{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		GlobalDir: globalDir,
		Volumes:   volumes,
	}
	ctx.ResolveVolumePaths()

	gv := &GitVolume{ctx: ctx, verbosity: VerbosityDetailed}

	// Sync with verbose output
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Verify all files created
	_, err := os.Lstat(filepath.Join(targetDir, "link.txt"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(targetDir, "copy.txt"))
	assert.NoError(t, err)

	// Sync with relative links and verbose
	require.NoError(t, gv.Sync(SyncOptions{RelativeLinks: true}))
}

func TestGitVolume_Sync_NonQuiet_Error(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "nonexistent.txt", Target: "dest.txt", Mode: ModeLink},
	}
	ctx := &Context{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Volumes:   volumes,
	}
	ctx.ResolveVolumePaths()

	gv := &GitVolume{ctx: ctx, verbosity: VerbosityNormal}

	// Should report error non-quietly
	err := gv.Sync(SyncOptions{})
	assert.Error(t, err)
}
