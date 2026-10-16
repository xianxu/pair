-- Fixture instrumentation only. Observe the resolver's existing completion
-- boundary without adding subprocesses, callbacks, scheduling, or hot-path IO.
local M={}
local unpack=unpack or table.unpack
local function pack(...) return {n=select('#',...),...} end
local function clip(value,limit)
  if type(value)=='string' then return value:sub(1,limit or 512) end
  if type(value)=='number' or type(value)=='boolean' then return value end
end
local function project(value,context)
  if type(value)~='table' then return {value_type=type(value)} end
  local out={}
  for _,key in ipairs(context and {'repo','branch','file','activation'} or
    {'status','repo','branch','head','file','diagnostic'}) do
    out[key]=clip(value[key],key=='diagnostic' and 1024 or 512)
  end
  return out
end
function M.install(opts)
  opts=opts or {}
  local events,dropped,sequence,call_id,request_id={},0,0,0,0
  local active_request
  local function record(event)
    sequence=sequence+1;event.seq=sequence
    event.at_ns=vim.uv.hrtime()
    if #events==256 then table.remove(events,1);dropped=dropped+1 end
    events[#events+1]=event
  end
  _G.PairReviewObservationTraceSnapshot=function()
    return {events=vim.deepcopy(events),dropped=dropped}
  end
  local system=vim.system
  vim.system=function(cmd,options,callback)
    if type(cmd)~='table' or cmd[2]~='review' or cmd[3]~='readiness' or cmd[4]~='--resolve' then
      return system(cmd,options,callback)
    end
    call_id=call_id+1
    local id,request,mode=call_id,active_request,callback and 'callback' or 'wait'
    local started=vim.uv.hrtime()
    local function event(kind,timeout)
      return {event=kind,call_id=id,request_id=request,mode=mode,timeout=timeout,repo=clip(cmd[5])}
    end
    record(event('start',options and options.timeout))
    local function complete(result,timeout)
      local out=event('completion',timeout)
      out.elapsed_ms=(vim.uv.hrtime()-started)/1e6
      if type(result)=='table' then
        out.code=clip(result.code);out.signal=clip(result.signal)
        local stdout=type(result.stdout)=='string' and result.stdout or ''
        out.stdout_bytes=#stdout;out.stderr=clip(result.stderr,1024)
        local ok,decoded=pcall(vim.json.decode,stdout)
        if ok and type(decoded)=='table' then out.identity=project(decoded)
        else out.decode_error='invalid review resolver JSON '..(ok and type(decoded) or 'syntax') end
      else out.result_type=type(result) end
      record(out)
    end
    local wrapped_callback
    if callback then
      wrapped_callback=function(result)
        complete(result,options and options.timeout)
        return callback(result)
      end
    end
    local process=system(cmd,options,wrapped_callback)
    -- Keep the actual SystemObj, including kill/is_closing and its process owner.
    -- Synchronous calls gain no completion callback; their existing wait is the seam.
    if not callback then
      local wait=process.wait
      process.wait=function(self,...)
        local result=pack(pcall(wait,self,...))
        if not result[1] then
          local failed=event('completion',select(1,...))
          failed.elapsed_ms=(vim.uv.hrtime()-started)/1e6
          failed.error=clip(tostring(result[2]),1024);record(failed)
          error(result[2],0)
        end
        complete(result[2],select(1,...))
        return unpack(result,2,result.n)
      end
    end
    return process
  end
  if _G.PairReviewPane and type(PairReviewPane.restore)=='function' then
    local restore=PairReviewPane.restore
    PairReviewPane.restore=function(raw,...)
      request_id=request_id+1
      local id,previous=request_id,active_request
      local ok,wanted=true,raw
      if type(raw)~='table' then ok,wanted=pcall(vim.json.decode,raw) end
      record({event='request-begin',request_id=id,
        identity=project(ok and type(wanted)=='table' and wanted.identity or nil)})
      active_request=id
      local returned=pack(pcall(restore,raw,...))
      active_request=previous
      local ending={event='request-end',request_id=id}
      if returned[1] then
        local ack=returned[2]
        if type(ack)=='table' then
          ending.ok=clip(ack.ok);ending.same=clip(ack.same);ending.error=clip(ack.error,1024)
          ending.identity=project(ack.identity);ending.context=project(ack.context,true)
        else ending.ack_type=type(ack) end
      else ending.error=clip(tostring(returned[2]),1024) end
      record(ending)
      if not returned[1] then error(returned[2],0) end
      return unpack(returned,2,returned.n)
    end
  end
  if opts.path then
    vim.api.nvim_create_autocmd('VimLeavePre',{once=true,callback=function()
      vim.fn.writefile({vim.json.encode(PairReviewObservationTraceSnapshot())},opts.path)
    end})
  end
end
return M
