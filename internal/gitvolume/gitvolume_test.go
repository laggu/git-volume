package gitvolume

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// createTestGitVolume creates a GitVolume for testing with pre-configured context
func createTestGitVolume(sourceDir, targetDir, globalDir string, volumes []Volume) *GitVolume {
	// Resolve paths for each volume
	for i := range volumes {
		v := &volumes[i]
		srcBase := sourceDir
		if v.IsGlobal {
			srcBase = globalDir
		}
		v.SourcePath = filepath.Join(srcBase, v.Source)
		v.TargetPath = filepath.Join(targetDir, v.Target)
	}

	return &GitVolume{
		ctx: &Context{
			SourceDir: sourceDir,
			TargetDir: targetDir,
			GlobalDir: globalDir,
			Volumes:   volumes,
		},
		verbose: false,
		quiet:   true,
	}
}

func TestGitVolume_Sync_Link(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	if err := gv.Sync(SyncOptions{}); err != nil {
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

func TestGitVolume_Sync_Copy(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source2.txt", Target: "copy2.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	if err := gv.Sync(SyncOptions{}); err != nil {
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

func TestGitVolume_Unsync(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
		{Source: "source2.txt", Target: "copy2.txt", Mode: ModeCopy},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync first
	gv.Sync(SyncOptions{})

	// Unsync
	if err := gv.Unsync(UnsyncOptions{}); err != nil {
		t.Fatalf("Unsync failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "link1.txt")); !os.IsNotExist(err) {
		t.Errorf("link1.txt should be removed")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "copy2.txt")); !os.IsNotExist(err) {
		t.Errorf("copy2.txt should be removed")
	}
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
	gv.Sync(SyncOptions{})

	// Modify Copy Target
	copyPath := filepath.Join(targetDir, "copy2.txt")
	os.WriteFile(copyPath, []byte("MODIFIED CONTENT"), 0644)

	// Modify Link Target (Replace link with real file)
	linkPath := filepath.Join(targetDir, "link1.txt")
	os.Remove(linkPath)
	os.WriteFile(linkPath, []byte("NOT A LINK"), 0644)

	// Unsync
	gv.Unsync(UnsyncOptions{})

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

func TestGitVolume_Sync_Force(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create existing file at target
	targetPath := filepath.Join(targetDir, "link1.txt")
	os.WriteFile(targetPath, []byte("existing content"), 0644)

	// Try sync without force - should fail
	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink, Force: false},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	err := gv.Sync(SyncOptions{})
	if err == nil {
		t.Error("expected error when target exists and force is false")
	}

	// Sync with force - should succeed
	volumes[0].Force = true
	gv = createTestGitVolume(sourceDir, targetDir, "", volumes)

	if err := gv.Sync(SyncOptions{}); err != nil {
		t.Fatalf("Sync with force failed: %v", err)
	}

	// Verify it's now a symlink
	info, err := os.Lstat(targetPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link1.txt should be symlink after forced sync")
	}
}

func TestGitVolume_Sync_ExistingDirectory(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create existing directory at target
	targetPath := filepath.Join(targetDir, "link1.txt")
	os.Mkdir(targetPath, 0755)
	os.WriteFile(filepath.Join(targetPath, "subfile.txt"), []byte("sub"), 0644)

	// Sync with force - should remove directory and create symlink
	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink, Force: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	if err := gv.Sync(SyncOptions{}); err != nil {
		t.Fatalf("Sync failed to replace directory: %v", err)
	}

	// Verify it's now a symlink
	info, err := os.Lstat(targetPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link1.txt should be symlink after replacing directory")
	}
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
	if err := gv.Sync(SyncOptions{DryRun: true}); err != nil {
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

func TestGitVolume_Unsync_DryRun(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// First sync normally
	gv.Sync(SyncOptions{})

	// Unsync with dry-run
	if err := gv.Unsync(UnsyncOptions{DryRun: true}); err != nil {
		t.Fatalf("Dry-run unsync failed: %v", err)
	}

	// Verify file still exists
	if _, err := os.Stat(filepath.Join(targetDir, "link1.txt")); os.IsNotExist(err) {
		t.Error("link1.txt should still exist after dry-run unsync")
	}
}

func TestGitVolume_Sync_RelativeLinks(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "link1.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	// Sync with relative links
	if err := gv.Sync(SyncOptions{RelativeLinks: true}); err != nil {
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

func TestGitVolume_Sync_NestedTarget(t *testing.T) {
	sourceDir, targetDir, cleanup := setupTestEnv(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "deep/nested/dir/link.txt", Mode: ModeLink},
	}
	gv := createTestGitVolume(sourceDir, targetDir, "", volumes)

	if err := gv.Sync(SyncOptions{}); err != nil {
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

func TestGitVolume_Sync_GlobalSource_Link(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "secrets/prod.key", Target: "config/key", Mode: ModeLink, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	if err := gv.Sync(SyncOptions{}); err != nil {
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

func TestGitVolume_Sync_GlobalSource_Copy(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "global.txt", Target: "copied.txt", Mode: ModeCopy, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	if err := gv.Sync(SyncOptions{}); err != nil {
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

func TestGitVolume_Sync_MixedLocalAndGlobal(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "local.txt", Mode: ModeLink, IsGlobal: false},
		{Source: "global.txt", Target: "global.txt", Mode: ModeLink, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	if err := gv.Sync(SyncOptions{}); err != nil {
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
	if err := gv.Sync(SyncOptions{}); err != nil {
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
	if err := gv.Unsync(UnsyncOptions{}); err != nil {
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

func TestGitVolume_Sync_GlobalSource_NotFound(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "nonexistent.txt", Target: "target.txt", Mode: ModeLink, IsGlobal: true},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	err := gv.Sync(SyncOptions{})
	if err == nil {
		t.Error("expected error for non-existent global source")
	}
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
	if err == nil {
		t.Error("expected error when GlobalDir is empty")
	}
	if err != nil && !strings.Contains(err.Error(), "global directory not configured") {
		t.Errorf("expected 'global directory not configured' error, got: %v", err)
	}
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
	if err == nil {
		t.Error("expected error when GlobalDir is empty")
	}
	if err != nil && !strings.Contains(err.Error(), "global directory not configured") {
		t.Errorf("expected 'global directory not configured' error, got: %v", err)
	}
}

func TestGitVolume_List(t *testing.T) {
	sourceDir, targetDir, globalDir, cleanup := setupTestEnvWithGlobal(t)
	defer cleanup()

	volumes := []Volume{
		{Source: "source1.txt", Target: "local.txt", Mode: ModeLink, IsGlobal: false},
		{Source: "global.txt", Target: "global.txt", Mode: ModeLink, IsGlobal: true},
		{Source: "nonexistent.txt", Target: "missing.txt", Mode: ModeLink, IsGlobal: false},
	}
	gv := createTestGitVolume(sourceDir, targetDir, globalDir, volumes)

	// Before sync
	statuses, err := gv.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(statuses) != 3 {
		t.Errorf("expected 3 statuses, got %d", len(statuses))
	}

	// Check not mounted status
	if statuses[0].Status != StatusNotMounted {
		t.Errorf("expected NOT MOUNTED for local.txt before sync, got %s", statuses[0].Status)
	}

	// Check missing source
	if statuses[2].Status != StatusMissingSource {
		t.Errorf("expected MISSING (Source) for missing.txt, got %s", statuses[2].Status)
	}

	// Sync local and global
	volumes = volumes[:2] // Remove the nonexistent one for sync
	gv.ctx.Volumes = volumes
	gv.Sync(SyncOptions{})

	// After sync
	statuses, _ = gv.List()
	if statuses[0].Status != StatusOKLinked {
		t.Errorf("expected OK (Linked) for local.txt after sync, got %s", statuses[0].Status)
	}
	if statuses[1].Status != StatusOKLinked {
		t.Errorf("expected OK (Linked) for global.txt after sync, got %s", statuses[1].Status)
	}

	// Check display source for global
	if statuses[1].Source != "@global/global.txt" {
		t.Errorf("expected @global/global.txt, got %s", statuses[1].Source)
	}
}
