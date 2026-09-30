-- Production authored-text delivery transaction over Zellij actions.
local M = {}

M.PASTE_START = '\27[200~'
M.PASTE_END = '\27[201~'

-- The body goes as ONE bracketed paste (pair#211). Unbracketed, the agent sees
-- a burst of typed input that macOS hands it in ~1 KiB tty reads and must
-- guess is a paste; Claude Code measurably dropped whole middle reads of
-- 3+-read sends (8 of 11 in the 2026-09 audit). Inside the markers the agent
-- collects until the end marker however the bytes are chunked. Every profiled
-- harness already takes bracketed pastes (wrapcmd/orientation.go sends one).
-- Paste markers are stripped from the body first, so no text can close the
-- paste early or open a nested one.
function M.frame(body)
  local clean = body:gsub('\27%[20[01]~', '')
  return M.PASTE_START .. clean .. M.PASTE_END
end

-- The inverse, for the stateful agent fakes that model a paste-aware composer:
-- the pasted content, or nil when the wire is not exactly one bracketed paste.
function M.unframe(wire)
  local s, f = M.PASTE_START, M.PASTE_END
  if #wire < #s + #f or wire:sub(1, #s) ~= s or wire:sub(-#f) ~= f then return nil end
  return wire:sub(#s + 1, -#f - 1)
end

function M.commands(body)
  local wire = M.frame(body)
  local cmds = {
    { kind = 'focus-agent', label = 'draft.send.focus-agent', argv = { 'zellij', 'action', 'move-focus', 'up' } },
    {
      kind = 'write',
      label = 'draft.send.write-body',
      argv = { 'zellij', 'action', 'write-chars', wire },
      opts = { redact = { [4] = wire } },
    },
  }
  cmds[#cmds + 1] = { kind = 'submit', label = 'draft.send.submit', argv = { 'zellij', 'action', 'send-keys', 'Alt Enter' } }
  cmds[#cmds + 1] = { kind = 'refocus', label = 'draft.send.focus-draft', argv = { 'zellij', 'action', 'move-focus', 'down' } }
  return cmds
end

function M.send(body, action, settle, resume_phase)
  resume_phase = resume_phase or 'start'
  if resume_phase == 'indeterminate' then
    return false, resume_phase, 'body write outcome is indeterminate; reconcile the agent composer manually'
  end
  local cmds = M.commands(body)
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
    if cmd.kind == 'write' then settle() end
  end
  return true, phase
end

return M
