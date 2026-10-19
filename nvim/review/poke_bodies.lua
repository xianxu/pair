-- nvim/review/poke_bodies.lua — pure builders for the commit-signal pokes the
-- review nvim sends the agent (issue #66 M4a). The nvim writes no git; instead it
-- pokes these NL signals and the agent commits the round (the agent round's body
-- comes from its own `pair review apply` reply). ONE source for the wording
-- (review-protocol.md seam #3) so the nvim and the tests assert the same strings.
-- PURE (string-only); standalone like record.lua so the colocated test runs under
-- `make test-lua` without dofile-ing the IO orchestrator.
local M = {}
function M.with_context(body, context)
  if not context then return body end
  return body .. '\nReview context: ' .. vim.json.encode(context)
    .. '\nPass this context unchanged to pair commands that take --context. Before Git effects, verify '
    .. 'this context against line 3 of PAIR_REVIEW_OPEN_PATH and the current repository/branch/file. '
    .. 'Preserve pending artifacts and stop on mismatch.'
end

-- After the human finished their turn (the nvim saved — but did NOT git-commit;
-- the agent commits the human round). "finished", not "committed" — precise.
function M.human_finished(file, mode, instruction, label, context_path, context)
  label = label or 'Edit'
  local suffix = ''
  if instruction and instruction ~= '' then
    suffix = '; instruction: ' .. instruction
  end
  if context_path and context_path ~= '' then
    suffix = suffix .. '; use stripped review context at ' .. context_path
      .. ' for reading, while editing the actual file'
  end
  return M.with_context(string.format('finished my edits to %s — please review in %s posture%s',
    file, label, suffix),context)
end

function M.ship_requested(file, context)
  return M.with_context(string.format('ship %s — run docflow ship for the active review branch; the agent owns git', file),context)
end

function M.definition_requested(file, request_id, term, context)
  local context_arg = context and (' --context ' .. vim.fn.shellescape(vim.json.encode(context))) or ''
  return M.with_context(string.format(
    'Definition requested in %s for %q. Read the tag-scoped review-definition-request artifact for context, answer concisely, then run: pair review definition --term %q %s <definition>',
    file, term or '', term or '', request_id):gsub('pair review definition', function() return 'pair review definition'..context_arg end),context)
end

return M
