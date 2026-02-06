/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"

	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

// unsyncCmd represents the unsync command
var unsyncCmd = &cobra.Command{
	Use:   "unsync",
	Short: "Removes volumes from the current worktree",
	Long: `Removes the files or symlinks created by git volume sync.
It uses the current git-volume.yaml to identify what to remove.

For copied files, it verifies that the file content has not changed compared
to the source before deleting. If changed, it skips deletion to prevent data loss.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gv, err := gitvolume.New(gitvolume.Options{
			ConfigPath: cfgFile,
			Verbose:    verbose,
			Quiet:      quiet,
		})
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		if err := gv.Load(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !quiet {
			fmt.Printf("📂 Using config from: %s\n", gv.SourceDir())
		}

		if err := gv.Unsync(gitvolume.UnsyncOptions{
			DryRun: dryRun,
		}); err != nil {
			return err
		}

		if !quiet && !dryRun {
			fmt.Println("✓ Unsync complete")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unsyncCmd)
	unsyncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")
}
