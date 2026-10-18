-- Isolated embedded Neovim UI conformance: observe rendered cells, not merely
-- the statusline model. Uses Neovim's bundled MessagePack implementation.
local uv = vim.uv or vim.loop
local root = vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':p:h:h')
local tmp = vim.fn.tempname()
vim.fn.mkdir(tmp .. '/queue', 'p')
vim.fn.writefile({'draft'}, tmp .. '/draft.md')
vim.fn.writefile({}, tmp .. '/log.md')
vim.fn.writefile({}, tmp .. '/agent-picks')
vim.fn.writefile({}, tmp .. '/agent-output')
local env = vim.fn.environ()
-- An attached UI exercises production startup hooks. Never inherit the live
-- operator's pane identity, PID files, sidecar paths, or Couch connection.
for key in pairs(env) do
  if key:match('^PAIR_') or key:match('^COUCH_') or key:match('^ZELLIJ') or key:match('^NVIM') then env[key]=nil end
end
for key,value in pairs({PAIR_HOME=root,PAIR_DATA_DIR=tmp,PAIR_TAG='ui-test',
  PAIR_DRAFT_PATH=tmp..'/draft.md',PAIR_LOG_PATH=tmp..'/log.md',
  PAIR_QUEUE_DIR=tmp..'/queue',PAIR_LAYOUT_MODE_PATH=tmp..'/layout',
  PAIR_AGENT_PICKS_PATH=tmp..'/agent-picks',PAIR_AGENT_OUTPUT_PATH=tmp..'/agent-output',
  XDG_STATE_HOME=tmp..'/state',XDG_CACHE_HOME=tmp..'/cache'}) do env[key]=value end
local child_env={}
for key,value in pairs(env) do child_env[#child_env+1]=key..'='..value end
local stdin,stdout,stderr=uv.new_pipe(false),uv.new_pipe(false),uv.new_pipe(false)
local responses,frames,markers,grid={},{},{},{}
local stderr_text,parse_error='',nil
local ready=false
local unpacker=vim.mpack.Unpacker()
local function screen()
  local rows={}
  for i=1,18 do rows[#rows+1]=table.concat(grid[i] or {}) end
  return table.concat(rows,'\n')
end
local function receive(message)
  if message[1]==2 and message[2]=='test_ready' then ready=true
  elseif message[1]==1 then responses[message[2]]=message
  elseif message[1]==2 and message[2]=='test_phase' then
    markers[message[3][1]]=screen()
  elseif message[1]==2 and message[2]=='redraw' then
    for _,event in ipairs(message[3]) do
      if event[1]=='grid_line' then
        for i=2,#event do
          local line=event[i]
          if line[1]==1 then
            local row,col=line[2]+1,line[3]+1
            grid[row]=grid[row] or {}
            for _,cell in ipairs(line[4]) do
              for _=1,cell[3] or 1 do grid[row][col]=cell[1];col=col+1 end
            end
          end
        end
      elseif event[1]=='grid_clear' then grid={}
      elseif event[1]=='flush' then frames[#frames+1]=screen() end
    end
  end
end
local child,exited
child=assert(uv.spawn(vim.v.progpath,{args={'--embed','--headless','--cmd',"autocmd VimEnter * lua vim.defer_fn(function() vim.g.dom_ui_fixture_ready=true; vim.rpcnotify(1,'test_ready') end,100)", '-u',root..'/nvim/init.lua',tmp..'/draft.md'},
  env=child_env,stdio={stdin,stdout,stderr}},function()exited=true end))
stdout:read_start(function(err,data)
  if err then parse_error=err end
  if data then
    local ok,why=pcall(function()
      local offset=1
      while offset<=#data do
        local message
        message,offset=unpacker(data,offset)
        if message then receive(message) end
      end
    end)
    if not ok then parse_error=why end
  end
end)
stderr:read_start(function(_,data)if data then stderr_text=stderr_text..data end end)
local serial=0
local function request(method,args,async)
  serial=serial+1
  local id=serial
  stdin:write(vim.mpack.encode({0,id,method,args}))
  if async then return id end
  assert(vim.wait(5000,function()return responses[id] or parse_error or exited end,5),'RPC timeout: '..method..' '..vim.inspect(args)..'\nSCREEN:\n'..screen()..'\nSTDERR:\n'..stderr_text)
  assert(not parse_error,tostring(parse_error))
  local response=assert(responses[id],'child exited: '..stderr_text)
  assert(response[3]==vim.NIL,vim.inspect(response[3]))
  return response[4]
end
local function lua(source)return request('nvim_exec_lua',{source,{}})end
local function saw(text)
  for _,frame in ipairs(frames) do if frame:find(text,1,true) then return true end end
  return false
end
local ok,err=xpcall(function()
  request('nvim_ui_attach',{120,18,{ext_linegrid=true}})
  -- UI attachment can service RPC before deferred startup has finished. The
  -- delayed VimEnter notification deliberately tests that the client waits.
  assert(vim.wait(5000,function()return ready or exited or parse_error end,5),
    'startup-ready timeout\nSCREEN:\n'..screen()..'\nSTDERR:\n'..stderr_text)
  assert(ready,'child failed before startup-ready: '..tostring(parse_error)..' '..stderr_text)
  lua([[
    assert(vim.g.dom_ui_fixture_ready, 'client started submission before startup-ready handshake')
    -- Exercise completion's environment contract deterministically: without
    -- its picks path, TextChangedI throws and the attached UI waits at ENTER.
    vim.api.nvim_exec_autocmds('TextChangedI', {buffer=0})
    vim.notify=function()end
    _G.PairTestSessionLogAppend=function()return true end
    _G.PairTestSessionLogCommit=function()return true end
    _G.PairTestDOMPin=function()
      vim.rpcnotify(1,'test_phase','pin')
      vim.wait(150,function()return false end,10)
      return {pane_id=7,validate=function()return true end,effect=function(invoke)return invoke()end}
    end
    _G.PairTestDOMRPC=function(op,id,cb)
      if op=='prepare' or op=='status' then _G.held=cb
      elseif op=='begin' then cb({state='delivering',slot='pair:6'})
      elseif op=='ack' then cb({state='resolved'}) end
    end
    _G.PairTestSendToAgent=function()
      vim.rpcnotify(1,'test_phase','send')
      vim.wait(150,function()return false end,10)
      return true,'dispatched'
    end
    vim.api.nvim_buf_set_lines(0,0,-1,false,{'draft'})
    PairSendDOM()
  ]])
  assert(markers.pin and markers.pin:find('preparing submission',1,true),'preparing was not rendered before held pin')
  assert(lua('return vim.bo.modifiable')==false,'active draft was editable')
  -- The real focus autocmd schedules an EOL padding write one loop later.
  lua([[vim.cmd('startinsert');vim.api.nvim_win_set_cursor(0,{1,5});vim.api.nvim_exec_autocmds('FocusLost',{})]])
  assert(lua('return vim.bo.modifiable')==false,'focus loss released ownership')
  lua([[vim.api.nvim_exec_autocmds('FocusGained',{})]])
  assert(lua("return vim.api.nvim_buf_get_lines(0,0,-1,false)[1]")=='draft','focus cycle changed owned text')
  lua([[local cb=held;held=nil;cb({state='preparing',slot='pair:6',progress={phase='settling',current=1,total=3}})]])
  assert(vim.wait(1500,function()return saw('waiting for the page to settle, crop 1/3')end,5),'settle progress never rendered')
  assert(vim.wait(1000,function()return lua('return held ~= nil')end,10))
  lua([[local cb=held;held=nil;cb({state='preparing',slot='pair:6',progress={phase='capturing',current=2,total=3}})]])
  assert(vim.wait(1000,function()return saw('capturing crop 2/3')end,5),'crop progress never rendered')
  assert(vim.wait(1000,function()return lua('return held ~= nil')end,10))
  lua([[held({state='prepared',slot='pair:6',path='/tmp/ui-test/submission.json'})]])
  assert(markers.send and markers.send:find('sending to pair:6',1,true),'sending was not rendered before held pane paste')
  assert(vim.wait(1000,function()return saw('sent')end,5),'sent terminal status missing')
  assert(lua('return vim.bo.modifiable')==true,'success did not release draft')
  assert(lua("return table.concat(vim.api.nvim_buf_get_lines(0,0,-1,false),'\\n')")~='draft','success did not clear')
  lua([[vim.api.nvim_buf_set_lines(0,0,-1,false,{'retry draft'});PairSendDOM();held({state='failed',error='capture refused'})]])
  assert(vim.wait(1000,function()return saw('capture refused')end,5),'failure was not rendered')
  assert(lua('return vim.bo.modifiable')==true,'failure retained draft lock')
  assert(lua("return vim.api.nvim_buf_get_lines(0,0,-1,false)[1]")=='retry draft','failure lost authored text')
  for _,outcome in ipairs({'failure','timeout'}) do
    -- Begin with genuine focus padding, then block its removal. FocusGained
    -- must retain the pad's buffer/row even after retiring the blink timer.
    lua([[vim.api.nvim_buf_set_lines(0,0,-1,false,{'focus draft'});vim.cmd('silent write');vim.cmd('startinsert');vim.api.nvim_win_set_cursor(0,{1,11});vim.api.nvim_exec_autocmds('FocusLost',{})]])
    assert(lua("return vim.api.nvim_buf_get_lines(0,0,-1,false)[1]")=='focus draft ','fixture did not create focus pad')
    lua([[vim.bo.modifiable=false;vim.api.nvim_exec_autocmds('FocusGained',{})]])
    assert(lua("return vim.api.nvim_buf_get_lines(0,0,-1,false)[1]")=='focus draft ','blocked cleanup changed text')
    lua([[
      vim.bo.modifiable=true
      _G.PairTestDOMArm=function(fn) _G.expire_submission=fn;return function()end end
      PairSendDOM()
    ]])
    assert(lua("return vim.api.nvim_buf_get_lines(0,0,-1,false)[1]")=='focus draft','submission included stale cosmetic pad')
    assert(lua('return vim.bo.modifiable')==false,'operation did not lock after pad cleanup')
    lua([[vim.api.nvim_win_set_cursor(0,{1,11});vim.api.nvim_exec_autocmds('FocusLost',{})]])
    assert(lua('return vim.bo.modifiable')==false,'focus loss unlocked active operation')
    lua([[vim.api.nvim_exec_autocmds('FocusGained',{})]])
    if outcome=='failure' then lua([[held({state='failed',error='focus capture refused'})]])
    else lua([[expire_submission()]]) end
    assert(lua('return vim.bo.modifiable')==true,outcome..' retained focus lock')
    assert(lua("return vim.api.nvim_buf_get_lines(0,0,-1,false)[1]")=='focus draft',outcome..' corrupted focus text')
    assert(lua("return table.concat(vim.fn.readfile(vim.env.PAIR_DRAFT_PATH),'\\n')")=='focus draft',outcome..' persisted cosmetic padding')
    -- Normal focus padding/removal still works after the terminal result.
    lua([[vim.api.nvim_win_set_cursor(0,{1,11});vim.api.nvim_exec_autocmds('FocusLost',{})]])
    assert(lua("return vim.api.nvim_buf_get_lines(0,0,-1,false)[1]")=='focus draft ','released focus padding did not resume')
    lua([[vim.api.nvim_exec_autocmds('FocusGained',{})]])
    assert(lua("return vim.api.nvim_buf_get_lines(0,0,-1,false)[1]")=='focus draft','released focus cleanup did not resume')
  end
  assert(not saw('Press ENTER'),'focus lifecycle raised a blocking prompt')
end,debug.traceback)
pcall(function()request('nvim_command',{'qa!'},true)end)
vim.wait(1000,function()return exited end,10)
if not exited then child:kill('sigkill');vim.wait(1000,function()return exited end,10) end
for _,handle in ipairs({stdin,stdout,stderr,child}) do if not handle:is_closing() then handle:close() end end
vim.fn.delete(tmp,'rf')
if not ok then error(err..'\nLAST SCREEN:\n'..screen()..'\nSTDERR:\n'..stderr_text) end
print('dom-submission-ui-test: passed')
