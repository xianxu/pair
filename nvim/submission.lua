local M = {}

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

  function submit.submit_operator_text(authored_body, agent_text, no_submit)
    if not submit.retry_pending_commit() then return false end
    if pending ~= nil and pending.phase == 'indeterminate' then
      notify_error('Pair input dispatch blocked — reconcile the agent composer after an indeterminate body write')
      return false
    end
    if pending ~= nil and pending.phase == 'written' and pending.body ~= authored_body then
      notify_error('Pair input dispatch blocked — retry the already staged authored body before editing')
      return false
    end
    if pending == nil or pending.body ~= authored_body then
      pending = { body = authored_body, id = mint_append_id(), phase = 'start' }
    end
    local ok, err = append_authored(authored_body, pending.id)
    if not ok then
      notify_error('Pair log append failed — ' .. tostring(err or 'unknown error'))
      return false
    end
    local send_ok, phase, send_err = send_low_level(agent_text, no_submit, pending.phase)
    pending.phase = phase or 'indeterminate'
    if not send_ok and pending.phase ~= 'dispatched' and pending.phase ~= 'composed' then
      notify_error('Pair input dispatch failed — ' .. tostring(send_err or 'unknown error'))
      return false
    end
    if not no_submit and pending.phase ~= 'dispatched' then
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
    if pending.phase == 'composed' and not send_ok then
      notify_error('Pair input composed with UI warning — ' .. tostring(send_err or 'unknown error'))
    end
    pending = nil -- completed compose-only transfer remains non-evidence.
    return true
  end

  function submit.send_generated_prompt(body)
    send_low_level(body)
    return true
  end

  return submit
end

return M
