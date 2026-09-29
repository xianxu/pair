#!/usr/bin/env bash
# tests/review-readiness-cli-test.sh — pair review readiness JSON shell seam.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-readiness-cli-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
PAIR_BIN="${PAIR_BIN:-$ROOT/bin/pair}"
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

out="$(PAIR_HOME="$ROOT" "$PAIR_BIN" review readiness "$REPO/doc \"quoted\".md")"
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
  "$PAIR_BIN" review readiness --prepare "$PREP/doc.md" 2>&1 || true)"
prep_branch="$(cd "$PREP" && git branch --show-current)"
prep_abs="$(cd "$PREP" && pwd -P)/doc.md"
target=""
[ -f "$RT/review-target-prep.json" ] && target="$(jq -r '.status + " " + .file + " " + .session' "$RT/review-target-prep.json")"
[ "$prep_branch" = "review/doc" ] && pass "prepare creates review branch for clean tracked file" || fail "prepare branch: $prep_branch"
case "$target" in "ready $prep_abs sid") pass "prepare marks review target ready";; *) fail "prepare target: $target";; esac
case "$prep_out" in *"review prepared:"*"review/doc"*"Do not load xx-fix for this ack"*"load the full xx-fix skill"*"Reply \"ready\"."*) pass "prepare emits xx-fix deferred-load ack instruction";; *) fail "prepare output: $prep_out";; esac

receipt="$(jq -r '.identity.head' "$RT/review-target-prep.json")"
resolved="$("$PAIR_BIN" review readiness --resolve "$PREP")"
if printf '%s' "$resolved" | jq -e '.status == "missing" and .branch == "review/doc"' >/dev/null; then pass "zero-round branch has no invented identity"; else fail "unexpected identity: $resolved"; fi
resolved="$("$PAIR_BIN" review readiness --resolve "$PREP" --selected doc.md --head "$receipt")"
if printf '%s' "$resolved" | jq -e '.status == "resolved" and .file == "doc.md"' >/dev/null; then pass "verified preparation receipt enables first open"; else fail "receipt failed: $resolved"; fi
( cd "$PREP"
  printf 'round\n' > doc.md
  git commit -qam 'review(doc): human r1'
  git commit --allow-empty -qm $'review(doc): agent r2\n\n[]'
)
resolved="$("$PAIR_BIN" review readiness --resolve "$PREP")"
if printf '%s' "$resolved" | jq -e '.status == "resolved" and .file == "doc.md" and .agent_round == 2 and (.latest_agent_body | startswith("[]"))' >/dev/null; then pass "current branch rounds resolve document and agent body"; else fail "history failed: $resolved"; fi

[ "$fails" -eq 0 ] || { printf 'review-readiness-cli-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-readiness-cli-test ok\n'
