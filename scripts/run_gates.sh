#!/usr/bin/env bash
# Runs the gate for one phase from a clean checkout of HEAD and, if it passes,
# records an empty commit "Gate: Phase <N> passed" on the current branch.
#   scripts/run_gates.sh <phase 0-7>
# Exit codes: 0 gate passed and marked; 1 a check failed (no commit);
# 2 usage error, missing tool, missing previous gate, or phase without a gate.
set -euo pipefail

usage() {
  echo "usage: scripts/run_gates.sh <phase 0-7>" >&2
  exit 2
}
if [ $# -ne 1 ] || ! [[ $1 =~ ^[0-7]$ ]]; then
  usage
fi
phase=$1

for tool in git go node bash; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "run_gates.sh: $tool is required but not found" >&2
    exit 2
  fi
done

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

# One phase at a time: phase N needs the exact marker of phase N-1. The subject
# must match exactly, since other commits mention "Phase 3" and the like.
# (grep without -q: under pipefail an early grep exit would fail git log.)
if [ "$phase" -gt 0 ]; then
  prev=$((phase - 1))
  if ! git log --format=%s | grep -Fx "Gate: Phase $prev passed" >/dev/null; then
    echo "run_gates.sh: phase $phase needs the phase $prev gate first; no 'Gate: Phase $prev passed' commit on this branch" >&2
    exit 2
  fi
fi

# Phase 0 directories, listed by name so this script's own tests are not part of the gate.
phase0_dirs=(acceptance/go-project-skeleton acceptance/six-target-build-script acceptance/simnet-harness acceptance/no-direct-check)
case "$phase" in
  0) ;;
  *)
    echo "run_gates.sh: gate for phase $phase is not defined yet" >&2
    exit 2
    ;;
esac

# Only committed files count: run in a temporary worktree of HEAD, removed on any exit.
validated="$(git rev-parse HEAD)"
tmp="$(mktemp -d)"
worktree="$tmp/checkout"
trap 'git -C "$root" worktree remove --force "$worktree" >/dev/null 2>&1 || true; git -C "$root" worktree prune; rm -rf "$tmp"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
git worktree add --quiet --detach "$worktree" "$validated"
cd "$worktree"
export GOTOOLCHAIN=local GOFLAGS=''

# Runs one check; on failure prints its output and stops the gate.
check() {
  local name="$1"
  shift
  if "$@" </dev/null >"$tmp/output" 2>&1; then
    echo "PASS $name"
  else
    echo "FAIL $name"
    cat "$tmp/output"
    exit 1
  fi
}

check build_all.sh bash scripts/build_all.sh
git ls-files "${phase0_dirs[@]}" | { grep '\.test\.mjs$' || true; } | LC_ALL=C sort >"$tmp/files"
if [ ! -s "$tmp/files" ]; then
  echo "FAIL acceptance tests: none found under ${phase0_dirs[*]}"
  exit 1
fi
while IFS= read -r file; do
  check "$file" node --test "$file"
done <"$tmp/files"

cd "$root"
if [ "$(git rev-parse HEAD)" != "$validated" ]; then
  echo "run_gates.sh: HEAD moved during the gate (was $validated); not marking it passed" >&2
  exit 1
fi
git commit --quiet --allow-empty -m "Gate: Phase $phase passed" -m "Validated $validated."
echo "Gate: Phase $phase passed"
