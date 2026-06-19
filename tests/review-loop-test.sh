#!/usr/bin/env bash
# tests/review-loop-test.sh — the M1 end-to-end vertical (issue #66), fake-agent
# driven, hermetic (fake-docflow + redirected XDG). Proves the whole contract:
# handoff → undo-able apply → docflow agent-round (records in body) → undo
# crosses the commit → decorations reconstruct from the commit body.
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
local OUT = io.open(os.getenv('RESULT'), 'w')
local fails = 0
local function ok(c, m)
  if c then OUT:write('ok ' .. m .. '\n') else OUT:write('FAIL ' .. m .. '\n'); fails = fails + 1 end
end
local function content(b) return table.concat(vim.api.nvim_buf_get_lines(b, 0, -1, false), '\n') end

vim.o.undodir = os.getenv('UNDODIR'); vim.fn.mkdir(vim.o.undodir, 'p')

vim.cmd('edit doc.md')
local buf = vim.api.nvim_get_current_buf()
review.start({ buf = buf, file = 'doc.md', tag = 'doc', watch_opts = { interval = 20 } })

-- human round: append a line, commit the incoming human edits
vim.api.nvim_buf_set_lines(buf, -1, -1, false, { 'human note' })
review.human_round(buf, 'incoming')

-- agent (separate process) writes the handoff; the watcher applies + commits
os.execute(ROOT .. '/tests/lib/fake-review-agent.sh doc')
local function committed()
  local r = vim.fn.system({ 'git', 'log', '--oneline', '--grep=agent r1' })
  return (r or '') ~= '' and not r:match('^%s*$')
end
vim.wait(5000, committed, 50)

local c = content(buf)
ok(c:match('FOO') and c:match('BAZ'), 'agent edits applied to the buffer')
ok(committed(), 'agent round committed')

local body = vim.fn.system({ 'git', 'log', '--format=%b', '--grep=agent r1', '-1' })
local recs = record.extract_from_body(body)
ok(recs and #recs == 2, 'records block embedded in the agent commit body')
if recs then
  ok(#reconstruct.decorate(recs, c, 'new').highlights == 2, 'decorations reconstruct from the commit')
end

-- undo crosses the agent commit: reverts FOO/BAZ, human round survives
vim.api.nvim_buf_call(buf, function() vim.cmd('silent undo') end)
local a = content(buf)
ok(a:match('foo') and not a:match('FOO'), 'one undo reverts the agent round')
ok(a:match('human note'), 'human round survives the agent-round undo')

-- (I3) a docflow failure must SURFACE (notify), not silently leave an
-- edited+saved buffer with no commit.
local notified = {}
local orig = vim.notify
vim.notify = function(msg) notified[#notified + 1] = tostring(msg) end
vim.env.DOCFLOW_BIN = ROOT .. '/tests/lib/fail-docflow.sh'
review.on_agent_round(buf, { { old = 'qux', occurrence = 1, new = 'QUX', explain = 'q' } })
vim.notify = orig
ok(#notified >= 1 and notified[1]:match('docflow'), 'docflow failure surfaces via notify (I3)')

review.stop(buf)
OUT:write(fails == 0 and 'loop_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

( cd "$REPO"
  PAIR_ROOT="$ROOT" RESULT="$RESULT" UNDODIR="$RT/undo" \
    XDG_DATA_HOME="$RT/xdg" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" \
    run_headless --timeout 40 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!' )

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'loop_test ok' "$RESULT"; then
  echo "review-loop-test FAILED"; exit 1
fi
echo "review-loop-test ok"
