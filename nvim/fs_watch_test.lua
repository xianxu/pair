local watch=dofile('nvim/fs_watch.lua')
local function fixture(failure)
  local state={stops=0,closes=0,queued={},delivered=0}
  local handle={}
  function handle:start(path,opts,callback)
    assert(path=='/fixture' and next(opts)==nil)
    state.event=callback
    if failure=='throw' then error('start refused') end
    if failure=='return' then return nil,'EMFILE' end
    return 0
  end
  function handle:is_closing() return state.closing or false end
  function handle:stop() state.stops=state.stops+1 end
  function handle:close() state.closes=state.closes+1;state.closing=true end
  local api={loop={new_fs_event=function() if failure~='allocate' then return handle end end},
    schedule=function(callback) table.insert(state.queued,callback) end,
    api={nvim_create_autocmd=function(event,opts)
      assert(event=='VimLeavePre' and opts.once);state.leave=opts.callback
    end}}
  local close=watch.start(api,'/fixture',function(err,name)
    assert(err==nil and name=='target');state.delivered=state.delivered+1
  end)
  function state.flush() for _,callback in ipairs(state.queued) do callback() end;state.queued={} end
  return state,close
end
local state,close=fixture()
state.event(nil,'target');state.flush();assert(state.delivered==1)
state.event(nil,'target') -- queued delivery must lose authority at shutdown
state.leave();state.flush();close();state.leave()
assert(state.stops==1 and state.closes==1 and state.delivered==1)
state.event(nil,'target');state.flush();assert(state.delivered==1)
for _,failure in ipairs({'allocate','throw','return'}) do
  local failed,stop=fixture(failure)
  assert(stop==nil and failed.leave==nil)
  assert(failed.closes==(failure=='allocate' and 0 or 1))
end
print('fs_watch: close ownership, late callbacks and start refusals passed')
