#!/usr/bin/env bash
# Review controls through real nvim maps and a stateful pane host (#340).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-review-controls.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
export REVIEW_TEST_ROOT="$RT" PAIR_HOME="$ROOT" PAIR_TAG=test
for seam in OPEN MODE TARGET CONTEXT HANDOFF LANDED DEFINITION_REQUEST DEFINITION_RESULT; do
  export "PAIR_REVIEW_${seam}_PATH=$RT/$seam"
done
mkdir "$RT/bin"
cat > "$RT/bin/zellij" <<'PY'
#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
p=Path(os.environ['REVIEW_TEST_ROOT'])/'host.json'
s=json.loads(p.read_text())
a=sys.argv[1:]
if a[:2] == ['action','list-panes']:
    print(json.dumps({'panes': [dict(id=4,is_plugin=False,is_floating=False,title='terminal',terminal_command='/bin/pair term'),dict(id=7,is_plugin=False,is_floating=False,title='codex',terminal_command='/bin/pair wrap --agent codex'),dict(id=3,is_plugin=False,is_floating=False,title='draft',terminal_command='/pair/nvim/init.lua nvim'),dict(id=9,is_plugin=False,is_floating=True,title='review')]}))
elif a == ['action','hide-floating-panes']: s['visible']=False
elif a[:2] == ['action','focus-pane-id']:
    assert not s['visible'], 'hide before focus'
    s['focused']=int(a[2])
elif a[:2] in [['action','write-chars'],['action','send-keys']]:
    assert a[2:4] == ['--pane-id','7'], 'review submission must target agent 7'
    s.setdefault('messages', []).append(a[1])
    s['document']=(Path(os.environ['REVIEW_TEST_ROOT'])/'doc.md').read_text()
else:
    raise Exception('unexpected command '+repr(a))
p.write_text(json.dumps(s))
PY
chmod +x "$RT/bin/zellij"
printf '{"visible":true,"focused":9}\n' > "$RT/host.json"
printf 'one 🤖[first]\ntwo\nthree 🤖[last]\n' > "$RT/doc.md"
cat > "$RT/check.lua" <<'LUA'
local function run()
  local pane = assert(_G.PairReviewPane)
  local buf = vim.api.nvim_get_current_buf()
  local root = os.getenv('REVIEW_TEST_ROOT')
  local function host() return vim.json.decode(table.concat(vim.fn.readfile(root .. '/host.json'), '\n')) end
  local function reset() vim.fn.writefile({'{"visible":true,"focused":9}'}, root .. '/host.json') end
  local function press(key)
    vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes(key, true, false, true), 'xt', false)
  end
  local function hints()
    local bar = vim.api.nvim_eval_statusline(vim.o.statusline, { maxwidth = 80 }).str
    assert(bar:find('Alt+c', 1, true) and not bar:find('Esc', 1, true), 'missing return hints: ' .. bar)
    assert(bar:find('Alt+Return', 1, true), 'missing submit hint: ' .. bar)
    assert(bar:find('draft', 1, true), 'missing draft return hint: ' .. bar)
    assert(bar:find('Alt+a/r', 1, true), 'missing accept/reject hint: ' .. bar)
  end
  hints()
  -- Thread save edits only the human buffer. The actual submit mapping saves
  -- the encoded document before the stateful agent host reads it.
  vim.api.nvim_win_set_cursor(0,{1,#'one '})
  press('<CR><Esc>')
  local thread=vim.api.nvim_get_current_buf()
  assert(thread~=buf and vim.bo[thread].buftype=='acwrite','thread mapping')
  vim.api.nvim_buf_set_lines(thread,0,-1,false,{'💬: first','second line'})
  vim.cmd('write')
  assert(not host().messages,'thread save must not submit')
  vim.cmd('q')
  assert(vim.api.nvim_get_current_buf()==buf)
  -- Exercise the real send mapping to enter awaiting, including disk save.
  press('<M-CR>'); hints()
  assert(vim.deep_equal(host().messages, {'write-chars', 'send-keys'}), 'body and submit must reach agent')
  assert(host().document:find('🤖[first<br>second line]',1,true),'agent host did not read saved encoded reply')
  press('<M-n>'); assert(vim.api.nvim_win_get_cursor(0)[1] == 3, 'next marker')
  press('<M-n>'); assert(vim.api.nvim_win_get_cursor(0)[1] == 1, 'next wraps')
  press('<M-N>'); assert(vim.api.nvim_win_get_cursor(0)[1] == 3, 'previous wraps')
  -- Escape must preserve the ordinary insert/visual transitions.
  press('iX<Esc>'); assert(host().visible and vim.fn.mode() == 'n', 'insert Escape hid review')
  press('v<Esc>'); assert(host().visible and vim.fn.mode() == 'n', 'visual Escape hid review')
  local float = vim.api.nvim_open_win(vim.api.nvim_create_buf(false, true), false,
    {relative='editor',row=1,col=1,width=15,height=2})
  press('<Esc>'); assert(not vim.api.nvim_win_is_valid(float) and host().visible, 'float must dismiss first')
  vim.diagnostic.set(vim.api.nvim_create_namespace('review_diag'), buf,
    {{lnum=2,col=0,message='explanation'}})
  local _, diagnostic = pane.open_diagnostic_float()
  assert(diagnostic and vim.api.nvim_win_is_valid(diagnostic), 'diagnostic float opened')
  vim.api.nvim_set_current_win(diagnostic)
  press('<Esc>'); assert(not vim.api.nvim_win_is_valid(diagnostic) and host().visible,
    'focused diagnostic must dismiss before review')
  press('<Esc><Esc>'); assert(host().visible and host().focused == 9, 'normal Escape must keep review open')
  press('iX<Esc><Esc>'); assert(host().visible and vim.fn.mode() == 'n', 'repeated insert Escape hid review')
  press('v<Esc><Esc>'); assert(host().visible and vim.fn.mode() == 'n', 'repeated visual Escape hid review')
  reset(); press('<M-c>'); assert(not host().visible and host().focused == 3, 'Alt+c must return to draft')
  reset(); press('i<M-c><Esc>'); assert(not host().visible and host().focused == 3, 'insert Alt+c must return')
  assert(vim.fn.maparg('<M-n>', 'n', false, true).buffer == 1, 'jump must be review-local')
  vim.cmd('qa!')
end
vim.schedule(function()
  local ok, err = pcall(run)
  if not ok then print(err); vim.cmd('cquit 1') end
end)
LUA
PATH="$RT/bin:$PATH" run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/review.lua" "$RT/doc.md" -c "luafile $RT/check.lua"
printf 'ok review hints, modes, float dismissal, marker wrapping and draft return\n'
