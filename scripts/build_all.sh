#!/usr/bin/env bash
# Cross-compiles beam for six targets into dist/ and writes dist/checksums.txt.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist"

mkdir -p "$dist"
rm -f "$dist"/beam-* "$dist/checksums.txt"
