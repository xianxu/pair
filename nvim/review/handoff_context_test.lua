local dir=vim.fn.tempname(); vim.fn.mkdir(dir,'p')
vim.env.PAIR_REVIEW_HANDOFF_PATH=dir..'/handoff.json'
local H=dofile('nvim/review/handoff.lua')
local current='a'; local received
local stop=H.watch('test',function(records,context) received={records,context}; return true end,{interval=10,
  admit=function(payload) return payload.context and payload.context.activation==current end})
local payload={context={activation='b'},records={{old='x',new='y',occurrence=1}}}
vim.fn.writefile({vim.json.encode(payload)},vim.env.PAIR_REVIEW_HANDOFF_PATH)
vim.wait(80,function() return received~=nil end,10)
assert(received==nil,'wrong activation applied')
assert(vim.fn.filereadable(vim.env.PAIR_REVIEW_HANDOFF_PATH)==1,'rejected payload consumed')
current='b'; assert(vim.wait(200,function() return received~=nil end,10))
assert(received[1][1].new=='y' and received[2].activation=='b')
assert(vim.fn.filereadable(vim.env.PAIR_REVIEW_HANDOFF_PATH)==0)
for _,raw in ipairs({'false','42','"wrong"','{"context":{"activation":"b"}}','{"context":{"activation":"b"},"records":[false]}'}) do
  received=nil
  vim.fn.writefile({raw},vim.env.PAIR_REVIEW_HANDOFF_PATH)
  vim.wait(50,function() return received~=nil end,10)
  assert(not received and vim.fn.filereadable(vim.env.PAIR_REVIEW_HANDOFF_PATH)==1,'malformed payload consumed')
end
stop(); vim.fn.delete(dir,'rf'); print('handoff_context_test ok')
