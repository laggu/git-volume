package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const SampleConfig = `# Optional: custom global directory (default: ~/.git-volume)
# globalDir: ~/.git-volume

volumes:
  # Example: mount a shared env file
  # - ".env.shared:.env"
  #
  # Example: copy a secret (required for Docker builds)
  # - mount: "secrets/prod.key:config/prod.key"
  #   mode: "copy"
  #
  # Example: mount from global directory (@global/ prefix)
  # - "@global/secrets/prod.key:config/key"
`

// GlobalPrefix is the prefix used to reference global directory files
const GlobalPrefix = "@global/"

// DefaultGlobalDir is the default global directory path
const DefaultGlobalDir = "~/.git-volume"

// Constants
const (
	ConfigFileName = "git-volume.yaml"
	ModeLink       = "link"
	ModeCopy       = "copy"

	// File permission constants
	DefaultDirPerm  os.FileMode = 0755
	DefaultFilePerm os.FileMode = 0644
)

// Config represents the top-level structure of git-volume.yaml
type Config struct {
	GlobalDir string   `yaml:"globalDir"` // Custom global directory (default: ~/.git-volume)
	Volumes   []Volume `yaml:"volumes"`
}

// Volume represents a single volume mapping
type Volume struct {
	Source   string `yaml:"-"` // Parsed from Mount or simple string
	Target   string `yaml:"-"` // Parsed from Mount or simple string
	Mode     string `yaml:"mode"`
	Force    bool   `yaml:"force"`
	IsGlobal bool   `yaml:"-"` // True if source uses @global/ prefix
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
// Handles Windows drive letters (e.g., C:\path:target) and @global/ prefix
func (v *Volume) parseMount(mount string) error {
	// Find the separator colon (not a Windows drive letter colon)
	sepIndex := findMountSeparator(mount)
	if sepIndex == -1 {
		return fmt.Errorf("invalid mount format: %s (expected source:target)", mount)
	}

	source := strings.TrimSpace(mount[:sepIndex])
	v.Target = strings.TrimSpace(mount[sepIndex+1:])

	if source == "" || v.Target == "" {
		return fmt.Errorf("invalid mount format: %s (source and target cannot be empty)", mount)
	}

	// Check for @global/ prefix
	if strings.HasPrefix(source, GlobalPrefix) {
		v.IsGlobal = true
		v.Source = strings.TrimPrefix(source, GlobalPrefix)
	} else {
		v.IsGlobal = false
		v.Source = source
	}

	// Note: Mode is set by caller (UnmarshalYAML handles defaults)
	return nil
}

// findMountSeparator finds the colon that separates source and target
// It skips Windows drive letter colons (e.g., C: in C:\path)
func findMountSeparator(mount string) int {
	if runtime.GOOS == "windows" {
		if len(mount) > 1 && mount[1] == ':' {
			isDrive := (mount[0] >= 'a' && mount[0] <= 'z') || (mount[0] >= 'A' && mount[0] <= 'Z')
			if isDrive {
				// It's a drive letter. Look for the next colon after it.
				nextColon := strings.Index(mount[2:], ":")
				if nextColon != -1 {
					return 2 + nextColon
				}
				// No other colon found, so there is no source:target separator.
				return -1
			}
		}
	}
	return strings.Index(mount, ":")
}

// validatePath checks for path traversal attacks and dangerous paths
func validatePath(path string) error {
	// Check for empty path
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for dangerous paths that could cause data loss
	// Block "." (current directory) and ".git" directory
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "." || cleanPath == ".." {
		return fmt.Errorf("path cannot be current or parent directory: %s", path)
	}

	// Block .git directory and its contents (case-insensitive for Windows/macOS)
	lowerPath := strings.ToLower(cleanPath)
	if lowerPath == ".git" || strings.HasPrefix(lowerPath, ".git/") || strings.HasPrefix(lowerPath, ".git\\") {
		return fmt.Errorf("path cannot be inside .git directory: %s", path)
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
func LoadConfig(path string, quiet bool) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Warn if no volumes defined
	if len(cfg.Volumes) == 0 && !quiet {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: no volumes defined in %s\n", path)
	}

	return &cfg, nil
}

// HasGlobalVolumes checks if any volume uses @global/ prefix
func HasGlobalVolumes(volumes []Volume) bool {
	for _, v := range volumes {
		if v.IsGlobal {
			return true
		}
	}
	return false
}

// ResolveGlobalDir resolves the global directory path.
// It handles ~ expansion and returns an absolute path.
// If globalDir is empty, it uses DefaultGlobalDir.
// If override is provided (non-empty), it takes precedence over both.
func ResolveGlobalDir(globalDir, override string) (string, error) {
	// Override takes precedence
	dir := globalDir
	if override != "" {
		dir = override
	}

	// Use default if empty
	if dir == "" {
		dir = DefaultGlobalDir
	}

	// Expand ~ to home directory
	if strings.HasPrefix(dir, "~/") || dir == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		if dir == "~" {
			dir = homeDir
		} else {
			dir = filepath.Join(homeDir, dir[2:])
		}
	}

	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for global directory: %w", err)
	}

	return absDir, nil
}
