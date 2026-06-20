-- nvim/review/poke_bodies.lua — pure builders for the commit-signal pokes the
-- review nvim sends the agent (issue #66 M4a). The nvim writes no git; instead it
-- pokes these NL signals and the agent commits the round (reading the
-- landed-artifact for the agent round's body). ONE source for the wording
-- (review-protocol.md seam #3) so the nvim and the tests assert the same strings.
-- PURE (string-only); standalone like record.lua so the colocated test runs under
-- `make test-lua` without dofile-ing the IO orchestrator.
local M = {}

-- After the nvim applied an agent handoff: `applied` records landed, `dropped`
-- did not (the "(M dropped)" segment is omitted when none were dropped).
function M.agent_applied(applied, dropped, file)
  local drop = (dropped and dropped > 0) and string.format(' (%d dropped)', dropped) or ''
  return string.format('applied %d edit(s)%s to %s — commit the agent round', applied, drop, file)
end

-- After the human finished their turn (the nvim saved the incoming edits).
function M.human_committed(file)
  return string.format('committed my edits to %s — please review', file)
end

return M
