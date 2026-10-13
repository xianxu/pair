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

-- Pure: the ordered zellij argv list for one poke. The body is one bracketed
-- paste, framed by the draft send's own rule (pair#211). Submit is a semantic
-- Alt+Enter key event so zellij delivers one modified chord to pair-wrap.
function M._cmds(body, agent_id, review_id)
  return {
    { 'zellij', 'action', 'write-chars', '--pane-id', tostring(agent_id), draft_send.frame(body) },
    { 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' },
  }
end

local function list_panes()
  local res = zellij_trace.action('review.poke.list-panes', { 'zellij', 'action', 'list-panes', '--json', '--command' }).stdout
  local ok, decoded = pcall(vim.json.decode, res)
  if not ok or type(decoded) ~= 'table' then return {} end
  return pane_identity.agents(decoded)
end

-- Send `body` to the agent without moving focus away from the caller.
-- Returns false (+ notify) if the agent pane can't be resolved.
function M.send(body)
  local panes = list_panes()
  local agent = panes[1] and panes[1].id
  if not agent then
    vim.notify('review: could not find the agent pane to poke', vim.log.levels.ERROR)
    return false
  end
  local cmds = M._cmds(body, agent)
  zellij_trace.action('review.poke.write-body', cmds[1], {
    redact = { [6] = cmds[1][6] },
  })
  zellij_trace.action('review.poke.submit', cmds[2])
  return true
end

return M
