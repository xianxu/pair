#!/usr/bin/env bash
# tests/review-loop-test.sh — the end-to-end vertical (issue #66), fake-agent
# driven, hermetic (fake-docflow + redirected XDG). M4a: the AGENT owns all git
# (branch + human + agent rounds). The nvim applies records undo-ably, saves,
# writes the landed-artifact (what landed), and pokes — it writes NO git
# (invariant #1). fake-agent-v2 commits the rounds, the agent round body taken
# VERBATIM from the landed-artifact (invariant #3). Proves: handoff → undo-able
# apply → landed-artifact → agent commits → undo crosses the commit → decorations
# + counts reconstruct from the AGENT's commit.
#
# Run: bash tests/review-loop-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-loop-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"
REPO="$RT/repo"

mkdir -p "$REPO"
( cd "$REPO"
  git init -q
  git config user.email t@example.com; git config user.name Tester
  printf 'alpha\nfoo bar\nbaz qux\n' > doc.md
  git add doc.md; git commit -q -m init )

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local review = dofile(ROOT .. '/nvim/review/init.lua')
local record = dofile(ROOT .. '/nvim/review/record.lua')
local reconstruct = dofile(ROOT .. '/nvim/review/reconstruct.lua')
local handoff = dofile(ROOT .. '/nvim/review/handoff.lua')
local OUT = io.open(os.getenv('RESULT'), 'w')
local AGENT_LOG = os.getenv('AGENT_LOG')
local fails = 0
local function ok(c, m)
  if c then OUT:write('ok ' .. m .. '\n') else OUT:write('FAIL ' .. m .. '\n'); fails = fails + 1 end
end
local function content(b) return table.concat(vim.api.nvim_buf_get_lines(b, 0, -1, false), '\n') end
local function grep(p) local r = vim.fn.system({ 'git', 'log', '--oneline', '--grep=' .. p }); return (r or '') ~= '' and not r:match('^%s*$') end

vim.o.undodir = os.getenv('UNDODIR'); vim.fn.mkdir(vim.o.undodir, 'p')

vim.cmd('edit doc.md')
local buf = vim.api.nvim_get_current_buf()
review.start({ buf = buf, file = 'doc.md', tag = 'doc', watch_opts = { interval = 20 } })

-- The nvim writes NO git: inject the poke seam to capture the commit signal
-- (and avoid shelling zellij headless).
local poked = {}
review.poke = { send = function(b) poked[#poked + 1] = b; return true end }

-- human round: append a line + SAVE (the nvim saves; the agent commits it).
vim.api.nvim_buf_set_lines(buf, -1, -1, false, { 'human note' })
review.human_round(buf, 'incoming')

-- the agent (separate process) owns all git: branch, human round, propose records,
-- then commit the agent round verbatim from the nvim's landed-artifact.
os.execute(ROOT .. '/tests/lib/fake-review-agent.sh doc doc.md >> ' .. AGENT_LOG .. ' 2>&1 &')
vim.wait(10000, function() return grep('agent r1') and grep('human r1') end, 50)

local c = content(buf)
ok(c:match('FOO') and c:match('BAZ'), 'agent edits applied to the buffer (by the nvim)')
ok(grep('human r1'), 'human round committed by the agent')
ok(grep('agent r1'), 'agent round committed by the agent')

-- the agent round body is the landed-artifact's body verbatim → records present,
-- and the indicator counts reconstruct from the AGENT's commit (Task 4).
local body = vim.fn.system({ 'git', 'log', '--format=%b', '--grep=agent r1', '-1' })
local recs = record.extract_from_body(body)
ok(recs and #recs == 2, 'records block in the agent commit body (verbatim from the landed-artifact)')
if recs then
  ok(#reconstruct.decorate(recs, c, 'new').highlights == 2, 'decorations reconstruct from the agent commit')
end
-- (Task 4 counts — _pair_review_bar's branch-scoped 🤖N/M from agent commits — is
-- covered by review-indicator-test; here we prove decorations reconstruct, above.)
-- the commit-signal poke fired with the agent_applied body (2 applied, 0 dropped)
local sig = false
for _, b in ipairs(poked) do if b:match('applied 2 edit%(s%) to doc.md') then sig = true end end
ok(sig, 'nvim poked the agent_applied commit signal (2 applied, 0 dropped)')

-- undo crosses the agent round: reverts FOO/BAZ, human round survives
vim.api.nvim_buf_call(buf, function() vim.cmd('silent undo') end)
local a = content(buf)
ok(a:match('foo') and not a:match('FOO'), 'one undo reverts the agent round')
ok(a:match('human note'), 'human round survives the agent-round undo')

-- (Task 1b, invariant #3) on_agent_round with a DROPPED record: one anchors
-- (qux→QUX), one cannot (NOTPRESENT). The nvim writes NO git; it writes the
-- landed-artifact (applied=1/dropped=1, body = the ONE landed record) + pokes
-- agent_applied(1,1) + WARNs the drop.
vim.api.nvim_buf_call(buf, function() vim.cmd('silent redo') end) -- back to FOO/BAZ state
local warns = {}
local on0 = vim.notify; vim.notify = function(msg) warns[#warns + 1] = tostring(msg) end
poked = {}
review.on_agent_round(buf, {
  { old = 'qux', occurrence = 1, new = 'QUX', explain = 'caps qux' },
  { old = 'NOTPRESENT', occurrence = 1, new = 'x', explain = 'no anchor' },
})
vim.notify = on0
ok(#warns >= 1 and warns[1]:match('anchor'), 'dropped proposal surfaced via WARN (invariant #3)')
local lf = io.open(handoff.landed_path('doc'), 'r')
local landed = lf and vim.json.decode(lf:read('*a')) or {}
if lf then lf:close() end
ok(landed.applied == 1 and landed.dropped == 1, 'landed-artifact records applied=1 dropped=1')
local lrecs = landed.body and record.extract_from_body(landed.body)
ok(lrecs and #lrecs == 1 and lrecs[1].new == 'QUX', 'landed body carries ONLY the record that landed')
local sig2 = false
for _, b in ipairs(poked) do if b:match('applied 1 edit%(s%) %(1 dropped%) to doc.md') then sig2 = true end end
ok(sig2, 'poke carries agent_applied(1, 1, file) for the dropped case')

-- (#89 M2) concurrent-edit reconcile through on_agent_round: snapshot v0, edit the
-- buffer concurrently so one record's span is gone → the clean record applies and
-- the overlapped one becomes a 🤖<…>[reconcile] marker; the landed-artifact reports
-- conflicts and its body embeds ONLY the clean record (Option A).
vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'keep this line', 'change this line' })
review.set_base(buf, content(buf))                                  -- v0 = what the agent reviews
vim.api.nvim_buf_set_lines(buf, 1, 2, false, { 'HUMAN rewrote it' }) -- concurrent edit on line 2
poked = {}
review.on_agent_round(buf, {
  { old = 'keep', occurrence = 1, new = 'KEEP', explain = 'clean edit' },            -- line 1 untouched → clean
  { old = 'change this line', occurrence = 1, new = 'CHANGE', explain = 'reword' },  -- line 2 gone → conflict
})
local rc = content(buf)
ok(rc:find('KEEP this line', 1, true) ~= nil, 'reconcile: clean record applied')
ok(rc:find('🤖<HUMAN rewrote it>', 1, true) ~= nil, 'reconcile: overlap became a marker on the human hunk')
ok(rc:find('reconcile', 1, true) ~= nil and rc:find('CHANGE', 1, true) ~= nil, 'reconcile: marker carries the agent intent')
local lf2 = io.open(handoff.landed_path('doc'), 'r')
local landed2 = lf2 and vim.json.decode(lf2:read('*a')) or {}
if lf2 then lf2:close() end
ok(landed2.applied == 1 and landed2.conflicts == 1, 'reconcile: landed reports applied=1 conflicts=1')
local lrecs2 = landed2.body and record.extract_from_body(landed2.body)
ok(lrecs2 and #lrecs2 == 1 and lrecs2[1].new == 'KEEP', 'reconcile: body embeds ONLY the clean record (Option A)')
local sig3 = false
for _, b in ipairs(poked) do if b:find('applied 1 edit(s) (1 to reconcile) to doc.md', 1, true) then sig3 = true end end
ok(sig3, 'reconcile: poke reports applied 1 (1 to reconcile)')

review.stop(buf)
OUT:write(fails == 0 and 'loop_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

( cd "$REPO"
  PAIR_ROOT="$ROOT" RESULT="$RESULT" UNDODIR="$RT/undo" AGENT_LOG="$RT/agent.log" \
    XDG_DATA_HOME="$RT/xdg" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" \
    run_headless --timeout 40 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!' )

echo "--- results ---"; cat "$RESULT"
[ -s "$RT/agent.log" ] && { echo "--- agent.log ---"; cat "$RT/agent.log"; }
if grep -q FAIL "$RESULT" || ! grep -q 'loop_test ok' "$RESULT"; then
  echo "review-loop-test FAILED"; exit 1
fi
echo "review-loop-test ok"
