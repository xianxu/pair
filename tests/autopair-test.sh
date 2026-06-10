#!/usr/bin/env bash
# Regression test for the hand-rolled autopair keymaps (nvim/init.lua,
# pair_insert_open). Drives the REAL init.lua headlessly — there is no way to
# unit-test it (monolithic config, all-local functions), so we boot nvim, set a
# buffer line + cursor, invoke the actual insert-mode keymap callback, and
# assert the expr string it would feed.
#
# Guards the "next-char gate": an opener only auto-inserts its closer when the
# cursor sits at end-of-line or in front of whitespace. If a real character
# follows (typing `(` before `foo`, or before a `)`), only the bare opener is
# inserted — no stranded closer. The pre-existing quote rules (jump-over a
# duplicate quote; keep apostrophes in "don't" single) must still hold.
#
# Run: bash tests/autopair-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/nvim/init.lua"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-autopair-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

printf '' > "$RT/draft.md"

# Driver: for each case, set `pre..post` as the line, park the cursor between
# pre and post, fire the opener's insert-mode mapping, and compare the returned
# expr string to the expectation. virtualedit=onemore lets the cursor sit at
# #line (true EOL), matching where it lives in insert mode.
cat > "$RT/driver.lua" <<'LUA'
vim.o.virtualedit = 'onemore'
local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
local function fire(key, pre, post)
  vim.api.nvim_buf_set_lines(0, 0, -1, false, { pre .. post })
  vim.api.nvim_win_set_cursor(0, { 1, #pre })
  local m = vim.fn.maparg(key, 'i', false, true)
  if type(m) ~= 'table' or not m.callback then return '<no-map:' .. key .. '>' end
  return m.callback()
end
-- {label, key, pre, post, expected-expr-string}
local cases = {
  -- Brackets: pair at EOL / before whitespace, bare opener before a real char.
  { 'paren EOL',        '(', 'foo',  '',     '()<C-G>U<Left>' },
  { 'bracket EOL',      '[', 'x',    '',     '[]<C-G>U<Left>' },
  { 'brace EOL',        '{', 'x',    '',     '{}<C-G>U<Left>' },
  { 'paren pre-space',  '(', 'a',    ' b',   '()<C-G>U<Left>' },
  { 'paren pre-word',   '(', '',     'foo',  '('             },
  { 'bracket pre-word', '[', '',     'foo',  '['             },
  { 'paren pre-close',  '(', '',     ')x',   '('             },
  -- Quotes: gate applies, but the prior quote rules win where they fire.
  { 'dquote EOL',       '"', 'say ', '',     '""<C-G>U<Left>' },
  { 'backtick EOL',     '`', 'run ', '',     '``<C-G>U<Left>' },
  { 'dquote pre-word',  '"', 'a ',   'word', '"'             },
  { 'apostrophe dont',  "'", 'don',  '',     "'"             },  -- prev word char -> single
  { 'dquote jump-over', '"', 'x',    '"',    '<C-G>U<Right>' },  -- next==quote -> jump
}
local fails = 0
for _, c in ipairs(cases) do
  local got = fire(c[2], c[3], c[4])
  local ok = got == c[5]
  if not ok then fails = fails + 1 end
  O:write(string.format('%s\t%s\t%q\t%q\n', ok and 'ok' or 'FAIL', c[1], got, c[5]))
end
O:write('TOTAL_FAILS=' .. fails .. '\n')
O:close()
vim.cmd('qall')
LUA

PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
  nvim --headless -u "$INIT" "$RT/draft.md" \
  -c "luafile $RT/driver.lua" >/dev/null 2>&1 || true

echo "autopair-test:"
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
  echo "autopair-test: $fails failure(s)"
  exit 1
fi
echo "autopair-test: all passed"
