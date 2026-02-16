package gitvolume

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// copyFile copies a regular file atomically using a temporary file and rename.
func copyFile(src, dst string) error {
	// Ensure directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, DefaultDirPerm); err != nil {
		return err
	}
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

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
			_ = os.Remove(tmpPath)
		}
	}()

	// Copy content
	if _, err := io.Copy(tmpFile, s); err != nil {
		_ = tmpFile.Close()
		return err
	}

	// Sync to ensure data is written to disk
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	// Copy permissions
	if err := os.Chmod(tmpPath, srcInfo.Mode()); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpPath, dst); err != nil {
		dstInfo, statErr := os.Lstat(dst)
		if statErr == nil {
			if dstInfo.IsDir() {
				return fmt.Errorf("failed to atomically replace destination: destination is a directory: %s", dst)
			}
			if dstInfo.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("failed to atomically replace destination: destination is a symlink: %s", dst)
			}
			if rmErr := os.Remove(dst); rmErr != nil {
				return fmt.Errorf("failed to remove existing destination %s after rename error: %w", dst, rmErr)
			}
			if retryErr := os.Rename(tmpPath, dst); retryErr == nil {
				success = true
				return nil
			}
		}
		return fmt.Errorf("failed to atomically replace destination: %w", err)
	}

	success = true
	return nil
}

// copyDirNoSymlink recursively copies a directory after rejecting any symlink entry.
func copyDirNoSymlink(src, dst string) error {
	return copyDirNoSymlinkWithForce(src, dst, true)
}

func copyDirNoSymlinkWithForce(src, dst string, force bool) error {
	return copyDirNoSymlinkRecursive(src, dst, force)
}

func copyDirNoSymlinkRecursive(src, dst string, force bool) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("source directory contains a symlink, which is not allowed for security reasons: %s", src)
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
			return fmt.Errorf("source directory contains a symlink, which is not allowed for security reasons: %s", srcPath)
		}

		if entryInfo.IsDir() {
			if dstInfo, err := os.Lstat(dstPath); err == nil && !dstInfo.IsDir() {
				return fmt.Errorf("target exists and is not a directory: %s", dstPath)
			} else if err != nil && !os.IsNotExist(err) {
				return err
			}

			if err := copyDirNoSymlinkRecursive(srcPath, dstPath, force); err != nil {
				return err
			}
			continue
		}

		if _, err := os.Lstat(dstPath); err == nil {
			if !force {
				continue
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
