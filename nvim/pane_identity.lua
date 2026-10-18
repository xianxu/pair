-- Pure role/context selection over strictly decoded native pane rows.
local M = {}

-- Names and list ordering cannot distinguish an agent from a right terminal.
function M.is_agent(pane)
  return pane.id ~= nil and pane.is_plugin == false and pane.is_floating == false
    and type(pane.terminal_command) == 'string'
    and pane.terminal_command:find('pair wrap', 1, true) ~= nil
end

-- The invoking terminal identity is independent of the query. A reply must
-- correlate it positively before any target in the same tab is authoritative.
-- Rows must be flat typed output from zellij_observation.panes with Tab.
-- On success return target, source; on refusal return nil, reason.
function M.select_in_tab(rows, source_id, predicate)
  if type(source_id) ~= 'string' or not source_id:match('^%d+$') then return nil, 'invoking terminal identity unavailable' end
  local id = tonumber(source_id)
  if not id or id > 9007199254740991 then return nil, 'invalid invoking terminal identity' end
  local source
  for _, p in ipairs(rows) do
    if p.is_plugin == false and p.id == id then
      if source then return nil, 'ambiguous invoking terminal' end
      source = p
    end
  end
  if not source then return nil, 'invoking terminal is absent' end
  for _, key in ipairs({'tab_id','tab_position'}) do
    local value = source[key]
    if type(value) ~= 'number' or value < 0 or value > 9007199254740991 or value ~= math.floor(value) then
      return nil, 'invoking terminal tab is unknown'
    end
  end
  local target
  for _, p in ipairs(rows) do
    if p.tab_id == source.tab_id and p.tab_position ~= source.tab_position then
      return nil, 'conflicting position in invoking tab'
    end
    if p.tab_id == source.tab_id and p.tab_position == source.tab_position and predicate(p) then
      if target then return nil, 'ambiguous target role' end
      target = p
    end
  end
  if not target then return nil, 'target role absent from invoking tab' end
  return target, source
end

function M.is_draft(pane)
  return pane.is_plugin == false and pane.is_floating == false
    and type(pane.terminal_command) == 'string'
    and pane.terminal_command:find('nvim', 1, true) ~= nil
    and pane.terminal_command:find('/nvim/init.lua', 1, true) ~= nil
end

-- Decision facts preserve native identity, tab and the launch template used
-- for role selection. Cosmetic titles and transient foreground text are absent.
function M.context(pane)
  return {id=pane.id,tab_id=pane.tab_id,tab_position=pane.tab_position,terminal_command=pane.terminal_command}
end

return M
