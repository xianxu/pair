local Observer=dofile('nvim/review/recovery_observer.lua')
local contexts={[1]={repo='/repo',branch='review/a',file='a.md',activation='a1'}}
local modified={[1]=true};local callbacks,handles,saved={},{},{}
local function observe(_,_,callback)
 callbacks[#callbacks+1]=callback
 local handle={killed=false};function handle:kill() self.killed=true end
 handles[#handles+1]=handle;return handle
end
local observer=Observer.new({delay=5,context=function(buf)return contexts[buf] end,
 modified=function(buf)return modified[buf] end,observe=observe,
 preserve=function(buf,ctx)saved[#saved+1]={buf=buf,context=ctx} end})
for _=1,20 do observer:request(1) end
assert(vim.wait(200,function()return #callbacks==1 end,5),'coalesced job did not start')
for _=1,20 do observer:request(1) end
assert(#callbacks==1,'parallel observations started')
callbacks[1]({status='resolved',repo='/repo',branch='review/b',file='b.md'})
assert(vim.wait(200,function()return #saved==1 end,5))
vim.wait(30,function()return false end,5)
assert(#callbacks==1,'newer edits with same context repeated the completed observation')
observer:request(1);assert(vim.wait(200,function()return #callbacks==2 end,5))
observer:cancel(1)
contexts[1]={repo='/repo',branch='review/b',file='b.md',activation='b1'}
callbacks[2]({status='resolved',repo='/repo',branch='review/other',file='other.md'})
vim.wait(20,function()return false end,5)
assert(handles[2].killed and #saved==1,'canceled old activation saved under new identity')
observer:request(1);assert(vim.wait(200,function()return #callbacks==3 end,5))
contexts[1]={repo='/repo',branch='review/b',file='b.md',activation='b2'}
callbacks[3]({status='invalid',diagnostic='timeout'})
vim.wait(20,function()return false end,5)
assert(#saved==1,'late result saved after activation changed')
observer:request(1);assert(vim.wait(200,function()return #callbacks==4 end,5))
modified[1]=false;callbacks[4]({status='invalid'})
vim.wait(20,function()return false end,5)
assert(#saved==1,'late result saved a clean buffer')
modified[1]=true;observer:request(1);assert(vim.wait(200,function()return #callbacks==5 end,5))
observer:close();callbacks[5]({status='invalid'})
vim.wait(20,function()return false end,5)
assert(handles[5].killed and #saved==1,'closed observer retained work')
print('recovery_observer_test ok')
