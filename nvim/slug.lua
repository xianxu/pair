-- nvim/slug.lua — pure decision for the orientation-slug dispose side
-- (issue #000027). No vim API here, so it runs under `nvim -l` for tests
-- (`make test-lua`). init.lua dofile's it and wraps the fs_event + buffer IO.
--
-- The proposer (cmd/pair-slug, a Stop hook) writes a candidate slug to
-- slug-proposed-<tag>; this module decides whether nvim may apply it to
-- draft line 1, and what the effective line 1 is (mirrored back to
-- slug-<tag> as the proposer's `prev`).
local M = {}

-- A structured slug is "=== <left> | <right> ===" — the machine format.
-- A "=== … ===" without a pipe is a freeform manual override.
function M.is_structured(s)
  return s:sub(1, 4) == '=== '
    and s:sub(-4) == ' ==='
    and s:find(' | ', 1, true) ~= nil
end

-- decide what to do with a freshly proposed slug, given the current draft
-- line 1 and the last value the machine itself applied (nil on restart).
--
-- Returns (action, prev):
--   action = 'apply' → set line 1 to `proposed`
--   action = 'hold'  → leave line 1 as-is
--   prev             → the effective line 1 to mirror into slug-<tag>
--
-- Safe by construction (never clobber the user):
--   • empty line 1                 → apply (no manual content to lose)
--   • machine slug, untouched      → apply (line1 == last_applied)
--   • machine slug, but edited     → hold  (line1 ~= last_applied; this also
--                                     covers last_applied == nil on restart,
--                                     so a slug surviving a restart is treated
--                                     as user content and never clobbered)
--   • freeform "=== … ===" no pipe → hold  (full manual override)
--   • any other non-empty line 1   → hold  (user's prompt text — never insert
--                                     a slug above it)
function M.decide(line1, proposed, last_applied)
  local apply
  if line1 == '' then
    apply = true
  elseif M.is_structured(line1) and last_applied ~= nil and line1 == last_applied then
    apply = true
  else
    apply = false
  end
  if apply then
    return 'apply', proposed
  end
  return 'hold', line1
end

-- apply reconciles `proposed` into buffer `buf`'s line 1, given the last value
-- the machine applied (nil on restart). Uses vim.api, so it's still headless-
-- testable under `nvim -l` (which provides the API) against a scratch buffer.
-- Returns (action, prev, new_last_applied):
--   action           'apply' | 'hold' (see decide)
--   prev             effective line 1 to mirror into slug-<tag>
--   new_last_applied the value to remember as last machine-applied
--
-- Safety: only ever rewrites line 1 (lines 2+ — the user's prompt — are never
-- touched). When the buffer is otherwise empty, a blank prompt line is added
-- below the slug so the user types under it, not onto it.
function M.apply(buf, proposed, last_applied)
  local line1 = (vim.api.nvim_buf_get_lines(buf, 0, 1, false))[1] or ''
  local action, prev = M.decide(line1, proposed, last_applied)
  if action ~= 'apply' then
    return action, prev, last_applied
  end
  local total = vim.api.nvim_buf_line_count(buf)
  local was_empty = (total <= 1 and line1 == '')
  vim.api.nvim_buf_set_lines(buf, 0, 1, false, { proposed })
  if was_empty then
    vim.api.nvim_buf_set_lines(buf, 1, 1, false, { '' })
  end
  return action, prev, proposed
end

return M
