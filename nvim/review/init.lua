-- nvim/review/init.lua — the review orchestrator (issue #66). Wires the pure core
-- + thin seams into the loop: enable persistent undo, watch for rounds the agent
-- applied through `pair review apply` (#661 M2), and on each one reload the file
-- and decorate the round's spans. The nvim writes NO git and never writes the
-- reviewed file for an agent round (pair#664): the shared engine writes it and
-- the AGENT commits the round from the apply reply.
--
-- This is the only stateful glue; every decision is delegated to the pure
-- modules (record/reconstruct/projection) and thin seams (round_watch/apply/oplock).
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local apply   = dofile(here .. 'apply.lua')
local define = dofile(here .. 'define.lua')
local record  = dofile(here .. 'record.lua')
local reconstruct = dofile(here .. 'reconstruct.lua') -- resume repaint (reconstruct-on-open)
local projection = dofile(here .. 'projection.lua')
local round_watch = dofile(here .. 'round_watch.lua') -- pair review apply rounds (#661 M2)
local oplock = dofile(here .. 'oplock.lua')

M.buf_content = apply.buf_content -- one join, shared with review.lua

local sessions = {} -- buf → { tag, file, stop }

local function save(buf)
  vim.api.nvim_buf_call(buf, function() vim.cmd('silent keepalt write') end)
end

-- An agent round applied through `pair review apply` (#661 M2): the shared
-- engine already wrote the file, so the pane only reloads it and decorates the
-- round's spans; it never writes the file for an agent round. A modified buffer
-- means the operator is mid-turn (their lock should have refused the apply), so
-- the pane keeps their text and says so. Exposed for testing.
function M.on_applied_round(buf, artifact)
  if vim.bo[buf].modified then
    local sess = sessions[buf] or {}
    if sess.warned ~= artifact.round.result_hash then -- once per declined round
      sess.warned = artifact.round.result_hash
      vim.notify('review: the agent applied a round while you have unsaved edits; your buffer was kept', vim.log.levels.WARN)
    end
    return false
  end
  local base = apply.buf_content(buf)
  projection.set_applying(buf, true)
  local ok, err = pcall(vim.api.nvim_buf_call, buf, function() vim.cmd('silent edit!') end)
  if ok then
    apply.apply_snapshot(buf, round_watch.decorations(artifact.round))
    projection.record_empty_for(buf, base)
    projection.record(buf)
    projection.ensure_watch(buf)
  end
  projection.set_applying(buf, false)
  if not ok then
    vim.notify('review: reloading the applied round failed: ' .. tostring(err), vim.log.levels.ERROR)
    return false
  end
  pcall(M.rehydrate_definitions, buf)
  if M.after_applied_round then pcall(M.after_applied_round, buf) end
  return true
end

-- The human finished their turn: save the incoming edits. The nvim writes no git;
-- the AGENT commits the human round (invariant #1). The commit-request poke is
-- issued by nvim/review.lua's finish_human_turn (the UI layer where the trigger lives).
function M.human_round(buf, summary, context)
  if M.authorize and not M.authorize(buf, context) then return false end
  save(buf)
  return true
end

function M.clear_decorations(buf)
  apply.clear_all(buf)
  projection.reset(buf)
end

function M.projected_mutation(buf, base_content, mutate, preserve_base)
  if preserve_base then projection.record(buf) end
  projection.set_applying(buf, true)
  local ok, err = pcall(mutate)
  if not ok then
    projection.set_applying(buf, false)
    error(err)
  end
  projection.record_empty_for(buf, base_content)
  projection.record(buf)
  projection.ensure_watch(buf)
  projection.set_applying(buf, false)
end

function M.rehydrate_definitions(buf)
  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  apply.place_definitions(buf, define.footnote_diagnostics(lines))
end

function M.clear_decoration_at_line(buf, row)
  local cleared = apply.clear_at_line(buf, row)
  if cleared then
    projection.record(buf)
  end
  return cleared
end

-- Reconstruct-on-open (M4a' resume): repaint decorations from the latest agent-round
-- commit body. Text already survives across sessions via `undofile`; the styling
-- (highlights + diagnosis) is rebuilt from the records-in-commit (the M0 decision).
-- No-op when there's no agent round yet (a fresh review) or not in a git repo.
-- Exposed for the resume test.
function M.reconstruct_on_open(buf, file, resolved)
  local file_abs=vim.uv.fs_realpath(file) or vim.fn.fnamemodify(file,':p')
  if not resolved then
    local reader=dofile(here..'identity.lua')
    resolved=reader.resolve(vim.fn.fnamemodify(file_abs,':h'))
    -- Explicit manual opening may precede the first file-changing round.
    if resolved.status=='missing' and resolved.repo and file_abs:sub(1,#resolved.repo+1)==resolved.repo..'/' then
      resolved=reader.resolve(resolved.repo,{file=file_abs:sub(#resolved.repo+2),head=resolved.head})
    end
  end
  if resolved.status~='resolved' or resolved.repo..'/'..resolved.file~=file_abs then return false end
  local body=resolved.latest_agent_body
  if not body or body=='' then return false end
  local ok, records = pcall(record.extract_from_body, body)
  if not ok or type(records) ~= 'table' or #records == 0 then return false end
  local content = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
  local dec = reconstruct.decorate(records, content, 'new')
  apply.apply_snapshot(buf, { hl = dec.highlights, diags = dec.diagnostics })
  M.rehydrate_definitions(buf)
  return #dec.highlights > 0
end

-- Start a review on a buffer. opts: { buf, file, tag, watch_opts }.
function M.start(opts)
  opts = opts or {}
  local buf = opts.buf or vim.api.nvim_get_current_buf()
  local file = opts.file or vim.api.nvim_buf_get_name(buf)
  local tag = opts.tag or vim.fn.fnamemodify(file, ':t:r')
  if sessions[buf] then M.stop(buf) end -- avoid orphaning a prior poll timer
  vim.bo[buf].undofile = true -- cross-session undo (decision 2)
  -- No `docflow.start` — the agent owns the `review/<slug>` branch too (seam #4,
  -- invariant #1). The nvim only opens the pane + watches for applied rounds.
  pcall(oplock.reclaim, file) -- a lock a crashed pane left would block every apply
  local stop_rounds = round_watch.watch(file, function(artifact) return M.on_applied_round(buf, artifact) end,
    opts.round_watch_opts)
  sessions[buf] = { tag = tag, file = file, stop = stop_rounds }
  apply.clear_all(buf)
  pcall(M.reconstruct_on_open, buf, file, opts.identity) -- scoped resume repaint
  pcall(M.rehydrate_definitions, buf)
  return sessions[buf]
end

function M.stop(buf)
  local s = sessions[buf]
  if s and s.stop then s.stop() end
  sessions[buf] = nil
  projection.reset(buf) -- remove the watcher so a surviving buffer doesn't double-attach
end

return M
