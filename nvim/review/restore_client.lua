-- Draft-side activation transaction. Git supplies identity before pane visibility
-- or cached targets are consulted; only a validated acknowledgment is published.
local here=debug.getinfo(1,'S').source:match('@?(.*/)') or './'
local identity=dofile(here..'identity.lua')
local M, C = {}, {}; C.__index=C
local function same(a,b,head)
  if type(a)~='table' or type(b)~='table' then return false end
  for _, key in ipairs(head and {'repo','branch','file','head'} or {'repo','branch','file'}) do
    if type(a[key])~='string' or a[key]=='' or a[key]~=b[key] then return false end
  end
  return true
end
local function valid_context(value,wanted)
  return same(value,wanted) and type(value.activation)=='string' and value.activation~=''
end
function C:finish(message)
  self.busy=false
  if message then (self.opts.notify or vim.notify)('PairReview: '..message,vim.log.levels.WARN) end
end
function C:run(cmd,opts,callback)
  opts=vim.tbl_extend('force',{text=true,timeout=5000},opts or {})
  local ok,err=pcall(self.opts.run or vim.system,cmd,opts,function(result)
    vim.schedule(function()
      if not self.busy then return end
      local success,failure=pcall(callback,result)
      if not success then self:finish(tostring(failure)) end
    end)
  end)
  if not ok then self:finish(tostring(err)) end
end
function C:resolve(callback)
  local dir=(self.opts.cwd or vim.fn.getcwd)()
  self:run(identity.command(dir),{timeout=2500},function(result)
    local observed=identity.decode(result)
    local target=self.opts.read_target()
    local receipt=target and target.identity
    -- Selection is an exception only for history-free, unchanged preparation in
    -- this conversation. The CLI verifies tracked regular file and pinned HEAD.
    if observed.status=='missing' and type(receipt)=='table' and receipt.repo==observed.repo
      and receipt.branch==observed.branch and receipt.head==observed.head and type(receipt.file)=='string' then
      self:run(identity.command(dir,receipt),{timeout=2500},function(selected)
        callback(identity.decode(selected))
      end)
    else callback(observed) end
  end)
end
function C:pane()
  local path=self.opts.state_file()
  if not path or vim.fn.filereadable(path)~=1 then return nil end
  local ok,lines=pcall(vim.fn.readfile,path)
  if not ok then return nil end
  local pid=tonumber(lines[1])
  if not pid or pid<1 or not (vim.uv or vim.loop).kill(pid,0) then return nil end
  local decoded,meta=pcall(vim.json.decode,lines[3] or '')
  if not decoded or type(meta)~='table' then return {legacy=true} end
  if meta.version~=1 or type(meta.endpoint)~='string' or meta.endpoint==''
    or type(meta.token)~='string' or meta.token=='' then return {legacy=true} end
  return meta
end
function C:show()
  self:run({'zellij','action','show-floating-panes'},nil,function(result)
    self:finish(result.code~=0 and 'could not show review pane' or nil)
  end)
end
function C:activate(pane,wanted,opened)
  if pane.legacy then self:finish('existing review pane cannot restore branches; finish and close it before reopening'); return end
  local session=self.opts.session() or ''
  if pane.session~=session then self:finish('existing review pane belongs to another conversation; finish and close it first'); return end
  local request={token=pane.token,session=session,identity=wanted}
  -- Only this constant expression is evaluated; request JSON remains a quoted
  -- Vim string argument, including document names containing quotes/newlines.
  local expression='luaeval('..vim.fn.string('vim.json.encode(PairReviewPane.restore(_A))')..','
    ..vim.fn.string(vim.json.encode(request))..')'
  self:run({vim.v.progpath,'--server',pane.endpoint,'--remote-expr',expression},nil,function(result)
    if result.code~=0 then self:finish('review activation outcome unknown; retry Alt+C to query the same pane (no replacement started)'); return end
    local ok,ack=pcall(vim.json.decode,result.stdout or '')
    if not ok or type(ack)~='table' or ack.ok~=true then
      self:finish(type(ack)=='table' and ack.error or 'invalid review activation acknowledgment'); return
    end
    if not same(ack.identity,wanted,true) or not valid_context(ack.context,wanted) then
      self:finish('review activation acknowledgment does not match the selected branch/document'); return
    end
    self:resolve(function(observed)
      if observed.status~='resolved' or not same(observed,wanted,true) then
        self:finish('branch changed during activation; invoke Alt+C again'); return
      end
      local current=self:pane()
      if (self.opts.session() or '')~=session or not current or current.token~=pane.token or current.endpoint~=pane.endpoint or current.session~=session
        or not valid_context(current.context,wanted) or current.context.activation~=ack.context.activation then
        self:finish('review pane changed during activation; invoke Alt+C again'); return
      end
      self.opts.write_target(wanted.repo..'/'..wanted.file,'ready',observed)
      if opened or ack.same~=true then self:show(); return end
      self:run({'zellij','action','are-floating-panes-visible'},nil,function(visibility)
        if visibility.code~=0 then self:finish('could not query review visibility'); return end
        if (visibility.stdout or ''):match('true') then
          self.opts.hide(); self:finish()
        else self:show() end
      end)
    end)
  end)
end
function C:open(wanted)
  local pair=self.opts.pair_bin()
  self:run({pair,'review','open',wanted.repo..'/'..wanted.file},
    {env={PAIR_REVIEW_IDENTITY=vim.json.encode(wanted),PAIR_SESSION_ID=self.opts.session() or ''}},function(result)
      if result.code~=0 then self:finish('review open failed; '..(result.stderr or '')); return end
      local deadline=(vim.uv or vim.loop).hrtime()+5e9
      local function poll()
        if not self.busy then return end
        local pane=self:pane()
        if pane and not pane.legacy then self:activate(pane,wanted,true); return end
        if (vim.uv or vim.loop).hrtime()>=deadline then
          self:finish('review pane did not acknowledge opening; retry Alt+C without replacing the pane'); return
        end
        vim.defer_fn(poll,25)
      end
      poll()
    end)
end
function C:toggle()
  if self.busy then return end
  self.busy=true
  self:resolve(function(observed)
    if observed.status=='non_review' or observed.status=='missing' then
      local target=self.opts.read_target()
      if target and target.status=='proposed' then self:finish('review prep in progress — check the agent pane')
      else self.opts.prompt(); self:finish() end
      return
    end
    if observed.status~='resolved' or not same(observed,observed,true) then
      self:finish(observed.diagnostic or 'review identity could not be resolved'); return
    end
    local pane=self:pane()
    if pane then self:activate(pane,observed,false) else self:open(observed) end
  end)
end
function M.new(opts) return setmetatable({opts=opts,busy=false},C) end
return M
