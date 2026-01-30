/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/laggu/git-volume/internal/config"
	"github.com/laggu/git-volume/internal/finder"
	"github.com/laggu/git-volume/internal/mounter"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "Lists all volumes and their status",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := finder.FindContext(cfgFile)
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		if !quiet {
			fmt.Printf("📂 Source Config: %s\n", filepath.Join(ctx.SourceDir, config.ConfigFileName))
			fmt.Printf("🎯 Target Root:   %s\n\n", ctx.TargetDir)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "SOURCE\tTARGET\tMODE\tSTATUS")

		for _, v := range ctx.Config.Volumes {
			status := checkStatus(v, ctx.SourceDir, ctx.TargetDir)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", v.Source, v.Target, v.Mode, status)
		}
		w.Flush()
		return nil
	},
}

func checkStatus(v config.Volume, srcBase, targetBase string) string {
	srcPath := filepath.Join(srcBase, v.Source)
	targetPath := filepath.Join(targetBase, v.Target)

	// Check Source
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "MISSING (Source)"
	}

	// Check Target
	info, err := os.Lstat(targetPath)
	if os.IsNotExist(err) {
		return "NOT MOUNTED"
	}
	if err != nil {
		return fmt.Sprintf("ERROR (%v)", err)
	}

	if v.Mode == config.ModeLink {
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(targetPath)
			if err != nil {
				return "ERROR (readlink)"
			}
			if mounter.PathsEqual(link, srcPath) {
				return "OK (Linked)"
			}
			return "WRONG LINK"
		}
		return "EXISTS (Not Link)"
	}

	// Copy Mode
	if info.Mode().IsRegular() {
		return "OK (Copied)" // We could check hash here for "MODIFIED" status but strict check might be slow for list
	}
	return "EXISTS (Not File)"
}

func init() {
	rootCmd.AddCommand(listCmd)
}
