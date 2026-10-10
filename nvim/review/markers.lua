-- nvim/review/markers.lua — pure 🤖 review-marker parser (issue #66 M2).
-- Ported from parley's review/init.lua (the LLM-invoke half is dropped). No vim
-- API — operates on a `lines` array → marker records.
--
-- Grammar: 🤖<quoted>?(~strike~)?([user]|{agent})*
--   <> quoted body (≤1, first slot only); ~~ strike = deletion proposal
--   [] human turns; {} agent turns
--   delimiter text is backslash-escaped via nvim/marker_codec.lua
--   ready   = last section is a non-empty [] (human spoke last; strikes never ready)
--   pending = last section is a non-empty {} (agent asked, awaiting human)
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local marker_codec = dofile(here .. '../marker_codec.lua')
local reconstruct = dofile(here .. 'reconstruct.lua')

local MARKER_CHAR = "🤖"
local MARKER_BYTE_LEN = 4
-- Per-section newline budget — a stray opener absorbs at most this many lines.
-- Raised to 200 for #89: reconcile conflict markers wrap the human's changed
-- hunk (which the reconciler caps at 200 lines), so the quoted body can be large.
local MULTILINE_LINE_BUDGET = 200

-- Find the close matching `open` at `start`, tracking nesting depth.
-- opts.budget: max newlines crossed before giving up; opts.is_excluded(off): a
-- bracket at an excluded offset (code span) doesn't affect depth.
local function find_matching_bracket(text, start, open, close, opts)
  opts = opts or {}
  local budget = opts.budget
  local is_excluded = opts.is_excluded
  local depth = 0
  local newlines = 0
  for i = start, #text do
    local ch = text:sub(i, i)
    if ch == "\n" then
      newlines = newlines + 1
      if budget and newlines > budget then return nil end
    elseif (ch == open or ch == close) and not (is_excluded and is_excluded(i))
        and not marker_codec.is_escaped(text, i, start) then
      if ch == open then
        depth = depth + 1
      else
        depth = depth - 1
        if depth == 0 then return i end
      end
    end
  end
  return nil
end

-- Parse a marker at `pos` → sections, cursor (one past last bracket), quoted, strike.
local function parse_marker_sections(text, pos, byte_len, opts)
  local cursor = pos + (byte_len or MARKER_BYTE_LEN)
  local sections = {}
  local quoted = nil
  local strike = nil

  -- Optional leading <...> OR ~...~ (mutually exclusive, first slot only).
  if cursor <= #text then
    local ch = text:sub(cursor, cursor)
    if ch == "<" then
      local close = find_matching_bracket(text, cursor, "<", ">", opts)
      if close then
        quoted = { text = marker_codec.unescape(text:sub(cursor + 1, close - 1)), byte_start = cursor, byte_end = close }
        cursor = close + 1
      end
    elseif ch == "~" then
      -- tildes don't nest and are bounded to one line (common in prose).
      local close = text:find("~", cursor + 1, true)
      local nl = text:find("\n", cursor + 1, true)
      if close and (not nl or close < nl) then
        strike = { text = marker_codec.unescape(text:sub(cursor + 1, close - 1)), byte_start = cursor, byte_end = close }
        cursor = close + 1
      end
    end
  end

  while cursor <= #text do
    local ch = text:sub(cursor, cursor)
    if ch == "[" then
      local close = find_matching_bracket(text, cursor, "[", "]", opts)
      if not close then break end
        table.insert(sections, { type = "user", raw_text = text:sub(cursor + 1, close - 1), text = marker_codec.unescape(text:sub(cursor + 1, close - 1)), byte_start = cursor, byte_end = close })
      cursor = close + 1
    elseif ch == "{" then
      local close = find_matching_bracket(text, cursor, "{", "}", opts)
      if not close then break end
      table.insert(sections, { type = "agent", raw_text = text:sub(cursor + 1, close - 1), text = marker_codec.unescape(text:sub(cursor + 1, close - 1)), byte_start = cursor, byte_end = close })
      cursor = close + 1
    else
      break
    end
  end

  return sections, cursor, quoted, strike
end

local function in_code_fence(fence_ranges, line_idx)
  for _, range in ipairs(fence_ranges) do
    if line_idx >= range[1] and line_idx <= range[2] then return true end
  end
  return false
end

local function compute_fence_ranges(lines)
  local ranges, opening = {}, nil
  for i, line in ipairs(lines) do
    local indent, run, tail = line:match('^( *)(`+)(.*)$')
    if not run then indent, run, tail = line:match('^( *)(~+)(.*)$') end
    if run and #indent <= 3 and #run >= 3 then
      local char = run:sub(1, 1)
      if opening then
        if char == opening.char and #run >= opening.width and tail:match('^[ \t]*$') then
          ranges[#ranges + 1] = { opening.row, i - 1 }
          opening = nil
        end
      elseif char ~= '`' or not tail:find('`', 1, true) then
        opening = { row = i - 1, char = char, width = #run }
      end
    end
  end
  if opening then ranges[#ranges + 1] = { opening.row, #lines - 1 } end
  return ranges
end

-- {start, finish} byte ranges for inline-code spans on a line (` … ` / ``` … ```).
local function inline_code_ranges(line)
  local ranges = {}
  local i = 1
  while i <= #line do
    if line:sub(i, i) == '`' and marker_codec.is_escaped(line, i) then
      i = i + 1
      goto next_byte
    end
    local bt_start = i
    while i <= #line and line:sub(i, i) == "`" do i = i + 1 end
    local bt_len = i - bt_start
    if bt_len > 0 then
      local delimiter = string.rep("`", bt_len)
      local close = line:find(delimiter, i, true)
      while close and marker_codec.is_escaped(line, close) do
        close = line:find(delimiter, close + 1, true)
      end
      if close then
        table.insert(ranges, { bt_start, close + bt_len - 1 })
        i = close + bt_len
      end
    else
      i = i + 1
    end
    ::next_byte::
  end
  return ranges
end

-- Parse 🤖 markers over the whole buffer (sections may span lines, bounded).
M.scan = function(lines)
  local fence_ranges = compute_fence_ranges(lines)
  local doc = table.concat(lines, "\n")

  local line_starts = reconstruct.line_starts(lines)

  local excluded = {}
  for i, line in ipairs(lines) do
    local base = line_starts[i]
    if in_code_fence(fence_ranges, i - 1) then
      table.insert(excluded, { base, base + #line })
    else
      for _, r in ipairs(inline_code_ranges(line)) do
        table.insert(excluded, { base + r[1] - 1, base + r[2] - 1 })
      end
    end
  end
  local function is_excluded(offset)
    for _, r in ipairs(excluded) do
      if r[1] > offset then break end
      if offset <= r[2] then return true end
    end
    return false
  end

  local opts = { budget = MULTILINE_LINE_BUDGET, is_excluded = is_excluded }
  local markers, diagnostics = {}, {}
  local search_start = 1
  while true do
    local pos = doc:find(MARKER_CHAR, search_start, true)
    if not pos then break end
    if is_excluded(pos) then
      search_start = pos + MARKER_BYTE_LEN
      goto continue
    end

    local sections, end_pos, quoted, strike = parse_marker_sections(doc, pos, MARKER_BYTE_LEN, opts)
    local next_byte = doc:sub(end_pos, end_pos)
    local complete = next_byte ~= '[' and next_byte ~= '{'
    if #sections == 0 and not quoted and not strike and next_byte == '<' then
      complete = false
    end
    local line0, col0 = reconstruct.pos_of(line_starts, pos)
    local end_row, end_col = reconstruct.pos_of(line_starts, end_pos)
    if not complete then
      diagnostics[#diagnostics + 1] = {
        row = line0, col = col0, end_row = line0, end_col = #lines[line0 + 1], kind = 'malformed',
      }
    end
    if strike and strike.text == "" then strike = nil end
    if #sections > 0 or quoted or strike then
      local last = sections[#sections]
      local ready = (not strike) and last and last.type == "user" and last.text ~= "" or false
      local pending = last and last.type == "agent" and last.text ~= "" or false
      table.insert(markers, {
        line = line0, col = col0, end_row = end_row, end_col = end_col, complete = complete,
        quoted = quoted, strike = strike, sections = sections,
        ready = ready, pending = pending,
        raw = doc:sub(pos, end_pos - 1),
      })
    end
    search_start = end_pos
    ::continue::
  end

  return { markers = markers, diagnostics = diagnostics }
end

-- Legacy successful-prefix API; compact consumers use scan and completeness.
M.parse_markers = function(lines)
  return M.scan(lines).markers
end

M.esc_quote = marker_codec.esc_quote
M.unescape = marker_codec.unescape

-- Exposed for the highlighter (the per-line section seam).
M._parse_marker_sections = parse_marker_sections

-- Multi-line highlight spans for 🤖 markers (issue #89 M1; supersedes the per-line
-- highlight_spans). Pure: lines → spans { row, col (0-based start), end_row,
-- end_col (0-based byte, exclusive), hl_group }. Derived from the multi-line
-- parse_markers, so a 🤖<…>/section span may cross rows (end_row > row) — the
-- reconcile conflict markers (#89) are inherently multi-line. Groups:
-- ParleyReviewQuoted (🤖<…>), ParleyReviewStrike (🤖~…~), ParleyReviewUser ([…]),
-- ParleyReviewAgent ({…}). Byte-accurate: quoted/strike start at the 🤖 itself,
-- sections at their bracket — matching the retired per-line highlighter.
function M.spans_multiline(lines)
  local line_starts = reconstruct.line_starts(lines)
  local spans = {}
  -- closer = 1-based doc offset of the closing char; exclusive end = pos just after it.
  local function push(sr, sc, closer, hl)
    local er, ec = reconstruct.pos_of(line_starts, closer + 1)
    spans[#spans + 1] = { row = sr, col = sc, end_row = er, end_col = ec, hl_group = hl }
  end
  for _, m in ipairs(M.parse_markers(lines)) do
    if m.quoted then push(m.line, m.col, m.quoted.byte_end, 'ParleyReviewQuoted') end
    if m.strike and m.strike.text ~= '' then push(m.line, m.col, m.strike.byte_end, 'ParleyReviewStrike') end
    for _, s in ipairs(m.sections) do
      local sr, sc = reconstruct.pos_of(line_starts, s.byte_start)
      push(sr, sc, s.byte_end, s.type == 'agent' and 'ParleyReviewAgent' or 'ParleyReviewUser')
    end
  end
  return spans
end

return M
