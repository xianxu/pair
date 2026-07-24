-- Shared routing for workbench-global shortcuts received by Pair-owned nvim
-- panes. Draft executes locally; overlays address the draft pane explicitly.
local M = {}

M.global_maps = {
  ['<M-d>'] = 'PairConfirmDetach',
  ['<M-x>'] = 'PairConfirmQuit',
  ['<M-n>'] = 'PairConfirmRestart',
  ['<C-M-n>'] = 'PairConfirmRestart',
  ['<M-N>'] = 'PairConfirmAgentRestart',
  ['<M-Up>'] = 'PairLayoutBigger',
  ['<M-Down>'] = 'PairLayoutSmaller',
  ['<M-c>'] = 'PairReviewToggle',
}

local function pane_id(pane)
  local id = pane.id
  if id == nil then id = pane.pane_id end
  return id == nil and nil or tostring(id)
end

function M.find_draft_pane(root)
  if type(root) ~= 'table' then return nil end
  local command = root.terminal_command
  if type(command) == 'string'
      and command:find('nvim', 1, true)
      and command:find('/nvim/init.lua', 1, true) then
    return pane_id(root)
  end
  for _, child in pairs(root) do
    if type(child) == 'table' then
      local id = M.find_draft_pane(child)
      if id then return id end
    end
  end
  return nil
end

function M.draft_commands(id, fn)
  id = tostring(id)
  return {
    { 'zellij', 'action', 'write', '--pane-id', id, '28' },
    { 'zellij', 'action', 'write', '--pane-id', id, '14' },
    { 'zellij', 'action', 'write-chars', '--pane-id', id, ':lua ' .. fn .. '()' },
    { 'zellij', 'action', 'write', '--pane-id', id, '13' },
  }
end

function M.validate_cached_draft(raw, session, alive)
  local ok, record = pcall(vim.json.decode, raw)
  if not ok or type(record) ~= 'table'
      or type(record.session) ~= 'string' or record.session == ''
      or record.session ~= session
      or record.pane_id == nil or tostring(record.pane_id) == ''
      or not alive(tonumber(record.pid)) then
    return nil
  end
  return tostring(record.pane_id)
end

local function cached_draft_pane()
  local data_dir = vim.env.PAIR_DATA_DIR
  local tag = vim.env.PAIR_TAG
  local session = vim.env.ZELLIJ_SESSION_NAME
  if not data_dir or data_dir == '' or not tag or tag == ''
      or not session or session == '' then
    return nil
  end
  local ok, lines = pcall(vim.fn.readfile, data_dir .. '/draft-pane-' .. tag .. '.json')
  if not ok then return nil end
  return M.validate_cached_draft(table.concat(lines, '\n'), session, function(pid)
    if not pid then return false end
    return vim.uv.kill(pid, 0) == 0
  end)
end

local function report(message)
  vim.schedule(function()
    vim.notify('pair shortcut: ' .. message, vim.log.levels.ERROR)
  end)
end

local function send_to_draft(id, fn)
  for _, command in ipairs(M.draft_commands(id, fn)) do
    vim.fn.system(command)
    if vim.v.shell_error ~= 0 then
      report('failed to route ' .. fn)
      return false
    end
  end
  return true
end

function M.route(fn)
  local id = cached_draft_pane()
  if id then return send_to_draft(id, fn) end
  local raw = vim.fn.system({
    'zellij', 'action', 'list-panes', '--json', '--command', '--state',
  })
  if vim.v.shell_error ~= 0 then
    report('cannot inspect panes')
    return false
  end
  local ok, panes = pcall(vim.json.decode, raw)
  id = ok and M.find_draft_pane(panes) or nil
  if not id then
    report('draft pane not found')
    return false
  end
  return send_to_draft(id, fn)
end

function M.install_global_maps(is_draft)
  for key, fn in pairs(M.global_maps) do
    vim.keymap.set({ 'n', 'i' }, key, function()
      if is_draft then
        local action = _G[fn]
        if type(action) == 'function' then
          action()
        else
          report(fn .. ' is unavailable')
        end
      else
        M.route(fn)
      end
    end, { silent = true, desc = 'pair global: ' .. fn })
  end
end

return M
