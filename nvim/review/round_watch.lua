-- nvim/review/round_watch.lua — the pane's side of `pair review apply` (pair#661 M2).
-- The agent applies through the shared engine, which writes the reviewed file and
-- the round artifact ($PAIR_REVIEW_ROUND_PATH: {file, round, body}). The pane only
-- notices a new round, reloads the file and decorates its spans; it never writes
-- the file for an agent round. decide/decorations are pure; watch is the poll.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local reconstruct = dofile(here .. 'reconstruct.lua')

function M.path() return vim.env.PAIR_REVIEW_ROUND_PATH end

local function canonical(p) return vim.fn.resolve(vim.fn.fnamemodify(p, ':p')) end

-- A round's identity is its result hash and its time (pair#690): an empty
-- round leaves the hash where the round before it did, and only its time
-- tells the two apart. nil for an artifact that is not a round.
function M.key(artifact)
  local round = type(artifact) == 'table' and artifact.round
  if type(round) ~= 'table' or type(round.result_hash) ~= 'string' or round.result_hash == '' then return nil end
  return round.result_hash .. '@' .. (type(round.at) == 'string' and round.at or '')
end

-- The empty round: the agent replied with no edits, so there is nothing to
-- reload or decorate, only a turn to end (pair#690).
function M.is_empty(round)
  return type(round) == 'table' and round.base_hash == round.result_hash
    and (type(round.spans) ~= 'table' or #round.spans == 0)
end

-- A round is the pane's to act on when it names the pane's file, is one not
-- seen yet, and the file on disk still is its result (current_hash); a file
-- that moved on since makes the round stale, and the pane leaves it.
function M.decide(artifact, file, seen, current_hash)
  local key = M.key(artifact)
  if not key or key == seen then return false end
  if type(artifact.file) ~= 'string' or canonical(artifact.file) ~= canonical(file) then return false end
  return current_hash == nil or current_hash == artifact.round.result_hash
end

-- hash is the engine's whole-file hash: the first 16 hex of sha256 of the bytes.
function M.hash(path)
  local f = io.open(path, 'rb')
  if not f then return nil end
  local raw = f:read('*a'); f:close()
  return vim.fn.sha256(raw):sub(1, 16)
end

-- The round's spans as the apply.apply_snapshot shape: a DiffChange highlight per
-- span (none for a deletion or a marker proposal) and an INFO diagnostic with the
-- span's reason, or "changed by the agent" when it has none.
function M.decorations(round)
  local hl, diags = {}, {}
  for _, s in ipairs(round.spans or {}) do
    local presentation = reconstruct.presentation(s, round.presentation)
    if presentation ~= 'direct' then
    local h = { line = s.start[1], col = s.start[2], end_line = s['end'][1], end_col = s['end'][2] }
    if presentation == 'highlight' and (s.new or '') ~= '' and not reconstruct.is_marker_proposal(s.new) then hl[#hl + 1] = h end
    local message = (s.explain and s.explain ~= '') and s.explain or 'changed by the agent'
    if (s.new or '') == '' then message = message .. ' (deleted: ' .. (s.old or '') .. ')' end
    diags[#diags + 1] = { lnum = h.line, col = h.col, end_lnum = h.end_line, end_col = h.end_col, message = message }
    end
  end
  return { hl = hl, diags = diags }
end

function M.read(path)
  local f = path and io.open(path, 'rb')
  if not f then return nil end
  local raw = f:read('*a'); f:close()
  local ok, artifact = pcall(vim.json.decode, raw)
  return ok and artifact or nil
end

-- Poll the round artifact; cb(artifact) for each new round of `file`. The round
-- present at start is already on disk (resume repaints it from the commit), so it
-- is marked seen, not replayed. Returns stop().
function M.watch(file, cb, opts)
  opts = opts or {}
  local path = opts.path or M.path()
  if not path or path == '' then return function() end end
  local current = M.read(path)
  local seen = M.key(current)
  local timer = vim.uv.new_timer()
  timer:start(opts.interval or 100, opts.interval or 100, vim.schedule_wrap(function()
    local artifact = M.read(path)
    -- A round the callback declines (a modified buffer) stays unseen, so it
    -- is delivered again once the pane can take it.
    -- Hash the file only for a round that is otherwise the pane's.
    if M.decide(artifact, file, seen) and M.hash(file) == artifact.round.result_hash
        and cb(artifact) ~= false then
      seen = M.key(artifact)
    end
  end))
  return function() if not timer:is_closing() then timer:stop(); timer:close() end end
end

return M
