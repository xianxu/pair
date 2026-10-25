-- Actual CLI integration: only identity/RPC are injected; native observations
-- and mutations execute production modules through the shared private fake.
local repo=vim.fn.getcwd()
package.path=repo..'/nvim/?.lua;'..package.path
local module=require('review.restore_client')
local route=require('workbench_route')
local old_remote=_G.PairReviewPane
local envkeys={'PATH','HOME','ZELLIJ_SOCKET_DIR','PAIR_FAKE_ZELLIJ_STATE','ZELLIJ_SESSION_NAME','ZELLIJ_PANE_ID','PAIR_ZELLIJ_TRACE'}
local original={}
for _,key in ipairs(envkeys)do original[key]=vim.env[key]end
local made=vim.system({'mktemp','-d',(vim.env.TMPDIR or '/tmp')..'/pair-review-admission.XXXXXX'},{text=true}):wait()
assert(made.code==0,made.stderr)
local root=vim.trim(made.stdout);assert(root~='')
local function write(path,value)vim.fn.writefile({vim.json.encode(value)},path)end
local function read(path)return vim.json.decode(table.concat(vim.fn.readfile(path),'\n'))end
local function pane(id,tab,command,floating)
 return {id=id,is_plugin=false,is_focused=id==2,is_floating=floating,is_fullscreen=false,
 tab_id=tab,tab_position=1,terminal_command=command}
end
local function model()
 return {sessions={['session-a']={panes={pane(2,4,'nvim -u /pair/nvim/init.lua',false),
 pane(7,4,'nvim -u /pair/nvim/review.lua',true)},tab={tab_id=4,position=1},floating_visible=true}}}
end
local function main()
 vim.fn.mkdir(root..'/bin','p')
 -- Use the suite-owned executable when supplied; standalone runs use the same
 -- builder. No fake Pair route may stand in for namespace admission.
 local built=vim.system({'bash',repo..'/tests/lib/pair-test-binary.sh',root..'/bin/pair'},
  {text=true}):wait()
 assert(built.code==0,'Pair helper build failed: '..(built.stderr or ''))
 vim.fn.mkdir(root..'/home','p')
 vim.env.HOME=root..'/home'
 -- No real native endpoints are published: the stateful fake's model supplies
 -- replies, while the real helper qualifies this private empty socket namespace.
 vim.env.ZELLIJ_SOCKET_DIR=root..'/native'
 vim.fn.writefile(vim.fn.readfile(repo..'/tests/lib/fake-zellij.py'),root..'/bin/zellij')
 vim.fn.writefile(vim.fn.readfile(repo..'/tests/lib/zellij_fixture_replies.py'),root..'/bin/zellij_fixture_replies.py')
 assert(vim.fn.setfperm(root..'/bin/zellij','rwx------')==1)
 vim.env.PATH=root..'/bin:'..original.PATH
 vim.env.ZELLIJ_SESSION_NAME='session-a';vim.env.ZELLIJ_PANE_ID='2';vim.env.PAIR_ZELLIJ_TRACE='0'
 for _,scenario in ipairs({'hide','show','activated-show','wrong-first','wrong-confirmation','context-drift','visibility-disagreement'})do
  local dir=root..'/'..scenario;vim.fn.mkdir(dir,'p')
  local statefile=dir..'/state.json';vim.env.PAIR_FAKE_ZELLIJ_STATE=statefile
  local state={model=model(),ordinal=0,queries={},effects={},faults={}}
  if scenario=='show' or scenario=='activated-show' then state.model.sessions['session-a'].floating_visible=false end
  if scenario=='show' then state.faults={{ordinal=2,error='false'},{ordinal=4,error='false'}} end
  if scenario=='wrong-first' then state.faults={{ordinal=1,reply='CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n'}} end
  if scenario=='wrong-confirmation' then state.faults={{ordinal=3,reply='{"wrong":"kind"}'}} end
  if scenario=='context-drift' then
   local changed=model();for _,p in ipairs(changed.sessions['session-a'].panes)do p.tab_id=9 end
   state.faults={{ordinal=3,model=changed}}
  end
  if scenario=='visibility-disagreement' then state.faults={{ordinal=4,error='false'}} end
  write(statefile,state)
  local wanted={status='resolved',repo='/private/repo',branch='review/a',file='review.md',head='sha-a'}
  local context={repo=wanted.repo,branch=wanted.branch,file=wanted.file,activation='active-a'}
  local metadata={version=1,endpoint='/private/fake-review-socket',token='private-token',session='session-a',context=context}
  local sf=dir..'/pane.open'
  vim.fn.writefile({tostring(vim.fn.getpid()),wanted.repo..'/old.md',vim.json.encode(metadata)},sf)
  local notices,published={},{}
  local pair_lookups,native_runs=0,0
  local client=module.new({state_file=function()return sf end,session=function()return 'session-a'end,
   cwd=function()return '/private/repo'end,read_target=function()return nil end,
   pair_bin=function()pair_lookups=pair_lookups+1;return root..'/bin/pair'end,
   prompt=function()error('must not prompt')end,
   write_target=function(file,status,id)published[#published+1]={file=file,status=status,identity=id}end,
   notify=function(msg)notices[#notices+1]=msg end,
   hide=function(b,admitted)return route.return_to_draft(b,admitted)end,
   run=function(cmd,opts,callback)
    assert(cmd[1]~='zellij','native action bypassed the guarded Pair helper')
    assert(not vim.tbl_contains(cmd,'open'),'must not open a replacement review pane')
    if cmd[1]==root..'/bin/pair' then
     native_runs=native_runs+1
     assert(pair_lookups==native_runs,'native binary lookup count diverged')
     assert(vim.deep_equal(vim.list_slice(cmd,2,4),{'internal','zellij-command','--'}),
      'native action must traverse the real internal helper')
     if vim.tbl_contains(cmd,'list-panes') or vim.tbl_contains(cmd,'are-floating-panes-visible') then
      assert(opts.timeout>0 and opts.timeout<=5000,'query lost admission timeout')
     end
     return vim.system(cmd,opts,callback)
    end
    assert(pair_lookups==native_runs,'identity/RPC triggered native binary lookup')
    local result={code=0,stdout='',stderr=''}
    if vim.tbl_contains(cmd,'--resolve') then result.stdout=vim.json.encode(wanted)
    elseif vim.tbl_contains(cmd,'--remote-expr') then
     _G.PairReviewPane={restore=function(raw)
      local request=vim.json.decode(raw)
      assert(request.token==metadata.token and request.session==metadata.session)
      assert(vim.deep_equal(request.identity,wanted))
      return {ok=true,same=scenario~='activated-show',identity=wanted,context=context}
     end}
     result.stdout=vim.fn.eval(cmd[#cmd])
    else error('unexpected non-native argv '..vim.inspect(cmd))end
    callback(result);return {}
   end})
  client:toggle()
  assert(vim.wait(5000,function()return not client.busy end,5),'restore did not finish: '..scenario)
  assert(pair_lookups==native_runs,'identity/RPC triggered native binary lookup: '..vim.inspect(notices))
  assert(native_runs>0,'native helper route was not exercised')
  local observed=read(statefile)
  assert(#published==1 and published[1].status=='ready' and published[1].file==wanted.repo..'/'..wanted.file,
   'activation target publication lost on '..scenario)
  local retained=vim.json.decode(vim.fn.readfile(sf)[3])
  assert(retained.context.activation=='active-a','activation metadata was lost')
  local healthy=scenario=='hide' or scenario=='show' or scenario=='activated-show'
  if healthy then
   assert(#notices==0,'healthy refusal '..scenario..': '..vim.inspect(notices)..' state='..vim.inspect(observed))
   if scenario=='hide' then
    assert(#observed.effects==2)
    assert(observed.effects[1].action=='hide-floating-panes' and observed.effects[1].target=='tab_4')
    assert(vim.deep_equal(observed.effects[1].args,{'action','hide-floating-panes','--tab-id','4'}))
    assert(observed.effects[2].action=='focus-pane-id' and observed.effects[2].target=='terminal_2')
    assert(vim.deep_equal(observed.effects[2].args,{'action','focus-pane-id','2'}))
    assert(#observed.queries==6,'hide made extra observations')
   else
    assert(#observed.effects==1 and observed.effects[1].action=='show-floating-panes' and observed.effects[1].target=='tab_4')
    assert(vim.deep_equal(observed.effects[1].args,{'action','show-floating-panes','--tab-id','4'}))
    assert(#observed.queries==(scenario=='show' and 5 or 3),'show query count changed')
   end
  else
   assert(#observed.effects==0,'uncertain native reply admitted effects: '..scenario)
   assert(#notices==1 and notices[1]:find('visibility refused',1,true),'refusal was not visible: '..scenario)
   local expected=scenario=='wrong-first' and 1 or scenario=='visibility-disagreement' and 4 or 3
   assert(#observed.queries==expected,'unknown observation did not terminate: '..scenario)
  end
  for _,p in ipairs(observed.model.sessions['session-a'].panes)do
   assert(p.input==nil or p.input=='','review visibility injected pane input')
  end
 end
end
local ok,reason=xpcall(main,debug.traceback)
for _,key in ipairs(envkeys)do vim.env[key]=original[key]end
_G.PairReviewPane=old_remote
vim.fn.delete(root,'rf')
assert(ok,reason)
print('review restore admission integration passed')
