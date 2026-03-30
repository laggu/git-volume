#!/usr/bin/env bash

set -euo pipefail

VERSION="${1:?version is required}"
SHA256="${2:?sha256 is required}"
OUTPUT="${3:?output path is required}"

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
      "-s", "-w",
      "-X", "github.com/laggu/git-volume/cmd.version=#{version}",
    ]

    system "go", "build", *std_go_args(ldflags: ldflags), "./main.go"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/git-volume version")
  end
end
EOF
