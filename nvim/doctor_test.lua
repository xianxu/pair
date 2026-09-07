-- Headless tests for nvim/doctor.lua — run via `nvim -l nvim/doctor_test.lua`
-- (or `make test-lua`). Pure Lua; no vim API. Exits non-zero on failure so the
-- make target fails loudly. Pins the two behaviors the Spec cares about:
-- $PAIR_HOME-absolute substitution and graceful nil-on-unset.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'doctor.lua')

local fails = 0
local function ok(cond, msg)
  if not cond then
    io.stderr:write('FAIL ' .. msg .. '\n')
    fails = fails + 1
  end
end

local HOME = '/Users/x/workspace/pair'
local p = M.payload(HOME)

ok(type(p) == 'string', 'payload returns a string for a real PAIR_HOME')
ok(p:find(HOME .. '/doctor/doctor.sh', 1, true) ~= nil, 'payload has the absolute doctor.sh path')
ok(p:find(HOME .. '/doctor/SKILL.md', 1, true) ~= nil, 'payload references SKILL.md (DRY pointer)')
-- The path must be SUBSTITUTED, not left as a literal for the agent's shell.
ok(p:find('$PAIR_HOME', 1, true) == nil, 'payload has no literal $PAIR_HOME (substituted, not deferred)')

-- Trailing slash on PAIR_HOME must not double up in the paths.
local q = M.payload(HOME .. '/')
ok(q:find(HOME .. '/doctor/doctor.sh', 1, true) ~= nil, 'trailing slash trimmed (no //)')
ok(q:find('//doctor', 1, true) == nil, 'no doubled slash before doctor/')

-- Graceful degrade: unset / empty ⇒ nil (caller notifies, no broken send).
ok(M.payload(nil) == nil, 'payload(nil) ⇒ nil')
ok(M.payload('') == nil, "payload('') ⇒ nil")


local function eq(got, want, msg)
  ok(got == want, string.format('%s: got %s want %s', msg, tostring(got), tostring(want)))
end

-- --- #208 performance capture -----------------------------------------------

eq(M.parse_duration('0:01.50'), 1.5, 'ps TIME mm:ss.ss')
eq(M.parse_duration('1:00:00'), 3600, 'ps TIME hh:mm:ss')
eq(M.parse_duration('2-03:00:00'), 2 * 86400 + 3 * 3600, 'etime with days')
eq(M.parse_duration('garbage'), nil, 'unreadable duration is nil, not 0')
eq(M.parse_duration(nil), nil, 'nil duration')

-- verdict: 16ms is one frame at 60Hz, the perceptibility floor.
eq(M.verdict(1, 2), 'fast', 'well inside a frame')
eq(M.verdict(15.9, 0), 'fast', 'just inside a frame')
eq(M.verdict(16, 0), 'slow', 'at the frame boundary')
eq(M.verdict(0, 40), 'slow', 'redraw alone can be slow')
eq(M.verdict(nil, nil), 'fast', 'missing timings do not fabricate a slow verdict')

-- note_from_lines: blank must be nil, not '' -- an empty note is a CLAIM that
-- the operator reported nothing.
eq(M.note_from_lines({ '', '   ', '' }), nil, 'blank buffer has no note')
eq(M.note_from_lines({}), nil, 'empty buffer has no note')
eq(M.note_from_lines(nil), nil, 'nil lines')
eq(M.note_from_lines({ 'typing slow', 'top took 10s' }),
   'typing slow\ntop took 10s', 'multi-line note preserved')
eq(M.note_from_lines({ '  padded  ' }), 'padded', 'note is trimmed')

-- delta: the three cases the naive pid join gets wrong.
local function sample(procs, cpu)
  return { procs = procs, cpu = cpu }
end

do -- a steady process: 1s of CPU over a 2s window is 50%
  local a = sample({ ['1'] = { etime = 100, rss = 10, comm = 'x' } }, { ['1'] = 5 })
  local b = sample({ ['1'] = { etime = 102, rss = 12, comm = 'x' } }, { ['1'] = 6 })
  local d = M.delta(a, b, 2)
  eq(#d.rates, 1, 'one rate')
  eq(d.rates[1].cpu_pct, 50, 'cpu rate over the window')
  eq(d.rates[1].rss_kb, 12, 'rss from the later sample')
end

do -- vanished: counted, never silently dropped
  local a = sample({ ['1'] = { etime = 10, comm = 'gone' } }, { ['1'] = 1 })
  local b = sample({}, {})
  local d = M.delta(a, b, 2)
  eq(d.vanished, 1, 'vanished counted')
  eq(#d.rates, 0, 'no rate for a vanished pid')
end

do -- started: no rate is computable, and 0 would be a different claim
  local a = sample({}, {})
  local b = sample({ ['9'] = { etime = 1, comm = 'new' } }, { ['9'] = 0.5 })
  local d = M.delta(a, b, 2)
  eq(d.started, 1, 'started counted')
  eq(#d.rates, 0, 'no rate for a started pid')
end

do -- reused pid: etime went DOWN, so it is a different process
  local a = sample({ ['7'] = { etime = 900, comm = 'old' } }, { ['7'] = 100 })
  local b = sample({ ['7'] = { etime = 2, comm = 'new' } }, { ['7'] = 0.1 })
  local d = M.delta(a, b, 2)
  eq(d.reused, 1, 'reused pid counted')
  eq(#d.rates, 0, 'a reused pid gets no nonsense rate')
end

do -- truncated samples are visible, so unequal captures are not compared blindly
  local a = sample({ ['1'] = { etime = 1 }, ['2'] = { etime = 1 } }, {})
  local b = sample({ ['1'] = { etime = 2 } }, {})
  local d = M.delta(a, b, 2)
  eq(d.rows_a, 2, 'rows_a reported')
  eq(d.rows_b, 1, 'rows_b reported')
end

do -- degenerate windows must not divide by zero or invent rates
  local a = sample({ ['1'] = { etime = 1 } }, { ['1'] = 1 })
  eq(#M.delta(a, a, 0).rates, 0, 'zero window yields no rates')
  eq(#M.delta(nil, nil, 2).rates, 0, 'nil samples yield no rates')
end

if fails > 0 then
  io.stderr:write(string.format('\n%d failure(s)\n', fails))
  os.exit(1)
end
print('all doctor.lua tests passed')
