# git-volume Command Specification

`git-volume` is a tool for centrally managing and dynamically mounting environment configuration files (like `.env` or `secrets`) in Git Worktree environments.

## 1. init

### Purpose
Initializes `git-volume` in the current directory.
- Creates the global configuration directory (`~/.git-volume`) if it does not exist.
- Generates a sample `git-volume.yaml` configuration file in the current directory.

### Usage
```bash
git volume init [flags]
```

### Flags
- `-h, --help`: help for init
- `-v, --verbose int`: verbosity level (0=errors only, 1=normal default, 2=detailed)

### Example
```bash
# Basic initialization
git volume init

# Errors only output
git volume init --verbose 0
```

---

## 2. sync

### Purpose
Mounts volumes to the current Worktree according to the `git-volume.yaml` configuration.
- Searches for settings in the current directory; if not found, traverses parent directories (towards the Git Common Dir) to inherit settings.
- Mounts files/directories according to the configured mode (`link` or `copy`).
- In `copy` mode with a directory source, overlays the source entries into the target directory instead of deleting the target root directory. Unrelated target files are preserved, file/symlink conflicts are replaced, and file-vs-directory conflicts fail safely.
- Rejects symlink sources for security.

### Usage
```bash
git volume sync [flags]
```

### Flags
- `--dry-run`: show what would be done without making actual changes
- `--relative`: create relative symbolic links instead of absolute ones
- `-v, --verbose int`: verbosity level (0=errors only, 1=normal default, 2=detailed)
- `-c, --config string`: manually specify config file path (default: auto-detected)

### Example
```bash
# Basic sync
git volume sync

# Preview changes
git volume sync --dry-run
```

---

## 3. unsync

### Purpose
Removes symbolic links or copied files created by the `sync` command.
- **Safety Mechanism**: If a file mounted in `copy` mode has been modified, deletion is skipped to prevent data loss.
- For directory sources in `copy` mode, removes only the copied source subset. Unrelated files already present in the target directory are preserved.

### Usage
```bash
git volume unsync [flags]
```

### Flags
- `--dry-run`: show what would be removed without actually deleting
- `-v, --verbose int`: verbosity level (0=errors only, 1=normal default, 2=detailed)

### Example
```bash
# Unmount
git volume unsync

# Check files to be deleted
git volume unsync --dry-run
```

---

## 4. status

### Purpose
Displays a list of the status of currently configured volumes.
- Shows Source, Target, Mode (Link/Copy), and Status.
- Common status values include `OK (Linked)`, `OK (Copied)`, `MODIFIED`, `NOT MOUNTED`, and `MISSING (Source)`.
- For directory sources in `copy` mode, `OK (Copied)` means every copied source entry matches in the target directory; extra unrelated target files do not affect status.

### Usage
```bash
git volume status [flags]
```

### Flags
- `-c, --config string`: specify configuration file path
- `-v, --verbose int`: verbosity level (0=errors only, 1=normal default, 2=detailed)

### Example
```bash
# Check status
git volume status

# Check status with verbose output info
git volume status -v
```

---

## 5. global

A group of commands for managing files in the global directory (`~/.git-volume`).

### 5.1 global add

#### Purpose
Copies and registers the specified file to the global storage.
- Rejects symlink sources. For directory sources, rejects if any nested entry is a symlink.

#### Usage
```bash
git volume global add [source_file]
```

#### Example
```bash
# Register .env file to global storage
git volume global add .env
```

### 5.2 global remove

#### Purpose
Deletes a file registered in the global storage.

#### Usage
```bash
git volume global remove [target_name]
```

#### Example
```bash
# Delete dev.env from global storage
git volume global remove dev.env
```

### 5.3 global list

#### Purpose
Shows a list of all files registered in the global storage.

#### Usage
```bash
git volume global list
```

### 5.4 global edit

#### Purpose
Opens a file in global storage with the default editor (`$EDITOR`).

#### Usage
```bash
git volume global edit [target_name]
```

#### Example
```bash
# Edit dev.env in global storage
git volume global edit dev.env
```

---

## Global Flags
Flags available for all commands.

- `-c, --config`: specify config file path
- `-v, --verbose int`: verbosity level (`0=errors only`, `1=normal`, `2=detailed`)
