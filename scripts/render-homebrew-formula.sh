#!/usr/bin/env bash

set -euo pipefail

VERSION="${1:?version is required}"
VERSION="${VERSION#v}"
SHA256="${2:?sha256 is required}"
COMMIT="${3:?commit is required}"
DATE="${4:?date is required}"
OUTPUT="${5:?output path is required}"

mkdir -p "$(dirname "$OUTPUT")"

cat >"$OUTPUT" <<EOF
class GitVolume < Formula
  desc "Manage environment files across Git worktrees"
  homepage "https://github.com/laggu/git-volume"
  url "https://github.com/laggu/git-volume/archive/refs/tags/v${VERSION}.tar.gz"
  sha256 "${SHA256}"
  license "GPL-3.0-only"

  depends_on "go" => :build

  def install
    ldflags = [
      "-s -w",
      "-X github.com/laggu/git-volume/cmd.version=#{version}",
      "-X github.com/laggu/git-volume/cmd.commit=${COMMIT}",
      "-X github.com/laggu/git-volume/cmd.date=${DATE}",
    ].join(" ")

    system "go", "build", *std_go_args(ldflags: ldflags)
  end

  test do
    output = shell_output("#{bin}/git-volume version")

    assert_includes output, version.to_s
    assert_includes output, "commit: ${COMMIT}"
    assert_includes output, "built:  ${DATE}"
  end
end
EOF
