#!/usr/bin/env python3
"""Exercise real Make entry points with a controllable Go output writer."""
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest

ROOT = Path(__file__).resolve().parent.parent
OLD = b'#!/bin/sh\nprintf "old\\n"\n'


class AtomicBuildTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='pair-atomic-build-', dir=os.environ.get('TMPDIR') or '/tmp')
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name).resolve()
        self.tree = self.base / 'worker tree'
        self.foreign = self.base / 'primary'
        self.control = self.base / 'control'
        for directory in (self.tree, self.foreign, self.control):
            directory.mkdir()
        for name in ('Makefile', 'Makefile.local', 'Makefile.workflow'):
            shutil.copyfile(ROOT / name, self.tree / name)
        for name in ('pair-go', 'couch', 'pair-launch-helper'):
            directory = self.tree / 'cmd' / name
            directory.mkdir(parents=True)
            (directory / 'main.go').write_text('package main\nfunc main() {}\n')
            sentinel = ROOT / 'cmd' / name / '.skip-make-build'
            if sentinel.exists():
                shutil.copyfile(sentinel, directory / sentinel.name)
        (self.tree / 'go.mod').write_text('module fixture\n')
        (self.tree / 'bin').mkdir()
        (self.foreign / 'bin').mkdir()
        for name in ('pair', 'couch', 'pair-launch-helper'):
            for tree in (self.tree, self.foreign):
                target = tree / 'bin' / name
                target.write_bytes(OLD)
                target.chmod(0o755)
        publisher = ROOT / 'scripts' / 'build-go-atomic.sh'
        if publisher.exists():
            (self.tree / 'scripts').mkdir()
            shutil.copyfile(publisher, self.tree / 'scripts' / publisher.name)
        tools = self.base / 'tools'
        tools.mkdir()
        fake = tools / 'go'
        fake.write_text('#!' + sys.executable + '\n' + '''import json, os, pathlib, sys, time
args=sys.argv[1:]
if args[0]=='env': sys.exit(0)
if args[0]=='run':
 out=pathlib.Path(args[args.index('--out')+1]);out.mkdir(parents=True,exist_ok=True);(out/'manifest.json').write_text('{}');sys.exit(0)
assert args[0]=='build', args
out=pathlib.Path(args[args.index('-o')+1]);out.parent.mkdir(parents=True,exist_ok=True)
control=pathlib.Path(os.environ['BUILD_CONTROL'])
with (control/'calls').open('a') as log: log.write(json.dumps({'cwd':os.getcwd(),'out':str(out),'args':args})+'\\n')
# Model Go's remove/truncate/copy publication, with a deterministic barrier.
if out.exists(): out.unlink()
out.write_bytes(b'partial')
(control/'started').touch()
if os.environ.get('BUILD_HOLD')=='1':
 deadline=time.monotonic()+20
 while not (control/'release').exists():
  if time.monotonic()>deadline: sys.exit(98)
  time.sleep(.005)
if os.environ.get('BUILD_FAIL')=='1': sys.exit(7)
out.write_bytes(b'#!/bin/sh\\nprintf "new\\\\n"\\n');out.chmod(0o755)
''')
        fake.chmod(0o755)
        self.env = dict(os.environ, PATH=str(tools) + os.pathsep + os.environ['PATH'],
                        BUILD_CONTROL=str(self.control), BUILD_HOLD='0', BUILD_FAIL='0',
                        PAIR_HOME=str(self.foreign), PAIR_DEV='1', GOFLAGS='')

    def make(self, *goals, **flags):
        return subprocess.Popen(['make', '-B', *goals, 'PAIR_GO_SRCS=', 'GOFLAGS='],
                                cwd=self.tree, env=dict(self.env, **flags),
                                stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                start_new_session=True)

    def wait_started(self, process, control=None):
        deadline = time.monotonic() + 15
        control = control or self.control
        while not (control / 'started').exists():
            if process.poll() is not None:
                self.fail(process.communicate()[0].decode())
            if time.monotonic() > deadline:
                self.fail('compiler did not reach its barrier')
            time.sleep(.005)

    def finish(self, process):
        (self.control / 'release').touch()
        try:
            output = process.communicate(timeout=20)[0]
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            process.communicate()
            raise
        return process.returncode, output.decode()

    def assert_clean(self):
        self.assertEqual(sorted(p.name for p in (self.tree / 'bin').iterdir()),
                         ['couch', 'pair', 'pair-launch-helper', 'pair-slug'] if (self.tree/'bin/pair-slug').exists()
                         else ['couch', 'pair', 'pair-launch-helper'])
        for target in (self.foreign / 'bin').iterdir():
            self.assertEqual(target.read_bytes(), OLD, 'foreign PAIR_HOME was modified')

    def test_direct_targets_preserve_running_binary_until_publish(self):
        for binary in ('pair', 'couch', 'pair-launch-helper'):
            with self.subTest(binary=binary):
                (self.control / 'started').unlink(missing_ok=True)
                (self.control / 'release').unlink(missing_ok=True)
                process = self.make('bin/' + binary, BUILD_HOLD='1')
                try:
                    self.wait_started(process)
                    self.assertEqual((self.tree / 'bin' / binary).read_bytes(), OLD)
                    self.assertEqual(subprocess.check_output([self.tree/'bin'/binary]), b'old\n')
                finally:
                    code, output = self.finish(process)
                self.assertEqual(code, 0, output)
                self.assertEqual(subprocess.check_output([self.tree/'bin'/binary]), b'new\n')
                self.assert_clean()

    def test_failed_build_retains_old_binary_and_cleans_staging(self):
        process = self.make('bin/couch', BUILD_FAIL='1')
        code, output = self.finish(process)
        self.assertNotEqual(code, 0, output)
        self.assertEqual((self.tree/'bin/couch').read_bytes(), OLD)
        self.assert_clean()

    def test_root_build_has_one_publisher_per_binary_and_no_foreign_writes(self):
        process = self.make('build')
        code, output = self.finish(process)
        self.assertEqual(code, 0, output)
        calls = [json.loads(line) for line in (self.control/'calls').read_text().splitlines()]
        self.assertEqual(len(calls), 3, calls)
        for call in calls:
            target = Path(call['out'])
            if not target.is_absolute():
                target = self.tree / target
            self.assertNotEqual(target.parent, self.tree/'bin', 'Go wrote directly to published output')
            self.assertEqual(Path(call['cwd']), self.tree)
        self.assert_clean()

    def test_cancelled_build_retains_old_binary_and_cleans_staging(self):
        process = self.make('bin/couch', BUILD_HOLD='1')
        try:
            self.wait_started(process)
            os.killpg(process.pid, signal.SIGTERM)
        finally:
            code, output = self.finish(process)
        self.assertNotEqual(code, 0, output)
        self.assertEqual((self.tree/'bin/couch').read_bytes(), OLD)
        self.assert_clean()

    def test_concurrent_builds_own_distinct_staging_directories(self):
        second = self.base/'second-control'
        second.mkdir()
        first_process = self.make('bin/couch', BUILD_HOLD='1')
        second_process = self.make('bin/couch', BUILD_HOLD='1', BUILD_CONTROL=str(second))
        try:
            self.wait_started(first_process)
            self.wait_started(second_process, second)
            first_out = json.loads((self.control/'calls').read_text())['out']
            second_out = json.loads((second/'calls').read_text())['out']
            self.assertNotEqual(Path(first_out).parent, Path(second_out).parent)
            self.assertEqual((self.tree/'bin/couch').read_bytes(), OLD)
        finally:
            (second/'release').touch()
            first_result = self.finish(first_process)
            second_result = self.finish(second_process)
        for code, output in (first_result, second_result):
            self.assertEqual(code, 0, output)
        self.assertEqual(subprocess.check_output([self.tree/'bin/couch']), b'new\n')
        self.assert_clean()

    def test_review_prepare_stays_in_worker_despite_foreign_pair_home(self):
        process = self.make('runtimebundle-generate')
        code, output = self.finish(process)
        self.assertEqual(code, 0, output)
        self.assertTrue((self.tree/'cmd/internal/runtimebundle/assets/runtime/manifest.json').exists())
        self.assertFalse((self.control/'calls').exists(), 'review prepare invoked go build')
        self.assert_clean()


if __name__ == '__main__':
    unittest.main()
