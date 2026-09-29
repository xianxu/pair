local R=dofile('nvim/review/init.lua')
local buf=vim.api.nvim_create_buf(true,false)
vim.api.nvim_buf_set_lines(buf,0,-1,false,{'old'})
R.authorize=function() return false end
local out=R.apply_round(buf,{{old='old',new='wrong',occurrence=1}})
assert(out==nil and vim.api.nvim_buf_get_lines(buf,0,-1,false)[1]=='old','rejected round changed buffer')
assert(R.human_round(buf,'no')==false,'rejected human round saved')
print('round_authority_test ok')
