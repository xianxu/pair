-- nvim/review/wrap.lua — hard-wrap a diagnostic "why" to N columns at word
-- boundaries (issue #66 M4a). `virtual_lines` does NOT soft-wrap, so a long
-- explain renders as one un-scrollable row unless we pre-wrap it. PURE (no vim
-- API); ported from parley.nvim's skill_render.M.wrap. Standalone like record.lua
-- so the colocated test runs under `make test-lua` without the IO orchestrator.
local M = {}

--- Greedy word-wrap `text` to `width` columns, preserving existing newlines.
--- A word longer than `width` stays on its own (overflowing) line rather than
--- being split mid-word. `width` defaults to 76.
--- @param text string
--- @param width number|nil
--- @return string
function M.wrap(text, width)
  width = width or 76
  local out = {}
  for para in (tostring(text) .. '\n'):gmatch('(.-)\n') do
    if para == '' then
      table.insert(out, '')
    else
      local line = ''
      for word in para:gmatch('%S+') do
        if line == '' then
          line = word
        elseif #line + 1 + #word <= width then
          line = line .. ' ' .. word
        else
          table.insert(out, line)
          line = word
        end
      end
      table.insert(out, line)
    end
  end
  return table.concat(out, '\n')
end

return M
