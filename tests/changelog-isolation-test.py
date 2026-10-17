"""Run the real changelog fixture with a deterministic overlapping invocation."""
import json
import os
from pathlib import Path
import selectors
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent


class ChangelogIsolationTest(unittest.TestCase):
    def setUp(self):
        self.storage = tempfile.TemporaryDirectory(prefix='pair-changelog-isolation-', dir='/tmp')
        self.addCleanup(self.storage.cleanup)
        self.temp = Path(self.storage.name)
        self.shared = self.temp / 'shared'
        self.shared.mkdir()
        self.sentinel = self.shared / 'caller-owned'
        self.sentinel.write_text('preserve me')
        # No editor configuration, plugins, caches or logs may reach caller state.
        self.env = {key: os.environ[key] for key in ('PATH', 'LANG') if key in os.environ}
        self.env.update(HOME=str(self.temp), TMPDIR=str(self.shared),
                        XDG_CONFIG_HOME=str(self.temp / 'config'),
                        XDG_DATA_HOME=str(self.temp / 'data'),
                        XDG_STATE_HOME=str(self.temp / 'state'),
                        XDG_CACHE_HOME=str(self.temp / 'cache'),
                        NVIM_LOG_FILE=str(self.temp / 'nvim.log'))

    def launch(self, mode):
        wrapper = self.temp / (mode + '.lua')
        wrapper.write_text('''
local function report(path)
  io.stdout:write('PATH ' .. path .. '\\n'); io.stdout:flush()
end
local remove = os.remove
os.remove = function(path) report(path); return remove(path) end
local set_name = vim.api.nvim_buf_set_name
vim.api.nvim_buf_set_name = function(buf, path) report(path); return set_name(buf, path) end
local write = vim.fn.writefile
vim.fn.writefile = function(lines, path, ...)
  local result = write(lines, path, ...)
  if path:match('/pair%-changelog%-reload%-test%.md$') then
    report(path)
    MODE
  end
  return result
end
dofile(TEST)
'''.replace('MODE', {
            'hold': "io.stdin:read('*l')",
            'normal': '',
            'assertion': "write({'wrong contents'}, path)",
            'error': "error('injected fixture error')",
        }[mode]).replace('TEST', json.dumps(str(ROOT / 'nvim/changelog_test.lua'))))
        process = subprocess.Popen(['nvim', '-i', 'NONE', '-l', str(wrapper)],
                                   cwd=ROOT, env=self.env, text=True,
                                   stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE)
        def join():
            if process.poll() is None:
                process.kill()
            process.communicate()
        self.addCleanup(join)
        return process

    def finish(self, process, release=None):
        out, err = process.communicate(release, timeout=10)
        return process.returncode, out + err

    def paths(self, output):
        return {Path(line[5:]) for line in output.splitlines() if line.startswith('PATH ')}

    def test_overlapping_invocations_keep_all_paths_private(self):
        first = self.launch('hold')
        with selectors.DefaultSelector() as ready:
            ready.register(first.stdout, selectors.EVENT_READ)
            self.assertTrue(ready.select(timeout=10), 'first fixture never wrote its reload file')
        line = first.stdout.readline()
        self.assertTrue(line.startswith('PATH '), line)
        first_path = Path(line[5:].strip())
        second = self.launch('normal')
        second_code, second_output = self.finish(second)
        first_survived = first_path.is_file()
        first_code, first_output = self.finish(first, '\n')
        self.assertEqual(second_code, 0, second_output)
        self.assertEqual(first_code, 0, first_output)
        self.assertTrue(first_survived, 'second invocation removed the first reload file')
        second_paths = self.paths(second_output)
        first_paths = self.paths(line + first_output)
        self.assertEqual(len(first_paths), 3, first_output)
        self.assertEqual(len(second_paths), 3, second_output)
        first_root = first_path.parent
        second_root = next(iter(second_paths)).parent
        self.assertNotEqual(first_root, second_root)
        for paths, root in ((first_paths, first_root), (second_paths, second_root)):
            self.assertEqual({path.parent for path in paths}, {root})
            self.assertEqual(root.parent, self.shared)
            self.assertFalse(root.exists(), 'completed fixture leaked its scratch root')
        self.assertEqual(list(self.shared.glob('pair-*')), [])
        self.assertEqual(self.sentinel.read_text(), 'preserve me')

    def test_cleanup_after_assertion_failure_and_lua_error(self):
        for mode, diagnostic in (('assertion', 'FAIL reload replaced buffer'),
                                 ('error', 'injected fixture error')):
            with self.subTest(mode=mode):
                code, output = self.finish(self.launch(mode))
                self.assertNotEqual(code, 0, output)
                self.assertIn(diagnostic, output)
                path = next(iter(self.paths(output)))
                self.assertEqual(path.parent.parent, self.shared)
                self.assertFalse(path.parent.exists(), 'failed fixture leaked scratch root')
                self.assertEqual(list(self.shared.glob('pair-*')), [])
                self.assertEqual(self.sentinel.read_text(), 'preserve me')


if __name__ == '__main__':
    unittest.main()
