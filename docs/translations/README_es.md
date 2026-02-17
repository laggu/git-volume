> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"Mantén el código en Git, monta tu entorno como volúmenes."**

`git-volume` es una herramienta de CLI que gestiona de forma centralizada los archivos de entorno (`.env`, secretos, etc.) entre los *worktrees* de Git y los monta dinámicamente.

## ✨ Características principales

- **Montaje de volúmenes**: Soporta modos de enlace simbólico o copia de archivos.
- **Herencia de configuración**: Los *worktrees* secundarios heredan automáticamente la configuración del padre.
- **Limpieza segura**: Los archivos modificados se preservan durante el *unsync*.
- **Optimizado para agentes de IA**: Crea un *worktree* y configura el entorno con un solo comando.

## 📦 Instalación

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

## 🚀 Inicio rápido

**1. Inicializar**
```bash
git volume init
```

**2. Crear git-volume.yaml**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. Montar volúmenes**
```bash
git volume sync
```

**4. Ver estado**
```bash
git volume status
```

## 📖 Comandos

| Comando             | Descripción                                                             |
| ------------------- | ----------------------------------------------------------------------- |
| `git volume init`   | Crea el directorio global y un archivo de configuración de ejemplo.     |
| `git volume sync`   | Monta volúmenes en el worktree actual según la configuración.           |
| `git volume unsync` | Elimina los volúmenes montados (los archivos modificados se conservan). |
| `git volume status` | Muestra el estado actual de los volúmenes.                              |

## ⚙️ Archivo de configuración (`git-volume.yaml`)

```yaml
volumes:
  # Formato simple (predeterminado: enlace simbólico)
  - ".env.shared:.env"

  # Con opciones
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (predeterminado) o copy

```

### Comparación de modos

| Modo   | Descripción              | Caso de uso                                                      |
| ------ | ------------------------ | ---------------------------------------------------------------- |
| `link` | Crea un enlace simbólico | Desarrollo local (los cambios se reflejan al instante).          |
| `copy` | Copia el archivo         | Builds de Docker (entornos sin soporte para enlaces simbólicos). |

## 🔄 Herencia de Worktree

Si un *worktree* secundario no tiene `git-volume.yaml`, utiliza automáticamente la configuración del *worktree* principal (padre).

```bash
# La configuración solo existe en el worktree principal
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # archivo de configuración
├── .env.shared       # archivo de origen
└── ...

# Ejecutar sync en un worktree secundario usa la configuración del padre
cd ../feature-branch
git volume sync  # usa el git-volume.yaml del padre
```

## 🛡️ Características de seguridad

- **Detección de cambios al desincronizar**: Los archivos copiados en modo *copy* se conservan si han sido modificados.
- **Idempotente**: Ejecutar `sync` varias veces es seguro.

## 📄 Licencia

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
