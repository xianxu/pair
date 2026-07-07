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

-- Pure: the ordered zellij argv list for one poke. Submit is a semantic
-- Alt+Enter key event so zellij delivers one modified chord to pair-wrap.
function M._cmds(body, agent_id, review_id)
  return {
    { 'zellij', 'action', 'write-chars', '--pane-id', tostring(agent_id), body },
    { 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' },
  }
end

-- Recursively collect pane objects (have both `id` and `is_floating`) from the
-- decoded list-panes JSON — mirrors scrollback-open's `.. | objects | select`.
local function collect_panes(node, out)
  if type(node) ~= 'table' then return out end
  if node.id ~= nil and node.is_floating ~= nil then out[#out + 1] = node end
  for _, v in pairs(node) do
    if type(v) == 'table' then collect_panes(v, out) end
  end
  return out
end

local function list_panes()
  local res = zellij_trace.action('review.poke.list-panes', { 'zellij', 'action', 'list-panes', '--json' }).stdout
  local ok, decoded = pcall(vim.json.decode, res)
  if not ok or type(decoded) ~= 'table' then return {} end
  return collect_panes(decoded, {})
end

-- The agent pane: a real terminal (not plugin), tiled (not floating), not the
-- draft — the same predicate `pair scrollback open` uses to find it.
local function find_agent(panes)
  for _, p in ipairs(panes) do
    if p.is_plugin == false and p.is_floating == false
        and p.title ~= nil and p.title ~= '' and p.title ~= 'draft' then
      return p.id
    end
  end
end

-- Send `body` to the agent without moving focus away from the caller.
-- Returns false (+ notify) if the agent pane can't be resolved.
function M.send(body)
  local panes = list_panes()
  local agent = find_agent(panes)
  if not agent then
    vim.notify('review: could not find the agent pane to poke', vim.log.levels.ERROR)
    return false
  end
  local cmds = M._cmds(body, agent)
  zellij_trace.action('review.poke.write-body', cmds[1], {
    redact = { [6] = body },
  })
  zellij_trace.action('review.poke.submit', cmds[2])
  return true
end

return M
