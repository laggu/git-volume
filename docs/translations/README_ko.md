> 🌐 [English](../../README.md) | [日本語](README_ja.md) | [中文](README_zh.md) | [Español](README_es.md) | [Português](README_pt.md) | [Français](README_fr.md) | [Deutsch](README_de.md) | [Italiano](README_it.md)

# git-volume

> **"Git에는 코드만, 환경은 볼륨으로."**

`git-volume`은 Git 워크트리 간에 환경 파일(`.env`, 시크릿 등)을 중앙에서 관리하고 동적으로 마운트해주는 CLI 도구입니다.

## ✨ 주요 기능

- **볼륨 마운트**: 심볼릭 링크 또는 파일 복사 방식 지원
- **설정 상속**: 자식 워크트리가 부모의 설정을 자동 상속
- **안전한 정리**: 사용자가 수정한 파일은 삭제하지 않음
- **AI 에이전트 최적화**: 한 줄 명령으로 워크트리 생성 + 환경 구성

## 📦 설치

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

| 명령어              | 설명                                        |
| ------------------- | ------------------------------------------- |
| `git volume init`   | 글로벌 디렉토리 생성 및 샘플 설정 파일 생성 |
| `git volume sync`   | 설정에 따라 볼륨을 현재 워크트리에 마운트   |
| `git volume unsync` | 마운트된 볼륨 제거 (수정된 파일은 보존)     |
| `git volume status` | 현재 볼륨 상태 표시                         |

## ⚙️ 설정 파일 (`git-volume.yaml`)

```yaml
volumes:
  # 단순 형식 (기본: 심볼릭 링크)
  - ".env.shared:.env"

  # 옵션이 필요할 때
  - mount: "secrets/prod.key:config/prod.key"
    mode: "copy"   # link (기본) 또는 copy
    force: true    # 존재 시 덮어쓰기
```

### 모드 비교

| 모드   | 설명             | 용도                                  |
| ------ | ---------------- | ------------------------------------- |
| `link` | 심볼릭 링크 생성 | 로컬 개발 (원본 수정 시 즉시 반영)    |
| `copy` | 파일 복사        | Docker 빌드 (심볼릭 링크 미지원 환경) |

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

## 🛡️ 안전 장치

- **Unsync 시 변경 감지**: Copy 모드로 복사된 파일이 수정되었으면 삭제하지 않고 보존
- **멱등성**: `sync`를 여러 번 실행해도 안전

## 📄 라이선스

[GNU GENERAL PUBLIC LICENSE](https://www.gnu.org/licenses/gpl-3.0.html)
