> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"コードはGitに、環境はボリュームとしてマウント。"**

`git-volume`は、複数のGitワークツリー間で環境設定ファイル（`.env`、シークレットなど）を一元管理し、動的にマウントするためのCLIツールです。

## ✨ 主な機能

- **ボリュームマウント**: シンボリックリンクまたはファイルコピーモードをサポート
- **設定の継承**: 子ワークツリーは親ツリーの設定を自動的に継承
- **安全なクリーンアップ**: 変更されたファイルはアンシンク時に保護
- **AIエージェント最適化**: ワークツリー作成と環境構築を1つのコマンドで実行

## 📦 インストール

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

## 🚀 クイックスタート

**1. 初期化**
```bash
git volume init
```

**2. git-volume.yaml の作成**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. ボリュームのマウント**
```bash
git volume sync
```

**4. ステータスの確認**
```bash
git volume status
```

## 📖 コマンド

| コマンド            | 説明                                       |
| ------------------- | ------------------------------------------ |
| `git volume init`   | グローバルディレクトリとサンプル設定の作成 |
| `git volume sync`   | 設定に基づきボリュームをマウント           |
| `git volume unsync` | マウントされたボリュームを削除             |
| `git volume status` | 現在のボリュームステータスを表示           |

## ⚙️ 設定ファイル (`git-volume.yaml`)

```yaml
volumes:
  # シンプルな形式（デフォルト：シンボリックリンク）
  - ".env.shared:.env"

  # オプション指定
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (デフォルト) または copy
    force: true    # すでに存在する場合に上書き
```

### モード比較

| モード | 説明                   | ユースケース                                     |
| ------ | ---------------------- | ------------------------------------------------ |
| `link` | シンボリックリンク作成 | ローカル開発（変更が即座に反映）                 |
| `copy` | ファイルコピー         | Dockerビルド（シンボリックリンク未サポート環境） |

## 🔄 ワークツリーの継承

子ワークツリーに `git-volume.yaml` がない場合、自動的に親（メイン）ワークツリーの設定を使用します。

```bash
# メインワークツリーのみに設定が存在
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # 設定ファイル
├── .env.shared       # ソースファイル
└── ...

# 子ワークツリーでsyncを実行すると親の設定が使用される
cd ../feature-branch
git volume sync  # 親の git-volume.yaml を使用
```

## 🛡️ 安全機能

- **アンシンク時の変更検知**: Copyモードでコピーされたファイルが変更されている場合、削除せずに保持
- **べき等性**: `sync`を複数回実行しても安全

## 📄 ライセンス

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
