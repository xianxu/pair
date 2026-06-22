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

-- After the human finished their turn (the nvim saved — but did NOT git-commit;
-- the agent commits the human round). "finished", not "committed" — precise.
function M.human_finished(file)
  return string.format('finished my edits to %s — please review in Copy Edit posture; '
    .. 'resolve 🤖[] comments as edits when possible, or punt explicitly when not', file)
end

function M.ship_requested(file)
  return string.format('ship %s — run docflow ship for the active review branch; the agent owns git', file)
end

function M.mode_switch(file, mode, instruction, label)
  local suffix = ''
  if instruction and instruction ~= '' then
    suffix = ' with instruction: ' .. instruction
  end
  return string.format('switch review mode to %s for %s%s — acknowledge the mode, write the review mode seam, then continue only when I send the next turn',
    label or mode, file, suffix)
end

-- Sent ONCE when the review pane opens — the missing review-START signal (M4a
-- smoke: a chat "please review" carried no workbench cue, so the agent fell back
-- to a standalone summarize-and-ask). Establishes context WITHOUT triggering a
-- review (no branch/commit until the operator actually asks): when asked, the
-- agent must use the xx-fix Pair-review-workbench protocol, not file-write.
function M.review_opened(file)
  return string.format(
    'Review workbench open on %s. When I ask you to review this doc, use the xx-fix '
    .. '"Pair review workbench" protocol: propose {old,occurrence,new,explain} records '
    .. 'to the handoff and own the git — do NOT edit the file in place or summarize '
    .. 'edits and ask to apply (that bypasses the pane). Default to Copy Edit posture; '
    .. 'resolve 🤖[] comments as edits when possible, or punt explicitly when not. Reply "ready".', file)
end

return M
