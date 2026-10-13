-- Private Couch RPC + an exact Zellij pane, without changing operator focus.
local M = {}
function M.new(v, action)
  local bin = (v.env.PAIR_HOME or '') ~= '' and (v.env.PAIR_HOME .. '/bin/couch') or 'couch'
  local function argv(op,id) return {bin,'--broadcast-web-submit',op,id} end
  local function decode(result)
    local ok,r=pcall(v.json.decode,result.stdout or '')
    local status=ok and type(r)=='table' and r.Submission or nil
    if result.code ~= 0 then
      -- Preserve a definite failed preparation so its retry can start afresh.
      if status and status.state=='failed' then return status end
      return nil,(result.stderr or ''):gsub('%s+$','')
    end
    if not status then return nil,'invalid Couch response' end
    return status
  end
  local function rpc(op,id,cb)
    v.system(argv(op,id),{text=true,timeout=4000},function(result)
      v.schedule(function()cb(decode(result))end)
    end)
  end
  local function sync(op,id)
    return decode(v.system(argv(op,id),{text=true,timeout=4000}):wait())
  end
  local function panes()
    local result=action('draft.dom.list-panes',{'zellij','action','list-panes','--json','--command'}) or {}
    if result.code~=0 then return nil end
    local ok,node=pcall(v.json.decode,result.stdout or '');if not ok then return nil end
    local found={}
    local function collect(n)
      if type(n)~='table' then return end
      if n.id~=nil and n.is_floating~=nil then
        if n.is_plugin==false and n.is_floating==false and type(n.terminal_command)=='string' and n.terminal_command:find('pair wrap',1,true) then found[#found+1]=n end
      else for _,child in pairs(n) do collect(child) end end
    end
    collect(node);return found
  end
  local function pin(id)
    local p=panes()
    if not p or #p~=1 then return nil,'cannot uniquely identify the agent pane' end
    local pane_id,command=p[1].id,p[1].terminal_command
    return {pane_id=pane_id,validate=function()
      local r,err=sync('begin',id)
      if not r or r.state~='delivering' then return false,err or 'recipient is no longer current' end
      local now=panes()
      if not now or #now~=1 or now[1].id~=pane_id or now[1].terminal_command~=command then return false,'agent pane was replaced' end
      return true
    end}
  end
  return {rpc=rpc,pin=pin}
end
return M
