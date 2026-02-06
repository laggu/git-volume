/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"

	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
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
		gv, err := gitvolume.New(gitvolume.Options{
			ConfigPath:        cfgFile,
			GlobalDirOverride: globalDir,
			Verbose:           verbose,
			Quiet:             quiet,
		})
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		if err := gv.Load(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !quiet {
			fmt.Printf("📂 Using config from: %s\n", gv.SourceDir())
			fmt.Printf("🎯 Target worktree: %s\n", gv.TargetDir())
			if gv.HasGlobalVolumes() {
				fmt.Printf("🌐 Global directory: %s\n", gv.GlobalDir())
			}
		}

		if err := gv.Sync(gitvolume.SyncOptions{
			DryRun:        dryRun,
			RelativeLinks: relativeLinks,
		}); err != nil {
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
