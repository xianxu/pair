local module=dofile('nvim/dom_submission.lua')
local function fixture()
 local f={current=true,calls={},messages={},sent={},done=0,preps={},acks={},deferred={},serial=0,fail_prepare=false,fail_send=false,fail_ack=false}
 f.time=0; f.timers={}; f.views={}
 local d={now=function() return f.time end,
 arm=function(fn,ms) local t={fn=fn,ms=ms};f.timers[#f.timers+1]=t;return function()t.cancelled=true end end,
 render=function(message,busy)f.views[#f.views+1]={message=message,busy=busy};f.locked=busy end,
 notify=function(e)table.insert(f.messages,e)end,
 id=function()f.serial=f.serial+1;return 'bundle-'..f.serial end,
 pin=function()return {pane_id=7}end,
 defer=function(fn)table.insert(f.deferred,fn)end,
 ordinary=function()return true end}
 d.rpc=function(op,id,cb)
  f.calls[#f.calls+1]={op=op,id=id}
  if op=='prepare' then
   f.preps[#f.preps+1]=id
   if f.fail_prepare then cb(nil,'unexpected submission entry .DS_Store')
   else cb({state='prepared',path='/store/'..id..'/submission.json',how_to_read='guide'})end
  elseif op=='begin' then cb({state='delivering'})
  elseif op=='ack' then
   f.acks[#f.acks+1]=id
   if f.fail_ack then cb(nil,'lost ack')else cb({state='resolved'})end
  else error(op) end
 end
 d.submit=function(body,text,attachment)
  f.sent[#f.sent+1]={body=body,text=text,id=attachment.id}
  assert(attachment.target.pane_id==7 and text:sub(-5)=='guide')
  return not f.fail_send
 end
 f.m=module.new(d)
 f.send=function(body)f.m.send(body,body,function()return f.current end,function()f.done=f.done+1 end)end
 return f,d
end
for _,failure in ipairs({'fail_prepare','fail_send'})do
 local f=fixture();f[failure]=true;f.send('original lost')
 assert(f.done==0)
 f[failure]=false;f.send('edited current')
 assert(f.done==1 and #f.preps==2 and f.preps[1]~=f.preps[2])
 assert(f.sent[#f.sent].body=='edited current' and f.sent[#f.sent].id=='bundle-2')
end

do
 local f=fixture();f.fail_ack=true;f.send('one')
 assert(f.done==1 and #f.deferred==1)
 f.fail_ack=false;f.send('two')
 assert(f.done==2 and #f.sent==2 and f.sent[2].body=='two')
 table.remove(f.deferred,1)()
 assert(f.done==2 and f.acks[#f.acks]=='bundle-1','old receipt consumed or lost newer draft')
end

do
 local f,d=fixture();local replies={};local rpc=d.rpc
 d.rpc=function(op,id,cb)if op=='prepare' then replies[id]=cb else rpc(op,id,cb)end end
 f.send('one');assert(f.timers[1], 'operation deadline was not armed');f.time=36001;f.timers[1].fn();f.send('two')
 replies['bundle-1']({state='prepared',path='/old'})
 assert(#f.sent==0 and f.done==0,'stale prepare sent')
 replies['bundle-2']({state='prepared',path='/new',how_to_read='guide'})
 assert(#f.sent==1 and f.sent[1].body=='two' and f.done==1)
 replies['bundle-2']({state='prepared',path='/new'})
 assert(#f.sent==1 and f.done==1,'duplicate prepare sent')
end

-- A new controller has no old in-memory body lock and accepts the saved edit.
do
 local old=fixture();old.fail_prepare=true;old.send('unrecoverable original')
 local reopened=fixture();reopened.send('saved edit')
 assert(reopened.done==1 and reopened.sent[1].body=='saved edit')
end
do
 local f,d=fixture();local submit=d.submit
 d.submit=function(...)
  f.current=false -- the operator edits during the pane-send settle window
  return submit(...)
 end
 f.send('old')
 assert(#f.sent==1 and #f.acks==1 and f.done==0,'send completion consumed edited draft')
end
print('dom_submission_test: passed')

do
 local f,d=fixture(); local cb
 d.rpc=function(op,id,answer)cb=answer end
 f.send('one')
 assert(f.locked and f.views[1].message=='preparing submission…','no immediate progress/lock')
 f.send('two');assert(f.serial==1 and f.views[#f.views].message=='already sending…')
 f.time=36001;f.timers[1].fn()
 assert(not f.locked and f.views[#f.views].message:find('timed out',1,true))
 cb({state='prepared',path='/old'})
 assert(#f.sent==0 and not f.locked)
end
for _,name in ipairs({'pin','rpc','submit','ordinary'})do
 local f,d=fixture()
 d[name]=function()error('injected '..name)end
 if name=='ordinary' then d.rpc=function(op,id,cb)cb({state='empty'})end end
 f.send('one')
 assert(not f.locked and f.done==0,'exception retained lock: '..name)
 assert(f.views[#f.views].message:find('injected '..name,1,true))
end

do
 local f,d=fixture();local submit=d.submit
 d.submit=function(...)
  f.time=36001;f.timers[1].fn()
  return submit(...)
 end
 f.send('one')
 assert(not f.locked and f.done==0 and #f.acks==1,'late confirmed delivery lost receipt or cleared expired draft')
end

do
 local f,d=fixture();local replies={};local rpc=d.rpc
 d.rpc=function(op,id,cb)
  if op=='ack' then replies[id]=cb else rpc(op,id,cb)end
 end
 f.send('sent')
 d.rpc=function(op,id,cb)end
 f.send('waiting')
 replies['bundle-1'](42)
 assert(f.locked and f.views[#f.views].busy,'malformed old receipt released newer operation')
end
