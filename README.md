> 🌐 [한국어](docs/ko/README.md)

# git-volume

> **"Keep code in Git, mount your environment as volumes."**

`git-volume` is a CLI tool that centrally manages environment files (`.env`, secrets, etc.) across Git worktrees and dynamically mounts them.

## ✨ Key Features

- **Volume Mounting**: Supports symbolic links or file copy modes
- **Configuration Inheritance**: Child worktrees automatically inherit parent settings
- **Safe Cleanup**: Modified files are preserved during unsync
- **AI Agent Optimized**: Create worktree + configure environment with a single command

## 📦 Installation

### Homebrew
```bash
brew install laggu/tap/git-volume
```

### Scoop (Windows)
```bash
scoop bucket add laggu https://github.com/laggu/scoop-bucket.git
scoop install git-volume
```

### Go
```bash
go install github.com/laggu/git-volume@latest
```

## 🚀 Quick Start

**1. Initialize**
```bash
git volume init
```

**2. Create git-volume.yaml**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. Mount Volumes**
```bash
git volume sync
```

**4. Check Status**
```bash
git volume list
```

## 📖 Commands

| Command             | Description                                              |
| ------------------- | -------------------------------------------------------- |
| `git volume init`   | Create global directory and sample configuration file    |
| `git volume sync`   | Mount volumes to current worktree based on configuration |
| `git volume unsync` | Remove mounted volumes (modified files are preserved)    |
| `git volume list`   | Display current volume status                            |

## ⚙️ Configuration File (`git-volume.yaml`)

```yaml
volumes:
  # Simple format (default: symbolic link)
  - ".env.shared:.env"

  # With options
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (default) or copy
    force: true    # overwrite if exists
```

### Mode Comparison

| Mode   | Description          | Use Case                                             |
| ------ | -------------------- | ---------------------------------------------------- |
| `link` | Create symbolic link | Local development (changes reflect immediately)      |
| `copy` | Copy file            | Docker builds (environments without symlink support) |

## 🔄 Worktree Inheritance

If a child worktree doesn't have `git-volume.yaml`, it automatically uses the parent (main) worktree's configuration.

```bash
# Configuration exists only in main worktree
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # configuration file
├── .env.shared       # source file
└── ...

# Running sync in child worktree uses parent's config
cd ../feature-branch
git volume sync  # uses parent's git-volume.yaml
```

## 🛡️ Safety Features

- **Change Detection on Unsync**: Files copied in copy mode are preserved if modified
- **Idempotent**: Running `sync` multiple times is safe

## 📄 License

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
