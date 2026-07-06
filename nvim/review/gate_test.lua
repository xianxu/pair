-- nvim/review/gate_test.lua — run via `nvim -l nvim/review/gate_test.lua`.
-- Pure; no vim API. The apply-gate: apply-now vs defer for a landed agent round.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local gate = dofile(here .. 'gate.lua')
local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

-- case 1: nothing changed since the agent reviewed → apply (even mid-edit).
eq(gate.decide_apply('x', 'x', true, 'i'), 'apply', 'case1 unchanged → apply')
-- case 2: human is in another pane → apply.
eq(gate.decide_apply('x', 'y', false, 'i'), 'apply', 'case2 not focused → apply')
-- case 3: on the pane but in normal mode (not editing) → apply.
eq(gate.decide_apply('x', 'y', true, 'n'), 'apply', 'case3 normal mode → apply')
-- case 4: focused + changed + mid-edit → defer.
eq(gate.decide_apply('x', 'y', true, 'i'), 'defer', 'case4 insert → defer')
eq(gate.decide_apply('x', 'y', true, 'R'), 'defer', 'case4 replace → defer')
eq(gate.decide_apply('x', 'y', true, 'v'), 'defer', 'case4 visual → defer')
eq(gate.decide_apply('x', 'y', true, 'V'), 'defer', 'case4 visual-line → defer')

if fails > 0 then os.exit(1) end
print('gate_test ok')
