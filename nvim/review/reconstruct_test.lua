-- nvim/review/reconstruct_test.lua — run via `nvim -l nvim/review/reconstruct_test.lua`.
-- Pure Lua; no vim API. Exits non-zero on failure.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'reconstruct.lua')
local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

-- which='new' locates by NEW_OCCURRENCE (Nth match of `new`), not `occurrence`.
local content = 'alpha\nthe value\nbeta\nthe value\n'
local out = M.decorate({ { new = 'the value', new_occurrence = 1, explain = 'first' } }, content, 'new')
eq(out.highlights[1].line, 1, 'new_occurrence=1 → line 1 (0-based)')
eq(out.highlights[1].col, 0, 'new_occurrence=1 → col 0')
eq(out.highlights[1].end_col, #'the value', 'new_occurrence=1 → exact end_col')
eq(out.diagnostics[1].lnum, 1, 'diagnostic lnum')
eq(out.diagnostics[1].message, 'first', 'diagnostic message')

local out2 = M.decorate({ { new = 'the value', new_occurrence = 2, explain = 'second' } }, content, 'new')
eq(out2.highlights[1].line, 3, 'new_occurrence=2 → line 3')

-- ADVERSARIAL (finding #1): `old` and `new` have DIFFERENT occurrence counts.
-- Base had three 'foo'; the edit replaced the 2nd 'foo' with 'bar'. Reusing the
-- old-occurrence (2) to find 'bar' would mis-land; new_occurrence=1 is correct.
local after = 'foo\nbar\nfoo\n'
local rec = { old = 'foo', occurrence = 2, new = 'bar', new_occurrence = 1, explain = 'x' }
eq(M.decorate({ rec }, after, 'new').highlights[1].line, 1, 'bar via new_occurrence=1 → line 1')

-- which='old' locates by `occurrence` against base content.
local base = 'foo\nfoo\nfoo\n'
eq(M.decorate({ rec }, base, 'old').highlights[1].line, 1, "which='old' uses occurrence=2 → line 1")

-- multi-line `new` spans the right line range.
local ml = 'x\nfoo\nbar\ny\n'
local oml = M.decorate({ { new = 'foo\nbar', new_occurrence = 1, explain = 'span' } }, ml, 'new')
eq(oml.highlights[1].line, 1, 'multi-line new starts line 1')
eq(oml.highlights[1].end_line, 2, 'multi-line new ends line 2')
eq(oml.highlights[1].col, 0, 'multi-line new starts at col 0')
eq(oml.highlights[1].end_col, 3, 'multi-line new ends at exact col')

-- Marker proposals carry their own visible delta; reconstruct should preserve
-- diagnostics but avoid an extra change highlight over the marker bytes.
local marker = 'prefix 🤖<old>{new} suffix'
local om = M.decorate({ { new = '🤖<old>{new}', new_occurrence = 1, explain = 'proposal' } }, marker, 'new')
eq(#om.highlights, 0, 'marker proposal has no change highlight')
eq(#om.diagnostics, 1, 'marker proposal keeps diagnostic')
eq(om.diagnostics[1].lnum, 0, 'marker proposal diagnostic line')

if fails > 0 then os.exit(1) end
print('reconstruct_test ok')
