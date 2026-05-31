-- Headless tests for nvim/slug.lua — run via `nvim -l nvim/slug_test.lua`
-- (or `make test-lua`). Pure Lua; no vim API needed. Exits non-zero on
-- failure so the make target fails loudly.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'slug.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

-- is_structured
eq(M.is_structured('=== a | b ==='), true, 'structured basic')
eq(M.is_structured('=== #42 winbar | doing y ==='), true, 'structured with hash')
eq(M.is_structured('=== freeform note ==='), false, 'freeform no pipe')
eq(M.is_structured('plain prompt text'), false, 'plain text')
eq(M.is_structured(''), false, 'empty not structured')

local PROPOSED = '=== #27 auto | new focus ==='

-- empty line 1 → apply
do
  local a, p = M.decide('', PROPOSED)
  eq(a, 'apply', 'empty→apply'); eq(p, PROPOSED, 'empty→prev=proposed')
end

-- structured slug (machine-written or user-edited) → apply. Soft policy: the
-- proposer only proposes when the model decided to change it; preservation of
-- a user edit happens upstream (no proposal written), not by holding here.
do
  local a, p = M.decide('=== #27 auto | old focus ===', PROPOSED)
  eq(a, 'apply', 'structured→apply'); eq(p, PROPOSED, 'structured→prev=proposed')
end
do
  local a = M.decide('=== #27 auto | my own words ===', PROPOSED)
  eq(a, 'apply', 'user-edited structured→apply (soft)')
end

-- freeform "=== … ===" without pipe → hold (full manual override)
do
  local fre = '=== my own note ==='
  local a, p = M.decide(fre, PROPOSED)
  eq(a, 'hold', 'freeform→hold'); eq(p, fre, 'freeform→prev=freeform')
end

-- user prompt text on line 1 → hold (never clobber it)
do
  local a, p = M.decide('please fix the winbar bug', PROPOSED)
  eq(a, 'hold', 'prompt→hold'); eq(p, 'please fix the winbar bug', 'prompt→prev=text')
end

-- ── M.apply: buffer-safety matrix (needs vim.api; provided by `nvim -l`) ──
if vim and vim.api then
  local function mkbuf(lines)
    local b = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_lines(b, 0, -1, false, lines)
    return b
  end
  local function lines(b)
    return vim.api.nvim_buf_get_lines(b, 0, -1, false)
  end

  -- apply over a structured slug + prompt: line 1 replaced, lines 2+ untouched
  do
    local b = mkbuf({ '=== #27 auto | old ===', 'my prompt line', 'second line' })
    local a = M.apply(b, PROPOSED)
    local L = lines(b)
    eq(a, 'apply', 'apply over prompt: action')
    eq(L[1], PROPOSED, 'apply: line1 replaced')
    eq(L[2], 'my prompt line', 'apply: line2 intact')
    eq(L[3], 'second line', 'apply: line3 intact')
    eq(#L, 3, 'apply: no spurious lines')
  end

  -- cursor intact when line 1 is replaced
  do
    local b = mkbuf({ '=== #27 auto | old ===', 'prompt', 'more here' })
    vim.api.nvim_set_current_buf(b)
    vim.api.nvim_win_set_cursor(0, { 3, 4 })
    M.apply(b, PROPOSED)
    local c = vim.api.nvim_win_get_cursor(0)
    eq(c[1], 3, 'cursor row intact'); eq(c[2], 4, 'cursor col intact')
    eq(lines(b)[2], 'prompt', 'cursor case: line2 intact')
  end

  -- empty buffer → slug on line 1 + blank prompt line below
  do
    local b = mkbuf({ '' })
    M.apply(b, PROPOSED)
    local L = lines(b)
    eq(L[1], PROPOSED, 'empty: slug on line1')
    eq(L[2], '', 'empty: blank prompt line added')
    eq(#L, 2, 'empty: exactly two lines')
  end

  -- freeform no-pipe line 1 → hold: buffer must NOT change (manual override)
  do
    local b = mkbuf({ '=== my own note ===', 'prompt' })
    local a, p = M.apply(b, PROPOSED)
    eq(a, 'hold', 'freeform: hold')
    eq(lines(b)[1], '=== my own note ===', 'freeform: line1 unchanged')
    eq(p, '=== my own note ===', 'freeform: prev = freeform line')
  end

  -- user prompt text on line 1 → hold: no slug inserted/overwritten
  do
    local b = mkbuf({ 'please fix the bug', 'detail' })
    local a = M.apply(b, PROPOSED)
    eq(a, 'hold', 'prompt-on-line1: hold')
    eq(lines(b)[1], 'please fix the bug', 'prompt-on-line1: unchanged')
    eq(#lines(b), 2, 'prompt-on-line1: no inserted line')
  end
else
  io.stderr:write('WARN: vim.api unavailable — M.apply buffer tests skipped\n')
end

if fails > 0 then
  io.stderr:write(fails .. ' failure(s)\n')
  os.exit(1)
end
print('nvim/slug.lua: all tests passed')
