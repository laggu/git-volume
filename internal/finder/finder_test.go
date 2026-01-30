package finder

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/laggu/git-volume/internal/config"
)

// resolvePath resolves symlinks to get the real path (handles /var -> /private/var on macOS)
func resolvePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func setupTestGitRepo(t *testing.T) (repoDir string, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "git-volume-finder-test")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal("failed to init git repo:", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	os.WriteFile(testFile, []byte("test"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	cmd.Run()

	cleanup = func() { os.RemoveAll(tmpDir) }
	return tmpDir, cleanup
}

func TestFindContext_LocalConfig(t *testing.T) {
	repoDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Resolve symlinks for comparison (handles /var -> /private/var on macOS)
	repoDir = resolvePath(repoDir)

	// Create config file
	configContent := `volumes:
  - "source.txt:target.txt"
`
	configPath := filepath.Join(repoDir, config.ConfigFileName)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create source file
	os.WriteFile(filepath.Join(repoDir, "source.txt"), []byte("content"), 0644)

	// Change to repo dir
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(repoDir)

	// Find context
	ctx, err := FindContext("")
	if err != nil {
		t.Fatalf("FindContext failed: %v", err)
	}

	if resolvePath(ctx.SourceDir) != repoDir {
		t.Errorf("SourceDir = %s, want %s", ctx.SourceDir, repoDir)
	}
	if resolvePath(ctx.TargetDir) != repoDir {
		t.Errorf("TargetDir = %s, want %s", ctx.TargetDir, repoDir)
	}
	if len(ctx.Config.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(ctx.Config.Volumes))
	}
}

func TestFindContext_CustomPath(t *testing.T) {
	repoDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Create config file in subdirectory
	customDir := filepath.Join(repoDir, "configs")
	os.Mkdir(customDir, 0755)
	configContent := `volumes:
  - "data.txt:output.txt"
`
	customConfigPath := filepath.Join(customDir, "custom.yaml")
	if err := os.WriteFile(customConfigPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create source file
	os.WriteFile(filepath.Join(customDir, "data.txt"), []byte("data"), 0644)

	// Change to repo dir
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(repoDir)

	// Find context with custom path
	ctx, err := FindContext(customConfigPath)
	if err != nil {
		t.Fatalf("FindContext with custom path failed: %v", err)
	}

	if ctx.SourceDir != customDir {
		t.Errorf("SourceDir = %s, want %s", ctx.SourceDir, customDir)
	}
	if len(ctx.Config.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(ctx.Config.Volumes))
	}
	if ctx.Config.Volumes[0].Source != "data.txt" {
		t.Errorf("Source = %s, want data.txt", ctx.Config.Volumes[0].Source)
	}
}

func TestFindContext_NoConfig(t *testing.T) {
	repoDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Change to repo dir (no config file)
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(repoDir)

	// Find context should fail
	_, err := FindContext("")
	if err == nil {
		t.Error("expected error when no config file exists")
	}
}

func TestFindContext_RelativeCustomPath(t *testing.T) {
	repoDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Resolve symlinks for comparison (handles /var -> /private/var on macOS)
	repoDir = resolvePath(repoDir)

	// Create config file
	configContent := `volumes:
  - "src.txt:dst.txt"
`
	configPath := filepath.Join(repoDir, "my-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to repo dir
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(repoDir)

	// Find context with relative path
	ctx, err := FindContext("my-config.yaml")
	if err != nil {
		t.Fatalf("FindContext with relative path failed: %v", err)
	}

	if resolvePath(ctx.SourceDir) != repoDir {
		t.Errorf("SourceDir = %s, want %s", ctx.SourceDir, repoDir)
	}
}
