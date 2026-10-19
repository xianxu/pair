local R=dofile('nvim/review/init.lua')
local buf=vim.api.nvim_create_buf(true,false)
vim.api.nvim_buf_set_lines(buf,0,-1,false,{'old'})
R.authorize=function() return false end
assert(R.human_round(buf,'no')==false,'rejected human round saved')
print('round_authority_test ok')
