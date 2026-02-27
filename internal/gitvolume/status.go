package gitvolume

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

func (g *GitVolume) Status() error {
	if err := g.beforeAllStatus(); err != nil {
		return err
	}

	statuses, err := g.status()
	return g.afterAllStatus(statuses, err)
}

func (g *GitVolume) beforeAllStatus() error {
	if err := g.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if g.isNormalOrHigher() {
		fmt.Printf("📂 Source Config: %s\n", filepath.Join(g.SourceDir(), ConfigFileName))
		fmt.Printf("🎯 Target Root:   %s\n", g.TargetDir())
		if g.HasGlobalVolumes() {
			fmt.Printf("🌐 Global Dir:    %s\n", g.GlobalDir())
		}
		fmt.Println()
	}

	return nil
}

func (g *GitVolume) status() ([]VolumeStatus, error) {
	statuses := make([]VolumeStatus, 0, len(g.ctx.Volumes))
	for _, v := range g.ctx.Volumes {
		statuses = append(statuses, v.CheckStatus())
	}
	return statuses, nil
}

func (g *GitVolume) afterAllStatus(statuses []VolumeStatus, err error) error {
	if err != nil {
		return err
	}
	if !g.isNormalOrHigher() {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "SOURCE\tTARGET\tMODE\tSTATUS")
	for _, s := range statuses {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Source, s.Target, s.Mode, s.Status)
	}
	_ = w.Flush()
	return nil
}
