package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitVolume_Unsync(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
		{Source: "source2.txt", Target: "copy2.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync first
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	_, err := os.Stat(filepath.Join(targetDir, "link1.txt"))
	assert.True(t, os.IsNotExist(err), "link1.txt should be removed")

	_, err = os.Stat(filepath.Join(targetDir, "copy2.txt"))
	assert.True(t, os.IsNotExist(err), "copy2.txt should be removed")
}

func TestGitVolume_Unsync_SafetyCheck(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
		{Source: "source2.txt", Target: "copy2.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync
	require.NoError(t, gv.Sync(SyncOptions{}), "Sync failed")

	// Modify Copy Target
	copyPath := filepath.Join(targetDir, "copy2.txt")
	require.NoError(t, os.WriteFile(copyPath, []byte("MODIFIED CONTENT"), 0644))

	// Modify Link Target (Replace link with real file)
	linkPath := filepath.Join(targetDir, "link1.txt")
	require.NoError(t, os.Remove(linkPath))
	require.NoError(t, os.WriteFile(linkPath, []byte("NOT A LINK"), 0644))

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}), "Unsync failed")

	// Verify Copy was Preserved (Skipped)
	data, _ := os.ReadFile(copyPath)
	assert.Equal(t, "MODIFIED CONTENT", string(data), "Modified copy file was deleted!")

	// Verify Link-replaced-file was Preserved (Skipped)
	dataLink, _ := os.ReadFile(linkPath)
	assert.Equal(t, "NOT A LINK", string(dataLink), "Replaced link file was deleted!")
}

func TestGitVolume_Unsync_DryRun(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// First sync normally
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Unsync with dry-run
	require.NoError(t, gv.Unsync(UnsyncOptions{DryRun: true}))

	// Verify file still exists
	_, err := os.Stat(filepath.Join(targetDir, "link1.txt"))
	assert.False(t, os.IsNotExist(err), "link1.txt should still exist after dry-run unsync")
}

func TestGitVolume_Unsync_GlobalSource(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "local.txt", Mode: ModeLink, IsGlobal: false},
		{Source: "global.txt", Target: "global.txt", Mode: ModeLink, IsGlobal: true},
		{Source: "secrets/prod.key", Target: "config/key", Mode: ModeCopy, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	// Sync first
	require.NoError(t, gv.Sync(SyncOptions{}), "Sync failed")

	// Verify all mounted
	_, err := os.Lstat(filepath.Join(targetDir, "local.txt"))
	assert.NoError(t, err, "local.txt should exist after sync")
	_, err = os.Lstat(filepath.Join(targetDir, "global.txt"))
	assert.NoError(t, err, "global.txt should exist after sync")
	_, err = os.Lstat(filepath.Join(targetDir, "config/key"))
	assert.NoError(t, err, "config/key should exist after sync")

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}), "Unsync failed")

	// Verify all removed
	_, err = os.Stat(filepath.Join(targetDir, "local.txt"))
	assert.True(t, os.IsNotExist(err), "local.txt should be removed")
	_, err = os.Stat(filepath.Join(targetDir, "global.txt"))
	assert.True(t, os.IsNotExist(err), "global.txt should be removed")
	_, err = os.Stat(filepath.Join(targetDir, "config/key"))
	assert.True(t, os.IsNotExist(err), "config/key should be removed")
}

func TestGitVolume_Unsync_GlobalSource_EmptyGlobalBase(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// GlobalDir is empty string
	volumes := []Volume{
		{Source: "secrets/key", Target: "config/key", Mode: ModeLink, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	err := gv.Unsync(UnsyncOptions{})
	assert.Error(t, err, "expected error when GlobalDir is empty")
	assert.Contains(t, err.Error(), "global directory not configured")
}

func TestGitVolume_Unsync_CleanEmptyParents(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Setup: Map to a deep nested path
	volumes := []Volume{
		{Source: "source1.txt", Target: "deep/nested/dir/link.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Verify existence
	targetPath := filepath.Join(targetDir, "deep/nested/dir/link.txt")
	_, err := os.Stat(targetPath) // Stat follows link, verifying source exists too
	require.NoError(t, err)

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	// Verify file is gone
	_, err = os.Lstat(targetPath)
	assert.True(t, os.IsNotExist(err), "Target link should be removed")

	// Verify deep/nested/dir is gone
	_, err = os.Stat(filepath.Join(targetDir, "deep/nested/dir"))
	assert.True(t, os.IsNotExist(err), "Empty parent dir 'deep/nested/dir' should be removed")

	// Verify deep/nested is gone
	_, err = os.Stat(filepath.Join(targetDir, "deep/nested"))
	assert.True(t, os.IsNotExist(err), "Empty parent dir 'deep/nested' should be removed")

	// Verify deep is gone
	_, err = os.Stat(filepath.Join(targetDir, "deep"))
	assert.True(t, os.IsNotExist(err), "Empty parent dir 'deep' should be removed")

	// Verify targetDir still exists
	_, err = os.Stat(targetDir)
	assert.NoError(t, err, "Root target dir should NOT be removed")
}

func TestGitVolume_Unsync_SafetyCheck_HijackedLink(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Setup
	volumes := []Volume{
		{Source: "source1.txt", Target: "link.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Hijack the link: Point it to source2.txt instead of source1.txt
	linkPath := filepath.Join(targetDir, "link.txt")
	require.NoError(t, os.Remove(linkPath))
	require.NoError(t, os.Symlink(filepath.Join(sourceDir, "source2.txt"), linkPath))

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	// Verify link was NOT removed (because it doesn't point to configured source)
	_, err := os.Lstat(linkPath)
	assert.NoError(t, err, "Hijacked link should NOT be removed")

	target, _ := os.Readlink(linkPath)
	assert.Contains(t, target, "source2.txt")
}

func TestGitVolume_Unsync_Copy_MissingSource(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Setup
	volumes := []Volume{
		{Source: "source1.txt", Target: "copy.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Delete Source File
	require.NoError(t, os.Remove(filepath.Join(sourceDir, "source1.txt")))

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	// Verify copy.txt was NOT removed (because hash verification failed due to missing source)
	// This is the safe behavior: if we can't verify it's ours/unmodified, we don't touch it.
	_, err := os.Stat(filepath.Join(targetDir, "copy.txt"))
	assert.NoError(t, err, "Copy should NOT be removed if source is missing")
}

func TestGitVolume_Unsync_Copy_SymlinkSourcePreservesTarget(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "copy.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	require.NoError(t, gv.Sync(SyncOptions{}))

	realSource := filepath.Join(sourceDir, "source1.txt")
	replacement := filepath.Join(sourceDir, "source2.txt")
	require.NoError(t, os.Remove(realSource))
	require.NoError(t, os.Symlink(replacement, realSource))

	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	content, err := os.ReadFile(filepath.Join(targetDir, "copy.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content), "copy target should be preserved when source becomes a symlink")
}

func TestGitVolume_Unsync_RelativeLink(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Setup
	volumes := []Volume{
		{Source: "source1.txt", Target: "rel_link.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync with RelativeLinks=true
	require.NoError(t, gv.Sync(SyncOptions{RelativeLinks: true}))

	linkPath := filepath.Join(targetDir, "rel_link.txt")

	// Verify it is indeed relative
	target, err := os.Readlink(linkPath)
	require.NoError(t, err)
	assert.False(t, filepath.IsAbs(target), "Link should be relative")

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	// Verify link is removed
	_, err = os.Lstat(linkPath)
	assert.True(t, os.IsNotExist(err), "Relative link should be correctly identified and removed")
}

func TestGitVolume_Unsync_CopyDirectory(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create source directory with files
	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.env"), []byte("A=1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "db.env"), []byte("B=2"), 0644))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Verify synced
	targetConfig := filepath.Join(targetDir, "config")
	_, err := os.Stat(filepath.Join(targetConfig, "app.env"))
	require.NoError(t, err, "app.env should exist after sync")
	_, err = os.Stat(filepath.Join(targetConfig, "db.env"))
	require.NoError(t, err, "db.env should exist after sync")

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	// Verify directory is completely removed
	_, err = os.Stat(targetConfig)
	assert.True(t, os.IsNotExist(err), "config directory should be removed after unsync")
}

func TestGitVolume_Unsync_CopyDirectory_PreservesExtraFiles(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "nested"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.env"), []byte("A=1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "nested", "db.env"), []byte("B=2"), 0644))

	targetConfig := filepath.Join(targetDir, "config")
	require.NoError(t, os.MkdirAll(filepath.Join(targetConfig, "keep"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(targetConfig, "keep", "local.txt"), []byte("LOCAL"), 0644))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	require.NoError(t, gv.Sync(SyncOptions{}))
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	_, err := os.Stat(filepath.Join(targetConfig, "app.env"))
	assert.True(t, os.IsNotExist(err), "managed file should be removed")
	_, err = os.Stat(filepath.Join(targetConfig, "nested", "db.env"))
	assert.True(t, os.IsNotExist(err), "managed nested file should be removed")

	content, err := os.ReadFile(filepath.Join(targetConfig, "keep", "local.txt"))
	require.NoError(t, err)
	assert.Equal(t, "LOCAL", string(content), "unrelated extra file should be preserved")

	_, err = os.Stat(targetConfig)
	assert.NoError(t, err, "target directory should remain when extra files exist")
}

func TestGitVolume_Unsync_CopyDirectory_Modified(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create source directory
	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.env"), []byte("A=1"), 0644))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Modify a file inside the copied directory
	modifiedPath := filepath.Join(targetDir, "config", "app.env")
	require.NoError(t, os.WriteFile(modifiedPath, []byte("MODIFIED"), 0644))

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	// Verify directory was preserved (hash mismatch → skip)
	_, err := os.Stat(filepath.Join(targetDir, "config"))
	assert.NoError(t, err, "Modified config directory should NOT be removed")

	data, _ := os.ReadFile(modifiedPath)
	assert.Equal(t, "MODIFIED", string(data), "Modified file content should be preserved")
}

func TestGitVolume_Unsync_CopyDirectory_MissingSource(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create source directory
	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.env"), []byte("A=1"), 0644))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync
	require.NoError(t, gv.Sync(SyncOptions{}))

	// Delete source directory
	require.NoError(t, os.RemoveAll(configDir))

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	// Verify target directory was NOT removed (source missing → safe skip)
	_, err := os.Stat(filepath.Join(targetDir, "config"))
	assert.NoError(t, err, "Config directory should NOT be removed if source is missing")
}

func TestGitVolume_Unsync_TypeMismatch(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// 1. Setup Source as Directory
	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Manually create Target as File (simulating a change or weird state)
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "config"), []byte("I am a file"), 0644))

	// Unsync
	require.NoError(t, gv.Unsync(UnsyncOptions{}))

	// Verify Target File Preserved (Safety check should fail due to type mismatch)
	// checkRemovable sees source is Dir, target is File -> mismatch -> returns false
	info, err := os.Stat(filepath.Join(targetDir, "config"))
	require.NoError(t, err)
	assert.True(t, !info.IsDir())

	data, _ := os.ReadFile(filepath.Join(targetDir, "config"))
	assert.Equal(t, "I am a file", string(data))
}
