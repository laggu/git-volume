package gitvolume

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/shlex"
)

// GlobalEdit opens a file from the global git-volume directory in the default editor
func (g *GitVolume) GlobalEdit(file string) error {
	targetPath, editor, err := g.beforeEdit(file)
	if err == nil {
		err = g.edit(targetPath, editor)
	}

	return g.afterEdit(targetPath, err)
}

func (g *GitVolume) beforeEdit(file string) (targetPath, editor string, err error) {
	globalDir := g.ctx.GlobalDir
	targetPath = file // Initialize with input file for logging in case of early error
	editor = os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi if EDITOR is not set
	}

	// Check if global directory exists
	if _, statErr := os.Stat(globalDir); os.IsNotExist(statErr) {
		err = fmt.Errorf("global directory does not exist: %s", globalDir)
		return
	}

	// Resolve target path
	targetPath = filepath.Join(globalDir, file)

	// Security: check for path traversal
	if err = verifyPathWithinBase(targetPath, globalDir); err != nil {
		err = fmt.Errorf("invalid file path: %w", err)
		return
	}

	// Check if file exists
	if _, statErr := os.Stat(targetPath); os.IsNotExist(statErr) {
		err = fmt.Errorf("file does not exist: %s", targetPath)
		return
	}

	return
}

func (g *GitVolume) edit(targetPath, editor string) error {
	parts, err := shlex.Split(editor)
	if err != nil {
		return fmt.Errorf("failed to parse EDITOR value %q: %w", editor, err)
	}
	if len(parts) == 0 {
		return fmt.Errorf("invalid EDITOR value: empty command")
	}

	cmd := exec.Command(parts[0], append(parts[1:], targetPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor %s: %w", editor, err)
	}

	return nil
}

func (g *GitVolume) afterEdit(targetPath string, err error) error {
	if err != nil {
		if g.isNormalOrHigher() {
			fmt.Fprintf(os.Stderr, "❌ Failed to edit %s: %v\n", targetPath, err)
		}
		return err
	}

	if g.isDetailed() {
		fmt.Printf("✓ Edited %s\n", targetPath)
	}
	return nil
}
