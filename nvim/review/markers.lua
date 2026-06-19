-- nvim/review/markers.lua — pure 🤖 review-marker parser (issue #66 M2).
-- Ported from parley's review/init.lua (the LLM-invoke half is dropped). No vim
-- API — operates on a `lines` array → marker records.
--
-- Grammar: 🤖<quoted>?(~strike~)?([user]|{agent})*
--   <> quoted body (≤1, first slot only); ~~ strike = deletion proposal
--   [] human turns; {} agent turns
--   ready   = last section is a non-empty [] (human spoke last; strikes never ready)
--   pending = last section is a non-empty {} (agent asked, awaiting human)
local M = {}

local MARKER_CHAR = "🤖"
local MARKER_BYTE_LEN = 4
-- Per-section newline budget — a stray opener absorbs at most this many lines.
local MULTILINE_LINE_BUDGET = 50

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
    elseif (ch == open or ch == close) and not (is_excluded and is_excluded(i)) then
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
        quoted = { text = text:sub(cursor + 1, close - 1), byte_start = cursor, byte_end = close }
        cursor = close + 1
      end
    elseif ch == "~" then
      -- tildes don't nest and are bounded to one line (common in prose).
      local close = text:find("~", cursor + 1, true)
      local nl = text:find("\n", cursor + 1, true)
      if close and (not nl or close < nl) then
        strike = { text = text:sub(cursor + 1, close - 1), byte_start = cursor, byte_end = close }
        cursor = close + 1
      end
    end
  end

  while cursor <= #text do
    local ch = text:sub(cursor, cursor)
    if ch == "[" then
      local close = find_matching_bracket(text, cursor, "[", "]", opts)
      if not close then break end
      table.insert(sections, { type = "user", text = text:sub(cursor + 1, close - 1), byte_start = cursor, byte_end = close })
      cursor = close + 1
    elseif ch == "{" then
      local close = find_matching_bracket(text, cursor, "{", "}", opts)
      if not close then break end
      table.insert(sections, { type = "agent", text = text:sub(cursor + 1, close - 1), byte_start = cursor, byte_end = close })
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
  local ranges = {}
  local fence_start = nil
  for i, line in ipairs(lines) do
    if line:match("^```") then
      if fence_start then
        table.insert(ranges, { fence_start, i - 1 }); fence_start = nil
      else
        fence_start = i - 1
      end
    end
  end
  if fence_start then table.insert(ranges, { fence_start, #lines - 1 }) end
  return ranges
end

-- {start, finish} byte ranges for inline-code spans on a line (` … ` / ``` … ```).
local function inline_code_ranges(line)
  local ranges = {}
  local i = 1
  while i <= #line do
    local bt_start = i
    while i <= #line and line:sub(i, i) == "`" do i = i + 1 end
    local bt_len = i - bt_start
    if bt_len > 0 then
      local delimiter = string.rep("`", bt_len)
      local close = line:find(delimiter, i, true)
      if close then
        table.insert(ranges, { bt_start, close + bt_len - 1 })
        i = close + bt_len
      end
    else
      i = i + 1
    end
  end
  return ranges
end

-- Parse 🤖 markers over the whole buffer (sections may span lines, bounded).
M.parse_markers = function(lines)
  local fence_ranges = compute_fence_ranges(lines)
  local doc = table.concat(lines, "\n")

  local line_starts = {}
  do
    local off = 1
    for i, line in ipairs(lines) do
      line_starts[i] = off
      off = off + #line + 1
    end
  end

  local function offset_to_pos(offset)
    local lo, hi = 1, #line_starts
    while lo < hi do
      local mid = math.floor((lo + hi) / 2) + 1
      if line_starts[mid] <= offset then lo = mid else hi = mid - 1 end
    end
    return lo - 1, offset - line_starts[lo]
  end

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
  local markers = {}
  local search_start = 1
  while true do
    local pos = doc:find(MARKER_CHAR, search_start, true)
    if not pos then break end
    if is_excluded(pos) then
      search_start = pos + MARKER_BYTE_LEN
      goto continue
    end

    local sections, end_pos, quoted, strike = parse_marker_sections(doc, pos, MARKER_BYTE_LEN, opts)
    if strike and strike.text == "" then strike = nil end
    if #sections > 0 or quoted or strike then
      local last = sections[#sections]
      local line0, col0 = offset_to_pos(pos)
      local ready = (not strike) and last and last.type == "user" and last.text ~= "" or false
      local pending = last and last.type == "agent" and last.text ~= "" or false
      table.insert(markers, {
        line = line0, col = col0,
        quoted = quoted, strike = strike, sections = sections,
        ready = ready, pending = pending,
        raw = doc:sub(pos, end_pos - 1),
      })
    end
    search_start = end_pos
    ::continue::
  end

  return markers
end

-- Exposed for the highlighter (the per-line section seam).
M._parse_marker_sections = parse_marker_sections

-- Per-line highlight spans for 🤖 markers (issue #66 M3). Pure: lines → spans
-- { row (0-based), col_start (0-based byte), col_end (0-based byte, exclusive),
-- hl_group }. Ported from parley's highlighter (per-line 🤖 scan). Groups:
-- ParleyReviewQuoted (🤖<…>), ParleyReviewStrike (🤖~…~), ParleyReviewUser ([…]),
-- ParleyReviewAgent ({…}). Scans per LINE (not the whole doc) — markers used as
-- review requests sit on one line; the multi-line parser is parse_markers.
function M.highlight_spans(lines)
  local spans = {}
  for i, line in ipairs(lines) do
    local row = i - 1
    local search_start = 1
    while true do
      local pos = line:find(MARKER_CHAR, search_start, true)
      if not pos then break end
      local sections, end_pos, quoted, strike = parse_marker_sections(line, pos, MARKER_BYTE_LEN)
      if quoted then
        spans[#spans + 1] = { row = row, col_start = pos - 1, col_end = quoted.byte_end, hl_group = 'ParleyReviewQuoted' }
      elseif strike and strike.text ~= '' then
        spans[#spans + 1] = { row = row, col_start = pos - 1, col_end = strike.byte_end, hl_group = 'ParleyReviewStrike' }
      end
      for _, section in ipairs(sections) do
        local hl = section.type == 'agent' and 'ParleyReviewAgent' or 'ParleyReviewUser'
        spans[#spans + 1] = { row = row, col_start = section.byte_start - 1, col_end = section.byte_end, hl_group = hl }
      end
      search_start = (end_pos > pos) and end_pos or (pos + MARKER_BYTE_LEN)
    end
  end
  return spans
end

return M
