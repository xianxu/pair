-- Real Neovim resource and write lifecycle; run with nvim -l.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local F = dofile(here .. 'comment_float.lua')
local P = dofile(here .. 'markers.lua')
vim.o.clipboard = ''
vim.notify = function() end
local function source(line)
  local b = vim.api.nvim_create_buf(true, false)
  vim.api.nvim_set_current_buf(b)
  vim.api.nvim_buf_set_lines(b, 0, -1, false, { line })
  vim.bo[b].modified = false
  return b
end
local function open(b, n)
  local m = P.parse_markers(vim.api.nvim_buf_get_lines(b, 0, -1, false))[n or 1]
  assert(F.open_thread(b, {marker=m,start=m.col,stop=m.col+#m.raw}))
  vim.cmd('stopinsert')
  return vim.api.nvim_get_current_buf(), vim.api.nvim_get_current_win()
end
local function lines(b) return vim.api.nvim_buf_get_lines(b, 0, -1, false) end
local function edit(b, ls) vim.api.nvim_buf_set_lines(b, 0, -1, false, ls) end
local function write() vim.cmd('write') end
local b = source('before 🤖[q]{a} after')
local fb, fw = open(b)
assert(vim.bo[fb].buftype == 'acwrite')
assert(lines(fb)[3] == '💬: ')
edit(fb, {'💬: q','🤖: a','💬: reply','two'})
write()
assert(lines(b)[1] == 'before 🤖[q]{a}[reply<br>two] after', lines(b)[1])
assert(not vim.bo[fb].modified)
edit(fb, {'💬: q','🤖: a','💬: again'})
write()
assert(lines(b)[1] == 'before 🤖[q]{a}[again] after', 'second save')
-- Source movement must preserve this instance's identity.
vim.api.nvim_buf_set_text(b, 0, 0, 0, 0, {'prefix '})
edit(fb, {'💬: q','🤖: a','💬: moved'})
write()
assert(lines(b)[1] == 'prefix before 🤖[q]{a}[moved] after')
-- Changed source refuses save, leaves scratch dirty and editable.
vim.api.nvim_buf_set_lines(b, 0, -1, false, {'different 🤖[q]{a}[moved]'})
edit(fb, {'💬: q','🤖: a','💬: rescue'})
assert(not pcall(write), 'source conflict must fail :write')
assert(vim.bo[fb].modified and lines(b)[1] == 'different 🤖[q]{a}[moved]')
assert(not pcall(function() vim.cmd('x') end), 'failed save must not close :x')
assert(vim.api.nvim_win_is_valid(fw))
F.close(b)
assert(not vim.api.nvim_win_is_valid(fw) and not vim.api.nvim_buf_is_valid(fb))
assert(vim.fn.getreg('"'):find('rescue',1,true), 'forced close rescues edits')
F.close(b) -- idempotent cleanup
-- Explicit discard via :q! leaves source untouched.
b = source('🤖[question]')
fb,fw = open(b)
edit(fb, {'💬: discard'})
vim.cmd('q!')
assert(lines(b)[1] == '🤖[question]')
assert(not vim.api.nvim_buf_is_valid(fb))
-- Same-line duplicate: source edit in the other instance cannot redirect save.
b = source('🤖[same] and 🤖[same]')
fb,fw = open(b,2)
vim.api.nvim_buf_set_text(b,0,0,0,#'🤖[same]',{'other'})
edit(fb,{'💬: chosen'})
write()
assert(lines(b)[1] == 'other and 🤖[chosen]')
F.close(b)
-- Source removal always releases resources and rescues unsaved text.
b = source('🤖[q]')
local source_win=vim.api.nvim_get_current_win()
fb,fw = open(b)
edit(fb, {'💬: source gone'})
vim.api.nvim_win_call(source_win,function() vim.api.nvim_buf_delete(b,{force=true}) end)
assert(vim.wait(1000,function() return not vim.api.nvim_win_is_valid(fw) end))
assert(vim.fn.getreg('"'):find('source gone',1,true))
-- Tiny editor geometry must be clamped to actual available cells.
vim.o.columns=20; vim.o.lines=8
b = source('🤖[q]')
fb,fw = open(b)
local cfg = vim.api.nvim_win_get_config(fw)
assert(cfg.width <= 18 and cfg.height <= 6)
F.close(b)
print('ok comment float save/conflict/movement/duplicate/discard/teardown/size')
