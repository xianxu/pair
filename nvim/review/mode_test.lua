-- nvim/review/mode_test.lua — run via `nvim -l nvim/review/mode_test.lua`.
-- Pure parse/directives; list() reads the real modes/ dir (vim.loop, available
-- under `nvim -l`). Exits non-zero on failure.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'mode.lua')
local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

-- parse with defaults
local m = M.parse('---\nname: t\n---\nbody here\n')
eq(m.name, 't', 'name')
eq(m.scope, 'markers-only', 'default scope')
eq(m.deletions, 'propose-strike', 'default deletions')
eq(m.frontier, 'on', 'default frontier')
eq(m.body, 'body here', 'body trimmed')

-- explicit flags
local m2 = M.parse('---\nname: x\nscope: whole-doc\ndeletions: apply\nfrontier: off\norder: 2\n---\nb')
eq(m2.scope, 'whole-doc', 'scope')
eq(m2.deletions, 'apply', 'deletions')
eq(m2.frontier, 'off', 'frontier')
eq(m2.order, 2, 'order')

-- invalid flag rejected; missing name rejected
eq((M.parse('---\nname: y\nscope: bogus\n---\nb')), nil, 'invalid scope rejected')
eq((M.parse('---\nfoo: bar\n---\nb')), nil, 'missing name rejected')

-- directives render the right lines
local d = M.directives(M.parse('---\nname: w\nscope: whole-doc\nfrontier: on\ndeletions: propose-strike\n---\nx'))
eq(d:match('## How to apply') ~= nil, true, 'directives header')
eq(d:match('whole document') ~= nil, true, 'whole-doc scope line')
eq(d:match('frontier') ~= nil, true, 'frontier line')
eq(d:match('strike marker') ~= nil, true, 'propose-strike line')

-- list over the real modes/ dir → 6, sorted by order (name == basename enforced)
local modes = M.list(here .. 'modes')
eq(#modes, 6, 'six stock modes')
eq(modes[1].name, 'developmental', 'first by order=1')
eq(modes[6].name, 'free-form', 'last by order=6')

if fails > 0 then os.exit(1) end
print('mode_test ok')
