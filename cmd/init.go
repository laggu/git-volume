/*
Copyright © 2026 laggu
*/
package cmd

import (
	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize git-volume in the current project",
	Long: `Creates the global ~/.git-volume directory and generates a sample
git-volume.yaml configuration file in the current directory.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return gitvolume.Init(gitvolume.InitOptions{Quiet: quiet})
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
