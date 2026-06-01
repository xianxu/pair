#!/usr/bin/env bash
# Regression test for send_and_clear's queue/history state machine
# (nvim/init.lua). Drives the REAL init.lua headlessly — there is no way to
# unit-test it (monolithic config, all-local functions), so we boot nvim on a
# draft fixture, invoke the actual <M-Right>/<M-CR> keymap callbacks, and
# assert the resulting on-disk queue / log / draft state.
#
# Covers the bug where sending from a future queue item while the draft `*`
# was non-empty left the sent item in BOTH the queue (+N) and history (-1):
# the item was resolved by a stale display index after the draft-enqueue
# shifted every index by one. The fix captures the item's filename key before
# mutating the queue. See workshop/issues + the AGENTS lesson.
#
# Run: bash tests/queue-send-test.sh   (also wired into `make test`)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/nvim/init.lua"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-queue-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# Reset the per-tag fixture. Args: draft-body, then queue item bodies (front→back).
setup() {
  rm -rf "$RT/queue-test"
  mkdir -p "$RT/queue-test"
  printf '%s' "$1" > "$RT/draft-test.md"
  : > "$RT/log-test.md"
  shift
  local key=500000
  for body in "$@"; do
    printf '%s' "$body" > "$RT/queue-test/$key.md"
    key=$((key + 1))
  done
}

# Run a driver lua under headless nvim against the fixture. Writes a result
# file the asserts below read. zellij send-to-agent calls fail harmlessly
# headless (stderr swallowed) — only the file state matters here.
run() {
  PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
    nvim --headless -u "$INIT" "$RT/draft-test.md" \
    -c "luafile $RT/driver.lua" >/dev/null 2>&1 || true
}

# A driver that navigates right $1 times then sends, and dumps state as
# `key=body` lines for the queue plus LOG/DRAFT markers.
write_driver() {
  cat > "$RT/driver.lua" <<LUA
local dd = os.getenv('PAIR_DATA_DIR')
local O = io.open(dd..'/result.txt','w')
local function cm(l)
  local m = vim.fn.maparg(l,'n',false,true)
  if type(m)=='table' and m.callback then pcall(m.callback) end
end
for _=1,$1 do cm('<M-Right>') end
cm('<M-CR>')
local qd = dd..'/queue-test'
local fs = vim.fn.readdir(qd, function(n) return n:match('%.md\$') and 1 or 0 end)
table.sort(fs)
for _,f in ipairs(fs) do
  local h=io.open(qd..'/'..f); O:write('Q '..f:gsub('%.md\$','')..'='..(h:read('*a') or '')..'\n'); h:close()
end
local lh=io.open(dd..'/log-test.md'); local lc=lh and lh:read('*a') or ''; if lh then lh:close() end
for body in (lc..'\n---\n'):gmatch('## %S+ %S+\n\n(.-)\n\n%-%-%-') do
  O:write('L '..body..'\n')
end
local d=io.open(dd..'/draft-test.md'); O:write('D '..((d and d:read('*a') or ''):gsub('%s+\$',''))..'\n'); if d then d:close() end
O:close(); vim.cmd('qall')
LUA
}

# assert that result.txt contains an exact line
has() { grep -qxF "$1" "$RT/result.txt"; }
# assert NO line starting with prefix matches value (duplication guard)
count() { grep -cE "$1" "$RT/result.txt" || true; }

echo "queue-send-test:"

# --- 1. THE BUG: send +3 with a non-empty draft. ----------------------------
# Expect: draft parked at front (+1), CCC removed from queue and now sole
# history entry, NO CCC left in the queue.
setup "DRAFT-WIP" "AAA" "BBB" "CCC"
write_driver 3
run
if has "L CCC" && [ "$(count '^Q .*=CCC')" = "0" ] && has "Q 499999=DRAFT-WIP" \
   && has "Q 500000=AAA" && has "Q 500001=BBB"; then
  pass "send +3 w/ draft: CCC→history, draft parked at +1, no duplicate"
else
  fail "send +3 w/ draft (see result below)"; sed 's/^/    /' "$RT/result.txt"
fi

# --- 2. send from a queue slot with an EMPTY draft: no spurious enqueue. -----
setup "" "AAA" "BBB" "CCC"
write_driver 2   # → +2 (BBB)
run
if has "L BBB" && [ "$(count '^Q .*=BBB')" = "0" ] && has "Q 500000=AAA" \
   && has "Q 500002=CCC" && [ "$(count '^Q ')" = "2" ]; then
  pass "send +2 w/ empty draft: BBB→history, nothing parked, no duplicate"
else
  fail "send +2 w/ empty draft"; sed 's/^/    /' "$RT/result.txt"
fi

# --- 3. send straight from * (no queue interaction). ------------------------
setup "HELLO" "AAA"
write_driver 0
run
if has "L HELLO" && has "Q 500000=AAA" && [ "$(count '^Q ')" = "1" ]; then
  pass "send from *: HELLO→history, queue untouched"
else
  fail "send from *"; sed 's/^/    /' "$RT/result.txt"
fi

if [ "$fails" -ne 0 ]; then
  echo "queue-send-test: $fails failure(s)"; exit 1
fi
echo "queue-send-test: all passed"
