#!/usr/bin/env bash
# tests/review-indicator-test.sh — the review-mode indicator (#66 M3): the draft's
# line-1 header in review mode, `=== review: <file>  (N agent · M human) ===`.
# Counts are scoped to the CURRENT review session — the `review/<slug>` branch's
# own rounds — NOT the repo's whole history (which can hold dozens of shipped
# reviews of OTHER docs; counting those gave the spurious "25 agent · 28 human").
#   • off any review branch (M3 render-only)         → 0/0, even with review
#     commits for other slugs in history.
#   • on `review/<slug>`                             → that slug's rounds only.
# The line-1 set/restore wiring is live smoke (M3 plan Task 5).
#
# Run: bash tests/review-indicator-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-indicator-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# main: an init + two shipped rounds of ANOTHER doc (noise that must NOT count).
# review/binary-skill (off main): this session's 2 agent + 1 human rounds.
REPO="$RT/repo"; mkdir -p "$REPO"
( cd "$REPO"
  git init -q -b main; git config user.email t@e.com; git config user.name T
  printf 'post\n' > binary-skill-and-dynamic-skill.md
  git add -A; git commit -q -m "init: not a round"
  git commit -q --allow-empty -m "review(a-blogging-workflow): agent r1 — copy-edit"
  git commit -q --allow-empty -m "review(a-blogging-workflow): human r1 — tighten"
  git commit -q --allow-empty -m "chore: also not a round"
  git checkout -q -b review/binary-skill
  git commit -q --allow-empty -m "review(binary-skill): agent r1 — first pass"
  git commit -q --allow-empty -m "review(binary-skill): human r1 — edits"
  git commit -q --allow-empty -m "review(binary-skill): agent r2 — second pass"
  git checkout -q main )

printf 'generate\n' > "$RT/review-test.mode"
cat > "$RT/driver.lua" <<'LUA'
local OUT = io.open(os.getenv('RESULT'), 'w')
local f = _G._pair_review_bar
local DIR, DOC = os.getenv('DIR'), os.getenv('DOC')
OUT:write((type(f) == 'function') and 'fn ok\n' or 'fn MISSING\n')
if type(f) == 'function' then
  vim.fn.system({ 'git', '-C', DIR, 'checkout', '-q', 'main' })
  OUT:write('main: ' .. f(DOC) .. '\n')
  vim.fn.system({ 'git', '-C', DIR, 'checkout', '-q', 'review/binary-skill' })
  OUT:write('review: ' .. f(DOC) .. '\n')
end
OUT:close()
vim.cmd('qa!')
LUA

PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
  DIR="$REPO" DOC="$REPO/binary-skill-and-dynamic-skill.md" RESULT="$RT/r" \
  run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/init.lua" \
    -c "luafile $RT/driver.lua"

grep -q 'fn ok' "$RT/r" && pass "_pair_review_bar exposed" || fail "bar fn not exposed"
# THE regression for the "25/28" bug: off a review branch → 🤖0/0, even though
# main's history carries other docs' shipped review rounds.
grep -qF 'main: 🪄 Generate • binary-skill-and-dynamic-skill.md • 🤖0/0' "$RT/r" \
  && pass "off a review branch → 🤖0/0 (ignores other docs' shipped rounds)" \
  || { fail "repo-wide over-count not fixed"; grep '^main:' "$RT/r" || true; }
grep -qF 'review: 🪄 Generate • binary-skill-and-dynamic-skill.md • 🤖2/1' "$RT/r" \
  && pass "on review/<slug> → that slug's rounds only (🤖agent/human)" \
  || { fail "wrong on-branch count"; grep '^review:' "$RT/r" || true; }

[ "$fails" -eq 0 ] || { printf 'review-indicator-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-indicator-test ok\n'
