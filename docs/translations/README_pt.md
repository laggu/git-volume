> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md) | [Português](README_pt.md)

# git-volume

> **"Mantenha o código no Git, monte o seu ambiente como volumes."**

`git-volume` é uma ferramenta CLI que gere centralizadamente ficheiros de ambiente (`.env`, segredos, etc.) entre os *worktrees* do Git e os monta dinamicamente.

## ✨ Principais Características

- **Montagem de Volumes**: Suporta modos de ligação simbólica ou cópia de ficheiros.
- **Herança de Configuração**: Os *worktrees* filhos herdam automaticamente as definições do pai.
- **Limpeza Segura**: Os ficheiros modificados são preservados durante o *unsync*.
- **Otimizado para Agentes de IA**: Crie um *worktree* e configure o ambiente com um único comando.

## 📦 Instalação

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

## 🚀 Início Rápido

**1. Inicializar**
```bash
git volume init
```

**2. Criar git-volume.yaml**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. Montar Volumes**
```bash
git volume sync
```

**4. Verificar Estado**
```bash
git volume status
```

## 📖 Comandos

| Comando             | Descrição                                                         |
| ------------------- | ----------------------------------------------------------------- |
| `git volume init`   | Cria o diretório global e um ficheiro de configuração de exemplo. |
| `git volume sync`   | Monta volumes no worktree atual com base na configuração.         |
| `git volume unsync` | Remove volumes montados (ficheiros modificados são preservados).  |
| `git volume status` | Exibe o estado atual dos volumes.                                 |

## ⚙️ Ficheiro de Configuração (`git-volume.yaml`)

```yaml
volumes:
  # Formato simples (predefinido: ligação simbólica)
  - ".env.shared:.env"

  # Com opções
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (predefinido) ou copy

```

### Comparação de Modos

| Modo   | Descrição              | Caso de Uso                                                        |
| ------ | ---------------------- | ------------------------------------------------------------------ |
| `link` | Cria ligação simbólica | Desenvolvimento local (as alterações refletem-se imediatamente).   |
| `copy` | Copia ficheiro         | Builds de Docker (ambientes sem suporte para ligações simbólicas). |

## 🔄 Herança de Worktree

Se um *worktree* filho não tiver `git-volume.yaml`, ele utiliza automaticamente a configuração do *worktree* pai (principal).

```bash
# A configuração existe apenas no worktree principal
main-repo/
├── .git/             # diretório comum do git
├── git-volume.yaml   # ficheiro de configuração
├── .env.shared       # ficheiro de origem
└── ...

# Executar sync num worktree filho usa a configuração do pai
cd ../feature-branch
git volume sync  # usa o git-volume.yaml do pai
```

## 🛡️ Funcionalidades de Segurança

- **Deteção de Alterações no Unsync**: Ficheiros copiados em modo *copy* são preservados se modificados.
- **Idempotente**: Executar `sync` várias vezes é seguro.

## 📄 Licença

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
