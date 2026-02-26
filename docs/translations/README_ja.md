> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **「コードはGitに、環境はボリュームとして。」**

`git-volume`は、Gitワークツリー間で環境ファイル（`.env`、シークレットなど）を一元管理し、動的にマウントするCLIツールです。

## ✨ 主な機能

- **ボリュームマウント**: シンボリックリンクまたはファイルコピーに対応
- **設定の継承**: 子ワークツリーが親の設定を自動継承
- **安全なクリーンアップ**: ユーザーが変更したファイルは削除しない
- **AIエージェント最適化**: 一つのコマンドでワークツリー作成+環境構成

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

**2. git-volume.yamlの作成**
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

**4. ステータス確認**
```bash
git volume status
```

## 📖 コマンド

| コマンド                   | 説明                                                    |
| -------------------------- | ------------------------------------------------------- |
| `git volume init`          | グローバルディレクトリとサンプル設定ファイルを作成      |
| `git volume sync`          | 設定に基づきボリュームを現在のワークツリーにマウント    |
| `git volume unsync`        | マウントされたボリュームを削除（変更ファイルは保持）    |
| `git volume status`        | 現在のボリュームステータスを表示                        |
| `git volume global add`    | グローバルストレージ(`~/.git-volume`)にファイルをコピー |
| `git volume global list`   | グローバルストレージのファイル一覧（ツリー表示）        |
| `git volume global edit`   | グローバルストレージのファイルを`$EDITOR`で編集         |
| `git volume global remove` | グローバルストレージからファイルを削除 (alias: `rm`)    |
| `git volume version`       | バージョン情報を表示                                    |

## ⚙️ 設定ファイル (`git-volume.yaml`)

```yaml
volumes:
  # シンプル形式（デフォルト: シンボリックリンク）
  - ".env.shared:.env"

  # オプション指定
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link（デフォルト）または copy

  # グローバルストレージからマウント (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # ディレクトリマウント（ディレクトリ全体をコピー）
  - mount: "configs:app/configs"
    mode: "copy"
```

### モード比較

| モード | 説明                   | 用途                                         |
| ------ | ---------------------- | -------------------------------------------- |
| `link` | シンボリックリンク作成 | ローカル開発（変更が即座に反映）             |
| `copy` | ファイルコピー         | Dockerビルド（シンボリックリンク非対応環境） |

## 🔄 ワークツリー継承

子ワークツリーに`git-volume.yaml`がない場合、親（メイン）ワークツリーの設定を自動的に使用します。

```bash
# メインワークツリーにのみ設定が存在
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # 設定ファイル
├── .env.shared       # ソースファイル
└── ...

# 子ワークツリーでsync実行時に親の設定を使用
cd ../feature-branch
git volume sync  # 親のgit-volume.yamlを使用
```

## 🌐 グローバルストレージ

グローバルストレージ(`~/.git-volume`)を使用すると、`@global/`プレフィックスで複数のプロジェクト間でファイルを共有できます。

```bash
# グローバルストレージにファイルを追加
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# 一覧表示、編集、削除
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

設定ファイルで`@global/`を使用して参照します:

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 CLIオプション

| フラグ            | 対象コマンド     | 説明                                     |
| ----------------- | ---------------- | ---------------------------------------- |
| `--dry-run`       | `sync`, `unsync` | 実際の変更なしに実行内容を表示           |
| `--relative`      | `sync`           | 絶対パスの代わりに相対パスのリンクを作成 |
| `--verbose`, `-v` | 全て             | 出力レベル: 0=エラーのみ、1=通常(既定)、2=詳細 |
| `--config`, `-c`  | 全て             | 設定ファイルパスの指定                   |

## 🛡️ 安全機能

- **シンボリックリンクソース拒否**: `sync`と`global add`でシンボリックリンクソースをセキュリティ上拒否
- **パストラバーサル防止**: 全てのパスに対してディレクトリエスケープ攻撃を検証
- **Unsync時の変更検出**: Copyモードでコピーされたファイル・ディレクトリは変更されていれば保持
- **Status変更検出**: コピーされたファイルが元ファイルと異なる場合`MODIFIED`と表示
- **べき等性保証**: `sync`を何度実行しても常に同じ結果

## 📄 ライセンス

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
