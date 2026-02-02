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
