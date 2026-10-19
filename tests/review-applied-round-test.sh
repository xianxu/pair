#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"
# tests/review-applied-round-test.sh — the review pane on `pair review apply`
# (pair#661 M2), hermetic and end to end. The agent applies through the shared
# engine (a separate `pair review apply` process); the pane only notices the
# round artifact, reloads the file and decorates the spans, and writes nothing
# for the agent round. The operator lock refuses an apply during their turn.
# The commit body pair returns reconstructs the decorations, as resume does.
#
# Run: bash tests/review-applied-round-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-applied-round-test.XXXXXX")" || exit 1
trap 'rm -rf "${RT:?}"' EXIT
RESULT="$RT/result.txt"
REPO="$RT/repo"
bash "$ROOT/tests/lib/pair-test-binary.sh" "$RT/pair"

mkdir -p "$REPO" "$RT/data"
( cd "$REPO"
  git init -q
  git config user.email t@example.com; git config user.name Tester
  printf 'alpha\nThe quick fox.\nomega\n' > doc.md
  git add doc.md; git commit -q -m init )

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local review = dofile(ROOT .. '/nvim/review/init.lua')
local record = dofile(ROOT .. '/nvim/review/record.lua')
local reconstruct = dofile(ROOT .. '/nvim/review/reconstruct.lua')
local oplock = dofile(ROOT .. '/nvim/review/oplock.lua')
local PAIR, ROUND = os.getenv('PAIR_BIN'), os.getenv('PAIR_REVIEW_ROUND_PATH')
local OUT = io.open(os.getenv('RESULT'), 'w')
local fails = 0
local function ok(c, m)
  if c then OUT:write('ok ' .. m .. '\n') else OUT:write('FAIL ' .. m .. '\n'); fails = fails + 1 end
end
local function content(b) return table.concat(vim.api.nvim_buf_get_lines(b, 0, -1, false), '\n') .. '\n' end
local function disk(p) local f = io.open(p, 'rb'); local s = f:read('*a'); f:close(); return s end
local function apply(file, records)
  local edits = os.getenv('RT') .. '/edits.json'
  local f = io.open(edits, 'w'); f:write(vim.json.encode(records)); f:close()
  local hash = vim.fn.sha256(disk(file)):sub(1, 16)
  local out = vim.fn.system({ PAIR, 'review', 'apply', edits, file, '--if-hash', hash, '--json' })
  return vim.v.shell_error, vim.json.decode(out)
end

vim.o.undodir = os.getenv('UNDODIR'); vim.fn.mkdir(vim.o.undodir, 'p')
vim.cmd('edit doc.md')
local buf = vim.api.nvim_get_current_buf()
local file = vim.fn.fnamemodify('doc.md', ':p')
review.start({ buf = buf, file = file, tag = 'doc', watch_opts = { interval = 20 },
  round_watch_opts = { interval = 20, path = ROUND } })

-- The agent applies (a separate process); the pane reloads and decorates.
local code, reply = apply(file, { { old = 'quick', occurrence = 1, new = 'slow', explain = 'pace' } })
ok(code == 0 and reply.Code == 'ok', 'pair review apply exit 0, Code ok')
vim.wait(3000, function() return content(buf):find('slow', 1, true) ~= nil end, 20)
ok(content(buf) == disk(file), 'the pane reloaded the file pair wrote')
ok(not vim.bo[buf].modified, 'the pane did not modify the buffer for the agent round')
local diags = vim.diagnostic.get(buf)
local saw = false
for _, d in ipairs(diags) do if d.message:find('pace', 1, true) then saw = true end end
ok(saw, "the round's explain is the diagnostic")

-- The commit body pair returned reconstructs the decorations (resume path).
local recs = record.extract_from_body(reply.Body)
ok(recs and #recs == 1 and recs[1].new == 'slow', 'the commit body carries the round as records')
ok(recs and #reconstruct.decorate(recs, disk(file), 'new').highlights == 1, 'decorations reconstruct from the body')

-- The operator's turn: their lock refuses the apply, nothing changes.
local before = disk(file)
ok(oplock.take(file), 'the operator lock is taken')
code, reply = apply(file, { { old = 'omega', occurrence = 1, new = 'OMEGA' } })
ok(code == 3 and reply.Code == 'operator-lock', 'apply under the lock exits 3 operator-lock')
ok(disk(file) == before, 'nothing written under the lock')
oplock.drop(file)
code, reply = apply(file, { { old = 'omega', occurrence = 1, new = 'OMEGA' } })
ok(code == 0, 'after the send the apply lands')
vim.wait(3000, function() return content(buf):find('OMEGA', 1, true) ~= nil end, 20)
ok(content(buf) == disk(file), 'the pane follows the next round')

-- A bad batch changes nothing and names the record.
before = disk(file)
code, reply = apply(file, { { old = 'absent', occurrence = 1, new = 'x' } })
ok(code == 1 and reply.Code == 'edits-invalid' and reply.Invalid and reply.Invalid[1].index == 1, 'bad batch: exit 1 edits-invalid, record listed')
ok(disk(file) == before, 'a refused batch writes nothing')

review.stop(buf)
OUT:write(fails == 0 and 'applied_round_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

( cd "$REPO"
  PAIR_ROOT="$ROOT" RESULT="$RESULT" UNDODIR="$RT/undo" RT="$RT" PAIR_BIN="$RT/pair" \
    PAIR_DATA_DIR="$RT/data" PAIR_TAG=doc PAIR_REVIEW_ROUND_PATH="$RT/data/review-round-doc.json" \
    run_headless --timeout 60 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!' )

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'applied_round_test ok' "$RESULT"; then
  echo "review-applied-round-test FAILED"; exit 1
fi
echo "review-applied-round-test ok"
