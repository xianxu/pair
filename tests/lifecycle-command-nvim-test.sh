#!/usr/bin/env bash
# tests/lifecycle-command-nvim-test.sh — the four confirmed lifecycle keybinds
# run their command through the shared reporting helper (#284).
#
# nvim/lifecycle_command_test.lua proves the helper notifies on a non-zero exit.
# This proves the WIRING that makes that reach an operator: each of Alt+x quit,
# Alt+d detach, Alt+n restart and Shift+Alt+N agent restart goes through
# `_G._pair_lifecycle.run`, and none of them calls `vim.fn.system` directly.
# Without this, reverting any one call site to a bare `vim.fn.system` leaves the
# suite green while restoring exactly the silent failure #284 was filed for.
#
# Run: bash tests/lifecycle-command-nvim-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/nvim/init.lua"
. "$ROOT/tests/lib/run-headless.sh"   # run_headless: timeout watchdog (#60)

RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-lifecycle-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
printf '' > "$RT/draft.md"

# Driver: confirm always answers Yes, the helper records the argv it was handed,
# and vim.fn.system records any call that bypassed the helper. Detach is gated
# on has_ui(), whose existing test seam is vim.g.pair_test_has_ui.
cat > "$RT/driver.lua" <<'LUA'
local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
vim.g.pair_test_has_ui = true

local ran, bypassed = {}, {}
_G._pair_lifecycle.run = function(argv)
  table.insert(ran, table.concat(argv, ' '))
  return true
end
vim.fn.confirm = function() return 1 end        -- Yes
vim.fn.system = function(argv)
  table.insert(bypassed, type(argv) == 'table' and table.concat(argv, ' ') or tostring(argv))
  return ''
end

local entry = {
  { 'Alt+x quit', _G.PairConfirmQuit, 'pair quit' },
  { 'Alt+d detach', _G.PairConfirmDetach, 'zellij action detach' },
  { 'Alt+n restart', _G.PairConfirmRestart, 'pair restart' },
  { 'Shift+Alt+N agent restart', _G.PairConfirmAgentRestart, 'pair agent restart' },
}
for _, e in ipairs(entry) do
  local label, fn, want = e[1], e[2], e[3]
  if type(fn) ~= 'function' then
    O:write(string.format('FAIL\t%s\t%s\t%s\n', label, 'not defined', want))
  else
    ran = {}
    local ok, err = pcall(fn)
    -- Never write an empty column: tab is IFS-whitespace, so the reader
    -- collapses consecutive tabs and a failure would misreport got/want.
    local got = ok and (#ran > 0 and table.concat(ran, ',') or '(nothing)')
      or ('error: ' .. tostring(err))
    O:write(string.format('%s\t%s\t%s\t%s\n', got == want and 'ok' or 'FAIL', label, got, want))
  end
end

-- The other half of the contract: nothing reached vim.fn.system behind the
-- helper's back.
O:write(string.format('%s\t%s\t%s\t%s\n',
  #bypassed == 0 and 'ok' or 'FAIL', 'no direct vim.fn.system call',
  #bypassed == 0 and 'none' or table.concat(bypassed, ','), 'none'))
O:close()
vim.cmd('qall!')
LUA

run_headless --timeout 30 -- \
  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
  PAIR_DRAFT_PATH="$RT/draft.md" PAIR_LAYOUT_MODE_PATH="$RT/layout-mode-test" \
  nvim --headless -u "$INIT" "$RT/draft.md" \
  -c "luafile $RT/driver.lua"

echo "lifecycle-command-nvim-test:"
if [ ! -f "$RT/result.txt" ]; then
  echo "  FAIL driver produced no result (nvim boot/driver error)"
  exit 1
fi
fails=0
while IFS=$'\t' read -r status label got want; do
  case "$status" in
    ok)   printf '  ok   %s\n' "$label" ;;
    FAIL) printf '  FAIL %s: got %s want %s\n' "$label" "$got" "$want"; fails=$((fails + 1)) ;;
  esac
done < "$RT/result.txt"
if [ "$fails" -ne 0 ]; then
  echo "  $fails failure(s)"
  exit 1
fi
echo "  all passed"
