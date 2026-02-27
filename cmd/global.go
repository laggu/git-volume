/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laggu/git-volume/internal/gitvolume"
)

// globalCmd represents the global command
var globalCmd = &cobra.Command{
	Use:   "global",
	Short: "Manage files in the global git-volume storage",
	Long: `Commands for managing files in the global storage directory (~/.git-volume).
These files can be mounted in any project using the @global/ prefix.`,
}

// globalListCmd represents the global list command
var globalListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List files in global storage",
	Long:         `Displays a tree of all files currently stored in the global git-volume directory.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gv, err := gitvolume.New(commandOptions(false))
		if err != nil {
			return err
		}

		if err := gv.GlobalList(); err != nil {
			return fmt.Errorf("failed to list global files: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(globalCmd)
	globalCmd.AddCommand(globalListCmd)
}
