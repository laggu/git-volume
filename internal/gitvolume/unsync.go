package gitvolume

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// UnsyncOptions configures the Unsync operation
type UnsyncOptions struct {
	DryRun bool // Show what would be done without making changes
}

// Unsync removes the volumes from the target workspace
func (g *GitVolume) Unsync(opts UnsyncOptions) error {
	var errs []error

	for _, vol := range g.ctx.Volumes {
		err := g.beforeUnsync(vol)
		if err == nil {
			err = g.unsync(vol, opts)
		}
		g.afterUnsync(vol, opts, err, &errs)
	}

	return g.afterAllUnsync(errs)
}

func (g *GitVolume) afterAllUnsync(errs []error) error {
	if len(errs) > 0 && !g.quiet {
		fmt.Printf("❌ Unsync completed with %d error(s)\n", len(errs))
	}
	return errors.Join(errs...)
}

func (g *GitVolume) beforeUnsync(vol Volume) error {
	if vol.IsGlobal && g.ctx.GlobalDir == "" {
		return fmt.Errorf("global source '@global/%s' used but global directory not configured", vol.Source)
	}

	// Security: verify target path doesn't escape base directory via symlinks
	if err := verifyPathWithinBase(vol.TargetPath, g.ctx.TargetDir); err != nil {
		return fmt.Errorf("security error for target %s: %w", vol.Target, err)
	}
	return nil
}

func (g *GitVolume) unsync(vol Volume, opts UnsyncOptions) error {
	removable, err := g.checkRemovable(vol)
	if err != nil {
		if !g.quiet {
			fmt.Printf("⚠️  Skipping %s: %v\n", vol.Target, err)
		}
		return nil
	}

	if !removable {
		if !g.quiet {
			fmt.Printf("⚠️  Skipping %s: modified or not managed by us\n", vol.Target)
		}
		return nil
	}

	if opts.DryRun {
		fmt.Printf("[dry-run] Would remove %s\n", vol.Target)
		return nil
	}

	return g.removeVolume(vol)
}

func (g *GitVolume) afterUnsync(vol Volume, opts UnsyncOptions, err error, errs *[]error) {
	if err != nil {
		if !g.quiet {
			fmt.Printf("❌ Failed to unsync %s: %v\n", vol.Target, err)
		}
		*errs = append(*errs, err)
	}
}

func (g *GitVolume) checkRemovable(vol Volume) (bool, error) {
	// Check if target exists
	info, err := os.Lstat(vol.TargetPath)
	if os.IsNotExist(err) {
		return false, nil // Already gone
	}
	if err != nil {
		return false, fmt.Errorf("failed to stat target: %w", err)
	}

	if vol.Mode == ModeCopy {
		// Copy Mode: Check Hash
		match, err := verifyHash(vol.SourcePath, vol.TargetPath)
		if err != nil {
			return false, fmt.Errorf("could not verify hash (source missing?)")
		}
		return match, nil
	}

	// Link Mode: Check Symlink Target
	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(vol.TargetPath)
		if err != nil {
			return false, nil
		}
		// Resolve relative symlink based on symlink's parent directory
		if !filepath.IsAbs(linkTarget) {
			linkTarget = filepath.Join(filepath.Dir(vol.TargetPath), linkTarget)
		}
		if pathsEqual(linkTarget, vol.SourcePath) {
			return true, nil
		}
	}
	return false, nil
}

func (g *GitVolume) removeVolume(vol Volume) error {
	if err := os.Remove(vol.TargetPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", vol.TargetPath, err)
	}
	if !g.quiet {
		fmt.Printf("✓ Removed %s\n", vol.Target)
	}

	// Clean up empty parent directories
	cleanEmptyParents(filepath.Dir(vol.TargetPath), g.ctx.TargetDir)
	return nil
}
