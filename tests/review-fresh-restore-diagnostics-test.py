"""The real fixture must explain even an editor that exits before its driver."""
import os
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent / 'lib'))
from review_test_env import ReviewTestEnvironment


class FreshRestoreDiagnosticsTest(unittest.TestCase):
    def test_draft_closes_its_watchers_before_neovim_teardown(self):
        root = Path(__file__).resolve().parent.parent
        with ReviewTestEnvironment() as isolated, tempfile.TemporaryDirectory(prefix='pair-watch-owner-') as storage:
            temp = Path(storage)
            draft = temp / 'draft.md'
            draft.write_text('fixture draft\n')
            init = temp / 'init.lua'
            init.write_text('''
_G.fixture_watchers={}
local new=vim.uv.new_fs_event
vim.uv.new_fs_event=function()
  local handle=new()
  table.insert(fixture_watchers,handle)
  return handle
end
dofile(''' + json.dumps(str(root / 'nvim/init.lua')) + ''')
vim.uv.new_fs_event=new
''')
            driver = temp / 'driver.lua'
            driver.write_text('''
local ok,err=xpcall(function()
  assert(#fixture_watchers==2,'did not exercise both real draft watchers')
  vim.api.nvim_exec_autocmds('VimLeavePre',{})
  for _,handle in ipairs(fixture_watchers) do
    assert(handle:is_closing(),'draft left its filesystem watcher for global teardown')
  end
end,debug.traceback)
if not ok then io.stderr:write(err..'\\n');vim.cmd('cquit 1') end
vim.cmd('qa!')
''')
            env = isolated.environment(temp, root)
            env.update(PAIR_DRAFT_PATH=str(draft), PAIR_TAG='fixture',
                       PATH=str(root / 'bin') + os.pathsep + os.environ['PATH'])
            result = subprocess.run(['nvim', '--headless', '-u', str(init), str(draft),
                                     '-c', 'luafile ' + str(driver)],
                                    env=env, cwd=temp, text=True, capture_output=True, timeout=15)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_early_editor_exits_report_streams_messages_and_phase(self):
        root = Path(__file__).resolve().parent.parent
        cases = [
            (37, "vim.cmd('cquit 37')", []),
            (38, "io.stdout:write('fixture stdout\\n'); io.stderr:write('fixture stderr\\n'); "
             "vim.api.nvim_echo({{'fixture message'}}, true, {}); vim.cmd('cquit 38')",
             ['fixture stdout', 'fixture stderr', 'fixture message']),
        ]
        with tempfile.TemporaryDirectory(prefix='pair-restore-diagnostics-') as storage:
            init = Path(storage) / 'init.lua'
            for code, body, evidence in cases:
                with self.subTest(code=code):
                    init.write_text(body)
                    result = subprocess.run(
                        ['bash', str(root / 'tests/review-fresh-restore-test.sh')],
                        env=dict(os.environ, PAIR_TEST_DRAFT_INIT=str(init)),
                        text=True, capture_output=True, timeout=20)
                    self.assertNotEqual(result.returncode, 0)
                    output = result.stdout + result.stderr
                    for required in [f'"returncode": {code}', '"phase": "startup"',
                                     '"stdout":', '"stderr":', '"messages":', *evidence]:
                        self.assertIn(required, output)


if __name__ == '__main__':
    unittest.main()
