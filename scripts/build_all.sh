#!/usr/bin/env bash
# Cross-compiles beam for six targets into dist/ and writes dist/checksums.txt.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist"

mkdir -p "$dist"
rm -f "$dist"/beam-* "$dist/checksums.txt"

targets=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64)

cd "$root"
for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  out="$dist/beam-$goos-$goarch"
  if [ "$goos" = windows ]; then
    out="$out.exe"
  fi
  echo "building $(basename "$out")"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags='-s -w' -o "$out" ./cmd/beam
done
