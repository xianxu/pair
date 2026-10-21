"""Nightly command contract against a real Git remote and stateful SDLC fake."""
import json
import os
from pathlib import Path
import pwd
import subprocess
import tempfile
import time
import unittest
from unittest import mock

SCRIPT = Path(__file__).with_name('nightly.sh')


class NightlyTests(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory()
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        self.repo = self.root / 'checkout ; literal'
        self.repo.mkdir()
        self.git('init', '-q', '-b', 'main')
        self.git('config', 'user.name', 'Fixture')
        self.git('config', 'user.email', 'fixture@example.invalid')
        (self.repo / 'tracked').write_text('main')
        self.git('add', 'tracked')
        self.git('commit', '-qm', 'main fixture')
        self.main = self.git('rev-parse', 'HEAD').strip()
        remote = self.root / 'remote.git'
        subprocess.run(['git', 'clone', '-q', '--bare', str(self.repo), str(remote)], check=True)
        self.git('remote', 'add', 'origin', str(remote))
        self.git('checkout', '-qb', 'operator-work')
        (self.repo / 'tracked').write_text('uncommitted operator work')
        self.artifacts = self.root / 'nightly evidence'
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        fake = self.bin / 'sdlc'
        fake.write_text('''#!/usr/bin/env python3
import json, os, signal, sys, time
from pathlib import Path
record = Path(os.environ['CALLS'])
with record.open('a') as f:
    f.write(json.dumps({'argv':sys.argv[1:], 'cwd':os.getcwd(), 'artifacts':os.environ.get('PAIR_VM_ARTIFACTS_DIR')})+'\\n')
if os.environ.get('HOLD'):
    def stop(sig, frame):
        Path(os.environ['JOINED']).write_text('cleanup joined')
        sys.exit(128+sig)
    signal.signal(signal.SIGTERM, stop)
    Path(os.environ['READY']).touch()
    while True: time.sleep(.01)
destination = Path(os.environ['PAIR_VM_ARTIFACTS_DIR'])
destination.mkdir()
(destination/'result.json').write_text(json.dumps({'exit_code':int(os.environ.get('EXIT','0')), 'image':{'fixture':'qualified'}, 'lane_claim_retained':False}))
sys.exit(int(os.environ.get('EXIT','0')))
''')
        fake.chmod(0o700)
        self.env = dict(os.environ, PATH=str(self.bin)+os.pathsep+os.environ['PATH'],
                        CALLS=str(self.root / 'calls'), READY=str(self.root / 'ready'), JOINED=str(self.root / 'joined'))

    def git(self, *args):
        return subprocess.check_output(['git', '-C', str(self.repo), *args], text=True)

    def command(self, date='2026-10-21'):
        return ['bash', str(SCRIPT), '--checkout', str(self.repo), '--artifacts-root', str(self.artifacts), '--date', date]

    def run_night(self, date='2026-10-21', **env):
        return subprocess.run(self.command(date), env=dict(self.env, **env), text=True, capture_output=True)

    def calls(self):
        path = self.root / 'calls'
        return [json.loads(row) for row in path.read_text().splitlines()] if path.exists() else []

    def test_exact_remote_main_once_without_switching_or_touching_source(self):
        result = self.run_night()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls(), [{'argv':['local-check','run','--name','conformance-vm','--rev',self.main],
            'cwd':str(self.repo), 'artifacts':str(self.artifacts / '2026-10-21' / 'vm')}])
        self.assertEqual(self.git('branch', '--show-current').strip(), 'operator-work')
        self.assertEqual((self.repo / 'tracked').read_text(), 'uncommitted operator work')
        manifest = json.loads((self.artifacts / '2026-10-21' / 'nightly.json').read_text())
        self.assertEqual((manifest['source'], manifest['exit_code']), (self.main, 0))

    def test_failure_is_retained_without_retry_and_duplicate_date_refuses(self):
        result = self.run_night(EXIT='37')
        self.assertEqual(result.returncode, 37, result.stderr)
        again = self.run_night()
        self.assertNotEqual(again.returncode, 0)
        self.assertIn('already reserved', again.stderr)
        self.assertEqual(len(self.calls()), 1)
        self.assertEqual(json.loads((self.artifacts / '2026-10-21' / 'nightly.json').read_text())['exit_code'], 37)

    def test_fetch_failure_keeps_exact_failure_and_never_invokes_check(self):
        self.git('remote', 'set-url', 'origin', str(self.root / 'missing.git'))
        result = self.run_night()
        self.assertEqual(result.returncode, 128, result.stderr)
        self.assertEqual(self.calls(), [])
        manifest = json.loads((self.artifacts / '2026-10-21' / 'nightly.json').read_text())
        self.assertEqual((manifest['source'], manifest['exit_code']), (None, 128))

    def test_sixteenth_small_night_is_admitted(self):
        self.artifacts.mkdir()
        for day in range(1, 16):
            night = self.artifacts / f'2026-10-{day:02d}'
            night.mkdir()
            (night / 'nightly.json').write_text(json.dumps({'exit_code':125}))
        result = self.run_night()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.calls()), 1)

    def test_partial_and_failed_nights_count_toward_sixteen_without_eviction(self):
        self.artifacts.mkdir()
        for day in range(1, 17):
            (self.artifacts / f'2026-10-{day:02d}').mkdir()
        result = self.run_night()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('archive', result.stderr)
        self.assertEqual(len(list(self.artifacts.glob('2026-*'))), 16)
        self.assertEqual(self.calls(), [])

    def test_incomplete_night_reserves_remaining_possible_writes(self):
        completed = self.artifacts / '2026-10-01'
        completed.mkdir(parents=True)
        (completed / 'nightly.json').write_text(json.dumps({'exit_code':125}))
        reservation = (256 << 20) + 65536 + 4096
        with (completed / 'sparse').open('wb') as stream:
            stream.truncate((4 << 30) - 2*reservation + 1)
        (self.artifacts / '2026-10-02').mkdir()  # wrapper died; its child may still write
        result = self.run_night()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('archive', result.stderr)
        self.assertEqual(self.calls(), [])

    def test_byte_saturation_refuses_before_launch(self):
        old = self.artifacts / '2026-10-01'
        old.mkdir(parents=True)
        with (old / 'sparse').open('wb') as stream:
            stream.truncate(4 << 30)
        result = self.run_night()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('archive', result.stderr)
        self.assertEqual(self.calls(), [])

    def test_symlink_evidence_refuses(self):
        self.artifacts.mkdir()
        (self.artifacts / '2026-10-01').symlink_to(self.repo, target_is_directory=True)
        result = self.run_night()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.calls(), [])

    def test_managed_storage_overlap_and_relative_root_refuse(self):
        for root in (Path.home() / '.tart/pair-runner/nightly', Path.home() / '.tart', Path('relative')):
            with self.subTest(root=root):
                self.artifacts = root
                result = self.run_night()
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.calls(), [])

    def test_managed_storage_authority_ignores_poisoned_home_before_any_write(self):
        api = {'__name__':'nightly_test'}
        source = SCRIPT.read_text().split("<<'PYTHON'\n", 1)[1].rsplit('\nPYTHON', 1)[0]
        exec(compile(source, str(SCRIPT), 'exec'), api)
        native_managed = Path(pwd.getpwuid(os.getuid()).pw_dir) / '.tart/pair-runner/nightly'
        with mock.patch.dict(os.environ, HOME=str(self.root / 'poison-home')):
            with self.assertRaisesRegex(ValueError, 'overlaps'), mock.patch.object(Path, 'mkdir', side_effect=AssertionError('native storage touched')):
                with api['reserve_nightly_artifacts'](native_managed, '2026-10-21'):
                    self.fail('admitted managed storage')

    def test_cancellation_waits_for_owned_check_and_concurrent_date_refuses(self):
        first = subprocess.Popen(self.command(), env=dict(self.env, HOLD='1'), stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        self.addCleanup(lambda: first.poll() is None and first.kill())
        deadline = time.monotonic()+10
        while not (self.root / 'ready').exists() and first.poll() is None and time.monotonic()<deadline:
            time.sleep(.01)
        self.assertTrue((self.root / 'ready').exists(), first.communicate() if first.poll() is not None else 'not ready')
        second = self.run_night('2026-10-22')
        self.assertNotEqual(second.returncode, 0)
        self.assertEqual(len(self.calls()), 1)
        first.terminate()
        _, stderr = first.communicate(timeout=10)
        self.assertEqual(first.returncode, 143, stderr)
        self.assertEqual((self.root / 'joined').read_text(), 'cleanup joined')
        self.assertEqual(json.loads((self.artifacts/'2026-10-21'/'nightly.json').read_text())['exit_code'], 143)
        self.assertNotEqual(self.run_night().returncode, 0)


if __name__ == '__main__':
    unittest.main()
