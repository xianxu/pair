"""The real fixture must explain even an editor that exits before its driver."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


class FreshRestoreDiagnosticsTest(unittest.TestCase):
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
