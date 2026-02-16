package gitvolume

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// copyFile copies a file atomically using a temporary file and rename.
// If src is a symlink, it delegates to copySymlink.
func copyFile(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return copySymlink(src, dst)
	}
	return copyRegularFile(src, dst)
}

// copySymlink recreates a symlink at dst pointing to the same target as src.
func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("failed to read symlink %s: %w", src, err)
	}

	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, DefaultDirPerm); err != nil {
		return err
	}

	if _, err := os.Lstat(dst); err == nil {
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("failed to remove existing destination %s: %w", dst, err)
		}
	}

	return os.Symlink(target, dst)
}

// copyRegularFile copies a regular file atomically using a temporary file and rename.
func copyRegularFile(src, dst string) error {
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

	// Atomic rename (may fail on cross-filesystem)
	if err := os.Rename(tmpPath, dst); err != nil {
		if info, lErr := os.Lstat(dst); lErr == nil && info.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(dst)
		}
		if err := copyFileContent(tmpPath, dst); err != nil {
			return fmt.Errorf("rename failed and fallback copy also failed: %w", err)
		}
		_ = os.Remove(tmpPath)
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
	defer func() { _ = s.Close() }()

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	d, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}

	return d.Sync()
}

// copyDirNoSymlink recursively copies a directory after rejecting any symlink entry.
func copyDirNoSymlink(src, dst string) error {
	return copyDirNoSymlinkRecursive(src, dst)
}

func copyDirNoSymlinkRecursive(src, dst string) error {
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
			if err := copyDirNoSymlinkRecursive(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		if err := copyRegularFile(srcPath, dstPath); err != nil {
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
