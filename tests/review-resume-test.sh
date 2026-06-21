#!/usr/bin/env bash
# tests/review-resume-test.sh — reconstruct-on-open (#66 M4a' resume): opening a
# review pane on a doc whose `review/<slug>` branch already carries an agent round
# repaints the decorations from the commit body. Text survives across sessions via
# nvim's undofile; the styling (change-highlights + diagnosis) is rebuilt from the
# records-in-commit (the M0 decision). Headless: drive M.reconstruct_on_open + assert
# the HL extmarks + diagnostics are placed.
#
# Run: bash tests/review-resume-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="${TMPDIR:-/tmp}/pair-resume-test.$$"; mkdir -p "$RT"
trap 'rm -rf "$RT"' EXIT
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# A repo on a review/<slug> branch with one agent round (body = records).
REPO="$RT/repo"; mkdir -p "$REPO"
( cd "$REPO"
  git init -q -b main; git config user.email t@e.com; git config user.name T
  printf 'hello\nthe value here\nworld\n' > doc.md
  git add doc.md; git commit -q -m init
  git checkout -q -b review/doc )

# Build the agent-round commit MESSAGE via the one encoder (record.embed_in_body) —
# write to a file (the body has ``` fences; -F avoids heredoc backtick surprises).
cat > "$RT/mkmsg.lua" <<'LUA'
local rec = dofile(os.getenv('ROOT') .. '/nvim/review/record.lua')
local records = { { old = 'the value', occurrence = 1, new = 'the value',
  new_occurrence = 1, explain = 'kept — the example why for resume' } }
local body = rec.embed_in_body('1 edit', records)
local f = io.open(os.getenv('MSG'), 'w')
f:write('review(doc): agent r1 — one edit\n\n' .. body .. '\n'); f:close()
LUA
ROOT="$ROOT" MSG="$RT/msg.txt" nvim -l "$RT/mkmsg.lua"
( cd "$REPO" && git commit -q --allow-empty -F "$RT/msg.txt" )

# Open the doc fresh (simulating a new session) and reconstruct-on-open.
cat > "$RT/driver.lua" <<'LUA'
local R = dofile(os.getenv('ROOT') .. '/nvim/review/init.lua')
local buf = vim.api.nvim_get_current_buf()
local placed = R.reconstruct_on_open(buf, vim.api.nvim_buf_get_name(buf))
local hl = vim.api.nvim_buf_get_extmarks(buf, vim.api.nvim_create_namespace('review'), 0, -1, {})
local diag = vim.diagnostic.get(buf, {})
local OUT = io.open(os.getenv('RESULT'), 'w')
OUT:write(string.format('placed=%s hl=%d diag=%d\n', tostring(placed), #hl, #diag))
OUT:close(); vim.cmd('qa!')
LUA
( cd "$REPO" && ROOT="$ROOT" RESULT="$RT/r" \
    run_headless --timeout 30 -- nvim --headless -u NONE "$REPO/doc.md" -c "luafile $RT/driver.lua" )

res="$(cat "$RT/r" 2>/dev/null || true)"
case "$res" in *placed=true*) pass "reconstruct-on-open placed decorations ($res)";; *) fail "no reconstruct ($res)";; esac
case "$res" in *hl=0*|'') fail "no change-highlights on resume ($res)";; *) pass "change-highlights repainted from commit";; esac
case "$res" in *diag=0*|'') fail "no diagnostics on resume ($res)";; *) pass "diagnosis repainted from commit";; esac

[ "$fails" -eq 0 ] || { printf 'review-resume-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-resume-test ok\n'
