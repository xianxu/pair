-- nvim/review/gate.lua — the pure apply-gate (issue #89 M3). Decides whether a
-- landed agent round applies NOW or DEFERS until the human reaches a safe point.
-- Deferral, not locking: nothing about editing is ever disabled — we just wait to
-- apply until the human isn't mid-edit on the pane. Pure: string/bool in, string
-- out; no vim API (mode is passed in from vim.fn.mode() by the caller).
local M = {}

-- decide_apply(v0, v1, focused, mode) → 'apply' | 'defer'
--   v1 == v0     → apply (nothing changed since the agent reviewed)
--   not focused  → apply (human is in another pane)
--   mode == 'n'  → apply (on the pane, normal mode — not editing)
--   otherwise    → defer (focused + changed + mid-edit: i/R/v/V/^V/s/…)
function M.decide_apply(v0, v1, focused, mode)
  if v1 == v0 then return 'apply' end
  if not focused then return 'apply' end
  if mode == 'n' then return 'apply' end
  return 'defer'
end

return M
