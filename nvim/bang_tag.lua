-- A single-line draft starting with `!` tags the hosting couch thread (#337):
-- the text after the `!` goes to the agent AND becomes the thread's
-- description. The draft owns a leading `!` outright; agents' own `!` modes
-- (Claude Code's bash mode) stay reachable by typing in the agent pane.
local M = {}

-- parse takes the comment-stripped agent text. It returns nil when the text is
-- not a bang line, otherwise { agent_text, description }. A bare `!` yields an
-- empty agent_text and no description: send nothing, change nothing.
function M.parse(text)
  local line = (text or ''):gsub('^%s+', ''):gsub('%s+$', '')
  if line:sub(1, 1) ~= '!' or line:find('\n', 1, true) then return nil end
  local rest = line:sub(2):gsub('^%s+', '')
  if rest == '' then return { agent_text = '' } end
  return { agent_text = rest, description = rest }
end

return M
