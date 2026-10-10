-- Pure thread presentation, serialization, and save/close transactions.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local codec = dofile(here .. 'comment_codec.lua')
local markers = dofile(here .. 'markers.lua')
local prefixes = { user = '💬:', agent = '🤖:' }
local function role_line(line)
  for role, prefix in pairs(prefixes) do
    if line:sub(1, #prefix) == prefix then
      local text = line:sub(#prefix + 1)
      if text:sub(1, 1) == ' ' then text = text:sub(2) end
      return role, text
    end
  end
end
local function prefix_after_slashes(line)
  local tail = line:gsub('^\\*', '')
  return role_line(tail) ~= nil
end
local function split_lines(text)
  local lines = {}
  for line in (text .. '\n'):gmatch('(.-)\n') do lines[#lines + 1] = line end
  return lines
end
function M.roles(lines)
  local roles, current = {}, 'user'
  for i, line in ipairs(lines) do
    current = role_line(line) or current
    roles[i] = current
  end
  return roles
end
function M.to_lines(marker)
  if not marker.complete or not marker.raw or marker.raw:find('\n', 1, true) then
    return nil, 'Only complete single-line threads can be edited'
  end
  local lines, turns, payload_bytes = {}, {}, 0
  for _, section in ipairs(marker.sections) do
    local text = codec.turn_text(marker, section)
    turns[#turns + 1] = { type = section.type, text = text }
    local turn_lines = split_lines(text)
    lines[#lines + 1] = prefixes[section.type] .. ' ' .. turn_lines[1]
    for i = 2, #turn_lines do
      local line = turn_lines[i]
      lines[#lines + 1] = prefix_after_slashes(line) and ('\\' .. line) or line
    end
    payload_bytes = payload_bytes + section.byte_end - section.byte_start + 1
  end
  local appended = #turns == 0 or turns[#turns].type ~= 'user'
  if appended then lines[#lines + 1] = prefixes.user .. ' ' end
  return lines, { prefix = marker.raw:sub(1, #marker.raw - payload_bytes), appended_reply = appended }
end
function M.from_lines(lines, meta)
  if type(meta) ~= 'table' or type(meta.prefix) ~= 'string' then return nil, 'Missing thread source' end
  local turns = {}
  for _, line in ipairs(lines) do
    if type(line) ~= 'string' or line:find('\n', 1, true) then return nil, 'Invalid thread line' end
    local role, text = role_line(line)
    if role then
      turns[#turns + 1] = { type = role, text = text }
    else
      if #turns == 0 then return nil, 'Start each turn with 💬: or 🤖:' end
      if line:sub(1, 1) == '\\' and prefix_after_slashes(line) then line = line:sub(2) end
      turns[#turns].text = turns[#turns].text .. '\n' .. line
    end
  end
  if #turns == 0 then return nil, 'The thread has no turns' end
  if meta.appended_reply and turns[#turns].type == 'user' and turns[#turns].text == '' then
    table.remove(turns)
  end
  local pieces = { meta.prefix }
  for _, turn in ipairs(turns) do
    local user = turn.type == 'user'
    pieces[#pieces + 1] = (user and '[' or '{') .. codec.encode_turn(turn.text) .. (user and ']' or '}')
  end
  local raw = table.concat(pieces)
  local scan = markers.scan({ raw })
  local parsed = scan.markers[1]
  if #scan.markers ~= 1 or #scan.diagnostics > 0 or not parsed.complete or parsed.col ~= 0
      or parsed.raw ~= raw or #parsed.sections ~= #turns then
    return nil, 'Thread serialization did not form one complete marker'
  end
  local section_bytes = 0
  for i, turn in ipairs(turns) do
    local section = parsed.sections[i]
    if section.type ~= turn.type or codec.turn_text(parsed, section) ~= turn.text then
      return nil, 'Thread serialization changed a turn'
    end
    section_bytes = section_bytes + section.byte_end - section.byte_start + 1
  end
  if raw:sub(1, #raw - section_bytes) ~= meta.prefix then return nil, 'Thread serialization changed its anchor' end
  return raw
end

function M.new_session(raw)
  local session = {}
  local state, expected, pending = 'editing', raw, nil
  function session:snapshot() return { state = state, expected = expected } end
  function session:transition(event)
    if state == 'closed' then return { kind = 'none' } end
    local function refuse(message)
      state, pending = 'conflicted', nil
      return { kind = 'refuse', message = message }
    end
    if event.kind == 'save' then
      if event.error or not event.proposed then return refuse(event.error or 'Invalid thread text') end
      if event.observed ~= expected then return refuse('Source thread changed or disappeared; edits retained') end
      pending = event.proposed
      return { kind = 'replace', raw = event.proposed }
    elseif event.kind == 'saved' then
      if not pending or event.raw ~= pending then return refuse('Unexpected thread save confirmation') end
      expected, pending, state = event.raw, nil, 'editing'
      return { kind = 'none' }
    elseif event.kind == 'failed' then
      return refuse(event.error or 'Could not save thread; edits retained')
    elseif event.kind == 'close' then
      state, pending = 'closed', nil
      return { kind = 'close', rescue = event.dirty == true and not event.discard }
    end
    return { kind = 'none' }
  end
  return session
end
return M
