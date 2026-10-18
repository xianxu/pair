-- Run: nvim --headless -u NONE -l tests/review-observation-gate-test.lua
local root = vim.fn.getcwd()
local path = root .. '/tests/lib/review_observation_gate.lua'
assert(vim.uv.fs_stat(path), 'test-only observation callback gate is missing')
local module = dofile(path)
local native = vim.system
local pending = {}
local fake = function(cmd, opts, callback)
  local process = {wait = function() return 'native wait' end, kill = function() return 'native kill' end}
  pending[#pending + 1] = {cmd = cmd, opts = opts, callback = callback, process = process}
  return process
end
vim.system = fake
local autocmds_before = #vim.api.nvim_get_autocmds({event = 'VimLeavePre'})
local gate = module.install()
assert(#vim.api.nvim_get_autocmds({event = 'VimLeavePre'}) == autocmds_before + 1)
assert(PairReviewObservationGate == gate)
local cmd = {'/bin/echo', 'review', 'readiness', '--resolve', 'private document'}
local opts = {text = true, timeout = 2500}
local seen = {}
local callback = function(result) seen[#seen + 1] = result end
assert(gate:arm())
local process = vim.system(cmd, opts, callback)
assert(process == pending[1].process and process:wait() == 'native wait' and process:kill() == 'native kill')
assert(pending[1].cmd == cmd and pending[1].opts == opts)
local entry = gate:snapshot().entries[1]
assert(entry.state == 'started' and not gate:release(entry.id))
assert(type(entry.started_ns) == 'number' and entry.completed_ns == nil and entry.released_ns == nil)
assert(entry.timeout == 2500)
assert(gate:snapshot().inflight == 1 and gate:snapshot().held == 0)
assert(not gate:arm())
local ok = pcall(vim.system, cmd, opts, callback)
assert(not ok, 'overlapping held observations must not retain more results')
local result = {code = 124, signal = 15, stdout = 'private document bytes', stderr = 'private stderr'}
pending[1].callback(result)
assert(#seen == 0)
local snapshot = gate:snapshot()
assert(snapshot.entries[1].state == 'result' and snapshot.inflight == 0 and snapshot.held == 1)
assert(snapshot.entries[1].code == 124 and snapshot.entries[1].signal == 15)
assert(snapshot.entries[1].stdout_bytes == #result.stdout)
assert(snapshot.entries[1].started_ns == entry.started_ns)
assert(snapshot.entries[1].completed_ns >= entry.started_ns and snapshot.entries[1].released_ns == nil)
assert(snapshot.entries[1].timeout == opts.timeout)
assert(not vim.json.encode(snapshot):find('private', 1, true), 'metadata must not contain document bytes')
snapshot.entries[1].state = 'tampered'
assert(gate:snapshot().entries[1].state == 'result')
assert(gate:release(entry.id) and seen[1] == result)
assert(gate:snapshot().entries[1].state == 'released' and not gate:release(entry.id))
assert(gate:snapshot().entries[1].released_ns >= snapshot.entries[1].completed_ns)
-- A second sequential completion is held while armed, with a distinct ID.
vim.system(cmd, opts, callback)
local second = gate:snapshot().entries[2]
assert(second.id ~= entry.id)
gate:disarm()
pending[2].callback(result)
assert(#seen == 1 and gate:release(second.id) and #seen == 2)
-- Disarmed, synchronous and unrelated calls preserve the exact callback boundary.
vim.system(cmd, opts, callback)
assert(gate:snapshot().inflight == 1 and not gate:arm())
pending[3].callback(result)
assert(gate:snapshot().inflight == 0 and gate:snapshot().held == 0 and seen[#seen] == result)
local passed = gate:snapshot().entries[3]
assert(passed.started_ns <= passed.completed_ns and passed.completed_ns <= passed.released_ns)
assert(gate:arm())
local sync = vim.system(cmd, opts)
assert(sync == pending[4].process and pending[4].callback == nil)
vim.system({'/bin/echo', 'other'}, opts, callback)
assert(pending[5].callback == callback)
-- Bounded history discards released metadata only; results remain exact.
for _ = 1, 40 do
  vim.system(cmd, opts, callback)
  local item = gate:snapshot().entries[#gate:snapshot().entries]
  pending[#pending].callback(result)
  assert(gate:release(item.id))
end
assert(#gate:snapshot().entries == 32)
vim.system(cmd, opts, callback)
local late = pending[#pending].callback
local count = #seen
gate:close()
late(result)
assert(#seen == count and #gate:snapshot().entries == 0 and not gate:snapshot().armed)
assert(vim.system == fake)
assert(#vim.api.nvim_get_autocmds({event = 'VimLeavePre'}) == autocmds_before)
assert(not gate:arm() and not gate:release(second.id))
-- Teardown drops a completed result without delivering it.
local held_gate = module.install()
assert(held_gate:arm())
vim.system(cmd, opts, callback)
pending[#pending].callback(result)
assert(held_gate:snapshot().held == 1)
local held_id = held_gate:snapshot().entries[1].id
held_gate:close()
assert(held_gate:snapshot().held == 0 and not held_gate:release(held_id) and #seen == count)
-- Teardown does not overwrite a newer wrapper installed by another fixture.
local next_gate = module.install()
local later_wrapper = function() end
vim.system = later_wrapper
next_gate:close()
assert(vim.system == later_wrapper)
vim.system = native
-- Native process completion, including native timeout options, precedes release.
local real_gate = module.install()
assert(real_gate:arm())
local real_result
local real_process = vim.system({'/bin/echo', 'review', 'readiness', '--resolve'},
  {text = true, timeout = 2500}, function(value) real_result = value end)
assert(type(real_process.wait) == 'function' and type(real_process.kill) == 'function')
assert(vim.wait(3000, function()
  local items = real_gate:snapshot().entries
  return items[1] and items[1].state == 'result'
end), 'native completion did not reach result boundary')
assert(real_result == nil)
local real_entry = real_gate:snapshot().entries[1]
assert(real_entry.timeout == 2500 and real_entry.started_ns <= real_entry.completed_ns)
assert(real_entry.released_ns == nil)
assert(real_entry.code == 0 and real_gate:release(real_entry.id))
assert(real_gate:snapshot().entries[1].released_ns >= real_entry.completed_ns)
assert(real_result.stdout == 'review readiness --resolve\n')
real_gate:close()
assert(vim.system == native)
print('review observation gate: all assertions passed')
