package gitvolume

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

func (g *GitVolume) RunSync(opts SyncOptions) error {
	if err := g.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !g.quiet {
		fmt.Printf("📂 Using config from: %s\n", g.SourceDir())
		fmt.Printf("🎯 Target worktree: %s\n", g.TargetDir())
		if g.HasGlobalVolumes() {
			fmt.Printf("🌐 Global directory: %s\n", g.GlobalDir())
		}
	}

	if err := g.Sync(opts); err != nil {
		return err
	}

	if !g.quiet && !opts.DryRun {
		fmt.Println("✓ Volumes successfully synced")
	}
	return nil
}

func (g *GitVolume) RunUnsync(opts UnsyncOptions) error {
	if err := g.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !g.quiet {
		fmt.Printf("📂 Using config from: %s\n", g.SourceDir())
	}

	if err := g.Unsync(opts); err != nil {
		return err
	}

	if !g.quiet && !opts.DryRun {
		fmt.Println("✓ Unsync complete")
	}
	return nil
}

func (g *GitVolume) RunStatus() error {
	if err := g.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !g.quiet {
		fmt.Printf("📂 Source Config: %s\n", filepath.Join(g.SourceDir(), ConfigFileName))
		fmt.Printf("🎯 Target Root:   %s\n", g.TargetDir())
		if g.HasGlobalVolumes() {
			fmt.Printf("🌐 Global Dir:    %s\n", g.GlobalDir())
		}
		fmt.Println()
	}

	statuses, err := g.Status()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "SOURCE\tTARGET\tMODE\tSTATUS")
	for _, s := range statuses {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Source, s.Target, s.Mode, s.Status)
	}
	_ = w.Flush()
	return nil
}
