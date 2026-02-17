# git-volume Go 스타일 가이드

이 프로젝트는 표준 Go 컨벤션과 "Effective Go"를 따릅니다.

## 1. 일반적인 포맷팅 (General Formatting)
- **gofmt**: 모든 코드는 반드시 `gofmt`로 포맷팅되어야 합니다.
- **Imports**: 표준 라이브러리, 서드파티 라이브러리, 로컬 패키지 순으로 그룹화합니다.

## 2. 네이밍 규칙 (Naming Conventions)
- **PascalCase**: 외부로 노출(Export)되는 타입, 함수, 변수에 사용합니다.
- **camelCase**: 내부 전용(Private) 식별자에 사용합니다.
- **Interfaces**: 메서드가 하나인 인터페이스는 `-er`로 끝납니다 (예: `Reader`, `Mounter`).
- **Variables**: 짧고 명확하게 짓습니다 (예: 루프에는 `i`, 컨텍스트는 `ctx`, 에러는 `err`).

## 3. 에러 처리 (Error Handling)
- **명시적 확인**: 에러는 항상 명시적으로 확인해야 합니다. `_`로 에러를 무시하지 마세요.
- **Wrapping**: `fmt.Errorf("...: %w", err)`를 사용하여 에러에 컨텍스트를 추가(Wrap)합니다.
- **Sentinel Errors**: 특정 체크가 필요한 에러는 Export된 Sentinel Error로 정의합니다 (예: `ErrNotFound`).
- **No Panic**: 일반적인 제어 흐름에서 `panic`을 사용하지 마세요. 복구 불가능한 초기화 에러에만 사용합니다.

## 4. 프로젝트 구조 (Project Structure)
- **cmd/**: 메인 애플리케이션 및 CLI 명령어 (Cobra)가 위치합니다. 여기에는 로직을 최소한으로 유지하세요.
- **internal/**: 외부에서 import 하면 안 되는 비공개 애플리케이션 및 라이브러리 코드입니다.
- **pkg/**: 외부 애플리케이션에서 사용해도 되는 라이브러리 코드입니다 (필요한 경우만).

## 5. 테스팅 (Testing)
- **Table-Driven Tests**: 여러 케이스를 커버할 때는 Table-Driven 테스트 방식을 사용합니다.
- **Test Packages**: 내부 접근이 필요하면 동일 패키지에서, 블랙박스 테스트가 필요하면 `_test` 패키지에서 테스트합니다.

## 6. CLI & UX (Cobra)
- **RunE**: 에러 반환을 위해 `Run` 대신 `RunE`를 사용합니다.
- **SilenceUsage**: 에러가 발생할 때마다 Help 메시지가 출력되지 않도록 `SilenceUsage: true`로 설정합니다.

## 7. git-volume 특화 규칙 (git-volume Specifics)
- **무상태 (Statelessness)**: 핵심 로직은 무상태여야 합니다. 전역 변수 사용을 피하세요.
- **멱등성 (Idempotency)**: `sync` 작업은 여러 번 실행해도 결과가 같아야 합니다.
- **보안 (Security)**: 시크릿 파일을 복사할 때 파일 권한 설정에 주의하세요.

## 8. 코드 리뷰 지침 (Code Review Guidelines)

### 보안 리뷰 시 맥락 고려
git-volume은 **개발자 로컬 환경에서 실행되는 CLI 도구**입니다. 보안 취약점을 지적할 때는 반드시 실제 공격 시나리오가 성립하는지 확인하세요.

**리뷰하지 말아야 할 것:**
- 공격자가 사용자의 로컬 파일시스템에 쓰기 권한이 있다고 가정하는 시나리오 (예: symlink 공격)
- 이미 시스템이 compromise된 상태를 전제로 하는 취약점
- 웹 서버나 다중 사용자 환경에서만 의미 있는 보안 경고를 로컬 CLI 도구에 적용

**리뷰해야 할 것:**
- 실제 사용 맥락에서 발생할 수 있는 보안 이슈
- 파일 권한 설정 오류
- 시크릿 노출 가능성
