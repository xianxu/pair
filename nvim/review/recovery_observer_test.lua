local Observer=dofile('nvim/review/recovery_observer.lua')
local contexts={[1]={repo='/repo',branch='review/a',file='a.md',activation='a1'}}
local modified={[1]=true};local revision=0
local callbacks,handles,saved,refreshed={},{},{},{}
local function observe(_,_,callback)
 callbacks[#callbacks+1]=callback
 local handle={killed=false};function handle:kill()self.killed=true end
 handles[#handles+1]=handle;return handle
end
local observer=Observer.new({delay=5,context=function(buf)return contexts[buf] end,
 modified=function(buf)return modified[buf] end,revision=function()return revision end,observe=observe,
 preserve=function(buf,ctx)saved[#saved+1]={buf=buf,context=ctx} end,
 refresh=function(_,_,observed)refreshed[#refreshed+1]=observed.snapshot end})
local function await(n)assert(vim.wait(200,function()return #callbacks==n end,5),'expected observation '..n)end
local function settle()vim.wait(25,function()return false end,5)end
local matching={status='resolved',repo='/repo',branch='review/a',file='a.md',snapshot='captured A'}
for _=1,20 do observer:request(1) end
await(1)
revision=1
for _=1,20 do observer:request(1) end
assert(#callbacks==1,'parallel observations started')
callbacks[1](matching);await(2)
assert(#saved==0,'matching observation created unnecessary recovery')
callbacks[2]({status='resolved',repo='/repo',branch='review/b',file='b.md'});settle()
assert(#callbacks==2 and #saved==1,'coalesced request was lost or repeated')
observer:request(1);await(3);observer:cancel(1)
contexts[1]={repo='/repo',branch='review/a',file='a.md',activation='a2'}
callbacks[3]({status='invalid'});settle()
assert(handles[3].killed and #saved==1,'canceled old activation saved under new identity')
observer:request(1);await(4)
contexts[1]={repo='/repo',branch='review/a',file='a.md',activation='a3'}
callbacks[4]({status='invalid'});settle()
assert(#saved==1,'late result saved after activation changed')
observer:request(1);await(5);modified[1]=false
callbacks[5]({status='invalid'});settle()
assert(#saved==1,'late result saved a clean buffer')
observer:request(1,true);await(6);revision=2
callbacks[6](matching);settle()
assert(#refreshed==0,'old snapshot overwrote a newer clean buffer revision')
observer:request(1,true);await(7);callbacks[7](matching);settle()
assert(refreshed[1]=='captured A','refresh did not receive identity-bound bytes')
observer:request(1,true);await(8)
callbacks[8]({status='resolved',repo='/repo',branch='review/a',file='a.md'});settle()
assert(#refreshed==1,'identity without captured bytes authorized refresh')
modified[1]=true;observer:request(1);await(9)
observer:close();callbacks[9]({status='invalid'});settle()
assert(handles[9].killed and #saved==1,'closed observer retained work')
print('recovery_observer_test ok')
