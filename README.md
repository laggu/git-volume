> 🌐 [한국어](docs/translations/README_ko.md) | [日本語](docs/translations/README_ja.md) | [中文](docs/translations/README_zh.md) | [Español](docs/translations/README_es.md) | [Português](docs/translations/README_pt.md) | [Français](docs/translations/README_fr.md) | [Deutsch](docs/translations/README_de.md) | [Italiano](docs/translations/README_it.md)

# git-volume

> **"Keep code in Git, mount your environment as volumes."**

`git-volume` is a CLI tool that centrally manages environment files (`.env`, secrets, etc.) across Git worktrees and dynamically mounts them.

## ✨ Key Features

- **Volume Mounting**: Supports symbolic links or file copy modes
- **Configuration Inheritance**: Child worktrees automatically inherit parent settings
- **Safe Cleanup**: Modified or unrelated copied contents are preserved during unsync
- **AI Agent Optimized**: Create worktree + configure environment with a single command

## 📦 Installation

### Homebrew
```bash
brew tap laggu/tap
brew install git-volume
```

Homebrew installation is distributed as a **formula** and builds `git-volume` from source on your machine. This avoids depending on unsigned prebuilt macOS binaries, but it means Homebrew will install Go as a build dependency.

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
git volume status
```

## 📖 Commands

| Command                    | Description                                              |
| -------------------------- | -------------------------------------------------------- |
| `git volume init`          | Create global directory and sample configuration file    |
| `git volume sync`          | Mount volumes to current worktree based on configuration |
| `git volume unsync`        | Remove mounted volumes (modified or unrelated contents are preserved) |
| `git volume status`        | Display current volume status                            |
| `git volume global add`    | Copy files to global storage (`~/.git-volume`)           |
| `git volume global list`   | List files in global storage (tree view)                 |
| `git volume global edit`   | Edit a file in global storage with `$EDITOR`             |
| `git volume global remove` | Remove files from global storage (alias: `rm`)           |
| `git volume version`       | Print version information                                |

## ⚙️ Configuration File (`git-volume.yaml`)

```yaml
volumes:
  # Simple format (default: symbolic link)
  - ".env.shared:.env"

  # With options
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (default) or copy

  # Mount from global storage (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # Directory mount (overlay-copies source entries into target directory)
  - mount: "configs:app/configs"
    mode: "copy"
```

### Mode Comparison

| Mode   | Description          | Use Case                                             |
| ------ | -------------------- | ---------------------------------------------------- |
| `link` | Create symbolic link | Local development (changes reflect immediately)      |
| `copy` | Copy file or overlay-copy directory | Docker builds (environments without symlink support) |

### Copy-mode directory behavior

When `mode: "copy"` uses a directory source, `sync` overlays the source entries into the target directory instead of deleting the target root directory. Existing unrelated files are preserved, file/symlink conflicts are replaced, and file-vs-directory conflicts fail safely. `status` and `unsync` operate on the copied source subset rather than requiring the whole target directory to match exactly.

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

## 🌐 Global Storage

Global storage (`~/.git-volume`) allows sharing files across multiple projects using the `@global/` prefix.

```bash
# Add files to global storage
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# List, edit, and remove
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

Use `@global/` in your configuration to reference these files:

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 CLI Options

| Flag              | Commands         | Description                                    |
| ----------------- | ---------------- | ---------------------------------------------- |
| `--dry-run`       | `sync`, `unsync` | Show what would be done without making changes |
| `--relative`      | `sync`           | Create relative symlinks instead of absolute   |
| `--verbose`, `-v` | All              | Verbosity level: 0=errors only, 1=normal (default), 2=detailed |
| `--config`, `-c`  | All              | Custom config file path                        |

## 🛡️ Safety Features

- **Symlink Source Rejection**: `sync` and `global add` reject symlink sources for security
- **Path Traversal Prevention**: All paths are validated to prevent directory escape attacks
- **Overlay Copy Safety**: Directory copy mode preserves unrelated target files, replaces file/symlink conflicts, and fails on file-vs-directory conflicts
- **Change Detection on Unsync**: `unsync` removes only the copied source subset and preserves modified or unrelated contents
- **Change Detection on Status**: `status` reports `MODIFIED` when copied source entries differ from target
- **Idempotent Sync**: Running `sync` multiple times always produces the same result

## 📄 License

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
