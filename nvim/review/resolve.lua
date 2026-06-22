-- nvim/review/resolve.lua — pure §5 accept/reject resolution for 🤖 marker chains
-- (issue #66 M4b). The human resolves the agent's suggestions one at a time: a
-- `🤖…{…}` chain becomes plain text per the review-convention §5 table (mirrored in
-- the xx-fix SKILL). PURE — takes a parsed marker (from markers.parse_markers:
-- {quoted, strike, sections}) + an action, returns the replacement text for the
-- marker's raw span.
--
-- §5 table:
--   🤖[H]            accept → ''      reject → ''
--   🤖<X>[H]         accept → X       reject → X
--   🤖<X>{Y}         accept → Y       reject → X
--   🤖{R}            accept → R       reject → ''
--   🤖[H]{R}/{R}[H]  accept → ''      reject → ''
--   🤖~D~            accept → ''      reject → D
--   🤖~D~{N}         accept → N       reject → D
--   🤖~D~[N]         accept → N       reject → D
--   longer []{} chain accept → ''     reject → ''
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

  -- Quoted instruction/comment/replacement:
  --   🤖<X>{Y} accept applies the agent replacement; reject keeps X.
  --   🤖<X>[H] accept/reject both keep X while removing the markup.
  if marker.quoted then
    if action == 'accept' and #sections == 1 and last and last.type == 'agent' then
      return last.text
    end
    return marker.quoted.text
  end

  -- Lone agent suggestion: 🤖{R} — accept applies R, reject discards it.
  if #sections == 1 and last and last.type == 'agent' then
    return action == 'accept' and last.text or ''
  end

  -- 🤖[H], mixed chains, longer chains: remove the markup; there is no plain-text
  -- body to preserve.
  return ''
end

return M
