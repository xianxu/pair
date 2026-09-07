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

function M.verdict(insert_ms, redraw_ms)
  local worst = math.max(tonumber(insert_ms) or 0, tonumber(redraw_ms) or 0)
  if worst >= M.FRAME_MS then return 'slow' end
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

return M
