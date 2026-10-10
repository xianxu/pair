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

-- escaped delimiter collisions round-trip through the parser
local tricky_text = 'a < b > c ] d \\ e'
local tricky = M.parse_markers({ '🤖<' .. M.esc_quote(tricky_text) .. '>[human]' })
eq(#tricky, 1, 'escaped quote marker parses')
eq(tricky[1].quoted.text, tricky_text, 'escaped quote text round-trips')
eq(tricky[1].raw, '🤖<' .. M.esc_quote(tricky_text) .. '>[human]', 'escaped raw spans the whole marker')

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

-- a stray opener beyond MULTILINE_LINE_BUDGET (200) yields no section
local budget = { '🤖{' }
for _ = 1, 210 do budget[#budget + 1] = 'x' end
budget[#budget + 1] = '}'
eq(#M.parse_markers(budget), 0, 'stray { beyond the newline budget yields no marker')

-- a marker carries its 0-based (line, col) position
local mlc = M.parse_markers({ 'x', 'ab🤖[y]' })
eq(mlc[1].line, 1, 'marker line (0-based)')
eq(mlc[1].col, 2, 'marker col (0-based byte)')

-- spans_multiline (#89 M1): multi-line-aware ParleyReview* spans, supersedes the
-- per-line highlight_spans. Spans carry {row, col, end_row, end_col} (0-based,
-- end_col exclusive) and may cross rows.
local function find_hl(spans, hl)
  for _, s in ipairs(spans) do if s.hl_group == hl then return s end end
end
-- single line: byte-accurate, preserving the retired highlight_spans behavior.
local hsp = M.spans_multiline({ 'before 🤖<q>[u]{a} after' })
eq(find_hl(hsp, 'ParleyReviewQuoted') ~= nil, true, 'quoted span present')
eq(find_hl(hsp, 'ParleyReviewUser') ~= nil, true, 'user span present')
eq(find_hl(hsp, 'ParleyReviewAgent') ~= nil, true, 'agent span present')
local q1 = find_hl(hsp, 'ParleyReviewQuoted')
eq(q1.row, 0, 'quoted row is 0-based')
-- byte-accurate: 'before ' = 7 bytes → 🤖 starts at 0-based col 7
eq(q1.col, 7, 'quoted col = 0-based 🤖 start')
eq(q1.end_row, 0, 'quoted end_row 0 (single line)')
-- '🤖<q>' → '>' at 1-based byte 14 → exclusive end_col 14
eq(q1.end_col, 14, 'quoted end_col exclusive through >')
local ss = M.spans_multiline({ '🤖~old~' })
eq(find_hl(ss, 'ParleyReviewStrike') ~= nil, true, 'strike span present')
eq(#M.spans_multiline({ 'no markers here' }), 0, 'no markers → no spans')
-- multi-line: a 🤖<…> whose quoted body spans two lines → end_row > row, span
-- starting at the 🤖 itself (col 7 after 'before ').
local ml = M.spans_multiline({ 'before 🤖<first', 'second>[note] after' })
local mq = find_hl(ml, 'ParleyReviewQuoted')
eq(mq ~= nil, true, 'multi-line quoted span present')
eq(mq.row, 0, 'multi-line quoted starts row 0')
eq(mq.col, 7, 'multi-line quoted starts at 🤖 col 7')
eq(mq.end_row, 1, 'multi-line quoted ends row 1')

-- budget (#89 M1): a 🤖<…> whose quoted body has >50 newlines still closes at
-- budget 200 (conflict hunks can be large; the reconciler caps them at 200).
do
  local blines = { '🤖<L0' }
  for i = 1, 60 do blines[#blines + 1] = 'L' .. i end
  blines[#blines + 1] = 'END>[x]'
  local bm = M.parse_markers(blines)
  eq(#bm, 1, '60-line quoted marker parses')
  eq(bm[1].quoted ~= nil, true, '60-line quoted body closes (budget ≥ 60)')
end

-- An escaped opening backtick is text, including encoded turn payloads.
local escaped_tick = M.parse_markers({ '🤖{\\`}[\\`]' })
eq(escaped_tick[1] and #escaped_tick[1].sections, 2, 'escaped ticks cannot mask a section boundary')

-- #426: raw payloads and scanner diagnostics share all parser exclusions.
eq(m[1].sections[1].raw_text, 'fix this', 'raw turn retained')
eq(type(M.scan), 'function', 'shared scanner exists')
if M.scan then
  local raw = 'xx 🤖<anchor>[ok]{broken'
  local scan = M.scan({ raw })
  eq(#scan.markers, 1, 'successful prefix retained')
  eq(scan.markers[1].complete, false, 'unfinished continuation ineligible')
  eq(scan.markers[1].end_row, 0, 'exclusive marker end row')
  eq(scan.markers[1].end_col, #('xx 🤖<anchor>[ok]'), 'exclusive successful end col')
  eq(#scan.diagnostics, 1, 'broken continuation diagnosed')
  eq(scan.diagnostics[1].col, 3, 'diagnostic starts at marker')
  eq(scan.diagnostics[1].end_col, #raw, 'diagnostic through EOL')
  eq(scan.diagnostics[1].kind, 'malformed', 'diagnostic kind')
  for _, opener in ipairs({ '[', '{', '<' }) do
    local bad = M.scan({ '🤖' .. opener .. 'broken' })
    eq(#bad.markers, 0, 'wholly malformed excluded ' .. opener)
    eq(#bad.diagnostics, 1, 'wholly malformed diagnosed ' .. opener)
  end
  local excluded = M.scan({ '```', '🤖{broken', '```', '`🤖{broken`', '🤖[okay]' })
  eq(#excluded.diagnostics, 0, 'code malformed markers excluded')
  eq(#excluded.markers, 1, 'code context shared')
  eq(excluded.markers[1].complete, true, 'valid complete flag')
  local legacy = M.scan({ '🤖[old', 'multiline]' })
  eq(legacy.markers[1].complete, true, 'legacy multiline complete')
  eq(legacy.markers[1].end_row, 1, 'legacy end row')
  eq(legacy.markers[1].end_col, #'multiline]', 'legacy end col')
  eq(#legacy.diagnostics, 0, 'legacy is not malformed')
  for n = 0, 30 do
    local prefix = string.rep('字', n)
    local document = { prefix .. '🤖[done]{bad', '`🤖[bad`', '```', '🤖{bad', '```' }
    local generated = M.scan(document)
    eq(#generated.markers, 1, 'generated successful prefix')
    eq(generated.markers[1].complete, false, 'generated ineligible prefix')
    eq(#generated.diagnostics, 1, 'generated exclusion consistency')
    eq(generated.diagnostics[1].col, #prefix, 'generated UTF-8 byte col')
  end
end

-- Markdown fences: up to three spaces, same delimiter, matching run length.
for _, char in ipairs({ '`', '~' }) do
  for indent = 0, 3 do
    for width = 3, 6 do
      local pad, fence = string.rep(' ', indent), string.rep(char, width)
      local other = char == '`' and '~~~' or '```'
      local lines = { pad .. fence .. 'lua', '🤖[hidden]', other, '🤖{broken',
        string.rep(char, width - 1), '🤖[still hidden]', fence .. ' not a close',
        '🤖[also hidden]', pad .. fence .. char .. '  ', '🤖[visible]' }
      local scan = M.scan(lines)
      eq(#scan.markers, 1, 'matching Markdown fence retains inner markers as code')
      eq(scan.markers[1] and scan.markers[1].line, 9, 'only outside Markdown fence parses')
      eq(#scan.diagnostics, 0, 'malformed marker inside Markdown fence ignored')
    end
  end
end
local unclosed_fence = M.scan({ '   ~~~~text', '🤖[hidden]' })
eq(#unclosed_fence.markers, 0, 'unclosed tilde fence excludes through EOF')
eq(#M.scan({ '    ~~~', '🤖[ordinary]' }).markers, 1, 'four spaces do not open fence')
eq(#M.scan({ '```info`invalid', '🤖[ordinary]' }).markers, 1, 'backtick info cannot contain backticks')

if fails > 0 then os.exit(1) end
print('markers_test ok')
