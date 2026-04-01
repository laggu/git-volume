> 🌐 [English](../../README.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"Git에는 코드만, 환경은 볼륨으로."**

`git-volume`은 Git 워크트리 간에 환경 파일(`.env`, 시크릿 등)을 중앙에서 관리하고 동적으로 마운트해주는 CLI 도구입니다.

## ✨ 주요 기능

- **볼륨 마운트**: 심볼릭 링크 또는 파일 복사 방식 지원
- **설정 상속**: 자식 워크트리가 부모의 설정을 자동 상속
- **안전한 정리**: 사용자가 수정했거나 무관한 복사 대상은 unsync에서 삭제하지 않음
- **AI 에이전트 최적화**: 한 줄 명령으로 워크트리 생성 + 환경 구성

## 📦 설치

### Homebrew
```bash
brew install laggu/tap/git-volume
```

Homebrew 설치는 **formula** 방식으로 배포되며, `git-volume`을 사용자 컴퓨터에서 소스 빌드합니다. 이 방식은 unsigned macOS 사전 빌드 바이너리에 의존하지 않지만, Homebrew가 빌드 의존성으로 Go를 설치하게 됩니다.

### Scoop (Windows)
```bash
scoop bucket add laggu https://github.com/laggu/scoop-bucket.git
scoop install git-volume
```

### Go
```bash
go install github.com/laggu/git-volume@latest
```

## 🚀 빠른 시작

**1. 초기화**
```bash
git volume init
```

**2. git-volume.yaml 작성**
```yaml
volumes:
  - ".env.shared:.env"
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"
```

**3. 볼륨 마운트**
```bash
git volume sync
```

**4. 상태 확인**
```bash
git volume status
```

## 📖 명령어

| 명령어                     | 설명                                        |
| -------------------------- | ------------------------------------------- |
| `git volume init`          | 글로벌 디렉토리 생성 및 샘플 설정 파일 생성 |
| `git volume sync`          | 설정에 따라 볼륨을 현재 워크트리에 마운트   |
| `git volume unsync`        | 마운트된 볼륨 제거 (수정되었거나 무관한 내용은 보존) |
| `git volume status`        | 현재 볼륨 상태 표시                         |
| `git volume global add`    | 글로벌 저장소(`~/.git-volume`)에 파일 복사  |
| `git volume global list`   | 글로벌 저장소의 파일 목록 (트리 뷰)         |
| `git volume global edit`   | 글로벌 저장소의 파일을 `$EDITOR`로 편집     |
| `git volume global remove` | 글로벌 저장소에서 파일 삭제 (alias: `rm`)   |
| `git volume version`       | 버전 정보 출력                              |

## ⚙️ 설정 파일 (`git-volume.yaml`)

```yaml
volumes:
  # 단순 형식 (기본: 심볼릭 링크)
  - ".env.shared:.env"

  # 옵션이 필요할 때
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (기본) 또는 copy

  # 글로벌 저장소에서 마운트 (~/.git-volume)
  - "@global/secrets/prod.key:config/key"

  # 디렉토리 마운트 (source 항목만 target 디렉토리에 overlay 복사)
  - mount: "configs:app/configs"
    mode: "copy"
```

### 모드 비교

| 모드   | 설명             | 용도                                  |
| ------ | ---------------- | ------------------------------------- |
| `link` | 심볼릭 링크 생성 | 로컬 개발 (원본 수정 시 즉시 반영)    |
| `copy` | 파일 복사 / 디렉토리 overlay 복사 | Docker 빌드 (심볼릭 링크 미지원 환경) |

### Copy 모드 디렉토리 동작

`mode: "copy"`에서 source가 디렉토리이면 `sync`는 target 루트 디렉토리를 지우지 않고 source 항목만 overlay 복사합니다. 무관한 기존 파일은 유지되고, 파일/심볼릭 링크 충돌은 교체되며, 파일-디렉토리 충돌은 안전하게 실패합니다. `status`와 `unsync`도 전체 디렉토리가 아니라 복사된 source subset 기준으로 동작합니다.

## 🔄 워크트리 상속

자식 워크트리에 `git-volume.yaml`이 없으면 부모(메인) 워크트리의 설정을 자동으로 사용합니다.

```bash
# 메인 워크트리에만 설정 존재
main-repo/
├── .git/             # git common dir
├── git-volume.yaml   # 설정 파일
├── .env.shared       # 소스 파일
└── ...

# 자식 워크트리에서 sync 실행 시 부모 설정 사용
cd ../feature-branch
git volume sync  # 부모의 git-volume.yaml 사용
```

## 🌐 글로벌 저장소

글로벌 저장소(`~/.git-volume`)를 사용하면 `@global/` 접두사를 통해 여러 프로젝트에서 파일을 공유할 수 있습니다.

```bash
# 글로벌 저장소에 파일 추가
git volume global add .env
git volume global add .env.local --as .env
git volume global add .env config.json --path myproject

# 목록 확인, 편집, 삭제
git volume global list
git volume global edit config.json
git volume global remove old-secret.key
```

설정 파일에서 `@global/`로 참조하여 사용합니다:

```yaml
volumes:
  - "@global/.env:.env"
  - mount: "@global/secrets/prod.key:config/key"
    mode: "copy"
```

## 🔧 CLI 옵션

| 플래그            | 대상 명령어      | 설명                                      |
| ----------------- | ---------------- | ----------------------------------------- |
| `--dry-run`       | `sync`, `unsync` | 실제 변경 없이 수행할 작업만 표시         |
| `--relative`      | `sync`           | 절대 경로 대신 상대 경로 심볼릭 링크 생성 |
| `--verbose`, `-v` | 전체             | 출력 레벨: 0=에러만, 1=기본(기본값), 2=상세 |
| `--config`, `-c`  | 전체             | 설정 파일 경로 지정                       |

## 🛡️ 안전 장치

- **심볼릭 링크 소스 차단**: `sync`와 `global add`에서 심볼릭 링크 소스를 보안상 거부
- **경로 탐색 방지**: 모든 경로에 대해 디렉토리 이스케이프 공격 검증
- **Overlay 복사 안전성**: 디렉토리 copy 모드는 무관한 target 파일을 유지하고, 파일/심볼릭 링크 충돌만 교체하며, 파일-디렉토리 충돌은 실패
- **Unsync 시 변경 감지**: `unsync`는 복사된 source subset만 제거하고, 수정되었거나 무관한 내용은 보존
- **Status 변경 감지**: 복사된 source 항목이 target과 다르면 `MODIFIED`로 표시
- **멱등성 보장**: `sync`를 여러 번 실행해도 항상 동일한 결과

## 📄 라이선스

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
