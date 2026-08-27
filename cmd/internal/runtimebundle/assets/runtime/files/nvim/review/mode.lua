-- nvim/review/mode.lua — pair-side review mode UI metadata.
--
-- Pair only needs stable IDs and menu order. The semantics of Generate, Edit,
-- and Proofread are deployment prompt content owned by ariadne's xx-fix skill:
-- /Users/xianxu/workspace/ariadne/construct/local/fix/SKILL.md
local M = {}

local MODES = {
  { name = 'generate', order = 1 },
  { name = 'edit', order = 2 },
  { name = 'proofread', order = 3 },
}

local BY_NAME = {}
for _, mode in ipairs(MODES) do
  BY_NAME[mode.name] = mode
end

local function copy_mode(mode)
  return { name = mode.name, order = mode.order }
end

--- Load one built-in review mode by name. The `dir` arg is accepted for
--- compatibility with the old markdown-backed call sites.
function M.load(_, name)
  local mode = BY_NAME[name]
  if not mode then return nil, "mode: no built-in mode for '" .. tostring(name) .. "'" end
  return copy_mode(mode)
end

--- List built-in review modes in menu order. The `dir` arg is accepted for
--- compatibility with the old markdown-backed call sites.
function M.list(_)
  local out = {}
  for i, mode in ipairs(MODES) do
    out[i] = copy_mode(mode)
  end
  return out
end

return M
