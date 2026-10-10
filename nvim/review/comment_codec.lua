-- Canonical single-line turn payloads. Raw slash parity is interpreted before
-- delimiter unescaping; composing whole-string codecs loses that information.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local marker_codec = dofile(here .. '../marker_codec.lua')
function M.encode_turn(text)
  local out, i = {}, 1
  while i <= #text do
    local ch = text:sub(i, i)
    if text:sub(i, i + 3) == '<br>' then
      out[#out + 1], i = '\\<br>', i + 4
    elseif ch == '\n' then
      out[#out + 1], i = '<br>', i + 1
    else
      out[#out + 1] = ch == '`' and '\\`' or marker_codec.escape_quote_byte(ch)
      i = i + 1
    end
  end
  return table.concat(out)
end
function M.decode_turn(raw)
  local out, i = {}, 1
  while i <= #raw do
    local finish = i
    while raw:sub(finish, finish) == '\\' do finish = finish + 1 end
    if raw:sub(finish, finish + 3) == '<br>' then
      local count = finish - i
      out[#out + 1] = string.rep('\\', math.floor(count / 2))
      out[#out + 1] = count % 2 == 0 and '\n' or '<br>'
      i = finish + 4
    elseif finish > i then
      local count = finish - i
      out[#out + 1] = string.rep('\\', math.floor(count / 2))
      i = finish
      if count % 2 == 1 then
        out[#out + 1] = i <= #raw and raw:sub(i, i) or '\\'
        i = i + 1
      end
    else
      out[#out + 1], i = raw:sub(i, i), i + 1
    end
  end
  return table.concat(out)
end
-- Legacy multiline records and manually constructed section records retain
-- their existing decoded-text contract. Anchors never use this function.
function M.turn_text(marker, section)
  if section.raw_text and marker.raw and not marker.raw:find('\n', 1, true) then
    return M.decode_turn(section.raw_text)
  end
  return section.text
end
return M
