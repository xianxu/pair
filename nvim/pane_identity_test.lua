local M = dofile('nvim/pane_identity.lua')
local agent = {id=7,is_floating=false,is_plugin=false,terminal_command='/bin/pair wrap codex'}
local terminal = {id=4,is_floating=false,is_plugin=false,terminal_command='/bin/pair term',title='codex'}
local floating = {id=8,is_floating=true,is_plugin=false,terminal_command='/bin/pair wrap codex'}
local plugin = {id=9,is_floating=false,is_plugin=true,terminal_command='/bin/pair wrap codex'}
local nested = {panes={terminal,floating,plugin},tab={panes={agent}}}
assert(#M.collect(nested)==4)
local agents = M.agents(nested)
assert(#agents==1 and agents[1]==agent)
assert(#M.collect(nil)==0 and #M.agents('invalid')==0)
assert(not M.is_agent({id=1,is_floating=false,is_plugin=false,title='pair wrap codex'}))
assert(not M.is_agent({id=1,is_floating=false,is_plugin=false,terminal_command=false}))
assert(not M.is_agent({id=1,is_plugin=false,terminal_command='pair wrap codex'}))
local second={id=10,is_floating=false,is_plugin=false,terminal_command='pair wrap claude'}
assert(#M.agents({agent,second})==2, 'caller must see ambiguity')
assert(#M.collect({outer=agent,ignored={id=11}})==1)
print('pane_identity_test: passed')
