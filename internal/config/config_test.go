package config

import (
	"os"
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

			// Run LoadConfig
			cfg, err := LoadConfig(tmpFile.Name(), false)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Verify results
			if len(cfg.Volumes) != len(tt.want) {
				t.Errorf("got %d volumes, want %d", len(cfg.Volumes), len(tt.want))
				return
			}

			for i, v := range cfg.Volumes {
				if v != tt.want[i] {
					t.Errorf("volume[%d] = %+v, want %+v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGlobalDirParsing(t *testing.T) {
	tests := []struct {
		name          string
		yamlData      string
		wantGlobalDir string
	}{
		{
			name: "Default GlobalDir (empty)",
			yamlData: `
volumes:
  - ".env:.env"
`,
			wantGlobalDir: "",
		},
		{
			name: "Custom GlobalDir",
			yamlData: `
globalDir: ~/.my-secrets
volumes:
  - "@global/key:config/key"
`,
			wantGlobalDir: "~/.my-secrets",
		},
		{
			name: "Absolute GlobalDir",
			yamlData: `
globalDir: /opt/secrets
volumes:
  - "@global/key:config/key"
`,
			wantGlobalDir: "/opt/secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "git-volume-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.yamlData); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			cfg, err := LoadConfig(tmpFile.Name(), true)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if cfg.GlobalDir != tt.wantGlobalDir {
				t.Errorf("GlobalDir = %q, want %q", cfg.GlobalDir, tt.wantGlobalDir)
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
			got, err := ResolveGlobalDir(tt.globalDir, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveGlobalDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Handle absolute path comparison
				if !strings.HasPrefix(tt.want, "/") {
					// For relative expectations, just check the result is absolute
					if !filepath.IsAbs(got) {
						t.Errorf("ResolveGlobalDir() = %q, expected absolute path", got)
					}
				} else {
					if got != tt.want {
						t.Errorf("ResolveGlobalDir() = %q, want %q", got, tt.want)
					}
				}
			}
		})
	}
}
