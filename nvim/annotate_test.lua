-- Headless tests for nvim/annotate.lua — run via `nvim -l nvim/annotate_test.lua`
-- (or `make test-lua`). Exercises the PURE marker core directly (no buffer, no
-- IO, no mocks) — this is the ARCH-PURE boundary made visible. The floating
-- prompt + read-only rewrite seam is covered by the viewer wiring tests
-- (scrollback_test.lua / changelog_test.lua) + manual interactive checks.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'annotate.lua')
local MARKER = '\240\159\164\150'  -- 🤖 = U+1F916, 4 UTF-8 bytes

local fails = 0
local function check(cond, msg)
  if not cond then
    io.stderr:write('FAIL ' .. msg .. '\n')
    fails = fails + 1
  end
end

-- ---- parse: bare + scoped ----
local bare = M.find_markers_in_line(MARKER .. '[hello]')
check(#bare == 1 and bare[1].kind == 'bare' and bare[1].Y == 'hello', 'bare parse')

local scoped = M.find_markers_in_line(MARKER .. '<sel>[c]')
check(#scoped == 1 and scoped[1].kind == 'scoped'
  and scoped[1].X == 'sel' and scoped[1].Y == 'c', 'scoped parse')

-- multiple markers on one line
local multi = M.find_markers_in_line('a ' .. MARKER .. '[one] b ' .. MARKER .. '[two]')
check(#multi == 2 and multi[1].Y == 'one' and multi[2].Y == 'two', 'two markers per line')

-- malformed (unclosed) marker is ignored, not half-parsed
local bad = M.find_markers_in_line('x ' .. MARKER .. '[unclosed')
check(#bad == 0, 'unclosed bare marker ignored')
local bad2 = M.find_markers_in_line('x ' .. MARKER .. '<sel>[unclosed')
check(#bad2 == 0, 'unclosed scoped marker ignored')

-- ---- escape/unescape round-trips (delimiter collisions) ----
check(M.unescape(M.esc_y('a]b')) == 'a]b', 'esc_y/unescape round-trip on ]')
check(M.unescape(M.esc_x('x>y]z')) == 'x>y]z', 'esc_x/unescape round-trip on >]')
check(M.unescape(M.esc_x('a\\>b')) == 'a\\>b', 'backslash-then-delim round-trip')

-- a scoped marker whose X contains an escaped `>` parses back to the raw X
local tricky = M.find_markers_in_line(MARKER .. '<' .. M.esc_x('git log >f') .. '>[why]')
check(#tricky == 1 and tricky[1].X == 'git log >f' and tricky[1].Y == 'why',
  'scoped X with embedded > survives parse')

-- ---- strip_markers: removal + whitespace trim ----
do
  local line = 'keep this ' .. MARKER .. '[note]'
  check(M.strip_markers(line, M.find_markers_in_line(line)) == 'keep this',
    'strip_markers removes marker + trims')
end

-- ---- marker_key: distinct (X,Y) pairs don't collide ----
do
  local a = M.find_markers_in_line(MARKER .. '<ab>[c]')[1]
  local b = M.find_markers_in_line(MARKER .. '<a>[bc]')[1]
  check(M.marker_key(a) ~= M.marker_key(b), 'marker_key avoids X/Y concat collision')
end

-- ---- format_extraction: baseline subtraction ----
do
  -- line 1 was present at load (baseline); line 2's marker is user-added.
  local lines    = { MARKER .. '[old]', 'ctx ' .. MARKER .. '[new]' }
  local baseline = M.collect_markers_by_line({ MARKER .. '[old]' })  -- only line 1 pre-existing
  local block    = M.format_extraction(lines, { [1] = baseline[1] })
  check(block:match('new') and not block:match('old'),
    'baseline subtracts load-time marker, keeps user-added')
end

-- empty-Y (unfinished) marker dropped
check(M.format_extraction({ MARKER .. '[]' }, {}) == '', 'empty-Y marker dropped')

-- scoped uses X as the quote; bare uses the stripped line
do
  local sb = M.format_extraction({ MARKER .. '<the selection>[q1]' }, {})
  check(sb:match('^> the selection\nq1'), 'scoped marker quotes X')
  local bb = M.format_extraction({ 'a bare line ' .. MARKER .. '[q2]' }, {})
  check(bb:match('^> a bare line\nq2'), 'bare marker quotes stripped line')
end

-- ---- source_label: per-quote prefix, count unchanged ----
do
  local labelled = M.format_extraction({ 'q ' .. MARKER .. '[why]' }, {},
    { source_label = 'change log' })
  check(labelled:match('^> %[change log%] q\nwhy'), 'source_label prefixes the quote')
  -- exactly one "> " per marker (the draft-pickup `\n> ` count must stay faithful)
  local _, count = labelled:gsub('> ', '> ')
  check(count == 1, 'source_label keeps exactly one "> " per marker')
end

-- ---- new_marker_count: matches the formatted block, label-independent ----
check(M.new_marker_count({ 'q ' .. MARKER .. '[why]' }, {}) == 1, 'new_marker_count = 1')
do
  local lines    = { MARKER .. '[old]', 'ctx ' .. MARKER .. '[new]' }
  local baseline = M.collect_markers_by_line({ MARKER .. '[old]' })
  check(M.new_marker_count(lines, { [1] = baseline[1] }) == 1,
    'new_marker_count subtracts baseline')
end
-- count ignores empty-Y markers (same as format_extraction)
check(M.new_marker_count({ MARKER .. '[]', 'x ' .. MARKER .. '[real]' }, {}) == 1,
  'new_marker_count ignores empty-Y')

if fails > 0 then
  io.stderr:write(string.format('annotate_test: %d failure(s)\n', fails))
  os.exit(1)
end
print('ok annotate_test')
