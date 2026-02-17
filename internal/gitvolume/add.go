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

type addPrepared struct {
	dstPath string
	srcInfo os.FileInfo
	srcAbs  string
}

// GlobalAdd copies files to the global git-volume directory
func (g *GitVolume) GlobalAdd(files []string, opts AddOptions) error {
	if err := g.beforeAllAdd(files, opts); err != nil {
		return err
	}

	var errs []error
	for _, file := range files {
		prepared, err := g.beforeAdd(file, opts)
		if err == nil {
			err = g.add(file, prepared, opts)
		}
		g.afterAdd(file, prepared.dstPath, opts, err, &errs)
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

	globalDir := g.ctx.GlobalDir
	if err := os.MkdirAll(globalDir, DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create global directory %s: %w", globalDir, err)
	}

	return nil
}

func (g *GitVolume) afterAllAdd(errs []error) error {
	if len(errs) > 0 && !g.quiet {
		fmt.Printf("❌ Global add completed with %d error(s)\n", len(errs))
	}
	return errors.Join(errs...)
}

func (g *GitVolume) beforeAdd(file string, opts AddOptions) (addPrepared, error) {
	globalDir := g.ctx.GlobalDir

	// Check if source exists (use Lstat to detect symlinks)
	srcInfo, err := os.Lstat(file)
	if os.IsNotExist(err) {
		return addPrepared{}, fmt.Errorf("source does not exist: %s", file)
	}
	if err != nil {
		return addPrepared{}, fmt.Errorf("failed to stat source %s: %w", file, err)
	}

	// Reject symlink sources for security
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return addPrepared{}, fmt.Errorf("source is a symlink, which is not allowed for security reasons: %s", file)
	}

	// Only allow regular files and directories
	if !srcInfo.Mode().IsRegular() && !srcInfo.IsDir() {
		return addPrepared{}, fmt.Errorf("source must be a regular file or directory: %s", file)
	}

	// Get absolute path of source
	srcAbs, err := filepath.Abs(file)
	if err != nil {
		return addPrepared{}, fmt.Errorf("failed to get absolute path: %w", err)
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
		return addPrepared{}, fmt.Errorf("security error for destination path %q: %w", targetSubPath, err)
	}

	// Check if destination exists and validate type compatibility
	if dstInfo, err := os.Lstat(dstPath); err == nil {
		// Check type compatibility: source and destination must be same type
		if srcInfo.IsDir() && !dstInfo.IsDir() {
			return addPrepared{}, fmt.Errorf("cannot overwrite file with directory: %s", displayDst)
		}
		if !srcInfo.IsDir() && dstInfo.IsDir() {
			return addPrepared{}, fmt.Errorf("cannot overwrite directory with file: %s", displayDst)
		}
		if !opts.Force {
			return addPrepared{}, fmt.Errorf("already exists: %s (use --force to overwrite)", displayDst)
		}
	} else if !os.IsNotExist(err) {
		return addPrepared{}, fmt.Errorf("failed to check destination %s: %w", displayDst, err)
	}

	return addPrepared{dstPath: dstPath, srcInfo: srcInfo, srcAbs: srcAbs}, nil
}

func (g *GitVolume) add(file string, prepared addPrepared, opts AddOptions) error {
	// Copy file or directory
	if prepared.srcInfo.IsDir() {
		if err := copyDirNoSymlink(prepared.srcAbs, prepared.dstPath, true); err != nil {
			return fmt.Errorf("failed to copy directory %s: %w", file, err)
		}
	} else {
		if err := copyFile(prepared.srcAbs, prepared.dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", file, err)
		}
	}

	return nil
}

func (g *GitVolume) afterAdd(file, dstPath string, opts AddOptions, err error, errs *[]error) {
	if err != nil {
		if !g.quiet {
			fmt.Printf("❌ Failed to add %s: %v\n", file, err)
		}
		*errs = append(*errs, err)
		return
	}

	if !g.quiet {
		relPath, err := filepath.Rel(g.ctx.GlobalDir, dstPath)
		if err != nil {
			// This should not happen due to prior validation, but as a fallback:
			relPath = filepath.Base(dstPath)
		}
		displayDst := "@global/" + filepath.ToSlash(relPath)
		fmt.Printf("✓ Added %s -> %s\n", file, displayDst)
	}
}
