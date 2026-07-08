-- nvim/review/spinner.lua — pure helpers for the review statusline's agent-working
-- spinner (#66 M4a'). The pane sets an "awaiting since" time when it pokes the agent
-- (Alt+Return → expect a handoff) and clears it when the handoff lands; this renders
-- the braille frame + compact elapsed as the statusline's leading cell. PURE.
local M = {}

M.frames = { '⣾', '⣽', '⣻', '⢿', '⡿', '⣟', '⣯', '⣷' }

-- compact elapsed: 45s / 2m 0s / 5m 10s / 1h 2m.
function M.elapsed(secs)
  if not secs or secs < 0 then secs = 0 end
  if secs < 60 then return secs .. 's' end
  if secs < 3600 then
    local minutes = math.floor(secs / 60)
    local seconds = secs % 60
    return minutes .. 'm ' .. seconds .. 's'
  end
  local hours = math.floor(secs / 3600)
  local minutes = math.floor((secs % 3600) / 60)
  return hours .. 'h ' .. minutes .. 'm'
end

-- The statusline's leading cell while awaiting the agent: braille frame + elapsed
-- (+ a trailing space before "Review"), e.g. "⠹ 45s ". '' when idle, so "Review …"
-- sits at the start. `tick` advances the frame; `now`/`awaiting_since` are seconds.
function M.cell(awaiting_since, now, tick)
  if not awaiting_since then return '' end
  local frame = M.frames[((tick or 0) % #M.frames) + 1]
  return frame .. ' ' .. M.elapsed((now or 0) - awaiting_since) .. ' '
end

return M
