/*
Copyright © 2026 laggu
*/
package cmd

import (
	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit <file>",
	Short: "Edit a file in the global git-volume directory",
	Long: `Opens a file from the global git-volume directory in your default editor.
The editor is determined by the EDITOR environment variable, defaulting to 'vi'.

Examples:
  git volume global edit config.json
  git volume global edit secrets/api.key`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gv, err := gitvolume.New(commandOptions(false))
		if err != nil {
			return err
		}
		return gv.GlobalEdit(args[0])
	},
}

func init() {
	globalCmd.AddCommand(editCmd)
}
