-- nvim/review/markers_test.lua — run via `nvim -l nvim/review/markers_test.lua`.
-- Pure Lua; no vim API. Exits non-zero on failure.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'markers.lua')
local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

-- ready: last section is a non-empty []
local m = M.parse_markers({ 'before 🤖[fix this] after' })
eq(#m, 1, 'one marker')
eq(m[1].sections[1].type, 'user', 'user section')
eq(m[1].sections[1].text, 'fix this', 'section text')
eq(m[1].ready, true, 'ready (last [])')
eq(m[1].pending, false, 'not pending')

-- quoted + agent → pending
local m2 = M.parse_markers({ '🤖<quoted>{agent note}' })
eq(m2[1].quoted.text, 'quoted', 'quoted text')
eq(m2[1].sections[1].type, 'agent', 'agent section')
eq(m2[1].pending, true, 'pending (last {})')
eq(m2[1].ready, false, 'not ready')

-- strike
local m3 = M.parse_markers({ '🤖~old~' })
eq(m3[1].strike.text, 'old', 'strike text')

-- fenced markers are excluded
local m4 = M.parse_markers({ '```', '🤖[in fence]', '```', '🤖[real]' })
eq(#m4, 1, 'fenced marker excluded')
eq(m4[1].sections[1].text, 'real', 'only the real marker')

-- alternating sections preserve order
local m5 = M.parse_markers({ '🤖[a]{b}[c]' })
eq(#m5[1].sections, 3, 'three sections')
eq(m5[1].sections[1].text .. m5[1].sections[2].text .. m5[1].sections[3].text, 'abc', 'order abc')

-- last-section rule: ending in {} → pending, not ready
local m6 = M.parse_markers({ '🤖[a]{b}' })
eq(m6[1].pending, true, 'last {} → pending')
eq(m6[1].ready, false, 'last {} → not ready')

-- strike is never "ready", even with a trailing []
local m7 = M.parse_markers({ '🤖~old~[reply]' })
eq(m7[1].ready, false, 'strike never ready')
eq(m7[1].strike.text, 'old', 'strike kept alongside section')

-- a marker inside an INLINE-code span is excluded
local mi = M.parse_markers({ 'see `🤖[x]` here' })
eq(#mi, 0, 'marker inside inline code excluded')

-- a section may span multiple lines (within the budget)
local mml = M.parse_markers({ '🤖[line one', 'line two]' })
eq(#mml, 1, 'multi-line section parses')
eq(mml[1].sections[1].text, 'line one\nline two', 'multi-line section text')

-- a stray opener beyond MULTILINE_LINE_BUDGET (50) yields no section
local budget = { '🤖{' }
for _ = 1, 60 do budget[#budget + 1] = 'x' end
budget[#budget + 1] = '}'
eq(#M.parse_markers(budget), 0, 'stray { beyond the newline budget yields no marker')

if fails > 0 then os.exit(1) end
print('markers_test ok')
