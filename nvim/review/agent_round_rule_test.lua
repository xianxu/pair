-- nvim/review/agent_round_rule_test.lua — the pair#664 rule: nvim never writes the
-- reviewed buffer or file for an agent round. `pair review apply` writes the file;
-- the pane only reloads it (`:edit!`, an on_reload, not an on_lines) and
-- decorates. Operator actions (send, marker accept/reject) still write, and a
-- positive control proves the detectors see them.
-- Run: nvim -l nvim/review/agent_round_rule_test.lua (or `make test-lua`).
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local R = dofile(here .. 'init.lua')
local round_watch = dofile(here .. 'round_watch.lua')

local fails = 0
local function ok(cond, msg)
  if not cond then io.stderr:write('FAIL ' .. msg .. '\n'); fails = fails + 1 end
end

local dir = vim.fn.tempname(); vim.fn.mkdir(dir, 'p')
local file = dir .. '/doc.md'
local round_path = dir .. '/round.json'
local function write_bytes(path, raw) local f = assert(io.open(path, 'wb')); f:write(raw); f:close() end
local function read_bytes(path) local f = io.open(path, 'rb'); if not f then return nil end
  local raw = f:read('*a'); f:close(); return raw end
write_bytes(file, 'alpha\nbeta\n')

vim.cmd('silent edit ' .. vim.fn.fnameescape(file))
local buf = vim.api.nvim_get_current_buf()

-- Detectors: (a) buffer line changes, (b) buffer writes, (c) file writes from Lua.
-- `:edit!` detaches a buffer listener (measured: on_detach, no on_lines), so
-- (a) counts a detach as the reload and re-attaches after it.
local line_changes, reloads, writes, file_writes = 0, 0, 0, 0
local function attach()
  vim.api.nvim_buf_attach(buf, false, {
    on_lines = function() line_changes = line_changes + 1 end,
    on_detach = function() reloads = reloads + 1; vim.schedule(attach) end,
  })
end
attach()
vim.api.nvim_create_autocmd('BufWritePre', { buffer = buf, callback = function() writes = writes + 1 end })
local canonical = vim.fn.resolve(file)
local function targets_doc(path) return type(path) == 'string' and vim.fn.resolve(vim.fn.fnamemodify(path, ':p')) == canonical end
local writefile, io_open, rename = vim.fn.writefile, io.open, os.rename
vim.fn.writefile = function(lines, path, ...) if targets_doc(path) then file_writes = file_writes + 1 end return writefile(lines, path, ...) end
io.open = function(path, mode) if mode and mode:find('[wa+]') and targets_doc(path) then file_writes = file_writes + 1 end return io_open(path, mode) end
os.rename = function(from, to) if targets_doc(to) then file_writes = file_writes + 1 end return rename(from, to) end

R.start({ buf = buf, file = file, tag = 'rule', round_watch_opts = { path = round_path, interval = 10 } })

-- The engine's write, made with the saved io.open so it is not counted.
local function engine_round(raw, spans)
  local f = io_open(file, 'wb'); f:write(raw); f:close()
  local artifact = vim.json.encode({ file = file, round = { result_hash = round_watch.hash(file), spans = spans } })
  f = io_open(round_path, 'wb'); f:write(artifact); f:close()
end
local result = 'alpha\nBETA\n'
engine_round(result, { { start = { 1, 0 }, ['end'] = { 1, 4 }, old = 'beta', new = 'BETA', explain = 'louder' } })

local reloaded = vim.wait(3000, function() return vim.api.nvim_buf_get_lines(buf, 1, 2, false)[1] == 'BETA' end, 10)
ok(reloaded and reloads == 1, 'the pane reloads the applied round (reloads=' .. reloads .. ')')
ok(line_changes == 0, 'an agent round changes no buffer lines through nvim (on_lines=' .. line_changes .. ')')
ok(writes == 0, 'an agent round writes no buffer (BufWritePre=' .. writes .. ')')
ok(file_writes == 0, 'an agent round writes no file from Lua (' .. file_writes .. ')')
ok(read_bytes(file) == result, 'the file holds exactly what the engine wrote')
ok(not vim.bo[buf].modified, 'the reloaded buffer is unmodified')
ok(#vim.diagnostic.get(buf) == 1, 'the round is decorated')

-- Positive controls: every detector is live after the reload. An operator edit
-- is an on_lines; the operator's send is a buffer write; a Lua write of the doc
-- trips the file spies.
vim.wait(100, function() return false end, 10) -- let the re-attach run
vim.api.nvim_buf_set_lines(buf, 0, 1, false, { 'operator edit' })
ok(line_changes == 1, 'the line detector sees an operator edit after the reload (on_lines=' .. line_changes .. ')')
ok(R.human_round(buf, 'updated') == true, 'the operator send saves')
ok(writes == 1, 'the write detector sees an operator write (BufWritePre=' .. writes .. ')')
local raw = read_bytes(file)
vim.fn.writefile(vim.split(raw, '\n', { plain = true }), file, 'b')
local h = io.open(file, 'wb'); h:write(raw); h:close()
ok(file_writes == 2, 'the file spies see Lua writes of the doc (' .. file_writes .. ')')
file_writes = 0

-- A declined round (the operator is mid-edit): nvim writes nothing either.
vim.api.nvim_buf_set_lines(buf, 0, 1, false, { 'operator typing' })
local typed = line_changes
engine_round('alpha\nBETA\ngamma\n', {})
vim.wait(200, function() return false end, 10)
ok(line_changes == typed and writes == 1 and file_writes == 0, 'a declined round writes nothing')
ok(vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1] == 'operator typing', 'a declined round keeps the operator buffer')
ok(read_bytes(file) == 'alpha\nBETA\ngamma\n', 'a declined round leaves the engine file alone')

-- An empty round (no edits, pair#690) over the operator's unsaved text: the turn
-- ends, and nvim writes nothing and reloads nothing.
local ended, reloads_before = 0, reloads
R.after_applied_round = function() ended = ended + 1 end
local eh = round_watch.hash(file)
local ef = io_open(round_path, 'wb')
ef:write(vim.json.encode({ file = file, round = { base_hash = eh, result_hash = eh, at = '2026-10-19T09:00:00Z', spans = {} }, body = '' }))
ef:close()
vim.wait(3000, function() return ended > 0 end, 10)
ok(ended == 1, 'an empty round ends the turn (after_applied_round=' .. ended .. ')')
ok(line_changes == typed and writes == 1 and file_writes == 0 and reloads == reloads_before, 'an empty round writes and reloads nothing')
ok(vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1] == 'operator typing' and vim.bo[buf].modified, 'an empty round keeps the operator buffer')
R.after_applied_round = nil

R.stop(buf)
vim.fn.writefile, io.open, os.rename = writefile, io_open, rename
vim.fn.delete(dir, 'rf')
if fails > 0 then os.exit(1) end
print('agent_round_rule_test ok')
