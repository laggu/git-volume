package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const SampleConfig = `volumes:
  # Example: mount a shared env file
  # - ".env.shared:.env"
  #
  # Example: copy a secret (required for Docker builds)
  # - mount: "secrets/prod.key:config/prod.key"
  #   mode: "copy"
`

// Constants
const (
	ConfigFileName = "git-volume.yaml"
	ModeLink       = "link"
	ModeCopy       = "copy"

	// File permission constants
	DefaultDirPerm  os.FileMode = 0755
	DefaultFilePerm os.FileMode = 0644
	SecretFilePerm  os.FileMode = 0600
)

// Config represents the top-level structure of git-volume.yaml
type Config struct {
	Volumes []Volume `yaml:"volumes"`
}

// Volume represents a single volume mapping
type Volume struct {
	Source string `yaml:"-"` // Parsed from Mount or simple string
	Target string `yaml:"-"` // Parsed from Mount or simple string
	Mode   string `yaml:"mode"`
	Force  bool   `yaml:"force"`
}

// rawVolume is used for intermediate parsing
type rawVolume struct {
	Mount string `yaml:"mount"`
	Mode  string `yaml:"mode"`
	Force bool   `yaml:"force"`
}

// UnmarshalYAML implements custom parsing to support both string "source:target"
// and object-based syntax with "mount" field.
func (v *Volume) UnmarshalYAML(value *yaml.Node) error {
	// Case 1: Simple string format "source:target"
	if value.Kind == yaml.ScalarNode {
		if err := v.parseMount(value.Value); err != nil {
			return err
		}
		v.Mode = ModeLink // Default for simple string format
		// Validate paths for security
		if err := validatePath(v.Source); err != nil {
			return fmt.Errorf("invalid source path: %w", err)
		}
		if err := validatePath(v.Target); err != nil {
			return fmt.Errorf("invalid target path: %w", err)
		}
		return nil
	}

	// Case 2: Object format with "mount" field
	var raw rawVolume
	if err := value.Decode(&raw); err != nil {
		return err
	}

	if raw.Mount == "" {
		return fmt.Errorf("missing 'mount' field in volume definition")
	}

	if err := v.parseMount(raw.Mount); err != nil {
		return err
	}

	v.Mode = raw.Mode
	v.Force = raw.Force

	// Set default mode if not specified
	if v.Mode == "" {
		v.Mode = ModeLink
	}

	// Validate mode
	if v.Mode != ModeLink && v.Mode != ModeCopy {
		return fmt.Errorf("invalid mode: %s (allowed: link, copy)", v.Mode)
	}

	// Validate paths for security
	if err := validatePath(v.Source); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := validatePath(v.Target); err != nil {
		return fmt.Errorf("invalid target path: %w", err)
	}

	return nil
}

// parseMount extracts source and target from "source:target" format
// Handles Windows drive letters (e.g., C:\path:target)
func (v *Volume) parseMount(mount string) error {
	// Find the separator colon (not a Windows drive letter colon)
	sepIndex := findMountSeparator(mount)
	if sepIndex == -1 {
		return fmt.Errorf("invalid mount format: %s (expected source:target)", mount)
	}

	v.Source = strings.TrimSpace(mount[:sepIndex])
	v.Target = strings.TrimSpace(mount[sepIndex+1:])

	if v.Source == "" || v.Target == "" {
		return fmt.Errorf("invalid mount format: %s (source and target cannot be empty)", mount)
	}

	// Note: Mode is set by caller (UnmarshalYAML handles defaults)
	return nil
}

// findMountSeparator finds the colon that separates source and target
// It skips Windows drive letter colons (e.g., C: in C:\path)
func findMountSeparator(mount string) int {
	for i := 0; i < len(mount); i++ {
		if mount[i] == ':' {
			// On Windows, skip drive letter colon (e.g., "C:" at position 1)
			if runtime.GOOS == "windows" && i == 1 && len(mount) > 2 {
				// This is likely a drive letter, continue looking
				continue
			}
			return i
		}
	}
	return -1
}

// validatePath checks for path traversal attacks
func validatePath(path string) error {
	// Check for empty path
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for absolute paths (should be relative)
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return fmt.Errorf("path must be relative, not absolute: %s", path)
	}

	// Check for Windows absolute paths (e.g., C:\path)
	if runtime.GOOS == "windows" && len(path) >= 2 && path[1] == ':' {
		return fmt.Errorf("path must be relative, not absolute: %s", path)
	}

	// Check for path traversal attempts
	// Split by both forward and backward slashes
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	})

	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("path traversal not allowed: %s", path)
		}
	}

	return nil
}

// LoadConfig reads and parses the configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Warn if no volumes defined
	if len(cfg.Volumes) == 0 {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: no volumes defined in %s\n", path)
	}

	return &cfg, nil
}
