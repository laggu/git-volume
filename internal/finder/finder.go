package finder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/laggu/git-volume/internal/config"
)

// Context encapsulates the execution context for git-volume
type Context struct {
	Config    *config.Config // Parsed configuration
	SourceDir string         // Base directory for resolving 'source' paths (where config lives)
	TargetDir string         // Base directory for resolving 'target' paths (current worktree root)
	GlobalDir string         // Resolved global directory absolute path for @global/ sources
}

// FindContextOptions configures the FindContext operation
type FindContextOptions struct {
	ConfigPath        string // Custom config file path (default: auto-detected)
	Quiet             bool   // Suppress non-error output
	GlobalDirOverride string // Override globalDir from config
}

// FindContext attempts to locate the git-volume.yaml and determine the context.
// If ConfigPath is provided in opts, it uses that file directly.
// Otherwise, it follows the inheritance logic:
// 1. Check current worktree root.
// 2. If not found, check main worktree (git-common-dir).
func FindContext(opts FindContextOptions) (*Context, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// 1. Find the root of the current git worktree
	worktreeRoot, err := FindGitWorktreeRoot(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to find git worktree root: %w", err)
	}

	override := opts.GlobalDirOverride

	// If custom config path is provided, use it directly
	if opts.ConfigPath != "" {
		absConfigPath := opts.ConfigPath
		if !filepath.IsAbs(opts.ConfigPath) {
			absConfigPath = filepath.Join(cwd, opts.ConfigPath)
		}
		cfg, err := config.LoadConfig(absConfigPath, opts.Quiet)
		if err != nil {
			return nil, err
		}
		globalDir, err := config.ResolveGlobalDir(cfg.GlobalDir, override)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve global directory: %w", err)
		}
		return &Context{
			Config:    cfg,
			SourceDir: filepath.Dir(absConfigPath),
			TargetDir: worktreeRoot,
			GlobalDir: globalDir,
		}, nil
	}

	// 2. Check for local override
	localConfigPath := filepath.Join(worktreeRoot, config.ConfigFileName)
	if _, err := os.Stat(localConfigPath); err == nil {
		cfg, err := config.LoadConfig(localConfigPath, opts.Quiet)
		if err != nil {
			return nil, err
		}
		globalDir, err := config.ResolveGlobalDir(cfg.GlobalDir, override)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve global directory: %w", err)
		}
		return &Context{
			Config:    cfg,
			SourceDir: worktreeRoot,
			TargetDir: worktreeRoot,
			GlobalDir: globalDir,
		}, nil
	}

	// 3. Fallback to main worktree (Inheritance)
	mainWorktreeRoot, err := findGitCommonDir(worktreeRoot)
	if err != nil {
		// If we can't find common dir, we might already be in main or not in a git repo properly
		// But if we are here, it means we didn't find local config.
		// If findGitCommonDir fails, it usually means we are in the main repo or standard repo.
		// So we just return error that config was not found.
		return nil, fmt.Errorf("config not found in current worktree, and failed to check main worktree: %v", err)
	}

	// If mainWorktreeRoot is different from worktreeRoot, check there
	if mainWorktreeRoot != "" && mainWorktreeRoot != worktreeRoot {
		mainConfigPath := filepath.Join(mainWorktreeRoot, config.ConfigFileName)
		if _, err := os.Stat(mainConfigPath); err == nil {
			cfg, err := config.LoadConfig(mainConfigPath, opts.Quiet)
			if err != nil {
				return nil, err
			}
			globalDir, err := config.ResolveGlobalDir(cfg.GlobalDir, override)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve global directory: %w", err)
			}
			return &Context{
				Config:    cfg,
				SourceDir: mainWorktreeRoot,
				TargetDir: worktreeRoot,
				GlobalDir: globalDir,
			}, nil
		}
	}

	return nil, fmt.Errorf("%s not found in current worktree or main worktree", config.ConfigFileName)
}

// FindGitWorktreeRoot returns the root directory of the current git worktree.
func FindGitWorktreeRoot(startDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = startDir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git rev-parse failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func findGitCommonDir(startDir string) (string, error) {
	// git rev-parse --git-common-dir returns the path to the .git dir of the main repo
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = startDir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git rev-parse --git-common-dir failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git rev-parse --git-common-dir failed: %w", err)
	}
	commonDir := strings.TrimSpace(string(out))

	var absCommonDir string
	if filepath.IsAbs(commonDir) {
		absCommonDir = commonDir
	} else {
		absCommonDir = filepath.Join(startDir, commonDir)
	}

	absCommonDir, err = filepath.Abs(absCommonDir)
	if err != nil {
		return "", err
	}

	// Check if this is a bare repository
	isBareCmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	isBareCmd.Dir = absCommonDir
	var isBare bool
	isBareOut, err := isBareCmd.Output()
	if err != nil {
		// If bare check fails, assume not bare (safer default)
		isBare = false
	} else {
		isBare = strings.TrimSpace(string(isBareOut)) == "true"
	}

	var mainRoot string
	if isBare {
		// For bare repos, the common dir itself is the repo root
		// Config files should be placed directly in the bare repo directory
		mainRoot = absCommonDir
	} else {
		// For regular repos, the root is the parent of .git
		mainRoot = filepath.Dir(absCommonDir)

		// Verify by checking if this looks like a valid git repo root
		verifyCmd := exec.Command("git", "rev-parse", "--show-toplevel")
		verifyCmd.Dir = mainRoot
		verifyOut, verifyErr := verifyCmd.Output()
		if verifyErr == nil {
			mainRoot = strings.TrimSpace(string(verifyOut))
		}
	}

	return mainRoot, nil
}
