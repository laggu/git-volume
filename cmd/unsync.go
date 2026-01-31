/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"

	"github.com/laggu/git-volume/internal/finder"
	"github.com/laggu/git-volume/internal/mounter"
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
		// 1. Find Context
		ctx, err := finder.FindContext(cfgFile, quiet)
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		if !quiet {
			fmt.Printf("📂 Using config from: %s\n", ctx.SourceDir)
		}

		// 2. Execute Unsync
		mnt := mounter.New(ctx.SourceDir, ctx.TargetDir)
		opts := mounter.UnsyncOptions{
			DryRun:  dryRun,
			Verbose: verbose,
			Quiet:   quiet,
		}
		if err := mnt.Unsync(ctx.Config.Volumes, opts); err != nil {
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
