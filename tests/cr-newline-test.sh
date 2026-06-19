#!/usr/bin/env bash
# Regression test for the insert-mode <CR> decision in nvim/init.lua (#65).
#
# Bug: when a completion popup was showing but nothing was Tab-selected, the
# <CR> map returned a bare <CR> — which, while the pum is up, only closes the
# menu and swallows the keystroke, so no newline lands. Under
# completeopt=...,noselect (init.lua) nothing is EVER auto-selected, so that was
# the common case, not an edge: Return stopped breaking the line.
#
# Fix: route <CR> through the pure cr_keys(visible, has_selection, momentary):
#   no popup                       → <CR>        (plain newline)
#   popup + selection              → <C-y>       (accept the highlighted item)
#   popup, no selection, typing    → <C-e><CR>   (dismiss the menu, THEN newline)
#   popup, no selection, momentary → <CR>        (z= clean dismiss, NO newline)
# <C-e> cancels completion keeping exactly what was typed, so the following <CR>
# is a normal newline — Return always breaks the line when nothing was picked
# during as-you-type completion; the momentary z= spell popup keeps its
# clean-dismiss contract (no spurious newline).
#
# Like autopair-test.sh, this asserts the *expr string* the decision yields (the
# live popup needs a UI headless nvim lacks; feedkeys timing is flaky). It boots
# the REAL init.lua so it also proves the live <CR> map routes through cr_keys.
#
# Run: bash tests/cr-newline-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/nvim/init.lua"
. "$ROOT/tests/lib/run-headless.sh"   # run_headless: timeout watchdog (#60)
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-cr-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

printf '' > "$RT/draft.md"

cat > "$RT/driver.lua" <<'LUA'
local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
local fails = 0
local function check(label, got, want)
  local ok = got == want
  if not ok then fails = fails + 1 end
  O:write(string.format('%s\t%s\t%q\t%q\n', ok and 'ok' or 'FAIL', label, got, want))
end

-- Decision table: cr_keys(visible, has_selection, momentary) → key string.
-- {label, visible, has_selection, momentary, expected}
-- momentary=false → as-you-type draft completion (the #65 fix path).
-- momentary=true  → the transient z= spell popup (clean-dismiss contract; must
--                   NOT inject a newline — the milestone-review regression).
local cases = {
  { 'no popup -> newline',                  false, false, false, '<CR>'      },
  { 'no popup (sel irrelevant)',            false, true,  false, '<CR>'      },
  { 'typing: popup + selection -> accept',  true,  true,  false, '<C-y>'     },
  -- the #65 fix: nothing picked -> cancel completion (keep typed text) + newline
  { 'typing: nothing picked -> C-e + newline', true, false, false, '<C-e><CR>' },
  -- z= momentary picker keeps its clean dismiss (no spurious newline)
  { 'z=: nothing picked -> clean dismiss, no newline', true, false, true, '<CR>'  },
  { 'z=: popup + selection -> accept',      true,  true,  true,  '<C-y>'     },
}
assert(type(_G.PairCRKeys) == 'function', '_G.PairCRKeys must be defined by init.lua')
for _, c in ipairs(cases) do
  check(c[1], _G.PairCRKeys(c[2], c[3], c[4]), c[5])
end

-- Wiring: the real insert-mode <CR> map must route through cr_keys. Headless
-- has no UI, so no popup is up -> the callback must return the plain '<CR>'.
local m = vim.fn.maparg('<CR>', 'i', false, true)
local got = (type(m) == 'table' and m.callback) and m.callback() or '<no-map>'
check('live <CR> map routes through cr_keys (no popup)', got, '<CR>')

O:write('TOTAL_FAILS=' .. fails .. '\n')
O:close()
vim.cmd('qall!')
LUA

run_headless --timeout 30 -- \
  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
  nvim --headless -u "$INIT" "$RT/draft.md" \
  -c "luafile $RT/driver.lua"

echo "cr-newline-test:"
fails=0
if [ ! -f "$RT/result.txt" ]; then
  echo "  FAIL driver produced no result (nvim boot/driver error)"
  exit 1
fi
while IFS=$'\t' read -r status label got want; do
  case "$status" in
    ok)   printf '  ok   %s\n' "$label" ;;
    FAIL) printf '  FAIL %s: got %s want %s\n' "$label" "$got" "$want"; fails=$((fails + 1)) ;;
    TOTAL_FAILS=*) ;;  # summary line, ignore (per-case status is authoritative)
  esac
done < "$RT/result.txt"

if [ "$fails" -ne 0 ]; then
  echo "cr-newline-test: $fails failure(s)"
  exit 1
fi
echo "cr-newline-test: all passed"
