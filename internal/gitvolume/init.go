package gitvolume

import (
	"fmt"
	"os"
	"path/filepath"
)

// Init initializes git-volume (creates global directory and sample config)
func (g *GitVolume) Init() error {
	// 1. Ensure Global Directory exists
	if err := os.MkdirAll(g.ctx.GlobalDir, DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create global directory %s: %w", g.ctx.GlobalDir, err)
	}
	if !g.quiet {
		fmt.Printf("✓ Global directory initialized: %s\n", g.ctx.GlobalDir)
	}

	// 2. Create Sample Config if not exists (at git root)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	gitRoot, err := FindWorktreeRoot(cwd)
	if err != nil {
		return fmt.Errorf("failed to find git repository root: %w", err)
	}
	configPath := filepath.Join(gitRoot, ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(SampleConfig), DefaultFilePerm); err != nil {
			return fmt.Errorf("failed to create sample config: %w", err)
		}
		if !g.quiet {
			fmt.Printf("✓ Created sample configuration: %s\n", configPath)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check config file: %w", err)
	} else {
		if !g.quiet {
			fmt.Printf("ℹ️  Configuration file already exists: %s\n", configPath)
		}
	}

	// 3. Guidance
	if !g.quiet {
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Add 'git-volume.yaml' to your .gitignore (optional but recommended)")
		fmt.Println("  2. Run 'git volume sync' to apply volumes")
	}
	return nil
}
