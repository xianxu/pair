-- nvim/review/round_watch_test.lua — run via `nvim -l nvim/review/round_watch_test.lua`.
-- The pane's side of `pair review apply` (pair#661 M2): which rounds it takes,
-- how spans become decorations, and the operator lock it holds during its turn.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local W = dofile(here .. 'round_watch.lua')
local L = dofile(here .. 'oplock.lua')
local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %s want %s\n', msg, vim.inspect(got), vim.inspect(want)))
    fails = fails + 1
  end
end

local dir = vim.fn.tempname(); vim.fn.mkdir(dir, 'p')
local file = dir .. '/doc.md'
vim.fn.writefile({ 'x' }, file)
local function art(f, hash) return { file = f, round = { result_hash = hash, spans = {} } } end

-- decide: the pane's file, a result not seen yet; nothing else.
eq(W.decide(art(file, 'h1'), file, nil), true, 'new round of this file')
eq(W.decide(art(file, 'h1'), file, 'h1'), false, 'a seen round is not replayed')
eq(W.decide(art(dir .. '/other.md', 'h2'), file, nil), false, 'another file')
eq(W.decide({ file = file }, file, nil), false, 'no round')
eq(W.decide(nil, file, nil), false, 'no artifact')
eq(W.decide(art(dir .. '/./doc.md', 'h3'), file, nil), true, 'paths compare canonically')

-- decorations: a highlight per changed span (none for a deletion or a marker
-- proposal), and the reason, or "changed by the agent", as the diagnostic.
local d = W.decorations({ spans = {
  { start = { 1, 4 }, ['end'] = { 1, 8 }, old = 'quick', new = 'slow', explain = 'pace' },
  { start = { 2, 0 }, ['end'] = { 2, 0 }, old = 'gone\n', new = '' },
  { start = { 3, 0 }, ['end'] = { 3, 20 }, old = 'a', new = '🤖~a~{b}' },
  { start = { 4, 0 }, ['end'] = { 4, 3 }, old = 'x', new = 'new' },
} })
eq(#d.hl, 2, 'highlights skip the deletion and the proposal')
eq(d.hl[1].line, 1, 'span line'); eq(d.hl[1].col, 4, 'span byte column'); eq(d.hl[1].end_col, 8, 'span end column')
eq(#d.diags, 4, 'every span gets a diagnostic')
eq(d.diags[1].message, 'pace', 'the explain is the message')
eq(d.diags[2].message, 'changed by the agent (deleted: gone\n)', 'a deletion names what went')
eq(d.diags[4].message, 'changed by the agent', 'no explain')

-- watch: the round present at start is not replayed; the next one is.
local path = dir .. '/round.json'
local function put(a) vim.fn.writefile({ vim.json.encode(a) }, path) end
put(art(file, 'old'))
local got = {}
local stop = W.watch(file, function(a) got[#got + 1] = a.round.result_hash end, { path = path, interval = 10 })
vim.wait(60, function() return false end)
eq(#got, 0, 'the existing round is not replayed')
put(art(file, 'new'))
vim.wait(500, function() return #got > 0 end)
eq(got[1], 'new', 'a new round is delivered')
stop()

-- oplock: taken on the operator's turn, refused if someone else's lock is
-- there, dropped only when it is ours.
eq(L.take(file), true, 'take the lock')
local text = table.concat(vim.fn.readfile(L.path(file)), '\n')
eq(text:match('^holder: operator\ntime: %d%d%d%d%-%d%d%-%d%dT%d%d:%d%d:%d%dZ$') ~= nil, true, 'S1 lock format')
L.drop(file)
eq(vim.uv.fs_stat(L.path(file)), nil, 'drop removes our lock')
vim.fn.writefile({ 'holder: operator' }, L.path(file))
eq(L.take(file), false, "someone else's lock is left alone")
L.drop(file)
eq(vim.uv.fs_stat(L.path(file)) ~= nil, true, "drop never removes a lock we didn't take")

vim.fn.delete(dir, 'rf')
if fails > 0 then os.exit(1) end
print('ok — round_watch and oplock')
