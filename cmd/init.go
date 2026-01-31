/*
Copyright © 2026 laggu
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/laggu/git-volume/internal/config"
	"github.com/laggu/git-volume/internal/finder"
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
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		// 1. Create Global Directory
		globalDir := filepath.Join(home, ".git-volume")
		if err := os.MkdirAll(globalDir, config.DefaultDirPerm); err != nil {
			return fmt.Errorf("failed to create global directory %s: %w", globalDir, err)
		}
		if !quiet {
			fmt.Printf("✓ Global directory initialized: %s\n", globalDir)
		}

		// 2. Create Sample Config if not exists (at git root)
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		gitRoot, err := finder.FindGitWorktreeRoot(cwd)
		if err != nil {
			return fmt.Errorf("failed to find git repository root: %w", err)
		}
		configPath := filepath.Join(gitRoot, config.ConfigFileName)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if err := os.WriteFile(configPath, []byte(config.SampleConfig), config.DefaultFilePerm); err != nil {
				return fmt.Errorf("failed to create sample config: %w", err)
			}
			if !quiet {
				fmt.Printf("✓ Created sample configuration: %s\n", configPath)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check config file: %w", err)
		} else {
			if !quiet {
				fmt.Printf("ℹ️  Configuration file already exists: %s\n", configPath)
			}
		}

		// 3. Guidance
		if !quiet {
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Add 'git-volume.yaml' to your .gitignore (optional but recommended)")
			fmt.Println("  2. Run 'git volume sync' to apply volumes")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
