#!/usr/bin/env bash
# tests/review-readiness-cli-test.sh — pair-review-readiness JSON shell seam.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-readiness-cli-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

REPO="$RT/repo"; mkdir -p "$REPO"
( cd "$REPO"
  git init -q
  git config user.email t@e.com
  git config user.name T
  printf 'doc\n' > 'doc "quoted".md'
  git add 'doc "quoted".md'
  git commit -q -m init
  git checkout -q -b 'review/a"b'
)

out="$(PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-readiness" "$REPO/doc \"quoted\".md")"
if printf '%s\n' "$out" | jq -e '.case and .branch and .scoped_file' >/dev/null; then
  pass "emits valid JSON with quoted branch/path fields"
else
  fail "invalid JSON: $out"
fi

[ "$fails" -eq 0 ] || { printf 'review-readiness-cli-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-readiness-cli-test ok\n'
