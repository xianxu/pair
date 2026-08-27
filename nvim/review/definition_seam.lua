-- nvim/review/definition_seam.lua -- tag-scoped definition request/result files.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local seam = dofile(here .. 'seam.lua')

function M.request_path(path)
  if not path or path == '' then return nil end
  return path
end

function M.result_path(path)
  if not path or path == '' then return nil end
  return path
end

local function write_json(path, doc)
  if not path then return false end
  local tmp = path .. '.tmp'
  if vim.fn.writefile({ vim.json.encode(doc) }, tmp) ~= 0 then return false end
  return os.rename(tmp, path) == true
end

local function read_json(path)
  if not path or vim.fn.filereadable(path) ~= 1 then return nil end
  local ok, decoded = pcall(vim.json.decode, table.concat(vim.fn.readfile(path), '\n'))
  if not ok or type(decoded) ~= 'table' then return nil end
  return decoded
end

function M.write_request(path, request)
  return write_json(M.request_path(path), request)
end

function M.read_result(path)
  return read_json(M.result_path(path))
end

function M.clear_result(path)
  path = M.result_path(path)
  if path then pcall(os.remove, path) end
end

return M
