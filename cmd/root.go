/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"

	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

// Global flags
var (
	cfgFile   string
	verbosity int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "git-volume",
	Short: "Manage environment files across Git worktrees",
	Long: `git-volume manages environment files (.env, secrets, etc.) across Git worktrees
by dynamically mounting them using a git-volume.yaml manifest.

"Keep code in Git, mount environments as volumes."`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if verbosity < gitvolume.VerbosityQuiet || verbosity > gitvolume.VerbosityDetailed {
			return fmt.Errorf("invalid --verbose level %d (allowed: 0, 1, 2)", verbosity)
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func commandOptions(useConfig bool) gitvolume.Options {
	opts := gitvolume.Options{Verbosity: verbosity}
	if useConfig {
		opts.ConfigPath = cfgFile
	}
	return opts
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default: auto-detected)")
	rootCmd.PersistentFlags().IntVarP(&verbosity, "verbose", "v", gitvolume.VerbosityNormal, "verbosity level (0=errors only, 1=normal, 2=detailed)")
	if flag := rootCmd.PersistentFlags().Lookup("verbose"); flag != nil {
		flag.NoOptDefVal = "2"
	}
}
