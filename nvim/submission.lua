local M = {}

function M.new(append_authored, send_low_level, notify_error, mint_append_id)
  local submit = {}
  local pending = nil

  mint_append_id = mint_append_id or function()
    error('Pair submission append ID generator is unavailable')
  end

  function submit.submit_operator_text(authored_body, agent_text, no_submit)
    if pending == nil or pending.body ~= authored_body then
      pending = { body = authored_body, id = mint_append_id() }
    end
    local ok, err = append_authored(authored_body, pending.id)
    if not ok then
      notify_error('Pair log append failed — ' .. tostring(err or 'unknown error'))
      return false
    end
    pending = nil
    send_low_level(agent_text, no_submit)
    return true
  end

  function submit.send_generated_prompt(body)
    send_low_level(body)
    return true
  end

  return submit
end

return M
