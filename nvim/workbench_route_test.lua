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
  ['<M-d>'] = 'PairConfirmDetach',
  ['<M-x>'] = 'PairConfirmQuit',
  ['<M-n>'] = 'PairConfirmRestart',
  ['<C-M-n>'] = 'PairConfirmRestart',
  ['<M-N>'] = 'PairConfirmAgentRestart',
  ['<M-Up>'] = 'PairLayoutBigger',
  ['<M-Down>'] = 'PairLayoutSmaller',
  ['<M-c>'] = 'PairReviewToggle',
}
for key, fn in pairs(expected) do
  assert(route.global_maps[key] == fn, key)
end

assert(vim.deep_equal(route.draft_commands(42, 'PairConfirmRestart'), {
  { 'zellij', 'action', 'write', '--pane-id', '42', '28' },
  { 'zellij', 'action', 'write', '--pane-id', '42', '14' },
  { 'zellij', 'action', 'write-chars', '--pane-id', '42', ':lua PairConfirmRestart()' },
  { 'zellij', 'action', 'write', '--pane-id', '42', '13' },
}))

print('workbench_route_test ok')
