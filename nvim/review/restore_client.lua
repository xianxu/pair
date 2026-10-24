-- Draft-side activation transaction. Git supplies identity before pane visibility
-- or cached targets are consulted; only a validated acknowledgment is published.
local here=debug.getinfo(1,'S').source:match('@?(.*/)') or './'
local identity=dofile(here..'identity.lua')
local observation=dofile(here..'../zellij_observation.lua')
local zellij_trace=dofile(here..'../zellij_trace.lua')
local panes_identity=dofile(here..'../pane_identity.lua')
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
  if cmd[1]=='zellij' then cmd=zellij_trace.command(cmd,self.opts.pair_bin()) end
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
function C:rpc(pane,request,callback)
  -- Only a constant expression is evaluated; arbitrary document names remain
  -- JSON inside a quoted Vim string argument.
  local expression='luaeval('..vim.fn.string('vim.json.encode(PairReviewPane.restore(_A))')..','
    ..vim.fn.string(vim.json.encode(request))..')'
  self:run({vim.v.progpath,'--server',pane.endpoint,'--remote-expr',expression},nil,callback)
end
function C:missing(dir,observed,receipt,callback)
  local function selected(file)
    local expected={repo=observed.repo,branch=observed.branch,head=observed.head,file=file}
    self:run(identity.command(dir,expected),{timeout=2500},function(result)
      local verified=identity.decode(result)
      if verified.status=='resolved' and same(verified,expected,true) then callback(verified)
      else callback({status='invalid',diagnostic='selected review changed; prepare it again'}) end
    end)
  end
  if same(receipt,receipt,true) and receipt.repo==observed.repo and receipt.branch==observed.branch
    and receipt.head==observed.head then selected(receipt.file); return end
  -- An empty human round advances HEAD without naming a document in history.
  -- Only an authenticated live activation can carry an earlier selection over
  -- that gap; metadata and stale target text alone are insufficient authority.
  local pane=self:pane()
  local session=self.opts.session() or ''
  if not pane or pane.legacy or pane.session~=session then callback(observed); return end
  self:rpc(pane,{token=pane.token,session=session,probe=true},function(result)
    local ok,ack=pcall(vim.json.decode,result.stdout or '')
    local ctx=ok and type(ack)=='table' and ack.context or nil
    local current=self:pane()
    if result.code~=0 or not ok or type(ack)~='table' or ack.ok~=true or not valid_context(ctx,ack.identity)
      or ctx.repo~=observed.repo or ctx.branch~=observed.branch
      or not current or current.token~=pane.token or current.session~=session or current.endpoint~=pane.endpoint
      or not valid_context(current.context,ctx) or current.context.activation~=ctx.activation
      or (self.opts.session() or '')~=session then
      callback({status='invalid',diagnostic='live review selection could not be authenticated; finish or reselect it'}); return
    end
    selected(ctx.file)
  end)
end
function C:resolve(callback)
  local dir=(self.opts.cwd or vim.fn.getcwd)()
  self:run(identity.command(dir),{timeout=2500},function(result)
    local observed=identity.decode(result)
    local target=self.opts.read_target()
    local receipt=target and target.identity
    if observed.status=='missing' then self:missing(dir,observed,receipt,callback); return end
    -- :PairReview may deliberately select a peer repository while the draft
    -- stays in Pair. A current review branch still wins. A positive non-review
    -- observation permits this current-conversation, explicit peer selection;
    -- unknown Git errors and a main checkout in the SAME repo never do.
    if observed.status=='non_review' and target and target.status=='ready'
      and same(receipt,receipt,true) and receipt.repo~=observed.repo then
      self:run(identity.command(receipt.repo),{timeout=2500},function(peer_result)
        local peer=identity.decode(peer_result)
        local function verified(value)
          if value.status=='resolved' and same(value,receipt) then callback(value)
          else callback({status='invalid',diagnostic='selected peer review changed; prepare it again'}) end
        end
        if peer.status=='missing' then self:missing(receipt.repo,peer,receipt,verified)
        else verified(peer) end
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
  if meta.version~=1 or type(meta.token)~='string' or meta.token=='' then return {legacy=true} end
  -- A tokened pane without an endpoint ran on a host that refused its socket.
  if type(meta.endpoint)~='string' or meta.endpoint=='' then return {legacy=true,unserved=true} end
  return meta
end
-- Visibility is a separate decision after successful activation. Native pane
-- facts authenticate an explicit tab, never whichever client is currently active.
local function draft_context(raw, caller)
  local panes,err=observation.panes(raw,{Command=true,Floating=true,Tab=true})
  if not panes then return nil,err end
  local selected,source=panes_identity.select_in_tab(panes,caller,panes_identity.is_draft)
  if not selected then return nil,source end
  if selected.id~=source.id then return nil,'invoking pane is not the current draft' end
  return panes_identity.context(selected)
end
local function decode_visibility(result)
  local out=(result.stdout or ''):match('^%s*(.-)%s*$')
  local err=(result.stderr or ''):match('^%s*(.-)%s*$')
  if result.signal and result.signal~=0 then return nil,'visibility query was interrupted' end
  if result.code==0 and out=='true' and err=='' then return true end
  if (result.code==1 and out=='false' and err=='') or (result.code==2 and out=='' and err=='false') then return false end
  return nil,'could not query review visibility'
end
function C:visibility(pane,wanted,session,activation,opened,native_pane,native_session)
  local function current_activation()
    local current=self:pane()
    if not self.busy or vim.env.ZELLIJ_PANE_ID~=native_pane or vim.env.ZELLIJ_SESSION_NAME~=native_session
      or (self.opts.session() or '')~=session or not current
      or current.token~=pane.token or current.endpoint~=pane.endpoint or current.session~=session
      or not valid_context(current.context,wanted) or current.context.activation~=activation then
      error('review activation changed before visibility effect',0)
    end
  end
  local budget=observation.budget({remaining=function() return 5000 end,check=current_activation})
  local caller=native_pane or ''
  local function refuse(reason) self:finish('visibility refused: '..tostring(reason)) end
  local function check()
    local ok,err=pcall(budget.check)
    if not ok then refuse(err);return false end
    return true
  end
  local function query(argv,callback)
    if not check() then return end
    local remaining=budget.remaining()
    if remaining<=0 then refuse('Zellij admission budget exhausted');return end
    self:run(argv,{timeout=math.max(1,math.floor(remaining))},function(result)
      if not check() then return end
      callback(result)
    end)
  end
  local function context(callback)
    query({'zellij','action','list-panes','--json','--command','--state','--tab'},function(result)
      if result.code~=0 or (result.signal and result.signal~=0) then refuse('could not inspect draft context');return end
      local facts,err=draft_context(result.stdout or '',caller)
      if not facts then refuse(err);return end
      if check() then callback(facts) end
    end)
  end
  local function visible(draft,callback)
    query({'zellij','action','are-floating-panes-visible','--tab-id',tostring(draft.tab_id)},function(result)
      local value,err=decode_visibility(result)
      if value==nil then refuse(err);return end
      callback(value)
    end)
  end
  local function apply(draft,hide)
    local admitted,err=observation.effect(budget,function()
      if hide then
        -- The callback performs its individual effects through the SAME token.
        if self.opts.hide(budget,draft)~=true then self:finish('could not return from review pane');return true end
        self:finish()
      else
        self:run({'zellij','action','show-floating-panes','--tab-id',tostring(draft.tab_id)},nil,function(result)
          self:finish(result.code~=0 and 'could not show review pane' or nil)
        end)
      end
      return true
    end)
    if not admitted then refuse(err) end
  end
  context(function(first)
    local function confirm_context(callback)
      context(function(second)
        if not vim.deep_equal(first,second) then refuse('draft context observations disagree');return end
        callback(second)
      end)
    end
    if opened then confirm_context(function(draft) apply(draft,false) end);return end
    visible(first,function(before)
      confirm_context(function(draft)
        visible(draft,function(after)
          if before~=after then refuse('visibility observations disagree');return end
          apply(draft,after)
        end)
      end)
    end)
  end)
end
function C:activate(pane,wanted,opened,native)
  local native_pane,native_session=native.pane,native.session
  if pane.unserved then self:finish('existing review pane has no restore socket (this host refused it), so it cannot switch branches; finish and close it to review another'); return end
  if pane.legacy then self:finish('existing review pane cannot restore branches; finish and close it before reopening'); return end
  local session=self.opts.session() or ''
  if pane.session~=session then self:finish('existing review pane belongs to another conversation; finish and close it first'); return end
  local request={token=pane.token,session=session,identity=wanted}
  self:rpc(pane,request,function(result)
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
      self:visibility(pane,wanted,session,ack.context.activation,opened or ack.same~=true,native_pane,native_session)
    end)
  end)
end
function C:open(wanted,native)
  if vim.env.ZELLIJ_PANE_ID~=native.pane or vim.env.ZELLIJ_SESSION_NAME~=native.session then
    self:finish('invoking terminal context changed before review open');return
  end
  local pair=self.opts.pair_bin()
  self:run({pair,'review','open',wanted.repo..'/'..wanted.file},
    {env={PAIR_REVIEW_IDENTITY=vim.json.encode(wanted),PAIR_SESSION_ID=self.opts.session() or ''}},function(result)
      if result.code~=0 then self:finish('review open failed; '..(result.stderr or '')); return end
      local deadline=(vim.uv or vim.loop).hrtime()+5e9
      local function poll()
        if not self.busy then return end
        local pane=self:pane()
        if pane and not pane.legacy then self:activate(pane,wanted,true,native); return end
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
  local native={pane=vim.env.ZELLIJ_PANE_ID,session=vim.env.ZELLIJ_SESSION_NAME}
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
    if pane then self:activate(pane,observed,false,native) else self:open(observed,native) end
  end)
end
function M.new(opts) return setmetatable({opts=opts,busy=false},C) end
return M
