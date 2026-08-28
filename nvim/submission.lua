local M = {}

function M.new(append_authored, send_low_level, notify_error)
  local submit = {}

  function submit.submit_operator_text(authored_body, agent_text, no_submit)
    local ok, err = append_authored(authored_body)
    if not ok then
      notify_error('Pair log append failed — ' .. tostring(err or 'unknown error'))
      return false
    end
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
