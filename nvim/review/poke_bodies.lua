-- nvim/review/poke_bodies.lua — pure builders for the commit-signal pokes the
-- review nvim sends the agent (issue #66 M4a). The nvim writes no git; instead it
-- pokes these NL signals and the agent commits the round (reading the
-- landed-artifact for the agent round's body). ONE source for the wording
-- (review-protocol.md seam #3) so the nvim and the tests assert the same strings.
-- PURE (string-only); standalone like record.lua so the colocated test runs under
-- `make test-lua` without dofile-ing the IO orchestrator.
local M = {}

-- After the nvim applied an agent handoff: `applied` records landed, `dropped`
-- did not, `conflicts` became 🤖<…>[reconcile] markers (#89). The "(M dropped)" /
-- "(K to reconcile)" segments are omitted when their count is zero.
function M.agent_applied(applied, dropped, file, conflicts)
  local drop = (dropped and dropped > 0) and string.format(' (%d dropped)', dropped) or ''
  local conf = (conflicts and conflicts > 0) and string.format(' (%d to reconcile)', conflicts) or ''
  return string.format('applied %d edit(s)%s%s to %s — commit the agent round', applied, drop, conf, file)
end

-- After the human finished their turn (the nvim saved — but did NOT git-commit;
-- the agent commits the human round). "finished", not "committed" — precise.
function M.human_finished(file, mode, instruction, label)
  label = label or 'Edit'
  local suffix = ''
  if instruction and instruction ~= '' then
    suffix = '; instruction: ' .. instruction
  end
  return string.format('finished my edits to %s — please review in %s posture%s',
    file, label, suffix)
end

function M.ship_requested(file)
  return string.format('ship %s — run docflow ship for the active review branch; the agent owns git', file)
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
    .. 'edits and ask to apply (that bypasses the pane). Default to Edit posture; '
    .. 'resolve 🤖[] comments as edits when possible, or punt explicitly when not. Reply "ready".', file)
end

return M
