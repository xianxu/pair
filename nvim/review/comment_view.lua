-- Pure compact projection over the shared, buffer-aware marker scan (#426).
-- Rows and columns are 0-based bytes; all span ends are exclusive. No editor
-- state is read: rendering, cursor protection and thread targeting use this
-- same projection. Legacy multiline markers stay with the existing renderer.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local markers = dofile(here .. 'markers.lua')
local reconstruct = dofile(here .. 'reconstruct.lua')
local M = {}
local TURN_HL = {user='ParleyReviewUser',agent='ParleyReviewAgent'}

function M.layout(lines)
  local scan = markers.scan(lines)
  local starts = reconstruct.line_starts(lines)
  local rows = {}
  local function push(row, item)
    rows[row] = rows[row] or {}
    rows[row][#rows[row]+1] = item
  end
  local function overlaps_warning(source)
    for _, diagnostic in ipairs(scan.diagnostics) do
      if diagnostic.row == source.line and diagnostic.kind == 'malformed'
          and source.col < diagnostic.end_col and source.end_col > diagnostic.col then
        return true
      end
    end
    return false
  end
  for _, source in ipairs(scan.markers) do
    if source.complete and source.end_row == source.line and not overlaps_warning(source) then
      local base = starts[source.line+1]
      local m = { marker=source, start=source.col, stop=source.end_col,
        hidden={}, brackets={}, turns_hl={} }
      local anchor = source.quoted or source.strike
      m.kind = anchor and (source.quoted and 'quoted' or 'strike') or 'bare'
      if anchor and anchor.text ~= '' then
        local open, close = anchor.byte_start-base, anchor.byte_end-base
        m.visible = {open+1,close,source.quoted and 'ParleyReviewQuoted' or 'ParleyReviewStrike'}
        m.hidden[#m.hidden+1] = {m.start,open+1,''}
        m.hidden[#m.hidden+1] = {close,close+1,''}
      end
      for i, section in ipairs(source.sections) do
        local open, close = section.byte_start-base, section.byte_end-base
        local human = section.type == 'user'
        m.turns_hl[#m.turns_hl+1] = {open,close+1,TURN_HL[section.type]}
        -- Override markdown shortcut-link conceal: brackets remain visible.
        m.brackets[#m.brackets+1] = {open,human and '[' or '{'}
        m.brackets[#m.brackets+1] = {close,human and ']' or '}'}
        if i == #source.sections and human then
          m.reply = {open+1,close}
        elseif close > open+1 then
          m.hidden[#m.hidden+1] = {open+1,close,'…'}
        end
      end
      push(source.line,m)
    end
  end
  for _, diagnostic in ipairs(scan.diagnostics) do
    if diagnostic.kind == 'malformed' then
      -- Scan warnings are line-local and cannot conceal any successful prefix.
      push(diagnostic.row,{start=diagnostic.col,stop=diagnostic.end_col,broken=true})
    end
  end
  for _, row in pairs(rows) do table.sort(row,function(a,b) return a.start<b.start end) end
  return rows
end

function M.marker_at(row, col)
  for _, marker in ipairs(row or {}) do
    if not marker.broken and col >= marker.start and col < marker.stop then return marker end
  end
end

local function blocked_spans(row, insert)
  local spans = {}
  for _, marker in ipairs(row or {}) do
    if not marker.broken then
      if not insert then
        for _, hidden in ipairs(marker.hidden) do spans[#spans+1] = {hidden[1],hidden[2]} end
      elseif #marker.hidden > 0 or marker.visible then
        -- Normal mode sits on bytes; insert mode sits BETWEEN them. Only the
        -- anchor, final human turn, and outside edges accept visible typing.
        local allowed = {{marker.start,marker.start}}
        if marker.visible then allowed[#allowed+1] = {marker.visible[1],marker.visible[2]} end
        if marker.reply then allowed[#allowed+1] = {marker.reply[1],marker.reply[2]} end
        allowed[#allowed+1] = {marker.stop,marker.stop}
        for i=1,#allowed-1 do
          local a,b = allowed[i][2]+1,allowed[i+1][1]
          if b>a then spans[#spans+1] = {a,b} end
        end
      end
    end
  end
  return spans
end
local function blocked_at(spans,col)
  for _, span in ipairs(spans) do if col>=span[1] and col<span[2] then return span end end
end
local function allowed_from(spans,col,dir,max_col)
  while col>=0 and col<=max_col do
    local span=blocked_at(spans,col)
    if not span then return col end
    col=dir>0 and span[2] or span[1]-1
  end
end
-- Pure UTF-8 alignment: a leftward jump can land on a continuation byte.
local function char_start(line,col)
  while col>0 do
    local byte=line:byte(col+1)
    if not byte or byte<128 or byte>=192 then break end
    col=col-1
  end
  return col
end
function M.snap(row,prev_col,col,max_col,line,insert)
  local spans=blocked_spans(row,insert)
  if not blocked_at(spans,col) then return nil end
  local dir=col>=prev_col and 1 or -1
  local to=allowed_from(spans,col,dir,max_col) or allowed_from(spans,col,-dir,max_col)
  if to and line then to=char_start(line,to) end
  return to
end
return M
