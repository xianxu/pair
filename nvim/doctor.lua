-- nvim/doctor.lua — pure payload builder for the `:PairDoctor` command
-- (issue #000048). No vim API here, so it runs under `nvim -l` for tests
-- (`make test-lua`, in Makefile.local). init.lua dofile's it and wraps the IO
-- (read $PAIR_HOME, hand the instruction to the agent pane via send_to_agent).
--
-- Why a command instead of a Claude skill: pair is agent-agnostic and runs in
-- arbitrary project dirs. A `.claude/skills/` entry only works under claude, and
-- a relative `doctor/doctor.sh` only resolves when the agent's cwd is the pair
-- checkout. nvim is the one substrate every agent shares, and it knows
-- $PAIR_HOME — so it can hand ANY agent a $PAIR_HOME-absolute instruction. The
-- procedure itself stays single-sourced in doctor/SKILL.md; this is the pointer.
local M = {}

-- payload(pair_home) → the instruction string with $PAIR_HOME-absolute paths
-- substituted in (NOT a literal `$PAIR_HOME` — the agent must not depend on its
-- shell to expand it), or nil when pair_home is missing/empty (the caller
-- notifies instead of sending a broken path).
function M.payload(pair_home)
  if not pair_home or pair_home == '' then return nil end
  local h = pair_home:gsub('/+$', '') -- trim trailing slash(es)
  return 'Run `bash ' .. h .. '/doctor/doctor.sh` (it reads this pair session\'s '
    .. 'adaptation flight recorder), then follow `' .. h .. '/doctor/SKILL.md`: '
    .. 'check the emitter-health line first, interpret any drift findings against `'
    .. h .. '/atlas/how-to-bring-up-a-new-harness-cli.md` §3, and propose concrete '
    .. 'matcher fixes for me to approve — don\'t edit anything silently.'
end



-- ---------------------------------------------------------------------------
-- Performance capture (#208). Everything below is pure: `nvim -l` runs it, and
-- `doctor/perf.sh` supplies the raw text these functions interpret.
-- ---------------------------------------------------------------------------

-- ps TIME is "MM:SS.ss" or "HH:MM:SS"; etime is "MM:SS", "HH:MM:SS" or
-- "D-HH:MM:SS". Both become seconds. A value we cannot read returns nil rather
-- than 0 -- 0 is a real number that would silently mean "used no CPU".
function M.parse_duration(text)
  if type(text) ~= 'string' then return nil end
  local days, rest = text:match('^(%d+)%-(.+)$')
  if days then text = rest end
  -- Validate BEFORE accumulating. The previous version pushed tonumber()'s nil
  -- into the table and checked for it inside the arithmetic loop, where Lua
  -- errors on nil*60 before any guard runs -- the guard was unreachable and
  -- malformed input either crashed or produced a fabricated number.
  local parts = {}
  for piece in text:gmatch('[^:]+') do
    local n = tonumber(piece)
    if not n then return nil end
    parts[#parts + 1] = n
  end
  if #parts == 0 then return nil end
  -- The h:m:s accumulator must stay SEPARATE from the day count: seeding it
  -- with days*86400 makes the *60 below multiply the days too.
  local hms = 0
  for _, v in ipairs(parts) do hms = hms * 60 + v end
  return hms + (days and tonumber(days) * 86400 or 0)
end

-- delta joins two samples into per-process CPU rates over `window` seconds.
--
-- This function exists BECAUSE the join is error-prone, which is why it is here
-- and not in perf.sh. Three cases the naive version gets wrong, each counted
-- rather than silently dropped:
--
--   vanished  a pid in A and not B. Under a spawn storm these are exactly the
--             processes that CHARACTERISE the storm, so losing them loses the
--             signal. Counted.
--   started   a pid in B and not A. No rate is computable; listing it as 0
--             would read as "used no CPU", which is a different claim.
--   reused    a pid in both whose etime went DOWN -- a different process
--             inherited the number. Its "rate" would be nonsense, so it is
--             treated as started.
--
-- Returns { rates = {{pid, comm, cpu_pct, rss_kb}, ...} sorted desc,
--           vanished, started, reused, rows_a, rows_b }.
function M.delta(a, b, window)
  local out = { rates = {}, vanished = 0, started = 0, reused = 0,
                unmeasured = 0, rows_a = 0, rows_b = 0 }
  window = tonumber(window)
  if type(a) ~= 'table' or type(b) ~= 'table' or not window or window <= 0 then
    return out
  end
  for _ in pairs(a.procs or {}) do out.rows_a = out.rows_a + 1 end
  for _ in pairs(b.procs or {}) do out.rows_b = out.rows_b + 1 end

  for pid, pa in pairs(a.procs or {}) do
    local pb = (b.procs or {})[pid]
    if not pb then
      out.vanished = out.vanished + 1
    elseif pa.etime and pb.etime and pb.etime < pa.etime then
      out.reused = out.reused + 1   -- a different process, not a negative rate
    else
      local ca, cb = (a.cpu or {})[pid], (b.cpu or {})[pid]
      if ca and cb then
        out.rates[#out.rates + 1] = {
          pid = pid, comm = pb.comm or pa.comm,
          cpu_pct = ((cb - ca) / window) * 100,
          rss_kb = pb.rss,
        }
      else
        -- Alive in both samples but one cputime row is missing (a truncated or
        -- racing `ps`). Counted, never silently dropped: the contract is that
        -- every pid lands in exactly one bucket, and a pid that reaches none is
        -- invisible to the reader.
        out.unmeasured = out.unmeasured + 1
      end
    end
  end
  for pid in pairs(b.procs or {}) do
    if not (a.procs or {})[pid] then out.started = out.started + 1 end
  end
  table.sort(out.rates, function(x, y)
    if x.cpu_pct ~= y.cpu_pct then return x.cpu_pct > y.cpu_pct end
    return x.pid < y.pid           -- stable: equal rates must not reorder
  end)
  return out
end

-- The editor-vs-environment discriminator, and the reason this capture is worth
-- taking from inside nvim at all.
--
-- 16 ms is one frame at 60 Hz -- the floor of human perceptibility. An editor
-- that handles a keystroke inside a frame cannot be what the operator is
-- feeling, so the cause is at or above the terminal and the whole scheduling
-- family is excluded for that symptom.
M.FRAME_MS = 16

-- Returns 'fast', 'slow', or 'unknown'. Absent timings are 'unknown', NOT
-- 'fast': claiming the editor is healthy on no evidence is the same class of
-- error as a probe reporting a failed command as excellent latency, and it
-- would send a reader hunting the environment on the strength of a measurement
-- that never happened.
-- Asymmetric on purpose: a PARTIAL measurement can prove 'slow' but cannot
-- clear the editor. One timing over the frame budget is positive evidence of
-- slowness; one timing under it says nothing about the measurement that is
-- missing, so it yields 'unknown' rather than a confident 'fast'. Rendering a
-- half-absent measurement as a full in-domain verdict is the same defect as a
-- probe reporting a failed command as excellent latency, at a different arity.
-- Variadic over the legs that were timed, because the leg set grew: input,
-- redraw, and the completion chain. The asymmetry is the whole point and holds
-- at any arity -- ONE slow leg proves slow, but `fast` requires EVERY leg to
-- have been measured. A missing leg yields `unknown`, never `fast`: partial
-- evidence can prove slow and never proves fast, and `editor: fast` is what
-- doctor/SKILL.md tells the reader to exclude #201/#203 on.
function M.verdict(...)
  -- select, not `{...}`: a table constructor with a leading nil has an
  -- unreliable length and ipairs stops at the first hole -- which is precisely
  -- the dropped-leg case this function exists to notice.
  local total = select('#', ...)
  if total == 0 then return 'unknown' end
  local measured = 0
  for i = 1, total do
    local n = tonumber((select(i, ...)))
    if n then
      measured = measured + 1
      if n >= M.FRAME_MS then return 'slow' end
    end
  end
  if measured < total then return 'unknown' end
  return 'fast'
end

-- The draft buffer's text as the operator's note. Blank returns nil rather than
-- '' -- an empty note would read as "the operator said nothing was wrong",
-- which is a claim, where absence is not.
function M.note_from_lines(lines)
  if type(lines) ~= 'table' then return nil end
  local text = table.concat(lines, '\n')
  if text:match('^%s*$') then return nil end
  return (text:gsub('^%s+', ''):gsub('%s+$', ''))
end

-- parse_samples turns perf.sh's raw report into the two structures delta joins.
--
-- This function is the CONTRACT between perf.sh and delta. Before it existed,
-- delta was tested against hand-written literals asserted by the same mental
-- model that wrote the code, and nothing pinned that perf.sh actually emits what
-- delta expects -- so a change to either could pass every test and break the
-- capture. The fixture in nvim/fixtures/ is real captured output.
-- Returns sample_a, sample_b, window_seconds. A sample the capture SHED is
-- returned as nil, never as an empty-but-present table: delta joining against
-- an empty sample reports every process as vanished, which is a catastrophic
-- reading invented by the budget path rather than observed.
function M.parse_samples(text)
  if type(text) ~= 'string' or text == '' then return nil, nil, nil end
  local samples, current, section, window = {}, nil, nil, nil
  for line in (text .. '\n'):gmatch('([^\n]*)\n') do
    local head = line:match('^## (sample_%a+)$')
    if head then
      current = { procs = {}, cpu = {}, skipped = false }
      samples[head] = current
      section = nil
    elseif current and line:match('^skipped=') then
      current.skipped = true
    elseif not current and line:match('^window_seconds=') then
      window = tonumber(line:match('^window_seconds=(%S+)'))
    elseif line:match('^## ') then
      current, section = nil, nil
    elseif current and line == '### cputime' then
      section = 'cpu'
    elseif current and line == '### procs' then
      section = 'procs'
    elseif current and section == 'cpu' then
      local pid, t = line:match('^(%d+)\t(%S+)$')
      if pid then current.cpu[pid] = M.parse_duration(t) end
    elseif current and section == 'procs' then
      local pid, et, rss, comm = line:match('^(%d+)\t(%S+)\t(%d+)\t(.*)$')
      if pid then
        current.procs[pid] = {
          etime = M.parse_duration(et), rss = tonumber(rss), comm = comm,
        }
      end
    end
  end
  local function usable(sample)
    if not sample or sample.skipped then return nil end
    if next(sample.procs) == nil then return nil end
    return sample
  end
  return usable(samples.sample_a), usable(samples.sample_b), window
end

-- perf_payload assembles what the agent receives: the operator's note, what
-- nvim measured about ITSELF, the environment snapshot, and the drift pointer.
--
-- The note LEADS, deliberately. An agent reading this needs the operator's
-- symptom before the numbers, or it explains whatever is largest in the report
-- rather than what was actually reported -- which is how the 2026-09-06
-- investigation spent a session on a machine that was fine.
--
-- The drift instruction is embedded VERBATIM (M.payload), so #48's procedure
-- cannot drift by being paraphrased here.
function M.perf_payload(pair_home, note, editor, env, sidecar)
  local drift = M.payload(pair_home)
  if not drift then return nil end
  local out = { 'Diagnose a performance slowdown on this workbench.', '' }
  if sidecar and sidecar ~= '' then
    -- FIRST, deliberately. The send path drops interior chunks (see headline),
    -- and the head is what survives -- so a truncated message must still carry
    -- the path to everything.
    out[#out + 1] = 'FULL capture (read this if anything below looks cut off): ' .. sidecar
    out[#out + 1] = ''
  end
  if note and note ~= '' then
    out[#out + 1] = 'What the operator reported, in their words:'
    out[#out + 1] = ''
    out[#out + 1] = note
  else
    out[#out + 1] = 'The operator left no note; go on the measurements alone.'
  end
  out[#out + 1] = ''
  out[#out + 1] = 'What nvim measured about ITSELF (the editor-vs-environment'
  out[#out + 1] = 'discriminator). `fast` requires EVERY leg below to have been'
  out[#out + 1] = 'measured and to be inside one frame; a leg rendered n/a forces'
  out[#out + 1] = '`unknown`, and NO exclusion may be drawn from `unknown`. On a'
  out[#out + 1] = 'genuine `fast` while typing feels slow, the cause is at or above'
  out[#out + 1] = 'the terminal and the scheduling family (pair#201/#203) is'
  out[#out + 1] = 'excluded for this symptom:'
  out[#out + 1] = ''
  out[#out + 1] = editor or 'editor: n/a (self-timing did not run)'
  out[#out + 1] = ''
  if sidecar and sidecar ~= '' then
    out[#out + 1] = 'Environment headline (the full snapshot is in the file named above):'
  else
    -- No file was written, so there is nothing to point at; saying otherwise
    -- would send a reader hunting for a path that does not exist.
    out[#out + 1] = 'Environment headline (the full snapshot could not be saved):'
  end
  out[#out + 1] = ''
  out[#out + 1] = env or 'n/a (capture did not run)'
  out[#out + 1] = ''
  out[#out + 1] = 'Read `n/a` as "not measured", never as zero. Then, for harness'
  out[#out + 1] = 'drift specifically: ' .. drift
  return table.concat(out, '\n')
end

-- capture_record is one row of the rolling log: enough to compare a later
-- reading against, without needing this file to interpret it. The baselines
-- travel WITH the row for that reason -- a row read a year from now must be
-- legible on its own.
function M.capture_record(now, note, editor, probes)
  return {
    at = now,
    note = note,               -- nil when the operator left none
    editor = editor,           -- 'fast' | 'slow' | 'unknown'
    probes = probes or {},
    -- Keyed off PROBE_KEYS so a row's baselines can never name a probe the
    -- reading half does not, which is how `hop_ms` vs `pipe_hop_ms` went unseen.
    baselines = M.BASELINES,
  }
end

-- strip_samples returns the report WITHOUT the two raw sample blocks.
--
-- The blocks are ~3,500 lines of cumulative ps output. Embedding them in the
-- agent's prompt (which the first version did) buries the twenty lines that
-- matter and costs a fortune in context for data nobody reads linearly. The
-- conditions, fleet, swap, disk and probe lines are compact and are what a
-- reader actually uses, so they stay inline; the raw goes to a sidecar file the
-- agent can open if it needs to drill in -- the same shape `doctor.sh` uses for
-- the flight recorder.
function M.strip_samples(text)
  if type(text) ~= 'string' then return '' end
  local out, skipping = {}, false
  for line in (text .. '\n'):gmatch('([^\n]*)\n') do
    if line:match('^## sample_%a+$') then
      skipping = true
    elseif skipping and line:match('^## ') then
      skipping = false
    end
    if not skipping then out[#out + 1] = line end
  end
  return table.concat(out, '\n')
end

-- format_delta renders the join as the handful of lines a reader needs: who is
-- actually burning CPU over the window, and the accounting that says whether the
-- picture is trustworthy.
-- A process name reaches this report straight from `ps` and is attacker- and
-- accident-controlled: any app can be named with control bytes in it. Observed
-- 2026-09-07, WhatsApp's argv rendered as `\040^NWhatsApp` -- and ^N is SO,
-- which switches a terminal to the alternate character set and garbles every
-- line after it. The report is written INTO a terminal, so escapes get stripped
-- rather than passed through.
local function safe_comm(c)
  if not c or c == '' then return '?' end
  return (c:gsub('%c', '?'):gsub('\\%d%d%d', '?'))
end

function M.format_delta(d, limit)
  if not d then return 'per-process rates: n/a (samples could not be joined)' end
  limit = limit or 10
  local out = {
    string.format('per-process CPU over the window (%d rows sampled, %d compared):',
      d.rows_a, #d.rates),
  }
  local shown = 0
  for _, r in ipairs(d.rates) do
    if shown >= limit then break end
    -- Below 1% is noise on a 12-core host and would push the interesting rows
    -- off the list.
    if r.cpu_pct >= 1.0 then
      out[#out + 1] = string.format('  %7.1f%%  %-6s  %s', r.cpu_pct, tostring(r.pid), safe_comm(r.comm))
      shown = shown + 1
    end
  end
  if shown == 0 then
    out[#out + 1] = '  (nothing above 1% -- the machine was idle across the window)'
  end
  -- These are not bookkeeping: a large `started` IS a spawn storm, which is the
  -- signature #203 cares about and which no single-sample view can show.
  out[#out + 1] = string.format('churn: %d started, %d vanished, %d reused-pid, %d unmeasured',
    d.started, d.vanished, d.reused, d.unmeasured)
  return table.concat(out, '\n')
end

-- headline pulls the handful of numbers worth carrying in the prompt itself.
--
-- Everything else lives in the sidecar. This is not only about size: the send
-- path drops chunks intermittently (measured 2026-09-07 -- exactly 1,025 bytes
-- vanished from the middle of a 2,447-byte payload while head and tail arrived,
-- and a 180KB payload had succeeded minutes earlier, so size does not predict
-- it). A short prompt is a smaller target, and pairing it with a path near the
-- TOP -- the part that survives -- means a truncated message still tells the
-- reader where the full capture is.
function M.headline(compact, delta_text)
  local want = {}
  for _, k in ipairs(M.HEADLINE_KEYS) do want[k] = true end
  local out = {}
  for line in ((compact or '') .. '\n'):gmatch('([^\n]*)\n') do
    local key = line:match('^([%w_]+)=')
    if key and want[key] then out[#out + 1] = line end
  end
  -- The rates and churn are the point of the capture; keep the top few.
  if delta_text and delta_text ~= '' then
    local kept, n = {}, 0
    for line in (delta_text .. '\n'):gmatch('([^\n]*)\n') do
      if line:match('^churn:') or line:match('^per%-process') then
        kept[#kept + 1] = line
      elseif line:match('^%s+%d') and n < 5 then
        kept[#kept + 1] = line; n = n + 1
      end
    end
    out[#out + 1] = ''
    out[#out + 1] = table.concat(kept, '\n')
  end
  return table.concat(out, '\n')
end

-- ONE declaration of the keys perf.sh emits, consumed by every reader.
--
-- These were restated in four hand-maintained places -- perf.sh's kv calls,
-- headline's allowlist, capture_record's baselines, and a regex in init.lua --
-- and three of the four had already drifted out of agreement. The regex was
-- silently wrong (see probes_from). A key set that lives in one place cannot
-- drift; one that lives in four always has.
M.PROBE_KEYS = { 'pipe_hop_ms', 'fork_exec_ms', 'zellij_action_ms' }

-- Measured on a healthy host, and written into every row so a row stays legible
-- on its own -- a number means nothing without what it is being compared to.
M.BASELINES = { pipe_hop_ms = 0.007, fork_exec_ms = 1.5, zellij_action_ms = 13 }

-- What the prompt carries. Everything else is in the sidecar. perf.sh renders a
-- FAILED collector under these same keys (`swapins_per_s=n/a (...)`), so this
-- list needs no knowledge of failure key names -- an earlier version needed
-- exactly that and silently dropped the probe and swap lines from every
-- degraded capture, which reads as "this tool has no such section".
M.HEADLINE_KEYS = {
  'load', 'cpu_idle_pct', 'windowserver_cpu_pct',
  'pair_family_procs', 'build_procs', 'swapins_per_s',
  'pipe_hop_ms', 'fork_exec_ms', 'zellij_action_ms', 'elapsed_seconds',
}

-- probes_from is the perf.sh -> rolling-log contract, and it lives HERE because
-- the caller that had its own copy got it wrong: `(%w+_ms)=` looks right and is
-- not -- Lua's %w excludes `_`, so the capture backtracks past the prefix and
-- `pipe_hop_ms` is recorded as `hop_ms`. Every row written that way carried
-- probe keys matching neither perf.sh's names nor the row's own baselines
-- table, and the unit test asserting the INTENDED key passed the whole time
-- because it never ran the caller.
--
-- A numeric reading is returned as a number; a collector that rendered `n/a` is
-- omitted, never coerced to 0.
function M.probes_from(text)
  local out = {}
  for _, key in ipairs(M.PROBE_KEYS) do
    local v = (text or ''):match('\n' .. key .. '=([^\n]*)')
      or (text or ''):match('^' .. key .. '=([^\n]*)')
    local n = v and tonumber(v)
    if n then out[key] = n end
  end
  return out
end

return M
