local M = {}

-- Targets are opaque to this transaction. Snapshot their data so a caller
-- cannot mutate an attachment target while its authored text is appended.
local function snapshot(value, seen)
  if type(value) ~= 'table' then return value end
  seen = seen or {}
  if seen[value] then return seen[value] end
  local copy = {}
  seen[value] = copy
  for key, item in pairs(value) do copy[key] = snapshot(item, seen) end
  return copy
end

function M.new(append_authored, commit_authored, send_low_level, notify_error, mint_append_id)
  local submit = {}
  local pending = nil

  mint_append_id = mint_append_id or function()
    error('Pair submission append ID generator is unavailable')
  end

  function submit.retry_pending_commit()
    if pending == nil or pending.phase ~= 'dispatched' then return true end
    local committed, commit_err = commit_authored(pending.id)
    if not committed then
      notify_error('Pair log submit marker failed — ' .. tostring(commit_err or 'unknown error'))
      return false
    end
    pending = nil
    return true
  end

  -- DOM sends are independent attempts. Their durable append remains evidence
  -- if dispatch is uncertain; a failed marker cannot freeze the next draft.
  function submit.submit_fresh_operator_text(authored_body, agent_text, attachment, operation)
    local function failure(message)
      notify_error(message)
      return false, message
    end
    if attachment ~= nil and (type(attachment) ~= 'table' or type(attachment.id) ~= 'string'
      or attachment.id == '' or type(attachment.target) ~= 'table') then
      return failure('Pair input dispatch blocked — attachment requires an ID and pinned target')
    end
    local target = attachment and snapshot(attachment.target) or nil
    local id
    -- Keep dispatch knowledge outside the protected pre-dispatch stage. Once
    -- confirmed, a marker failure (including its timeout) is only a warning.
    local called, sent, phase, send_err = pcall(function()
      if operation then operation.check() end
      id = mint_append_id()
      local ok, err = append_authored(authored_body, id, operation)
      if not ok then return false, 'start', 'Pair log append failed — ' .. tostring(err or 'unknown error') end
      if operation then operation.check() end
      return send_low_level(agent_text, 'start', target, operation)
    end)
    if not called then return failure('Pair input dispatch failed — ' .. tostring(sent)) end
    if phase ~= 'dispatched' then
      return failure('Pair input dispatch failed — ' .. tostring(send_err or 'submit was not confirmed'))
    end
    local marker_called, committed, commit_err = pcall(commit_authored, id, operation)
    if not marker_called or not committed then
      notify_error('Pair log submit marker failed — ' .. tostring(marker_called and (commit_err or 'unknown error') or committed))
    end
    if not sent then
      notify_error('Pair input dispatched with UI warning — ' .. tostring(send_err or 'unknown error'))
    end
    return true
  end

  function submit.submit_operator_text(authored_body, agent_text, attachment)
    if attachment ~= nil then return submit.submit_fresh_operator_text(authored_body, agent_text, attachment) end
    if not submit.retry_pending_commit() then return false end
    if pending ~= nil and pending.phase == 'indeterminate' then
      notify_error('Pair input dispatch blocked — reconcile the agent composer after an indeterminate body write')
      return false
    end
    if pending ~= nil and pending.phase == 'written' and pending.body ~= authored_body then
      notify_error('Pair input dispatch blocked — retry the already staged authored body before editing')
      return false
    end
    if pending ~= nil then
      if pending.body == authored_body and pending.agent_text ~= agent_text then
        notify_error('Pair input dispatch blocked — pending generated text changed')
        return false
      end
    end
    if pending == nil or pending.body ~= authored_body then
      pending = { body = authored_body, agent_text = agent_text,
        id = mint_append_id(), phase = 'start' }
    end
    local ok, err = append_authored(authored_body, pending.id)
    if not ok then
      notify_error('Pair log append failed — ' .. tostring(err or 'unknown error'))
      return false
    end
    local send_ok, phase, send_err = send_low_level(pending.agent_text, pending.phase)
    pending.phase = phase or 'indeterminate'
    if not send_ok and pending.phase ~= 'dispatched' then
      notify_error('Pair input dispatch failed — ' .. tostring(send_err or 'unknown error'))
      return false
    end
    if pending.phase ~= 'dispatched' then
      notify_error('Pair input dispatch failed — submit was not confirmed')
      return false
    end
    if pending.phase == 'dispatched' then
      submit.retry_pending_commit()
      if not send_ok then
        notify_error('Pair input dispatched with UI warning — ' .. tostring(send_err or 'unknown error'))
      end
      -- Dispatch already happened: never invite retransmission even while the
      -- commit-only retry remains pending.
      return true
    end
  end

  -- Returns whether the send actually reached the agent. It used to discard
  -- send_low_level's result and return an unconditional `true`, which made a
  -- failed send indistinguishable from a successful one -- and :PairDoctor
  -- consumed the operator's note on the strength of that `true`, so a failing
  -- `zellij action` (exactly the degraded machine the capture exists for) left
  -- a cleared draft, no notify, and the one input that cannot be re-measured
  -- gone. The other two callers ignore the result, so returning it is additive.
  function submit.send_generated_prompt(body)
    local ok, phase, err = send_low_level(body)
    if not ok then
      notify_error('Pair generated-prompt send failed — ' ..
        tostring(err or 'unknown error') .. ' (phase ' .. tostring(phase or '?') .. ')')
    end
    return ok and true or false
  end

  return submit
end

return M
