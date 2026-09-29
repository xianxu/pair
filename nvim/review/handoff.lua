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

-- Poll for the handoff; on appearance decode → unlink → cb(records).
-- Returns a stop() function. opts.interval ms (default 100).
function M.watch(tag, cb, opts)
  opts = opts or {}
  local p = M.path(tag)
  if not p or p == '' then return function() end end
  local timer = vim.uv.new_timer()
  local rejected
  timer:start(0, opts.interval or 100, vim.schedule_wrap(function()
    if not vim.uv.fs_stat(p) then return end
    local fh = io.open(p, 'r')
    if not fh then return end
    local data = fh:read('*a'); fh:close()
    local ok, payload = pcall(record.decode, data)
    if not ok or not valid_payload(payload) then
      if rejected~=data then vim.notify('review: handoff decode failed — payload preserved', vim.log.levels.WARN) end
      rejected=data
      return
    end
    if opts.admit then
      local allowed, reason = opts.admit(ok and payload or nil)
      if not allowed then
        if rejected ~= data then
          vim.notify('review: '..(reason or 'handoff context mismatch; payload preserved'), vim.log.levels.WARN)
          rejected=data
        end
        return
      end
    end
    if ok and payload then
      local recs = payload.records or payload
      os.remove(p)
      rejected=nil
      cb(recs, payload.context)
    else
      if rejected~=data then vim.notify('review: handoff decode failed — payload preserved', vim.log.levels.WARN) end
      rejected=data
    end
  end))
  return function()
    if timer and not timer:is_closing() then timer:stop(); timer:close() end
  end
end

return M
