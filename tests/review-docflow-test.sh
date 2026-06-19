#!/usr/bin/env bash
# tests/review-docflow-test.sh — review.docflow forwards verbs/flags to docflow,
# and docflow produces real commits with the right shape. Hermetic: DOCFLOW_BIN
# points at tests/lib/fake-docflow.sh (no ariadne checkout needed). A gated smoke
# test runs the REAL ariadne docflow when present (catches drift — finding #3).
#
# Run: bash tests/review-docflow-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-docflow-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
ARGLOG="$RT/arglog.txt"
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# Drive review.docflow (start → human round → agent round w/ records body) in a
# fresh temp git repo, using the docflow binary in $1.
run_case() {
  local REPO="$RT/repo"; rm -rf "$REPO"; mkdir -p "$REPO"
  ( cd "$REPO"
    git init -q
    git config user.email t@example.com; git config user.name Tester
    printf 'hello\nx here\n' > doc.md
    git add doc.md; git commit -q -m init )
  : > "$ARGLOG"
  cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local docflow = dofile(ROOT .. '/nvim/review/docflow.lua')
local record  = dofile(ROOT .. '/nvim/review/record.lua')
docflow.start('doc.md')
-- an out-of-scope file present during the rounds: must NOT be staged (the real
-- docflow stages in-scope-only, never -A).
local o = io.open('other.md', 'w'); o:write('out of scope\n'); o:close()
local h = io.open('doc.md', 'a'); h:write('\nhuman\n'); h:close()
docflow.round('human', 'incoming')
local a = io.open('doc.md', 'a'); a:write('\nagent\n'); a:close()
local recs = { { old = 'x', occurrence = 1, new = 'y', new_occurrence = 1, explain = 'because' } }
docflow.round('agent', 'two edits', record.embed_in_body('two edits', recs))
LUA
  ( cd "$REPO"
    PAIR_ROOT="$ROOT" DOCFLOW_BIN="$1" DOCFLOW_ARGLOG="$ARGLOG" \
      run_headless --timeout 30 -- nvim -l "$RT/driver.lua" )
}

# ── hermetic case (fake docflow) ──────────────────────────────────────────────
run_case "$ROOT/tests/lib/fake-docflow.sh"
REPO="$RT/repo"

grep -q 'round --side agent' "$ARGLOG" && pass "forwards 'round --side agent'" || fail "agent round not forwarded"
grep -q -- '--body' "$ARGLOG"          && pass "forwards --body"               || fail "--body not forwarded"

subj="$(cd "$REPO" && git log --format='%s' --grep='agent r1' -1)"
[ "$subj" = "review(doc): agent r1 — two edits" ] && pass "agent subject shape" || fail "subject: $subj"
ae="$(cd "$REPO" && git log --format='%ae' --grep='agent r1' -1)"
[ "$ae" = "noreply@anthropic.com" ] && pass "agent round authored by agent" || fail "agent author: $ae"
body="$(cd "$REPO" && git log --format='%b' --grep='agent r1' -1)"
case "$body" in *'```review-records'*) pass "records block in agent commit body";; *) fail "no records block in body";; esac
he="$(cd "$REPO" && git log --format='%ae' --grep='human r1' -1)"
[ "$he" = "t@example.com" ] && pass "human round authored by operator" || fail "human author: $he"

# scoping: the agent round must stage the in-scope doc only, not other.md
files="$(cd "$REPO" && git log -1 --name-only --format= --grep='agent r1')"
case "$files" in *doc.md*) pass "agent round staged the in-scope doc";; *) fail "doc.md missing from agent commit";; esac
case "$files" in *other.md*) fail "out-of-scope other.md leaked into the round (staging not in-scope-only)";; *) pass "out-of-scope file NOT staged";; esac

# ── gated smoke test against the REAL ariadne docflow ─────────────────────────
REAL="${DOCFLOW_BIN_REAL:-$ROOT/../ariadne/scripts/docflow.sh}"
if [ -x "$REAL" ]; then
  run_case "$REAL"
  subj="$(cd "$RT/repo" && git log --format='%s' --grep='agent r1' -1)"
  [ "$subj" = "review(doc): agent r1 — two edits" ] && pass "REAL docflow subject shape" || fail "REAL subject: $subj"
else
  printf '  skip real-docflow smoke (ariadne not found at %s)\n' "$REAL"
fi

[ "$fails" -eq 0 ] || { printf 'FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-docflow-test ok\n'
