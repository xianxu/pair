-- nvim/review/define.lua -- pure helpers for durable review-pane definitions.
-- No Neovim APIs here; review.lua owns visual selection IO and buffer writes.
local M = {}

local function trim(s)
  return (s or ''):gsub('^%s*(.-)%s*$', '%1')
end

local function copy_lines(lines)
  local out = {}
  for i, line in ipairs(lines or {}) do out[i] = line end
  return out
end

function M.slice_selection(lines, l1, c1, l2, c2)
  if l1 > l2 or (l1 == l2 and c1 > c2) then return '' end
  if l1 == l2 then
    local line = lines[l1] or ''
    return line:sub(c1 + 1, math.min(c2 + 1, #line))
  end
  local out = {}
  for l = l1, l2 do
    local line = lines[l] or ''
    if l == l1 then
      out[#out + 1] = line:sub(c1 + 1)
    elseif l == l2 then
      out[#out + 1] = line:sub(1, math.min(c2 + 1, #line))
    else
      out[#out + 1] = line
    end
  end
  return table.concat(out, '\n')
end

function M.footnote_id(term)
  local id = tostring(term or ''):lower()
  id = id:gsub('[^%w]+', '-')
  id = id:gsub('^%-+', ''):gsub('%-+$', '')
  if id == '' then id = 'definition' end
  return id
end

function M.format_footnote_line(id, definition)
  definition = trim(definition)
  if definition == '' then definition = '(no definition)' end
  return string.format('[^%s]: %s', id, definition)
end

local function split_text_lines(text)
  text = text or ''
  local lines, start = {}, 1
  while true do
    local nl = text:find('\n', start, true)
    if not nl then
      lines[#lines + 1] = text:sub(start)
      break
    end
    lines[#lines + 1] = text:sub(start, nl - 1)
    start = nl + 1
  end
  if #lines > 1 and lines[#lines] == '' then table.remove(lines) end
  return lines
end

local function is_divider(line)
  return trim(line) == '---'
end

local function is_footnote_line(line)
  return trim(line):match('^%[%^[^%]]+%]:') ~= nil
end

local function managed_footer_start(lines)
  for i = #lines, 1, -1 do
    if is_divider(lines[i]) then
      local has_footnote = false
      for j = i + 1, #lines do
        local line = lines[j] or ''
        if trim(line) ~= '' then
          if not is_footnote_line(line) then return nil end
          has_footnote = true
        end
      end
      if has_footnote then return i end
      return nil
    end
  end
  return nil
end

local function parse_footnote_line(line)
  local id, definition = trim(line):match('^%[%^([^%]]+)%]:%s*(.-)%s*$')
  if not id then return nil end
  definition = trim(definition)
  if definition == '' then definition = '(no definition)' end
  return id, definition
end

function M.strip_definition_footnote_footer(text)
  local lines = split_text_lines(text or '')
  local start = managed_footer_start(lines)
  if not start then return text or '' end
  while start > 1 and trim(lines[start - 1]) == '' do start = start - 1 end
  local kept = {}
  for i = 1, start - 1 do kept[#kept + 1] = lines[i] end
  while #kept > 0 and trim(kept[#kept]) == '' do table.remove(kept) end
  return table.concat(kept, '\n')
end

local function replace_or_append_footnote(lines, id, definition)
  local out = copy_lines(lines)
  local footer = managed_footer_start(out)
  local footnote_line = M.format_footnote_line(id, definition)
  if footer then
    local escaped_id = id:gsub('([^%w])', '%%%1')
    for i = footer + 1, #out do
      if trim(out[i]):match('^%[%^' .. escaped_id .. '%]:') then
        out[i] = footnote_line
        return out
      end
    end
    out[#out + 1] = footnote_line
    return out
  end

  while #out > 0 and trim(out[#out]) == '' do table.remove(out) end
  out[#out + 1] = ''
  out[#out + 1] = '---'
  out[#out + 1] = ''
  out[#out + 1] = footnote_line
  return out
end

function M.apply_definition_footnote(lines, l1, c1, l2, c2, term, definition)
  local id = M.footnote_id(term)
  local ref = '[^' .. id .. ']'
  local out = copy_lines(lines)
  local target_line = l2
  local line = out[target_line] or ''
  local ec = math.min(c2 + 1, #line)
  if line:sub(ec + 1, ec + #ref) ~= ref then
    out[target_line] = line:sub(1, ec) .. ref .. line:sub(ec + 1)
  end
  out = replace_or_append_footnote(out, id, definition)
  local normalized = trim(definition)
  if normalized == '' then normalized = '(no definition)' end
  return {
    lines = out,
    id = id,
    definition = normalized,
    diagnostic_span = {
      line = l1 - 1,
      col = c1,
      end_line = l2 - 1,
      end_col = c2 + 1 + #ref,
    },
  }
end

local function is_term_byte(ch)
  return ch:match('[%w_-]') ~= nil
end

local function expand_term_start(line, ref_start)
  local start = ref_start
  while start > 1 and is_term_byte(line:sub(start - 1, start - 1)) do
    start = start - 1
  end
  return start
end

function M.footnote_diagnostics(lines)
  lines = lines or {}
  local footer = managed_footer_start(lines)
  if not footer then return {} end

  local definitions = {}
  for i = footer + 1, #lines do
    local id, definition = parse_footnote_line(lines[i] or '')
    if id then definitions[id] = definition end
  end

  local diagnostics = {}
  for lnum = 1, footer - 1 do
    local line = lines[lnum] or ''
    local search = 1
    while true do
      local ref_start, ref_end, id = line:find('%[%^([^%]]+)%]', search)
      if not ref_start then break end
      local definition = definitions[id]
      if definition then
        local term_start = expand_term_start(line, ref_start)
        local term = line:sub(term_start, ref_start - 1)
        diagnostics[#diagnostics + 1] = {
          id = id,
          term = term ~= '' and term or nil,
          definition = definition,
          line = lnum - 1,
          col = term_start - 1,
          end_line = lnum - 1,
          end_col = ref_end,
        }
      end
      search = ref_end + 1
    end
  end
  return diagnostics
end

return M
