-- nvim/interrupt.lua — which byte interrupts each agent's running turn.
--
-- The draft pane's Ctrl+C writes this byte into the agent pane. Most agents
-- stop their stream on ESC; Grok stops on Ctrl+C itself (its own key hint:
-- "Ctrl+c :cancel" while a turn runs) and treats ESC as ordinary input. Pure,
-- so it is unit-tested without a UI.
local M = {}

local ESC, CTRL_C = 27, 3

-- Agents whose interrupt is not ESC. Every other agent, and an unknown one,
-- gets ESC.
M.byte_by_agent = { grok = CTRL_C }

function M.byte_for(agent)
  return M.byte_by_agent[agent] or ESC
end

-- current_agent reads the tag's agent file (written by the launcher and
-- rewritten by switch-agent), so the draft follows a mid-session agent
-- switch; it falls back to the PAIR_AGENT the draft was started with.
function M.current_agent(agent_path, fallback)
  local f = agent_path and agent_path ~= '' and io.open(agent_path, 'r')
  if f then
    local agent = f:read('*l')
    f:close()
    if agent and agent ~= '' then return agent end
  end
  return (fallback and fallback ~= '') and fallback or 'claude'
end

return M
