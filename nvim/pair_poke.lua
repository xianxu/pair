-- nvim/pair_poke.lua — inject an instruction into the agent pane from ANY pane
-- (issue #66 M3). The draft's send_to_agent uses RELATIVE move-focus, which does
-- NOT escape a floating pane (documented in scrollback.lua). The review pane
-- resolves the agent's ABSOLUTE pane id from `zellij action list-panes --json`
-- and writes directly to that pane id, so review focus never moves.
-- Deliberately NO has_ui() short-circuit, so the headless test can stub `zellij`
-- and record the calls.
local M = {}

local pair_nvim_dir = vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':p:h')
local zellij_trace = dofile(pair_nvim_dir .. '/zellij_trace.lua')
local draft_send = dofile(pair_nvim_dir .. '/draft_send.lua')
local pane_identity = dofile(pair_nvim_dir .. '/pane_identity.lua')
local observation = dofile(pair_nvim_dir .. '/zellij_observation.lua')

-- Pure: the ordered zellij argv list for one poke. The body is one bracketed
-- paste, framed by the draft send's own rule (pair#211). Submit is a semantic
-- Alt+Enter key event so zellij delivers one modified chord to pair-wrap.
function M._cmds(body, agent_id, review_id)
  return {
    { 'zellij', 'action', 'write-chars', '--pane-id', tostring(agent_id), draft_send.frame(body) },
    { 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' },
  }
end

-- Resolve the unique agent in the independently known invoking terminal's
-- tab, then confirm only the identity/tab/launch-template decision facts.
local function agent_context(b)
  local source_id = vim.env.ZELLIJ_PANE_ID
  return observation.confirm(b, function()
    local result, reason = observation.query(b, 'review.poke.list-panes',
      {'zellij','action','list-panes','--json','--command','--state','--tab'}, zellij_trace.action)
    if not result then return nil, reason end
    if result.code ~= 0 then return nil, 'pane query failed' end
    return result.stdout
  end, function(raw)
    local rows, reason = observation.panes(raw, {Command=true,State=true,Tab=true})
    if not rows then return nil, reason end
    local agent, source = pane_identity.select_in_tab(rows, source_id, pane_identity.is_agent)
    if not agent then return nil, source end
    return {agent=pane_identity.context(agent),source=pane_identity.context(source)}
  end)
end

function M.send(body, parent_budget)
  local b = observation.capture_environment(vim.env)(observation.budget(parent_budget))
  local facts, reason = agent_context(b)
  if not facts then
    vim.notify('review: could not confirm the agent pane to poke: ' .. tostring(reason), vim.log.levels.ERROR)
    return false
  end
  local cmds = M._cmds(body, facts.agent.id)
  for i, command in ipairs(cmds) do
    local called, result, err = pcall(observation.effect, b, function()
      return zellij_trace.action(i == 1 and 'review.poke.write-body' or 'review.poke.submit', command,
        i == 1 and {redact={[6]=command[6]}} or nil)
    end)
    if not called then err, result = tostring(result), nil end
    if not result or result.code ~= 0 then
      vim.notify('review: poke failed: ' .. tostring(err or 'native action failed'), vim.log.levels.ERROR)
      return false
    end
  end
  return true
end

return M
