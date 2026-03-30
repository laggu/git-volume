> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"代码留在 Git，环境作为卷挂载。"**

`git-volume` 是一个 CLI 工具，用于在 Git 工作树之间集中管理环境文件（`.env`、密钥等）并动态挂载。

## ✨ 主要功能

- **卷挂载**: 支持符号链接或文件复制模式
- **配置继承**: 子工作树自动继承父工作树的配置
- **安全清理**: 用户修改过的内容或无关的复制内容在 unsync 时不会被删除
- **AI 代理优化**: 一条命令即可创建工作树并配置环境

## 📦 安装

### Homebrew
```bash
brew tap laggu/tap
brew install git-volume
```

Homebrew 安装以 **formula** 形式发布，会在用户机器上从源码构建 `git-volume`。这样可以避免依赖未签名的 macOS 预编译二进制文件，但 Homebrew 会安装 Go 作为构建依赖。

### Scoop (Windows)
```bash
scoop bucket add laggu https://github.com/laggu/scoop-bucket.git
scoop install git-volume
```

### Go
```bash
go install github.com/laggu/git-volume@latest
```

## 🚀 快速开始

**1. 初始化**
```bash
git volume init
```

**2. 创建 git-volume.yaml**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. 挂载卷**
```bash
git volume sync
```

**4. 检查状态**
```bash
git volume status
```

## 📖 命令

| 命令                       | 说明                                  |
| -------------------------- | ------------------------------------- |
| `git volume init`          | 创建全局目录和示例配置文件            |
| `git volume sync`          | 根据配置将卷挂载到当前工作树          |
| `git volume unsync`        | 移除已挂载的卷（已修改或无关的内容会保留） |
| `git volume status`        | 显示当前卷的状态                      |
| `git volume global add`    | 将文件复制到全局存储(`~/.git-volume`) |
| `git volume global list`   | 列出全局存储中的文件（树形视图）      |
| `git volume global edit`   | 使用 `$EDITOR` 编辑全局存储中的文件   |
| `git volume global remove` | 从全局存储中删除文件 (alias: `rm`)    |
| `git volume version`       | 打印版本信息                          |

## ⚙️ 配置文件 (`git-volume.yaml`)

```yaml
volumes:
  # 简单格式（默认：符号链接）
  - ".env.shared:.env"

  # 指定选项
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link（默认）或 copy

  # 从全局存储挂载 (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # 目录挂载（仅将 source 条目 overlay 复制到 target 目录）
  - mount: "configs:app/configs"
    mode: "copy"
```

### 模式比较

| 模式   | 说明         | 用途                                |
| ------ | ------------ | ----------------------------------- |
| `link` | 创建符号链接 | 本地开发（修改立即反映）            |
| `copy` | 复制文件 / overlay 复制目录 | Docker 构建（不支持符号链接的环境） |

### Copy 模式目录行为

当 `mode: "copy"` 的 source 是目录时，`sync` 不会删除 target 根目录，而是只把 source 条目 overlay 到 target 目录中。无关的现有文件会保留，文件/符号链接冲突会被替换，文件与目录冲突会安全失败。`status` 和 `unsync` 也只按复制的 source 子集处理，而不是要求整个目标目录完全一致。

## 🔄 工作树继承

如果子工作树中没有 `git-volume.yaml`，将自动使用父（主）工作树的配置。

```bash
# 仅在主工作树中存在配置
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # 配置文件
├── .env.shared       # 源文件
└── ...

# 在子工作树中运行 sync 时使用父级配置
cd ../feature-branch
git volume sync  # 使用父级的 git-volume.yaml
```

## 🌐 全局存储

全局存储(`~/.git-volume`)允许通过 `@global/` 前缀在多个项目之间共享文件。

```bash
# 向全局存储添加文件
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# 列出、编辑和删除
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

在配置文件中使用 `@global/` 引用这些文件：

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 CLI 选项

| 标志              | 适用命令         | 说明                             |
| ----------------- | ---------------- | -------------------------------- |
| `--dry-run`       | `sync`, `unsync` | 不做实际更改，仅显示将执行的操作 |
| `--relative`      | `sync`           | 创建相对路径符号链接而非绝对路径 |
| `--verbose`, `-v` | 全部             | 输出级别：0=仅错误，1=普通（默认），2=详细 |
| `--config`, `-c`  | 全部             | 指定配置文件路径                 |

## 🛡️ 安全功能

- **符号链接源拒绝**: `sync` 和 `global add` 出于安全考虑拒绝符号链接源
- **路径遍历防护**: 所有路径都经过验证以防止目录逃逸攻击
- **Overlay 复制安全性**: 目录 copy 模式会保留无关的 target 文件，仅替换文件/符号链接冲突，并在文件与目录冲突时安全失败
- **Unsync 变更检测**: `unsync` 只移除复制的 source 子集，并保留已修改或无关的内容
- **Status 变更检测**: 当复制的 source 条目与 target 不一致时显示 `MODIFIED`
- **幂等性保证**: 多次运行 `sync` 始终产生相同结果

## 📄 许可证

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
