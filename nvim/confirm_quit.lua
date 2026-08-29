local M = {}

function M.prompt(config)
  local prompt = 'Quit pair session? This kills the session and all its processes.'
  if not config then return prompt end
  local args_line = '<none>'
  if type(config.args) == 'table' and #config.args > 0 then
    args_line = table.concat(config.args, ' ')
  end
  local sid_line = config.session_id and config.session_id ~= '' and config.session_id or '<not captured>'
  return prompt
    .. '\n\nResumable later via `pair resume ' .. config.tag .. '`:'
    .. '\n  agent:      ' .. config.agent
    .. '\n  args:       ' .. args_line
    .. '\n  session id: ' .. sid_line
end

function M.run(deps)
  local answer = deps.confirm(M.prompt(deps.config), '&Yes\n&No', 2)
  if answer == 1 then deps.quit() end
end

return M
