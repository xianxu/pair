#!/usr/bin/env python3
"""Real-process lifetime regressions for the reparenting test helper."""
import fcntl
import os
from pathlib import Path
import select
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
    def setUp(self):
        self.outputs = {}

    def launch(self, directory, mode="wait", timeout="0.5", wait_ready=True,
               helper=HELPER, pass_fds=()):
        lease = Path(directory) / "lease"
        output = tempfile.TemporaryFile()
        self.addCleanup(output.close)
        # Background launchers may ignore SIGINT. Python preserves SIG_IGN
        # across exec, losing cancellation before the helper installs cancel.
        # This fixture is single-threaded; establish the child's exec default
        # without a preexec_fn, then immediately restore the caller's handler.
        inherited = signal.signal(signal.SIGINT, signal.default_int_handler)
        try:
            process = subprocess.Popen(
                [sys.executable, str(helper), "--timeout", timeout,
                 sys.executable, "-c", PARENT, CHILD, str(lease), mode],
                stdout=output, stderr=subprocess.STDOUT, pass_fds=pass_fds)
        finally:
            signal.signal(signal.SIGINT, inherited)
        self.outputs[process.pid] = output
        # Register cleanup after the directory, so process cleanup runs first.
        self.addCleanup(self.cleanup, process, lease)
        if not wait_ready:
            return process, lease
        deadline = time.monotonic() + 3
        while not lease.exists() or not lease.read_text():
            if time.monotonic() >= deadline:
                self.fail(self.diagnostics(process, lease, "awaiting descendant"))
            time.sleep(0.01)
        return process, lease

    def diagnostics(self, process, lease, phase):
        output = self.outputs[process.pid]
        # A positional bounded read leaves the child's shared write offset alone.
        captured = os.pread(output.fileno(), 65536, 0).decode(errors="replace")
        group = lease.read_text() if lease.exists() else "not published"
        return (f"phase={phase}; helper_pid={process.pid}; "
                f"returncode={process.poll()}; descendant_group={group}; "
                f"caller_SIGINT={signal.getsignal(signal.SIGINT)}; "
                f"output (first 64 KiB):\n{captured}")

    def wait_for_exit(self, process, lease, phase):
        try:
            return process.wait(timeout=3)
        except subprocess.TimeoutExpired:
            self.fail(self.diagnostics(process, lease, phase))

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
        self.assertEqual(self.wait_for_exit(process, lease, "timeout"), 1)
        self.assert_terminated(lease)

    def test_supervisor_cancellation_terminates_descendants(self):
        for cancellation in (signal.SIGTERM, signal.SIGINT):
            with self.subTest(signal=cancellation):
                directory = self.directory()
                process, lease = self.launch(directory, timeout="20")
                process.send_signal(cancellation)
                self.assertEqual(self.wait_for_exit(process, lease,
                                 f"ready; signal={cancellation}"), 128 + cancellation)
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
                    self.assertNotEqual(self.wait_for_exit(
                        process, lease, f"startup; signal={cancellation}; delay={delay}"), 0)
                    # Allow any worker caught before setsid to observe the
                    # closed handshake; it must never launch a late descendant.
                    time.sleep(0.05)
                    if lease.exists() and lease.read_text():
                        self.assert_terminated(lease)

    def test_ignored_parent_sigint_cancels_before_helper_entry(self):
        directory = self.directory()
        ready_read, ready_write = os.pipe()
        release_read, release_write = os.pipe()
        self.addCleanup(os.close, ready_read)
        self.addCleanup(os.close, release_write)
        bootstrap = Path(directory) / "before-helper.py"
        bootstrap.write_text(
            "import os, runpy, signal, sys\n"
            "print('pre-handler gate reached', file=sys.stderr, flush=True)\n"
            f"os.write({ready_write}, str(signal.getsignal(signal.SIGINT)).encode())\n"
            f"if os.read({release_read}, 1) != b'x': raise SystemExit(1)\n"
            f"sys.argv[0] = {str(HELPER.resolve())!r}\n"
            "runpy.run_path(sys.argv[0], run_name='__main__')\n")
        inherited = signal.signal(signal.SIGINT, signal.SIG_IGN)
        try:
            try:
                process, lease = self.launch(
                    directory, timeout="20", wait_ready=False, helper=bootstrap,
                    pass_fds=(ready_write, release_read))
            finally:
                os.close(ready_write)
                os.close(release_read)
            self.assertEqual(signal.getsignal(signal.SIGINT), signal.SIG_IGN,
                             "launch changed the caller's signal disposition")
            readable, _, _ = select.select([ready_read], [], [], 3)
            self.assertTrue(readable, "pre-handler bootstrap did not start")
            disposition = os.read(ready_read, 1024).decode()
            diagnostic = self.diagnostics(process, lease, "before helper entry")
            self.assertIn("pre-handler gate reached", diagnostic)
            self.assertIn(f"helper_pid={process.pid}", diagnostic)
            self.assertIn("descendant_group=not published", diagnostic)
            process.send_signal(signal.SIGINT)
            code = self.wait_for_exit(process, lease,
                                     f"before helper entry; child_SIGINT={disposition}")
            self.assertNotEqual(code, 0)
            self.assertFalse(lease.exists(), "command started before gate release")
        finally:
            signal.signal(signal.SIGINT, inherited)

    def test_success_preserves_sidecar(self):
        directory = self.directory()
        process, lease = self.launch(directory, mode="exit")
        self.assertEqual(self.wait_for_exit(process, lease, "successful command"), 0)
        self.assertTrue(self.alive(lease))

    def test_command_exit_status(self):
        result = subprocess.run([sys.executable, str(HELPER), sys.executable,
                                 "-c", "raise SystemExit(7)"], timeout=3)
        self.assertEqual(result.returncode, 7)


if __name__ == "__main__":
    unittest.main()
