-- adapt.lua — Lua emitter for the adaptation flight recorder.
--
-- Writes one JSON line per harness-adaptation event to
-- $PAIR_DATA_DIR/adapt-<tag>.jsonl, byte-identical in schema + field order to
-- the Go (cmd/internal/adapt) and shell (bin/lib/adapt-log.sh) emitters, so
-- doctor/doctor.sh reads every component's lines uniformly. See
-- atlas/how-to-bring-up-a-new-harness-cli.md §3.
--
-- Returns a module table; load with `dofile(<dir>/adapt.lua)`.
local M = {}

local MAX_DETAIL = 200

-- log appends one event. No-op when PAIR_TAG is unset (not in a pair session).
-- comp defaults to 'nvim'. outcome ∈ 'fired'|'bypass'|'near-miss'|'fail'.
function M.log(aspect, signal, outcome, detail, comp)
  local tag = vim.env.PAIR_TAG
  if not tag or tag == '' then return end
  comp = comp or 'nvim'

  local dir = vim.env.PAIR_DATA_DIR
  if not dir or dir == '' then
    local xdg = vim.env.XDG_DATA_HOME
    local base = (xdg and xdg ~= '') and xdg or ((vim.env.HOME or '') .. '/.local/share')
    dir = base .. '/pair'
  end

  detail = detail or ''
  if #detail > MAX_DETAIL then detail = detail:sub(1, MAX_DETAIL) end
  local ts = os.date('!%Y-%m-%dT%H:%M:%SZ') -- '!' = UTC

  -- Fixed field order matching the Go struct; vim.json.encode handles string
  -- escaping so we only assemble the envelope. detail omitted when empty
  -- (matches Go's `omitempty`).
  local parts = {
    '"ts":' .. vim.json.encode(ts),
    '"comp":' .. vim.json.encode(comp),
    '"agent":' .. vim.json.encode(vim.env.PAIR_AGENT or ''),
    '"aspect":' .. tostring(aspect),
    '"signal":' .. vim.json.encode(signal),
    '"outcome":' .. vim.json.encode(outcome),
  }
  if detail ~= '' then
    parts[#parts + 1] = '"detail":' .. vim.json.encode(detail)
  end
  local line = '{' .. table.concat(parts, ',') .. '}\n'

  local f = io.open(dir .. '/adapt-' .. tag .. '.jsonl', 'a')
  if f then
    f:write(line)
    f:close()
  end
end

return M
