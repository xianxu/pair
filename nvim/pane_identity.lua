-- Pure list-panes parsing shared by review pokes and DOM submission routing.
local M = {}

function M.collect(node)
  local out = {}
  local function visit(value)
    if type(value) ~= 'table' then return end
    if value.id ~= nil and value.is_floating ~= nil then out[#out + 1] = value end
    for _, child in pairs(value) do
      if type(child) == 'table' then visit(child) end
    end
  end
  visit(node)
  return out
end

-- Names and list ordering cannot distinguish an agent from a right terminal.
function M.is_agent(pane)
  return pane.id ~= nil and pane.is_plugin == false and pane.is_floating == false
    and type(pane.terminal_command) == 'string'
    and pane.terminal_command:find('pair wrap', 1, true) ~= nil
end

function M.agents(decoded)
  local out = {}
  for _, pane in ipairs(M.collect(decoded)) do
    if M.is_agent(pane) then out[#out + 1] = pane end
  end
  return out
end

return M
