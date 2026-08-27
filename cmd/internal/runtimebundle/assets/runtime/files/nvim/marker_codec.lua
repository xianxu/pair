-- nvim/marker_codec.lua — shared escaping for 🤖 marker delimiter text.
-- Pure: string in/string out; no vim dependency.
local M = {}

local function escape_for(s, delims)
  s = s or ''
  local out = {}
  for i = 1, #s do
    local ch = s:sub(i, i)
    if ch == '\\' or delims[ch] then
      out[#out + 1] = '\\' .. ch
    else
      out[#out + 1] = ch
    end
  end
  return table.concat(out)
end

function M.esc_x(s)
  return escape_for(s, { ['>'] = true, [']'] = true })
end

function M.esc_y(s)
  return escape_for(s, { [']'] = true })
end

function M.esc_quote(s)
  return escape_for(s, {
    ['<'] = true, ['>'] = true,
    ['['] = true, [']'] = true,
    ['{'] = true, ['}'] = true,
  })
end

function M.unescape(s)
  local out = {}
  local i = 1
  while i <= #s do
    local c = s:sub(i, i)
    if c == '\\' and i < #s then
      out[#out + 1] = s:sub(i + 1, i + 1)
      i = i + 2
    else
      out[#out + 1] = c
      i = i + 1
    end
  end
  return table.concat(out)
end

function M.is_escaped(text, idx, start_pos)
  local bs = 0
  local j = idx - 1
  start_pos = start_pos or 1
  while j >= start_pos and text:sub(j, j) == '\\' do
    bs = bs + 1
    j = j - 1
  end
  return bs % 2 == 1
end

function M.find_unescaped(text, char, start_pos)
  local i = start_pos or 1
  while true do
    local idx = text:find(char, i, true)
    if not idx then return nil end
    if not M.is_escaped(text, idx, start_pos or 1) then return idx end
    i = idx + 1
  end
end

return M
