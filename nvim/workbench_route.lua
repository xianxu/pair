-- Shared routing for workbench shortcuts received by Pair-owned nvim panes.
-- Direct commands run in the invoking editor, preserving its pane identity.
-- Other actions execute locally in the draft or route there from overlays;
-- draft-scoped maps are installed only in the draft.
local M = {}

local here = debug.getinfo(1, 'S').source:sub(2):match('(.*/)') or './'
M.global_maps = dofile(here .. 'workbench_actions.lua')

local observation = dofile(here .. 'zellij_observation.lua')
local trace = dofile(here .. 'zellij_trace.lua')
local identity = dofile(here .. 'pane_identity.lua')

local function report(message)
  vim.schedule(function() vim.notify('pair shortcut: ' .. message, vim.log.levels.ERROR) end)
end

local function operation_budget(parent)
  return observation.capture_environment(vim.env)(observation.budget(parent))
end

local function confirmed_draft(b)
  local source_id = vim.env.ZELLIJ_PANE_ID
  return observation.confirm(b, function()
    local result, reason = observation.query(b, 'workbench.route.list-panes',
      {'zellij','action','list-panes','--json','--command','--state','--tab'}, trace.action)
    if not result then return nil, reason end
    if result.code ~= 0 then return nil, 'cannot inspect panes' end
    return result.stdout
  end, function(raw)
    local rows, reason = observation.panes(raw, {Command=true,State=true,Tab=true})
    if not rows then return nil, reason end
    local target, source = identity.select_in_tab(rows, source_id, identity.is_draft)
    if not target then return nil, source end
    return {draft=identity.context(target),source=identity.context(source)}
  end)
end

local function mutate(b, label, argv)
  local called, result, reason = pcall(observation.effect, b, function() return trace.action(label, argv) end)
  if not called then return nil, tostring(result) end
  if not result then return nil, reason end
  if result.code ~= 0 then return nil, 'native action failed' end
  return true
end

-- An internal admitted draft from review restoration shares that operation's
-- token and skips another observation pair. Ordinary callers confirm once.
function M.return_to_draft(parent_budget, admitted_draft)
  if admitted_draft and not parent_budget then report('admitted draft has no operation budget'); return false end
  local b = admitted_draft and parent_budget or operation_budget(parent_budget)
  if not b then report('admitted draft has no operation budget'); return false end
  local draft = admitted_draft
  if not draft then
    local facts, reason = confirmed_draft(b)
    if not facts then report(reason); return false end
    draft = facts.draft
  end
  for _, argv in ipairs({
    {'zellij','action','hide-floating-panes','--tab-id',tostring(draft.tab_id)},
    {'zellij','action','focus-pane-id',tostring(draft.id)},
  }) do
    local ok, reason = mutate(b, 'workbench.return', argv)
    if not ok then report(reason); return false end
  end
  return true
end

-- Viewer annotations emit on VimLeavePre; return afterwards so draft focus
-- observes the completed sidecar. Headless library tests never touch a host.
function M.install_viewer_return()
  vim.api.nvim_create_autocmd('VimLeave', { once = true, callback = function()
    if #vim.api.nvim_list_uis() > 0 then M.return_to_draft() end
  end })
end

function M.draft_commands(id, fn, focus)
  id = tostring(id)
  local commands = {}
  if focus then
    table.insert(commands, { 'zellij', 'action', 'focus-pane-id', id })
  end
  vim.list_extend(commands, {
    { 'zellij', 'action', 'write', '--pane-id', id, '28' },
    { 'zellij', 'action', 'write', '--pane-id', id, '14' },
    { 'zellij', 'action', 'write-chars', '--pane-id', id, ':lua ' .. fn .. '()' },
    { 'zellij', 'action', 'write', '--pane-id', id, '13' },
  })
  return commands
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
  local path = vim.env.PAIR_DRAFT_PANE_PATH
  local session = vim.env.ZELLIJ_SESSION_NAME
  if not path or path == ''
      or not session or session == '' then
    return nil
  end
  local ok, lines = pcall(vim.fn.readfile, path)
  if not ok then return nil end
  return M.validate_cached_draft(table.concat(lines, '\n'), session, function(pid)
    if not pid then return false end
    return vim.uv.kill(pid, 0) == 0
  end)
end

local function send_to_draft(b, id, fn, focus)
  for _, command in ipairs(M.draft_commands(id, fn, focus)) do
    local ok, reason = mutate(b, 'workbench.route.input', command)
    if not ok then report(reason .. ': failed to route ' .. fn); return false end
  end
  return true
end

function M.route(fn, focus, parent_budget)
  local b = operation_budget(parent_budget)
  local id = cached_draft_pane()
  if id then return send_to_draft(b, id, fn, focus) end
  local facts, reason = confirmed_draft(b)
  if not facts then report(reason); return false end
  return send_to_draft(b, facts.draft.id, fn, focus)
end

function M.install_global_maps(is_draft)
  for key, binding in pairs(M.global_maps) do
    if binding.scope ~= 'draft' or is_draft then
      vim.keymap.set({ 'n', 'i' }, key, function()
        if binding.direct_command then
          -- Preserve this process's ZELLIJ_PANE_ID. The Go executor owns all
          -- diagnostics; capture output and ignore its already-logged failure.
          local home = vim.env.PAIR_HOME or ''
          local command = { home ~= '' and (home .. '/bin/pair') or 'pair' }
          vim.list_extend(command, binding.direct_command)
          vim.fn.system(command)
        elseif is_draft then
          local action = _G[binding.fn]
          if type(action) == 'function' then
            action()
          else
            report(binding.fn .. ' is unavailable')
          end
        else
          M.route(binding.fn, binding.focus)
        end
      end, { silent = true, desc = 'pair global: ' .. binding.fn })
    end
  end
end

-- The argv `PairTermPrevTab`/`PairTermNextTab` shell out with (#216). Pure and
-- here rather than inline in init.lua so the command is pinned by a test: the
-- key -> function-name mapping was covered, the argv was not.
function M.switch_terminal_tab_command(pair_bin, direction)
  return { pair_bin, 'layout', 'switch-terminal-tab', direction }
end

return M
