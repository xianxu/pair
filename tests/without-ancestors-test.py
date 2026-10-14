#!/usr/bin/env python3
"""Real-process lifetime regressions for the reparenting test helper."""
import fcntl
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest

HELPER = Path(__file__).with_name("without-ancestors.py")
CHILD = """
import fcntl, os, sys, time
with open(sys.argv[1], 'w') as lease:
    fcntl.flock(lease, fcntl.LOCK_EX)
    lease.write(str(os.getpgrp()))
    lease.flush()
    time.sleep(30)
"""
PARENT = """
import subprocess, sys, time
subprocess.Popen([sys.executable, '-c', sys.argv[1], sys.argv[2]])
if sys.argv[3] == 'wait':
    time.sleep(30)
"""


class LifetimeTests(unittest.TestCase):
    def launch(self, directory, mode="wait", timeout="0.5", wait_ready=True):
        lease = Path(directory) / "lease"
        process = subprocess.Popen(
            [sys.executable, str(HELPER), "--timeout", timeout,
             sys.executable, "-c", PARENT, CHILD, str(lease), mode],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        # Register cleanup after the directory, so process cleanup runs first.
        self.addCleanup(self.cleanup, process, lease)
        if not wait_ready:
            return process, lease
        deadline = time.monotonic() + 3
        while not lease.exists() or not lease.read_text():
            self.assertLess(time.monotonic(), deadline, "descendant did not start")
            time.sleep(0.01)
        return process, lease

    def directory(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        return directory.name

    @staticmethod
    def cleanup(process, lease):
        if lease.exists() and lease.read_text() and LifetimeTests.alive(lease):
            try:
                os.killpg(int(lease.read_text()), signal.SIGKILL)
            except ProcessLookupError:
                pass
        if process.poll() is None:
            process.kill()
        process.wait(timeout=3)

    @staticmethod
    def alive(lease):
        with lease.open() as handle:
            try:
                fcntl.flock(handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
                return False
            except BlockingIOError:
                return True

    def assert_terminated(self, lease):
        deadline = time.monotonic() + 2
        while self.alive(lease) and time.monotonic() < deadline:
            time.sleep(0.01)
        self.assertFalse(self.alive(lease), "grandchild still owns its live-process lease")

    def test_timeout_terminates_descendants(self):
        directory = self.directory()
        process, lease = self.launch(directory)
        self.assertEqual(process.wait(timeout=3), 1)
        self.assert_terminated(lease)

    def test_supervisor_cancellation_terminates_descendants(self):
        for cancellation in (signal.SIGTERM, signal.SIGINT):
            with self.subTest(signal=cancellation):
                directory = self.directory()
                process, lease = self.launch(directory, timeout="20")
                process.send_signal(cancellation)
                self.assertEqual(process.wait(timeout=3), 128 + cancellation)
                self.assert_terminated(lease)

    def test_cancellation_during_startup(self):
        for cancellation in (signal.SIGTERM, signal.SIGINT):
            for delay in (0, 0.01, 0.02, 0.04):
                with self.subTest(signal=cancellation, delay=delay):
                    directory = self.directory()
                    process, lease = self.launch(directory, timeout="20", wait_ready=False)
                    time.sleep(delay)
                    process.send_signal(cancellation)
                    # Python startup can itself translate SIGINT into exit 1;
                    # exact signal status is tested after readiness above.
                    self.assertNotEqual(process.wait(timeout=3), 0)
                    # Allow any worker caught before setsid to observe the
                    # closed handshake; it must never launch a late descendant.
                    time.sleep(0.05)
                    if lease.exists() and lease.read_text():
                        self.assert_terminated(lease)

    def test_success_preserves_sidecar(self):
        directory = self.directory()
        process, lease = self.launch(directory, mode="exit")
        self.assertEqual(process.wait(timeout=3), 0)
        self.assertTrue(self.alive(lease))

    def test_command_exit_status(self):
        result = subprocess.run([sys.executable, str(HELPER), sys.executable,
                                 "-c", "raise SystemExit(7)"], timeout=3)
        self.assertEqual(result.returncode, 7)


if __name__ == "__main__":
    unittest.main()
