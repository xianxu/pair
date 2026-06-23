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

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local out = io.open(os.getenv('RESULT'), 'w')
local ok, trace = pcall(dofile, ROOT .. '/nvim/zellij_trace.lua')
if not ok then
  out:write('load-error=' .. tostring(trace) .. '\n')
  out:close()
  vim.cmd('cquit')
end
local body = 'secret repro prompt body'
local res = trace.action('draft.write-body', { 'zellij', 'action', 'write-chars', body }, {
  redact = { [4] = body },
})
out:write('rc=' .. tostring(res.code) .. '\n')
out:write('stdout=' .. tostring(res.stdout) .. '\n')
out:close()
vim.cmd('qa!')
LUA

PATH="$RT/bin:$PATH" \
PAIR_ROOT="$ROOT" \
PAIR_DATA_DIR="$RT" \
PAIR_TAG=trace \
RESULT="$RESULT" \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua"

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
