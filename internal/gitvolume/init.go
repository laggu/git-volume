package gitvolume

import (
	"errors"
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
	state, err := g.beforeAllInit()
	if err != nil {
		return err
	}

	steps := []string{"global-dir", "config"}
	var errs []error

	for _, step := range steps {
		if err := g.beforeInit(step, state); err != nil {
			g.afterInit(step, state, err, &errs)
			continue
		}

		err := g.init(step, state)
		g.afterInit(step, state, err, &errs)
	}

	return g.afterAllInit(state, errs)
}

func (g *GitVolume) beforeAllInit() (*initState, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	gitRoot, err := FindWorktreeRoot(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to find git repository root: %w", err)
	}

	return &initState{configPath: filepath.Join(gitRoot, ConfigFileName)}, nil
}

func (g *GitVolume) beforeInit(step string, state *initState) error {
	return nil
}

func (g *GitVolume) init(step string, state *initState) error {
	switch step {
	case "global-dir":
		if err := os.MkdirAll(g.ctx.GlobalDir, DefaultDirPerm); err != nil {
			return fmt.Errorf("failed to create global directory %s: %w", g.ctx.GlobalDir, err)
		}
		return nil
	case "config":
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
	default:
		return fmt.Errorf("unknown init step: %s", step)
	}
}

func (g *GitVolume) afterInit(step string, state *initState, err error, errs *[]error) {
	if err != nil {
		if !g.quiet {
			fmt.Printf("❌ Failed to initialize %s: %v\n", step, err)
		}
		*errs = append(*errs, err)
		return
	}

	if g.quiet {
		return
	}

	switch step {
	case "global-dir":
		fmt.Printf("✓ Global directory initialized: %s\n", g.ctx.GlobalDir)
	case "config":
		if state.configCreated {
			fmt.Printf("✓ Created sample configuration: %s\n", state.configPath)
		} else if state.configExists {
			fmt.Printf("ℹ️  Configuration file already exists: %s\n", state.configPath)
		}
	}
}

func (g *GitVolume) afterAllInit(state *initState, errs []error) error {
	if len(errs) > 0 {
		if !g.quiet {
			fmt.Printf("❌ Init completed with %d error(s)\n", len(errs))
		}
		return errors.Join(errs...)
	}

	if !g.quiet {
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Add 'git-volume.yaml' to your .gitignore (optional but recommended)")
		fmt.Println("  2. Run 'git volume sync' to apply volumes")
	}

	return nil
}
