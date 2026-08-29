-- Production authored-text delivery transaction over Zellij actions.
local M = {}

function M.commands(body, no_submit)
  local cmds = {
    { kind = 'focus-agent', label = 'draft.send.focus-agent', argv = { 'zellij', 'action', 'move-focus', 'up' } },
    {
      kind = 'write',
      label = 'draft.send.write-body',
      argv = { 'zellij', 'action', 'write-chars', body },
      opts = { redact = { [4] = body } },
    },
  }
  if no_submit then
    cmds[#cmds + 1] = { kind = 'compose', label = 'draft.send.newline', argv = { 'zellij', 'action', 'write', '13' } }
  else
    cmds[#cmds + 1] = { kind = 'submit', label = 'draft.send.submit', argv = { 'zellij', 'action', 'send-keys', 'Alt Enter' } }
  end
  cmds[#cmds + 1] = { kind = 'refocus', label = 'draft.send.focus-draft', argv = { 'zellij', 'action', 'move-focus', 'down' } }
  return cmds
end

function M.send(body, no_submit, action, settle, resume_phase)
  resume_phase = resume_phase or 'start'
  if resume_phase == 'indeterminate' then
    return false, resume_phase, 'body write outcome is indeterminate; reconcile the agent composer manually'
  end
  local cmds = M.commands(body, no_submit)
  if resume_phase == 'written' then
    cmds = { cmds[1], cmds[3], cmds[4] }
  elseif resume_phase ~= 'start' then
    return false, resume_phase, 'delivery phase cannot be resumed: ' .. tostring(resume_phase)
  end
  local phase = resume_phase
  for i, cmd in ipairs(cmds) do
    local result = action(cmd.label, cmd.argv, cmd.opts) or {}
    if result.code ~= 0 then
      if i > 1 and cmd.kind ~= 'refocus' then
        local refocus = cmds[#cmds]
        action(refocus.label, refocus.argv, refocus.opts)
      end
      if cmd.kind == 'write' then phase = 'indeterminate' end
      return false, phase, cmd.label .. ' exited ' .. tostring(result.code or 'without status')
    end
    if cmd.kind == 'write' then phase = 'written' end
    if cmd.kind == 'submit' then phase = 'dispatched' end
    if cmd.kind == 'compose' then phase = 'composed' end
    if cmd.kind == 'write' and (body:find('\n') or #body > 200) then settle() end
  end
  return true, phase
end

return M
