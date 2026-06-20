-- nvim/review/docflow.lua — thin wrapper shelling ariadne's docflow (issue #66
-- M1). Reuses the proven script; no commit/branch logic reimplemented. The
-- binary is `$DOCFLOW_BIN` (default `docflow`), so tests point it at a fake.
local M = {}

local function bin()
  return vim.env.DOCFLOW_BIN or 'docflow'
end

local function run(args)
  local cmd = { bin() }
  vim.list_extend(cmd, args)
  -- A missing binary makes vim.system raise ENOENT (not a non-zero exit), which
  -- would crash the caller (e.g. review.start on VimEnter). Catch it and return
  -- a synthetic failure result so callers' check()/notify path handles it and
  -- the review pane still opens (render-only until docflow is on PATH/$DOCFLOW_BIN).
  local ok, res = pcall(function() return vim.system(cmd, { text = true }):wait() end)
  if not ok then
    -- `unavailable` marks "docflow isn't on PATH / $DOCFLOW_BIN" — an EXPECTED
    -- state in a live M3 pane (render-only; round commits are agent-side, M4), as
    -- distinct from a real non-zero docflow failure. Callers degrade quietly on it.
    return { code = 127, unavailable = true, stdout = '',
      stderr = 'docflow not runnable (' .. bin() .. '): ' .. tostring(res) }
  end
  return res
end

function M.start(file)
  return run({ 'start', file })
end

-- The agent round carries the records inside `body` (record.embed_in_body).
function M.round(side, summary, body)
  local args = { 'round', '--side', side }
  if summary and summary ~= '' then vim.list_extend(args, { '-m', summary }) end
  if body and body ~= '' then vim.list_extend(args, { '--body', body }) end
  return run(args)
end

function M.status()
  return run({ 'status' })
end

function M.ship()
  return run({ 'ship' })
end

return M
