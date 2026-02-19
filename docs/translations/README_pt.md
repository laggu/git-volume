> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"Código no Git, ambiente como volumes."**

`git-volume` é uma ferramenta CLI que gerencia centralmente arquivos de ambiente (`.env`, segredos, etc.) entre árvores de trabalho Git e os monta dinamicamente.

## ✨ Principais recursos

- **Montagem de volumes**: Suporte para links simbólicos ou cópia de arquivos
- **Herança de configuração**: Árvores de trabalho filhas herdam automaticamente a configuração pai
- **Limpeza segura**: Arquivos modificados pelo usuário não são excluídos
- **Otimizado para agentes IA**: Criar árvore de trabalho + configurar ambiente com um único comando

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

## 🚀 Início rápido

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

**3. Montar volumes**
```bash
git volume sync
```

**4. Verificar status**
```bash
git volume status
```

## 📖 Comandos

| Comando                    | Descrição                                                        |
| -------------------------- | ---------------------------------------------------------------- |
| `git volume init`          | Criar diretório global e arquivo de configuração exemplo         |
| `git volume sync`          | Montar volumes na árvore de trabalho atual                       |
| `git volume unsync`        | Remover volumes montados (arquivos modificados são preservados)  |
| `git volume status`        | Exibir o status atual dos volumes                                |
| `git volume global add`    | Copiar arquivos para o armazenamento global (`~/.git-volume`)    |
| `git volume global list`   | Listar arquivos no armazenamento global (visualização em árvore) |
| `git volume global edit`   | Editar um arquivo do armazenamento global com `$EDITOR`          |
| `git volume global remove` | Remover arquivos do armazenamento global (alias: `rm`)           |
| `git volume version`       | Exibir informações de versão                                     |

## ⚙️ Arquivo de configuração (`git-volume.yaml`)

```yaml
volumes:
  # Formato simples (padrão: link simbólico)
  - ".env.shared:.env"

  # Com opções
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (padrão) ou copy

  # Montar do armazenamento global (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # Montagem de diretórios (copia o diretório inteiro)
  - mount: "configs:app/configs"
    mode: "copy"
```

### Comparação de modos

| Modo   | Descrição            | Caso de uso                                               |
| ------ | -------------------- | --------------------------------------------------------- |
| `link` | Criar link simbólico | Desenvolvimento local (alterações refletem imediatamente) |
| `copy` | Copiar arquivo       | Builds Docker (ambientes sem suporte a links simbólicos)  |

## 🔄 Herança de árvores de trabalho

Se uma árvore de trabalho filha não tiver `git-volume.yaml`, a configuração da árvore de trabalho pai (principal) é usada automaticamente.

```bash
# Configuração existe apenas na árvore de trabalho principal
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # arquivo de configuração
├── .env.shared       # arquivo fonte
└── ...

# Ao executar sync na árvore de trabalho filha, usa a config do pai
cd ../feature-branch
git volume sync  # usa o git-volume.yaml do pai
```

## 🌐 Armazenamento global

O armazenamento global (`~/.git-volume`) permite compartilhar arquivos entre múltiplos projetos usando o prefixo `@global/`.

```bash
# Adicionar arquivos ao armazenamento global
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# Listar, editar e remover
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

Use `@global/` em sua configuração para referenciar esses arquivos:

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 Opções CLI

| Flag              | Comandos         | Descrição                                            |
| ----------------- | ---------------- | ---------------------------------------------------- |
| `--dry-run`       | `sync`, `unsync` | Mostrar o que seria feito sem realizar alterações    |
| `--relative`      | `sync`           | Criar links simbólicos relativos em vez de absolutos |
| `--verbose`, `-v` | Todos            | Saída detalhada                                      |
| `--quiet`, `-q`   | Todos            | Ocultar saída não relacionada a erros                |
| `--config`, `-c`  | Todos            | Caminho personalizado do arquivo de configuração     |

## 🛡️ Recursos de segurança

- **Rejeição de fontes simbólicas**: `sync` e `global add` rejeitam fontes que são links simbólicos por segurança
- **Prevenção de travessia de caminho**: Todos os caminhos são validados para prevenir ataques de escape de diretório
- **Detecção de alterações no Unsync**: Arquivos e diretórios copiados são preservados se modificados
- **Detecção de alterações no Status**: Arquivos copiados diferentes do original são exibidos como `MODIFIED`
- **Sync idempotente**: Executar `sync` múltiplas vezes sempre produz o mesmo resultado

## 📄 Licença

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
