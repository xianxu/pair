-- One live operator attempt. Durable delivery truth is owned by Couch; this
-- adapter never interprets a missing acknowledgment as permission to resend.
local M = {}
function M.new(d)
  local pending, ack_id, busy = nil, nil, false
  local m = {}
  local function report(err) d.notify('DOM submission: ' .. tostring(err)) end
  local function acknowledge(id)
    local function observed(r, err, polls)
      if not err and r and r.state == 'resolved' then ack_id=nil;busy=false;return end
      if not err and r and r.state == 'dispatched' and (not r.error or r.error=='') and polls < 360 then
        d.defer(function()d.rpc('status',id,function(next_r,next_err)observed(next_r,next_err,polls+1)end)end,100)
        return
      end
      -- A dispatch remains final even when note-state persistence fails.
      ack_id=id;busy=false
      report(err or (r and r.error ~= '' and r.error) or 'delivered; note resolution is pending (press the chord to retry acknowledgment)')
    end
    d.rpc('ack',id,function(r,err)observed(r,err,0)end)
  end
  local function deliver(done)
    if not pending.current() then busy = false; report('draft or view changed; return to the original draft to retry'); return end
    d.rpc('begin', pending.id, function(r, err)
      if err or not r or r.state ~= 'delivering' then busy = false;report(err or 'delivery intent was not confirmed');return end
      if not pending.current() then busy = false;report('draft changed before delivery; retry the original draft');return end
      local sent = d.submit(pending.body, pending.agent_text, {id=pending.id,target=pending.target})
      if not sent then busy = false; return end
      local id = pending.id
      pending = nil; ack_id = id
      done() -- Dispatch, not resolution, consumes the authored draft.
      acknowledge(id)
    end)
  end
  local function prepared(r, err, done, polls)
    if err or not r then busy=false;report(err or 'preparation response missing');return end
    if r.state == 'preparing' then
      if polls >= 360 then busy=false;report('preparation timed out; retry to inspect the same submission');return end
      d.defer(function() d.rpc('status',pending.id,function(next_r,next_err) prepared(next_r,next_err,done,polls+1) end) end,100)
    elseif r.state == 'prepared' then
      if type(r.path) ~= 'string' or r.path:sub(1,1) ~= '/' then busy=false;report('missing absolute bundle path');return end
      pending.agent_text = pending.text .. (pending.text ~= '' and '\n\n' or '') .. 'DOM notes and current PNG crops: read ' .. r.path .. ' (submission.json).'
      deliver(done)
    elseif r.state == 'empty' then
      local p = pending;pending=nil;busy=false
      if p.current() and p.text:match('%S') and d.ordinary(p.body,p.text) then done() end
    elseif r.state == 'failed' then
      pending=nil;busy=false;report(r.error or 'preparation failed')
    else
      busy=false;report('existing submission is ' .. tostring(r.state) .. '; reconcile delivery at ' .. tostring(r.path))
    end
  end
  function m.send(body,text,current,done)
    if busy then report('already preparing or resolving');return end
    if ack_id then busy=true;acknowledge(ack_id);return end
    if pending and (pending.body ~= body or pending.text ~= text) then report('retry the original draft before changing its submission');return end
    if not pending then
      local id=d.id()
      local target,err=d.pin(id)
      if not target then report(err or 'agent pane unavailable');return end
      pending={id=id,body=body,text=text,target=target,current=current}
    else pending.current=current end
    busy=true
    if pending.agent_text then deliver(done);return end
    d.rpc('prepare',pending.id,function(r,err) prepared(r,err,done,0) end)
  end
  return m
end
return M
