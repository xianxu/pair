#!/usr/bin/env bash
# Painted compact comments and actual thread keymaps in a real Neovim PTY (#426).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
python3 - "$ROOT" <<'PY'
import errno, fcntl, json, os, pathlib, pty, select, signal, struct, subprocess, sys, tempfile, termios, time
root = pathlib.Path(sys.argv[1])
sys.path.insert(0, str(root / 'tests/lib'))
from review_test_env import ReviewTestEnvironment

driver = r'''
local function run()
  -- The production init enables unnamedplus; rescue assertions must not write
  -- the operator's system clipboard.
  vim.o.clipboard = ''
  local source = vim.api.nvim_get_current_buf()
  local source_win = vim.api.nvim_get_current_win()
  local original = vim.api.nvim_buf_get_lines(source, 0, -1, false)
  local tick = vim.api.nvim_buf_get_changedtick(source)
  local function press(keys)
    vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes(keys, true, false, true), 'xt', false)
  end
  local function paint()
    vim.cmd('redraw!')
    -- Return to Neovim's main loop so real TextChanged/CursorMoved events
    -- run; vim.wait alone inside one Lua callback does not dispatch them.
    coroutine.yield(40)
    vim.cmd('redraw!')
  end
  local function cells(line)
    local pos = vim.fn.screenpos(source_win, line, 1)
    assert(pos.row > 0, 'fixture line not visible: ' .. line)
    local text, attrs = {}, {}
    for col = pos.col, vim.o.columns do
      text[#text + 1] = vim.fn.screenstring(pos.row, col)
      attrs[#attrs + 1] = vim.fn.screenattr(pos.row, col)
    end
    return table.concat(text):gsub('%s+$', ''), attrs, pos
  end
  local function contains(line, expected)
    local text = cells(line)
    assert(text:find(expected, 1, true), 'painted line ' .. line .. ': expected ' .. expected .. '; got ' .. text)
    return text
  end
  local function open(line)
    vim.api.nvim_set_current_win(source_win)
    local raw = vim.api.nvim_buf_get_lines(source, line - 1, line, false)[1]
    vim.api.nvim_win_set_cursor(source_win, {line, assert(raw:find('🤖', 1, true)) - 1})
    press('<CR>')
    local buf = vim.api.nvim_get_current_buf()
    assert(buf ~= source and vim.bo[buf].buftype == 'acwrite', 'Enter did not focus acwrite thread')
    assert(vim.api.nvim_buf_get_name(buf):find('pair-comment://', 1, true), 'thread buffer identity')
    return buf
  end
  local function no_submit()
    local path = os.getenv('REVIEW_COMMENT_TEST_ROOT') .. '/host.log'
    for _, line in ipairs(vim.fn.filereadable(path) == 1 and vim.fn.readfile(path) or {}) do
      assert(not line:find('write-chars', 1, true) and not line:find('send-keys', 1, true),
        'thread save submitted an agent round: ' .. line)
    end
  end
  vim.api.nvim_win_set_cursor(source_win, {15, 0})
  paint()
  local compact = contains(1, '[…]{…}[last human]')
  assert(not compact:find('first hidden', 1, true) and not compact:find('agent hidden', 1, true), 'earlier turns leaked')
  local anchored = contains(2, 'anchor{…}')
  assert(not anchored:find('<anchor>', 1, true) and not anchored:find('🤖', 1, true), 'anchor delimiters visible')
  contains(3, 'deleted{…}')
  contains(4, '🤖[unclosed')
  contains(5, '`🤖[inline literal]`')
  contains(7, '🤖[fenced literal]')
  contains(9, '🤖<legacy')
  contains(10, 'anchor>[legacy reply]')
  -- Attribute checks inspect the painted cells too, not extmark metadata.
  local _, _, pos = cells(1)
  local seen = {}
  for col = pos.col, vim.o.columns do
    local cell = vim.fn.screenstring(pos.row, col)
    if cell == '[' or cell == '{' then seen[cell] = vim.fn.screenattr(pos.row, col) end
  end
  assert(seen['['] and seen['{'] and seen['['] ~= seen['{'], 'speaker brackets painted with the same attribute')
  assert(vim.api.nvim_buf_get_changedtick(source) == tick and not vim.bo[source].modified, 'rendering changed source')
  assert(vim.deep_equal(original, vim.api.nvim_buf_get_lines(source, 0, -1, false)), 'rendering changed bytes')
  -- Enter on excluded examples cannot launch a thread; ordinary counted Enter survives.
  for _, line in ipairs({4, 5, 7, 9}) do
    local raw = vim.api.nvim_buf_get_lines(source, line - 1, line, false)[1]
    vim.api.nvim_win_set_cursor(source_win, {line, assert(raw:find('🤖', 1, true)) - 1})
    press('<CR>')
    assert(vim.api.nvim_get_current_buf() == source, 'literal marker opened a thread')
  end
  vim.api.nvim_win_set_cursor(source_win, {12, 0}); press('2<CR>')
  assert(vim.api.nvim_win_get_cursor(source_win)[1] == 14, 'counted Enter lost native behavior')
  local thread = open(1)
  local lines = vim.api.nvim_buf_get_lines(thread, 0, -1, false)
  assert(table.concat(lines, '\n'):find('💬: first hidden', 1, true), 'human turn missing')
  assert(table.concat(lines, '\n'):find('🤖: agent hidden', 1, true), 'robot turn missing')
  vim.api.nvim_buf_set_lines(thread, 0, -1, false, {'💬: first hidden', '🤖: agent hidden', '💬: saved first', 'saved second'})
  vim.cmd('write')
  local saved = vim.api.nvim_buf_get_lines(source, 0, 1, false)[1]
  assert(saved == 'chain 🤖[first hidden]{agent hidden}[saved first<br>saved second]', 'thread serialization: ' .. saved)
  assert(vim.api.nvim_buf_line_count(source) == #original, 'save expanded marker into several source lines')
  assert(vim.api.nvim_get_current_buf() == thread, ':w closed thread')
  no_submit()
  vim.cmd('write')
  assert(vim.api.nvim_buf_get_lines(source, 0, 1, false)[1] == saved, 'second save changed bytes')
  vim.api.nvim_buf_set_lines(thread, -1, -1, false, {'discard this'})
  vim.cmd('quit!')
  assert(vim.api.nvim_get_current_buf() == source, ':q! did not return to source')
  assert(vim.api.nvim_buf_get_lines(source, 0, 1, false)[1] == saved, ':q! saved discarded changes')
  thread = open(1)
  assert(table.concat(vim.api.nvim_buf_get_lines(thread, 0, -1, false), '\n'):find('saved first\nsaved second', 1, true), 'reopen did not decode newline')
  vim.api.nvim_buf_set_lines(thread, 0, -1, false, {'💬: q saved'})
  press('<Esc>q')
  assert(vim.api.nvim_get_current_buf() == source, 'q did not close thread')
  assert(vim.api.nvim_buf_get_lines(source, 0, 1, false)[1] == 'chain 🤖[q saved]', 'q did not save thread')
  press('u')
  assert(vim.api.nvim_buf_get_lines(source, 0, 1, false)[1] == saved, 'undo did not restore previous source thread')
  vim.api.nvim_win_set_cursor(source_win, {15, 0})
  paint()
  contains(1, '[…]{…}[saved first<br>saved second]')
  press('<C-r>')
  assert(vim.api.nvim_buf_get_lines(source, 0, 1, false)[1] == 'chain 🤖[q saved]', 'redo did not restore saved thread')
  vim.api.nvim_win_set_cursor(source_win, {15, 0})
  paint()
  contains(1, '🤖[q saved]')
  thread = open(1)
  vim.api.nvim_buf_set_lines(thread, 0, -1, false, {'💬: preserve my unsaved reply'})
  vim.api.nvim_buf_set_lines(source, 0, 1, false, {'chain 🤖[external edit]'})
  pcall(vim.cmd, 'write')
  assert(vim.api.nvim_get_current_buf() == thread and vim.bo[thread].modified, 'conflict lost editable thread')
  assert(vim.api.nvim_buf_get_lines(source, 0, 1, false)[1] == 'chain 🤖[external edit]', 'save overwrote changed source')
  -- The PTY driver acknowledges the deliberately refused write's hit-enter
  -- prompt without suppressing the actual mapping or its error.
  press('<Esc>q')
  assert(vim.api.nvim_get_current_buf() == thread, 'q closed after failed save')
  local thread_win = vim.api.nvim_get_current_win()
  vim.api.nvim_win_close(thread_win, true)
  vim.wait(40)
  assert(not vim.api.nvim_win_is_valid(thread_win), 'forced close left thread window')
  assert(vim.fn.getreg('"'):find('preserve my unsaved reply', 1, true), 'forced teardown did not rescue unsaved thread')
  no_submit()
  vim.fn.writefile({'ok painted compact comments, literal fallback, speaker colors, native Enter, thread save/discard and undo/redo repaint'},
    os.getenv('REVIEW_COMMENT_TEST_ROOT') .. '/result')
end
local task = coroutine.create(run)
local function step()
  local ok, delay = coroutine.resume(task)
  if not ok then
    vim.fn.writefile({debug.traceback(task, tostring(delay))}, os.getenv('REVIEW_COMMENT_TEST_ROOT') .. '/error')
    vim.cmd('cquit 1')
  elseif coroutine.status(task) == 'dead' then
    vim.cmd('qa!')
  else
    vim.defer_fn(step, delay)
  end
end
vim.defer_fn(step, 200)
'''

with ReviewTestEnvironment() as isolated, tempfile.TemporaryDirectory(prefix='pair-comments-', dir='/tmp') as storage:
    fixture = pathlib.Path(storage)
    env = isolated.environment(fixture, root)
    env = {k: v for k, v in env.items() if not k.startswith('COUCH_')}
    env.update(TERM='xterm-256color', REVIEW_COMMENT_TEST_ROOT=storage, TMPDIR=storage)
    bindir = fixture / 'bin'
    bindir.mkdir()
    zellij = bindir / 'zellij'
    zellij.write_text('#!/usr/bin/env python3\nimport json,os,sys\nfrom pathlib import Path\np=Path(os.environ["REVIEW_COMMENT_TEST_ROOT"])/"host.log"\nwith p.open("a") as f: f.write(json.dumps(sys.argv[1:])+"\\n")\nif sys.argv[1:3] == ["action","list-panes"]: print(json.dumps({"panes":[]}))\n')
    zellij.chmod(0o755)
    env['PATH'] = str(bindir) + os.pathsep + env['PATH']
    (fixture / 'doc.md').write_text('chain 🤖[first hidden]{agent hidden}[last human]\nanchor 🤖<anchor>{replacement}\nstrike 🤖~deleted~{replacement}\nmalformed 🤖[unclosed\n`🤖[inline literal]`\n```markdown\n🤖[fenced literal]\n```\n🤖<legacy\nanchor>[legacy reply]\n\nplain one\nplain two\nplain three\nplain four\n')
    (fixture / 'check.lua').write_text(driver)
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', 30, 110, 0, 0))
    process = subprocess.Popen(['nvim', '-i', 'NONE', '-u', str(root / 'nvim/review.lua'), str(fixture / 'doc.md'), '-c', 'luafile ' + str(fixture / 'check.lua')], stdin=slave, stdout=slave, stderr=slave, cwd=fixture, env=env, start_new_session=True)
    os.close(slave)
    output = bytearray()
    deadline = time.monotonic() + 30
    try:
        while process.poll() is None and time.monotonic() < deadline:
            if select.select([master], [], [], 0.1)[0]:
                try:
                    chunk = os.read(master, 65536)
                    previous_size = len(output)
                    output.extend(chunk)
                    # Expected write refusal can display a hit-enter prompt.
                    # Feed the real terminal, as an operator would acknowledge
                    # the message, instead of suppressing the mapping's errors.
                    if b'Press ENTER' in output[max(0, previous_size - 10):]:
                        os.write(master, b'\r')
                except OSError as error:
                    if error.errno != errno.EIO:
                        raise
        if process.poll() is None:
            raise AssertionError('Neovim PTY test exceeded 30 seconds: ' + output.decode(errors='replace')[-8000:])
        error = fixture / 'error'
        assert process.returncode == 0, error.read_text() if error.exists() else output.decode(errors='replace')[-8000:]
        result = fixture / 'result'
        assert result.exists(), 'Neovim exited without running assertions: ' + output.decode(errors='replace')[-8000:]
        print(result.read_text().strip())
    finally:
        if process.poll() is None:
            os.killpg(process.pid, signal.SIGKILL)
        process.wait(timeout=5)
        os.close(master)
PY
