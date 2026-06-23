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

PREP="$RT/prep"; mkdir -p "$PREP"
( cd "$PREP"
  git init -q
  git config user.email t@e.com
  git config user.name T
  printf 'doc\n' > doc.md
  git add doc.md
  git commit -q -m init
)
prep_out="$(PAIR_HOME="$ROOT" PAIR_DATA_DIR="$RT" PAIR_TAG=prep PAIR_SESSION_ID=sid \
  "$ROOT/bin/pair-review-readiness" --prepare "$PREP/doc.md" 2>&1 || true)"
prep_branch="$(cd "$PREP" && git branch --show-current)"
prep_abs="$(cd "$PREP" && pwd -P)/doc.md"
target=""
[ -f "$RT/review-target-prep.json" ] && target="$(jq -r '.status + " " + .file + " " + .session' "$RT/review-target-prep.json")"
[ "$prep_branch" = "review/doc" ] && pass "prepare creates review branch for clean tracked file" || fail "prepare branch: $prep_branch"
case "$target" in "ready $prep_abs sid") pass "prepare marks review target ready";; *) fail "prepare target: $target";; esac
case "$prep_out" in *"review prepared:"*"review/doc"*"Do not load xx-fix for this ack"*"load the full xx-fix skill"*"Reply \"ready\"."*) pass "prepare emits xx-fix deferred-load ack instruction";; *) fail "prepare output: $prep_out";; esac

[ "$fails" -eq 0 ] || { printf 'review-readiness-cli-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-readiness-cli-test ok\n'
