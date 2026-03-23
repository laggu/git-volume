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
		return err
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("security: source directory is a symlink, which is not allowed: %s", src)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		entryInfo, err := os.Lstat(srcPath)
		if err != nil {
			return err
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
				return err
			}

			if err := copyDir(srcPath, dstPath); err != nil {
				return err
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
			return err
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return err
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

func hashDir(path string) (string, error) {
	h := sha256.New()

	err := filepath.WalkDir(path, func(current string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(path, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		info, err := os.Lstat(current)
		if err != nil {
			return err
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(current)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(h, "L|%s|%s\n", rel, target)
			return err
		case info.IsDir():
			_, err = fmt.Fprintf(h, "D|%s\n", rel)
			return err
		default:
			fileHash, err := hashFile(current)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(h, "F|%s|%s\n", rel, fileHash)
			return err
		}
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func verifyDirHash(dir1, dir2 string) (bool, error) {
	h1, err := hashDir(dir1)
	if err != nil {
		return false, err
	}
	h2, err := hashDir(dir2)
	if err != nil {
		return false, err
	}
	return h1 == h2, nil
}

var errDirSubsetMismatch = errors.New("directory subset mismatch")

func verifyDirSubset(src, dst string) (bool, error) {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return false, err
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("security: source directory is a symlink, which is not allowed: %s", src)
	}

	dstInfo, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !dstInfo.IsDir() {
		return false, nil
	}

	err = filepath.WalkDir(src, func(current string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		srcEntryInfo, err := os.Lstat(current)
		if err != nil {
			return err
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
			return err
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
			return err
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
		return false, err
	}
	return true, nil
}

func removeCopiedDirSubset(src, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("security: source directory is a symlink, which is not allowed: %s", src)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		entryInfo, err := os.Lstat(srcPath)
		if err != nil {
			return err
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
				return err
			}
			if !dstInfo.IsDir() {
				return fmt.Errorf("managed target path is not a directory: %s", dstPath)
			}
			if err := removeCopiedDirSubset(srcPath, dstPath); err != nil {
				return err
			}
			if err := removeIfEmpty(dstPath); err != nil {
				return err
			}
			continue
		}

		if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
			return err
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
		return err
	}
	if len(entries) > 0 {
		return nil
	}
	return os.Remove(dir)
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
