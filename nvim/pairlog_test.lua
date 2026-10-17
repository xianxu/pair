local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local pairlog = dofile(here .. 'pairlog.lua')

local old_separator = '\n\n---\n\n'
local body = 'before' .. old_separator .. '## 2026-08-28 02:03:04\n\nafter'
local v1 = '## 2026-08-28 01:02:03\n<!-- pair-log-v1 bytes=' .. #body .. ' -->\n\n' .. body .. old_separator
local legacy = '## 2026-08-27 01:02:03\n\nlegacy body' .. old_separator
local entries, err = pairlog.parse(legacy .. v1)
assert(err == nil, err)
assert(#entries == 2)
assert(entries[1].body == 'legacy body')
assert(entries[2].body == body)

local replaced, replace_err = pairlog.replace(legacy .. v1, 1, 'changed ' .. old_separator .. ' body')
assert(replace_err == nil, replace_err)
local roundtrip, parse_err = pairlog.parse(replaced)
assert(parse_err == nil, parse_err)
assert(#roundtrip == 2)
assert(roundtrip[1].body == 'legacy body')
assert(roundtrip[2].body == 'changed ' .. old_separator .. ' body')
assert(replaced:match('<!%-%- pair%-log%-v1 bytes=' .. #roundtrip[2].body .. ' %-%->'))

local with_id = '## 2026-08-28 01:02:04\n<!-- pair-log-v1 bytes=' .. #body .. ' append_id=attempt-a -->\n\n' .. body .. old_separator
local id_entries, id_err = pairlog.parse(with_id)
assert(id_err == nil, id_err)
assert(id_entries[1].append_id == 'attempt-a')
local id_replaced = pairlog.replace(with_id, 1, 'changed')
assert(id_replaced:match('append_id=attempt%-a'))

local prepared = '## 2026-08-28 01:02:04\n<!-- pair-log-v1 bytes=' .. #body .. ' append_id=attempt-b state=prepared -->\n\n' .. body .. old_separator
local prepared_entries = assert(pairlog.parse(prepared))
assert(prepared_entries[1].append_id == 'attempt-b' and prepared_entries[1].state == 'prepared')
assert(assert(pairlog.replace(prepared, 1, 'edited')):match('state=prepared'))

print('pairlog_test ok')

-- pair#618: pair-wrap's peer-turn entries stay out of the operator's history,
-- and a history edit addresses operator entries only.
local function v1_entry(id, text)
  return '## 2026-10-16 18:00:00\n<!-- pair-log-v1 bytes=' .. #text .. ' append_id=' .. id .. ' state=submitted -->\n\n' .. text .. old_separator
end
local mixed = v1_entry('aaa', 'older prompt') .. v1_entry('bbb', 'newer prompt') .. v1_entry(pairlog.PEER_PREFIX .. 'd-1', '[Couch peer from ops:0; delivery d-1]\nhi')
local operator = pairlog.operator_entries(assert(pairlog.parse(mixed)))
assert(#operator == 2 and operator[2].body == 'newer prompt', 'peer entry leaked into history')
local edited = assert(pairlog.replace(mixed, 1, 'newer prompt === note'))
local after = assert(pairlog.parse(edited))
assert(#after == 3)
assert(after[2].body == 'newer prompt === note', 'edit missed the newest operator entry')
assert(after[3].body == '[Couch peer from ops:0; delivery d-1]\nhi', 'edit rewrote the peer entry')
print('pairlog_test peer ok')
