-- nvim/review/record.lua — pure record serialization (issue #66 M1).
--
-- A Record is the proposal unit. It carries TWO anchors:
--   occurrence     = Nth match of `old` in the round's BASE content (the agent
--                    emits this; review.apply uses it to locate the edit).
--   new_occurrence = Nth match of `new` in the POST-apply content (review.apply
--                    computes + adds this; resume-reconstruction places the
--                    decoration by it). They differ whenever `old` and `new`
--                    occur a different number of times — never cross them.
--
-- The SAME JSON is written to the handoff file AND embedded in the agent commit
-- body. Uses vim.json (deterministic, available under `nvim -l`); no IO/state.
local M = {}

local OPEN = '```review-records'

function M.encode(records)
  return vim.json.encode(records)
end

function M.decode(s)
  return vim.json.decode(s)
end

-- Build an agent commit body: prose summary + a fenced records block.
function M.embed_in_body(summary, records)
  return table.concat({ summary, '', OPEN, M.encode(records), '```' }, '\n')
end

-- Pull the records back out of a commit body; nil if no block present. The
-- non-greedy capture tolerates trailing content (e.g. a Co-Authored-By trailer).
function M.extract_from_body(body)
  local block = body:match('```review%-records\n(.-)\n```')
  if not block then return nil end
  return M.decode(block)
end

return M
