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
-- BR-23: absent timings must NOT read as 'fast'. Claiming the editor is
-- healthy on no evidence would send a reader hunting the environment on the
-- strength of a measurement that never happened.
eq(M.verdict(nil, nil), 'unknown', 'no timings yields unknown, not fast')
-- Asymmetric: partial evidence can prove slow, never fast.
eq(M.verdict(nil, 2), 'unknown', 'half a measurement cannot clear the editor')
eq(M.verdict(2, nil), 'unknown', 'the same at the other arity')
eq(M.verdict(nil, 99), 'slow', 'one slow timing IS positive evidence')
eq(M.verdict(99, nil), 'slow', 'positive evidence at either arity')

-- parse_samples must not turn absent input into empty samples that delta would
-- join into a confident "nothing is running".
do
  local a, b = M.parse_samples('')
  ok(a == nil and b == nil, 'empty capture yields no samples')
  local c = M.parse_samples('# unrelated text\nkey=value')
  ok(c == nil, 'a capture with no sample blocks yields no samples')
end

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


do -- BR-3: alive in both samples but a cputime row is missing. Every pid must
   -- land in exactly one bucket; one that reaches none is invisible.
  local a = { procs = { ['5'] = { etime = 10, comm = 'x' } }, cpu = {} }
  local b = { procs = { ['5'] = { etime = 12, comm = 'x' } }, cpu = { ['5'] = 3 } }
  local d = M.delta(a, b, 2)
  eq(#d.rates, 0, 'no rate without both cputime rows')
  eq(d.unmeasured, 1, 'unmeasured counted, not dropped')
  eq(d.vanished + d.started + d.reused, 0, 'and not miscounted as something else')
end

-- BR-8: the validity guard must actually run. Malformed input returns nil
-- rather than erroring or fabricating a number.
eq(M.parse_duration('1:xx:00'), nil, 'non-numeric field rejected')
eq(M.parse_duration('::'), nil, 'empty fields rejected')
eq(M.parse_duration('12-'), nil, 'days with no time rejected')


-- BR-9: delta driven from a REAL recorded capture rather than literals, so the
-- perf.sh -> delta contract is pinned. A change to either side that breaks the
-- other now fails here instead of at the operator's next slowdown.
do
  -- The fixture lives under doctor/, NOT nvim/: the runtime bundle walks nvim/
  -- wholesale, so a fixture there would ship to every user's extracted session.
  local fh = io.open(here .. '../doctor/fixtures/perf_capture.txt', 'r')
  ok(fh ~= nil, 'recorded perf capture fixture exists')
  if fh then
    local raw = fh:read('*a'); fh:close()
    local a, b = M.parse_samples(raw)
    ok(a ~= nil and b ~= nil, 'both samples parsed from the real capture')
    if a and b then
      local n = 0
      for _ in pairs(a.procs) do n = n + 1 end
      ok(n > 20, 'the fixture carries a realistic process count, got ' .. n)
      local d = M.delta(a, b, 2)
      ok(#d.rates > 0, 'real samples yield rates')
      ok(d.rows_a > 0 and d.rows_b > 0, 'row counts reported')
      -- Every pid lands in exactly one bucket: the contract delta promises.
      local accounted = #d.rates + d.vanished + d.reused + d.unmeasured
      eq(accounted, d.rows_a, 'every pid in sample_a is accounted for')
      for _, r in ipairs(d.rates) do
        ok(r.cpu_pct >= 0, 'no negative cpu rate for pid ' .. tostring(r.pid))
      end
    end
  end
end


-- BR-35/BR-40: the budget-shed path must not fabricate a reading. A shed
-- sample_b still emits its header with `skipped=`, and an empty-but-present
-- sample made delta report EVERY process as vanished.
do
  local shed = table.concat({
    'window_seconds=2',
    '## sample_a', '### cputime', '1\t0:01.00', '### procs', '1\t100\t50\tsh',
    '## sample_b', 'skipped=budget reserved for probes; no rates computable',
  }, '\n')
  local a, b, w = M.parse_samples(shed)
  ok(a ~= nil, 'sample_a still parsed')
  eq(b, nil, 'a SHED sample_b is nil, not an empty sample')
  eq(w, 2, 'window_seconds parsed for delta to divide by')
  local d = M.delta(a, b, w)
  eq(d.vanished, 0, 'a shed sample must not report every process as vanished')
  eq(#d.rates, 0, 'and yields no rates')
end

-- window arriving as a string (which is what a parsed report gives) must not
-- crash the join.
do
  local a = { procs = { ['1'] = { etime = 1 } }, cpu = { ['1'] = 1 } }
  local b = { procs = { ['1'] = { etime = 3 } }, cpu = { ['1'] = 2 } }
  eq(#M.delta(a, b, '2').rates, 1, 'a string window is coerced, not fatal')
  eq(#M.delta(a, b, 'garbage').rates, 0, 'an unparseable window yields no rates')
end

if fails > 0 then
  io.stderr:write(string.format('\n%d failure(s)\n', fails))
  os.exit(1)
end
print('all doctor.lua tests passed')
