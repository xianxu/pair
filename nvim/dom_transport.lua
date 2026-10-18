-- Private Couch RPC + an exact Zellij pane, without changing operator focus.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local pane_identity = dofile(here .. 'pane_identity.lua')
local observation = dofile(here .. 'zellij_observation.lua')
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
    -- Pin is candidate evidence under the existing transaction lifetime. Only
    -- the first pre-write validation creates the native admission window.
    local query_budget=operation or {check=function()end,remaining=function()return 4000 end}
    local result,err=observation.query(query_budget,'draft.dom.list-panes',
      {'zellij','action','list-panes','--json','--command','--state','--tab'},action)
    if not result then return nil,err end
    if result.code~=0 then return nil,'could not inspect agent pane' end
    local rows,reason=observation.panes(result.stdout or '',{Command=true,Floating=true,Tab=true})
    if not rows then return nil,reason end
    local agent,source=pane_identity.select_in_tab(rows,v.env.ZELLIJ_PANE_ID,pane_identity.is_agent)
    if not agent then return nil,source end
    operation.check()
    return {source={id=source.id,tab_id=source.tab_id,tab_position=source.tab_position,
      terminal_command=source.terminal_command},
      agent={id=agent.id,tab_id=agent.tab_id,tab_position=agent.tab_position,
        terminal_command=agent.terminal_command}}
  end
  local function pin(id,operation)
    local guard=observation.capture_environment(v.env)
    local candidate_context=guard(operation or {check=function()end,remaining=function()return 4000 end})
    candidate_context.check()
    local candidate,err=panes(candidate_context)
    if not candidate then return nil,err end
    local admission,refused
    local function validate()
      candidate_context.check()
      if not admission then admission=guard(observation.budget(operation)) end
      admission.check()
      local r,reason=sync('begin',id,admission)
      admission.check()
      if not r or r.state~='delivering' then return false,reason or 'recipient is no longer current' end
      local current,why=panes(admission)
      if not current then return false,why end
      if not vim.deep_equal(candidate,current) then return false,'agent pane was replaced' end
      return true
    end
    return {pane_id=candidate.agent.id,validate=function()
      if refused then return false,refused end
      local called,valid,reason=pcall(validate)
      if not called then
        refused=tostring(valid)
        error(refused,0)
      end
      if not valid then refused=reason or 'unknown agent context' end
      return valid,reason
    end,effect=function(invoke)
      if refused then return nil,refused end
      if not admission then return nil,'target has not been admitted' end
      local result,reason=observation.effect(admission,invoke)
      if result==nil then refused=reason or 'unknown target effect' end
      return result,reason
    end}
  end
  return {rpc=rpc,pin=pin}
end
return M
