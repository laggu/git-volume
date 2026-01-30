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
go test ./internal/config
go test ./internal/mounter

# 특정 테스트 함수 실행
go test ./internal/mounter -run TestMounter_Sync_Link

# 의존성 설치
go mod download
```

## 아키텍처

### CLI 구조 (Cobra 기반)
- `cmd/root.go` - 루트 명령어 및 전역 설정
- `cmd/*.go` - 각 서브커맨드 (init, sync, unsync, list, worktree)

### 핵심 패키지

**`internal/finder`** - 설정 파일 탐색 및 컨텍스트 결정
- `FindContext()`: git-volume.yaml 위치를 찾고 실행 컨텍스트(Config, SourceDir, TargetDir) 반환
- 상속 로직: 현재 워크트리에 설정이 없으면 메인 워크트리(git common dir)에서 설정을 상속

**`internal/config`** - YAML 설정 파싱
- 두 가지 볼륨 정의 형식 지원:
  - 단순 문자열: `"source:target"`
  - 객체 형식: `mount`, `mode`, `force` 필드 포함
- `UnmarshalYAML()`: 커스텀 YAML 파싱 로직

**`internal/mounter`** - 볼륨 마운트/언마운트 실행
- `Sync()`: 볼륨을 타겟 워크스페이스에 적용 (link 또는 copy 모드)
- `Unsync()`: 마운트된 볼륨 제거 (상태 없이 해시/심볼릭 링크 검증으로 안전하게 처리)
- 안전 장치: 수정된 파일은 삭제하지 않고 보존

### 데이터 흐름
1. `finder.FindContext()` → 설정 파일 위치와 소스/타겟 디렉토리 결정
2. `config.LoadConfig()` → YAML 파싱하여 Volume 슬라이스 반환
3. `mounter.Sync/Unsync()` → SourceDir 기준 소스 경로, TargetDir 기준 타겟 경로로 파일 조작

## 설정 파일 형식 (git-volume.yaml)

```yaml
volumes:
  - ".env.shared:.env"                    # 단순 형식 (기본: link 모드)
  - mount: "secrets/prod.key:config/key"  # 객체 형식
    mode: "copy"                          # link 또는 copy
    force: true                           # 기존 파일 덮어쓰기
```
