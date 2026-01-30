package config

import (
	"os"
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
			cfg, err := LoadConfig(tmpFile.Name())
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
