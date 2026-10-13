-- Private Couch RPC + an exact Zellij pane, without changing operator focus.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local pane_identity = dofile(here .. 'pane_identity.lua')
function M.new(v, action)
  local bin = (v.env.PAIR_HOME or '') ~= '' and (v.env.PAIR_HOME .. '/bin/couch') or 'couch'
  local function argv(op,id) return {bin,'--broadcast-web-submit',op,id} end
  local function decode(result)
    local ok,r=pcall(v.json.decode,result.stdout or '')
    local status=ok and type(r)=='table' and r.Submission or nil
    if result.code ~= 0 then
      -- Preserve a definite failed preparation so its retry can start afresh.
      if status and status.state=='failed' then return status end
      local reason=(result.stderr or ''):gsub('%s+$','')
      if reason=='' then reason=result.code==124 and 'Couch request timed out' or ('Couch request exited '..tostring(result.code)) end
      return nil,reason
    end
    if not status then return nil,'invalid Couch response' end
    return status
  end
  local function timeout(operation)
    if not operation then return 4000 end
    operation.check()
    local remaining = operation.remaining()
    assert(remaining > 0, 'submission timed out')
    return math.max(1, math.floor(math.min(4000, remaining)))
  end
  local function rpc(op,id,cb,operation)
    v.system(argv(op,id),{text=true,timeout=timeout(operation)},function(result)
      v.schedule(function()cb(decode(result))end)
    end)
  end
  local function sync(op,id,operation)
    return decode(v.system(argv(op,id),{text=true,timeout=timeout(operation)}):wait())
  end
  local function panes(operation)
    local result=action('draft.dom.list-panes',{'zellij','action','list-panes','--json','--command'}, operation and {timeout_ms=timeout(operation)} or nil) or {}
    if result.code~=0 then return nil end
    local ok,node=pcall(v.json.decode,result.stdout or '');if not ok then return nil end
    return pane_identity.agents(node)
  end
  local function pin(id,operation)
    local p=panes(operation)
    if not p or #p~=1 then return nil,'cannot uniquely identify the agent pane' end
    local pane_id,command=p[1].id,p[1].terminal_command
    return {pane_id=pane_id,validate=function()
      local r,err=sync('begin',id,operation)
      if not r or r.state~='delivering' then return false,err or 'recipient is no longer current' end
      local now=panes(operation)
      if not now or #now~=1 or now[1].id~=pane_id or now[1].terminal_command~=command then return false,'agent pane was replaced' end
      return true
    end}
  end
  return {rpc=rpc,pin=pin}
end
return M
