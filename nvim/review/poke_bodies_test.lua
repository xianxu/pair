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

do -- with_context: names the context hand-off, not the retired handoff file (pair#664)
  local s = M.with_context('body', { activation = 'a1' })
  local function has(sub, msg) if not s:find(sub, 1, true) then
    io.stderr:write('FAIL with_context ' .. msg .. ': ' .. s .. '\n'); fails = fails + 1 end end
  has('Review context: {"activation":"a1"}', 'embeds the context')
  has('--context', 'names the --context flag')
  if s:find('handoff', 1, true) then io.stderr:write('FAIL with_context names the retired handoff\n'); fails = fails + 1 end
end

if fails > 0 then os.exit(1) end
print('poke_bodies_test ok')
