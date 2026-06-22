-- nvim/review/menu.lua — self-contained review send menu for pair's review pane.
-- Shape follows parley.nvim: a mode selector plus an optional instruction buffer.
local M = {}

local last_mode

local function layout(count)
  local ui = vim.api.nvim_list_uis()[1] or { width = 80, height = 24 }
  local width = math.min(70, math.max(40, ui.width - 8))
  local list_h = math.min(math.max(count, 1), math.max(3, ui.height - 12))
  local col = math.max(0, math.floor((ui.width - width) / 2))
  local total_h = list_h + 7
  local row = math.max(0, math.floor((ui.height - total_h) / 2))
  return width, list_h, row, col, row + list_h + 1
end

function M.open(opts)
  opts = opts or {}
  local modes = opts.modes or {}
  if #modes == 0 then
    vim.notify('review: no review modes found', vim.log.levels.WARN)
    return nil
  end
  local seam = opts.seam
  local on_submit = opts.on_submit or function() end
  local current = opts.mode or last_mode
  local start_line = 1
  for i, mode in ipairs(modes) do
    if mode.name == current then start_line = i; break end
  end

  local width, list_h, row, col, instr_row = layout(#modes)
  local list_buf = vim.api.nvim_create_buf(false, true)
  vim.bo[list_buf].buftype = 'nofile'
  vim.bo[list_buf].bufhidden = 'wipe'
  local lines = {}
  for i, mode in ipairs(modes) do lines[i] = seam.mode_label(mode.name) end
  vim.api.nvim_buf_set_lines(list_buf, 0, -1, false, lines)
  vim.bo[list_buf].modifiable = false
  local list_win = vim.api.nvim_open_win(list_buf, true, {
    relative = 'editor', row = row, col = col, width = width, height = list_h,
    style = 'minimal', border = 'rounded',
    title = ' Review mode - j/k select · Enter run · Tab→instruction ',
  })
  vim.wo[list_win].cursorline = true
  vim.api.nvim_win_set_cursor(list_win, { start_line, 0 })

  local instr_buf = vim.api.nvim_create_buf(false, true)
  vim.bo[instr_buf].buftype = ''
  vim.bo[instr_buf].bufhidden = 'wipe'
  if opts.instruction and opts.instruction ~= '' then
    vim.api.nvim_buf_set_lines(instr_buf, 0, -1, false, vim.split(opts.instruction, '\n', { plain = true }))
  end
  local instr_win = vim.api.nvim_open_win(instr_buf, false, {
    relative = 'editor', row = instr_row, col = col, width = width, height = 5,
    style = 'minimal', border = 'rounded',
    title = ' Instruction - optional (M-CR/C-s submit · Tab/Esc→list) ',
  })

  local closed = false
  local function close()
    if closed then return end
    closed = true
    pcall(vim.api.nvim_win_close, list_win, true)
    pcall(vim.api.nvim_win_close, instr_win, true)
  end
  local function selected()
    local line = 1
    if vim.api.nvim_win_is_valid(list_win) then
      line = vim.api.nvim_win_get_cursor(list_win)[1]
    end
    return modes[line] or modes[1]
  end
  local function move(delta)
    if not vim.api.nvim_win_is_valid(list_win) then return end
    local line = vim.api.nvim_win_get_cursor(list_win)[1]
    vim.api.nvim_win_set_cursor(list_win, { ((line - 1 + delta) % #modes) + 1, 0 })
  end
  local function submit()
    local mode = selected()
    local instruction = table.concat(vim.api.nvim_buf_get_lines(instr_buf, 0, -1, false), '\n')
      :gsub('^%s+', ''):gsub('%s+$', '')
    if mode.name == 'free-form' and instruction == '' then
      vim.notify('review: free-form mode requires an instruction', vim.log.levels.WARN)
      return
    end
    last_mode = mode.name
    close()
    on_submit({ mode = mode.name, instruction = instruction })
  end
  local function focus_instr()
    if vim.api.nvim_win_is_valid(instr_win) then
      vim.api.nvim_set_current_win(instr_win)
      pcall(vim.cmd, 'startinsert')
    end
  end
  local function focus_list()
    if vim.api.nvim_win_is_valid(list_win) then
      pcall(vim.cmd, 'stopinsert')
      vim.api.nvim_set_current_win(list_win)
    end
  end

  local function lmap(lhs, fn)
    vim.keymap.set('n', lhs, fn, { buffer = list_buf, nowait = true, silent = true })
  end
  lmap('<CR>', submit)
  lmap('<M-CR>', submit)
  lmap('<C-s>', submit)
  lmap('<C-j>', function() move(1) end)
  lmap('<C-k>', function() move(-1) end)
  lmap('<Tab>', focus_instr)
  lmap('i', focus_instr)
  lmap('a', focus_instr)
  lmap('<Esc>', close)
  lmap('<C-c>', close)

  local function imap(modes_, lhs, fn)
    vim.keymap.set(modes_, lhs, fn, { buffer = instr_buf, nowait = true, silent = true })
  end
  imap({ 'n', 'i' }, '<M-CR>', submit)
  imap({ 'n', 'i' }, '<C-s>', submit)
  imap({ 'n', 'i' }, '<C-c>', close)
  imap('n', '<Tab>', focus_list)
  imap('n', '<Esc>', focus_list)

  return {
    list_win = list_win,
    instr_win = instr_win,
    submit = submit,
    move = move,
    close = close,
    selected = function() return selected().name end,
  }
end

return M
