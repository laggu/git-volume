> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"代码归 Git，环境归卷。"**

`git-volume` 是一个 CLI 工具，用于在 Git 工作区（worktrees）之间集中管理环境配置文件（如 `.env`、机密信息等）并动态挂载它们。

## ✨ 主要功能

- **卷挂载**：支持符号链接或文件复制模式
- **配置继承**：子工作区自动继承父工作区的设置
- **安全清理**：在取消同步（unsync）时保护已修改的文件
- **AI 代理优化**：通过一条命令即可创建工作区并配置环境

## 📦 安装

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

## 🚀 快速入门

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

**4. 查看状态**
```bash
git volume status
```

## 📖 常用命令

| 命令                | 描述                               |
| ------------------- | ---------------------------------- |
| `git volume init`   | 创建全局目录和示例配置文件         |
| `git volume sync`   | 根据配置将卷挂载到当前工作区       |
| `git volume unsync` | 移除已挂载的卷（保留已修改的文件） |
| `git volume status` | 显示当前卷状态                     |

## ⚙️ 配置文件 (`git-volume.yaml`)

```yaml
volumes:
  # 简单格式（默认：符号链接）
  - ".env.shared:.env"

  # 带有选项的格式
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (默认) 或 copy
    force: true    # 如果已存在则覆盖
```

### 模式对比

| 模式   | 描述         | 使用场景                            |
| ------ | ------------ | ----------------------------------- |
| `link` | 创建符号链接 | 本地开发（更改立即生效）            |
| `copy` | 复制文件     | Docker 构建（不支持符号链接的环境） |

## 🔄 工作区继承

如果子工作区没有 `git-volume.yaml`，它会自动使用父（主）工作区的配置。

```bash
# 仅在主工作区存在配置
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # 配置文件
├── .env.shared       # 源文件
└── ...

# 在子工作区运行 sync 将使用父配置
cd ../feature-branch
git volume sync  # 使用父工作区的 git-volume.yaml
```

## 🛡️ 安全特性

- **取消同步时的更改检测**：以 copy 模式复制的文件如果已被修改，则会被保留
- **幂等性**：多次运行 `sync` 是安全的

## 📄 许可证

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
