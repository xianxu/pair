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

-- pure decision (5 cases: a live pane → hide/show; else target-driven prompt/open/wait)
local A = _G._pair_review_toggle_action
OUT:write((A(false, false, nil) == 'prompt') and 'pure-prompt ok\n' or 'pure-prompt FAIL\n')
OUT:write((A(false, false, 'ready') == 'open') and 'pure-open ok\n' or 'pure-open FAIL\n')
OUT:write((A(false, false, 'proposed') == 'wait') and 'pure-wait ok\n' or 'pure-wait FAIL\n')
OUT:write((A(true, true) == 'hide') and 'pure-hide ok\n' or 'pure-hide FAIL\n')
OUT:write((A(true, false) == 'show') and 'pure-show ok\n' or 'pure-show FAIL\n')

local R = _G._pair_review
local target = vim.env.PAIR_DATA_DIR .. '/review-target-' .. vim.env.PAIR_TAG .. '.json'
local draft = vim.env.PAIR_DATA_DIR .. '/draft.md' -- exists (the test wrote it)

-- conversation-scope (#66 smoke #6): a target written under a DIFFERENT session
-- (PAIR_SESSION_ID=oldsid, pre-written below) is ignored by this session (testsid),
-- so a fresh session prompts instead of reopening the previous review.
OUT:write((R.read_target() == nil) and 'session-scope ok\n' or 'session-scope FAIL\n')

-- pure target_stale: same id → fresh; different / empty-current / no-id → stale.
local TS = R.target_stale
OUT:write((TS({ session = 'testsid' }, 'testsid') == false) and 'ts-same ok\n' or 'ts-same FAIL\n')
OUT:write((TS({ session = 'oldsid' }, 'testsid') == true) and 'ts-diff ok\n' or 'ts-diff FAIL\n')
OUT:write((TS({ session = 'x' }, '') == true) and 'ts-nocur ok\n' or 'ts-nocur FAIL\n')
OUT:write((TS({}, 'testsid') == true) and 'ts-noid ok\n' or 'ts-noid FAIL\n')

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

-- no live pane, NO target → prompt: no open (zellij run), no show/hide
os.remove(sf); os.remove(target)
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
OUT:write((not has(d, 'run --floating') and not has(d, 'hide-floating-panes')
  and not has(d, 'show-floating-panes')) and 'prompt ok\n' or 'prompt FAIL\n')

-- no live pane, target READY → open the pane (pair-review-open → zellij run)
R.write_target(draft, 'ready')
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
OUT:write(has(d, 'run --floating') and 'targetopen ok\n' or 'targetopen FAIL\n')

-- no live pane, target PROPOSED → wait: do NOT open
R.write_target(draft, 'proposed')
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
OUT:write((not has(d, 'run --floating')) and 'wait ok\n' or 'wait FAIL\n')

-- footgun: never toggle-floating-panes anywhere
OUT:write(has(read_zlog(), 'toggle-floating-panes') and 'footgun FAIL\n' or 'footgun ok\n')
OUT:close()
vim.cmd('qa!')
LUA

# a STALE review-target from a DIFFERENT conversation (session=oldsid). This session
# runs as PAIR_SESSION_ID=testsid, so read_target must ignore it (a fresh session
# prompts; an Alt+n resume — same id — would keep its target). (#66 smoke #6.)
printf '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}\n' > "$RT/review-target-test.json"
( cd "$RT" && PATH="$RT/bin:$PATH" \
    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" PAIR_SESSION_ID=testsid \
    RESULT="$RESULT" ZLOG="$ZLOG" FLOATVIS="$FLOATVIS" \
    run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/init.lua" "$RT/draft.md" \
      -c "luafile $RT/driver.lua" )

grep -q 'session-scope ok' "$RESULT" && pass "other-session target ignored (smoke #6)" || fail "stale (other-session) target not ignored"
for c in ts-same ts-diff ts-nocur ts-noid; do
  grep -q "$c ok" "$RESULT" && pass "pure target_stale: $c" || fail "target_stale $c"
done
grep -q 'pure-prompt ok'  "$RESULT" && pass "pure: no target → prompt"        || fail "pure prompt"
grep -q 'pure-open ok'    "$RESULT" && pass "pure: target ready → open"       || fail "pure open"
grep -q 'pure-wait ok'    "$RESULT" && pass "pure: target proposed → wait"    || fail "pure wait"
grep -q 'pure-hide ok'    "$RESULT" && pass "pure: alive+visible → hide"      || fail "pure hide"
grep -q 'pure-show ok'    "$RESULT" && pass "pure: alive+hidden → show"       || fail "pure show"
grep -q '^hide ok$'       "$RESULT" && pass "live+visible → hide-floating-panes" || fail "hide branch"
grep -q '^show ok$'       "$RESULT" && pass "live+hidden → show-floating-panes" || fail "show branch"
grep -q '^prompt ok$'     "$RESULT" && pass "no target → :PairReview prompt (no open/show/hide)" || fail "prompt branch"
grep -q '^targetopen ok$' "$RESULT" && pass "target ready → opens the pane (pair-review-open)" || fail "open branch"
grep -q '^wait ok$'       "$RESULT" && pass "target proposed → wait (no open)" || fail "wait branch"
grep -q '^footgun ok$'    "$RESULT" && pass "never toggle-floating-panes" || fail "footgun (toggle-floating-panes used)"

# ── config lint ───────────────────────────────────────────────────────────────
grep -q 'bind "Alt r"' "$ROOT/zellij/config.kdl" && pass "Alt+r bound in config.kdl" || fail "no Alt+r bind"
grep -q ':lua PairReviewToggle()' "$ROOT/zellij/config.kdl" && pass "Alt+r routes to :lua PairReviewToggle()" || fail "Alt+r target wrong"
grep -q 'Run "pair-review-toggle"' "$ROOT/zellij/config.kdl" && fail "Alt+r still spawns the old toggle pane" || pass "old pair-review-toggle pane gone"

[ "$fails" -eq 0 ] || { printf 'review-toggle-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-toggle-test ok\n'
