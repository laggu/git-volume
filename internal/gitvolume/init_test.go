package gitvolume

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	// 1. Setup git repo
	tmpDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "repo"), 0755))
	repoDir := filepath.Join(tmpDir, "repo")
	cmd := exec.Command("git", "init", repoDir)
	require.NoError(t, cmd.Run())

	// Change CWD to repo
	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(wd) }() // Restore CWD
	require.NoError(t, os.Chdir(repoDir))

	// Setup GitVolume (GlobalDir will be in tmp)
	globalDir := filepath.Join(tmpDir, "global")
	gv := createTestGitVolume(repoDir, repoDir, globalDir, nil)
	gv.quiet = true

	// Test 1: Init success
	err = gv.Init()
	require.NoError(t, err)

	assert.DirExists(t, globalDir)
	assert.FileExists(t, filepath.Join(repoDir, "git-volume.yaml"))

	// Verify content
	content, err := os.ReadFile(filepath.Join(repoDir, "git-volume.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "volumes:")

	// Test 2: Idempotency (should not overwrite)
	err = os.WriteFile(filepath.Join(repoDir, "git-volume.yaml"), []byte("modified: true"), 0644)
	require.NoError(t, err)

	err = gv.Init()
	require.NoError(t, err)

	content, err = os.ReadFile(filepath.Join(repoDir, "git-volume.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "modified: true", string(content))
}

func TestInit_OutsideGit(t *testing.T) {
	// 1. Setup non-git dir
	tmpDir := t.TempDir()

	// Change CWD
	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(wd) }()
	require.NoError(t, os.Chdir(tmpDir))

	gv := createTestGitVolume(tmpDir, tmpDir, filepath.Join(tmpDir, "global"), nil)
	gv.quiet = true

	// Test: Init fails
	err = gv.Init()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find git repository root")
}

func TestInit_BareRepository(t *testing.T) {
	tmpDir := t.TempDir()
	bareDir := filepath.Join(tmpDir, "repo.git")

	cmd := exec.Command("git", "init", "--bare", bareDir)
	require.NoError(t, cmd.Run())

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(wd) }()
	require.NoError(t, os.Chdir(bareDir))

	globalDir := filepath.Join(tmpDir, "global")
	gv := createTestGitVolume(bareDir, bareDir, globalDir, nil)
	gv.quiet = true

	err = gv.Init()
	require.NoError(t, err)

	assert.DirExists(t, globalDir)
	assert.FileExists(t, filepath.Join(bareDir, "git-volume.yaml"))

	content, err := os.ReadFile(filepath.Join(bareDir, "git-volume.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "volumes:")
}

func TestInit_NonQuiet(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.Mkdir(repoDir, 0755))
	cmd := exec.Command("git", "init", repoDir)
	require.NoError(t, cmd.Run())

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(wd) }()
	require.NoError(t, os.Chdir(repoDir))

	globalDir := filepath.Join(tmpDir, "global")
	gv := createTestGitVolume(repoDir, repoDir, globalDir, nil)
	gv.quiet = false // Non-quiet to cover afterInit output branches

	// First init (configCreated branch)
	err = gv.Init()
	require.NoError(t, err)
	assert.DirExists(t, globalDir)
	assert.FileExists(t, filepath.Join(repoDir, "git-volume.yaml"))

	// Second init (configExists branch)
	err = gv.Init()
	require.NoError(t, err)
}

func TestInit_NonQuiet_Error(t *testing.T) {
	tmpDir := t.TempDir()

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(wd) }()
	require.NoError(t, os.Chdir(tmpDir))

	gv := createTestGitVolume(tmpDir, tmpDir, filepath.Join(tmpDir, "global"), nil)
	gv.quiet = false // Non-quiet to cover error branch

	err = gv.Init()
	assert.Error(t, err)
}
