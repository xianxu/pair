#!/usr/bin/env bash
# Regression test for the normal-mode position marker in the draft statusline
# (nvim/init.lua, _G.PairStatusline). Drives the REAL init.lua headlessly —
# there is no way to unit-test it (monolithic config, all-local functions) — so
# we boot nvim on a history+queue fixture, fire the actual Alt+←/→ nav keymaps,
# and assert the normal-mode bar carries the *, -N, +N marker.
#
# Guards the bug where leaving insert mode dropped the whole position cluster,
# leaving the user with no "you are here" cue (is this *, -N, or +N?) while
# navigating in normal mode. The marker rides the PairPosLabel highlight, same
# as the insert-mode cluster.
#
# Run: bash tests/statusline-pos-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/nvim/init.lua"
. "$ROOT/tests/lib/run-headless.sh"   # run_headless: timeout watchdog (#60)
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-statusline-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

# Fixture for tag=test: a draft (*), two history entries (oldest→newest, so
# pos.n=1 is "second entry"), and one queue item (+1).
mkdir -p "$RT/queue-test"
printf 'my draft text' > "$RT/draft-test.md"
printf '## 2026-01-01 00:00\n\nfirst entry\n\n---\n\n## 2026-01-02 00:00\n\nsecond entry' > "$RT/log-test.md"
printf 'queued item' > "$RT/queue-test/500000.md"

# Driver: navigate via the real keymaps, sampling _G.PairStatusline() (which
# returns its normal-mode branch since headless boots in normal mode) at each
# stop. We assert the highlighted marker substring is present.
cat > "$RT/driver.lua" <<'LUA'
local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
local ok, err = pcall(function()
  local function nav(key)
    local m = vim.fn.maparg(key, 'n', false, true)
    if type(m) == 'table' and m.callback then pcall(m.callback) end
  end
  local function check(label, want)
    local s = _G.PairStatusline()
    local hit = s:find(want, 1, true) ~= nil and s:find('<LOCKED>', 1, true) ~= nil
    O:write(string.format('%s\t%s\n', hit and 'ok' or 'FAIL', label))
  end
  check('star',  '%#PairPosLabel#*%*')            -- at the draft
  nav('<M-Left>');  check('hist-1', '%#PairPosLabel#-1%*')
  nav('<M-Left>');  check('hist-2', '%#PairPosLabel#-2%*')
  nav('<M-Right>'); nav('<M-Right>')              -- -2 -> -1 -> *
  nav('<M-Right>'); check('queue-1', '%#PairPosLabel#+1%*')
end)
if not ok then O:write('FAIL\tdriver-error: ' .. tostring(err) .. '\n') end
O:close()
vim.cmd('qall!')  -- force: headless boot may dirty the buffer; bare qall → E37 → hang (#60)
LUA

run_headless --timeout 30 -- \
  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
  nvim --headless -u "$INIT" "$RT/draft-test.md" \
  -c "luafile $RT/driver.lua"

echo "statusline-pos-test:"
if [ ! -f "$RT/result.txt" ]; then
  echo "  FAIL driver produced no result (nvim boot/driver error)"
  exit 1
fi
fails=0
while IFS=$'\t' read -r status label; do
  case "$status" in
    ok)   printf '  ok   %s shows marker in normal mode\n' "$label" ;;
    FAIL) printf '  FAIL %s\n' "$label"; fails=$((fails + 1)) ;;
  esac
done < "$RT/result.txt"

if [ "$fails" -ne 0 ]; then
  echo "statusline-pos-test: $fails failure(s)"
  exit 1
fi
echo "statusline-pos-test: all passed"
