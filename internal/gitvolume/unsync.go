package gitvolume

import (
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
	for _, vol := range g.ctx.Volumes {
		// Check global directory
		if vol.IsGlobal && g.ctx.GlobalDir == "" {
			return fmt.Errorf("global source '@global/%s' used but global directory not configured", vol.Source)
		}

		// Security: verify target path doesn't escape base directory via symlinks
		if err := verifyPathWithinBase(vol.TargetPath, g.ctx.TargetDir); err != nil {
			return fmt.Errorf("security error for target %s: %w", vol.Target, err)
		}

		// Check if target exists
		info, err := os.Lstat(vol.TargetPath)
		if os.IsNotExist(err) {
			continue // Already gone
		}
		if err != nil {
			return fmt.Errorf("failed to stat target %s: %w", vol.TargetPath, err)
		}

		// Stateless Verification
		shouldRemove := false

		if vol.Mode == ModeCopy {
			// Copy Mode: Check Hash
			match, err := verifyHash(vol.SourcePath, vol.TargetPath)
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
				linkTarget, err := os.Readlink(vol.TargetPath)
				if err == nil {
					// Resolve relative symlink based on symlink's parent directory
					if !filepath.IsAbs(linkTarget) {
						linkTarget = filepath.Join(filepath.Dir(vol.TargetPath), linkTarget)
					}
					if pathsEqual(linkTarget, vol.SourcePath) {
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
			if err := os.Remove(vol.TargetPath); err != nil {
				return fmt.Errorf("failed to remove %s: %w", vol.TargetPath, err)
			}
			if !g.quiet {
				fmt.Printf("✓ Removed %s\n", vol.Target)
			}

			// Clean up empty parent directories
			cleanEmptyParents(filepath.Dir(vol.TargetPath), g.ctx.TargetDir)
		} else {
			if !g.quiet {
				fmt.Printf("⚠️  Skipping %s: modified or not managed by us\n", vol.Target)
			}
		}
	}
	return nil
}
