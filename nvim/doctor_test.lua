-- Headless tests for nvim/doctor.lua — run via `nvim -l nvim/doctor_test.lua`
-- (or `make test-lua`). Pure Lua; no vim API. Exits non-zero on failure so the
-- make target fails loudly. Pins the two behaviors the Spec cares about:
-- $PAIR_HOME-absolute substitution and graceful nil-on-unset.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'doctor.lua')

local fails = 0
local function ok(cond, msg)
  if not cond then
    io.stderr:write('FAIL ' .. msg .. '\n')
    fails = fails + 1
  end
end

local HOME = '/Users/x/workspace/pair'
local p = M.payload(HOME)

ok(type(p) == 'string', 'payload returns a string for a real PAIR_HOME')
ok(p:find(HOME .. '/doctor/doctor.sh', 1, true) ~= nil, 'payload has the absolute doctor.sh path')
ok(p:find(HOME .. '/doctor/SKILL.md', 1, true) ~= nil, 'payload references SKILL.md (DRY pointer)')
-- The path must be SUBSTITUTED, not left as a literal for the agent's shell.
ok(p:find('$PAIR_HOME', 1, true) == nil, 'payload has no literal $PAIR_HOME (substituted, not deferred)')

-- Trailing slash on PAIR_HOME must not double up in the paths.
local q = M.payload(HOME .. '/')
ok(q:find(HOME .. '/doctor/doctor.sh', 1, true) ~= nil, 'trailing slash trimmed (no //)')
ok(q:find('//doctor', 1, true) == nil, 'no doubled slash before doctor/')

-- Graceful degrade: unset / empty ⇒ nil (caller notifies, no broken send).
ok(M.payload(nil) == nil, 'payload(nil) ⇒ nil')
ok(M.payload('') == nil, "payload('') ⇒ nil")

if fails > 0 then
  io.stderr:write(string.format('\n%d failure(s)\n', fails))
  os.exit(1)
end
print('all doctor.lua tests passed')
