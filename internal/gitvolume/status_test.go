package gitvolume

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitVolume_status(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "local.txt", Mode: ModeLink, IsGlobal: false},
		{Source: "global.txt", Target: "global.txt", Mode: ModeLink, IsGlobal: true},
		{Source: "nonexistent.txt", Target: "missing.txt", Mode: ModeLink, IsGlobal: false},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	// Before sync
	statuses, err := gv.status()
	require.NoError(t, err)
	assert.Equal(t, 3, len(statuses))

	// Check not mounted status
	assert.Equal(t, StatusNotMounted, statuses[0].Status)

	// Check missing source
	assert.Equal(t, StatusMissingSource, statuses[2].Status)

	// Sync local and global
	volumes = volumes[:2] // Remove the nonexistent one for sync
	gv.ctx.Volumes = volumes
	require.NoError(t, gv.Sync(SyncOptions{}))

	// After sync
	statuses, _ = gv.status()
	assert.Equal(t, StatusOKLinked, statuses[0].Status)
	assert.Equal(t, StatusOKLinked, statuses[1].Status)

	// Check display source for global
	assert.Equal(t, "@global/global.txt", statuses[1].Source)
}

func TestGitVolume_status_CopyDirectoryModified(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	configDir := filepath.Join(sourceDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.env"), []byte("A=1"), 0644))

	volumes := []Volume{
		{Source: "config", Target: "config", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	require.NoError(t, gv.Sync(SyncOptions{}))

	statuses, err := gv.status()
	require.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, StatusOKCopied, statuses[0].Status)

	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "config", "app.env"), []byte("A=2"), 0644))

	statuses, err = gv.status()
	require.NoError(t, err)
	assert.Equal(t, StatusModified, statuses[0].Status)
}

func TestAfterAllStatus(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "local.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Test afterAllStatus with nil error
	statuses := []VolumeStatus{
		{Source: "source1.txt", Target: "local.txt", Mode: ModeLink, Status: StatusNotMounted},
	}
	err := gv.afterAllStatus(statuses, nil)
	assert.NoError(t, err)

	// Test afterAllStatus with error
	err = gv.afterAllStatus(nil, fmt.Errorf("test error"))
	assert.Error(t, err)
}
