-- nvim/review/record_test.lua — run via `nvim -l nvim/review/record_test.lua`
-- (or `make test-lua`). Pure Lua + vim.json; no buffer/IO. Exits non-zero on
-- failure so the make target fails loudly. Models nvim/slug_test.lua.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'record.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

local recs = {
  { old = 'teh', occurrence = 2, new = 'the', new_occurrence = 1, explain = 'typo' },
  { old = 'old API', occurrence = 1, new = 'new API', explain = 'v2 is gone' },
}

-- round-trip (incl. the apply-added new_occurrence field — vim.json keeps it)
local s = M.encode(recs)
local back = M.decode(s)
eq(#back, 2, 'decode count')
eq(back[1].old, 'teh', 'decode old')
eq(back[1].occurrence, 2, 'decode occurrence')
eq(back[1].new_occurrence, 1, 'decode new_occurrence')
eq(back[2].explain, 'v2 is gone', 'decode explain')

-- embed/extract in a commit body that also has prose
local body = M.embed_in_body('three edits', recs)
local ex = M.extract_from_body(body)
eq(#ex, 2, 'extract count')
eq(ex[1].new, 'the', 'extract new')
eq(M.extract_from_body('no block here'), nil, 'extract returns nil when absent')

-- a real docflow body carries a Co-Authored-By trailer AFTER the fenced block;
-- extraction must still find the block (non-greedy match).
local body2 = body .. '\nCo-Authored-By: Claude <noreply@anthropic.com>'
eq(#M.extract_from_body(body2), 2, 'extract works with trailer after block')

if fails > 0 then os.exit(1) end
print('record_test ok')
