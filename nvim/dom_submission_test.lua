local module=dofile('nvim/dom_submission.lua')
local function fixture()
 local f={calls={},messages={},sent=0,done=0,preps=0,acks=0,fail_send=true,fail_ack=true}
 local d={notify=function(e)table.insert(f.messages,e)end,id=function()return 'bundle' end,pin=function()return {pane_id=7} end,defer=function(fn)fn()end,ordinary=function()error('unexpected ordinary send')end}
 d.rpc=function(op,id,cb)
 table.insert(f.calls,op)
 if op=='prepare' then f.preps=f.preps+1;cb({state='prepared',path='/store/bundle/submission.json'})
 elseif op=='begin' then cb({state='delivering'})
 elseif op=='ack' then f.acks=f.acks+1;if f.fail_ack then cb(nil,'lost ack')else cb({state='resolved'})end end
 end
 d.submit=function(body,text,attachment)
 f.sent=f.sent+1;assert(body=='draft');assert(attachment.id=='bundle');assert(text:find('/store/bundle/submission.json',1,true));return not f.fail_send
 end
 f.m=module.new(d);f.current=function()return true end;f.complete=function()f.done=f.done+1 end
 return f,d
end
local f=fixture()
f.m.send('draft','draft',f.current,f.complete)
assert(f.preps==1 and f.sent==1 and f.done==0)
f.m.send('edited','edited',f.current,f.complete);assert(f.sent==1)
f.fail_send=false;f.m.send('draft','draft',f.current,f.complete)
assert(f.preps==1 and f.sent==2 and f.done==1 and f.acks==1)
f.fail_ack=false;f.m.send('new draft','new draft',f.current,f.complete)
assert(f.sent==2 and f.done==1 and f.acks==2)
local g,d=fixture();g.fail_send=false;g.current=function()return false end
g.m.send('draft','draft',g.current,g.complete);assert(g.sent==0 and g.done==0)
local h,hd=fixture();hd.rpc=function(op,id,cb)cb({state='empty'})end;hd.ordinary=function(body,text)return true end
h.m.send('draft','draft',h.current,h.complete);assert(h.done==1 and h.sent==0)
local k,kd=fixture();kd.rpc=function(op,id,cb)cb({state='failed',error='page changed'})end
k.m.send('draft','draft',k.current,k.complete);assert(k.sent==0 and k.done==0)
local a,ad=fixture();a.fail_send=false
ad.rpc=function(op,id,cb)
 if op=='prepare' then a.preps=a.preps+1;cb({state='prepared',path='/store/bundle/submission.json'})
 elseif op=='begin' then cb({state='delivering'})
 elseif op=='ack' then a.acks=a.acks+1;cb({state='dispatched'})
 elseif op=='status' then cb({state='resolved'})end
end
a.m.send('draft','draft',a.current,a.complete)
assert(a.done==1 and a.sent==1 and a.acks==1)
a.m.send('draft','draft',a.current,a.complete)
assert(a.done==2 and a.sent==2 and a.preps==2,'successful async ack must not swallow the next chord')
print('dom_submission_test: passed')
