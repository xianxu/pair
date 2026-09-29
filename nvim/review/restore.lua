-- Pure review activation policy. The controller executes the declared effect.
local M = {}
function M.same(a, b)
  return type(a) == 'table' and type(b) == 'table'
    and a.repo == b.repo and a.branch == b.branch and a.file == b.file
end
function M.validate_context(got, expected, legacy)
  if got == nil then return legacy == true end
  if type(got) ~= 'table' or type(expected) ~= 'table' then return false end
  for _, k in ipairs({ 'repo', 'branch', 'file', 'activation' }) do
    if type(got[k]) ~= 'string' or got[k] == '' or got[k] ~= expected[k] then return false end
  end
  return true
end
function M.transition(state, event)
  local next = {}; for k, v in pairs(state) do next[k] = v end
  if event.kind == 'activated' and state.status == 'switching' then
    next.status, next.context, next.identity, next.reason = 'active', event.context, state.desired, nil
    next.desired = nil
    return next, 'ack'
  end
  local function refuse(reason)
    next.status, next.reason = 'blocked', reason
    return next, 'refuse'
  end
  if event.kind == 'failed' then return refuse(event.reason) end
  if event.kind ~= 'request' then return refuse('unknown activation event') end
  if state.status == 'switching' then return state, 'refuse' end
  if not event.identity or event.identity.status ~= 'resolved' then
    return refuse((event.identity or {}).diagnostic or 'review branch has no unique document')
  end
  if M.same(state.context, event.identity) then return state, 'toggle' end
  if event.pending then return refuse(event.pending) end
  next.status, next.desired, next.reason = 'switching', event.identity, nil
  return next, 'activate'
end
return M
