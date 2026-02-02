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

// hashFile calculates SHA256 hash of a file
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
