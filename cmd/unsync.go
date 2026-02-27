/*
Copyright © 2026 laggu
*/
package cmd

import (
	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

// unsyncDryRun is the local flag for the unsync command
var unsyncDryRun bool

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
		gv, err := gitvolume.New(commandOptions(true))
		if err != nil {
			return err
		}

		return gv.Unsync(gitvolume.UnsyncOptions{
			DryRun: unsyncDryRun,
		})
	},
}

func init() {
	rootCmd.AddCommand(unsyncCmd)
	unsyncCmd.Flags().BoolVar(&unsyncDryRun, "dry-run", false, "show what would be done without making changes")
}
