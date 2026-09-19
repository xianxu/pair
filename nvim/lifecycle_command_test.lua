local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'lifecycle_command.lua')

local failures = 0
local function check(ok, message)
  if not ok then
    io.stderr:write('FAIL ' .. message .. '\n')
    failures = failures + 1
  end
end

local function run_case(name, output, status)
  local ran, notes = nil, {}
  local ok = M.run({ 'pair', 'restart' }, {
    system = function(argv) ran = argv; return output end,
    status = function() return status end,
    notify = function(text, level) table.insert(notes, { text = text, level = level }) end,
  })
  check(ran and ran[1] == 'pair' and ran[2] == 'restart', name .. ': ran the exact argv')
  return ok, notes
end

-- Success says nothing: quit/restart end the session, and detach leaves it.
local ok, notes = run_case('success', '', 0)
check(ok and #notes == 0, 'success: no notification')

-- The #284 refusal: its text reaches the operator at ERROR, trailing newline
-- trimmed.
ok, notes = run_case('refusal', "pair restart: this session's restarts belong to Couch; relaunch the thread from Couch (Alt+n)\n", 1)
check(not ok and #notes == 1, 'refusal: one notification')
check(notes[1] and notes[1].level == vim.log.levels.ERROR, 'refusal: ERROR level')
check(notes[1] and notes[1].text == "pair restart: this session's restarts belong to Couch; relaunch the thread from Couch (Alt+n)", 'refusal: output verbatim')

-- A silent failure still says what failed and how.
ok, notes = run_case('silent failure', '  \n', 2)
check(not ok and notes[1] and notes[1].text == 'pair restart: exit 2', 'silent failure: names the command and status')

if failures > 0 then os.exit(1) end
print('lifecycle_command_test: ok')
