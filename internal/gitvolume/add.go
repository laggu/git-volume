package gitvolume

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AddOptions configures the Add operation
type AddOptions struct {
	Force bool   // Overwrite existing files in global directory
	As    string // Save as specific path/name (single file only)
	Path  string // Save to subdirectory within global directory
}

// Add copies files to the global git-volume directory
func (g *GitVolume) Add(files []string, opts AddOptions) error {
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

	globalDir := g.ctx.GlobalDir

	// Ensure global directory exists
	if err := os.MkdirAll(globalDir, DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create global directory %s: %w", globalDir, err)
	}

	var errs []error
	for _, file := range files {
		if err := g.addFile(file, globalDir, opts); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// addFile copies a single file or directory to the global directory
func (g *GitVolume) addFile(file, globalDir string, opts AddOptions) error {
	// Check if source exists (use Lstat to detect symlinks)
	srcInfo, err := os.Lstat(file)
	if os.IsNotExist(err) {
		return fmt.Errorf("source does not exist: %s", file)
	}
	if err != nil {
		return fmt.Errorf("failed to stat source %s: %w", file, err)
	}

	// Reject symlink sources for security
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("source is a symlink, which is not allowed for security reasons: %s", file)
	}

	// Only allow regular files and directories
	if !srcInfo.Mode().IsRegular() && !srcInfo.IsDir() {
		return fmt.Errorf("source must be a regular file or directory: %s", file)
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

	// Security: ensure the final destination path is within the global directory
	if err := verifyPathWithinBase(dstPath, globalDir); err != nil {
		return fmt.Errorf("security error for destination path %q: %w", targetSubPath, err)
	}

	// Check if destination exists and validate type compatibility
	if dstInfo, err := os.Lstat(dstPath); err == nil {
		// Check type compatibility: source and destination must be same type
		if srcInfo.IsDir() && !dstInfo.IsDir() {
			return fmt.Errorf("cannot overwrite file with directory: %s", displayDst)
		}
		if !srcInfo.IsDir() && dstInfo.IsDir() {
			return fmt.Errorf("cannot overwrite directory with file: %s", displayDst)
		}
		if !opts.Force {
			return fmt.Errorf("already exists: %s (use --force to overwrite)", displayDst)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check destination %s: %w", displayDst, err)
	}

	// Copy file or directory
	if srcInfo.IsDir() {
		if err := copyDirNoSymlink(srcAbs, dstPath); err != nil {
			return fmt.Errorf("failed to copy directory %s: %w", file, err)
		}
	} else {
		if err := copyFile(srcAbs, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", file, err)
		}
	}

	if !g.quiet {
		fmt.Printf("✓ Added %s -> %s\n", file, displayDst)
	}

	return nil
}
