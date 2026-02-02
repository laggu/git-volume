/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/laggu/git-volume/internal/gitvolume"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "Lists all volumes and their status",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gv, err := gitvolume.New(gitvolume.Options{
			ConfigPath:        cfgFile,
			GlobalDirOverride: globalDir,
			Quiet:             quiet,
		})
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		if !quiet {
			fmt.Printf("📂 Source Config: %s\n", filepath.Join(gv.SourceDir(), gitvolume.ConfigFileName))
			fmt.Printf("🎯 Target Root:   %s\n", gv.TargetDir())
			if gv.HasGlobalVolumes() {
				fmt.Printf("🌐 Global Dir:    %s\n", gv.GlobalDir())
			}
			fmt.Println()
		}

		statuses, err := gv.List()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "SOURCE\tTARGET\tMODE\tSTATUS")
		for _, s := range statuses {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Source, s.Target, s.Mode, s.Status)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
