package gitvolume

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)
			defer func() { _ = os.Remove(tmpFile.Name()) }()

			_, err = tmpFile.WriteString(tt.yamlData)
			require.NoError(t, err)
			_ = tmpFile.Close()

			// Run loadConfig
			cfg, err := loadConfig(tmpFile.Name(), true)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify results
			assert.Equal(t, len(tt.want), len(cfg.Volumes))

			for i, v := range cfg.Volumes {
				assert.Equal(t, tt.want[i], v)
			}
		})
	}
}

func TestResolveGlobalDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	got, err := resolveGlobalDir()
	assert.NoError(t, err)

	want := filepath.Join(homeDir, ".git-volume")
	assert.Equal(t, want, got)
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
	require.NoError(t, err)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err, "failed to init git repo")

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cleanup = func() { _ = os.RemoveAll(tmpDir) }
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
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create source file
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "source.txt"), []byte("content"), 0644))

	// Change to repo dir
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	require.NoError(t, os.Chdir(repoDir))

	// Create context and load config
	ctx, err := NewContext()
	require.NoError(t, err)
	require.NoError(t, ctx.Load("", true))

	assert.Equal(t, repoDir, resolvePath(ctx.SourceDir))
	assert.Equal(t, repoDir, resolvePath(ctx.TargetDir))
	assert.Equal(t, 1, len(ctx.Volumes))
}

func TestNewWorkspace_CustomPath(t *testing.T) {
	repoDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Create config file in subdirectory
	customDir := filepath.Join(repoDir, "configs")
	require.NoError(t, os.Mkdir(customDir, 0755))
	configContent := `volumes:
  - "data.txt:output.txt"
`
	customConfigPath := filepath.Join(customDir, "custom.yaml")
	require.NoError(t, os.WriteFile(customConfigPath, []byte(configContent), 0644))

	// Create source file
	require.NoError(t, os.WriteFile(filepath.Join(customDir, "data.txt"), []byte("data"), 0644))

	// Change to repo dir
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	require.NoError(t, os.Chdir(repoDir))

	// Create context with custom path
	ctx, err := NewContext()
	require.NoError(t, err)
	require.NoError(t, ctx.Load(customConfigPath, true))

	assert.Equal(t, customDir, ctx.SourceDir)
	assert.Equal(t, 1, len(ctx.Volumes))
	assert.Equal(t, "data.txt", ctx.Volumes[0].Source)
}

func TestNewWorkspace_NoConfig(t *testing.T) {
	repoDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Change to repo dir
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	require.NoError(t, os.Chdir(repoDir))

	// Create context and load should fail
	ctx, err := NewContext()
	require.NoError(t, err)
	assert.Error(t, ctx.Load("", true))
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
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Change to repo dir
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	require.NoError(t, os.Chdir(repoDir))

	// Create context with relative path
	ctx, err := NewContext()
	require.NoError(t, err)
	require.NoError(t, ctx.Load("my-config.yaml", true))

	assert.Equal(t, repoDir, resolvePath(ctx.SourceDir))
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
			assert.Equal(t, tt.want, ctx.HasGlobalVolumes())
		})
	}
}
