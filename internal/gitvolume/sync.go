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
	if err := g.beforeAllSync(opts); err != nil {
		return g.afterAllSync([]error{err}, opts)
	}

	var errs []error

	for _, vol := range g.ctx.Volumes {
		srcInfo, err := g.beforeSync(vol)
		if err == nil {
			err = g.sync(vol, srcInfo, opts)
		}
		g.afterSync(vol, opts, err, &errs)
	}

	return g.afterAllSync(errs, opts)
}

func (g *GitVolume) beforeAllSync(opts SyncOptions) error {
	if len(g.ctx.Volumes) == 0 {
		if err := g.Load(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if g.isNormalOrHigher() {
		fmt.Printf("📂 Using config from: %s\n", g.SourceDir())
		fmt.Printf("🎯 Target worktree: %s\n", g.TargetDir())
		if g.HasGlobalVolumes() {
			fmt.Printf("🌐 Global directory: %s\n", g.GlobalDir())
		}
	}

	return nil
}

func (g *GitVolume) afterAllSync(errs []error, opts SyncOptions) error {
	if len(errs) > 0 && g.isNormalOrHigher() {
		fmt.Fprintf(os.Stderr, "❌ Sync completed with %d error(s)\n", len(errs))
	}
	if len(errs) == 0 && g.isNormalOrHigher() && !opts.DryRun {
		fmt.Println("✓ Volumes successfully synced")
	}
	return errors.Join(errs...)
}

func (g *GitVolume) beforeSync(vol Volume) (os.FileInfo, error) {
	// Check global directory
	if vol.IsGlobal && g.ctx.GlobalDir == "" {
		return nil, fmt.Errorf("global source '@global/%s' used but global directory not configured", vol.Source)
	}

	displaySource := vol.DisplaySource()

	// Security: verify paths don't escape base directories via symlinks
	srcBase := g.ctx.SourceDir
	if vol.IsGlobal {
		srcBase = g.ctx.GlobalDir
	}
	if err := verifyPathWithinBase(vol.SourcePath, srcBase); err != nil {
		return nil, fmt.Errorf("security error for source %s: %w", displaySource, err)
	}
	if err := verifyPathWithinBase(vol.TargetPath, g.ctx.TargetDir); err != nil {
		return nil, fmt.Errorf("security error for target %s: %w", vol.Target, err)
	}

	// Check if source exists and is not a symlink
	srcInfo, err := os.Lstat(vol.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("source file not found: %s", vol.SourcePath)
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("source file is a symlink, which is not allowed for security reasons: %s", vol.SourcePath)
	}

	return srcInfo, nil
}

func (g *GitVolume) sync(vol Volume, srcInfo os.FileInfo, opts SyncOptions) error {
	if opts.DryRun {
		return nil
	}

	// Delete-and-Recreate strategy (Idempotency):
	// ensuring the target state exactly matches the source state.
	// We always remove the target path before syncing to guarantees a clean slate.
	// This approach handles several edge cases automatically:
	// 1. Switching between file and directory (e.g., symlink to directory)
	// 2. Switching modes (copy <-> link)
	// 3. Removing stale files in directories (mirroring) when switching from a directory that had extra files.
	if err := os.RemoveAll(vol.TargetPath); err != nil {
		return fmt.Errorf("failed to clean target %s: %w", vol.Target, err)
	}

	if vol.Mode == ModeCopy {
		if err := g.syncCopy(vol.SourcePath, vol.TargetPath, srcInfo); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", vol.SourcePath, vol.TargetPath, err)
		}
	} else {
		if err := g.syncLink(vol.SourcePath, vol.TargetPath, opts.RelativeLinks); err != nil {
			return fmt.Errorf("failed to link %s to %s: %w", vol.SourcePath, vol.TargetPath, err)
		}
	}
	return nil
}

func (g *GitVolume) afterSync(vol Volume, opts SyncOptions, err error, errs *[]error) {
	displaySource := vol.DisplaySource()

	if err != nil {
		if g.isNormalOrHigher() {
			fmt.Fprintf(os.Stderr, "❌ Failed to sync %s: %v\n", vol.Target, err)
		}
		*errs = append(*errs, err)
		return
	}

	if opts.DryRun {
		if g.isNormalOrHigher() {
			action := "link"
			if vol.Mode == ModeCopy {
				action = "copy"
			}
			fmt.Printf("[dry-run] Would %s %s -> %s\n", action, displaySource, vol.Target)
		}
		return
	}

	if g.isDetailed() {
		if vol.Mode == ModeCopy {
			fmt.Printf("✓ Copied %s -> %s\n", displaySource, vol.Target)
		} else {
			linkType := "absolute"
			if opts.RelativeLinks {
				linkType = "relative"
			}
			fmt.Printf("✓ Linked (%s) %s -> %s\n", linkType, displaySource, vol.Target)
		}
	}
}

// syncCopy handles copy mode synchronization
func (g *GitVolume) syncCopy(src, dst string, srcInfo os.FileInfo) error {
	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// syncLink handles link mode synchronization
func (g *GitVolume) syncLink(src, dst string, relativeLink bool) error {
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
