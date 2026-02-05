package gitvolume

// List returns the status of all volumes
func (g *GitVolume) List() ([]VolumeStatus, error) {
	statuses := make([]VolumeStatus, 0, len(g.ctx.Volumes))
	for _, v := range g.ctx.Volumes {
		statuses = append(statuses, v.CheckStatus())
	}
	return statuses, nil
}
