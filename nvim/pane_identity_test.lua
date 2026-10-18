local M = dofile('nvim/pane_identity.lua')
local agent = {id=7,is_floating=false,is_plugin=false,terminal_command='/bin/pair wrap codex',tab_id=2,tab_position=1}
local terminal = {id=4,is_floating=false,is_plugin=false,terminal_command='/bin/pair term',title='codex',tab_id=2,tab_position=1}
local floating = {id=8,is_floating=true,is_plugin=false,terminal_command='/bin/pair wrap codex',tab_id=2,tab_position=1}
local plugin = {id=9,is_floating=false,is_plugin=true,terminal_command='/bin/pair wrap codex',tab_id=2,tab_position=1}
local source = {id=9,is_plugin=false,is_floating=true,tab_id=2,tab_position=1,terminal_command='nvim review.lua'}
assert(M.select_in_tab({terminal,floating,plugin,agent,source},'9',M.is_agent)==agent)
assert(not M.is_agent({id=1,is_floating=false,is_plugin=false,title='pair wrap codex'}))
assert(not M.is_agent({id=1,is_floating=false,is_plugin=false,terminal_command=false}))
assert(not M.is_agent({id=1,is_plugin=false,terminal_command='pair wrap codex'}))
local second={id=10,is_floating=false,is_plugin=false,terminal_command='pair wrap claude',tab_id=2,tab_position=1}
assert(M.select_in_tab({source,agent,second},'9',M.is_agent)==nil,'ambiguous agent role')
local unrelated={id=11,is_floating=false,is_plugin=false,terminal_command='pair wrap claude',tab_id=8,tab_position=3,is_focused=true}
assert(M.select_in_tab({source,unrelated,agent},'9',M.is_agent)==agent,'unrelated tab focus must not select the target')
local conflict={id=10,is_plugin=false,is_floating=false,tab_id=2,tab_position=3,terminal_command='pair term'}
assert(M.select_in_tab({source,agent,conflict},'9',M.is_agent)==nil,'conflicting position in invoking tab ignored')
assert(M.select_in_tab({{id=9,is_plugin=false},{id=7,is_plugin=false,is_floating=false,terminal_command='pair wrap codex'}},'9',M.is_agent)==nil,'missing source tab granted authority')
assert(M.select_in_tab({source,agent},nil,M.is_agent)==nil)
assert(M.select_in_tab({source,agent},'plugin_9',M.is_agent)==nil)
assert(M.select_in_tab({source,agent},'9007199254740992',M.is_agent)==nil)
assert(M.select_in_tab({plugin,agent},'9',M.is_agent)==nil,'plugin identity cannot replace terminal source')
assert(M.select_in_tab({source,agent,source},'9',M.is_agent)==nil,'duplicate source')
assert(M.select_in_tab({source,terminal},'9',M.is_agent)==nil)
print('pane_identity_test: passed')
