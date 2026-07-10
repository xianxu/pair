#!/usr/bin/env bash
# Regression test for draft typeahead completion mode safety (nvim/init.lua).
#
# TextChangedI/P can debounce completion through a scheduled callback. If the
# user leaves Insert mode before that callback runs, the shared completion
# runner must no-op; otherwise word_complete/path_complete/spell_complete can
# call vim.fn.complete() from Normal/Visual mode and raise E785.
#
# Run: bash tests/draft-complete-mode-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/nvim/init.lua"
. "$ROOT/tests/lib/run-headless.sh"   # run_headless: timeout watchdog (#60)
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-draft-complete-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

printf '' > "$RT/draft.md"

cat > "$RT/driver.lua" <<'LUA'
local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
local fails = 0
local function check(cond, msg, detail)
  if cond then
    O:write('ok\t' .. msg .. '\n')
  else
    fails = fails + 1
    O:write('FAIL\t' .. msg .. '\t' .. tostring(detail or '') .. '\n')
  end
end

check(type(_G.PairDraftCompleteTest) == 'table', 'test seam exported')
check(type(_G.PairDraftCompleteTest.run_completers) == 'function', 'runner exported')

local function exit_to_normal()
  vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes('<Esc>', true, false, true), 'nx', false)
  vim.cmd('stopinsert')
end

local function prime_completion_candidate()
  exit_to_normal()
  -- Prefix "complet" has a same-buffer candidate "completion", so without the
  -- shared mode guard run_completers reaches word_complete() and vim.fn.complete().
  vim.api.nvim_buf_set_lines(0, 0, -1, false, { 'complet completion' })
  vim.api.nvim_win_set_cursor(0, { 1, #'complet' })
end

local function runner_skips(label, enter_mode)
  prime_completion_candidate()
  enter_mode()
  local mode = vim.api.nvim_get_mode().mode
  check(mode:sub(1, 1) ~= 'i', label .. ': driver is outside insert mode', mode)
  local ok, err = pcall(_G.PairDraftCompleteTest.run_completers)
  check(ok, label .. ': runner skips without E785', err)
end

runner_skips('normal', function() vim.cmd('stopinsert') end)
runner_skips('visual', function() vim.cmd('normal! v') end)

prime_completion_candidate()
vim.cmd('startinsert')
exit_to_normal()
local scheduled = false
vim.schedule(function()
  local mode = vim.api.nvim_get_mode().mode
  check(mode:sub(1, 1) ~= 'i', 'scheduled: driver is outside insert mode', mode)
  local ok, err = pcall(_G.PairDraftCompleteTest.run_completers)
  check(ok, 'scheduled: runner skips without E785', err)
  scheduled = true
end)
check(vim.wait(100, function() return scheduled end), 'scheduled callback ran')

O:write('TOTAL_FAILS=' .. fails .. '\n')
O:close()
vim.cmd('qall!')
LUA

if ! run_headless --timeout 30 -- \
  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
  nvim --headless -u "$INIT" "$RT/draft.md" \
  -c "luafile $RT/driver.lua"; then
  echo "draft-complete-mode-test: nvim driver failed"
  exit 1
fi

echo "draft-complete-mode-test:"
fails=0
if [ ! -f "$RT/result.txt" ]; then
  echo "  FAIL driver produced no result (nvim boot/driver error)"
  exit 1
fi
while IFS=$'\t' read -r status label detail; do
  case "$status" in
    ok)   printf '  ok   %s\n' "$label" ;;
    FAIL) printf '  FAIL %s: %s\n' "$label" "$detail"; fails=$((fails + 1)) ;;
    TOTAL_FAILS=*) ;;
  esac
done < "$RT/result.txt"

if [ "$fails" -ne 0 ]; then
  echo "draft-complete-mode-test: $fails failure(s)"
  exit 1
fi
echo "draft-complete-mode-test: all passed"
