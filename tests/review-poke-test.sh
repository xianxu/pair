#!/usr/bin/env bash
# tests/review-poke-test.sh — pair_poke writes to the agent pane by ABSOLUTE id
# without moving focus away from the review pane.
# zellij is stubbed on $PATH: list-panes returns a canned pane set, every other
# action is recorded. Asserts the id-based sequence + that pair_poke does NOT
# short-circuit headless (has_ui).
#
# Run: bash tests/review-poke-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-poke-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"; ZLOG="$RT/zlog.txt"; : > "$ZLOG"

# canned panes: agent (id 7, tiled, "claude"), draft (id 3), review (id 9, floating, focused)
cat > "$RT/panes.json" <<'JSON'
{"tab_one":{"panes":[
  {"id":7,"is_plugin":false,"is_floating":false,"is_focused":false,"title":"claude"},
  {"id":3,"is_plugin":false,"is_floating":false,"is_focused":false,"title":"draft"},
  {"id":9,"is_plugin":false,"is_floating":true,"is_focused":true,"title":"review"}
]}}
JSON

mkdir -p "$RT/bin"
cat > "$RT/bin/zellij" <<EOF
#!/usr/bin/env bash
if [ "\$1" = action ] && [ "\$2" = list-panes ]; then cat "$RT/panes.json"; exit 0; fi
printf '%s\n' "\$*" >> "$ZLOG"
EOF
chmod +x "$RT/bin/zellij"

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local poke = dofile(ROOT .. '/nvim/pair_poke.lua')
local OUT = io.open(os.getenv('RESULT'), 'w')
-- unit: pure _cmds shape
local c = poke._cmds('hello', 7, 9)
local ok_cmds = c[1][3] == 'write-chars' and c[1][4] == '--pane-id'
  and c[1][5] == '7' and c[1][6] == 'hello'
  and c[2][3] == 'send-keys' and c[2][4] == '--pane-id'
  and c[2][5] == '7' and c[2][6] == 'Alt Enter'
OUT:write(ok_cmds and 'cmds ok\n' or 'cmds FAIL\n')
-- send resolves agent (7) and writes by pane id without changing focus.
poke.send('updated, please review foo.md')
OUT:write('sent\n'); OUT:close()
LUA

PATH="$RT/bin:$PATH" PAIR_ROOT="$ROOT" RESULT="$RESULT" PAIR_DATA_DIR="$RT" PAIR_TAG=poke \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }
grep -q 'cmds ok' "$RESULT" && pass "_cmds builds the id-based argv" || fail "_cmds shape"
grep -q '^action write-chars --pane-id 7 updated, please review foo.md$' "$ZLOG" && pass "writes the please-review body to agent pane" || fail "no pane-id write-chars body"
grep -q '^action send-keys --pane-id 7 Alt Enter$' "$ZLOG" && pass "submits with semantic Alt+Enter to agent pane" || fail "no pane-id submit"
grep -q 'focus-pane-id' "$ZLOG" && fail "changed focus while poking agent" || pass "does not change focus"
grep -q 'move-focus' "$ZLOG" && fail "used relative move-focus (must be id-based)" || pass "no relative move-focus"
TRACE="$RT/zellij-actions-poke.jsonl"
test -s "$TRACE" && pass "writes zellij action trace" || fail "missing zellij action trace"
grep -q '"label":"review.poke.list-panes"' "$TRACE" && pass "traces pane lookup" || fail "missing list-panes trace"
grep -q '"label":"review.poke.write-body"' "$TRACE" && pass "traces body write" || fail "missing write-body trace"
grep -q '"body_len":29' "$TRACE" && pass "records redacted body length" || fail "missing body length"
if grep -q 'updated, please review foo.md' "$TRACE"; then
  fail "trace leaked review poke body"
else
  pass "redacts review poke body"
fi

[ "$fails" -eq 0 ] || { printf 'review-poke-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-poke-test ok\n'
