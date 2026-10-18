-- Strict native observations and a single admission token for synchronous decisions.
local M = {}
local MAX_EXACT = '9007199254740991'
local function fail(reason) error(reason, 0) end
local function decimal(s)
  if type(s) ~= 'string' or not s:match('^%d+$') then fail('invalid integer identity') end
  local canonical = s:gsub('^0+', '')
  if canonical == '' then canonical = '0' end
  if #canonical > #MAX_EXACT or (#canonical == #MAX_EXACT and canonical > MAX_EXACT) then
    fail('integer is not exactly representable')
  end
  return tonumber(canonical)
end

-- This structural pass retains JSON kinds and number spellings that Lua's
-- decoder would otherwise erase. Every object, including unknown extensions,
-- is checked for duplicate decoded keys before decoding the document.
local function scan(raw)
  if type(raw) ~= 'string' then fail('expected JSON text') end
  local i, n = 1, #raw
  local function ws()
    while i <= n and raw:sub(i,i):match('[ \t\r\n]') do i=i+1 end
  end
  local function str()
    local start=i
    if raw:sub(i,i) ~= '"' then fail('expected JSON string') end
    i=i+1
    while i<=n do
      local ch=raw:sub(i,i)
      if ch=='"' then
        i=i+1
        return raw:sub(start,i-1)
      end
      if ch=='\\' then i=i+1 end
      i=i+1
    end
    fail('unterminated JSON string')
  end
  local value
  value=function()
    ws()
    local ch=raw:sub(i,i)
    local node={}
    if ch=='{' then
      node.kind='object';node.children={};local seen={};i=i+1;ws()
      if raw:sub(i,i)=='}' then i=i+1;return node end
      while true do
        ws();local key=vim.json.decode(str())
        if seen[key] then fail('duplicate JSON key') end
        seen[key]=true;ws()
        if raw:sub(i,i)~=':' then fail('expected colon') end
        i=i+1;node.children[key]=value();ws()
        ch=raw:sub(i,i);i=i+1
        if ch=='}' then return node end
        if ch~=',' then fail('expected object delimiter') end
      end
    elseif ch=='[' then
      node.kind='array';node.children={};i=i+1;ws()
      if raw:sub(i,i)==']' then i=i+1;return node end
      while true do
        node.children[#node.children+1]=value();ws();ch=raw:sub(i,i);i=i+1
        if ch==']' then return node end
        if ch~=',' then fail('expected array delimiter') end
      end
    elseif ch=='"' then str();node.kind='string'
    else
      local start=i
      while i<=n and not raw:sub(i,i):match('[%s,%]%}]') do i=i+1 end
      if start==i then fail('expected JSON value') end
      node.token=raw:sub(start,i-1)
      node.kind=node.token=='null' and 'null' or (node.token=='true' or node.token=='false') and 'boolean' or 'number'
    end
    return node
  end
  local shape=value();ws()
  if i<=n then fail('trailing JSON data') end
  return shape
end
local function decode(raw)
  local shape=scan(raw)
  return vim.json.decode(raw),shape
end
local function safe(fn,...)
  local ok,value=pcall(fn,...)
  if ok then return value end
  return nil,tostring(value)
end
local function field(row,shape,key,kind,required,nullable)
  local node=shape.children[key]
  if not node then
    if required then fail('missing '..key) end
    return nil
  end
  if nullable and node.kind=='null' then return '' end
  if node.kind~=kind then fail('invalid '..key) end
  if kind=='number' then return decimal(node.token) end
  return row[key]
end
local function qualified(s)
  if type(s)~='string' then fail('invalid qualified pane identity') end
  local role,id=s:match('^(%a+)_(%d+)$')
  if role~='terminal' and role~='plugin' then fail('invalid qualified pane identity') end
  return decimal(id),role=='plugin'
end
local function identity(v,node,plugin)
  if node.kind=='number' then return decimal(node.token) end
  if node.kind~='string' then fail('invalid pane identity') end
  if v:find('_',1,true) then
    local id,role=qualified(v)
    if role~=plugin then fail('contradictory pane role') end
    return id
  end
  return decimal(v)
end
function M.panes(raw,req)
  return safe(function()
    req=req or {}
    local root,shape=decode(raw)
    local rows,nodes={},{}
    local function append(group,node)
      if node.kind~='array' then fail('expected pane array') end
      for i,p in ipairs(group) do rows[#rows+1]=p;nodes[#nodes+1]=node.children[i] end
    end
    if shape.kind=='array' then append(root,shape)
    elseif shape.kind=='object' then
      local keys={}
      for k in pairs(root) do decimal(k);keys[#keys+1]=k end
      if #keys==0 then fail('missing pane array') end
      table.sort(keys)
      for _,k in ipairs(keys) do append(root[k],shape.children[k]) end
    else fail('expected pane array') end
    local out,seen={},{}
    for i,row in ipairs(rows) do
      local node=nodes[i]
      if node.kind~='object' then fail('expected pane object') end
      local p={is_plugin=field(row,node,'is_plugin','boolean',true)}
      for _,key in ipairs({'id','pane_id'}) do
        if node.children[key] then
          local id=identity(row[key],node.children[key],p.is_plugin)
          if p.id~=nil and p.id~=id then fail('contradictory pane identities') end
          p.id=id
        end
      end
      if p.id==nil then fail('missing pane identity') end
      local key=tostring(p.is_plugin)..':'..string.format('%.0f',p.id)
      if seen[key] then fail('duplicate pane identity') end
      seen[key]=true
      p.terminal_command=field(row,node,'terminal_command','string',req.Command,true) or ''
      p.pane_command=field(row,node,'pane_command','string',req.ActualCommand)
      if req.ActualCommand and not p.pane_command:match('%S') then fail('missing actual pane command') end
      p.pane_cwd=field(row,node,'pane_cwd','string')
      p.title=field(row,node,'title','string')
      for _,flag in ipairs({'focused','floating','fullscreen'}) do
        p['is_'..flag]=field(row,node,'is_'..flag,'boolean',req.State or req[({focused='Focus',floating='Floating',fullscreen='Fullscreen'})[flag]])
      end
      for _,key in ipairs({'pane_x','pane_y','pane_columns','pane_rows','tab_id','tab_position'}) do
        local geometry=key:sub(1,5)=='pane_'
        p[key]=field(row,node,key,'number',geometry and req.Geometry or (not geometry and req.Tab))
        if (key=='pane_columns' or key=='pane_rows') and p[key]==0 then fail('invalid pane size') end
      end
      out[#out+1]=p
    end
    return out
  end)
end
function M.clients(raw)
  return safe(function()
    if type(raw)~='string' then fail('expected clients text') end
    local lines={}
    for line in (raw:gsub('^%s+',''):gsub('%s+$','')..'\n'):gmatch('(.-)\n') do lines[#lines+1]=line end
    if not lines[1] or lines[1]:gsub('%s+',' ')~='CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND' then fail('invalid clients header') end
    local out,seen={},{}
    for i=2,#lines do
      if lines[i]:match('%S') then
        local id,pane,command=lines[i]:match('^%s*(%S+)%s+(%S+)%s+(.+)%s*$')
        if not id or not command:match('%S') then fail('invalid client row') end
        id=decimal(id);qualified(pane)
        if seen[id] then fail('duplicate client identity') end
        seen[id]=true
        out[#out+1]={client_id=id,zellij_pane_id=pane,running_command=command:gsub('%s+$','')}
      end
    end
    return out
  end)
end
function M.tab(raw)
  return safe(function()
    local row,node=decode(raw)
    if node.kind~='object' then fail('expected tab object') end
    local layout=field(row,node,'active_swap_layout_name','string',true,true)
    if node.children.active_swap_layout_name.kind=='null' then layout=vim.NIL end
    local t={active_swap_layout_name=layout,is_swap_layout_dirty=field(row,node,'is_swap_layout_dirty','boolean',true),
      tab_id=field(row,node,'tab_id','number'),position=field(row,node,'position','number')}
    local legacy=field(row,node,'tab_position','number')
    if legacy~=nil and t.position~=nil and legacy~=t.position then fail('contradictory tab position') end
    if t.position==nil then t.position=legacy end
    return t
  end)
end
-- Capture independent local identity once; decorate existing operation tokens
-- without creating a deadline or mutating their preparation/receipt lifetime.
function M.capture_environment(environment)
  local pane,session=environment.ZELLIJ_PANE_ID,environment.ZELLIJ_SESSION_NAME
  return function(token)
    return {remaining=token.remaining,check=function()
      token.check()
      assert(environment.ZELLIJ_PANE_ID==pane and environment.ZELLIJ_SESSION_NAME==session,
        'invoking terminal context changed')
      -- Reading a local fence may itself observe cancellation of its parent.
      token.check()
    end}
  end
end
function M.budget(parent)
  local deadline=vim.loop.hrtime()/1000000+5000
  local b={}
  function b.remaining()
    local remaining=deadline-vim.loop.hrtime()/1000000
    if parent then remaining=math.min(remaining,parent.remaining()) end
    return math.max(0,remaining)
  end
  function b.check()
    if parent then parent.check() end
    if b.remaining()<=0 then fail('Zellij admission budget exhausted') end
  end
  return b
end
local function checked(b)
  local ok,reason=pcall(b.check)
  if not ok then return nil,tostring(reason) end
  return true
end
function M.query(b,label,argv,action)
  local ok,reason=checked(b);if not ok then return nil,reason end
  local remaining=b.remaining()
  if remaining<=0 then return nil,'Zellij admission budget exhausted' end
  local invoked,result,err=pcall(action,label,argv,{timeout_ms=remaining})
  ok,reason=checked(b);if not ok then return nil,reason end
  if not invoked then return nil,tostring(result) end
  return result,err
end
function M.confirm(b,read,project)
  local prior
  for i=1,2 do
    local ok,reason=checked(b);if not ok then return nil,reason end
    local raw,err=read();ok,reason=checked(b);if not ok then return nil,reason end
    if raw==nil then return nil,err or 'unknown Zellij observation' end
    local facts;facts,err=project(raw);ok,reason=checked(b);if not ok then return nil,reason end
    if type(facts)~='table' then return nil,err or 'invalid Zellij observation' end
    if i==2 then
      if not vim.deep_equal(prior,facts) then return nil,'Zellij observations disagree' end
      return facts
    end
    prior=facts
  end
end
function M.effect(b,invoke)
  local ok,reason=checked(b);if not ok then return nil,reason end
  return invoke()
end
return M
