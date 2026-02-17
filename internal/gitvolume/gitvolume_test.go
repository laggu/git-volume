package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestEnv(t *testing.T) (sourceDir, targetDir string, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "git-volume-test")
	require.NoError(t, err)

	sourceDir = filepath.Join(tmpDir, "source")
	targetDir = filepath.Join(tmpDir, "target")
	require.NoError(t, os.Mkdir(sourceDir, 0755))
	require.NoError(t, os.Mkdir(targetDir, 0755))

	// Create source files
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "source1.txt"), []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "source2.txt"), []byte("content2"), 0644))

	cleanup = func() { _ = os.RemoveAll(tmpDir) }
	return
}

func setupTestEnvWithGlobal(t *testing.T) (sourceDir, targetDir, globalDir string, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "git-volume-test")
	require.NoError(t, err)

	sourceDir = filepath.Join(tmpDir, "source")
	targetDir = filepath.Join(tmpDir, "target")
	globalDir = filepath.Join(tmpDir, "global")
	require.NoError(t, os.Mkdir(sourceDir, 0755))
	require.NoError(t, os.Mkdir(targetDir, 0755))
	require.NoError(t, os.Mkdir(globalDir, 0755))

	// Create source files
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "source1.txt"), []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "source2.txt"), []byte("content2"), 0644))

	// Create global files
	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "secrets"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "secrets", "prod.key"), []byte("global-secret"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "global.txt"), []byte("global-content"), 0644))

	cleanup = func() { _ = os.RemoveAll(tmpDir) }
	return
}

// createTestGitVolume creates a GitVolume for testing with pre-configured context
func createTestGitVolume(sourceDir, targetDir, globalDir string, volumes []Volume) *GitVolume {
	ctx := &Context{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		GlobalDir: globalDir,
		Volumes:   volumes,
	}
	ctx.ResolveVolumePaths()

	return &GitVolume{
		ctx:     ctx,
		verbose: false,
		quiet:   true,
	}
}
