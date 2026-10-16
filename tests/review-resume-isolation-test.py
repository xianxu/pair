"""The real resume fixture must use its checkout and leave caller artifacts alone."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / 'tests/lib'))
from review_test_env import ARTIFACT_KEYS, ReviewTestEnvironment


class ResumeIsolationTest(unittest.TestCase):
    def test_inherited_executable_selectors_cannot_escape_the_checkout(self):
        for selector in ('PAIR_HOME', 'PAIR_BIN'):
            with self.subTest(selector=selector), ReviewTestEnvironment() as isolated, \
                    tempfile.TemporaryDirectory(prefix='pair-resume-isolation-') as storage:
                temp = Path(storage)
                foreign = temp / 'foreign'
                (foreign / 'bin').mkdir(parents=True)
                marker = temp / 'foreign-invoked'
                binary = foreign / 'bin/pair'
                binary.write_text('#!' + sys.executable + '\nfrom pathlib import Path\n'
                                  + 'Path(' + repr(str(marker)) + ').write_text("invoked\\n")\n'
                                  + 'print(' + repr(json.dumps(dict(status='invalid',
                                        diagnostic='controlled foreign resolver'))) + ')\n')
                binary.chmod(0o700)
                env = isolated.environment(temp, ROOT)
                # Keep Git deterministic while pointing every inherited editor
                # artifact at a caller-owned sentinel outside the fixture tree.
                for key in (*ARTIFACT_KEYS, 'NVIM_LOG_FILE'):
                    env[key] = isolated.inherited[key]
                env.update(PAIR_DATA_DIR=str(isolated.caller),
                           PAIR_BIN=str(ROOT / 'bin/pair'))
                env[selector] = str(foreign if selector == 'PAIR_HOME' else binary)
                result = subprocess.run(['bash', str(ROOT / 'tests/review-resume-test.sh')],
                                        cwd=ROOT, env=env, text=True, capture_output=True)
                output = result.stdout + result.stderr
                self.assertEqual(result.returncode, 0, output)
                self.assertFalse(marker.exists(), 'fixture invoked inherited ' + selector)
                for assertion in ('placed=true hl=1 diag=1',
                                  'change-highlights repainted from commit',
                                  'diagnosis repainted from commit',
                                  'resolved history repaints selected branch',
                                  'ambiguous branch refuses reconstruction',
                                  'resolution and reconstruction leave checkout unchanged'):
                    self.assertIn(assertion, output)
                isolated.assert_untouched()


if __name__ == '__main__':
    unittest.main()
