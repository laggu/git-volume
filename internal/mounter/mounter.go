package mounter

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/laggu/git-volume/internal/config"
)

type Mounter struct {
	SourceBase string
	TargetBase string
	GlobalBase string // Base directory for @global/ sources
}

// SyncOptions configures the Sync operation
type SyncOptions struct {
	DryRun        bool // Show what would be done without making changes
	RelativeLinks bool // Create relative symlinks instead of absolute
	Verbose       bool // Verbose output
	Quiet         bool // Suppress non-error output
}

// UnsyncOptions configures the Unsync operation
type UnsyncOptions struct {
	DryRun  bool // Show what would be done without making changes
	Verbose bool // Verbose output
	Quiet   bool // Suppress non-error output
}

func New(sourceBase, targetBase, globalBase string) *Mounter {
	return &Mounter{
		SourceBase: sourceBase,
		TargetBase: targetBase,
		GlobalBase: globalBase,
	}
}

// verifyPathWithinBase checks that a path's parent directory resolves to a location within the base directory,
// even when intermediate path components are symlinks. This prevents symlink-based path traversal attacks.
// Note: This checks the parent directory, not the file itself, because the file might be a symlink we manage.
func verifyPathWithinBase(path, base string) error {
	// Resolve the base directory
	resolvedBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}
	resolvedBase = filepath.Clean(resolvedBase)

	// Check the parent directory of the path (not the file itself, which might be our symlink)
	parentDir := filepath.Dir(path)

	// Resolve the parent directory
	resolvedParent, err := filepath.EvalSymlinks(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Parent doesn't exist yet, check its parent recursively
			// For simplicity, we allow this case (directory will be created)
			return nil
		}
		return fmt.Errorf("failed to resolve parent path: %w", err)
	}
	resolvedParent = filepath.Clean(resolvedParent)

	// Check that resolved parent is within resolved base
	if !strings.HasPrefix(resolvedParent+string(filepath.Separator), resolvedBase+string(filepath.Separator)) &&
		resolvedParent != resolvedBase {
		return fmt.Errorf("path escapes base directory via symlink: parent %s resolves to %s", parentDir, resolvedParent)
	}

	return nil
}

// Sync applies the volumes to the target workspace
func (m *Mounter) Sync(volumes []config.Volume, opts SyncOptions) error {
	for _, vol := range volumes {
		// Determine source base directory based on IsGlobal flag
		var srcBase string
		var displaySource string
		if vol.IsGlobal {
			if m.GlobalBase == "" {
				return fmt.Errorf("global source '@global/%s' used but global directory not configured", vol.Source)
			}
			srcBase = m.GlobalBase
			displaySource = "@global/" + vol.Source
		} else {
			srcBase = m.SourceBase
			displaySource = vol.Source
		}

		srcPath := filepath.Join(srcBase, vol.Source)
		dstPath := filepath.Join(m.TargetBase, vol.Target)

		// Security: verify paths don't escape base directories via symlinks
		if err := verifyPathWithinBase(srcPath, srcBase); err != nil {
			return fmt.Errorf("security error for source %s: %w", displaySource, err)
		}
		if err := verifyPathWithinBase(dstPath, m.TargetBase); err != nil {
			return fmt.Errorf("security error for target %s: %w", vol.Target, err)
		}

		// Check if source exists and is not a symlink (security: prevent reading sensitive files outside repo)
		srcInfo, err := os.Lstat(srcPath)
		if err != nil {
			return fmt.Errorf("source file not found: %s", srcPath)
		}
		if srcInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("source file is a symlink, which is not allowed for security reasons: %s", srcPath)
		}

		if opts.DryRun {
			action := "link"
			if vol.Mode == config.ModeCopy {
				action = "copy"
			}
			fmt.Printf("[dry-run] Would %s %s -> %s\n", action, displaySource, vol.Target)
			continue
		}

		if vol.Mode == config.ModeCopy {
			if err := m.syncCopy(srcPath, dstPath, vol.Force); err != nil {
				return fmt.Errorf("failed to copy %s to %s: %w", srcPath, dstPath, err)
			}
			if opts.Verbose && !opts.Quiet {
				fmt.Printf("✓ Copied %s -> %s\n", displaySource, vol.Target)
			}
		} else {
			if err := m.syncLink(srcPath, dstPath, vol.Force, opts.RelativeLinks); err != nil {
				return fmt.Errorf("failed to link %s to %s: %w", srcPath, dstPath, err)
			}
			if opts.Verbose && !opts.Quiet {
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
func (m *Mounter) Unsync(volumes []config.Volume, opts UnsyncOptions) error {
	for _, vol := range volumes {
		// Determine source base directory based on IsGlobal flag
		var srcBase string
		if vol.IsGlobal {
			if m.GlobalBase == "" {
				return fmt.Errorf("global source '@global/%s' used but global directory not configured", vol.Source)
			}
			srcBase = m.GlobalBase
		} else {
			srcBase = m.SourceBase
		}

		srcPath := filepath.Join(srcBase, vol.Source)
		dstPath := filepath.Join(m.TargetBase, vol.Target)

		// Security: verify target path doesn't escape base directory via symlinks
		if err := verifyPathWithinBase(dstPath, m.TargetBase); err != nil {
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

		if vol.Mode == config.ModeCopy {
			// Copy Mode: Check Hash
			match, err := m.verifyHash(srcPath, dstPath)
			if err != nil {
				// If source is gone, we can't verify.
				// Decision: if source is gone, we technically can't be sure if target is ours.
				// But user might want to cleanup.
				// For safety, let's Skip if verification fails.
				if !opts.Quiet {
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
					if PathsEqual(linkTarget, srcPath) {
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
			if !opts.Quiet {
				fmt.Printf("✓ Removed %s\n", vol.Target)
			}

			// Clean up empty parent directories
			cleanEmptyParents(filepath.Dir(dstPath), m.TargetBase)
		} else {
			if !opts.Quiet {
				fmt.Printf("⚠️  Skipping %s: modified or not managed by us\n", vol.Target)
			}
		}
	}
	return nil
}

func (m *Mounter) syncCopy(src, dst string, force bool) error {
	// Check exist
	if info, err := os.Stat(dst); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("target exists and is not a regular file")
		}
		// Calculate hash to see if identical
		match, err := m.verifyHash(src, dst)
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

func (m *Mounter) syncLink(src, dst string, force bool, relativeLink bool) error {
	// Check file existence
	if info, err := os.Lstat(dst); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			currentTarget, err := os.Readlink(dst)
			if err == nil {
				// Resolve relative symlink based on symlink's parent directory
				if !filepath.IsAbs(currentTarget) {
					currentTarget = filepath.Join(filepath.Dir(dst), currentTarget)
				}
				if PathsEqual(currentTarget, src) {
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
	if err := os.MkdirAll(filepath.Dir(dst), config.DefaultDirPerm); err != nil {
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

func (m *Mounter) verifyHash(file1, file2 string) (bool, error) {
	h1, err := hashFile(file1)
	if err != nil {
		return false, err
	}
	h2, err := hashFile(file2)
	if err != nil {
		return false, err
	}
	return h1 == h2, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// PathsEqual compares two paths after normalizing them
// This handles differences in path separators and relative vs absolute paths
func PathsEqual(path1, path2 string) bool {
	// Clean and normalize both paths
	clean1 := filepath.Clean(path1)
	clean2 := filepath.Clean(path2)

	// Try to get absolute paths for comparison
	abs1, err1 := filepath.Abs(clean1)
	abs2, err2 := filepath.Abs(clean2)

	if err1 == nil && err2 == nil {
		return abs1 == abs2
	}

	// Fallback to cleaned path comparison
	return clean1 == clean2
}

// cleanEmptyParents removes empty parent directories up to (but not including) stopAt
func cleanEmptyParents(dir, stopAt string) {
	stopAt = filepath.Clean(stopAt)

	for {
		dir = filepath.Clean(dir)

		// Check if dir is within stopAt using filepath.Rel
		rel, err := filepath.Rel(stopAt, dir)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return
		}

		// Check if directory is empty
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return // Not empty or error reading
		}

		// Remove empty directory
		if err := os.Remove(dir); err != nil {
			return // Can't remove, stop
		}

		// Move to parent
		dir = filepath.Dir(dir)
	}
}

func copyFile(src, dst string) error {
	// Ensure directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, config.DefaultDirPerm); err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create temporary file in the same directory for atomic rename
	tmpFile, err := os.CreateTemp(dstDir, ".git-volume-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	// Cleanup on failure
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Copy content
	if _, err := io.Copy(tmpFile, s); err != nil {
		tmpFile.Close()
		return err
	}

	// Sync to ensure data is written to disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	// Copy permissions
	if err := os.Chmod(tmpPath, srcInfo.Mode()); err != nil {
		return err
	}

	// Atomic rename (may fail on cross-filesystem)
	if err := os.Rename(tmpPath, dst); err != nil {
		// Fallback: copy and delete for cross-filesystem
		if err := copyFileContent(tmpPath, dst); err != nil {
			return fmt.Errorf("rename failed and fallback copy also failed: %w", err)
		}
		os.Remove(tmpPath)
	}

	success = true
	return nil
}

// copyFileContent copies file content (used as fallback for cross-filesystem rename)
func copyFileContent(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	d, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}

	return d.Sync()
}
