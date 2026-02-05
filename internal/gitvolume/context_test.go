package gitvolume

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		want     []Volume
		wantErr  bool
	}{
		{
			name: "Simple String Format",
			yamlData: `
volumes:
  - ".env.shared:.env"
  - "configs/dev.json:src/config.json"
`,
			want: []Volume{
				{Source: ".env.shared", Target: ".env", Mode: ModeLink},
				{Source: "configs/dev.json", Target: "src/config.json", Mode: ModeLink},
			},
			wantErr: false,
		},
		{
			name: "Object Format with Mount",
			yamlData: `
volumes:
  - mount: "secrets/prod.key:app.key"
    mode: "copy"
    force: true
  - mount: "configs/base.yaml:config.yaml"
`,
			want: []Volume{
				{Source: "secrets/prod.key", Target: "app.key", Mode: ModeCopy, Force: true},
				{Source: "configs/base.yaml", Target: "config.yaml", Mode: ModeLink, Force: false},
			},
			wantErr: false,
		},
		{
			name: "Mixed Format",
			yamlData: `
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:app.key"
    mode: "copy"
`,
			want: []Volume{
				{Source: ".env.shared", Target: ".env", Mode: ModeLink},
				{Source: "secrets/prod.key", Target: "app.key", Mode: ModeCopy},
			},
			wantErr: false,
		},
		{
			name: "Invalid Simple Format",
			yamlData: `
volumes:
  - "invalid-format-without-separator"
`,
			wantErr: true,
		},
		{
			name: "Invalid Mode",
			yamlData: `
volumes:
  - mount: "a:b"
    mode: "magic"
`,
			wantErr: true,
		},
		{
			name: "Missing Mount Field",
			yamlData: `
volumes:
  - mode: "copy"
`,
			wantErr: true,
		},
		{
			name: "Path Traversal in Source",
			yamlData: `
volumes:
  - "../../../etc/passwd:.env"
`,
			wantErr: true,
		},
		{
			name: "Path Traversal in Target",
			yamlData: `
volumes:
  - ".env:../../../tmp/evil"
`,
			wantErr: true,
		},
		{
			name: "Absolute Path Not Allowed",
			yamlData: `
volumes:
  - "/etc/passwd:.env"
`,
			wantErr: true,
		},
		{
			name: "Global Prefix Simple Format",
			yamlData: `
volumes:
  - "@global/secrets/prod.key:config/key"
  - "@global/certs/ca.pem:certs/ca.pem"
`,
			want: []Volume{
				{Source: "secrets/prod.key", Target: "config/key", Mode: ModeLink, IsGlobal: true},
				{Source: "certs/ca.pem", Target: "certs/ca.pem", Mode: ModeLink, IsGlobal: true},
			},
			wantErr: false,
		},
		{
			name: "Global Prefix Object Format",
			yamlData: `
volumes:
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
    force: true
`,
			want: []Volume{
				{Source: "secrets/prod.key", Target: "config/key", Mode: ModeCopy, Force: true, IsGlobal: true},
			},
			wantErr: false,
		},
		{
			name: "Mixed Local and Global Sources",
			yamlData: `
volumes:
  - ".env.shared:.env"
  - "@global/secrets/prod.key:config/key"
  - mount: "local/config.json:app/config.json"
    mode: "copy"
`,
			want: []Volume{
				{Source: ".env.shared", Target: ".env", Mode: ModeLink, IsGlobal: false},
				{Source: "secrets/prod.key", Target: "config/key", Mode: ModeLink, IsGlobal: true},
				{Source: "local/config.json", Target: "app/config.json", Mode: ModeCopy, IsGlobal: false},
			},
			wantErr: false,
		},
		{
			name: "Global Prefix Path Traversal Not Allowed",
			yamlData: `
volumes:
  - "@global/../../../etc/passwd:config/key"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write temp file
			tmpFile, err := os.CreateTemp("", "git-volume-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.yamlData); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Run loadConfig
			volumes, err := loadConfig(tmpFile.Name(), true)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Verify results
			if len(volumes) != len(tt.want) {
				t.Errorf("got %d volumes, want %d", len(volumes), len(tt.want))
				return
			}

			for i, v := range volumes {
				if v != tt.want[i] {
					t.Errorf("volume[%d] = %+v, want %+v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestResolveGlobalDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name      string
		globalDir string
		override  string
		want      string
		wantErr   bool
	}{
		{
			name:      "Default (empty) resolves to ~/.git-volume",
			globalDir: "",
			override:  "",
			want:      filepath.Join(homeDir, ".git-volume"),
			wantErr:   false,
		},
		{
			name:      "Tilde expansion",
			globalDir: "~/.my-secrets",
			override:  "",
			want:      filepath.Join(homeDir, ".my-secrets"),
			wantErr:   false,
		},
		{
			name:      "Override takes precedence",
			globalDir: "~/.config-value",
			override:  "~/.override-value",
			want:      filepath.Join(homeDir, ".override-value"),
			wantErr:   false,
		},
		{
			name:      "Override with absolute path",
			globalDir: "~/.config-value",
			override:  "/opt/secrets",
			want:      "/opt/secrets",
			wantErr:   false,
		},
		{
			name:      "Tilde only",
			globalDir: "~",
			override:  "",
			want:      homeDir,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveGlobalDir(tt.globalDir, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveGlobalDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Handle absolute path comparison
				if !strings.HasPrefix(tt.want, "/") {
					// For relative expectations, just check the result is absolute
					if !filepath.IsAbs(got) {
						t.Errorf("resolveGlobalDir() = %q, expected absolute path", got)
					}
				} else {
					if got != tt.want {
						t.Errorf("resolveGlobalDir() = %q, want %q", got, tt.want)
					}
				}
			}
		})
	}
}

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

func TestNewWorkspace_LocalConfig(t *testing.T) {
	repoDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Resolve symlinks for comparison (handles /var -> /private/var on macOS)
	repoDir = resolvePath(repoDir)

	// Create config file
	configContent := `volumes:
  - "source.txt:target.txt"
`
	configPath := filepath.Join(repoDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create source file
	os.WriteFile(filepath.Join(repoDir, "source.txt"), []byte("content"), 0644)

	// Change to repo dir
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(repoDir)

	// Create context and load config
	ctx, err := NewContext("")
	if err != nil {
		t.Fatalf("NewContext failed: %v", err)
	}
	if err := ctx.Load("", true); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if resolvePath(ctx.SourceDir) != repoDir {
		t.Errorf("SourceDir = %s, want %s", ctx.SourceDir, repoDir)
	}
	if resolvePath(ctx.TargetDir) != repoDir {
		t.Errorf("TargetDir = %s, want %s", ctx.TargetDir, repoDir)
	}
	if len(ctx.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(ctx.Volumes))
	}
}

func TestNewWorkspace_CustomPath(t *testing.T) {
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

	// Create context with custom path
	ctx, err := NewContext("")
	if err != nil {
		t.Fatalf("NewContext failed: %v", err)
	}
	if err := ctx.Load(customConfigPath, true); err != nil {
		t.Fatalf("Load with custom path failed: %v", err)
	}

	if ctx.SourceDir != customDir {
		t.Errorf("SourceDir = %s, want %s", ctx.SourceDir, customDir)
	}
	if len(ctx.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(ctx.Volumes))
	}
	if ctx.Volumes[0].Source != "data.txt" {
		t.Errorf("Source = %s, want data.txt", ctx.Volumes[0].Source)
	}
}

func TestNewWorkspace_NoConfig(t *testing.T) {
	repoDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Change to repo dir (no config file)
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(repoDir)

	// Create context and load should fail
	ctx, err := NewContext("")
	if err != nil {
		t.Fatalf("NewContext should not fail: %v", err)
	}
	if err := ctx.Load("", true); err == nil {
		t.Error("expected error when no config file exists")
	}
}

func TestNewWorkspace_RelativeCustomPath(t *testing.T) {
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

	// Create context with relative path
	ctx, err := NewContext("")
	if err != nil {
		t.Fatalf("NewContext failed: %v", err)
	}
	if err := ctx.Load("my-config.yaml", true); err != nil {
		t.Fatalf("Load with relative path failed: %v", err)
	}

	if resolvePath(ctx.SourceDir) != repoDir {
		t.Errorf("SourceDir = %s, want %s", ctx.SourceDir, repoDir)
	}
}

func TestHasGlobalVolumes(t *testing.T) {
	tests := []struct {
		name    string
		volumes []Volume
		want    bool
	}{
		{
			name:    "No volumes",
			volumes: []Volume{},
			want:    false,
		},
		{
			name: "Only local volumes",
			volumes: []Volume{
				{Source: "a", Target: "b", IsGlobal: false},
				{Source: "c", Target: "d", IsGlobal: false},
			},
			want: false,
		},
		{
			name: "Has global volume",
			volumes: []Volume{
				{Source: "a", Target: "b", IsGlobal: false},
				{Source: "c", Target: "d", IsGlobal: true},
			},
			want: true,
		},
		{
			name: "All global volumes",
			volumes: []Volume{
				{Source: "a", Target: "b", IsGlobal: true},
				{Source: "c", Target: "d", IsGlobal: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{Volumes: tt.volumes}
			if got := ctx.HasGlobalVolumes(); got != tt.want {
				t.Errorf("HasGlobalVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}
