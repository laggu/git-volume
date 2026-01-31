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
	Use:   "worktree",
	Short: "Wrapper for git worktree that automatically syncs volumes",
	Long: `Wrapper for git worktree commands. Currently supports 'add' subcommand.
After creating the worktree, it automatically runs 'git volume sync' inside it.`,
}

// worktreeAddCmd represents the worktree add subcommand
var worktreeAddCmd = &cobra.Command{
	Use:   "add <path> [<commit-ish>] [-- <git-worktree-options>...]",
	Short: "Create a worktree and sync volumes",
	Long: `Creates a new git worktree and immediately runs 'git volume sync' inside it.

All arguments after 'add' are passed directly to 'git worktree add', allowing
full flexibility with git worktree options like -b, --force, etc.

Examples:
  git volume worktree add ../feature-branch feature-branch
  git volume worktree add -b new-branch ../new-branch main
  git volume worktree add --detach ../detached-worktree HEAD~3`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Args:                  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find the path argument (first non-flag argument)
		var path string
		for _, arg := range args {
			if !strings.HasPrefix(arg, "-") {
				path = arg
				break
			}
		}

		if path == "" {
			return fmt.Errorf("path argument is required")
		}

		// 1. Run git worktree add with all provided arguments
		if !quiet {
			fmt.Printf("🔨 Creating worktree at '%s'...\n", path)
		}
		gitArgs := append([]string{"worktree", "add"}, args...)
		gitCmd := exec.Command("git", gitArgs...)
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

		if !quiet {
			fmt.Printf("\n🔄 Syncing volumes in %s...\n", absPath)
		}
		syncCmd := exec.Command(selfExe, "sync")
		syncCmd.Dir = absPath
		syncCmd.Stdout = os.Stdout
		syncCmd.Stderr = os.Stderr

		if err := syncCmd.Run(); err != nil {
			fmt.Printf("⚠️  Volume sync failed: %v\n", err)
			return fmt.Errorf("worktree created but volume sync failed")
		}

		if !quiet {
			fmt.Println("✨ Worktree ready with volumes mounted!")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(worktreeCmd)
	worktreeCmd.AddCommand(worktreeAddCmd)
}
