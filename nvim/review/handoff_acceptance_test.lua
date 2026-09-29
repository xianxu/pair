-- Production watcher/orchestrator interleavings at the consume boundary.
local here=debug.getinfo(1,'S').source:match('@?(.*/)') or './'
local review=dofile(here..'init.lua')
local handoff=dofile(here..'handoff.lua')
local root=vim.fn.tempname(); vim.fn.mkdir(root,'p')
vim.env.PAIR_REVIEW_HANDOFF_PATH=root..'/handoff.json'
vim.env.PAIR_REVIEW_LANDED_PATH=root..'/landed.json'
local path=handoff.path('test')
local notices={}; local notify=vim.notify
vim.notify=function(msg) notices[#notices+1]=tostring(msg) end
review.poke={send=function() return true end}
local recs={{old='before',new='after',occurrence=1,explain='change'}}
local function begin()
  vim.fn.delete(path); vim.fn.delete(vim.env.PAIR_REVIEW_LANDED_PATH)
  local file=root..'/doc-'..tostring(vim.uv.hrtime())..'.md'
  vim.fn.writefile({'before'},file)
  local b=vim.fn.bufadd(file); vim.fn.bufload(b)
  review.authorize=nil; review.on_defer=nil; review.before_agent_round=nil; review.pane_state=nil
  review.admit=function() return true end
  review.start({buf=b,file=file,tag='test',identity={status='non_review'},watch_opts={interval=10}})
  return b
end
local b=begin(); local checks=0
review.authorize=function() checks=checks+1; return checks%2==1 end
handoff.write('test',recs)
assert(vim.wait(1000,function() return checks>=2 end,10))
review.stop(b)
assert(vim.fn.filereadable(path)==1,'final application refusal must preserve admitted handoff')
assert(vim.api.nvim_buf_get_lines(b,0,-1,false)[1]=='before','refusal must not mutate text')

b=begin(); local deferred=0
review.set_base(b,'before')
vim.api.nvim_buf_set_lines(b,0,-1,false,{'human edit'})
review.pane_state=function() return {focused=true,mode='i'} end
review.on_defer=function() deferred=deferred+1; return false end
handoff.write('test',recs)
assert(vim.wait(1000,function() return deferred>0 end,10))
review.stop(b)
assert(vim.fn.filereadable(path)==1,'refused deferral must preserve handoff')

b=begin(); deferred=0
review.set_base(b,'before')
vim.api.nvim_buf_set_lines(b,0,-1,false,{'human edit'})
review.pane_state=function() return {focused=true,mode='i'} end
review.on_defer=function(_,received) assert(vim.deep_equal(received,recs)); deferred=deferred+1; return true end
handoff.write('test',recs)
assert(vim.wait(1000,function() return deferred>0 and vim.fn.filereadable(path)==0 end,10))
review.stop(b)
assert(deferred==1,'accepted deferral consumes exactly once')

b=begin(); local called=0
review.authorize=function() called=called+1; error('injected authorization failure') end
handoff.write('test',recs)
assert(vim.wait(1000,function() return called>0 end,10))
review.stop(b)
assert(vim.fn.filereadable(path)==1,'callback error must preserve handoff')

-- Captured activation must reach the FINAL apply/defer authorization, even
-- when a pane-state observation changes the activation after first admission.
for _, defer in ipairs({false,true}) do
  b=begin()
  local active='old'
  local captured={repo='/repo',branch='review/doc',file='doc.md',activation='old'}
  local final_checks=0
  review.authorize=function(_,ctx)
    if ctx and ctx.activation==active then return true end
    final_checks=final_checks+1; return false
  end
  review.set_base(b,'before')
  if defer then vim.api.nvim_buf_set_lines(b,0,-1,false,{'human edit'}) end
  review.pane_state=function() active='new'; return {focused=defer,mode=defer and 'i' or 'n'} end
  review.on_defer=function(buf,_,ctx) return review.human_round(buf,'defer',ctx) end
  handoff.write('test',{context=captured,records=recs})
  assert(vim.wait(1000,function() return final_checks>0 end,10))
  review.stop(b)
  assert(vim.fn.filereadable(path)==1,'changed activation at final '..(defer and 'defer' or 'apply')..' guard must preserve payload')
  assert(vim.api.nvim_buf_get_lines(b,0,-1,false)[1]==(defer and 'human edit' or 'before'))
end

-- A caught apply failure is not a completed agent round. In particular, it
-- must not invoke the UI hook that clears an already-deferred retry slot.
b=begin()
local pending=recs
review.after_agent_round=function() pending=nil end
local original_set_text=vim.api.nvim_buf_set_text
vim.api.nvim_buf_set_text=function(...) error('injected apply failure') end
local applied,dropped,uncertain=review.apply_round(b,recs)
vim.api.nvim_buf_set_text=original_set_text
review.after_agent_round=nil
review.stop(b)
assert(applied==nil and uncertain==true,'caught apply failure must report uncertain non-acceptance')
assert(pending==recs,'failed apply must retain the deferred round retry slot')

-- A write failure happens AFTER buffer mutation. Preserve the response without
-- silently replaying it every poll; a producer's new generation may be offered.
b=begin(); checks=0
review.authorize=function() checks=checks+1; return true end
local fail_save=vim.api.nvim_create_autocmd('BufWriteCmd',{buffer=b,callback=function() error('injected save failure') end})
handoff.write('test',recs)
assert(vim.wait(1000,function() return checks>=2 end,10))
vim.wait(100,function() return false end,10)
assert(vim.fn.filereadable(path)==1,'save error must preserve admitted payload')
assert(checks==2,'uncertain mutated round must not be replayed every poll')
assert(vim.api.nvim_buf_get_lines(b,0,-1,false)[1]=='after')
vim.api.nvim_del_autocmd(fail_save)
handoff.write('test',{{old='after',new='final',occurrence=1}})
assert(vim.wait(1000,function() return vim.fn.filereadable(path)==0 end,10))
review.stop(b)
assert(vim.api.nvim_buf_get_lines(b,0,-1,false)[1]=='final','new generation remains eligible after uncertain failure')

-- Accepted work whose cleanup fails retries removal, never application.
vim.fn.delete(path)
local old_remove=os.remove
local removals,accepted=0,0
os.remove=function(p)
  if p==path then removals=removals+1; if removals==1 then return nil,'injected unlink failure' end end
  return old_remove(p)
end
local stop_cleanup=handoff.watch('test',function() accepted=accepted+1; return true end,{interval=10})
handoff.write('test',recs)
assert(vim.wait(1000,function() return removals>1 and vim.fn.filereadable(path)==0 end,10))
stop_cleanup(); os.remove=old_remove
assert(accepted==1,'unlink failure must not repeat accepted callback')

-- Callback publishes a replacement while accepting the old generation. The
-- watcher must not unlink the new payload, including byte-identical rewrites.
for _, replacement in ipairs({{{old='other',new='OTHER',occurrence=1}},recs}) do
  vim.fn.delete(path)
  local fired=false
  local stop=handoff.watch('test',function()
    if fired then return false end
    fired=true; handoff.write('test',replacement); return true
  end,{interval=10})
  handoff.write('test',recs)
  assert(vim.wait(1000,function() return fired end,10))
  stop()
  assert(vim.fn.filereadable(path)==1,'accepting old generation must preserve atomic replacement')
  assert(vim.deep_equal(vim.json.decode(table.concat(vim.fn.readfile(path),'\n')),replacement))
end

-- Waiting for an external guard/application can pump the event loop. A second
-- timer delivery must not reenter the same admission/application transaction.
vim.fn.delete(path)
local reentrant=0
local stop_reentrant=handoff.watch('test',function()
  reentrant=reentrant+1
  if reentrant==1 then vim.wait(80,function() return false end,10) end
  return true
end,{interval=10})
handoff.write('test',recs)
assert(vim.wait(1000,function() return vim.fn.filereadable(path)==0 end,10))
stop_reentrant()
assert(reentrant==1,'event-loop reentry must not offer one payload twice')

-- stop() invalidates callbacks already queued by the timer, as happens when
-- activation transfers ownership before scheduled delivery reaches the editor.
vim.fn.delete(path)
local schedule_wrap=vim.schedule_wrap
local queued={}
vim.schedule_wrap=function(fn) return function() queued[#queued+1]=fn end end
local called_after_stop=0
local stop_queued=handoff.watch('test',function() called_after_stop=called_after_stop+1; return true end,{interval=10})
vim.schedule_wrap=schedule_wrap
handoff.write('test',recs)
assert(vim.wait(1000,function() return #queued>0 end,10))
stop_queued()
for _, callback in ipairs(queued) do callback() end
assert(called_after_stop==0 and vim.fn.filereadable(path)==1,'stopped watcher must not consume queued work')

b=begin()
handoff.write('test',recs)
assert(vim.wait(1000,function() return vim.fn.filereadable(path)==0 end,10),'applied round must consume after acceptance')
review.stop(b)
assert(vim.api.nvim_buf_get_lines(b,0,-1,false)[1]=='after')
assert(vim.fn.filereadable(vim.env.PAIR_REVIEW_LANDED_PATH)==1)
vim.notify=notify
vim.fn.delete(root,'rf')
print('handoff acceptance tests passed')
