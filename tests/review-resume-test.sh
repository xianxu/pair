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
PAIR_BIN="${PAIR_BIN:-$ROOT/bin/pair}"
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

# Establish the document with a current-slug human round. The legacy fixture
# above deliberately had an empty agent commit and remains supported explicitly.
( cd "$REPO"
  printf 'hello\nthe value here\nworld\nscoped round\n' > doc.md
  git commit -qam 'review(doc): human r2'
  git checkout -qb review/other
  cp doc.md other.md
  git add other.md
)
# The unrelated round has identical anchors AND a body that contains the exact
# current branch marker. A broad git --grep fallback would repaint the wrong why.
cat > "$RT/unrelated.lua" <<'LUA'
local rec=dofile(os.getenv('ROOT')..'/nvim/review/record.lua')
local body=rec.embed_in_body('unrelated',{ {old='the value',occurrence=1,new='the value',new_occurrence=1,
  explain='UNRELATED review(doc): agent r99 must never decorate doc.md'} })
local f=assert(io.open(os.getenv('MSG'),'w'))
f:write('review(other): agent r99\n\n'..body..'\n');f:close()
LUA
ROOT="$ROOT" MSG="$RT/unrelated.txt" nvim -l "$RT/unrelated.lua"
( cd "$REPO"
  git commit -q -F "$RT/unrelated.txt"
  git checkout -q review/doc
  git merge --ff-only -q review/other
)
"$PAIR_BIN" review readiness --resolve "$REPO" > "$RT/identity.json"
cat > "$RT/scoped.lua" <<'LUA'
local ok,err=xpcall(function()
local R=dofile(os.getenv('ROOT')..'/nvim/review/init.lua')
local f=assert(io.open(os.getenv('IDENTITY'),'r'))
local identity=vim.json.decode(f:read('*a'));f:close()
assert(identity.status==os.getenv('EXPECTED_STATUS'),vim.inspect(identity))
local buf=vim.api.nvim_get_current_buf()
local placed=false
if identity.status=='resolved' then
  assert(identity.file=='doc.md',vim.inspect(identity))
  assert(identity.agent_round==1,vim.inspect(identity))
  -- The scoped path must not consult a previous command's global exit status.
  vim.fn.system({'false'})
  assert(vim.v.shell_error~=0)
  placed=R.reconstruct_on_open(buf,vim.api.nvim_buf_get_name(buf),identity)
end
local hl=vim.api.nvim_buf_get_extmarks(buf,vim.api.nvim_create_namespace('review'),0,-1,{})
local diag=vim.diagnostic.get(buf,{})
if identity.status=='resolved' then
  assert(placed and #hl>0 and #diag>0,'resolved identity did not restore decorations')
  local correct=false
  for _,d in ipairs(diag) do
    assert(not d.message:find('UNRELATED',1,true),'unrelated branch decorated selected document')
    if d.message:find('example why for resume',1,true) then correct=true end
  end
  assert(correct,'selected branch explanation was not restored')
else
  assert(not placed and #hl==0 and #diag==0,'unresolved history invoked reconstruction')
end
end,debug.traceback)
if not ok then io.stderr:write(err..'\n');vim.cmd('cquit 1') end
vim.cmd('qa!')
LUA
if ( cd "$REPO" && ROOT="$ROOT" IDENTITY="$RT/identity.json" EXPECTED_STATUS=resolved \
  run_headless --timeout 10 -- nvim --headless -u NONE "$REPO/doc.md" -c "luafile $RT/scoped.lua" ); then
  pass "resolved history repaints selected branch despite inherited identical anchors and stale shell status"
else
  fail "scoped reconstruction failed"
fi

# Adding a second current-slug document is ambiguous. The caller must refuse to
# reconstruct rather than feeding partial history or guessing the current file.
( cd "$REPO"
  printf 'ambiguous\n' >> other.md
  git commit -qam 'review(doc): human r3'
)
head_before="$(git -C "$REPO" rev-parse HEAD)"
"$PAIR_BIN" review readiness --resolve "$REPO" > "$RT/ambiguous.json"
if ( cd "$REPO" && ROOT="$ROOT" IDENTITY="$RT/ambiguous.json" EXPECTED_STATUS=ambiguous \
  run_headless --timeout 10 -- nvim --headless -u NONE "$REPO/doc.md" -c "luafile $RT/scoped.lua" ); then
  pass "ambiguous branch refuses reconstruction without decorations"
else
  fail "ambiguous reconstruction admission failed"
fi
[ "$(git -C "$REPO" rev-parse HEAD)" = "$head_before" ] && [ -z "$(git -C "$REPO" status --porcelain)" ] \
  && pass "resolution and reconstruction leave checkout unchanged" || fail "read-only reconstruction mutated checkout"

[ "$fails" -eq 0 ] || { printf 'review-resume-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-resume-test ok\n'
