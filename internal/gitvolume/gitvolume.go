package gitvolume

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitVolume is the main entry point for git-volume operations
type GitVolume struct {
	ws      *Workspace
	verbose bool
	quiet   bool
}

// Options configures GitVolume creation
type Options struct {
	ConfigPath        string // Custom config file path (optional)
	GlobalDirOverride string // Override globalDir from config (optional)
	Verbose           bool   // Verbose output
	Quiet             bool   // Suppress non-error output
}

// SyncOptions configures the Sync operation
type SyncOptions struct {
	DryRun        bool // Show what would be done without making changes
	RelativeLinks bool // Create relative symlinks instead of absolute
}

// UnsyncOptions configures the Unsync operation
type UnsyncOptions struct {
	DryRun bool // Show what would be done without making changes
}

// AddOptions configures the Add operation
type AddOptions struct {
	Force bool   // Overwrite existing files in global directory
	As    string // Save as specific path/name (single file only)
	Path  string // Save to subdirectory within global directory
	Quiet bool   // Suppress non-error output
}

// InitOptions configures the Init operation
type InitOptions struct {
	Quiet bool // Suppress non-error output
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
	StatusNotMounted    = "NOT MOUNTED"
	StatusMissingSource = "MISSING (Source)"
	StatusWrongLink     = "WRONG LINK"
	StatusExistsNotLink = "EXISTS (Not Link)"
	StatusExistsNotFile = "EXISTS (Not File)"
	StatusError         = "ERROR"
)

// New creates a new GitVolume instance
func New(opts Options) (*GitVolume, error) {
	ws, err := newWorkspace(opts.ConfigPath, opts.GlobalDirOverride, opts.Quiet)
	if err != nil {
		return nil, err
	}
	return &GitVolume{
		ws:      ws,
		verbose: opts.Verbose,
		quiet:   opts.Quiet,
	}, nil
}

// SourceDir returns the source directory (where config lives)
func (g *GitVolume) SourceDir() string { return g.ws.sourceDir }

// TargetDir returns the target directory (current worktree root)
func (g *GitVolume) TargetDir() string { return g.ws.targetDir }

// GlobalDir returns the global directory for @global/ sources
func (g *GitVolume) GlobalDir() string { return g.ws.globalDir }

// HasGlobalVolumes returns true if any volume uses @global/ prefix
func (g *GitVolume) HasGlobalVolumes() bool { return hasGlobalVolumes(g.ws.volumes) }

// Init initializes git-volume (creates global directory and sample config)
// This is a standalone function that doesn't require a GitVolume instance
func Init(opts InitOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 1. Create Global Directory
	globalDir := filepath.Join(home, ".git-volume")
	if err := os.MkdirAll(globalDir, DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create global directory %s: %w", globalDir, err)
	}
	if !opts.Quiet {
		fmt.Printf("✓ Global directory initialized: %s\n", globalDir)
	}

	// 2. Create Sample Config if not exists (at git root)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	gitRoot, err := FindWorktreeRoot(cwd)
	if err != nil {
		return fmt.Errorf("failed to find git repository root: %w", err)
	}
	configPath := filepath.Join(gitRoot, ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(SampleConfig), DefaultFilePerm); err != nil {
			return fmt.Errorf("failed to create sample config: %w", err)
		}
		if !opts.Quiet {
			fmt.Printf("✓ Created sample configuration: %s\n", configPath)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check config file: %w", err)
	} else {
		if !opts.Quiet {
			fmt.Printf("ℹ️  Configuration file already exists: %s\n", configPath)
		}
	}

	// 3. Guidance
	if !opts.Quiet {
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Add 'git-volume.yaml' to your .gitignore (optional but recommended)")
		fmt.Println("  2. Run 'git volume sync' to apply volumes")
	}
	return nil
}

// Sync applies the volumes to the target workspace
func (g *GitVolume) Sync(opts SyncOptions) error {
	for _, vol := range g.ws.volumes {
		// Determine source base directory based on IsGlobal flag
		var srcBase string
		var displaySource string
		if vol.IsGlobal {
			if g.ws.globalDir == "" {
				return fmt.Errorf("global source '@global/%s' used but global directory not configured", vol.Source)
			}
			srcBase = g.ws.globalDir
			displaySource = "@global/" + vol.Source
		} else {
			srcBase = g.ws.sourceDir
			displaySource = vol.Source
		}

		srcPath := filepath.Join(srcBase, vol.Source)
		dstPath := filepath.Join(g.ws.targetDir, vol.Target)

		// Security: verify paths don't escape base directories via symlinks
		if err := verifyPathWithinBase(srcPath, srcBase); err != nil {
			return fmt.Errorf("security error for source %s: %w", displaySource, err)
		}
		if err := verifyPathWithinBase(dstPath, g.ws.targetDir); err != nil {
			return fmt.Errorf("security error for target %s: %w", vol.Target, err)
		}

		// Check if source exists and is not a symlink
		srcInfo, err := os.Lstat(srcPath)
		if err != nil {
			return fmt.Errorf("source file not found: %s", srcPath)
		}
		if srcInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("source file is a symlink, which is not allowed for security reasons: %s", srcPath)
		}

		if opts.DryRun {
			action := "link"
			if vol.Mode == ModeCopy {
				action = "copy"
			}
			fmt.Printf("[dry-run] Would %s %s -> %s\n", action, displaySource, vol.Target)
			continue
		}

		if vol.Mode == ModeCopy {
			if err := g.syncCopy(srcPath, dstPath, vol.Force); err != nil {
				return fmt.Errorf("failed to copy %s to %s: %w", srcPath, dstPath, err)
			}
			if g.verbose && !g.quiet {
				fmt.Printf("✓ Copied %s -> %s\n", displaySource, vol.Target)
			}
		} else {
			if err := g.syncLink(srcPath, dstPath, vol.Force, opts.RelativeLinks); err != nil {
				return fmt.Errorf("failed to link %s to %s: %w", srcPath, dstPath, err)
			}
			if g.verbose && !g.quiet {
				linkType := "absolute"
				if opts.RelativeLinks {
					linkType = "relative"
				}
				fmt.Printf("✓ Linked (%s) %s -> %s\n", linkType, displaySource, vol.Target)
			}
		}
	}
	return nil
}

// Unsync removes the volumes from the target workspace
func (g *GitVolume) Unsync(opts UnsyncOptions) error {
	for _, vol := range g.ws.volumes {
		// Determine source base directory based on IsGlobal flag
		var srcBase string
		if vol.IsGlobal {
			if g.ws.globalDir == "" {
				return fmt.Errorf("global source '@global/%s' used but global directory not configured", vol.Source)
			}
			srcBase = g.ws.globalDir
		} else {
			srcBase = g.ws.sourceDir
		}

		srcPath := filepath.Join(srcBase, vol.Source)
		dstPath := filepath.Join(g.ws.targetDir, vol.Target)

		// Security: verify target path doesn't escape base directory via symlinks
		if err := verifyPathWithinBase(dstPath, g.ws.targetDir); err != nil {
			return fmt.Errorf("security error for target %s: %w", vol.Target, err)
		}

		// Check if target exists
		info, err := os.Lstat(dstPath)
		if os.IsNotExist(err) {
			continue // Already gone
		}
		if err != nil {
			return fmt.Errorf("failed to stat target %s: %w", dstPath, err)
		}

		// Stateless Verification
		shouldRemove := false

		if vol.Mode == ModeCopy {
			// Copy Mode: Check Hash
			match, err := verifyHash(srcPath, dstPath)
			if err != nil {
				if !g.quiet {
					fmt.Printf("⚠️  Skipping %s: could not verify hash (source missing?)\n", vol.Target)
				}
				continue
			}
			shouldRemove = match
		} else {
			// Link Mode: Check Symlink Target
			if info.Mode()&os.ModeSymlink != 0 {
				linkTarget, err := os.Readlink(dstPath)
				if err == nil {
					// Resolve relative symlink based on symlink's parent directory
					if !filepath.IsAbs(linkTarget) {
						linkTarget = filepath.Join(filepath.Dir(dstPath), linkTarget)
					}
					if pathsEqual(linkTarget, srcPath) {
						shouldRemove = true
					}
				}
			}
		}

		if shouldRemove {
			if opts.DryRun {
				fmt.Printf("[dry-run] Would remove %s\n", vol.Target)
				continue
			}
			if err := os.Remove(dstPath); err != nil {
				return fmt.Errorf("failed to remove %s: %w", dstPath, err)
			}
			if !g.quiet {
				fmt.Printf("✓ Removed %s\n", vol.Target)
			}

			// Clean up empty parent directories
			cleanEmptyParents(filepath.Dir(dstPath), g.ws.targetDir)
		} else {
			if !g.quiet {
				fmt.Printf("⚠️  Skipping %s: modified or not managed by us\n", vol.Target)
			}
		}
	}
	return nil
}

// List returns the status of all volumes
func (g *GitVolume) List() ([]VolumeStatus, error) {
	statuses := make([]VolumeStatus, 0, len(g.ws.volumes))
	for _, v := range g.ws.volumes {
		statuses = append(statuses, g.checkStatus(v))
	}
	return statuses, nil
}

// checkStatus checks the mount status of a single volume
func (g *GitVolume) checkStatus(v Volume) VolumeStatus {
	// 1. Determine source base directory
	srcBase := g.ws.sourceDir
	displaySource := v.Source
	if v.IsGlobal {
		srcBase = g.ws.globalDir
		displaySource = "@global/" + v.Source
	}

	srcPath := filepath.Join(srcBase, v.Source)
	targetPath := filepath.Join(g.ws.targetDir, v.Target)

	// 2. Check if source exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusMissingSource}
	}

	// 3. Check if target exists
	info, err := os.Lstat(targetPath)
	if os.IsNotExist(err) {
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusNotMounted}
	}
	if err != nil {
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusError}
	}

	// 4. Check status by mode
	if v.Mode == ModeLink {
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(targetPath)
			if err != nil {
				return VolumeStatus{displaySource, v.Target, v.Mode, StatusError}
			}
			// Resolve relative symlink based on symlink's parent directory
			if !filepath.IsAbs(link) {
				link = filepath.Join(filepath.Dir(targetPath), link)
			}
			if pathsEqual(link, srcPath) {
				return VolumeStatus{displaySource, v.Target, v.Mode, StatusOKLinked}
			}
			return VolumeStatus{displaySource, v.Target, v.Mode, StatusWrongLink}
		}
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusExistsNotLink}
	}

	// Copy Mode
	if info.Mode().IsRegular() {
		return VolumeStatus{displaySource, v.Target, v.Mode, StatusOKCopied}
	}
	return VolumeStatus{displaySource, v.Target, v.Mode, StatusExistsNotFile}
}

// syncCopy handles copy mode synchronization
func (g *GitVolume) syncCopy(src, dst string, force bool) error {
	// Check exist
	if info, err := os.Stat(dst); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("target exists and is not a regular file")
		}
		// Calculate hash to see if identical
		match, err := verifyHash(src, dst)
		if err != nil {
			return err
		}
		if match {
			return nil // Already synced
		}
		if !force {
			return fmt.Errorf("target exists and differs from source (use force: true to overwrite)")
		}
	}

	return copyFile(src, dst)
}

// Add copies files to the global git-volume directory
// This is a standalone function that doesn't require a GitVolume instance
func Add(files []string, opts AddOptions) error {
	// Validate: --as can only be used with single file
	if opts.As != "" && len(files) > 1 {
		return fmt.Errorf("--as can only be used with a single file")
	}

	// Validate: paths must not contain .. or be absolute
	if strings.Contains(opts.As, "..") {
		return fmt.Errorf("--as path cannot contain '..'")
	}
	if strings.Contains(opts.Path, "..") {
		return fmt.Errorf("--path cannot contain '..'")
	}
	if filepath.IsAbs(opts.As) {
		return fmt.Errorf("--as must be a relative path")
	}
	if filepath.IsAbs(opts.Path) {
		return fmt.Errorf("--path must be a relative path")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	globalDir := filepath.Join(home, ".git-volume")

	// Ensure global directory exists
	if err := os.MkdirAll(globalDir, DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create global directory %s: %w", globalDir, err)
	}

	var errs []error
	for _, file := range files {
		if err := addFile(file, globalDir, opts); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// addFile copies a single file to the global directory
func addFile(file, globalDir string, opts AddOptions) error {
	// Check if source file exists
	srcInfo, err := os.Stat(file)
	if os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", file)
	}
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", file, err)
	}

	// Only allow regular files
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file: %s", file)
	}

	// Get absolute path of source
	srcAbs, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Determine destination path
	var targetSubPath string
	if opts.As != "" {
		// --as: use specified path/name
		targetSubPath = opts.As
	} else {
		basename := filepath.Base(srcAbs)
		if opts.Path != "" {
			// --path: use subdirectory + original basename
			targetSubPath = filepath.Join(opts.Path, basename)
		} else {
			// Default: basename only
			targetSubPath = basename
		}
	}
	dstPath := filepath.Join(globalDir, targetSubPath)
	displayDst := "@global/" + targetSubPath

	// Check if destination exists
	if _, err := os.Stat(dstPath); err == nil {
		if !opts.Force {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", displayDst)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check destination %s: %w", displayDst, err)
	}

	// Copy file
	if err := copyFile(srcAbs, dstPath); err != nil {
		return fmt.Errorf("failed to copy %s: %w", file, err)
	}

	if !opts.Quiet {
		fmt.Printf("✓ Added %s -> %s\n", file, displayDst)
	}

	return nil
}

// syncLink handles link mode synchronization
func (g *GitVolume) syncLink(src, dst string, force bool, relativeLink bool) error {
	// Check file existence
	if info, err := os.Lstat(dst); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			currentTarget, err := os.Readlink(dst)
			if err == nil {
				// Resolve relative symlink based on symlink's parent directory
				if !filepath.IsAbs(currentTarget) {
					currentTarget = filepath.Join(filepath.Dir(dst), currentTarget)
				}
				if pathsEqual(currentTarget, src) {
					return nil // Already linked correctly
				}
			}
		}
		if !force {
			return fmt.Errorf("target exists (use force: true to overwrite)")
		}
		// Remove existing to create link
		if info.IsDir() {
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("failed to remove existing directory %s: %w", dst, err)
			}
		} else {
			if err := os.Remove(dst); err != nil {
				return fmt.Errorf("failed to remove existing target %s: %w", dst, err)
			}
		}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	linkTarget := src
	if relativeLink {
		// Calculate relative path from destination to source
		rel, err := filepath.Rel(filepath.Dir(dst), src)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}
		linkTarget = rel
	}

	return os.Symlink(linkTarget, dst)
}
