package gitvolume

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// verifyPathWithinBase checks that a path's parent directory resolves to a location within the base directory,
// even when intermediate path components are symlinks. This prevents symlink-based path traversal attacks.
// Note: This checks the parent directory, not the file itself, because the file might be a symlink we manage.
//
// Security: This function iteratively checks each component of the path from the base down,
// ensuring no existing component resolves to a location outside the base directory.
// This prevents attacks where a malicious symlink in an intermediate directory could
// cause os.MkdirAll to create directories outside the intended workspace.
func verifyPathWithinBase(path, base string) error {
	// Resolve the base directory
	resolvedBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}
	resolvedBase = filepath.Clean(resolvedBase)

	// Get relative path from base to target's parent directory
	parentDir := filepath.Dir(path)
	relPath, err := filepath.Rel(base, parentDir)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// If the relative path starts with "..", it's outside the base
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path escapes base directory: %s", path)
	}

	// If it's the base itself, it's valid
	if relPath == "." {
		return nil
	}

	// Split the relative path into components and check each one iteratively
	components := strings.Split(relPath, string(filepath.Separator))
	currentPath := base

	for _, component := range components {
		if component == "" || component == "." {
			continue
		}

		currentPath = filepath.Join(currentPath, component)

		// Check if this path component exists
		info, err := os.Lstat(currentPath)
		if err != nil {
			if os.IsNotExist(err) {
				// This component doesn't exist yet, and all previous components
				// have been verified to be within base. Safe to stop here.
				// os.MkdirAll will create this directory within the verified path.
				return nil
			}
			return fmt.Errorf("failed to stat path component: %w", err)
		}

		// If it's a symlink, resolve it and verify it stays within base
		if info.Mode()&os.ModeSymlink != 0 {
			resolvedPath, err := filepath.EvalSymlinks(currentPath)
			if err != nil {
				return fmt.Errorf("failed to resolve symlink %s: %w", currentPath, err)
			}
			resolvedPath = filepath.Clean(resolvedPath)

			// Verify the resolved path is within the base directory
			if !strings.HasPrefix(resolvedPath+string(filepath.Separator), resolvedBase+string(filepath.Separator)) &&
				resolvedPath != resolvedBase {
				return fmt.Errorf("path escapes base directory via symlink: %s resolves to %s", currentPath, resolvedPath)
			}
		}
	}

	return nil
}

// pathsEqual compares two paths after normalizing them
// This handles differences in path separators and relative vs absolute paths
func pathsEqual(path1, path2 string) bool {
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
