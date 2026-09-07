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


-- perf_payload: the note LEADS, and the drift instruction survives verbatim.
do
  local drift = M.payload('/h')
  local out = M.perf_payload('/h', 'typing slow, top took 10s', 'editor: fast', 'load=9')
  ok(out ~= nil, 'perf_payload built')
  ok(out:find('typing slow', 1, true) < out:find('load=9', 1, true),
     'the operator note precedes the numbers')
  ok(out:find(drift, 1, true) ~= nil, "#48's drift instruction is embedded verbatim")
  ok(out:find('n/a', 1, true) ~= nil, 'the report tells the reader how to read n/a')

  -- A blank note must not silently look like "nothing was wrong".
  local none = M.perf_payload('/h', nil, 'editor: fast', 'load=9')
  ok(none:find('left no note', 1, true) ~= nil, 'an absent note is stated, not omitted')

  -- Missing halves degrade rather than vanishing.
  local bare = M.perf_payload('/h', nil, nil, nil)
  ok(bare:find('self%-timing did not run') ~= nil, 'absent editor timing is named')
  ok(bare:find('capture did not run', 1, true) ~= nil, 'absent env capture is named')

  eq(M.perf_payload(nil, 'x', 'y', 'z'), nil, 'no PAIR_HOME yields no payload')
end

-- capture_record carries its baselines so a row stays legible on its own.
do
  local r = M.capture_record(1757000000, 'slow', 'fast', { pipe_hop_ms = 0.9 })
  eq(r.at, 1757000000, 'timestamp recorded')
  eq(r.editor, 'fast', 'verdict recorded')
  eq(r.probes.pipe_hop_ms, 0.9, 'probes recorded')
  ok(r.baselines.pipe_hop_ms ~= nil, 'baselines travel with the row')
  eq(M.capture_record(1, nil, 'unknown').note, nil, 'an absent note stays absent')
end


-- The sidecar split: raw samples out of the prompt, everything else kept.
do
  local raw = table.concat({
    '# pair perf capture', 'load=2.0', '',
    '## conditions', 'cpu_idle_pct=90',
    '## sample_a', '### procs', '1\t100\t50\tsh', '2\t100\t50\tzsh',
    '## sample_b', '### procs', '1\t102\t50\tsh',
    '## probes', 'pipe_hop_ms=0.007',
  }, '\n')
  local lean = M.strip_samples(raw)
  ok(lean:find('cpu_idle_pct=90', 1, true) ~= nil, 'conditions survive the strip')
  ok(lean:find('pipe_hop_ms', 1, true) ~= nil, 'probes survive the strip')
  ok(lean:find('## probes', 1, true) ~= nil, 'the section after the samples is not swallowed')
  ok(lean:find('sh', 1, true) == nil, 'sample rows are gone')
  ok(#lean < #raw, 'the stripped report is smaller')
  eq(M.strip_samples(nil), '', 'nil input is empty, not an error')
end

-- format_delta: the churn counts are the spawn-storm signature, so they are
-- always printed even when no process clears the 1% floor.
do
  local d = { rates = {
      { pid = '7', comm = 'go', cpu_pct = 55.0 },
      { pid = '8', comm = 'quiet', cpu_pct = 0.2 },
    }, started = 9, vanished = 2, reused = 1, unmeasured = 0, rows_a = 800, rows_b = 807 }
  local text = M.format_delta(d)
  ok(text:find('55.0%%') ~= nil, 'a busy process is listed')
  ok(text:find('quiet', 1, true) == nil, 'sub-1%% noise is filtered out')
  ok(text:find('9 started', 1, true) ~= nil, 'churn is reported -- a large started count IS a spawn storm')
  ok(M.format_delta(nil):find('n/a', 1, true) ~= nil, 'an unjoinable capture says so')
  local idle = M.format_delta({ rates = {}, started = 0, vanished = 0, reused = 0, unmeasured = 0, rows_a = 5, rows_b = 5 })
  ok(idle:find('idle across the window', 1, true) ~= nil, 'an idle window says so rather than showing nothing')
end

-- headline + the sidecar's position in the payload.
--
-- These pin the truncation-survival design (#211): the send path drops interior
-- chunks, so the path to the full capture must sit in the HEAD -- ahead of the
-- note -- and the prompt must stay small enough that little of it is exposed.
do
  local compact = table.concat({
    '# pair perf capture',
    'captured_at=2026-09-07T10:00:00-0700',
    'window_seconds=2',
    'budget_seconds=6',
    'host_cores=12',
    'load=9.51/8.10/7.44',
    'memory_pressure_level=1',
    'process_count=812',
    'cpu_idle_pct=71.2',
    'windowserver_cpu_pct=7.0',
    'pair_family_procs=14',
    'build_procs=0',
    'swapins_per_s=0.0',
    'pipe_hop_ms=0.007',
    'pipe_hop_p90=0.009',
    'zellij_action_ms=13.5',
    'elapsed_seconds=6',
  }, '\n')
  local rates = table.concat({
    'per-process CPU over the 2s window:',
    '   55.0%  go',
    '    4.0%  nvim',
    'churn: 9 started, 2 vanished',
  }, '\n')
  local h = M.headline(compact, rates)

  ok(h:find('windowserver_cpu_pct=7.0', 1, true) ~= nil, 'the headline keeps the numbers that matter')
  ok(h:find('load=9.51', 1, true) ~= nil, 'load survives')
  ok(h:find('zellij_action_ms=13.5', 1, true) ~= nil, 'the probes survive -- they are the point of the capture')
  ok(h:find('churn: 9 started', 1, true) ~= nil, 'churn survives; a spawn storm is visible without opening the file')
  ok(h:find('55.0%%') ~= nil, 'the top CPU consumer survives')
  ok(h:find('budget_seconds', 1, true) == nil, 'bookkeeping keys are dropped')
  ok(h:find('memory_pressure_level', 1, true) == nil, 'second-tier keys are dropped -- they are in the file')
  ok(#h < #compact, 'the headline is smaller than the compact report it summarizes')

  local body = M.perf_payload('/tmp/home', 'typing went slow', 'fast (12.0ms)', h, '/tmp/cap.txt')
  local at = body:find('/tmp/cap.txt', 1, true)
  ok(at ~= nil, 'the sidecar path is in the payload')
  ok(at < body:find('typing went slow', 1, true),
    'the path precedes even the note: the head is what survives a truncated send')
  ok(at < 400, 'the path is in the first few hundred bytes, well inside any surviving head')
  ok(#body < 1800, 'the whole payload stays small: ' .. #body .. ' bytes')

  local nofile = M.perf_payload('/tmp/home', 'the note', 'fast', h, nil)
  ok(nofile:find('the note', 1, true) ~= nil, 'a payload without a sidecar still renders')
  ok(nofile:find('named above', 1, true) == nil,
    'with no sidecar the prompt does not point at a file that was never written')
end

-- A process name comes from ps and can hold anything. ^N is SO: it switches the
-- terminal to the alternate character set and garbles every following line.
do
  local text = M.format_delta({ rates = {
      { pid = '1', comm = '/Applications/\014WhatsApp.app/x', cpu_pct = 3.0 },
    }, started = 0, vanished = 0, reused = 0, unmeasured = 0, rows_a = 5, rows_b = 5 })
  ok(text:find('\014') == nil, 'a control byte in a process name never reaches the terminal')
  ok(text:find('WhatsApp', 1, true) ~= nil, 'the readable part of the name survives')
end

-- C1's second half: a DEGRADED capture must still render every headline key.
-- perf.sh used to report a failed collector under a different key than its
-- success form (`swap=n/a` vs `swapins_per_s=`, `probes=n/a` vs
-- `pipe_hop_ms=`), so the allowlist dropped those rows and the prompt simply
-- had no probe line -- indistinguishable from a tool with no such section, and
-- the exact opposite of the n/a-is-not-zero rule. perf.sh now renders failures
-- under the success key; this pins the consumer half.
do
  local degraded = {}
  for _, k in ipairs(M.HEADLINE_KEYS) do
    degraded[#degraded + 1] = k .. '=n/a (collector failed)'
  end
  local h = M.headline(table.concat(degraded, '\n'), '')
  for _, k in ipairs(M.HEADLINE_KEYS) do
    ok(h:find(k .. '=n/a', 1, true) ~= nil,
      'a degraded capture still renders ' .. k .. ' as n/a rather than omitting it')
  end
end

-- probes_from is the perf.sh -> rolling-log contract. `(%w+_ms)` looks right
-- and is not: Lua's %w excludes `_`, so it captured `hop_ms` from
-- `pipe_hop_ms`. Driven by the key names perf.sh actually emits.
do
  local cap = table.concat({
    '# pair perf capture',
    'pipe_hop_ms=0.007',
    'pipe_hop_p90=0.009',
    'fork_exec_ms=1.506',
    'zellij_action_ms=n/a (zellij not on PATH)',
  }, '\n')
  local p = M.probes_from(cap)
  ok(p.pipe_hop_ms == 0.007, 'the full key survives; %w would have yielded hop_ms')
  ok(p.fork_exec_ms == 1.506, 'fork_exec_ms, not exec_ms')
  ok(p.zellij_action_ms == nil, 'an n/a probe is omitted, never coerced to 0')
  ok(p.hop_ms == nil and p.exec_ms == nil, 'no truncated key is produced')
  for k in pairs(p) do
    ok(M.BASELINES[k] ~= nil, 'every probe key has a baseline: ' .. k)
  end
  ok(next(M.probes_from(nil)) == nil, 'nil input yields no probes rather than erroring')
end

-- The verdict must never say `fast` on a leg it did not measure: `editor: fast`
-- is what doctor/SKILL.md tells the reader to exclude #201/#203 on.
do
  ok(M.verdict(1, 1, nil) == 'unknown', 'an unmeasured leg forbids fast')
  ok(M.verdict(1, 1, 1) == 'fast', 'all legs measured and quick is fast')
  ok(M.verdict(nil, nil, 99) == 'slow', 'one slow leg proves slow even alone')
end

if fails > 0 then
  io.stderr:write(string.format('\n%d failure(s)\n', fails))
  os.exit(1)
end
print('all doctor.lua tests passed')
