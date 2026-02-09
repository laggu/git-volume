> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"Gardez le code dans Git, montez votre environnement comme des volumes."**

`git-volume` est un outil CLI qui gère de manière centralisée les fichiers d'environnement (`.env`, secrets, etc.) à travers les worktrees Git et les monte dynamiquement.

## ✨ Caractéristiques principales

- **Montage de volumes** : Prend en charge les modes liens symboliques ou copie de fichiers
- **Héritage de configuration** : Les worktrees enfants héritent automatiquement des paramètres parents
- **Nettoyage sécurisé** : Les fichiers modifiés sont préservés lors de la désynchronisation (unsync)
- **Optimisé pour les agents IA** : Créez un worktree + configurez l'environnement avec une seule commande

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

## 🚀 Démarrage rapide

**1. Initialisation**
```bash
git volume init
```

**2. Créer git-volume.yaml**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. Monter les volumes**
```bash
git volume sync
```

**4. Vérifier l'état**
```bash
git volume status
```

## 📖 Commandes

| Commande            | Description                                                        |
| ------------------- | ------------------------------------------------------------------ |
| `git volume init`   | Crée le répertoire global et un fichier de configuration d'exemple |
| `git volume sync`   | Monte les volumes dans le worktree actuel selon la configuration   |
| `git volume unsync` | Supprime les volumes montés (les fichiers modifiés sont préservés) |
| `git volume status` | Affiche l'état actuel des volumes                                  |

## ⚙️ Fichier de configuration (`git-volume.yaml`)

```yaml
volumes:
  # Format simple (par défaut : lien symbolique)
  - ".env.shared:.env"

  # Avec options
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (défaut) ou copy
    force: true    # écraser si existe
```

### Comparaison des modes

| Mode   | Description             | Cas d'utilisation                                    |
| ------ | ----------------------- | ---------------------------------------------------- |
| `link` | Crée un lien symbolique | Développement local (les changements sont immédiats) |
| `copy` | Copie le fichier        | Builds Docker (environnements sans support symlink)  |

## 🔄 Héritage de Worktree

Si un worktree enfant n'a pas de `git-volume.yaml`, il utilise automatiquement la configuration du worktree parent (principal).

```bash
# La configuration n'existe que dans le worktree principal
main-repo/
├── .git/             # répertoire commun git
├── git-volume.yaml   # fichier de configuration
├── .env.shared       # fichier source
└── ...

# L'exécution de sync dans un worktree enfant utilise la config parente
cd ../feature-branch
git volume sync  # utilise le git-volume.yaml du parent
```

## 🛡️ Fonctionnalités de sécurité

- **Détection de changement lors de l'unsync** : Les fichiers copiés en mode `copy` sont préservés s'ils ont été modifiés
- **Idempotent** : L'exécution de `sync` plusieurs fois est sûre

## 📄 Licence

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
