package gitvolume

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// GlobalRemove removes files from the global git-volume directory
func (g *GitVolume) GlobalRemove(files []string) error {
	if err := g.beforeAllRemove(); err != nil {
		return err
	}

	var errs []error
	for _, file := range files {
		target, err := g.beforeRemove(file)
		if err == nil {
			err = g.remove(target)
		}
		g.afterRemove(file, err, &errs)
	}

	return g.afterAllRemove(errs)
}

func (g *GitVolume) beforeAllRemove() error {
	if _, err := os.Stat(g.ctx.GlobalDir); os.IsNotExist(err) {
		return fmt.Errorf("global storage not initialized")
	}
	return nil
}

func (g *GitVolume) afterAllRemove(errs []error) error {
	if len(errs) > 0 && g.isNormalOrHigher() {
		fmt.Fprintf(os.Stderr, "❌ Global remove completed with %d error(s)\n", len(errs))
	}
	return errors.Join(errs...)
}

func (g *GitVolume) beforeRemove(file string) (string, error) {
	globalDir := g.ctx.GlobalDir

	// Target path is directly relative to globalDir
	targetPath := filepath.Join(globalDir, file)

	// Security check: ensure targetPath is within globalDir
	if err := verifyPathWithinBase(targetPath, globalDir); err != nil {
		return "", fmt.Errorf("invalid path %s: %w", file, err)
	}

	// Check existence
	if _, err := os.Lstat(targetPath); os.IsNotExist(err) {
		return "", fmt.Errorf("not found: %s", file)
	} else if err != nil {
		return "", fmt.Errorf("failed to stat %s: %w", file, err)
	}

	return targetPath, nil
}

func (g *GitVolume) remove(file string) error {
	// Remove
	if err := os.RemoveAll(file); err != nil {
		return fmt.Errorf("failed to remove: %w", err)
	}

	// Cleanup empty parents
	cleanEmptyParents(filepath.Dir(file), g.ctx.GlobalDir)

	return nil
}

func (g *GitVolume) afterRemove(file string, err error, errs *[]error) {
	if err != nil {
		if g.isNormalOrHigher() {
			fmt.Fprintf(os.Stderr, "❌ Failed to remove %s: %v\n", file, err)
		}
		*errs = append(*errs, err)
		return
	}

	if g.isDetailed() {
		fmt.Printf("✓ Removed %s\n", file)
	}
}
