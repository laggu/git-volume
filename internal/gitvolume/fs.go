package gitvolume

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// copyFile copies a regular file directly to the destination.
// If copy fails, the source file remains intact and sync can be re-run.
func copyFile(src, dst string) error {
	// Ensure directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, DefaultDirPerm); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		_ = dstFile.Close()
		return err
	}

	if err := dstFile.Sync(); err != nil {
		_ = dstFile.Close()
		return err
	}

	return dstFile.Close()
}

// copyDir recursively copies a directory tree.
// It explicitly prohibits symlinks in the source directory for security reasons,
// to prevent potential path traversal or circular reference attacks.
func copyDir(src, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory %s: %w", src, err)
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("security: source directory is a symlink, which is not allowed: %s", src)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		entryInfo, err := os.Lstat(srcPath)
		if err != nil {
			return fmt.Errorf("failed to stat source entry %s: %w", srcPath, err)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("security: source directory contains a symlink, which is not allowed: %s", srcPath)
		}

		if entryInfo.IsDir() {
			if dstInfo, err := os.Lstat(dstPath); err == nil {
				if !dstInfo.IsDir() {
					if dstInfo.Mode()&os.ModeSymlink != 0 || dstInfo.Mode().IsRegular() {
						if err := os.Remove(dstPath); err != nil {
							return fmt.Errorf("failed to remove conflicting target path: %w", err)
						}
					} else {
						return fmt.Errorf("target exists and is not a directory: %s", dstPath)
					}
				}
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("failed to stat target path %s: %w", dstPath, err)
			}

			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy directory %s to %s: %w", srcPath, dstPath, err)
			}
			continue
		}

		// If target exists, verify it's a regular file or symlink
		if info, err := os.Lstat(dstPath); err == nil {
			if info.IsDir() {
				return fmt.Errorf("target exists and is a directory: %s", dstPath)
			}
			if info.Mode()&os.ModeSymlink != 0 || info.Mode().IsRegular() {
				if err := os.Remove(dstPath); err != nil {
					return fmt.Errorf("failed to remove existing target path: %w", err)
				}
			} else {
				return fmt.Errorf("target exists and is not replaceable: %s", dstPath)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat target path %s: %w", dstPath, err)
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy file %s to %s: %w", srcPath, dstPath, err)
		}
	}

	return nil
}

// hashFile calculates SHA256 hash of a file
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// verifyHash compares hashes of two files
func verifyHash(file1, file2 string) (bool, error) {
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

var errDirSubsetMismatch = errors.New("directory subset mismatch")

func verifyDirSubset(src, dst string) (bool, error) {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return false, fmt.Errorf("failed to stat source directory %s: %w", src, err)
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("security: source directory is a symlink, which is not allowed: %s", src)
	}

	dstInfo, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to stat target directory %s: %w", dst, err)
	}
	if !dstInfo.IsDir() {
		return false, nil
	}

	err = filepath.WalkDir(src, func(current string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("failed to walk source directory %s: %w", current, walkErr)
		}

		rel, err := filepath.Rel(src, current)
		if err != nil {
			return fmt.Errorf("failed to resolve relative path for %s from %s: %w", current, src, err)
		}
		if rel == "." {
			return nil
		}

		srcEntryInfo, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("failed to stat source entry %s: %w", current, err)
		}
		if srcEntryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("security: source directory contains a symlink, which is not allowed: %s", current)
		}

		targetPath := filepath.Join(dst, rel)
		targetInfo, err := os.Lstat(targetPath)
		if os.IsNotExist(err) {
			return errDirSubsetMismatch
		}
		if err != nil {
			return fmt.Errorf("failed to stat target path %s: %w", targetPath, err)
		}

		if srcEntryInfo.IsDir() {
			if !targetInfo.IsDir() {
				return errDirSubsetMismatch
			}
			return nil
		}

		if !targetInfo.Mode().IsRegular() {
			return errDirSubsetMismatch
		}

		match, err := verifyHash(current, targetPath)
		if err != nil {
			return fmt.Errorf("failed to compare %s and %s: %w", current, targetPath, err)
		}
		if !match {
			return errDirSubsetMismatch
		}
		return nil
	})
	if errors.Is(err, errDirSubsetMismatch) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to verify copied directory subset %s against %s: %w", src, dst, err)
	}
	return true, nil
}

func removeCopiedDirSubset(src, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory %s: %w", src, err)
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("security: source directory is a symlink, which is not allowed: %s", src)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		entryInfo, err := os.Lstat(srcPath)
		if err != nil {
			return fmt.Errorf("failed to stat source entry %s: %w", srcPath, err)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("security: source directory contains a symlink, which is not allowed: %s", srcPath)
		}

		if entryInfo.IsDir() {
			dstInfo, err := os.Lstat(dstPath)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to stat target path %s: %w", dstPath, err)
			}
			if !dstInfo.IsDir() {
				return fmt.Errorf("managed target path is not a directory: %s", dstPath)
			}
			if err := removeCopiedDirSubset(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to remove copied directory subset %s from %s: %w", srcPath, dstPath, err)
			}
			if err := removeIfEmpty(dstPath); err != nil {
				return fmt.Errorf("failed to clean empty target directory %s: %w", dstPath, err)
			}
			continue
		}

		if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove target path %s: %w", dstPath, err)
		}
	}

	return nil
}

func removeIfEmpty(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	if len(entries) > 0 {
		return nil
	}
	if err := os.Remove(dir); err != nil {
		return fmt.Errorf("failed to remove empty directory %s: %w", dir, err)
	}
	return nil
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
