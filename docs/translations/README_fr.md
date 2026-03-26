> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **« Le code dans Git, l'environnement en volumes. »**

`git-volume` est un outil CLI qui gère centralement les fichiers d'environnement (`.env`, secrets, etc.) entre les arbres de travail Git et les monte dynamiquement.

## ✨ Fonctionnalités principales

- **Montage de volumes** : Support des liens symboliques ou de la copie de fichiers
- **Héritage de configuration** : Les arbres de travail enfants héritent automatiquement de la configuration parent
- **Nettoyage sécurisé** : Le contenu modifié par l'utilisateur ou non lié n'est pas supprimé pendant `unsync`
- **Optimisé pour les agents IA** : Créer un arbre de travail + configurer l'environnement en une seule commande

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

**4. Vérifier le statut**
```bash
git volume status
```

## 📖 Commandes

| Commande                   | Description                                                         |
| -------------------------- | ------------------------------------------------------------------- |
| `git volume init`          | Créer le répertoire global et le fichier de configuration exemple   |
| `git volume sync`          | Monter les volumes dans l'arbre de travail actuel                   |
| `git volume unsync`        | Supprimer les volumes montés (le contenu modifié ou non lié est préservé) |
| `git volume status`        | Afficher le statut actuel des volumes                               |
| `git volume global add`    | Copier des fichiers dans le stockage global (`~/.git-volume`)       |
| `git volume global list`   | Lister les fichiers du stockage global (vue arborescente)           |
| `git volume global edit`   | Éditer un fichier du stockage global avec `$EDITOR`                 |
| `git volume global remove` | Supprimer des fichiers du stockage global (alias: `rm`)             |
| `git volume version`       | Afficher les informations de version                                |

## ⚙️ Fichier de configuration (`git-volume.yaml`)

```yaml
volumes:
  # Format simple (par défaut : lien symbolique)
  - ".env.shared:.env"

  # Avec options
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (par défaut) ou copy

  # Monter depuis le stockage global (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # Montage de répertoires (copie en overlay uniquement les entrées source dans le répertoire cible)
  - mount: "configs:app/configs"
    mode: "copy"
```

### Comparaison des modes

| Mode   | Description              | Cas d'utilisation                                       |
| ------ | ------------------------ | ------------------------------------------------------- |
| `link` | Créer un lien symbolique | Développement local (les modifications sont immédiates) |
| `copy` | Copier le fichier / copier un répertoire en overlay | Builds Docker (environnements sans support symlink)     |

### Comportement des répertoires en mode copy

Quand `mode: "copy"` utilise un répertoire source, `sync` superpose les entrées source dans le répertoire cible sans supprimer le répertoire racine cible. Les fichiers existants non liés sont préservés, les conflits fichier/lien symbolique sont remplacés, et les conflits fichier-vs-répertoire échouent de manière sûre. `status` et `unsync` fonctionnent aussi sur le sous-ensemble copié depuis la source au lieu d'exiger que tout le répertoire cible corresponde exactement.

## 🔄 Héritage des arbres de travail

Si un arbre de travail enfant n'a pas de `git-volume.yaml`, la configuration de l'arbre de travail parent (principal) est automatiquement utilisée.

```bash
# La configuration n'existe que dans l'arbre de travail principal
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # fichier de configuration
├── .env.shared       # fichier source
└── ...

# Exécuter sync dans l'arbre de travail enfant utilise la config du parent
cd ../feature-branch
git volume sync  # utilise le git-volume.yaml du parent
```

## 🌐 Stockage global

Le stockage global (`~/.git-volume`) permet de partager des fichiers entre plusieurs projets en utilisant le préfixe `@global/`.

```bash
# Ajouter des fichiers au stockage global
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# Lister, éditer et supprimer
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

Utilisez `@global/` dans votre configuration pour référencer ces fichiers :

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 Options CLI

| Drapeau           | Commandes        | Description                                               |
| ----------------- | ---------------- | --------------------------------------------------------- |
| `--dry-run`       | `sync`, `unsync` | Afficher ce qui serait fait sans effectuer de changements |
| `--relative`      | `sync`           | Créer des liens symboliques relatifs au lieu d'absolus    |
| `--verbose`, `-v` | Toutes           | Niveau de sortie : 0=erreurs seulement, 1=normal (par défaut), 2=détaillé |
| `--config`, `-c`  | Toutes           | Chemin personnalisé du fichier de configuration           |

## 🛡️ Fonctionnalités de sécurité

- **Rejet des sources symboliques** : `sync` et `global add` rejettent les sources qui sont des liens symboliques pour la sécurité
- **Prévention de traversée de chemin** : Tous les chemins sont validés pour prévenir les attaques d'échappement de répertoire
- **Sécurité de la copie overlay** : Le mode copy des répertoires préserve les fichiers cible non liés, remplace seulement les conflits fichier/lien symbolique et échoue sur les conflits fichier-vs-répertoire
- **Détection de modifications à l'Unsync** : `unsync` supprime uniquement le sous-ensemble copié depuis la source et préserve le contenu modifié ou non lié
- **Détection de modifications au Status** : `status` affiche `MODIFIED` quand les entrées copiées depuis la source diffèrent de la cible
- **Sync idempotent** : Exécuter `sync` plusieurs fois produit toujours le même résultat

## 📄 Licence

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
