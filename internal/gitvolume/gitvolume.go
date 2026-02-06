package gitvolume

// GitVolume is the main entry point for git-volume operations
type GitVolume struct {
	ctx        *Context
	verbose    bool
	quiet      bool
	configPath string // stored for Load()
}

// Options configures GitVolume creation
type Options struct {
	ConfigPath string // Custom config file path (optional)
	Verbose    bool   // Verbose output
	Quiet      bool   // Suppress non-error output
}

// New creates a new GitVolume instance without loading config
func New(opts Options) (*GitVolume, error) {
	ctx, err := NewContext()
	if err != nil {
		return nil, err
	}
	return &GitVolume{
		ctx:        ctx,
		verbose:    opts.Verbose,
		quiet:      opts.Quiet,
		configPath: opts.ConfigPath,
	}, nil
}

// Load loads configuration from the config file.
// Must be called before Sync, Unsync, or List operations.
func (g *GitVolume) Load() error {
	return g.ctx.Load(g.configPath, g.quiet)
}

// Context returns the underlying context
func (g *GitVolume) Context() *Context { return g.ctx }

// SourceDir returns the source directory (where config lives)
func (g *GitVolume) SourceDir() string { return g.ctx.SourceDir }

// TargetDir returns the target directory (current worktree root)
func (g *GitVolume) TargetDir() string { return g.ctx.TargetDir }

// GlobalDir returns the global directory for @global/ sources
func (g *GitVolume) GlobalDir() string { return g.ctx.GlobalDir }

// HasGlobalVolumes returns true if any volume uses @global/ prefix
func (g *GitVolume) HasGlobalVolumes() bool { return g.ctx.HasGlobalVolumes() }
