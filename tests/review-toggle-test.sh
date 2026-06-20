#!/usr/bin/env bash
# tests/review-toggle-test.sh — the Alt+r review toggle, now a draft-nvim lua
# fn (#66 M3 rework; the old transient pair-review-toggle floating pane caused
# the open delay / auto-hide / half-size / mis-fire smoke bugs).
#
#   _pair_review_toggle_action(alive, visible) (pure):
#     not alive          → 'open'   (file-select)
#     alive  + visible   → 'hide'
#     alive  + hidden    → 'show'
#   PairReviewToggle() (integration, zellij stubbed on $PATH):
#     live state file + are-floating-panes-visible=true  → hide-floating-panes
#     live state file + are-floating-panes-visible=false → show-floating-panes
#     no state file → file-select (no visibility query, no show/hide)
#   and NEVER toggle-floating-panes (the footgun).
#
# Live zellij pane/focus behaviour is the manual smoke (M3 plan Task 5). Here
# zellij is a $PATH stub that records argv and answers are-floating-panes-visible
# from a file the driver rewrites between branches.
#
# Run: bash tests/review-toggle-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-toggle-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"; ZLOG="$RT/zlog.txt"; FLOATVIS="$RT/floatvis"; : > "$ZLOG"
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# zellij stub: record every action; answer are-floating-panes-visible from a file.
mkdir -p "$RT/bin"
cat > "$RT/bin/zellij" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$ZLOG"
if [ "\$1" = action ] && [ "\$2" = are-floating-panes-visible ]; then
  cat "$FLOATVIS" 2>/dev/null || echo false
fi
exit 0
EOF
chmod +x "$RT/bin/zellij"

printf 'draft\n' > "$RT/draft.md"
cat > "$RT/driver.lua" <<'LUA'
local OUT = io.open(os.getenv('RESULT'), 'w')
local ZLOG = os.getenv('ZLOG')
local FLOATVIS = os.getenv('FLOATVIS')
local sf = vim.env.PAIR_DATA_DIR .. '/review-' .. vim.env.PAIR_TAG .. '.open'

local function read_zlog()
  local f = io.open(ZLOG, 'r'); if not f then return {} end
  local t = {}; for l in f:lines() do t[#t + 1] = l end; f:close(); return t
end
local function new_since(n)
  local all = read_zlog(); local out = {}
  for i = n + 1, #all do out[#out + 1] = all[i] end; return out
end
local function has(lines, pat)
  for _, l in ipairs(lines) do if l:find(pat, 1, true) then return true end end
  return false
end
local function setfloat(v) local f = io.open(FLOATVIS, 'w'); f:write(v); f:close() end

-- pure decision
local A = _G._pair_review_toggle_action
OUT:write((A(false) == 'open') and 'pure-open ok\n' or 'pure-open FAIL\n')
OUT:write((A(true, true) == 'hide') and 'pure-hide ok\n' or 'pure-hide FAIL\n')
OUT:write((A(true, false) == 'show') and 'pure-show ok\n' or 'pure-show FAIL\n')

-- live + visible → hide  (state file holds OUR pid, so kill -0 says alive)
vim.fn.writefile({ tostring(vim.fn.getpid()) }, sf); setfloat('true')
local n = #read_zlog(); _G.PairReviewToggle()
local d = new_since(n)
OUT:write((has(d, 'action are-floating-panes-visible') and has(d, 'action hide-floating-panes'))
  and 'hide ok\n' or 'hide FAIL\n')

-- live + hidden → show
vim.fn.writefile({ tostring(vim.fn.getpid()) }, sf); setfloat('false')
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
OUT:write(has(d, 'action show-floating-panes') and 'show ok\n' or 'show FAIL\n')

-- no review → file-select: no visibility query, no show/hide
os.remove(sf)
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
local quiet = not has(d, 'are-floating-panes-visible')
  and not has(d, 'hide-floating-panes') and not has(d, 'show-floating-panes')
OUT:write(quiet and 'open ok\n' or 'open FAIL\n')

-- footgun: never toggle-floating-panes anywhere
OUT:write(has(read_zlog(), 'toggle-floating-panes') and 'footgun FAIL\n' or 'footgun ok\n')
OUT:close()
vim.cmd('qa!')
LUA

( cd "$RT" && PATH="$RT/bin:$PATH" \
    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
    RESULT="$RESULT" ZLOG="$ZLOG" FLOATVIS="$FLOATVIS" \
    run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/init.lua" "$RT/draft.md" \
      -c "luafile $RT/driver.lua" )

grep -q 'pure-open ok'  "$RESULT" && pass "pure: not alive → open"        || fail "pure open"
grep -q 'pure-hide ok'  "$RESULT" && pass "pure: alive+visible → hide"    || fail "pure hide"
grep -q 'pure-show ok'  "$RESULT" && pass "pure: alive+hidden → show"     || fail "pure show"
grep -q '^hide ok$'     "$RESULT" && pass "live+visible → hide-floating-panes (after are-visible)" || fail "hide branch"
grep -q '^show ok$'     "$RESULT" && pass "live+hidden → show-floating-panes" || fail "show branch"
grep -q '^open ok$'     "$RESULT" && pass "no review → file-select (no visibility query / show / hide)" || fail "open branch"
grep -q '^footgun ok$'  "$RESULT" && pass "never toggle-floating-panes" || fail "footgun (toggle-floating-panes used)"

# ── config lint ───────────────────────────────────────────────────────────────
grep -q 'bind "Alt r"' "$ROOT/zellij/config.kdl" && pass "Alt+r bound in config.kdl" || fail "no Alt+r bind"
grep -q ':lua PairReviewToggle()' "$ROOT/zellij/config.kdl" && pass "Alt+r routes to :lua PairReviewToggle()" || fail "Alt+r target wrong"
grep -q 'Run "pair-review-toggle"' "$ROOT/zellij/config.kdl" && fail "Alt+r still spawns the old toggle pane" || pass "old pair-review-toggle pane gone"

[ "$fails" -eq 0 ] || { printf 'review-toggle-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-toggle-test ok\n'
