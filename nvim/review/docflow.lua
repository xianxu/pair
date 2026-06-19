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
  return vim.system(cmd, { text = true }):wait()
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
