/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// worktreeCmd represents the worktree command
var worktreeCmd = &cobra.Command{
	Use:   "worktree [add] [path] [branch]",
	Short: "Wrapper for git worktree that automatically syncs volumes",
	Long: `Creates a new git worktree and immediately runs 'git volume sync' inside it.
Example: git volume worktree add ../feature-branch feature-branch`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Verify args
		if len(args) < 3 || args[0] != "add" {
			return fmt.Errorf("usage: git volume worktree add <path> <branch>")
		}

		path := args[1]
		branch := args[2]

		// Validate arguments to prevent option injection
		if strings.HasPrefix(path, "-") {
			return fmt.Errorf("invalid path: cannot start with '-'")
		}
		if strings.HasPrefix(branch, "-") {
			return fmt.Errorf("invalid branch: cannot start with '-'")
		}
		if path == "" || branch == "" {
			return fmt.Errorf("path and branch cannot be empty")
		}

		// 1. Run git worktree add
		fmt.Printf("🔨 Creating worktree '%s' at '%s'...\n", branch, path)
		gitCmd := exec.Command("git", "worktree", "add", path, branch)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			return fmt.Errorf("git worktree add failed: %w", err)
		}

		// 2. Resolve absolute path for the new worktree
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// 3. Run git volume sync inside the new worktree
		selfExe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine executable path: %w", err)
		}

		fmt.Printf("\n🔄 Syncing volumes in %s...\n", absPath)
		syncCmd := exec.Command(selfExe, "sync")
		syncCmd.Dir = absPath
		syncCmd.Stdout = os.Stdout
		syncCmd.Stderr = os.Stderr

		if err := syncCmd.Run(); err != nil {
			fmt.Printf("⚠️  Volume sync failed: %v\n", err)
			return fmt.Errorf("worktree created but volume sync failed")
		}

		fmt.Println("✨ Worktree ready with volumes mounted!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(worktreeCmd)
}
