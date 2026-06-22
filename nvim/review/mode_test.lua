-- nvim/review/mode_test.lua — run via `nvim -l nvim/review/mode_test.lua`.
-- Pure review-mode UI data. Mode meanings live in ariadne's xx-fix skill, not
-- in pair-side markdown prompt files.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'mode.lua')
local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

eq(M.parse, nil, 'mode parser removed; no pair-side markdown prompt parsing')
eq(M.directives, nil, 'mode directive prompts live in ariadne xx-fix skill')

-- list returns the built-in 3 UI modes, sorted by order.
local modes = M.list(here .. 'modes')
eq(#modes, 3, 'three stock modes')
eq(modes[1].name, 'generate', 'first by order=1')
eq(modes[2].name, 'edit', 'second by order=2')
eq(modes[3].name, 'proofread', 'third by order=3')
eq(modes[1].order, 1, 'generate order')
eq(modes[2].order, 2, 'edit order')
eq(modes[3].order, 3, 'proofread order')

local generate = M.load(here .. 'modes', 'generate')
eq(generate.name, 'generate', 'load generate')
eq(generate.body, nil, 'generate has no pair-side prompt body')
local edit = M.load(here .. 'modes', 'edit')
eq(edit.name, 'edit', 'load edit')
eq(edit.body, nil, 'edit has no pair-side prompt body')
local proofread = M.load(here .. 'modes', 'proofread')
eq(proofread.name, 'proofread', 'load proofread')
eq(proofread.body, nil, 'proofread has no pair-side prompt body')
eq((M.load(here .. 'modes', 'copy-editing')), nil, 'legacy mode file is not loadable')

local missing = M.load(here .. 'modes', 'missing')
eq(missing, nil, 'unknown mode missing')

if fails > 0 then os.exit(1) end
print('mode_test ok')
