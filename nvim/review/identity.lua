-- One CLI owns Git history interpretation; callers only validate its projection.
local M = {}
local here=debug.getinfo(1,'S').source:match('@?(.*/)') or './'
function M.command(dir, selected)
  local home = vim.env.PAIR_HOME
  local bin = home and home ~= '' and (home .. '/bin/pair') or vim.fn.fnamemodify(here..'../../bin/pair',':p')
  local cmd = { bin, 'review', 'readiness', '--resolve', dir }
  if selected and selected.file and selected.head then
    vim.list_extend(cmd, {'--selected',selected.file,'--head',selected.head})
  end
  return cmd
end
function M.decode(result)
  if result.code ~= 0 then return {status='invalid',diagnostic=result.stderr or 'review resolver failed'} end
  local ok, out = pcall(vim.json.decode, result.stdout or '')
  if not ok or type(out) ~= 'table' or not out.status then
    return {status='invalid',diagnostic='invalid review resolver response'}
  end
  return out
end
function M.resolve(dir, selected)
  local ok, out = pcall(function()
    return vim.system(M.command(dir,selected), {text=true}):wait(2500)
  end)
  return ok and M.decode(out) or {status='invalid',diagnostic=tostring(out)}
end
return M
