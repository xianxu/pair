-- Stateful completion seam: no real resolver, timers, or subprocess ownership.
local module = 'tests/lib/review_observation_trace.lua'
assert(vim.fn.filereadable(module)==1, 'missing observation trace implementation')
local command={'pair','review','readiness','--resolve','/repo'}
local sentinel='DOCUMENT_BYTES_MUST_NOT_APPEAR'
local good={status='resolved',repo='/repo',branch='review/a',head='abc',file='a.md',snapshot=sentinel}
local result={code=0,signal=0,stdout=vim.json.encode(good),stderr=''}
local processes, pending={},{}
local expected_opts, expected_timeout
local original_system=vim.system
local wait_failure
vim.system=function(cmd,opts,callback)
  if expected_opts then assert(opts==expected_opts,'system options were copied or changed') end
  local process={}
  process.wait=function(self,...)
    assert(self==process,'wait receiver changed')
    local args={...};assert(#args==1 and args[1]==expected_timeout,'wait arguments changed')
    if wait_failure then error(wait_failure) end
    return result
  end
  process.kill=function() return 'kill unchanged' end
  processes[#processes+1]=process
  if callback then pending[#pending+1]=function()callback(result)end end
  return process
end
local ack={ok=true,identity=good,context={repo='/repo',branch='review/a',file='a.md',activation='activation'}}
local late
local expected_raw
PairReviewPane={restore=function(raw)
  if expected_raw then assert(raw==expected_raw,'restore input object replaced') end
  if raw=='throw' then error('restore failure') end
  expected_opts={text=true};expected_timeout=123
  local process=vim.system(command,expected_opts)
  assert(process==processes[#processes],'process object replaced')
  assert(process:wait(expected_timeout)==result,'wait result replaced')
  assert(process:kill()=='kill unchanged')
  expected_opts={text=true,timeout=456}
  process=vim.system(command,expected_opts,function(actual)
    assert(actual==result,'callback result replaced');late='called'
  end)
  assert(process==processes[#processes],'async process object replaced')
  return ack,'second return'
end}
local fallback=vim.fn.tempname()
local writes=0
local original_write=vim.fn.writefile
vim.fn.writefile=function(...)writes=writes+1;return original_write(...)end
dofile(module).install({path=fallback})
local returned,second=PairReviewPane.restore(vim.json.encode({identity=good}))
assert(returned==ack and second=='second return','restore returns changed')
assert(late==nil and writes==0,'completion scheduled or hot-path file write')
local before=PairReviewObservationTraceSnapshot()
assert(#before.events==5,'expected request begin, two starts, wait completion, request end')
local request=before.events[1].request_id
assert(request~=nil,'request id absent')
assert(before.events[3].event=='completion' and before.events[3].mode=='wait')
assert(before.events[3].timeout==123 and before.events[3].identity.status=='resolved')
assert(before.events[4].timeout==456 and before.events[4].mode=='callback')
pending[1]()
assert(late=='called','callback not called inline')
local snapshot=PairReviewObservationTraceSnapshot()
local completed=snapshot.events[#snapshot.events]
assert(completed.event=='completion' and completed.mode=='callback' and completed.request_id==request,'late callback lost originating request')
for i,event in ipairs(snapshot.events) do assert(event.seq==i and event.request_id==request,'ordering or request correlation lost') end
for i,event in ipairs(snapshot.events) do
  assert(type(event.at_ns)=='number','monotonic event time absent')
  if i>1 then assert(event.at_ns>=snapshot.events[i-1].at_ns,'monotonic event time regressed') end
  if event.event=='completion' then assert(type(event.elapsed_ms)=='number' and event.elapsed_ms>=0,'resolver elapsed time absent') end
end
assert(not vim.json.encode(snapshot):find(sentinel,1,true),'snapshot bytes retained')
assert(writes==0 and vim.fn.filereadable(fallback)==0,'trace wrote during observation')
expected_raw={identity=good}
assert(PairReviewPane.restore(expected_raw)==ack,'table restore return changed')
snapshot=PairReviewObservationTraceSnapshot()
local begin=snapshot.events[#snapshot.events-4]
assert(begin.event=='request-begin' and begin.identity.repo=='/repo' and begin.identity.head=='abc','table wanted identity absent')
expected_raw=nil
-- Non-resolver calls pass straight through and add no events.
expected_opts={};local n=#snapshot.events
vim.system({'git','status'},expected_opts)
assert(#PairReviewObservationTraceSnapshot().events==n,'non-resolver traced')
-- Preserve malformed and invalid completion evidence without copying stdout.
expected_opts={};expected_timeout=9
for _,stdout in ipairs({'{broken',vim.json.encode({status='invalid',diagnostic='deadline exceeded',repo='/repo'})}) do
  result={code=0,signal=0,stdout=stdout,stderr=string.rep('x',5000)}
  vim.system(command,expected_opts):wait(expected_timeout)
  completed=PairReviewObservationTraceSnapshot().events
  completed=completed[#completed]
  assert(completed.stdout_bytes==#stdout and #completed.stderr<=1024,'result evidence absent or unbounded')
  if stdout=='{broken' then assert(completed.decode_error,'invalid JSON diagnostic absent')
  else assert(completed.identity.status=='invalid' and completed.identity.diagnostic=='deadline exceeded') end
end
wait_failure='controlled wait failure'
local failed,wait_error=pcall(function()vim.system(command,expected_opts):wait(expected_timeout)end)
assert(not failed and tostring(wait_error):find(wait_failure,1,true),'wait exception changed')
snapshot=PairReviewObservationTraceSnapshot()
assert(snapshot.events[#snapshot.events].event=='completion' and snapshot.events[#snapshot.events].error:find(wait_failure,1,true),'wait failure outcome absent')
wait_failure=nil
local ok,err=pcall(PairReviewPane.restore,'throw')
assert(not ok and tostring(err):find('restore failure',1,true),'restore exception swallowed')
snapshot=PairReviewObservationTraceSnapshot()
assert(snapshot.events[#snapshot.events].event=='request-end' and snapshot.events[#snapshot.events].error:find('restore failure',1,true),'restore exception not retained')
for _=1,150 do vim.system(command,expected_opts):wait(expected_timeout) end
snapshot=PairReviewObservationTraceSnapshot()
assert(#snapshot.events<=256 and snapshot.dropped>0,'event retention unbounded')
for i=2,#snapshot.events do assert(snapshot.events[i].seq>snapshot.events[i-1].seq,'sequence reset during eviction') end
assert(writes==0,'hot-path write during retention overflow')
vim.api.nvim_exec_autocmds('VimLeavePre',{})
assert(writes==1,'fallback did not flush once')
local flushed=vim.json.decode(table.concat(vim.fn.readfile(fallback),'\n'))
assert(flushed.dropped==snapshot.dropped and #flushed.events==#snapshot.events,'fallback lost events')
vim.fn.delete(fallback)
vim.system=original_system;vim.fn.writefile=original_write
-- The real SystemObj keeps its actual wait receiver and process methods.
dofile(module).install()
local actual=vim.system({'/bin/echo','review','readiness','--resolve','/repo'},{text=true})
assert(type(actual.kill)=='function' and type(actual.is_closing)=='function','SystemObj methods lost')
local actual_result=actual:wait()
assert(actual_result.code==0 and actual_result.stdout=='review readiness --resolve /repo\n','actual SystemObj result changed')
snapshot=PairReviewObservationTraceSnapshot()
assert(#snapshot.events==2 and snapshot.events[2].decode_error and snapshot.events[2].stdout_bytes==#actual_result.stdout,'actual wait outcome absent')
print('review-observation-trace-test ok')
