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

  -- (2) marker poll: dropping the marker fires the flash + consumes it (≤2s).
  local dd = os.getenv('PAIR_DATA_DIR')
  local marker = dd .. '/changelog-test-claude.ready'
  local mf = assert(io.open(marker, 'w')); mf:write('2026-06-14T00:00:00Z\n'); mf:close()
  local fired = vim.wait(4000, function()
    return _G.PairStatusline():find('change log ready', 1, true) ~= nil
  end, 50)
  emit(fired, 'dropped marker fires the flash')
  emit(vim.loop.fs_stat(marker) == nil, 'marker consumed (deleted, one-shot)')
end)
if not ok then emit(false, 'driver-error: ' .. tostring(err)) end
O:close()
vim.cmd('qall')
LUA

PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
  nvim --headless -u "$INIT" "$RT/draft-test.md" \
  -c "luafile $RT/driver.lua" >/dev/null 2>&1 || true

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
