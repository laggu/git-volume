#!/usr/bin/env bash

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
OUTPUT_DIR="$(mktemp -d)"
OUTPUT_FILE="${OUTPUT_DIR}/git-volume.rb"
VERSION="0.3.1"
SHA256="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
COMMIT="27836373d30ec5d4b20eaba0fcb32197c13eee4a"
DATE="2026-04-05T02:26:42+09:00"

cleanup() {
    rm -rf "${OUTPUT_DIR}"
}
trap cleanup EXIT

"${PROJECT_ROOT}/scripts/render-homebrew-formula.sh" \
    "${VERSION}" \
    "${SHA256}" \
    "${COMMIT}" \
    "${DATE}" \
    "${OUTPUT_FILE}"

grep -F 'url "https://github.com/laggu/git-volume/archive/refs/tags/v0.3.1.tar.gz"' "${OUTPUT_FILE}"
grep -F 'sha256 "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"' "${OUTPUT_FILE}"
grep -F -- '-X github.com/laggu/git-volume/cmd.version=#{version}' "${OUTPUT_FILE}"
grep -F -- "-X github.com/laggu/git-volume/cmd.commit=${COMMIT}" "${OUTPUT_FILE}"
grep -F -- "-X github.com/laggu/git-volume/cmd.date=${DATE}" "${OUTPUT_FILE}"
grep -F "assert_includes output, \"commit: ${COMMIT}\"" "${OUTPUT_FILE}"
grep -F "assert_includes output, \"built:  ${DATE}\"" "${OUTPUT_FILE}"
