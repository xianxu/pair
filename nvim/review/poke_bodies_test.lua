-- nvim/review/poke_bodies_test.lua — run via `nvim -l nvim/review/poke_bodies_test.lua`
-- (or `make test-lua`). Pure strings; no buffer/IO. Models record_test.lua.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'poke_bodies.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s:\n  got  %q\n  want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

eq(M.agent_applied(2, 1, '/a/doc.md'),
  'applied 2 edit(s) (1 dropped) to /a/doc.md — commit the agent round',
  'agent_applied includes the dropped segment when dropped>0')
eq(M.agent_applied(2, 0, '/a/doc.md'),
  'applied 2 edit(s) to /a/doc.md — commit the agent round',
  'agent_applied omits the dropped segment when dropped==0')
eq(M.agent_applied(1, 0, '/a/doc.md', 2),
  'applied 1 edit(s) (2 to reconcile) to /a/doc.md — commit the agent round',
  'agent_applied includes the reconcile segment when conflicts>0 (#89)')
eq(M.agent_applied(0, 1, '/a/doc.md', 1),
  'applied 0 edit(s) (1 dropped) (1 to reconcile) to /a/doc.md — commit the agent round',
  'agent_applied composes dropped + reconcile segments')
eq(M.human_finished('/a/doc.md'),
  'finished my edits to /a/doc.md — please review in Edit posture',
  'human_finished')

eq(M.human_finished('/a/doc.md', 'proofread', 'keep the title', 'Proofread'),
  'finished my edits to /a/doc.md — please review in Proofread posture; instruction: keep the title',
  'human_finished with mode and instruction')

eq(M.human_finished('/a/doc.md', 'edit', '', 'Edit', '/tmp/review-context-t.md'),
  'finished my edits to /a/doc.md — please review in Edit posture; use stripped review context at /tmp/review-context-t.md for reading, while editing the actual file',
  'human_finished with stripped definition context')

eq(M.ship_requested('/a/doc.md'),
  'ship /a/doc.md — run docflow ship for the active review branch; the agent owns git',
  'ship_requested')

do -- definition_requested: the body names the result command and request artifact, not document content
  local s = M.definition_requested('/a/doc.md', 'req-1', 'ASIN')
  local function has(sub, msg) if not s:find(sub, 1, true) then
    io.stderr:write('FAIL definition_requested ' .. msg .. ': ' .. s .. '\n'); fails = fails + 1 end end
  has('/a/doc.md', 'names the file')
  has('ASIN', 'names the term')
  has('review-definition-request', 'points at request artifact')
  has('pair review definition --term "ASIN" req-1 <definition>', 'names the response command')
end

do -- review_opened: the review-START announce poke names the file + the workbench protocol
  local s = M.review_opened('/a/doc.md')
  local function has(sub, msg) if not s:find(sub, 1, true) then
    io.stderr:write('FAIL review_opened ' .. msg .. ': ' .. s .. '\n'); fails = fails + 1 end end
  has('/a/doc.md', 'names the file')
  has('Pair review workbench', 'names the workbench protocol')
  has('records', 'says propose records')
  has('do NOT edit the file', 'forbids file-write')
  has('Edit', 'names the default posture')
  has('resolve 🤖[] comments', 'names fulfill/punt marker handling')
end

if fails > 0 then os.exit(1) end
print('poke_bodies_test ok')
