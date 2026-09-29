local M = dofile('nvim/review/restore.lua')
local a = { repo = '/r', branch = 'review/a', file = 'a.md', head = '1', status = 'resolved' }
local b = { repo = '/r', branch = 'review/b', file = 'b.md', head = '2', status = 'resolved' }
local ctx = { repo = '/r', branch = 'review/a', file = 'a.md', activation = 'nonce' }
assert(M.validate_context(ctx, ctx))
for _, key in ipairs({ 'repo', 'branch', 'file', 'activation' }) do
  local wrong = vim.deepcopy(ctx); wrong[key] = 'wrong'
  assert(not M.validate_context(wrong, ctx), key)
  wrong[key] = nil; assert(not M.validate_context(wrong, ctx), key .. ' missing')
end
assert(not M.validate_context(nil, ctx))
assert(M.validate_context(nil, ctx, true))
local s, effect = M.transition({ status = 'idle' }, { kind = 'request', identity = a })
assert(s.status == 'switching' and effect == 'activate')
s = M.transition(s, { kind = 'activated', context = ctx })
assert(s.status == 'active' and s.context.activation == 'nonce')
local same, action = M.transition(s, { kind = 'request', identity = a, pending = 'agent working' })
assert(action == 'toggle' and same.context.activation == 'nonce')
for _, pending in ipairs({ 'unsaved edits', 'agent working', 'deferred round', 'definition', 'uncommitted round', 'handoff' }) do
  local blocked, why = M.transition(s, { kind = 'request', identity = b, pending = pending })
  assert(blocked.status == 'blocked' and why == 'refuse' and blocked.context.activation == 'nonce')
  assert(blocked.reason == pending)
  local retry, next_action = M.transition(blocked, { kind = 'request', identity = b })
  assert(retry.status == 'switching' and next_action == 'activate')
  local failed = M.transition(retry, { kind = 'failed', reason = 'lost acknowledgment' })
  assert(failed.context.activation == 'nonce' and failed.status == 'blocked')
end
local invalid, why = M.transition(s, { kind = 'request', identity = {status='ambiguous'} })
assert(why == 'refuse' and invalid.context.activation == 'nonce')
local busy, why_busy = M.transition({status='switching',context=ctx}, {kind='request',identity=b})
assert(why_busy == 'refuse' and busy.context.activation == 'nonce')
print('restore_test ok')
