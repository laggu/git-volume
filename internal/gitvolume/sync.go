package gitvolume

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SyncOptions configures the Sync operation
type SyncOptions struct {
	DryRun        bool // Show what would be done without making changes
	RelativeLinks bool // Create relative symlinks instead of absolute
}

// Sync applies the volumes to the target workspace
func (g *GitVolume) Sync(opts SyncOptions) error {
	var errs []error

	for _, vol := range g.ctx.Volumes {
		// Check global directory
		if vol.IsGlobal && g.ctx.GlobalDir == "" {
			errs = append(errs, fmt.Errorf("global source '@global/%s' used but global directory not configured", vol.Source))
			continue
		}

		displaySource := vol.DisplaySource()

		// Security: verify paths don't escape base directories via symlinks
		srcBase := g.ctx.SourceDir
		if vol.IsGlobal {
			srcBase = g.ctx.GlobalDir
		}
		if err := verifyPathWithinBase(vol.SourcePath, srcBase); err != nil {
			errs = append(errs, fmt.Errorf("security error for source %s: %w", displaySource, err))
			continue
		}
		if err := verifyPathWithinBase(vol.TargetPath, g.ctx.TargetDir); err != nil {
			errs = append(errs, fmt.Errorf("security error for target %s: %w", vol.Target, err))
			continue
		}

		// Check if source exists and is not a symlink
		srcInfo, err := os.Lstat(vol.SourcePath)
		if err != nil {
			errs = append(errs, fmt.Errorf("source file not found: %s", vol.SourcePath))
			continue
		}
		if srcInfo.Mode()&os.ModeSymlink != 0 {
			errs = append(errs, fmt.Errorf("source file is a symlink, which is not allowed for security reasons: %s", vol.SourcePath))
			continue
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
			if err := g.syncCopy(vol.SourcePath, vol.TargetPath, vol.Force); err != nil {
				errs = append(errs, fmt.Errorf("failed to copy %s to %s: %w", vol.SourcePath, vol.TargetPath, err))
				continue
			}
			if g.verbose && !g.quiet {
				fmt.Printf("✓ Copied %s -> %s\n", displaySource, vol.Target)
			}
		} else {
			if err := g.syncLink(vol.SourcePath, vol.TargetPath, vol.Force, opts.RelativeLinks); err != nil {
				errs = append(errs, fmt.Errorf("failed to link %s to %s: %w", vol.SourcePath, vol.TargetPath, err))
				continue
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

	return errors.Join(errs...)
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
