package gitvolume

import (
	"fmt"
	"os"
	"path/filepath"
)

type initState struct {
	configPath    string
	configCreated bool
	configExists  bool
}

// Init initializes git-volume (creates global directory and sample config)
func (g *GitVolume) Init() error {
	state := &initState{}
	err := g.beforeInit(state)
	if err == nil {
		err = g.init(state)
	}
	return g.afterInit(state, err)
}

func (g *GitVolume) beforeInit(state *initState) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	gitRoot, err := findInitRoot(cwd)
	if err != nil {
		return fmt.Errorf("failed to find git repository root: %w", err)
	}
	state.configPath = filepath.Join(gitRoot, ConfigFileName)
	return nil
}

// findInitRoot returns the directory where git-volume.yaml should be created.
// In a normal repository or worktree, that is the current worktree root.
// In a bare repository, that is the bare repository root itself.
func findInitRoot(startDir string) (string, error) {
	worktreeRoot, err := FindWorktreeRoot(startDir)
	if err == nil {
		return worktreeRoot, nil
	}

	commonDir, commonErr := findCommonDir(startDir)
	if commonErr == nil {
		return commonDir, nil
	}

	return "", fmt.Errorf("could not determine repository root: %w", commonErr)
}

func (g *GitVolume) init(state *initState) error {
	if err := os.MkdirAll(g.ctx.GlobalDir, DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create global directory %s: %w", g.ctx.GlobalDir, err)
	}

	if _, err := os.Stat(state.configPath); os.IsNotExist(err) {
		if err := os.WriteFile(state.configPath, []byte(SampleConfig), DefaultFilePerm); err != nil {
			return fmt.Errorf("failed to create sample config: %w", err)
		}
		state.configCreated = true
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check config file: %w", err)
	}

	state.configExists = true
	return nil
}

func (g *GitVolume) afterInit(state *initState, err error) error {
	if err != nil {
		if !g.quiet {
			fmt.Printf("❌ Init failed: %v\n", err)
		}
		return err
	}

	if g.quiet {
		return nil
	}

	fmt.Printf("✓ Global directory initialized: %s\n", g.ctx.GlobalDir)
	if state.configCreated {
		fmt.Printf("✓ Created sample configuration: %s\n", state.configPath)
	} else if state.configExists {
		fmt.Printf("ℹ️  Configuration file already exists: %s\n", state.configPath)
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add 'git-volume.yaml' to your .gitignore (optional but recommended)")
	fmt.Println("  2. Run 'git volume sync' to apply volumes")

	return nil
}
