import os
import pathlib
import shutil
import unittest
from unittest import mock

import zellij_oracle as oracle


class CleanupTest(unittest.TestCase):
    def setUp(self):
        self.paths = []
        self.mkdir = oracle.tempfile.mkdtemp
        def record(*args, **kwargs):
            path = self.mkdir(*args, **kwargs)
            self.paths.append(pathlib.Path(path))
            return path
        self.patch = mock.patch.object(oracle.tempfile, 'mkdtemp', side_effect=record)
        self.patch.start()

    def tearDown(self):
        self.patch.stop()
        # Remove the red test's leaked fixture directories as well.
        for path in self.paths:
            shutil.rmtree(path, ignore_errors=True)

    def assert_clean(self):
        self.assertTrue(self.paths)
        for path in self.paths:
            self.assertFalse(path.exists(), f'leaked {path}')

    @unittest.skipUnless(shutil.which('zellij'), 'native discovery requires Zellij')
    def test_success_removes_temporary_tree(self):
        result = oracle.run('cleanup sentinel', cols=20, rows=4)
        self.assertIn('cleanup sentinel', result['stdout'])
        self.assert_clean()

    def test_spawn_failure_removes_temporary_tree(self):
        opened = []
        openpty = oracle.pty.openpty
        def record_pty():
            pair = openpty()
            opened.extend(pair)
            return pair
        with mock.patch.object(oracle.pty, 'openpty', side_effect=record_pty):
            with mock.patch.object(oracle.subprocess, 'Popen', side_effect=OSError('spawn failed')):
                with self.assertRaisesRegex(OSError, 'spawn failed'):
                    oracle.run('ignored')
        self.assert_clean()
        for fd in opened:
            with self.assertRaises(OSError):
                os.fstat(fd)

    @unittest.skipUnless(shutil.which('zellij'), 'native discovery requires Zellij')
    def test_post_spawn_failure_removes_temporary_tree(self):
        real_run = oracle.subprocess.run
        def fail_dump(args, **kwargs):
            if 'dump-screen' in args:
                raise RuntimeError('dump failed')
            return real_run(args, **kwargs)
        with mock.patch.object(oracle.subprocess, 'run', side_effect=fail_dump):
            with self.assertRaisesRegex(RuntimeError, 'dump failed'):
                oracle.run('ignored')
        self.assert_clean()


    @unittest.skipUnless(shutil.which('zellij'), 'native discovery requires Zellij')
    def test_teardown_failure_removes_temporary_tree(self):
        real_run = oracle.subprocess.run
        def fail_after_kill(args, **kwargs):
            result = real_run(args, **kwargs)
            if 'kill-session' in args:
                raise RuntimeError('teardown failed')
            return result
        with mock.patch.object(oracle.subprocess, 'run', side_effect=fail_after_kill):
            with self.assertRaisesRegex(RuntimeError, 'teardown failed'):
                oracle.run('ignored')
        self.assert_clean()

class CaptureTest(unittest.TestCase):
    def test_capture_retains_only_bounded_tail(self):
        capture = oracle.CaptureTail(8192)
        for _ in range(256):
            capture.append(b'x' * 65536)
            self.assertLessEqual(len(capture.data), 8192)
        capture.append(b'final diagnostic')
        self.assertTrue(capture.data.endswith(b'final diagnostic'))


if __name__ == '__main__':
    unittest.main()
