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
-- A round whose result is the file as it now is on disk, for the watch tests.
local function live(tag) vim.fn.writefile({ tag }, file); return art(file, W.hash(file)) end

-- decide: the pane's file, a result not seen yet; nothing else.
eq(W.decide(art(file, 'h1'), file, nil), true, 'new round of this file')
eq(W.decide(art(file, 'h1'), file, W.key(art(file, 'h1'))), false, 'a seen round is not replayed')
eq(W.decide(art(dir .. '/other.md', 'h2'), file, nil), false, 'another file')
eq(W.decide({ file = file }, file, nil), false, 'no round')
eq(W.decide(nil, file, nil), false, 'no artifact')
eq(W.decide(art(dir .. '/./doc.md', 'h3'), file, nil), true, 'paths compare canonically')
eq(W.decide(art(file, 'h4'), file, nil, 'h4'), true, 'the file still is the round result')
eq(W.decide(art(file, 'h4'), file, nil, 'other'), false, 'a file that moved on makes the round stale')
eq(W.hash(file), vim.fn.sha256('x\n'):sub(1, 16), 'the engine whole-file hash')

-- key and is_empty (pair#690): a no-edit reply keeps the hash of the round
-- before it, so a round is the hash and its time; an empty round is one whose
-- result is its base, with no spans.
local function at(hash, t) local a = art(file, hash); a.round.at = t; return a end
eq(W.decide(at('h5', 't2'), file, W.key(at('h5', 't1'))), true, 'the same hash at a new time is a new round')
eq(W.decide(at('h5', 't1'), file, W.key(at('h5', 't1'))), false, 'the same hash and time is the seen round')
eq(W.key({ file = file, round = { spans = {} } }), nil, 'no hash, no key')
eq(W.is_empty({ base_hash = 'h', result_hash = 'h', spans = {} }), true, 'result is base, no spans: empty')
eq(W.is_empty({ base_hash = 'g', result_hash = 'h', spans = {} }), false, 'a moved result is not empty')
eq(W.is_empty({ base_hash = 'h', result_hash = 'h', spans = { {} } }), false, 'spans are not empty')
eq(W.is_empty({ base_hash = 'h', result_hash = 'h', spans = vim.NIL }), true, 'null spans are none')

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
local mixed = W.decorations({ presentation = 'direct', spans = {
  { start = {0,0}, ['end'] = {0,1}, old = 'a', new = 'b', presentation = 'highlight' },
  { start = {1,0}, ['end'] = {1,1}, old = 'a', new = 'b', presentation = 'propose' },
  { start = {2,0}, ['end'] = {2,1}, old = 'a', new = 'b', presentation = 'direct' },
  { start = {3,0}, ['end'] = {3,1}, old = 'a', new = 'b' },
} })
eq(#mixed.hl, 1, 'only explicit highlight span receives change ink')
eq(#mixed.diags, 2, 'direct spans and inherited direct have no diagnostics')

local path = dir .. '/round.json'
local function put(a) vim.fn.writefile({ vim.json.encode(a) }, path) end
put(live('old'))
local got = {}
local stop = W.watch(file, function(a) got[#got + 1] = a.round.result_hash end, { path = path, interval = 10 })
vim.wait(60, function() return false end)
eq(#got, 0, 'the existing round is not replayed')
local fresh = live('new')
put(fresh)
vim.wait(500, function() return #got > 0 end)
eq(got[1], fresh.round.result_hash, 'a new round is delivered')
put(art(file, 'not-the-file'))
vim.wait(60, function() return false end)
eq(#got, 1, 'a round the file no longer matches is not delivered')
stop()

-- A round the pane declines (a modified buffer) is delivered again.
local tries = 0
stop = W.watch(file, function() tries = tries + 1; return tries > 1 end, { path = path, interval = 10 })
put(live('again'))
vim.wait(500, function() return tries >= 2 end)
eq(tries >= 2, true, 'a declined round is redelivered')
stop()

-- oplock: taken on the operator's turn, refused if someone else's lock is
-- there, dropped only when it is ours.
eq(L.take(file), true, 'take the lock')
local text = table.concat(vim.fn.readfile(L.path(file)), '\n')
eq(text:match('^holder: operator\ntime: %d%d%d%d%-%d%d%-%d%dT%d%d:%d%d:%d%dZ\npid: %d+$') ~= nil, true, 'S1 lock format, with the owner pid')
L.drop(file)
eq(vim.uv.fs_stat(L.path(file)), nil, 'drop removes our lock')
vim.fn.writefile({ 'holder: operator' }, L.path(file))
eq(L.take(file), false, "someone else's lock is left alone")
L.drop(file)
eq(vim.uv.fs_stat(L.path(file)) ~= nil, true, "drop never removes a lock we didn't take")

-- A lock left by a pane that died is reclaimed; a live owner's is not (BR-11).
os.remove(L.path(file))
local dead = 2147483000 -- far above any live pid on a test host
vim.fn.writefile({ 'holder: operator', 'time: 2026-10-18T00:00:00Z', 'pid: ' .. dead }, L.path(file))
eq(L.reclaim(file), true, "a dead pane's lock is reclaimed")
eq(vim.uv.fs_stat(L.path(file)), nil, 'reclaim removed it')
vim.fn.writefile({ 'holder: operator', 'time: 2026-10-18T00:00:00Z', 'pid: ' .. dead }, L.path(file))
eq(L.take(file), true, "take reclaims a dead pane's lock")
L.drop(file)
vim.fn.writefile({ 'holder: operator', 'time: 2026-10-18T00:00:00Z', 'pid: ' .. vim.uv.os_getppid() }, L.path(file))
eq(L.take(file), false, "a live owner's lock is left alone")
eq(L.reclaim(file), false, "reclaim leaves a live owner's lock")
os.remove(L.path(file))

vim.fn.delete(dir, 'rf')
if fails > 0 then os.exit(1) end
print('ok — round_watch and oplock')
