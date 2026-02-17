package gitvolume

import (
	"crypto/sha256"
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

func copyDir(src, dst string) error {
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

			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		// If target exists, verify it's a regular file or symlink
		if info, err := os.Lstat(dstPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(dstPath); err != nil {
					return fmt.Errorf("failed to remove existing symlink: %w", err)
				}
			} else if !info.Mode().IsRegular() {
				return fmt.Errorf("target exists and is not a regular file: %s", dstPath)
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
