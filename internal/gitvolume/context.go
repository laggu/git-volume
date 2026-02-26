package gitvolume

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Constants
const (
	ModeLink        = "link"
	ModeCopy        = "copy"
	ConfigFileName  = "git-volume.yaml"
	GlobalPrefix    = "@global/"
	GlobalDirectory = "~/.git-volume"
	DefaultDirPerm  = 0755
	DefaultFilePerm = 0644
)

// SampleConfig is the sample configuration for init command
const SampleConfig = `volumes:
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

// =============================================================================
// Volume
// =============================================================================

// Volume represents a single volume mapping
type Volume struct {
	Source     string // Source path (relative, for display)
	Target     string // Target path (relative)
	SourcePath string // Resolved absolute source path
	TargetPath string // Resolved absolute target path
	Mode       string // "link" or "copy"
	IsGlobal   bool   // True if source uses @global/ prefix
}

// DisplaySource returns the display name for the source path
// (with @global/ prefix for global volumes)
func (v *Volume) DisplaySource() string {
	if v.IsGlobal {
		return GlobalPrefix + v.Source
	}
	return v.Source
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
	var raw struct {
		Mount string `yaml:"mount"`
		Mode  string `yaml:"mode"`
	}
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

// VolumeStatus represents the status of a single volume
type VolumeStatus struct {
	Source string // Display source path (includes @global/ prefix if applicable)
	Target string
	Mode   string
	Status string
}

// Status constants
const (
	StatusOKLinked      = "OK (Linked)"
	StatusOKCopied      = "OK (Copied)"
	StatusModified      = "MODIFIED"
	StatusNotMounted    = "NOT MOUNTED"
	StatusMissingSource = "MISSING (Source)"
	StatusWrongLink     = "WRONG LINK"
	StatusExistsNotLink = "EXISTS (Not Link)"
	StatusExistsNotFile = "EXISTS (Not File)"
	StatusError         = "ERROR"
)

// CheckStatus checks the mount status of the volume
// Uses the pre-resolved SourcePath and TargetPath
func (v *Volume) CheckStatus() VolumeStatus {
	displaySource := v.DisplaySource()

	// Check if source exists
	if _, err := os.Stat(v.SourcePath); os.IsNotExist(err) {
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusMissingSource}
	}

	// Check if target exists
	info, err := os.Lstat(v.TargetPath)
	if os.IsNotExist(err) {
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusNotMounted}
	}
	if err != nil {
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusError}
	}

	// Check status by mode
	if v.Mode == ModeLink {
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(v.TargetPath)
			if err != nil {
				return VolumeStatus{displaySource, v.Target, v.Mode, StatusError}
			}
			// Resolve relative symlink based on symlink's parent directory
			if !filepath.IsAbs(link) {
				link = filepath.Join(filepath.Dir(v.TargetPath), link)
			}
			if pathsEqual(link, v.SourcePath) {
				return VolumeStatus{displaySource, v.Target, v.Mode, StatusOKLinked}
			}
			return VolumeStatus{displaySource, v.Target, v.Mode, StatusWrongLink}
		}
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusExistsNotLink}
	}

	// Copy Mode
	srcInfo, err := os.Stat(v.SourcePath)
	if err != nil {
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusError}
	}

	if srcInfo.IsDir() {
		if !info.IsDir() {
			return VolumeStatus{displaySource, v.Target, v.Mode, StatusExistsNotFile}
		}
		match, err := verifyDirHash(v.SourcePath, v.TargetPath)
		if err != nil {
			return VolumeStatus{displaySource, v.Target, v.Mode, StatusError}
		}
		if !match {
			return VolumeStatus{displaySource, v.Target, v.Mode, StatusModified}
		}
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusOKCopied}
	}

	if info.Mode().IsRegular() {
		match, err := verifyHash(v.SourcePath, v.TargetPath)
		if err != nil {
			return VolumeStatus{displaySource, v.Target, v.Mode, StatusError}
		}
		if !match {
			return VolumeStatus{displaySource, v.Target, v.Mode, StatusModified}
		}
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusOKCopied}
	}
	return VolumeStatus{displaySource, v.Target, v.Mode, StatusExistsNotFile}
}

// =============================================================================
// Context
// =============================================================================

// Context holds the execution state for git-volume operations
type Context struct {
	SourceDir string   // Base directory for resolving 'source' paths (where config lives)
	TargetDir string   // Base directory for resolving 'target' paths (current worktree root)
	GlobalDir string   // Resolved global directory absolute path for @global/ sources
	Volumes   []Volume // Parsed volume list
}

// HasGlobalVolumes returns true if any volume uses @global/ prefix
func (c *Context) HasGlobalVolumes() bool {
	for _, v := range c.Volumes {
		if v.IsGlobal {
			return true
		}
	}
	return false
}

// NewContext creates a new Context with only GlobalDir resolved.
// GlobalDir is always ~/.git-volume (GlobalDirectory).
// Config loading is deferred to the Load method.
func NewContext() (*Context, error) {
	globalDir, err := resolveGlobalDir()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve global directory: %w", err)
	}
	return &Context{
		GlobalDir: globalDir,
	}, nil
}

// Load loads configuration from config file and populates the Context.
// configPath: custom config file path (empty string for auto-detection)
func (c *Context) Load(configPath string, verbosity int) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// 1. Find the root of the current git worktree
	worktreeRoot, err := FindWorktreeRoot(cwd)
	if err != nil {
		return fmt.Errorf("failed to find git worktree root: %w", err)
	}

	// 2. Find config file path and source directory
	absConfigPath, sourceDir, err := c.findConfigPath(configPath, cwd, worktreeRoot)
	if err != nil {
		return err
	}

	// 3. Load and apply config
	cfg, err := loadConfig(absConfigPath, verbosity)
	if err != nil {
		return err
	}

	// 4. Set context fields
	c.SourceDir = sourceDir
	c.TargetDir = worktreeRoot
	c.Volumes = cfg.Volumes

	// Resolve absolute paths for each volume
	c.ResolveVolumePaths()

	return nil
}

// ResolveVolumePaths resolves the absolute SourcePath and TargetPath for all volumes.
// This should be called after SourceDir, TargetDir, GlobalDir, and Volumes are set.
func (c *Context) ResolveVolumePaths() {
	for i := range c.Volumes {
		v := &c.Volumes[i]
		srcBase := c.SourceDir
		if v.IsGlobal {
			srcBase = c.GlobalDir
		}
		v.SourcePath = filepath.Join(srcBase, v.Source)
		v.TargetPath = filepath.Join(c.TargetDir, v.Target)
	}
}

// findConfigPath finds the config file path and returns (absConfigPath, sourceDir, error)
// Search order: custom path > local worktree > main worktree
func (c *Context) findConfigPath(configPath, cwd, worktreeRoot string) (string, string, error) {
	// 1. Custom config path provided
	if configPath != "" {
		absConfigPath := configPath
		if !filepath.IsAbs(configPath) {
			absConfigPath = filepath.Join(cwd, configPath)
		}
		return absConfigPath, filepath.Dir(absConfigPath), nil
	}

	// 2. Local worktree config
	localConfigPath := filepath.Join(worktreeRoot, ConfigFileName)
	if _, err := os.Stat(localConfigPath); err == nil {
		return localConfigPath, worktreeRoot, nil
	}

	// 3. Main worktree config (Inheritance)
	mainWorktreeRoot, err := findCommonDir(worktreeRoot)
	if err != nil {
		return "", "", fmt.Errorf("config not found in current worktree, and failed to check main worktree: %w", err)
	}

	if mainWorktreeRoot != "" && mainWorktreeRoot != worktreeRoot {
		mainConfigPath := filepath.Join(mainWorktreeRoot, ConfigFileName)
		if _, err := os.Stat(mainConfigPath); err == nil {
			return mainConfigPath, mainWorktreeRoot, nil
		}
	}

	return "", "", fmt.Errorf("%s not found in current worktree or main worktree", ConfigFileName)
}

// =============================================================================
// Config Loading Helpers
// =============================================================================

// rawConfig represents the raw configuration file structure
type rawConfig struct {
	Volumes []Volume `yaml:"volumes"`
}

// loadConfig reads and parses the configuration file
// Returns rawConfig and error
func loadConfig(path string, verbosity int) (*rawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg rawConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Warn if no volumes defined
	if len(cfg.Volumes) == 0 && verbosity >= VerbosityNormal {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: no volumes defined in %s\n", path)
	}

	return &cfg, nil
}

// resolveGlobalDir resolves the global directory path (~/.git-volume).
// It handles ~ expansion and returns an absolute path.
func resolveGlobalDir() (string, error) {
	dir := GlobalDirectory

	// Expand ~ to home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	dir = filepath.Join(homeDir, dir[2:]) // Remove "~/"

	return dir, nil
}
