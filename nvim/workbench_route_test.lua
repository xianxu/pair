local here = debug.getinfo(1, 'S').source:sub(2):match('(.*/)') or './'
local route = dofile(here .. 'workbench_route.lua')

local panes = {
  { id = 7, title = 'codex', terminal_command = 'pair wrap codex' },
  {
    id = 42,
    title = 'draft',
    terminal_command = 'nvim -u /opt/pair/nvim/init.lua /tmp/draft.md',
  },
  {
    id = 81,
    title = 'review',
    terminal_command = 'nvim -u /opt/pair/nvim/review.lua /tmp/review.md',
  },
}

assert(route.find_draft_pane(panes) == '42')
assert(route.find_draft_pane({ panes[1], panes[3] }) == nil)

local expected = {
  ['<M-d>'] = { fn = 'PairConfirmDetach', focus = true },
  ['<M-x>'] = { fn = 'PairConfirmQuit', focus = true },
  ['<M-n>'] = { fn = 'PairConfirmRestart', focus = true },
  ['<C-M-n>'] = { fn = 'PairConfirmRestart', focus = true },
  ['<M-N>'] = { fn = 'PairConfirmAgentRestart', focus = true },
  ['<M-Up>'] = { fn = 'PairLayoutBigger', focus = false },
  ['<M-Down>'] = { fn = 'PairLayoutSmaller', focus = false },
  ['<M-c>'] = { fn = 'PairReviewToggle', focus = false },
}
for key, binding in pairs(expected) do
  assert(vim.deep_equal(route.global_maps[key], binding), key)
end

assert(vim.deep_equal(route.draft_commands(42, 'PairConfirmRestart', true), {
  { 'zellij', 'action', 'focus-pane-id', '42' },
  { 'zellij', 'action', 'write', '--pane-id', '42', '28' },
  { 'zellij', 'action', 'write', '--pane-id', '42', '14' },
  { 'zellij', 'action', 'write-chars', '--pane-id', '42', ':lua PairConfirmRestart()' },
  { 'zellij', 'action', 'write', '--pane-id', '42', '13' },
}))

assert(vim.deep_equal(route.draft_commands(42, 'PairLayoutBigger', false), {
  { 'zellij', 'action', 'write', '--pane-id', '42', '28' },
  { 'zellij', 'action', 'write', '--pane-id', '42', '14' },
  { 'zellij', 'action', 'write-chars', '--pane-id', '42', ':lua PairLayoutBigger()' },
  { 'zellij', 'action', 'write', '--pane-id', '42', '13' },
}))

local record = vim.json.encode({
  session = 'pair-work',
  pane_id = '42',
  pid = 9001,
})
assert(route.validate_cached_draft(record, 'pair-work', function(pid)
  return pid == 9001
end) == '42')
assert(route.validate_cached_draft(record, 'pair-other', function() return true end) == nil)
assert(route.validate_cached_draft(record, 'pair-work', function() return false end) == nil)
assert(route.validate_cached_draft('bad json', 'pair-work', function() return true end) == nil)

print('workbench_route_test ok')
