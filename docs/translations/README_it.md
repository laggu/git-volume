> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md)

# git-volume

> **"Il codice in Git, l'ambiente come volumi."**

`git-volume` è uno strumento CLI che gestisce centralmente i file di ambiente (`.env`, segreti, ecc.) tra i worktree Git e li monta dinamicamente.

## ✨ Funzionalità principali

- **Montaggio volumi**: Supporto per link simbolici o copia di file
- **Ereditarietà della configurazione**: I worktree figli ereditano automaticamente la configurazione del genitore
- **Pulizia sicura**: I file modificati dall'utente non vengono eliminati
- **Ottimizzato per agenti IA**: Creare worktree + configurare l'ambiente con un singolo comando

## 📦 Installazione

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

## 🚀 Avvio rapido

**1. Inizializzazione**
```bash
git volume init
```

**2. Creare git-volume.yaml**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. Montare i volumi**
```bash
git volume sync
```

**4. Verificare lo stato**
```bash
git volume status
```

## 📖 Comandi

| Comando                    | Descrizione                                                          |
| -------------------------- | -------------------------------------------------------------------- |
| `git volume init`          | Creare la directory globale e il file di configurazione esempio      |
| `git volume sync`          | Montare i volumi nel worktree attuale basandosi sulla configurazione |
| `git volume unsync`        | Rimuovere i volumi montati (i file modificati vengono preservati)    |
| `git volume status`        | Mostrare lo stato attuale dei volumi                                 |
| `git volume global add`    | Copiare file nell'archivio globale (`~/.git-volume`)                 |
| `git volume global list`   | Elencare i file nell'archivio globale (vista ad albero)              |
| `git volume global edit`   | Modificare un file dell'archivio globale con `$EDITOR`               |
| `git volume global remove` | Rimuovere file dall'archivio globale (alias: `rm`)                   |
| `git volume version`       | Mostrare le informazioni sulla versione                              |

## ⚙️ File di configurazione (`git-volume.yaml`)

```yaml
volumes:
  # Formato semplice (predefinito: link simbolico)
  - ".env.shared:.env"

  # Con opzioni
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (predefinito) o copy

  # Montare dall'archivio globale (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # Montaggio directory (copia l'intera directory)
  - mount: "configs:app/configs"
    mode: "copy"
```

### Confronto modi

| Modo   | Descrizione           | Caso d'uso                                                  |
| ------ | --------------------- | ----------------------------------------------------------- |
| `link` | Creare link simbolico | Sviluppo locale (le modifiche si riflettono immediatamente) |
| `copy` | Copiare il file       | Build Docker (ambienti senza supporto link simbolici)       |

## 🔄 Ereditarietà dei worktree

Se un worktree figlio non ha un `git-volume.yaml`, viene automaticamente utilizzata la configurazione del worktree genitore (principale).

```bash
# La configurazione esiste solo nel worktree principale
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # file di configurazione
├── .env.shared       # file sorgente
└── ...

# Eseguire sync nel worktree figlio usa la config del genitore
cd ../feature-branch
git volume sync  # usa il git-volume.yaml del genitore
```

## 🌐 Archivio globale

L'archivio globale (`~/.git-volume`) consente di condividere file tra più progetti usando il prefisso `@global/`.

```bash
# Aggiungere file all'archivio globale
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# Elencare, modificare e rimuovere
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

Usa `@global/` nella configurazione per fare riferimento a questi file:

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 Opzioni CLI

| Flag              | Comandi          | Descrizione                                            |
| ----------------- | ---------------- | ------------------------------------------------------ |
| `--dry-run`       | `sync`, `unsync` | Mostrare cosa verrebbe fatto senza apportare modifiche |
| `--relative`      | `sync`           | Creare link simbolici relativi invece di assoluti      |
| `--verbose`, `-v` | Tutti            | Output dettagliato                                     |
| `--quiet`, `-q`   | Tutti            | Nascondere l'output non correlato agli errori          |
| `--config`, `-c`  | Tutti            | Percorso personalizzato del file di configurazione     |

## 🛡️ Funzionalità di sicurezza

- **Rifiuto sorgenti simboliche**: `sync` e `global add` rifiutano sorgenti che sono link simbolici per sicurezza
- **Prevenzione path traversal**: Tutti i percorsi vengono validati per prevenire attacchi di escape dalla directory
- **Rilevamento modifiche all'Unsync**: I file e le directory copiate vengono preservati se modificati
- **Rilevamento modifiche allo Status**: I file copiati diversi dall'originale vengono mostrati come `MODIFIED`
- **Sync idempotente**: Eseguire `sync` più volte produce sempre lo stesso risultato

## 📄 Licenza

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
