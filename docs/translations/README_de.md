> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"Behalte den Code in Git, mounte deine Umgebung als Volumes."**

`git-volume` ist ein CLI-Tool zur zentralen Verwaltung von Umgebungsdateien (`.env`, Secrets usw.) über Git-Worktrees hinweg und deren dynamischem Mounten.

## ✨ Hauptmerkmale

- **Volume-Mounting**: Unterstützt symbolische Links oder Dateikopiermodi
- **Konfigurationsvererbung**: Untergeordnete Worktrees erben automatisch die Einstellungen der Eltern
- **Sicheres Aufräumen**: Geänderte Dateien bleiben beim Unsync erhalten
- **Optimiert für KI-Agenten**: Worktree erstellen + Umgebung konfigurieren mit einem einzigen Befehl

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

**1. Initialisieren**
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

**3. Volumes mounten**
```bash
git volume sync
```

**4. Status prüfen**
```bash
git volume status
```

## 📖 Befehle

| Befehl              | Beschreibung                                                          |
| ------------------- | --------------------------------------------------------------------- |
| `git volume init`   | Erstellt das globale Verzeichnis und eine Beispielkonfigurationsdatei |
| `git volume sync`   | Mountet Volumes im aktuellen Worktree basierend auf der Konfiguration |
| `git volume unsync` | Entfernt gemountete Volumes (geänderte Dateien bleiben erhalten)      |
| `git volume status` | Zeigt den aktuellen Volume-Status an                                  |

## ⚙️ Konfigurationsdatei (`git-volume.yaml`)

```yaml
volumes:
  # Einfaches Format (Standard: symbolischer Link)
  - ".env.shared:.env"

  # Mit Optionen
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (Standard) oder copy
    force: true    # überschreiben, falls vorhanden
```

### Modus-Vergleich

| Modus  | Beschreibung               | Anwendungsfall                                        |
| ------ | -------------------------- | ----------------------------------------------------- |
| `link` | Erstellt symbolischen Link | Lokale Entwicklung (Änderungen werden sofort wirksam) |
| `copy` | Kopiert Datei              | Docker-Builds (Umgebungen ohne Symlink-Unterstützung) |

## 🔄 Worktree-Vererbung

Wenn ein untergeordneter Worktree keine `git-volume.yaml` hat, verwendet er automatisch die Konfiguration des übergeordneten (Haupt-)Worktrees.

```bash
# Konfiguration existiert nur im Haupt-Worktree
main-repo/
├── .git/             # gemeinsames Git-Verzeichnis
├── git-volume.yaml   # Konfigurationsdatei
├── .env.shared       # Quelldatei
└── ...

# Das Ausführen von sync in einem untergeordneten Worktree verwendet die Eltern-Konfig
cd ../feature-branch
git volume sync  # verwendet die git-volume.yaml des Eltern-Worktrees
```

## 🛡️ Sicherheitsmerkmale

- **Änderungserkennung beim Unsync**: Im Copy-Modus kopierte Dateien bleiben erhalten, wenn sie geändert wurden
- **Idempotent**: Das mehrmalige Ausführen von `sync` ist sicher

## 📄 Lizenz

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
