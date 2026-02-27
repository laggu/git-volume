# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 프로젝트 개요

`git-volume`은 Git 워크트리 간에 환경 파일(.env, 시크릿 등)을 중앙에서 관리하고 심볼릭 링크 또는 파일 복사 방식으로 마운트하는 Go CLI 도구입니다.

## 빌드 및 테스트 명령어

```bash
# 빌드
go build -o git-volume .

# 전체 테스트 실행
go test ./...

# 특정 패키지 테스트
go test ./internal/gitvolume

# 특정 테스트 함수 실행
go test ./internal/gitvolume -run TestContext_Load

# 의존성 설치
go mod download
```

## 아키텍처

### CLI 구조 (Cobra 기반)
- `cmd/root.go` - 루트 명령어 및 전역 플래그 (`--config`, `--verbose`)
- `cmd/*.go` - 각 서브커맨드 (init, sync, unsync, list, add, version)

### 핵심 패키지 (`internal/gitvolume`)

모든 핵심 로직이 단일 패키지에 통합되어 있습니다:

**`gitvolume.go`** - 메인 진입점
- `GitVolume` 구조체: CLI 명령어들이 사용하는 메인 타입
- `New()`: 옵션으로 인스턴스 생성
- `Load()`: 설정 파일 로드 (sync, unsync, list 전에 호출 필수)

**`context.go`** - 설정 파싱 및 컨텍스트 관리
- `Context`: SourceDir, TargetDir, GlobalDir, Volumes 보관
- `Volume`: 단순 문자열(`"source:target"`) 및 객체 형식 YAML 파싱 지원
- `@global/` 접두사로 글로벌 디렉토리(`~/.git-volume`) 참조 가능
- 설정 상속: 현재 워크트리에 설정이 없으면 메인 워크트리에서 상속

**`git.go`** - Git 워크트리 탐지
- `FindWorktreeRoot()`: 현재 워크트리 루트 찾기
- `findCommonDir()`: 메인 워크트리 루트 찾기 (bare repo 지원)

**`sync.go` / `unsync.go`** - 볼륨 마운트/언마운트
- `Sync()`: link 또는 copy 모드로 볼륨 적용
- `Unsync()`: 해시/심볼릭 링크 검증으로 안전하게 볼륨 제거
- 보안: 경로 탈출 공격 방지, 소스 심볼릭 링크 거부

### 데이터 흐름
1. `New()` → GitVolume 인스턴스 생성 (GlobalDir만 해석)
2. `Load()` → 설정 파일 탐색, YAML 파싱, SourceDir/TargetDir/Volumes 설정
3. `Sync()/Unsync()/List()` → 해석된 경로로 파일 조작

## 설정 파일 형식 (git-volume.yaml)

```yaml
volumes:
  - ".env.shared:.env"                    # 단순 형식 (기본: link 모드)
  - mount: "secrets/prod.key:config/key"  # 객체 형식
    mode: "copy"                          # link 또는 copy

  - "@global/secrets/key:config/key"      # 글로벌 디렉토리 참조
```
