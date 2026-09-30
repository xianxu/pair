-- A single-line draft starting with `!` tags the hosting couch thread (#337):
-- the text after the `!` goes to the agent AND becomes the thread's
-- description. `!!` sets the description after the fact and sends nothing
-- (#358). The draft owns a leading `!` outright; agents' own `!` modes
-- (Claude Code's bash mode) stay reachable by typing in the agent pane.
local M = {}

-- Couch clips descriptions to its terminal width when it renders them; the cap
-- only bounds what a description stores.
local MAX_DESCRIPTION_CHARS = 120

-- one_line reduces text to a description: its first non-blank line, runs of
-- whitespace collapsed to one space, capped at MAX_DESCRIPTION_CHARS
-- characters with `…` marking a cut. Blank text yields nil.
function M.one_line(text)
  for line in ((text or '') .. '\n'):gmatch('([^\n]*)\n') do
    line = line:gsub('%s+', ' '):gsub('^ ', ''):gsub(' $', '')
    if line ~= '' then
      local chars = {}
      for ch in line:gmatch('[%z\1-\127\194-\244][\128-\191]*') do chars[#chars + 1] = ch end
      if #chars <= MAX_DESCRIPTION_CHARS then return line end
      return table.concat(chars, '', 1, MAX_DESCRIPTION_CHARS - 1) .. '…'
    end
  end
  return nil
end

-- parse takes the comment-stripped agent text. It returns nil when the text is
-- not a bang line, otherwise one of:
--   { agent_text, description }  `! text`: send text, describe the thread
--   { agent_text = '' }          bare `!`: send nothing, change nothing
--   { describe_previous = true } bare `!!`: describe with the previous prompt
--   { description }              `!! text`: describe with text, send nothing
function M.parse(text)
  local line = (text or ''):gsub('^%s+', ''):gsub('%s+$', '')
  if line:sub(1, 1) ~= '!' or line:find('\n', 1, true) then return nil end
  if line:sub(2, 2) == '!' then
    local description = M.one_line(line:sub(3))
    if not description then return { describe_previous = true } end
    return { description = description }
  end
  local rest = line:sub(2):gsub('^%s+', '')
  if rest == '' then return { agent_text = '' } end
  return { agent_text = rest, description = rest }
end

-- previous_description derives a description from a logged prompt's
-- comment-stripped text: what the agent received, so a `! text` prompt
-- contributes text. Nil when nothing describable remains.
function M.previous_description(text)
  local tag = M.parse(text)
  if tag and tag.agent_text then text = tag.agent_text end
  return M.one_line(text)
end

return M
