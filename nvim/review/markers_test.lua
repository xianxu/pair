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

-- a marker carries its 0-based (line, col) position
local mlc = M.parse_markers({ 'x', 'ab🤖[y]' })
eq(mlc[1].line, 1, 'marker line (0-based)')
eq(mlc[1].col, 2, 'marker col (0-based byte)')

-- highlight_spans (#66 M3): per-line spans for the ParleyReview* groups
local function find_hl(spans, hl)
  for _, s in ipairs(spans) do if s.hl_group == hl then return s end end
end
local hsp = M.highlight_spans({ 'before 🤖<q>[u]{a} after' })
eq(find_hl(hsp, 'ParleyReviewQuoted') ~= nil, true, 'quoted span present')
eq(find_hl(hsp, 'ParleyReviewUser') ~= nil, true, 'user span present')
eq(find_hl(hsp, 'ParleyReviewAgent') ~= nil, true, 'agent span present')
eq(hsp[1].row, 0, 'span row is 0-based')
-- byte-accurate: 'before ' = 7 bytes → 🤖 starts at 0-based col 7
eq(find_hl(hsp, 'ParleyReviewQuoted').col_start, 7, 'quoted col_start = 0-based 🤖 start')
-- '🤖<q>' spans 1-based bytes 8..14 → quoted closes at byte 14 → col_end 14
eq(find_hl(hsp, 'ParleyReviewQuoted').col_end, 14, 'quoted col_end through >')
local ss = M.highlight_spans({ '🤖~old~' })
eq(find_hl(ss, 'ParleyReviewStrike') ~= nil, true, 'strike span present')
eq(#M.highlight_spans({ 'no markers here' }), 0, 'no markers → no spans')

if fails > 0 then os.exit(1) end
print('markers_test ok')
