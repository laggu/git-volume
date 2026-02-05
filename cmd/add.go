/*
Copyright © 2026 laggu
*/
package cmd

import (
	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

var (
	addForce bool
	addAs    string
	addPath  string
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <file>...",
	Short: "Add files to the global git-volume directory",
	Long: `Copies one or more local files to the global git-volume directory (~/.git-volume).

This makes the files available to be mounted via @global/ prefix in git-volume.yaml.

Examples:
  git-volume add .env
  git-volume add .env config.json --force
  git-volume add secrets/api.key

  # Save with a different name/path (single file only)
  git-volume add .env.local --as .env
  git-volume add .env --as secrets/prod.env

  # Save multiple files to a subdirectory
  git-volume add .env config.json --path myproject`,
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gv, err := gitvolume.New(gitvolume.Options{Quiet: quiet})
		if err != nil {
			return err
		}
		return gv.Add(args, gitvolume.AddOptions{
			Force: addForce,
			As:    addAs,
			Path:  addPath,
		})
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVarP(&addForce, "force", "f", false, "overwrite existing files in global directory")
	addCmd.Flags().StringVarP(&addAs, "as", "a", "", "save as specific path/name (single file only)")
	addCmd.Flags().StringVarP(&addPath, "path", "p", "", "save to subdirectory within global directory")
	addCmd.MarkFlagsMutuallyExclusive("as", "path")
}
