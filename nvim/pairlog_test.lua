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

print('pairlog_test ok')
