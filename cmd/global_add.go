/*
Copyright © 2026 laggu
*/
package cmd

import (
	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

var (
	addAs   string
	addPath string
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <file>...",
	Short: "Add files to the global git-volume directory",
	Long: `Copies one or more local files to the global git-volume directory (~/.git-volume).

This makes the files available to be mounted via @global/ prefix in git-volume.yaml.

Examples:
  git volume global add .env
  git volume global add secrets/api.key

  # Save with a different name/path (single file only)
  git volume global add .env.local --as .env
  git volume global add .env --as secrets/prod.env

  # Save multiple files to a subdirectory
  git volume global add .env config.json --path myproject`,
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gv, err := gitvolume.New(gitvolume.Options{Quiet: quiet})
		if err != nil {
			return err
		}
		return gv.GlobalAdd(args, gitvolume.AddOptions{
			As:   addAs,
			Path: addPath,
		})
	},
}

func init() {
	globalCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addAs, "as", "a", "", "save as specific path/name (single file only)")
	addCmd.Flags().StringVarP(&addPath, "path", "p", "", "save to subdirectory within global directory")
	addCmd.MarkFlagsMutuallyExclusive("as", "path")
}
