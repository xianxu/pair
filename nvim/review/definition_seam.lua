-- nvim/review/definition_seam.lua -- tag-scoped definition request/result files.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local seam = dofile(here .. 'seam.lua')

function M.request_path(data_dir, env_tag)
  if not data_dir or data_dir == '' then return nil end
  return data_dir .. '/review-definition-request-' .. seam.tag(env_tag) .. '.json'
end

function M.result_path(data_dir, env_tag)
  if not data_dir or data_dir == '' then return nil end
  return data_dir .. '/review-definition-result-' .. seam.tag(env_tag) .. '.json'
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

function M.write_request(data_dir, env_tag, request)
  return write_json(M.request_path(data_dir, env_tag), request)
end

function M.read_result(data_dir, env_tag)
  return read_json(M.result_path(data_dir, env_tag))
end

function M.clear_result(data_dir, env_tag)
  local path = M.result_path(data_dir, env_tag)
  if path then pcall(os.remove, path) end
end

return M
