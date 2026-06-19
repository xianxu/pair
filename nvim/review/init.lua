-- nvim/review/init.lua — the review orchestrator (issue #66 M1). Wires the
-- pure core + thin seams into the loop: start a docflow review on a buffer,
-- enable persistent undo, watch for agent handoffs, and on each handoff apply
-- the records undo-ably + commit the agent round (records in the commit body).
--
-- This is the only stateful glue; every decision is delegated to the pure
-- modules (record/reconstruct) and thin seams (docflow/handoff/apply).
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local docflow = dofile(here .. 'docflow.lua')
local handoff = dofile(here .. 'handoff.lua')
local apply   = dofile(here .. 'apply.lua')
local record  = dofile(here .. 'record.lua')
local projection = dofile(here .. 'projection.lua')

local sessions = {} -- buf → { tag, file, stop }

local function save(buf)
  vim.api.nvim_buf_call(buf, function() vim.cmd('silent keepalt write') end)
end

-- Surface docflow failures instead of swallowing them (milestone review I3):
-- a failed round leaves an edited+saved buffer with no commit — never silent.
local function check(result, what)
  if result and result.code and result.code ~= 0 then
    vim.notify(
      string.format('review: docflow %s failed (exit %d): %s', what, result.code, result.stderr or ''),
      vim.log.levels.ERROR)
    return false
  end
  return true
end

-- Apply an agent handoff: undo-able apply → save → commit the agent round with
-- the (enriched) records in the body. Exposed for testing.
function M.on_agent_round(buf, records)
  local base = apply.buf_content(buf) -- pre-round content, for the projection empty snapshot
  projection.set_applying(buf, true) -- suppress the watcher during the round's own apply
  local ok_apply, enriched, dropped = pcall(apply.apply, buf, records)
  if not ok_apply then
    projection.set_applying(buf, false) -- never leave the watcher permanently suppressed
    vim.notify('review: apply failed: ' .. tostring(enriched), vim.log.levels.ERROR)
    return {}, {}
  end
  if #dropped > 0 then
    -- never silent: a partial review must not look complete (milestone review)
    vim.notify(string.format('review: %d proposal(s) did not anchor — dropped', #dropped),
      vim.log.levels.WARN)
  end
  if #enriched > 0 then
    -- record_empty_for FIRST (its guard skips it if `base` is a prior round's
    -- output), then snapshot the placed decorations, then attach the watcher
    -- lazily — only after the snapshots exist, so it never fires before them.
    projection.record_empty_for(buf, base)
    projection.record(buf)
    projection.ensure_watch(buf)
  end
  projection.set_applying(buf, false)
  if #enriched == 0 then return enriched, dropped end
  save(buf)
  local summary = string.format('%d edit(s)', #enriched)
  check(docflow.round('agent', summary, record.embed_in_body(summary, enriched)), 'agent round')
  return enriched, dropped
end

-- Commit the human's incoming edits as a human round.
function M.human_round(buf, summary)
  save(buf)
  return check(docflow.round('human', summary or 'incoming'), 'human round')
end

-- Start a review on a buffer. opts: { buf, file, tag, watch_opts }.
function M.start(opts)
  opts = opts or {}
  local buf = opts.buf or vim.api.nvim_get_current_buf()
  local file = opts.file or vim.api.nvim_buf_get_name(buf)
  local tag = opts.tag or vim.fn.fnamemodify(file, ':t:r')
  if sessions[buf] then M.stop(buf) end -- avoid orphaning a prior poll timer
  vim.bo[buf].undofile = true -- cross-session undo (decision 2)
  check(docflow.start(file), 'start')
  local stop = handoff.watch(tag, function(records)
    M.on_agent_round(buf, records)
  end, opts.watch_opts)
  sessions[buf] = { tag = tag, file = file, stop = stop }
  return sessions[buf]
end

function M.stop(buf)
  local s = sessions[buf]
  if s and s.stop then s.stop() end
  sessions[buf] = nil
  projection.reset(buf) -- remove the watcher so a surviving buffer doesn't double-attach
end

return M
