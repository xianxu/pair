-- nvim/review/menu_test.lua — run via `nvim -l nvim/review/menu_test.lua`.
-- Headless smoke for the review send menu's instruction field affordances.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'menu.lua')

local fails = 0
local function ok(cond, msg)
  if not cond then
    io.stderr:write('FAIL ' .. msg .. '\n')
    fails = fails + 1
  end
end
local function contains(s, needle, msg)
  ok(tostring(s):find(needle, 1, true) ~= nil, msg .. ': ' .. tostring(s))
end
local function title_text(title)
  if type(title) == 'table' then
    local out = {}
    for _, chunk in ipairs(title) do out[#out + 1] = chunk[1] or '' end
    return table.concat(out, '')
  end
  return tostring(title or '')
end

local original_guicursor = vim.o.guicursor
vim.o.guicursor = 'n:block'

local handle = M.open({
  modes = {
    { name = 'generate' },
    { name = 'edit' },
    { name = 'proofread' },
  },
  seam = { mode_label = function(name) return name end },
  on_submit = function() end,
})

ok(handle ~= nil, 'menu opens')
ok(type(handle.focus_instruction) == 'function', 'menu exposes instruction focus')

local list_lines = vim.api.nvim_buf_get_lines(vim.api.nvim_win_get_buf(handle.list_win), 0, -1, false)
ok(list_lines[1] == '✦ Generate', 'menu labels generate with review glyph/title case')
ok(list_lines[2] == '✎ Edit', 'menu labels edit with review glyph/title case')
ok(list_lines[3] == '✓ Proofread', 'menu labels proofread with review glyph/title case')

local list_hl = vim.wo[handle.list_win].winhighlight
local instr_hl = vim.wo[handle.instr_win].winhighlight
contains(list_hl, 'Normal:PairReviewMenuNormal', 'mode list uses review menu normal highlight')
contains(list_hl, 'CursorLine:PairReviewMenuSelected', 'mode list uses selected highlight')
contains(list_hl, 'FloatBorder:PairReviewMenuBorder', 'mode list uses review border highlight')
contains(instr_hl, 'Normal:PairReviewInstructionNormal', 'instruction box uses instruction normal highlight')
contains(instr_hl, 'FloatBorder:PairReviewInstructionBorder', 'instruction box uses instruction border highlight')

local instr_config = vim.api.nvim_win_get_config(handle.instr_win)
contains(title_text(instr_config.title), 'One-round instruction', 'instruction title names one-round scope')

if handle.focus_instruction then
  handle.focus_instruction()
  ok(vim.api.nvim_get_current_win() == handle.instr_win, 'focus_instruction selects instruction window')
  ok(vim.wo[handle.instr_win].cursorline == true, 'instruction window uses cursorline')
  contains(vim.o.guicursor, 'blinkon', 'menu enables blinking cursor')
end

handle.close()
ok(vim.o.guicursor == 'n:block', 'menu restores guicursor on close')

vim.o.guicursor = original_guicursor

if fails > 0 then os.exit(1) end
print('menu_test ok')
