#!/usr/bin/env bash
# tests/review-definition-test.sh — review-pane inline definitions persist as
# managed footnotes and render exact-span diagnostics/highlights.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-definition-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"

cat > "$RT/doc.md" <<'DOC'
here is ASIN in context
DOC

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
dofile(ROOT .. '/nvim/review.lua')
local apply = dofile(ROOT .. '/nvim/review/apply.lua')

local OUT = io.open(os.getenv('RESULT'), 'w')
local fails = 0
local function ok(cond, msg)
  if cond then OUT:write('ok ' .. msg .. '\n') else OUT:write('FAIL ' .. msg .. '\n'); fails = fails + 1 end
end
local function content(buf)
  return table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
end
local function read_json(path)
  local body = table.concat(vim.fn.readfile(path), '\n')
  return vim.json.decode(body)
end

local ok_run, err = pcall(function()
local file = os.getenv('DOC')
local buf = vim.api.nvim_create_buf(true, false)
vim.api.nvim_set_current_buf(buf)
vim.api.nvim_buf_set_name(buf, file)
vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'here is ASIN in context' })

local req = _G.PairReviewPane.request_definition(buf, file, { 1, 8 }, { 1, 11 }, { poke = false })
ok(type(req) == 'table' and req.request_id ~= nil and req.term == 'ASIN', 'request helper returns request metadata')
local req_path = os.getenv('PAIR_DATA_DIR') .. '/review-definition-request-def.json'
local request_doc = read_json(req_path)
ok(request_doc.request_id == req.request_id and request_doc.term == 'ASIN', 'request json records id and term')
ok(request_doc.context == 'here is ASIN in context', 'request context is current document text')

local result_path = os.getenv('PAIR_DATA_DIR') .. '/review-definition-result-def.json'
vim.fn.writefile({
  vim.json.encode({
    request_id = req.request_id,
    term = 'ASIN',
    definition = 'Amazon Standard Identification Number.',
    session = 'sid',
  })
}, result_path)
ok(_G.PairReviewPane.apply_definition_result(buf) == true, 'result applies')
ok(content(buf) == table.concat({
  'here is ASIN[^asin] in context',
  '',
  '---',
  '',
  '[^asin]: Amazon Standard Identification Number.',
}, '\n'), 'definition persisted as managed footnote')

local marks = vim.api.nvim_buf_get_extmarks(buf, apply.DEF_HL, 0, -1, { details = true })
ok(#marks == 1 and marks[1][2] == 0 and marks[1][3] == 8 and marks[1][4].end_col == 19,
  'definition highlight spans only term plus footnote ref')
local diags = vim.diagnostic.get(buf, { namespace = apply.DIAG })
ok(#diags == 1 and diags[1].lnum == 0 and diags[1].col == 8 and diags[1].end_col == 19
  and diags[1].message:match('ASIN') and diags[1].message:match('Amazon Standard'),
  'definition diagnostic uses exact span and stored definition')

local req2 = _G.PairReviewPane.request_definition(buf, file, { 1, 8 }, { 1, 11 }, { poke = false })
local request_doc2 = read_json(req_path)
ok(request_doc2.context == 'here is ASIN[^asin] in context',
  'definition request context strips managed footnote footer')
vim.fn.writefile({
  vim.json.encode({
    request_id = req2.request_id,
    term = 'ASIN',
    definition = 'Updated definition.',
    session = 'sid',
  })
}, result_path)
ok(_G.PairReviewPane.apply_definition_result(buf) == true, 'redefinition applies')
ok(content(buf) == table.concat({
  'here is ASIN[^asin] in context',
  '',
  '---',
  '',
  '[^asin]: Updated definition.',
}, '\n'), 'redefinition updates existing footnote without duplicate ref')
apply.clear_all(buf)
ok(#vim.api.nvim_buf_get_extmarks(buf, apply.DEF_HL, 0, -1, {}) == 0,
  'clear_all removes definition highlights before rehydrate')
_G.PairReviewPane.rehydrate_definitions(buf)
local restored = vim.api.nvim_buf_get_extmarks(buf, apply.DEF_HL, 0, -1, { details = true })
ok(#restored == 1 and restored[1][3] == 8 and restored[1][4].end_col == 19,
  'rehydrate_definitions redraws exact span from durable footnote')
end)

if not ok_run then
  fails = fails + 1
  OUT:write('FAIL lua error: ' .. tostring(err) .. '\n')
end

OUT:write(fails == 0 and 'definition_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

PAIR_ROOT="$ROOT" RESULT="$RESULT" DOC="$RT/doc.md" PAIR_DATA_DIR="$RT" PAIR_TAG=def \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'definition_test ok' "$RESULT"; then
  echo "review-definition-test FAILED"; exit 1
fi
echo "review-definition-test ok"
