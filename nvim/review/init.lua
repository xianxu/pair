-- nvim/review/init.lua — the review orchestrator (issue #66). Wires the pure core
-- + thin seams into the loop: enable persistent undo, watch for agent handoffs,
-- and on each handoff apply the records undo-ably, save, and signal the agent to
-- commit (the nvim writes NO git — invariant #1; the AGENT owns branch + rounds).
--
-- M4a: the round commit moved to the agent. on_agent_round (the apply authority)
-- writes what LANDED to the landed-artifact (seam #2b) and pokes the agent; the
-- agent commits the round verbatim from it. So the body's single encoder
-- (record.embed_in_body) stays here, but the git write is the agent's.
--
-- This is the only stateful glue; every decision is delegated to the pure
-- modules (record/reconstruct/poke_bodies) and thin seams (handoff/apply/poke).
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local handoff = dofile(here .. 'handoff.lua')
local apply   = dofile(here .. 'apply.lua')
local record  = dofile(here .. 'record.lua')
local reconstruct = dofile(here .. 'reconstruct.lua') -- resume repaint (reconstruct-on-open)
local projection = dofile(here .. 'projection.lua')
local poke_bodies = dofile(here .. 'poke_bodies.lua')

-- The agent-poke seam, injectable: on_agent_round is driven directly by the
-- headless tests, which swap `M.poke` for a recorder so they never shell zellij.
M.poke = dofile(here .. '../pair_poke.lua')

local sessions = {} -- buf → { tag, file, stop }

local function save(buf)
  vim.api.nvim_buf_call(buf, function() vim.cmd('silent keepalt write') end)
end

-- Apply an agent handoff: undo-able apply → save → write the landed-artifact
-- (what landed) → poke the agent to commit the round. Exposed for testing.
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
  -- The nvim writes NO git. As the apply authority it records WHAT LANDED to the
  -- landed-artifact (seam #2b) — the body via the one encoder (record.embed_in_body),
  -- drops filtered, new_occurrence computed — then pokes the agent to commit the
  -- round verbatim from it (invariants #1 + #3).
  local sess = sessions[buf] or {}
  local file = sess.file or vim.api.nvim_buf_get_name(buf)
  local tag = sess.tag or vim.fn.fnamemodify(file, ':t:r')
  local summary = string.format('%d edit(s)', #enriched)
  handoff.write_landed(tag, {
    summary = summary,
    body = record.embed_in_body(summary, enriched),
    applied = #enriched,
    dropped = #dropped,
  })
  M.poke.send(poke_bodies.agent_applied(#enriched, #dropped, file))
  return enriched, dropped
end

-- The human finished their turn: save the incoming edits. The nvim writes no git;
-- the AGENT commits the human round (invariant #1). The commit-request poke is
-- issued by nvim/review.lua's finish_human_turn (the UI layer where the trigger lives).
function M.human_round(buf, summary)
  save(buf)
  return true
end

-- Reconstruct-on-open (M4a' resume): repaint decorations from the latest agent-round
-- commit body. Text already survives across sessions via `undofile`; the styling
-- (highlights + diagnosis) is rebuilt from the records-in-commit (the M0 decision).
-- No-op when there's no agent round yet (a fresh review) or not in a git repo.
-- Exposed for the resume test.
function M.reconstruct_on_open(buf, file)
  local dir = vim.fn.fnamemodify(file, ':h')
  -- the latest agent round's body (subject `review(<slug>): agent r<N> — …`);
  -- -F so the paren-bearing marker is a fixed string, not a regex.
  local body = vim.fn.system({ 'git', '-C', dir, 'log', '-1', '--pretty=%b', '-F', '--grep=): agent r' })
  if vim.v.shell_error ~= 0 or not body or body == '' then return false end
  local ok, records = pcall(record.extract_from_body, body)
  if not ok or type(records) ~= 'table' or #records == 0 then return false end
  local content = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
  local dec = reconstruct.decorate(records, content, 'new')
  apply.apply_snapshot(buf, { hl = dec.highlights, diags = dec.diagnostics })
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
  -- invariant #1). The nvim only opens the pane + watches for handoffs.
  local stop = handoff.watch(tag, function(records)
    M.on_agent_round(buf, records)
  end, opts.watch_opts)
  sessions[buf] = { tag = tag, file = file, stop = stop }
  pcall(M.reconstruct_on_open, buf, file) -- resume repaint (no-op on a fresh review)
  return sessions[buf]
end

function M.stop(buf)
  local s = sessions[buf]
  if s and s.stop then s.stop() end
  sessions[buf] = nil
  projection.reset(buf) -- remove the watcher so a surviving buffer doesn't double-attach
end

return M
