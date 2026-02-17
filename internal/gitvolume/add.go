package gitvolume

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AddOptions configures the GlobalAdd operation
type AddOptions struct {
	Force bool   // Overwrite existing files in global directory
	As    string // Save as specific path/name (single file only)
	Path  string // Save to subdirectory within global directory
}

// GlobalAdd copies files to the global git-volume directory
func (g *GitVolume) GlobalAdd(files []string, opts AddOptions) error {
	if err := g.beforeAllAdd(files, opts); err != nil {
		return err
	}

	globalDir := g.ctx.GlobalDir

	// Ensure global directory exists
	if err := os.MkdirAll(globalDir, DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create global directory %s: %w", globalDir, err)
	}

	var errs []error
	for _, file := range files {
		targetPath, err := g.beforeAdd(file, opts)
		if err != nil {
			g.afterAdd(file, "", opts, err)
			errs = append(errs, err)
			continue
		}

		err = g.add(file, targetPath, opts)
		if handledErr := g.afterAdd(file, targetPath, opts, err); handledErr != nil {
			errs = append(errs, handledErr)
		}
	}

	return g.afterAllAdd(errs)
}

func (g *GitVolume) beforeAllAdd(files []string, opts AddOptions) error {
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

	return nil
}

func (g *GitVolume) afterAllAdd(errs []error) error {
	return errors.Join(errs...)
}

func (g *GitVolume) beforeAdd(file string, opts AddOptions) (string, error) {
	globalDir := g.ctx.GlobalDir

	// Check if source exists (use Lstat to detect symlinks)
	srcInfo, err := os.Lstat(file)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("source does not exist: %s", file)
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat source %s: %w", file, err)
	}

	// Reject symlink sources for security
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("source is a symlink, which is not allowed for security reasons: %s", file)
	}

	// Only allow regular files and directories
	if !srcInfo.Mode().IsRegular() && !srcInfo.IsDir() {
		return "", fmt.Errorf("source must be a regular file or directory: %s", file)
	}

	// Get absolute path of source
	srcAbs, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
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
		return "", fmt.Errorf("security error for destination path %q: %w", targetSubPath, err)
	}

	// Check if destination exists and validate type compatibility
	if dstInfo, err := os.Lstat(dstPath); err == nil {
		// Check type compatibility: source and destination must be same type
		if srcInfo.IsDir() && !dstInfo.IsDir() {
			return "", fmt.Errorf("cannot overwrite file with directory: %s", displayDst)
		}
		if !srcInfo.IsDir() && dstInfo.IsDir() {
			return "", fmt.Errorf("cannot overwrite directory with file: %s", displayDst)
		}
		if !opts.Force {
			return "", fmt.Errorf("already exists: %s (use --force to overwrite)", displayDst)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check destination %s: %w", displayDst, err)
	}

	return dstPath, nil
}

func (g *GitVolume) add(file, dstPath string, opts AddOptions) error {
	srcInfo, err := os.Lstat(file)
	if err != nil {
		return err
	}

	srcAbs, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	// Copy file or directory
	if srcInfo.IsDir() {
		if err := copyDirNoSymlink(srcAbs, dstPath, true); err != nil {
			return fmt.Errorf("failed to copy directory %s: %w", file, err)
		}
	} else {
		if err := copyFile(srcAbs, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", file, err)
		}
	}

	return nil
}

func (g *GitVolume) afterAdd(file, dstPath string, opts AddOptions, err error) error {
	if err != nil {
		return err
	}

	if !g.quiet {
		displayDst := "@global/" + strings.TrimPrefix(dstPath, g.ctx.GlobalDir+"/")
		// Fix display path if separator is different
		if os.PathSeparator == '\\' {
			displayDst = "@global/" + strings.TrimPrefix(dstPath, g.ctx.GlobalDir+"\\")
		}
		fmt.Printf("✓ Added %s -> %s\n", file, displayDst)
	}
	return nil
}
