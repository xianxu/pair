-- nvim/review/resolve.lua — pure §5 accept/reject resolution for 🤖 marker chains
-- (issue #66 M4b). The human resolves the agent's suggestions one at a time: a
-- `🤖…{…}` chain becomes plain text per the review-convention §5 table (mirrored in
-- the xx-fix SKILL). PURE — takes a parsed marker (from markers.parse_markers:
-- {quoted, strike, sections}) + an action, returns the replacement text for the
-- marker's raw span, or nil for a no-op ("reject → same": the marker is kept).
--
-- §5 table:
--   🤖[H]            accept → ''      reject → same
--   🤖<X>[H]         accept → X       reject → same
--   🤖{R}            accept → R       reject → ''
--   🤖[H]{R}/{R}[H]  accept → ''      reject → same
--   🤖~D~            accept → ''      reject → D
--   🤖~D~{N}         accept → N       reject → D
--   🤖~D~[N]         accept → N       reject → D
--   longer []{} chain accept → ''     reject → same
local M = {}

-- @param marker table  a parse_markers record: { quoted?, strike?, sections }
-- @param action string 'accept' | 'reject'
-- @return string|nil   replacement for the marker's raw span; nil = no-op (same)
function M.resolve(marker, action)
  local sections = marker.sections or {}
  local last = sections[#sections]

  -- Strike (deletion / replacement proposals): 🤖~D~ / ~D~{N} / ~D~[N].
  if marker.strike then
    if action == 'reject' then return marker.strike.text end -- keep D
    return last and last.text or '' -- accept: the new text, or delete (bare ~D~)
  end

  -- Quoted instruction: 🤖<X>[H] — accept keeps X, reject leaves the marker.
  if marker.quoted then
    if action == 'accept' then return marker.quoted.text end
    return nil -- reject → same
  end

  -- Lone agent suggestion: 🤖{R} — accept applies R, reject discards it.
  if #sections == 1 and last and last.type == 'agent' then
    return action == 'accept' and last.text or ''
  end

  -- 🤖[H], mixed chains, longer chains: accept discards (surrounding text
  -- untouched), reject keeps the marker.
  return action == 'accept' and '' or nil
end

return M
