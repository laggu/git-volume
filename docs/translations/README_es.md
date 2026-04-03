> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"El código en Git, el entorno como volúmenes."**

`git-volume` es una herramienta CLI que gestiona centralmente archivos de entorno (`.env`, secretos, etc.) entre árboles de trabajo Git y los monta dinámicamente.

## ✨ Características principales

- **Montaje de volúmenes**: Soporte para enlaces simbólicos o copia de archivos
- **Herencia de configuración**: Los árboles de trabajo hijos heredan automáticamente la configuración del padre
- **Limpieza segura**: El contenido modificado por el usuario o no relacionado no se elimina durante `unsync`
- **Optimizado para agentes IA**: Crear árbol de trabajo + configurar entorno con un solo comando

## 📦 Instalación

### Homebrew
```bash
brew install laggu/tap/git-volume
```

La instalación con Homebrew se distribuye como **formula** y compila `git-volume` desde el código fuente en la máquina del usuario. Esto evita depender de binarios precompilados de macOS sin firmar, pero significa que Homebrew instalará Go como dependencia de compilación.

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

**4. Verificar estado**
```bash
git volume status
```

## 📖 Comandos

| Comando                    | Descripción                                                     |
| -------------------------- | --------------------------------------------------------------- |
| `git volume init`          | Crear directorio global y archivo de configuración ejemplo      |
| `git volume sync`          | Montar volúmenes en el árbol de trabajo actual                  |
| `git volume unsync`        | Eliminar volúmenes montados (se preserva el contenido modificado o no relacionado) |
| `git volume status`        | Mostrar el estado actual de los volúmenes                       |
| `git volume global add`    | Copiar archivos al almacenamiento global (`~/.git-volume`)      |
| `git volume global list`   | Listar archivos en el almacenamiento global (vista árbol)       |
| `git volume global edit`   | Editar un archivo del almacenamiento global con `$EDITOR`       |
| `git volume global remove` | Eliminar archivos del almacenamiento global (alias: `rm`)       |
| `git volume version`       | Mostrar información de versión                                  |

## ⚙️ Archivo de configuración (`git-volume.yaml`)

```yaml
volumes:
  # Formato simple (por defecto: enlace simbólico)
  - ".env.shared:.env"

  # Con opciones
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (por defecto) o copy

  # Montar desde almacenamiento global (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # Montaje de directorios (copia en overlay solo las entradas del source al directorio target)
  - mount: "configs:app/configs"
    mode: "copy"
```

### Comparación de modos

| Modo   | Descripción            | Caso de uso                                                         |
| ------ | ---------------------- | ------------------------------------------------------------------- |
| `link` | Crear enlace simbólico | Desarrollo local (los cambios se reflejan al instante)              |
| `copy` | Copiar archivo / copiar directorio en overlay | Compilaciones Docker (entornos sin soporte para enlaces simbólicos) |

### Comportamiento de directorios en modo copy

Cuando `mode: "copy"` usa un directorio como source, `sync` superpone las entradas del source en el directorio target sin borrar el directorio raíz target. Los archivos existentes no relacionados se preservan, los conflictos con archivo/enlace simbólico se reemplazan y los conflictos archivo-vs-directorio fallan de forma segura. `status` y `unsync` también operan sobre el subconjunto copiado del source, no sobre una coincidencia exacta de todo el directorio.

## 🔄 Herencia de árboles de trabajo

Si un árbol de trabajo hijo no tiene `git-volume.yaml`, se usa automáticamente la configuración del árbol de trabajo padre (principal).

```bash
# La configuración solo existe en el árbol de trabajo principal
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # archivo de configuración
├── .env.shared       # archivo fuente
└── ...

# Al ejecutar sync en el árbol de trabajo hijo se usa la config del padre
cd ../feature-branch
git volume sync  # usa el git-volume.yaml del padre
```

## 🌐 Almacenamiento global

El almacenamiento global (`~/.git-volume`) permite compartir archivos entre múltiples proyectos usando el prefijo `@global/`.

```bash
# Agregar archivos al almacenamiento global
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# Listar, editar y eliminar
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

Use `@global/` en su configuración para referenciar estos archivos:

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 Opciones CLI

| Bandera           | Comandos         | Descripción                                            |
| ----------------- | ---------------- | ------------------------------------------------------ |
| `--dry-run`       | `sync`, `unsync` | Mostrar qué se haría sin realizar cambios              |
| `--relative`      | `sync`           | Crear enlaces simbólicos relativos en vez de absolutos |
| `--verbose`, `-v` | Todos            | Nivel de salida: 0=solo errores, 1=normal (por defecto), 2=detallado |
| `--config`, `-c`  | Todos            | Ruta personalizada del archivo de configuración        |

## 🛡️ Características de seguridad

- **Rechazo de fuentes simbólicas**: `sync` y `global add` rechazan fuentes que son enlaces simbólicos por seguridad
- **Prevención de recorrido de rutas**: Todas las rutas se validan para prevenir ataques de escape de directorio
- **Seguridad de copia overlay**: El modo copy de directorios preserva archivos target no relacionados, reemplaza solo conflictos de archivo/enlace simbólico y falla en conflictos archivo-vs-directorio
- **Detección de cambios en Unsync**: `unsync` elimina solo el subconjunto copiado del source y preserva el contenido modificado o no relacionado
- **Detección de cambios en Status**: `status` muestra `MODIFIED` cuando las entradas copiadas del source difieren del target
- **Sync idempotente**: Ejecutar `sync` múltiples veces siempre produce el mismo resultado

## 📄 Licencia

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
