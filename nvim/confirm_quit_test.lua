local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'confirm_quit.lua')

local failures = 0
local function check(ok, message)
  if not ok then
    io.stderr:write('FAIL ' .. message .. '\n')
    failures = failures + 1
  end
end

local function run_case(name, answer, cfg, inventory)
  local order = {}
  local quit_calls = 0
  local inventory_calls = 0
  M.run({
    config = cfg,
    inventory = function()
      inventory_calls = inventory_calls + 1
      return inventory()
    end,
    confirm = function(prompt)
      table.insert(order, 'confirm')
      check(quit_calls == 0, name .. ': confirmation preceded system work')
      if cfg then
        check(prompt:find(cfg.tag, 1, true) ~= nil, name .. ': prompt retained tag')
        check(prompt:find(cfg.agent, 1, true) ~= nil, name .. ': prompt retained agent')
      end
      return answer
    end,
    quit = function()
      quit_calls = quit_calls + 1
      table.insert(order, 'quit')
    end,
  })
  check(order[1] == 'confirm', name .. ': confirmation was first')
  check(quit_calls == (answer == 1 and 1 or 0), name .. ': Yes/No action')
  check(inventory_calls == 0, name .. ': inventory seam was not started')
end

run_case('blocking inventory', 1, { tag = 'work', agent = 'claude', args = { '--model', 'opus' }, session_id = 'native-a' }, function() while true do end end)
run_case('failed inventory', 2, { tag = 'work', agent = 'claude', args = {}, session_id = 'native-a' }, function() error('unavailable') end)
run_case('missing inventory and config', 2, nil, function() return nil end)

if failures > 0 then os.exit(1) end
print('confirm_quit_test: ok')
