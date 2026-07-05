-- nvim/review/reconcile_test.lua — run via `nvim -l nvim/review/reconcile_test.lua`.
-- Pure Lua (classify/conflict_marker/plan_conflicts); no buffer/IO. The glue
-- `reconcile_round` is exercised by tests/review-reconcile-test.sh. Exits non-zero
-- on failure. Models markers_test.lua's eq/fails harness.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local reconcile = dofile(here .. 'reconcile.lua')
local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

-- classify: a record is clean iff its `old` still anchors in the live buffer;
-- otherwise it's a conflict (the human edited that exact span).
do
  local v1 = 'alpha kept gamma'
  local recs = {
    { old = 'kept', occurrence = 1, new = 'KEPT', explain = 'a' },
    { old = 'gone', occurrence = 1, new = 'GONE', explain = 'b' },
  }
  local r = reconcile.classify(recs, v1)
  eq(#r.clean, 1, 'one clean record')
  eq(r.clean[1] and r.clean[1].old, 'kept', 'kept is clean')
  eq(#r.conflicts, 1, 'one conflict record')
  eq(r.conflicts[1] and r.conflicts[1].old, 'gone', 'gone is a conflict')
end

-- classify honors the `occurrence or 1` fallback that apply.apply uses.
do
  local v1 = 'x y x y'
  local r = reconcile.classify({ { old = 'x', new = 'X' } }, v1) -- no occurrence → 1
  eq(#r.clean, 1, 'missing occurrence defaults to 1 (clean)')
end

-- conflict_marker: 🤖<human hunk>[reconcile — agent wanted: …]. BOTH sections
-- escaped, so unbalanced brackets in quoted code (a[0], stray ]) can't break parse.
do
  local markers = dofile(here .. 'markers.lua')
  local s = reconcile.conflict_marker('human [text]', {
    { old = 'a[0]', new = 'b', explain = 'why' },
  })
  eq(s:sub(1, #'🤖<'), '🤖<', 'starts with quoted marker')
  local parsed = markers.parse_markers(vim.split(s, '\n', { plain = true }))
  eq(#parsed, 1, 'exactly one marker parses despite brackets in code')
  eq(parsed[1] and parsed[1].quoted and parsed[1].quoted.text, 'human [text]',
    'quoted body round-trips unescaped')
  -- the agent's blocked intent (old → new) is carried in the [...] section
  local sect = parsed[1] and parsed[1].sections[1]
  eq(sect ~= nil and sect.text:find('a[0]', 1, true) ~= nil, true, 'intent old present')
  eq(sect ~= nil and sect.text:find('→ b', 1, true) ~= nil, true, 'intent new present')
end

-- multi-line hunk → multi-line quoted body still parses (budget 200, M1).
do
  local markers = dofile(here .. 'markers.lua')
  local s = reconcile.conflict_marker('line one\nline two', { { old = 'o', new = 'n' } })
  local parsed = markers.parse_markers(vim.split(s, '\n', { plain = true }))
  eq(#parsed, 1, 'multi-line hunk marker parses')
  eq(parsed[1] and parsed[1].quoted and parsed[1].quoted.text, 'line one\nline two',
    'multi-line quoted body round-trips')
end

-- plan_conflicts: coalesce conflicts by the changed hunk they fall in → one
-- synthetic replacement record per hunk (old = v1 hunk text, tagged reconcile=true).
do
  local v0 = 'title\nold para here\ntail'
  local v1 = 'title\nHUMAN para now\ntail'
  local hunks = { { 2, 1, 2, 1 } } -- v0 line 2 (1 line) ↔ v1 line 2 (1 line)
  local conflicts = {
    { old = 'old', occurrence = 1, new = 'OLD', explain = 'x' },
    { old = 'para here', occurrence = 1, new = 'PARA', explain = 'y' },
  }
  local synth = reconcile.plan_conflicts(conflicts, v0, v1, hunks)
  eq(#synth, 1, 'two conflicts in one hunk coalesce')
  eq(synth[1] and synth[1].old, 'HUMAN para now', 'old = v1 hunk text')
  eq(synth[1] and synth[1].new:find('OLD', 1, true) ~= nil, true, 'intent OLD present')
  eq(synth[1] and synth[1].new:find('PARA', 1, true) ~= nil, true, 'intent PARA present')
  eq(synth[1] and synth[1].reconcile, true, 'synthetic tagged reconcile=true for body filter')
end

-- repeated hunk text: the changed line's text also appears earlier verbatim → the
-- synthetic record's occurrence must anchor the changed (2nd) copy, not the 1st.
do
  local v0 = 'dup line\nZ\ndup line\ntail'
  local v1 = 'dup line\ndup line\ndup line\ntail'
  local hunks = { { 2, 1, 2, 1 } }
  local conflicts = { { old = 'Z', occurrence = 1, new = 'ZED', explain = 'z' } }
  local synth = reconcile.plan_conflicts(conflicts, v0, v1, hunks)
  eq(#synth, 1, 'one synthetic record')
  eq(synth[1] and synth[1].old, 'dup line', 'hunk text is the changed line')
  eq(synth[1] and synth[1].occurrence, 2, 'anchors the 2nd (changed) occurrence')
end

-- deletion (cb==0): the human removed the region → never drop the intent; append a
-- marker anchored on a nearby kept line.
do
  local v0 = 'keep\ndrop me\ntail'
  local v1 = 'keep\ntail'
  local hunks = { { 2, 1, 2, 0 } } -- v0 line 2 deleted; nothing on the v1 side
  local conflicts = { { old = 'drop me', occurrence = 1, new = 'DROP', explain = 'd' } }
  local synth = reconcile.plan_conflicts(conflicts, v0, v1, hunks)
  eq(#synth, 1, 'deletion still yields a synthetic record (no silent drop)')
  eq(synth[1] and synth[1].new:find('reconcile', 1, true) ~= nil, true, 'carries a reconcile marker')
  eq(synth[1] and synth[1].reconcile, true, 'deletion synthetic tagged reconcile=true')
  eq(synth[1] and synth[1].old ~= '' and v1:find(synth[1].old, 1, true) ~= nil, true,
    'deletion anchors on a real kept v1 line')
end

-- blank hunk (M2 review 3.1): the human BLANKED the exact line the agent targeted →
-- the changed hunk's v1 text is '' → must still yield a marker (never a silent drop).
do
  local v0 = 'alpha\nbeta content\ngamma'
  local v1 = 'alpha\n\ngamma'                 -- line 2 blanked
  local hunks = { { 2, 1, 2, 1 } }             -- v0 line 2 ↔ v1 line 2 (empty)
  local conflicts = { { old = 'beta content', occurrence = 1, new = 'BETA', explain = 'x' } }
  local synth = reconcile.plan_conflicts(conflicts, v0, v1, hunks)
  eq(#synth, 1, 'blank hunk still yields a synthetic record (no silent drop)')
  eq(synth[1] and synth[1].old ~= '', true, 'blank hunk anchors on a non-empty line')
  eq(synth[1] and synth[1].new:find('BETA', 1, true) ~= nil, true, 'blank hunk carries the agent intent')
end

-- blank-line-1 fallback (M2 review 3.1): no hunk, doc starts with a blank line →
-- the fallback must skip the empty line and anchor on a real one.
do
  local v0 = '\nsecond line here'
  local v1 = '\nHUMAN wrote this'
  local conflicts = { { old = 'second line here', occurrence = 1, new = 'SEC', explain = 'y' } }
  local synth = reconcile.plan_conflicts(conflicts, v0, v1, {}) -- no hunks → fallback group
  eq(#synth, 1, 'no-hunk fallback yields a synthetic record')
  eq(synth[1] and synth[1].old ~= '', true, 'fallback skips the blank line-1 anchor')
end

-- huge hunk (>200 lines) — keep the human text, reference the region by size.
do
  local v0lines, v1lines = {}, {}
  for i = 1, 205 do v0lines[i] = 'v0 line ' .. i; v1lines[i] = 'HUMAN line ' .. i end
  local v0 = table.concat(v0lines, '\n')
  local v1 = table.concat(v1lines, '\n')
  local hunks = { { 1, 205, 1, 205 } }
  local conflicts = { { old = 'v0 line 3', occurrence = 1, new = 'X', explain = 'z' } }
  local synth = reconcile.plan_conflicts(conflicts, v0, v1, hunks)
  eq(#synth, 1, 'huge hunk yields a synthetic record')
  eq(synth[1] and synth[1].new:find('region changed', 1, true) ~= nil, true, 'huge hunk references the region size')
  eq(synth[1] and synth[1].new:find('205 lines', 1, true) ~= nil, true, 'huge hunk states the line count')
end

-- fold (M2-review 3.1, option a): a CLEAN record sharing a human-changed line with
-- a conflict would overlap the conflict's whole-line synthetic marker → absorb its
-- intent into that marker instead of letting apply drop it. plan_conflicts takes an
-- optional trailing `clean` and returns (synth, folded).
do
  local v0 = 'the foo and bar here'
  local v1 = 'the foo and baz here'          -- human changed bar→baz
  local hunks = { { 1, 1, 1, 1 } }
  local conflicts = { { old = 'bar', occurrence = 1, new = 'BAR', explain = 'c' } } -- bar gone from v1
  local clean = { { old = 'foo', occurrence = 1, new = 'FOO', explain = 'cl' } }    -- foo still in v1
  local synth, folded = reconcile.plan_conflicts(conflicts, v0, v1, hunks, clean)
  eq(#synth, 1, 'one synthetic record for the contested line')
  eq(synth[1] and synth[1].new:find('BAR', 1, true) ~= nil, true, 'marker carries the conflict intent')
  eq(synth[1] and synth[1].new:find('FOO', 1, true) ~= nil, true, 'marker ALSO carries the folded clean intent')
  eq(#folded, 1, 'the clean record was folded (not left to be dropped)')
  eq(folded[1] and folded[1].old, 'foo', 'the folded record is the clean one')
end

-- a clean record on a DIFFERENT line than any conflict hunk is NOT folded.
do
  local v0 = 'line one\nold two'
  local v1 = 'LINE one\nHUMAN two'           -- both lines changed by human? no — line1 by clean, line2 conflict
  -- line 1: agent clean edit 'one'→'ONE' (one still present); line 2: conflict.
  local v1b = 'line ONE\nHUMAN two'
  local hunks = { { 2, 1, 2, 1 } }           -- only line 2 is the conflict hunk
  local conflicts = { { old = 'old', occurrence = 1, new = 'OLD', explain = 'c' } }
  local clean = { { old = 'one', occurrence = 1, new = 'ONE', explain = 'cl' } }
  local synth, folded = reconcile.plan_conflicts(conflicts, v0, v1b, hunks, clean)
  eq(#folded, 0, 'clean edit on a different line is not folded (applies normally)')
end

if fails > 0 then os.exit(1) end
print('reconcile_test ok')
