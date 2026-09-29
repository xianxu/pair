-- Pane-owned activation transaction. Policy lives in restore.lua; this shell
-- revalidates observations immediately before buffer or protocol effects.
local here = debug.getinfo(1,'S').source:match('@?(.*/)') or './'
local policy = dofile(here .. 'restore.lua')
local identity = dofile(here .. 'identity.lua')
local document_bytes = dofile(here .. 'document_bytes.lua')
local artifact = dofile(here .. 'artifact.lua')
local M = {}
local C = {}; C.__index=C
local function nonce() return vim.fn.sha256(vim.fn.tempname() .. tostring(vim.uv.hrtime())) end
local function notify(s) vim.notify('PairReview: '..s,vim.log.levels.WARN) end
function M.new(opts)
  return setmetatable({opts=opts or {}, state={status='idle'}, bindings={}, token=nonce(), legacy=true},C)
end
function C:context(buf)
  if not buf then return self.state.context end
  return self.bindings[buf]
end
function C:resolve(want)
  local run=self.opts.resolve or identity.resolve
  local observed=run(want.repo)
  if observed.status=='missing' and observed.repo==want.repo and observed.branch==want.branch
    and (observed.head==want.head or policy.same(self:context(),want)) then
    -- An established pane already owns this selection. An empty first human
    -- round advances HEAD without yet giving history a document pathname.
    observed=run(want.repo,{file=want.file,head=observed.head})
  end
  return observed
end
function C:publish()
  if not self.opts.open_path then return end
  local file=self.buf and vim.api.nvim_buf_get_name(self.buf) or ''
  local meta={version=1,endpoint=self.endpoint,token=self.token,session=self.opts.session or '',context=self.state.context}
  local tmp=self.opts.open_path..'.tmp'
  vim.fn.writefile({tostring(vim.fn.getpid()),file,vim.json.encode(meta)},tmp)
  assert(vim.uv.fs_rename(tmp,self.opts.open_path))
end
function C:serve()
  if self.endpoint or not self.opts.open_path then return end
  self.socket_dir=vim.fn.tempname(); vim.fn.mkdir(self.socket_dir,'p',448)
  self.endpoint=vim.fn.serverstart(self.socket_dir..'/review.sock')
  self:publish()
end
function C:init(buf, observed, strict)
  if observed and observed.status=='resolved' then
    assert((vim.uv.fs_realpath(vim.api.nvim_buf_get_name(buf)) or vim.api.nvim_buf_get_name(buf))==observed.repo..'/'..observed.file,
      'review startup document does not match selected branch')
  end
  self.buf=buf
  if observed and observed.status == 'resolved' then
    self.state={status='switching',desired=observed}
    self.state=policy.transition(self.state,{kind='activated',context={repo=observed.repo,
      branch=observed.branch,file=observed.file,activation=nonce()}})
    self.bindings[buf]=self.state.context
  end
  self.legacy=not strict
  self:serve()
end
function C:guard(buf, quiet)
  local ctx=self:context(buf)
  if not ctx then return true end -- uninterrupted legacy/manual render-only pane
  local observed=self:resolve(self.state.identity)
  local path=vim.api.nvim_buf_get_name(buf)
  local ok=buf==self.buf and (vim.uv.fs_realpath(path) or path)==ctx.repo..'/'..ctx.file and observed.status=='resolved' and policy.same(ctx,observed)
  local reason=not ok and (observed.status=='invalid' and observed.diagnostic
    or 'branch changed; return to '..ctx.branch..' before saving or sending this review') or nil
  if reason and not quiet then notify(reason) end
  return ok,reason
end
function C:admit(payload,buf)
  local allowed,reason=self:guard(buf,true)
  if not allowed then return false,(reason or 'review authorization refused')..'; pending handoff preserved' end
  local context=type(payload)=='table' and payload.context or nil
  if not policy.validate_context(context,self:context(buf),self.legacy) then
    return false,'handoff context does not match this activation; ask the agent to reissue it with the current context'
  end
  return true
end
function C:pending()
  if self.buf and vim.bo[self.buf].modified then return 'unsaved edits; save on the original branch first' end
  local why=self.opts.pending and self.opts.pending()
  if why then return why end
  local path=vim.env.PAIR_REVIEW_HANDOFF_PATH
  if path and vim.uv.fs_stat(path) then return 'unconsumed handoff; finish the original review first' end
  if self.landed then
    local l,ctx=self.landed,self:context()
    if ctx then
      -- Check the original branch even after the operator checked out another
      -- one: an already-committed round must not become permanently pending.
      local prefix='review('..ctx.branch:sub(8)..'): agent r'
      local result=vim.system({'git','-C',ctx.repo,'log','refs/heads/'..ctx.branch,'-1',
        '--format=%H%x00%s%x00%b','--fixed-strings','--grep='..prefix,'--'}):wait(2000)
      if result.code==0 then
        local sha,subject,body=(result.stdout or ''):match('^([^%z]+)%z([^%z]+)%z(.*)$')
        if sha and subject:sub(1,#prefix)==prefix and body:find(l.body,1,true) then
          local r=vim.system({'git','-C',ctx.repo,'show',sha..':'..ctx.file}):wait(2000)
          if r.code==0 and r.stdout==l.content then self.landed=nil end
        end
      end
    end
    if self.landed then return 'applied round not yet committed; finish it on the original branch first' end
  end
end
function C:did_land(body,content) self.landed={body=body,content=content} end
function C:request(req)
  if type(req)~='table' or req.token~=self.token or req.session~=(self.opts.session or '') then
    return {ok=false,error='review pane belongs to another conversation or incarnation'}
  end
  if req.probe then return {ok=true,context=self:context(),identity=self.state.identity} end
  local wanted=req.identity
  if type(wanted)~='table' or type(wanted.repo)~='string' then return {ok=false,error='invalid review identity'} end
  local observed=self:resolve(wanted)
  if observed.status~='resolved' or not policy.same(wanted,observed) or observed.head~=wanted.head then
    return {ok=false,error=observed.status=='invalid' and observed.diagnostic or 'branch changed during restoration; invoke Alt+C again'}
  end
  local pending=self:pending()
  local previous=self.state
  local previous_legacy=self.legacy
  local next, effect=policy.transition(self.state,{kind='request',identity=observed,pending=pending})
  self.state=next
  if effect=='refuse' then return {ok=false,error=next.reason or 'review activation in progress'} end
  if effect=='toggle' then
    self.legacy=false
    self:publish()
    return {ok=true,same=true,context=self:context(),identity=observed}
  end
  local oldbuf=self.buf
  local file=observed.repo..'/'..observed.file
  local newbuf=vim.fn.bufnr(file)
  local previous_binding=self.bindings[newbuf]
  if newbuf~=-1 and vim.bo[newbuf].modified then
    self.state=policy.transition(self.state,{kind='failed',reason='destination review has unsaved edits'})
    return {ok=false,error=self.state.reason}
  end
  local ok,err=pcall(function()
    -- Read without entering the buffer (BufEnter/checktime must not mutate it).
    local decoded,decode_error=document_bytes.decode(artifact.read(file))
    assert(decoded,decode_error)
    local lines=decoded.lines
    if newbuf==-1 then newbuf=vim.fn.bufadd(file) end
    vim.fn.bufload(newbuf)
    local confirm=self:resolve(wanted)
    assert(confirm.status=='resolved' and policy.same(confirm,observed) and confirm.head==observed.head,'branch changed during activation')
    if self.opts.stop and oldbuf then self.opts.stop(oldbuf) end
    if not vim.deep_equal(lines,vim.api.nvim_buf_get_lines(newbuf,0,-1,false)) then
      vim.api.nvim_buf_call(newbuf,function()
        vim.cmd('silent! let &undolevels = &undolevels')
        vim.api.nvim_buf_set_lines(newbuf,0,-1,false,lines)
      end)
    end
    for _,option in ipairs({'endofline','fileformat','fileencoding','bomb','fixendofline'}) do vim.bo[newbuf][option]=decoded[option] end
    vim.bo[newbuf].modified=false
    self.buf=newbuf
    self.state=policy.transition(self.state,{kind='activated',context={repo=observed.repo,
      branch=observed.branch,file=observed.file,activation=nonce()}})
    self.bindings[newbuf]=self.state.context
    self.legacy=false
    vim.api.nvim_set_current_buf(newbuf)
    if self.opts.start then self.opts.start(newbuf,file,observed) end
    if self.opts.reconstruct then self.opts.reconstruct(newbuf,file,observed) end
    self:publish()
  end)
  if not ok then
    -- Activation changes editor state only. Restore the prior buffer and owner
    -- if setup fails; its watcher remains guarded against the changed branch.
    if self.buf and self.opts.stop then pcall(self.opts.stop,self.buf) end
    self.buf=oldbuf
    self.legacy=previous_legacy
    if newbuf then self.bindings[newbuf]=previous_binding end
    self.state=policy.transition(previous,{kind='failed',reason=tostring(err)})
    if oldbuf and vim.api.nvim_buf_is_valid(oldbuf) then
      pcall(vim.api.nvim_set_current_buf,oldbuf)
      if self.opts.start then pcall(self.opts.start,oldbuf,vim.api.nvim_buf_get_name(oldbuf),previous.identity) end
    end
    pcall(self.publish,self)
    return {ok=false,error=tostring(err)}
  end
  return {ok=true,same=false,context=self:context(),identity=observed}
end
function C:close()
  if self.endpoint then pcall(vim.fn.serverstop,self.endpoint) end
  if self.socket_dir then vim.fn.delete(self.socket_dir,'d') end
  local path=self.opts.open_path
  if path and vim.fn.filereadable(path)==1 then
    local lines=vim.fn.readfile(path)
    local ok,meta=pcall(vim.json.decode,lines[3] or '')
    if ok and meta.token==self.token then vim.fn.delete(path) end
  end
end
return M
