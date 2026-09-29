-- Proactive recovery is observation, never mutation permission. One owned timer
-- and one bounded asynchronous resolver coalesce edits; activation/quit cancel
-- work, and late callbacks must still belong to the exact captured binding.
local here=debug.getinfo(1,'S').source:match('@?(.*/)') or './'
local identity=dofile(here..'identity.lua')
local M,C={},{};C.__index=C
function M.new(opts)
  return setmetatable({opts=opts,timer=assert(vim.uv.new_timer())},C)
end
function C:current(item)
  return not self.closed and self.opts.context(item.buf)==item.context
end
function C:schedule()
  if self.closed or self.active or not self.pending then return end
  self.timer:start(self.opts.delay or 100,0,vim.schedule_wrap(function()self:flush()end))
end
function C:request(buf,refresh)
  local ctx=self.opts.context(buf)
  if self.closed or not ctx or (not refresh and not self.opts.modified(buf)) then return end
  local previous=self.pending
  self.pending={buf=buf,context=ctx,refresh=refresh or (previous and previous.buf==buf and previous.context==ctx and previous.refresh)}
  self:schedule()
end
function C:flush()
  if self.closed or self.active then return end
  local item=self.pending;self.pending=nil
  if not item or not self:current(item) or (not item.refresh and not self.opts.modified(item.buf)) then return end
  item.revision=self.opts.revision and self.opts.revision(item.buf)
  self.active=item
  local observe=self.opts.observe or identity.resolve_async
  local selected=self.opts.selection and self.opts.selection(item.buf)
  local ok,handle=pcall(observe,item.context.repo,selected,function(observed)
    vim.schedule(function()
      if self.closed or self.active~=item then return end
      self.active=nil
      if self:current(item) then
        local ctx=item.context
        local matches=observed.repo==ctx.repo and observed.branch==ctx.branch
          and (observed.status=='missing' or (observed.status=='resolved' and observed.file==ctx.file))
        local revision=self.opts.revision and self.opts.revision(item.buf)
        local unchanged=vim.deep_equal(item.revision,revision)
        local action
        if self.opts.modified(item.buf) then
          -- Save this buffer's current bytes under its own binding. Matching
          -- results cannot discard newer requests; those remain queued below.
          if not matches then action=self.opts.preserve end
        elseif matches and item.refresh and unchanged and observed.status=='resolved' and type(observed.snapshot)=='string' then
          action=self.opts.refresh
        end
        if action then
          local success,err=pcall(action,item.buf,ctx,observed)
          if not success then (self.opts.notify or vim.notify)(tostring(err),vim.log.levels.ERROR) end
        end
      end
      -- New requests need a new observation: this result predates those
      -- events even if they happen to name the same activation and file.
      self:schedule()
    end)
  end)
  if ok then item.handle=handle else
    self.active=nil
    (self.opts.notify or vim.notify)(tostring(handle),vim.log.levels.ERROR)
    self:schedule()
  end
end
function C:cancel(buf)
  if self.closed then return end
  if self.pending and (not buf or self.pending.buf==buf) then self.pending=nil end
  if self.active and (not buf or self.active.buf==buf) then
    local old=self.active;self.active=nil
    if old.handle then pcall(old.handle.kill,old.handle,15) end
  end
  self.timer:stop()
  self:schedule()
end
function C:close()
  if self.closed then return end
  self:cancel();self.closed=true;self.timer:close()
end
return M
