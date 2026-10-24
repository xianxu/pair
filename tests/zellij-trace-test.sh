#!/usr/bin/env bash
# tests/zellij-trace-test.sh — zellij action tracing logs metadata without
# leaking prompt bodies.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-zellij-trace-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"
ZLOG="$RT/zlog.txt"
: > "$ZLOG"

mkdir -p "$RT/bin"
cat > "$RT/bin/zellij" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$ZLOG"
printf 'zellij-ok'
EOF
chmod +x "$RT/bin/zellij"
cat > "$RT/bin/pair" <<'SH'
#!/bin/sh
[ "$1 $2 $3" = 'internal zellij-command --' ] || exit 92
shift 3
printf '%s\n' helper >> "$PAIR_HELPER_LOG"
exec zellij "$@"
SH
chmod +x "$RT/bin/pair"

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local out = io.open(os.getenv('RESULT'), 'w')
local ok, trace = pcall(dofile, ROOT .. '/nvim/zellij_trace.lua')
if not ok then
  out:write('load-error=' .. tostring(trace) .. '\n')
  out:close()
  vim.cmd('cquit')
end
assert(vim.deep_equal(trace.command({'zellij','action','write-chars',"one 'quoted' body"}),
 {'pair','internal','zellij-command','--','action','write-chars',"one 'quoted' body"}))
local old_home = vim.env.PAIR_HOME
vim.env.PAIR_HOME = '/pair home'
assert(trace.command({'zellij','action','noop'})[1] == 'pair', 'installed runtime has no bin/pair; use launcher PATH')
assert(trace.command({'zellij','action','noop'}, '/pair home/bin/pair')[1] == 'pair', 'missing explicit runtime helper falls back to launcher PATH')
vim.env.PAIR_HOME = old_home
local unchanged = {'zellij','action','write-chars','', 'one\ntwo'}
assert(vim.deep_equal(trace.command(unchanged), {'pair','internal','zellij-command','--','action','write-chars','', 'one\ntwo'}))
assert(unchanged[1] == 'zellij' and #unchanged == 5, 'adapter must not mutate caller argv')
local body = 'secret repro prompt body'
local res = trace.action('draft.write-body', { 'zellij', 'action', 'write-chars', body }, {
  redact = { [4] = body },
  timeout_ms = 1000,
})
out:write('rc=' .. tostring(res.code) .. '\n')
out:write('stdout=' .. tostring(res.stdout) .. '\n')
assert(trace.action('ordinary', {'zellij', 'action', 'noop'}).code == 0)
local start = (vim.uv or vim.loop).hrtime()
local blocked = trace.action('draft.blocked', { 'sleep', '2' }, { timeout_ms = 40 })
assert(blocked.code ~= 0, 'blocked child must time out')
assert(((vim.uv or vim.loop).hrtime()-start)/1e6 < 1500, 'child ignored timeout')
local expired = trace.action('draft.expired', { '__must_not_spawn__' }, { timeout_ms = 0 })
assert(expired.code ~= 0, 'expired action must refuse before spawn')
out:close()
vim.cmd('qa!')
LUA

PATH="$RT/bin:$PATH" \
PAIR_ROOT="$ROOT" \
PAIR_HOME= PAIR_HELPER_LOG="$RT/helper-log" \
PAIR_DATA_DIR="$RT" \
PAIR_TAG=trace \
PAIR_ZELLIJ_ACTIONS_PATH="$RT/zellij-actions-trace.jsonl" \
RESULT="$RESULT" \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua"

test "$(wc -l < "$RT/helper-log" | tr -d ' ')" = 2
TRACE="$RT/zellij-actions-trace.jsonl"
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

grep -q '^rc=0$' "$RESULT" && pass "returns zellij exit code" || fail "missing rc=0"
grep -q '^stdout=zellij-ok$' "$RESULT" && pass "returns zellij stdout" || fail "missing stdout"
test -s "$TRACE" && pass "writes trace file" || fail "trace file missing"
grep -q '"label":"draft.write-body"' "$TRACE" && pass "records label" || fail "label missing"
grep -q '"body_len":24' "$TRACE" && pass "records body length" || fail "body length missing"
grep -q '"body_sha256_12":"' "$TRACE" && pass "records body hash" || fail "body hash missing"
if grep -q 'secret repro prompt body' "$TRACE"; then
  fail "trace leaked prompt body"
else
  pass "redacts prompt body"
fi
grep -q '^action write-chars secret repro prompt body$' "$ZLOG" \
  && pass "still sends full body to zellij" \
  || fail "zellij did not receive original body"

[ "$fails" -eq 0 ] || { printf 'zellij-trace-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'zellij-trace-test ok\n'
