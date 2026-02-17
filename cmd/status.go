/*
Copyright © 2026 laggu
*/
package cmd

import (
	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show the status of all volumes",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gv, err := gitvolume.New(gitvolume.Options{
			ConfigPath: cfgFile,
			Quiet:      quiet,
		})
		if err != nil {
			return err
		}

		return gv.RunStatus()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
