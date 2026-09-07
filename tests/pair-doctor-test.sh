#!/usr/bin/env bash
# Regression test for the :PairDoctor wiring (nvim/init.lua).
#
# WHY THIS FILE EXISTS. `make test-lua` covers nvim/doctor.lua's pure functions
# well, and both defects that reached the operator in #208 M2 still shipped:
# the rolling row's probe keys were read with `(%w+_ms)` (Lua's %w excludes `_`,
# so `pipe_hop_ms` was recorded as `hop_ms`), and the editor discriminator timed
# an insert-mode gate returning rather than the completion chain it named. Both
# lived in the glue between doctor.lua and init.lua, which no suite executed —
# and the pure test asserting the INTENDED key passed the entire time, because
# it never ran the caller. A green suite over pure functions says nothing about
# the wrapper that calls them.
#
# Run: bash tests/pair-doctor-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/nvim/init.lua"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-doctor-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

printf '' > "$RT/draft.md"

# A recorded perf.sh capture. The PRODUCER pins the CONSUMER: these are the key
# names perf.sh actually emits, so a reader that invents its own drifts visibly.
cat > "$RT/capture.txt" <<'CAP'
# pair perf capture
captured_at=2026-09-07T10:00:00-0700
window_seconds=2
load=9.51/8.10/7.44
cpu_idle_pct=71.2
windowserver_cpu_pct=7.0
pair_family_procs=14
build_procs=0
swapins_per_s=n/a (vm_stat unavailable)
pipe_hop_ms=0.007
fork_exec_ms=1.506
zellij_action_ms=n/a (zellij not on PATH)
elapsed_seconds=4

# end
CAP

cat > "$RT/driver.lua" <<'LUA'
local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
local fails = 0
local function check(cond, msg, detail)
  if cond then O:write('ok\t' .. msg .. '\n')
  else fails = fails + 1; O:write('FAIL\t' .. msg .. '\t' .. tostring(detail or '') .. '\n') end
end

check(type(_G.PairDoctorTest) == 'table', 'test seam exported')

local data = os.getenv('PAIR_DATA_DIR')
local fh = assert(io.open(data .. '/capture.txt')); local capture = fh:read('a'); fh:close()

local function set_note(text)
  vim.api.nvim_buf_set_lines(0, 0, -1, false, vim.split(text, '\n'))
end
local function jsonl_rows()
  local rows = {}
  local f = io.open(data .. '/perf-captures.jsonl')
  if not f then return rows end
  for line in f:lines() do rows[#rows + 1] = vim.json.decode(line) end
  f:close()
  return rows
end

-- 1. A successful capture: the row's probe keys must be perf.sh's own names.
_G.PairDoctorTest.set_runner(function(cb) cb({ code = 0, stdout = capture }) end)
set_note('typing went slow')
_G.PairDoctorTest.run()
vim.wait(2000, function() return #jsonl_rows() > 0 end)
local rows = jsonl_rows()
check(#rows == 1, 'a capture appends exactly one row', #rows)
local p = rows[1] and rows[1].probes or {}
check(p.pipe_hop_ms ~= nil, 'row uses perf.sh key pipe_hop_ms, not hop_ms', vim.inspect(p))
check(p.fork_exec_ms ~= nil, 'row uses fork_exec_ms, not exec_ms', vim.inspect(p))
check(p.zellij_action_ms == nil,
  'an n/a probe is OMITTED from the row, never coerced to a number', vim.inspect(p))
for k in pairs(p) do
  check(rows[1].baselines[k] ~= nil, 'every probe key has a matching baseline: ' .. k)
end

-- 2. The buffer-changed-mid-flight branch: the plan calls this the only
--    DATA-LOSS path in the design, and it was the one interleaving no test
--    drove. The injected runner owns the ordering, so it can rewrite the
--    buffer before handing back the capture.
_G.PairDoctorTest.set_runner(function(cb)
  vim.api.nvim_buf_set_lines(0, 0, -1, false, { 'typed AFTER invocation' })
  cb({ code = 0, stdout = capture })
end)
set_note('the original note')
_G.PairDoctorTest.run()
vim.wait(2000, function() return not _G.PairDoctorTest.is_running() end)
local mid = table.concat(vim.api.nvim_buf_get_lines(0, 0, -1, false), '\n')
check(mid:find('AFTER invocation', 1, true) ~= nil,
  'text typed during the capture is never destroyed', mid)

-- 3. A FAILED capture must keep the operator's note: the symptom description is
--    the one thing in this flow that cannot be re-measured.
_G.PairDoctorTest.set_runner(function(cb) cb({ code = 1, stdout = '' }) end)
set_note('the note that must survive')
_G.PairDoctorTest.run()
vim.wait(2000, function() return not _G.PairDoctorTest.is_running() end)
local kept = table.concat(vim.api.nvim_buf_get_lines(0, 0, -1, false), '\n')
check(kept:find('must survive', 1, true) ~= nil, 'a failed capture preserves the note', kept)

-- 4. The editor legs. C1 shipped twice because nothing asserted that the timed
--    completion chain did any work: it bailed at the insert-mode gate, then at
--    `col == 0`, while the report said `fast` and SKILL.md told the reader to
--    exclude #201/#203 on it.
_G.PairDoctorTest.set_runner(function(cb) cb({ code = 0, stdout = capture }) end)
local before_work = _G.PairDoctorCompleteProbe.work_count()
local bufs_before = #vim.api.nvim_list_bufs()
set_note('leg check')
_G.PairDoctorTest.run()
vim.wait(2000, function() return not _G.PairDoctorTest.is_running() end)
check(_G.PairDoctorCompleteProbe.work_count() > before_work,
  'the timed completion chain reaches its candidate build, not just its gates',
  'work counter did not move')
check(#vim.api.nvim_list_bufs() <= bufs_before,
  'time_editor leaves no scratch buffer behind', #vim.api.nvim_list_bufs())

-- 5. A failed SEND must keep the note too. A successful capture says nothing
--    about whether the agent received it, and a cleared draft is what tells the
--    operator it went through -- so this is the path where the note is lost
--    while they believe it was delivered.
local real_send = _G.send_generated_prompt
_G.send_generated_prompt = function() return false end
_G.PairDoctorTest.set_runner(function(cb) cb({ code = 0, stdout = capture }) end)
set_note('note that outlives a failed send')
_G.PairDoctorTest.run()
vim.wait(2000, function() return not _G.PairDoctorTest.is_running() end)
local after_send = table.concat(vim.api.nvim_buf_get_lines(0, 0, -1, false), '\n')
check(after_send:find('outlives a failed send', 1, true) ~= nil,
  'a failed send preserves the note', after_send)
_G.send_generated_prompt = real_send

-- 6. The in-flight guard must reset even when the runner throws, or :PairDoctor
--    is dead for the session on exactly the struggling machine it exists for.
_G.PairDoctorTest.set_runner(function() error('spawn exploded') end)
set_note('x')
_G.PairDoctorTest.run()
check(not _G.PairDoctorTest.is_running(), 'a throwing spawn resets the in-flight guard')

O:write('TOTAL_FAILS=' .. fails .. '\n')
O:close()
vim.cmd('qall!')
LUA

if ! run_headless --timeout 40 -- \
  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
  PAIR_DRAFT_PATH="$RT/draft.md" PAIR_LAYOUT_MODE_PATH="$RT/layout-mode-test" \
  nvim --headless -u "$INIT" "$RT/draft.md" \
  -c "luafile $RT/driver.lua"; then
  echo "pair-doctor-test: nvim driver failed"
  exit 1
fi

echo "pair-doctor-test:"
fails=0
if [ ! -f "$RT/result.txt" ]; then
  echo "  FAIL driver produced no result (nvim boot/driver error)"
  exit 1
fi
while IFS=$'\t' read -r status label detail; do
  case "$status" in
    ok)   printf '  ok   %s\n' "$label" ;;
    FAIL) printf '  FAIL %s: %s\n' "$label" "$detail"; fails=$((fails + 1)) ;;
  esac
done < "$RT/result.txt"

if [ "$fails" -ne 0 ]; then
  echo "pair-doctor-test: $fails failure(s)"
  exit 1
fi
echo "pair-doctor-test: all passed"
