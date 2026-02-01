package mounter

import (
	"os"
	"path/filepath"
	"strings"
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

func setupTestEnvWithGlobal(t *testing.T) (sourceDir, targetDir, globalDir string, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "git-volume-test")
	if err != nil {
		t.Fatal(err)
	}

	sourceDir = filepath.Join(tmpDir, "source")
	targetDir = filepath.Join(tmpDir, "target")
	globalDir = filepath.Join(tmpDir, "global")
	os.Mkdir(sourceDir, 0755)
	os.Mkdir(targetDir, 0755)
	os.Mkdir(globalDir, 0755)

	// Create source files
	os.WriteFile(filepath.Join(sourceDir, "source1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "source2.txt"), []byte("content2"), 0644)

	// Create global files
	os.MkdirAll(filepath.Join(globalDir, "secrets"), 0755)
	os.WriteFile(filepath.Join(globalDir, "secrets", "prod.key"), []byte("global-secret"), 0644)
	os.WriteFile(filepath.Join(globalDir, "global.txt"), []byte("global-content"), 0644)

	cleanup = func() { os.RemoveAll(tmpDir) }
	return
}

func TestMounter_Sync_Link(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	m := New(sourceDir, targetDir, "")
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

	m := New(sourceDir, targetDir, "")
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

	m := New(sourceDir, targetDir, "")
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

	m := New(sourceDir, targetDir, "")
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

	m := New(sourceDir, targetDir, "")

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

	m := New(sourceDir, targetDir, "")

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

	m := New(sourceDir, targetDir, "")
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

	m := New(sourceDir, targetDir, "")
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

	m := New(sourceDir, targetDir, "")
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

	m := New(sourceDir, targetDir, "")
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

func TestMounter_Sync_GlobalSource_Link(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	m := New(sourceDir, targetDir, globalDir)
	volumes := []config.Volume{
		{Source: "secrets/prod.key", Target: "config/key", Mode: config.ModeLink, IsGlobal: true},
	}

	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	linkPath := filepath.Join(targetDir, "config/key")
	info, err := os.Lstat(linkPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("config/key should be valid symlink")
	}

	// Verify link points to global directory
	target, _ := os.Readlink(linkPath)
	expected := filepath.Join(globalDir, "secrets", "prod.key")
	if target != expected {
		t.Errorf("link target = %s, want %s", target, expected)
	}

	// Verify content is readable
	content, err := os.ReadFile(linkPath)
	if err != nil {
		t.Errorf("Failed to read through symlink: %v", err)
	}
	if string(content) != "global-secret" {
		t.Errorf("content = %q, want %q", string(content), "global-secret")
	}
}

func TestMounter_Sync_GlobalSource_Copy(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	m := New(sourceDir, targetDir, globalDir)
	volumes := []config.Volume{
		{Source: "global.txt", Target: "copied.txt", Mode: config.ModeCopy, IsGlobal: true},
	}

	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	copyPath := filepath.Join(targetDir, "copied.txt")
	info, err := os.Stat(copyPath)
	if err != nil || !info.Mode().IsRegular() {
		t.Errorf("copied.txt should be regular file")
	}

	data, _ := os.ReadFile(copyPath)
	if string(data) != "global-content" {
		t.Errorf("content = %q, want %q", string(data), "global-content")
	}
}

func TestMounter_Sync_MixedLocalAndGlobal(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	m := New(sourceDir, targetDir, globalDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "local.txt", Mode: config.ModeLink, IsGlobal: false},
		{Source: "global.txt", Target: "global.txt", Mode: config.ModeLink, IsGlobal: true},
	}

	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify local link points to source directory
	localPath := filepath.Join(targetDir, "local.txt")
	localTarget, _ := os.Readlink(localPath)
	if localTarget != filepath.Join(sourceDir, "source1.txt") {
		t.Errorf("local link target = %s, want %s", localTarget, filepath.Join(sourceDir, "source1.txt"))
	}

	// Verify global link points to global directory
	globalPath := filepath.Join(targetDir, "global.txt")
	globalTarget, _ := os.Readlink(globalPath)
	if globalTarget != filepath.Join(globalDir, "global.txt") {
		t.Errorf("global link target = %s, want %s", globalTarget, filepath.Join(globalDir, "global.txt"))
	}
}

func TestMounter_Unsync_GlobalSource(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	m := New(sourceDir, targetDir, globalDir)
	volumes := []config.Volume{
		{Source: "source1.txt", Target: "local.txt", Mode: config.ModeLink, IsGlobal: false},
		{Source: "global.txt", Target: "global.txt", Mode: config.ModeLink, IsGlobal: true},
		{Source: "secrets/prod.key", Target: "config/key", Mode: config.ModeCopy, IsGlobal: true},
	}

	// Sync first
	if err := m.Sync(volumes, SyncOptions{}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify all mounted
	if _, err := os.Lstat(filepath.Join(targetDir, "local.txt")); err != nil {
		t.Error("local.txt should exist after sync")
	}
	if _, err := os.Lstat(filepath.Join(targetDir, "global.txt")); err != nil {
		t.Error("global.txt should exist after sync")
	}
	if _, err := os.Lstat(filepath.Join(targetDir, "config/key")); err != nil {
		t.Error("config/key should exist after sync")
	}

	// Unsync
	if err := m.Unsync(volumes, UnsyncOptions{Quiet: true}); err != nil {
		t.Fatalf("Unsync failed: %v", err)
	}

	// Verify all removed
	if _, err := os.Stat(filepath.Join(targetDir, "local.txt")); !os.IsNotExist(err) {
		t.Errorf("local.txt should be removed")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "global.txt")); !os.IsNotExist(err) {
		t.Errorf("global.txt should be removed")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "config/key")); !os.IsNotExist(err) {
		t.Errorf("config/key should be removed")
	}
}

func TestMounter_Sync_GlobalSource_NotFound(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	m := New(sourceDir, targetDir, globalDir)
	volumes := []config.Volume{
		{Source: "nonexistent.txt", Target: "target.txt", Mode: config.ModeLink, IsGlobal: true},
	}

	err := m.Sync(volumes, SyncOptions{})
	if err == nil {
		t.Error("expected error for non-existent global source")
	}
}

func TestMounter_Sync_GlobalSource_EmptyGlobalBase(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// GlobalBase is empty string
	m := New(sourceDir, targetDir, "")
	volumes := []config.Volume{
		{Source: "secrets/key", Target: "config/key", Mode: config.ModeLink, IsGlobal: true},
	}

	err := m.Sync(volumes, SyncOptions{})
	if err == nil {
		t.Error("expected error when GlobalBase is empty")
	}
	if err != nil && !strings.Contains(err.Error(), "global directory not configured") {
		t.Errorf("expected 'global directory not configured' error, got: %v", err)
	}
}

func TestMounter_Unsync_GlobalSource_EmptyGlobalBase(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// GlobalBase is empty string
	m := New(sourceDir, targetDir, "")
	volumes := []config.Volume{
		{Source: "secrets/key", Target: "config/key", Mode: config.ModeLink, IsGlobal: true},
	}

	err := m.Unsync(volumes, UnsyncOptions{})
	if err == nil {
		t.Error("expected error when GlobalBase is empty")
	}
	if err != nil && !strings.Contains(err.Error(), "global directory not configured") {
		t.Errorf("expected 'global directory not configured' error, got: %v", err)
	}
}
