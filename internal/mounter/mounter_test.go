package mounter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/laggu/git-volume/internal/config"
)

func setupTestEnv(t *testing.T) (sourceDir, targetDir string, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "git-volume-test")
	if err != nil {
		t.Fatal(err)
	}

	sourceDir = filepath.Join(tmpDir, "source")
	targetDir = filepath.Join(tmpDir, "target")
	os.Mkdir(sourceDir, 0755)
	os.Mkdir(targetDir, 0755)

	// Create source files
	os.WriteFile(filepath.Join(sourceDir, "source1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "source2.txt"), []byte("content2"), 0644)

	cleanup = func() { os.RemoveAll(tmpDir) }
	return
}

func TestMounter_Sync_Link(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: config.ModeLink},
	}

	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	linkPath := filepath.Join(targetDir, "link1.txt")
	info, err := os.Lstat(linkPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link1.txt should be valid symlink")
	}
	target, _ := os.Readlink(linkPath)
	expected := filepath.Join(sourceDir, "source1.txt")
	if target != expected {
		t.Errorf("link target = %s, want %s", target, expected)
	}
}

func TestMounter_Sync_Copy(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)
	volumes := []config.Volume{
		{Source: "source2.txt", Target: "copy2.txt", Mode: config.ModeCopy},
	}

	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	copyPath := filepath.Join(targetDir, "copy2.txt")
	info, err := os.Stat(copyPath)
	if err != nil || !info.Mode().IsRegular() {
		t.Errorf("copy2.txt should be regular file")
	}
	data, _ := os.ReadFile(copyPath)
	if string(data) != "content2" {
		t.Errorf("copy content mismatch")
	}
}

func TestMounter_Unsync(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: config.ModeLink},
		{Source: "source2.txt", Target: "copy2.txt", Mode: config.ModeCopy},
	}

	// Sync first
	m.Sync(volumes, SyncOptions{})

	// Unsync
	if err := m.Unsync(volumes, UnsyncOptions{Quiet: true}); err != nil {
		t.Fatalf("Unsync failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "link1.txt")); !os.IsNotExist(err) {
		t.Errorf("link1.txt should be removed")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "copy2.txt")); !os.IsNotExist(err) {
		t.Errorf("copy2.txt should be removed")
	}
}

func TestMounter_Unsync_SafetyCheck(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: config.ModeLink},
		{Source: "source2.txt", Target: "copy2.txt", Mode: config.ModeCopy},
	}

	// Sync
	m.Sync(volumes, SyncOptions{})

	// Modify Copy Target
	copyPath := filepath.Join(targetDir, "copy2.txt")
	os.WriteFile(copyPath, []byte("MODIFIED CONTENT"), 0644)

	// Modify Link Target (Replace link with real file)
	linkPath := filepath.Join(targetDir, "link1.txt")
	os.Remove(linkPath)
	os.WriteFile(linkPath, []byte("NOT A LINK"), 0644)

	// Unsync
	m.Unsync(volumes, UnsyncOptions{Quiet: true})

	// Verify Copy was Preserved (Skipped)
	data, _ := os.ReadFile(copyPath)
	if string(data) != "MODIFIED CONTENT" {
		t.Errorf("Modified copy file was deleted! Unsync failed safety check.")
	}

	// Verify Link-replaced-file was Preserved (Skipped)
	dataLink, _ := os.ReadFile(linkPath)
	if string(dataLink) != "NOT A LINK" {
		t.Errorf("Replaced link file was deleted! Unsync failed safety check.")
	}
}

func TestMounter_Sync_Force(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)

	// Create existing file at target
	targetPath := filepath.Join(targetDir, "link1.txt")
	os.WriteFile(targetPath, []byte("existing content"), 0644)

	// Try sync without force - should fail
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: config.ModeLink, Force: false},
	}
	err := m.Sync(volumes, SyncOptions{})
	if err == nil {
		t.Error("expected error when target exists and force is false")
	}

	// Sync with force - should succeed
	volumes[0].Force = true
	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync with force failed: %v", err)
	}

	// Verify it's now a symlink
	info, err := os.Lstat(targetPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link1.txt should be symlink after forced sync")
	}
}

func TestMounter_Sync_ExistingDirectory(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)

	// Create existing directory at target
	targetPath := filepath.Join(targetDir, "link1.txt")
	os.Mkdir(targetPath, 0755)
	os.WriteFile(filepath.Join(targetPath, "subfile.txt"), []byte("sub"), 0644)

	// Sync with force - should remove directory and create symlink
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: config.ModeLink, Force: true},
	}
	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync failed to replace directory: %v", err)
	}

	// Verify it's now a symlink
	info, err := os.Lstat(targetPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link1.txt should be symlink after replacing directory")
	}
}

func TestMounter_Sync_DryRun(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: config.ModeLink},
		{Source: "source2.txt", Target: "copy2.txt", Mode: config.ModeCopy},
	}

	// Sync with dry-run
	if err := m.Sync(volumes, SyncOptions{DryRun: true}); err != nil {
		t.Fatalf("Dry-run sync failed: %v", err)
	}

	// Verify nothing was created
	if _, err := os.Stat(filepath.Join(targetDir, "link1.txt")); !os.IsNotExist(err) {
		t.Error("link1.txt should not exist in dry-run mode")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "copy2.txt")); !os.IsNotExist(err) {
		t.Error("copy2.txt should not exist in dry-run mode")
	}
}

func TestMounter_Unsync_DryRun(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: config.ModeLink},
	}

	// First sync normally
	m.Sync(volumes, SyncOptions{})

	// Unsync with dry-run
	if err := m.Unsync(volumes, UnsyncOptions{DryRun: true}); err != nil {
		t.Fatalf("Dry-run unsync failed: %v", err)
	}

	// Verify file still exists
	if _, err := os.Stat(filepath.Join(targetDir, "link1.txt")); os.IsNotExist(err) {
		t.Error("link1.txt should still exist after dry-run unsync")
	}
}

func TestMounter_Sync_RelativeLinks(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: config.ModeLink},
	}

	// Sync with relative links
	if err := m.Sync(volumes, SyncOptions{RelativeLinks: true}); err != nil {
		t.Fatalf("Sync with relative links failed: %v", err)
	}

	linkPath := filepath.Join(targetDir, "link1.txt")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Failed to read link: %v", err)
	}

	// Relative link should not be absolute path
	if filepath.IsAbs(target) {
		t.Errorf("link target should be relative, got absolute: %s", target)
	}

	// Verify the link still works (can read through it)
	content, err := os.ReadFile(linkPath)
	if err != nil {
		t.Errorf("Failed to read through relative symlink: %v", err)
	}
	if string(content) != "content1" {
		t.Errorf("content mismatch through relative symlink")
	}
}

func TestMounter_Sync_NestedTarget(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "deep/nested/dir/link.txt", Mode: config.ModeLink},
	}

	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	linkPath := filepath.Join(targetDir, "deep/nested/dir/link.txt")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Failed to stat nested link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("nested target should be symlink")
	}
}
