-- nvim/review/handoff.lua — the ephemeral records handoff file (issue #66 M1).
-- The agent writes it (atomic temp+rename); nvim watches, consumes, unlinks.
-- Its appearance IS the "round ready" signal — both data and signal in one file.
--
-- Detection is a TIMER POLL, not fs_event: macOS FSEvents is flaky/laggy and
-- init.lua's scrollback watcher already polls for that reason. The atomic write
-- guarantees the poll never reads a half-written file.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local record = dofile(here .. 'record.lua')
local function valid_payload(payload)
  if type(payload)~='table' then return false end
  local records=payload.context~=nil and payload.records or payload
  if type(records)~='table' then return false end
  for k,r in pairs(records) do
    if type(k)~='number' or k<1 or k>#records or k%1~=0 or type(r)~='table'
      or type(r.old)~='string' or type(r.new)~='string'
      or type(r.occurrence)~='number' or r.occurrence<1 or r.occurrence%1~=0 then return false end
  end
  return true
end

function M.path(tag)
  return vim.env.PAIR_REVIEW_HANDOFF_PATH
end

-- The landed-artifact (seam #2b, nvim → agent) — the REVERSE channel of the
-- handoff, so it lives in the same agent↔nvim data dir. on_agent_round writes
-- what actually landed ({summary, body=embed_in_body(enriched), applied, dropped});
-- the agent reads it and commits the agent round verbatim (it owns git; the nvim,
-- the apply authority, owns the body content — invariants #1 + #3).
function M.landed_path(tag)
  return vim.env.PAIR_REVIEW_LANDED_PATH
end

function M.write_landed(tag, landed)
  local p = M.landed_path(tag)
  vim.fn.mkdir(vim.fn.fnamemodify(p, ':h'), 'p')
  local tmp = p .. '.tmp'
  local f = assert(io.open(tmp, 'w'))
  f:write(vim.json.encode(landed))
  f:close()
  assert(os.rename(tmp, p))
  return p
end

-- Write records atomically (temp + rename). Used by the agent / fake / tests.
function M.write(tag, records)
  local p = M.path(tag)
  vim.fn.mkdir(vim.fn.fnamemodify(p, ':h'), 'p')
  local tmp = p .. '.tmp'
  local f = assert(io.open(tmp, 'w'))
  f:write(record.encode(records))
  f:close()
  assert(os.rename(tmp, p))
  return p
end

local artifact=dofile(here..'artifact.lua')

-- Poll, decode, authorize, then offer to cb(records,context). Only an explicit
-- true accepts ownership (applied or durably deferred); refusal/error preserves
-- the payload. Returns stop(). opts.interval ms defaults to 100.
function M.watch(tag, cb, opts)
  opts = opts or {}
  local p = M.path(tag)
  if not p or p == '' then return function() end end
  local timer = vim.uv.new_timer()
  local rejected,settled,stopped,running
  local function reject(data,reason)
    if rejected~=data then vim.notify('review: '..reason..' — payload preserved',vim.log.levels.WARN) end
    rejected=data
  end
  local function consume(data,generation)
    local removed,err=artifact.consume(p,data,generation)
    if removed==nil then reject(data,'handoff accepted but could not remove it: '..tostring(err)); return end
    rejected=nil
  end
  local function poll()
    if stopped then return end
    local data,generation=artifact.read(p)
    if not data then return end
    if settled and data==settled.data and artifact.same(generation,settled.generation) then
      if settled.accepted then consume(data,generation) end
      return -- uncertain or already accepted: never replay this generation
    end
    settled=nil
    local ok, payload = pcall(record.decode, data)
    if not ok or not valid_payload(payload) then reject(data,'handoff decode failed'); return end
    if opts.admit then
      local admitted,allowed,reason=pcall(opts.admit,payload)
      if not admitted or allowed~=true then
        reject(data,admitted and (reason or 'handoff context mismatch') or tostring(allowed)); return
      end
    end
    local completed,accepted,uncertain=pcall(cb,payload.records or payload,payload.context)
    if not completed or uncertain then
      settled={data=data,generation=generation,accepted=false}
      reject(data,'handoff outcome uncertain; inspect the buffer before reissuing'..(not completed and ': '..tostring(accepted) or ''))
      return
    end
    if accepted~=true then reject(data,'handoff was not accepted'); return end
    settled={data=data,generation=generation,accepted=true}
    consume(data,generation)
  end
  timer:start(0, opts.interval or 100, vim.schedule_wrap(function()
    if stopped or running then return end
    running=true
    local ok,err=pcall(poll)
    running=false
    if not ok then vim.notify('review: handoff observation failed — payload preserved: '..tostring(err),vim.log.levels.WARN) end
  end))
  return function()
    stopped=true
    if timer and not timer:is_closing() then timer:stop(); timer:close() end
  end
end

return M
