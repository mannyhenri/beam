#!/usr/bin/env bash
# Cross-compiles beam for six targets into dist/ and writes dist/checksums.txt.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist"

mkdir -p "$dist"
rm -f "$dist"/beam-* "$dist/checksums.txt"

max_bytes=$((15 * 1024 * 1024))
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

  size=$(($(wc -c <"$out")))
  if [ "$size" -ge "$max_bytes" ]; then
    echo "build_all.sh: $(basename "$out") is $size bytes; limit is under $max_bytes" >&2
    exit 1
  fi
done
