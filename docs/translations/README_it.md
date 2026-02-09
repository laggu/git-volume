> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"Mantieni il codice in Git, monta il tuo ambiente come volumi."**

`git-volume` è uno strumento CLI che gestisce centralmente i file di ambiente (`.env`, segreti, ecc.) tra i worktree di Git e li monta dinamicamente.

## ✨ Caratteristiche principali

- **Montaggio dei volumi**: Supporta modalità link simbolico o copia file
- **Ereditarietà della configurazione**: I worktree figli ereditano automaticamente le impostazioni del genitore
- **Pulizia sicura**: I file modificati vengono preservati durante l'unsync
- **Ottimizzato per agenti AI**: Crea un worktree + configura l'ambiente con un singolo comando

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

## 🚀 Guida rapida

**1. Inizializzazione**
```bash
git volume init
```

**2. Creazione di git-volume.yaml**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. Montaggio dei volumi**
```bash
git volume sync
```

**4. Verifica dello stato**
```bash
git volume status
```

## 📖 Comandi

| Comando             | Descrizione                                                      |
| ------------------- | ---------------------------------------------------------------- |
| `git volume init`   | Crea la directory globale e un file di configurazione di esempio |
| `git volume sync`   | Monta i volumi nel worktree attuale in base alla configurazione  |
| `git volume unsync` | Rimuove i volumi montati (i file modificati vengono preservati)  |
| `git volume status` | Visualizza lo stato attuale dei volumi                           |

## ⚙️ File di configurazione (`git-volume.yaml`)

```yaml
volumes:
  # Formato semplice (predefinito: link simbolico)
  - ".env.shared:.env"

  # Con opzioni
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (predefinito) o copy
    force: true    # sovrascrivi se esistente
```

### Confronto modalità

| Modalità | Descrizione            | Caso d'uso                                     |
| -------- | ---------------------- | ---------------------------------------------- |
| `link`   | Crea un link simbolico | Sviluppo locale (le modifiche sono immediate)  |
| `copy`   | Copia il file          | Build Docker (ambienti senza supporto symlink) |

## 🔄 Ereditarietà Worktree

Se un worktree figlio non ha `git-volume.yaml`, utilizza automaticamente la configurazione del worktree genitore (principale).

```bash
# La configurazione esiste solo nel worktree principale
main-repo/
├── .git/             # directory comune git
├── git-volume.yaml   # file di configurazione
├── .env.shared       # file sorgente
└── ...

# L'esecuzione di sync in un worktree figlio usa la config del genitore
cd ../feature-branch
git volume sync  # usa il git-volume.yaml del genitore
```

## 🛡️ Funzionalità di sicurezza

- **Rilevamento modifiche all'unsync**: I file copiati in modalità `copy` vengono preservati se modificati
- **Idempotente**: L'esecuzione di `sync` più volte è sicura

## 📄 Licenza

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
