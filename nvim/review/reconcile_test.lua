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

if fails > 0 then os.exit(1) end
print('reconcile_test ok')
