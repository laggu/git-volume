/*
Copyright © 2026 laggu
*/
package cmd

import (
	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

// globalRemoveCmd represents the global remove command
var globalRemoveCmd = &cobra.Command{
	Use:   "remove <file>...",
	Short: "Remove files from global storage",
	Long: `Removes files or directories from the global git-volume storage (~/.git-volume).

Examples:
  git volume global remove .env
  git volume global remove secrets/api.key`,
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	Aliases:      []string{"rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		gv, err := gitvolume.New(gitvolume.Options{Quiet: quiet})
		if err != nil {
			return err
		}

		return gv.GlobalRemove(args)
	},
}

func init() {
	globalCmd.AddCommand(globalRemoveCmd)
}
