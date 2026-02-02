/*
Copyright © 2026 laggu
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Global flags
var (
	cfgFile   string
	globalDir string
	verbose   bool
	quiet     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "git-volume",
	Short: "Manage environment files across Git worktrees",
	Long: `git-volume manages environment files (.env, secrets, etc.) across Git worktrees
by dynamically mounting them using a git-volume.yaml manifest.

"Keep code in Git, mount environments as volumes."`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default: auto-detected)")
	rootCmd.PersistentFlags().StringVar(&globalDir, "global-dir", "", "global directory for @global/ sources (default: ~/.git-volume)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet")
}
