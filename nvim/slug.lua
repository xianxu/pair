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

-- decide what to do with a freshly proposed slug given the current line 1.
-- Soft policy (issue #000027): nvim trusts the proposer. The proposer only
-- writes a proposal when the model decided the slug should change (and it
-- saw the user's edit as `prev` and biased toward KEEP), so nvim's only job
-- is to refuse to clobber content that is the user's to own.
--
-- Returns (action, prev):
--   action = 'apply' → set line 1 to `proposed`
--   action = 'hold'  → leave line 1 as-is
--   prev             → the effective line 1 to mirror into slug-<tag>
--
--   • empty line 1                 → apply
--   • structured slug              → apply (incl. a user-edited one — the
--                                     model decided to change it; if it
--                                     should have stayed, no proposal was
--                                     written and apply isn't reached)
--   • freeform "=== … ===" no pipe → hold (full manual override — yours)
--   • any other non-empty line 1   → hold (user's prompt text; never clobber)
function M.decide(line1, proposed)
  if line1 == '' or M.is_structured(line1) then
    return 'apply', proposed
  end
  return 'hold', line1
end

-- apply reconciles `proposed` into buffer `buf`'s line 1. Uses vim.api, so
-- it's headless-testable under `nvim -l` against a scratch buffer.
-- Returns (action, prev):
--   action 'apply' | 'hold' (see decide)
--   prev   effective line 1 to mirror into slug-<tag>
--
-- Safety: only ever rewrites line 1 (lines 2+ — the user's prompt — are never
-- touched). When the buffer is otherwise empty, a blank prompt line is added
-- below the slug so the user types under it, not onto it.
function M.apply(buf, proposed)
  local line1 = (vim.api.nvim_buf_get_lines(buf, 0, 1, false))[1] or ''
  local action, prev = M.decide(line1, proposed)
  if action ~= 'apply' then
    return action, prev
  end
  local total = vim.api.nvim_buf_line_count(buf)
  local was_empty = (total <= 1 and line1 == '')
  vim.api.nvim_buf_set_lines(buf, 0, 1, false, { proposed })
  if was_empty then
    vim.api.nvim_buf_set_lines(buf, 1, 1, false, { '' })
    local win = vim.fn.bufwinid(buf)
    if win ~= -1 and vim.api.nvim_win_get_cursor(win)[1] == 1 then
      vim.api.nvim_win_set_cursor(win, { 2, 0 })
    end
  end
  return action, prev
end

return M
