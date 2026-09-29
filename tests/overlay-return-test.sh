#!/usr/bin/env bash
# Actual viewer shutdown must hide older floating panes and return to draft.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-overlay-return.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
export OVERLAY_TEST_ROOT="$RT" PAIR_HOME="$ROOT" PAIR_TAG=test
export PAIR_SCROLLBACK_PENDING_PATH="$RT/pending.md"
export PAIR_DRAFT_PANE_PATH="$RT/no-cached-draft" PAIR_NVIM_PID_FILE="$RT/viewer.pid"
export PAIR_RETENTION_PROTOCOL=0 PAIR_CHANGELOG_LOG=''
mkdir "$RT/bin"
cat > "$RT/bin/zellij" <<'PY'
#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
root = Path(os.environ['OVERLAY_TEST_ROOT'])
p = root / 'host.json'
s = json.loads(p.read_text())
a = sys.argv[1:]
if a[:2] == ['action', 'list-panes']:
    assert '--command' in a, 'draft discovery requires command metadata'
    print(json.dumps({'panes': [
        dict(id=7, is_plugin=False, is_floating=False, title='agent', terminal_command='/bin/pair wrap'),
        dict(id=3, is_plugin=False, is_floating=False, title='draft', terminal_command='/pair/bin/nvim -u /pair/nvim/init.lua draft.md'),
        dict(id=9, is_plugin=False, is_floating=True, title='review'),
        dict(id=11, is_plugin=False, is_floating=True, title='viewer')]}))
elif a == ['action', 'hide-floating-panes']:
    s['visible'] = False
    s['events'].append('hide')
elif a[:2] == ['action', 'focus-pane-id']:
    assert not s['visible'], 'all floating panes must hide before focusing draft'
    assert 'overlay annotation' in (root / 'pending.md').read_text(), 'annotation must emit before draft focus'
    s['focused'] = int(a[2])
    s['events'].append('focus')
else:
    raise AssertionError('unexpected zellij command: ' + repr(a))
p.write_text(json.dumps(s))
PY
chmod +x "$RT/bin/zellij"
cat > "$RT/check.lua" <<'LUA'
vim.schedule(function()
  local ok, err = pcall(function()
    local buf = vim.api.nvim_get_current_buf()
    assert(vim.b[buf].pair_annotate, 'actual viewer must attach annotations')
    vim.bo[buf].readonly = false
    vim.bo[buf].modifiable = true
    vim.api.nvim_buf_set_lines(buf, 0, 1, false, {'agent text 🤖[overlay annotation]'})
    vim.bo[buf].modified = false
    vim.bo[buf].modifiable = false
    vim.bo[buf].readonly = true
    -- Accept the actual annotation confirmation when exiting with Escape.
    vim.fn.confirm = function() return 1 end
    if os.getenv('OVERLAY_EXIT') == 'esc' then
      vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes('<Esc>', true, false, true), 'xt', false)
    else
      vim.cmd('qa')
    end
  end)
  if not ok then print(err); vim.cmd('cquit 1') end
end)
LUA
failed=0
for viewer in scrollback changelog; do
  for exit_key in esc qa; do
    printf '{"visible":true,"focused":11,"events":[]}\n' > "$RT/host.json"
    printf 'agent text\n' > "$RT/input.md"
    : > "$RT/pending.md"
    if ! PATH="$RT/bin:$PATH" OVERLAY_EXIT="$exit_key" run_headless --timeout 10 -- \
      nvim --headless --cmd 'lua vim.api.nvim_list_uis = function() return {{width=100,height=35}} end' \
      -u "$ROOT/nvim/$viewer.lua" "$RT/input.md" -c "luafile $RT/check.lua"; then
      printf 'FAIL %s %s: viewer did not exit cleanly\n' "$viewer" "$exit_key" >&2
      failed=1
      continue
    fi
    if ! python3 - "$viewer" "$exit_key" <<'PY'
import json, os, sys
from pathlib import Path
root = Path(os.environ['OVERLAY_TEST_ROOT'])
s = json.loads((root / 'host.json').read_text())
assert not s['visible'] and s['focused'] == 3 and s['events'] == ['hide', 'focus'], \
    f'{sys.argv[1]} {sys.argv[2]} must hide underlying review and focus draft: {s}'
assert 'overlay annotation' in (root / 'pending.md').read_text()
PY
    then failed=1; fi
  done
done
if [ "$failed" -ne 0 ]; then exit 1; fi
printf 'ok scrollback/changelog Escape and :qa hide overlays, emit annotations, then focus draft\n'
