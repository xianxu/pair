-- Isolated embedded Neovim UI conformance: observe rendered cells, not merely
-- the statusline model. Uses Neovim's bundled MessagePack implementation.
local uv = vim.uv or vim.loop
local root = vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':p:h:h')
local tmp = vim.fn.tempname()
vim.fn.mkdir(tmp .. '/queue', 'p')
vim.fn.writefile({'draft'}, tmp .. '/draft.md')
vim.fn.writefile({}, tmp .. '/log.md')
local env = vim.fn.environ()
for key,value in pairs({PAIR_HOME=root,PAIR_DATA_DIR=tmp,PAIR_TAG='ui-test',
  PAIR_DRAFT_PATH=tmp..'/draft.md',PAIR_LOG_PATH=tmp..'/log.md',
  PAIR_QUEUE_DIR=tmp..'/queue',PAIR_LAYOUT_MODE_PATH=tmp..'/layout'}) do env[key]=value end
local child_env={}
for key,value in pairs(env) do child_env[#child_env+1]=key..'='..value end
local stdin,stdout,stderr=uv.new_pipe(false),uv.new_pipe(false),uv.new_pipe(false)
local responses,frames,markers,grid={},{},{},{}
local stderr_text,parse_error='',nil
local unpacker=vim.mpack.Unpacker()
local function screen()
  local rows={}
  for i=1,18 do rows[#rows+1]=table.concat(grid[i] or {}) end
  return table.concat(rows,'\n')
end
local function receive(message)
  if message[1]==1 then responses[message[2]]=message
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
child=assert(uv.spawn(vim.v.progpath,{args={'--embed','--headless','-u',root..'/nvim/init.lua',tmp..'/draft.md'},
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
  assert(vim.wait(5000,function()return responses[id] or parse_error or exited end,5),'RPC timeout: '..method..' '..stderr_text)
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
  lua([[
    vim.notify=function()end
    _G.PairTestSessionLogAppend=function()return true end
    _G.PairTestSessionLogCommit=function()return true end
    _G.PairTestDOMPin=function()
      vim.rpcnotify(1,'test_phase','pin')
      vim.wait(150,function()return false end,10)
      return {pane_id=7,validate=function()return true end}
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
end,debug.traceback)
pcall(function()request('nvim_command',{'qa!'},true)end)
vim.wait(1000,function()return exited end,10)
if not exited then child:kill('sigkill');vim.wait(1000,function()return exited end,10) end
for _,handle in ipairs({stdin,stdout,stderr,child}) do if not handle:is_closing() then handle:close() end end
vim.fn.delete(tmp,'rf')
if not ok then error(err) end
print('dom-submission-ui-test: passed')
