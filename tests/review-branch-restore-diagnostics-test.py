"""Exercise publication timing and failure evidence through the real fixture."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent


class BranchRestoreDiagnosticsTests(unittest.TestCase):
    def test_slow_successful_resolvers_complete_the_real_handoff(self):
        with tempfile.TemporaryDirectory(prefix='pair-branch-slow-') as storage:
            temp = Path(storage)
            home = temp / 'home'
            (home / 'bin').mkdir(parents=True)
            calls = temp / 'calls'
            wrapper = home / 'bin/pair'
            wrapper.write_text('#!' + sys.executable + '\n' +
                'import os,sys,time\nfrom pathlib import Path\n' +
                'if sys.argv[1:4]==["review","readiness","--resolve"] and '
                'Path(os.environ["PAIR_REVIEW_HANDOFF_PATH"]).exists():\n' +
                ' with Path(' + repr(str(calls)) + ').open("a") as log: log.write("wait\\n")\n' +
                ' time.sleep(1.05)\n' +
                'os.execv(' + repr(str(ROOT / 'bin/pair')) + ',[' +
                repr(str(ROOT / 'bin/pair')) + ',*sys.argv[1:]])\n')
            wrapper.chmod(0o700)
            init = temp / 'review-init.lua'
            init.write_text('vim.env.PAIR_HOME=' + json.dumps(str(home)) + '\n' +
                            'dofile(' + json.dumps(str(ROOT / 'nvim/review.lua')) + ')\n')
            result = subprocess.run(['bash', str(ROOT / 'tests/review-branch-restore-test.sh')],
                                    env=dict(os.environ, PAIR_TEST_REVIEW_INIT=str(init)),
                                    text=True, capture_output=True, timeout=90)
            self.assertGreaterEqual(len(calls.read_text().splitlines()), 5,
                                    'controlled real resolver path was not exercised')
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertIn('review-branch-restore-test ok', result.stdout)
            completion = next(json.loads(line.split(': ', 1)[1])
                              for line in result.stdout.splitlines()
                              if line.startswith('handoff completion: '))
            self.assertGreater(completion['elapsed_seconds'], 5,
                               'regression did not exceed the old aggregate deadline')
            self.assertLess(completion['elapsed_seconds'], completion['budget_seconds'])
            self.assertGreaterEqual(completion['resolver_calls'], 5)
            print('slow handoff completion:', json.dumps(completion))

    def test_publication_refusal_still_fails_with_pane_evidence(self):
        with tempfile.TemporaryDirectory(prefix='pair-branch-refusal-') as storage:
            init = Path(storage) / 'review-init.lua'
            init.write_text('dofile(' + json.dumps(str(ROOT / 'nvim/review.lua')) + ')\n' +
                'local rename=os.rename\n'
                'os.rename=function(from,to)\n'
                ' if to==vim.env.PAIR_REVIEW_LANDED_PATH then return nil,"controlled publication refusal" end\n'
                ' return rename(from,to)\nend\n')
            result = subprocess.run(['bash', str(ROOT / 'tests/review-branch-restore-test.sh')],
                                    env=dict(os.environ, PAIR_TEST_REVIEW_INIT=str(init)),
                                    text=True, capture_output=True, timeout=60)
            self.assertNotEqual(result.returncode, 0, 'missing acknowledgment was accepted')
            for expected in ('did not publish landed acknowledgment',
                             'controlled publication refusal', 'handoff outcome uncertain',
                             '"handoff_exists": true', '"landed_exists": false',
                             '"elapsed_ms"', '"status": "resolved"'):
                self.assertIn(expected, result.stdout + result.stderr)
            self.assertNotIn('"snapshot":', result.stdout + result.stderr)
            self.assertNotIn('"latest_agent_body":', result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
