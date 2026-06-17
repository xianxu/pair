#!/usr/bin/env bash
# Regression test for the change-log "build ready" statusline notification (#58).
# Drives the REAL nvim/init.lua headlessly and checks two things:
#   (1) _G.PairFlashNotify renders a green PairNotify message at the right end
#       (replacing the Alt+h/Alt+⏎ cheatsheet) and reverts to the cheatsheet
#       after the ~2s flash timer.
#   (2) the marker-poll timer turns a dropped changelog-<tag>-<agent>.ready
#       marker (what the detached distiller writes on a real-change build) into
#       that flash, and consumes the marker (one-shot).
#
# This is the draft-side half of the build-complete signal: the operator
# triggers Alt+l, leaves to work, and the persistently-visible draft statusline
# flashes when the slow background build finishes.
#
# Run: bash tests/changelog-notify-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/nvim/init.lua"
. "$ROOT/tests/lib/run-headless.sh"   # run_headless: timeout watchdog (#60)
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-changelog-notify-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

printf 'draft text' > "$RT/draft-test.md"

cat > "$RT/driver.lua" <<'LUA'
local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
local function emit(ok, label) O:write(string.format('%s\t%s\n', ok and 'ok' or 'FAIL', label)) end
local ok, err = pcall(function()
  -- Pin a wide terminal so the cheatsheet always fits the right end (its budget
  -- is columns-relative); otherwise a narrow headless default could drop it and
  -- make the "replaces cheatsheet" / "reverts" checks ambiguous.
  vim.o.columns = 120

  -- Baseline: cheatsheet present, no notification.
  local base = _G.PairStatusline()
  emit(base:find('help', 1, true) ~= nil, 'baseline shows cheatsheet')
  emit(base:find('change log ready', 1, true) == nil, 'baseline has no notification')

  -- (1) Direct flash render: message + green PairNotify, cheatsheet gone.
  _G.PairFlashNotify('done change log ready Alt+l')
  local s = _G.PairStatusline()
  emit(s:find('change log ready', 1, true) ~= nil, 'flash shows the message')
  emit(s:find('PairNotify', 1, true) ~= nil, 'flash uses the PairNotify highlight')
  emit(s:find('help', 1, true) == nil, 'flash replaces the cheatsheet')

  -- Reverts to the cheatsheet after the ~2s timer.
  local reverted = vim.wait(3000, function()
    return _G.PairStatusline():find('help', 1, true) ~= nil
  end, 50)
  emit(reverted, 'reverts to the cheatsheet after the timer')

  -- (2) marker poll keyed on session id (#63). The watcher resolves the id
  -- (PAIR_SESSION_ID -> per-tag config -> none) on each tick and polls the
  -- matching changelog-<tag>-<agent>[-<sid>].ready. We drive all three branches
  -- in one boot by mutating vim.env, asserting KEYED-MARKER CONSUMPTION -- the
  -- unambiguous signal the watcher resolved that exact path (the flash text would
  -- overlap across phases on its ~2s revert, so consumption is the clean probe).
  local dd = os.getenv('PAIR_DATA_DIR')
  local function drop(name)
    local f = assert(io.open(dd .. '/' .. name, 'w')); f:write('2026-06-14T00:00:00Z\n'); f:close()
  end
  local function consumed(name)
    return vim.wait(4000, function() return vim.loop.fs_stat(dd .. '/' .. name) == nil end, 50)
  end

  -- (2a) no id -> legacy unsuffixed marker (backward compat + the flash render).
  vim.env.PAIR_SESSION_ID = nil
  drop('changelog-test-claude.ready')
  emit(consumed('changelog-test-claude.ready'), 'legacy (no-id) marker consumed')
  emit(_G.PairStatusline():find('change log ready', 1, true) ~= nil, 'legacy marker fires the flash')

  -- (2b) PAIR_SESSION_ID set -> keyed marker (the claude-fresh / resume path).
  local SID = 'deadbeef-0000-1111-2222-333344445555'
  vim.env.PAIR_SESSION_ID = SID
  drop('changelog-test-claude-' .. SID .. '.ready')
  emit(consumed('changelog-test-claude-' .. SID .. '.ready'), 'env-keyed marker consumed')

  -- (2c) env cleared + config written -> config-resolved keyed marker (the
  -- codex/agy async path: the id lands in the config mid-session, no nvim restart).
  vim.env.PAIR_SESSION_ID = nil
  local CSID = 'beadfeed-9999-8888-7777-666655554444'
  local cf = assert(io.open(dd .. '/config-test-claude.json', 'w'))
  cf:write('{"agent":"claude","args":[],"session_id":"' .. CSID .. '"}'); cf:close()
  drop('changelog-test-claude-' .. CSID .. '.ready')
  emit(consumed('changelog-test-claude-' .. CSID .. '.ready'), 'config-resolved keyed marker consumed')
end)
if not ok then emit(false, 'driver-error: ' .. tostring(err)) end
O:close()
vim.cmd('qall!')  -- force: headless boot may dirty the buffer; bare qall → E37 → hang (#60)
LUA

run_headless --timeout 45 -- \
  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
  nvim --headless -u "$INIT" "$RT/draft-test.md" \
  -c "luafile $RT/driver.lua"

echo "changelog-notify-test:"
if [ ! -f "$RT/result.txt" ]; then
  echo "  FAIL driver produced no result (nvim boot/driver error)"
  exit 1
fi
fails=0
while IFS=$'\t' read -r status label; do
  case "$status" in
    ok)   printf '  ok   %s\n' "$label" ;;
    FAIL) printf '  FAIL %s\n' "$label"; fails=$((fails + 1)) ;;
  esac
done < "$RT/result.txt"

if [ "$fails" -ne 0 ]; then
  echo "changelog-notify-test: $fails failure(s)"
  exit 1
fi
echo "changelog-notify-test: all passed"
