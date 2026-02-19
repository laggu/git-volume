> 🌐 [English](../../README.md) | [한국어](README_ko.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"El código en Git, el entorno como volúmenes."**

`git-volume` es una herramienta CLI que gestiona centralmente archivos de entorno (`.env`, secretos, etc.) entre árboles de trabajo Git y los monta dinámicamente.

## ✨ Características principales

- **Montaje de volúmenes**: Soporte para enlaces simbólicos o copia de archivos
- **Herencia de configuración**: Los árboles de trabajo hijos heredan automáticamente la configuración del padre
- **Limpieza segura**: Los archivos modificados por el usuario no se eliminan
- **Optimizado para agentes IA**: Crear árbol de trabajo + configurar entorno con un solo comando

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

**4. Verificar estado**
```bash
git volume status
```

## 📖 Comandos

| Comando                    | Descripción                                                     |
| -------------------------- | --------------------------------------------------------------- |
| `git volume init`          | Crear directorio global y archivo de configuración ejemplo      |
| `git volume sync`          | Montar volúmenes en el árbol de trabajo actual                  |
| `git volume unsync`        | Eliminar volúmenes montados (se preservan archivos modificados) |
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

  # Montaje de directorios (copia el directorio completo)
  - mount: "configs:app/configs"
    mode: "copy"
```

### Comparación de modos

| Modo   | Descripción            | Caso de uso                                                         |
| ------ | ---------------------- | ------------------------------------------------------------------- |
| `link` | Crear enlace simbólico | Desarrollo local (los cambios se reflejan al instante)              |
| `copy` | Copiar archivo         | Compilaciones Docker (entornos sin soporte para enlaces simbólicos) |

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
| `--verbose`, `-v` | Todos            | Salida detallada                                       |
| `--quiet`, `-q`   | Todos            | Ocultar salida no relacionada con errores              |
| `--config`, `-c`  | Todos            | Ruta personalizada del archivo de configuración        |

## 🛡️ Características de seguridad

- **Rechazo de fuentes simbólicas**: `sync` y `global add` rechazan fuentes que son enlaces simbólicos por seguridad
- **Prevención de recorrido de rutas**: Todas las rutas se validan para prevenir ataques de escape de directorio
- **Detección de cambios en Unsync**: Los archivos y directorios copiados se preservan si fueron modificados
- **Detección de cambios en Status**: Los archivos copiados que difieren del origen se muestran como `MODIFIED`
- **Sync idempotente**: Ejecutar `sync` múltiples veces siempre produce el mismo resultado

## 📄 Licencia

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
