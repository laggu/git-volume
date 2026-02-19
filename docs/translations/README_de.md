> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Italiano](README_it.md)

# git-volume

> **„Code in Git, Umgebung als Volumes."**

`git-volume` ist ein CLI-Tool, das Umgebungsdateien (`.env`, Secrets usw.) zentral über Git-Worktrees hinweg verwaltet und dynamisch einbindet.

## ✨ Hauptfunktionen

- **Volume-Mounting**: Unterstützung für symbolische Links oder Dateikopien
- **Konfigurationsvererbung**: Kind-Worktrees erben automatisch die Eltern-Konfiguration
- **Sichere Bereinigung**: Vom Benutzer geänderte Dateien werden nicht gelöscht
- **KI-Agenten-optimiert**: Worktree erstellen + Umgebung konfigurieren mit einem einzigen Befehl

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

## 🚀 Schnellstart

**1. Initialisierung**
```bash
git volume init
```

**2. git-volume.yaml erstellen**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. Volumes einbinden**
```bash
git volume sync
```

**4. Status prüfen**
```bash
git volume status
```

## 📖 Befehle

| Befehl                     | Beschreibung                                                        |
| -------------------------- | ------------------------------------------------------------------- |
| `git volume init`          | Globales Verzeichnis und Beispielkonfiguration erstellen            |
| `git volume sync`          | Volumes basierend auf Konfiguration im aktuellen Worktree einbinden |
| `git volume unsync`        | Eingebundene Volumes entfernen (geänderte Dateien bleiben erhalten) |
| `git volume status`        | Aktuellen Volume-Status anzeigen                                    |
| `git volume global add`    | Dateien in den globalen Speicher kopieren (`~/.git-volume`)         |
| `git volume global list`   | Dateien im globalen Speicher auflisten (Baumansicht)                |
| `git volume global edit`   | Eine Datei im globalen Speicher mit `$EDITOR` bearbeiten            |
| `git volume global remove` | Dateien aus dem globalen Speicher entfernen (alias: `rm`)           |
| `git volume version`       | Versionsinformationen anzeigen                                      |

## ⚙️ Konfigurationsdatei (`git-volume.yaml`)

```yaml
volumes:
  # Einfaches Format (Standard: symbolischer Link)
  - ".env.shared:.env"

  # Mit Optionen
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (Standard) oder copy

  # Aus globalem Speicher einbinden (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # Verzeichnis-Mounting (kopiert gesamtes Verzeichnis)
  - mount: "configs:app/configs"
    mode: "copy"
```

### Modusvergleich

| Modus  | Beschreibung                | Anwendungsfall                                  |
| ------ | --------------------------- | ----------------------------------------------- |
| `link` | Symbolischen Link erstellen | Lokale Entwicklung (Änderungen sofort wirksam)  |
| `copy` | Datei kopieren              | Docker-Builds (Umgebungen ohne Symlink-Support) |

## 🔄 Worktree-Vererbung

Wenn ein Kind-Worktree keine `git-volume.yaml` hat, wird automatisch die Konfiguration des Eltern-Worktrees (Haupt-Worktree) verwendet.

```bash
# Konfiguration existiert nur im Haupt-Worktree
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # Konfigurationsdatei
├── .env.shared       # Quelldatei
└── ...

# Sync im Kind-Worktree verwendet die Eltern-Konfiguration
cd ../feature-branch
git volume sync  # verwendet git-volume.yaml des Eltern-Worktrees
```

## 🌐 Globaler Speicher

Der globale Speicher (`~/.git-volume`) ermöglicht das Teilen von Dateien über mehrere Projekte hinweg mit dem `@global/`-Präfix.

```bash
# Dateien zum globalen Speicher hinzufügen
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# Auflisten, bearbeiten und entfernen
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

Verwenden Sie `@global/` in Ihrer Konfiguration, um auf diese Dateien zu verweisen:

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 CLI-Optionen

| Flag              | Befehle          | Beschreibung                                        |
| ----------------- | ---------------- | --------------------------------------------------- |
| `--dry-run`       | `sync`, `unsync` | Zeigen was getan würde, ohne Änderungen vorzunehmen |
| `--relative`      | `sync`           | Relative statt absolute symbolische Links erstellen |
| `--verbose`, `-v` | Alle             | Ausführliche Ausgabe                                |
| `--quiet`, `-q`   | Alle             | Nicht-Fehler-Ausgaben unterdrücken                  |
| `--config`, `-c`  | Alle             | Benutzerdefinierter Konfigurationsdateipfad         |

## 🛡️ Sicherheitsfunktionen

- **Ablehnung symbolischer Quellen**: `sync` und `global add` lehnen Quellen ab, die symbolische Links sind
- **Pfad-Traversal-Schutz**: Alle Pfade werden validiert, um Verzeichnis-Escape-Angriffe zu verhindern
- **Änderungserkennung bei Unsync**: Kopierte Dateien und Verzeichnisse werden beibehalten, wenn sie geändert wurden
- **Änderungserkennung bei Status**: Kopierte Dateien, die vom Original abweichen, werden als `MODIFIED` angezeigt
- **Idempotentes Sync**: Mehrfaches Ausführen von `sync` erzeugt immer das gleiche Ergebnis

## 📄 Lizenz

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
