-- Versioned Pair prompt-log framing shared by history read and rewrite paths.
local M = {}

local separator = '\n\n---\n\n'

local function parse_one(text, start_pos)
  if text:sub(start_pos, start_pos + 2) ~= '## ' then
    return nil, 'missing timestamp header'
  end
  local line_end = text:find('\n', start_pos, true)
  if not line_end then return nil, 'unterminated timestamp header' end
  local timestamp = text:sub(start_pos, line_end - 1)
  local cursor = line_end + 1
  local marker_end = text:find('\n\n', cursor, true)
  local marker = marker_end and text:sub(cursor, marker_end - 1) or ''
  local count = marker:match('^<!%-%- pair%-log%-v1 bytes=(%d+) %-%->$')
  local body_start
  local body_end
  local finish
  local format
  if count then
    count = tonumber(count)
    body_start = marker_end + 2
    body_end = body_start + count - 1
    finish = body_end + 1
    if text:sub(finish, finish + #separator - 1) ~= separator then
      return nil, 'invalid byte-counted entry suffix'
    end
    finish = finish + #separator
    format = 'v1'
  else
    if text:sub(cursor, cursor) ~= '\n' then return nil, 'missing legacy header separator' end
    body_start = cursor + 1
    local sep_start = text:find(separator, body_start, true)
    if sep_start then
      body_end = sep_start - 1
      finish = sep_start + #separator
    else
      body_end = #text
      finish = #text + 1
    end
    format = 'legacy'
  end
  return {
    body = text:sub(body_start, body_end),
    format = format,
    timestamp = timestamp,
    start_pos = start_pos,
    finish = finish,
  }
end

function M.parse(text)
  local entries = {}
  local cursor = 1
  while cursor <= #text do
    local entry, err = parse_one(text, cursor)
    if not entry then return nil, err end
    entries[#entries + 1] = entry
    cursor = entry.finish
  end
  return entries
end

local function encode(entry, body)
  if entry.format == 'v1' then
    return entry.timestamp .. '\n<!-- pair-log-v1 bytes=' .. #body .. ' -->\n\n' .. body .. separator
  end
  return entry.timestamp .. '\n\n' .. body .. separator
end

function M.replace(text, newest_index, body)
  local entries, err = M.parse(text)
  if not entries then return nil, err end
  local index = #entries - newest_index + 1
  local target = entries[index]
  if not target then return nil, 'history index out of range' end
  return text:sub(1, target.start_pos - 1) .. encode(target, body) .. text:sub(target.finish)
end

return M
