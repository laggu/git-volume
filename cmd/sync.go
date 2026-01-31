/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laggu/git-volume/internal/finder"
	"github.com/laggu/git-volume/internal/mounter"
)

var (
	dryRun        bool
	relativeLinks bool
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Mounts defined volumes into the current worktree",
	Long: `Reads git-volume.yaml configuration and applies the defined volumes
to the current worktree.

It checks the current directory for git-volume.yaml first. If not found,
it looks for it in the main Git worktree (inheritance).`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Find Context (Config + Dirs)
		ctx, err := finder.FindContext(cfgFile, quiet)
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		if !quiet {
			fmt.Printf("📂 Using config from: %s\n", ctx.SourceDir)
			fmt.Printf("🎯 Target worktree: %s\n", ctx.TargetDir)
		}

		// 2. Execute Sync
		mnt := mounter.New(ctx.SourceDir, ctx.TargetDir)
		opts := mounter.SyncOptions{
			DryRun:        dryRun,
			RelativeLinks: relativeLinks,
			Verbose:       verbose,
			Quiet:         quiet,
		}
		if err := mnt.Sync(ctx.Config.Volumes, opts); err != nil {
			return err
		}

		if !quiet && !dryRun {
			fmt.Println("✓ Volumes successfully synced")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")
	syncCmd.Flags().BoolVar(&relativeLinks, "relative", false, "create relative symlinks instead of absolute")
}
