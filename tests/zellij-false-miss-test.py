#!/usr/bin/env python3
"""Exercise the standalone repro's actual worker without a live Zellij server."""

import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "scripts/repro/zellij-false-miss.sh"
TARGET = "TARGET"
MISS = f"Session '{TARGET}' not found. The following sessions are active:\n"
ABSENT = "There is no active session!\n"
QUERY = ["--session", TARGET, "action", "list-panes", "--json", "--command"]
ALIVE = ["--session", TARGET, "action", "dump-layout"]


class WorkerTest(unittest.TestCase):
    def setUp(self):
        scratch = Path("/Volumes/TestScratch")
        base = str(scratch) if scratch.is_dir() else os.environ.get("TMPDIR", "/tmp")
        self.temp = tempfile.TemporaryDirectory(prefix="zf-", dir=base)
        if len(os.fsencode(str(Path(self.temp.name) / "s/contract_version_1" / TARGET))) >= 104:
            self.temp.cleanup()
            self.temp = tempfile.TemporaryDirectory(prefix="zf-", dir="/tmp")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.config = self.root / "replies.json"
        self.log = self.root / "requests.jsonl"
        fake = self.bin / "zellij"
        fake.write_text(
            f"#!{sys.executable}\n"
            "import json, os, pathlib, sys\n"
            "root = pathlib.Path(os.environ['FAKE_ROOT'])\n"
            "log = root / 'requests.jsonl'\n"
            "calls = log.read_text().splitlines() if log.exists() else []\n"
            "replies = json.loads((root / 'replies.json').read_text())\n"
            "argv = sys.argv[1:]\n"
            "with log.open('a') as f: f.write(json.dumps(argv) + '\\n')\n"
            "if len(calls) >= len(replies):\n"
            "    sys.exit('unexpected extra zellij request')\n"
            "reply = replies[len(calls)]\n"
            "if argv != reply['argv']:\n"
            "    sys.exit('unexpected zellij argv: ' + repr(argv))\n"
            "sys.stdout.write(reply['stdout'])\n"
            "sys.stderr.write(reply['stderr'])\n"
            "sys.exit(reply['rc'])\n"
        )
        fake.chmod(0o755)
        self.env = dict(os.environ, FAKE_ROOT=str(self.root),
                        PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
                        ZELLIJ_SOCKET_DIR=str(self.root / "s"))

    def run_bash(self, code):
        result = subprocess.run(["bash", "-c", code, "test", str(SCRIPT), str(self.root)],
                                env=self.env, capture_output=True, text=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)
        return result

    def check_worker(self, *, stdout="", stderr=ABSENT, rc=1, socket_present=True,
                     alive=None, expected=(0, 0, 0, 0)):
        replies = [dict(argv=QUERY, stdout=stdout, stderr=stderr, rc=rc)]
        if alive is not None:
            replies.append(dict(argv=ALIVE, stdout=alive[0], stderr=alive[1], rc=alive[2]))
        self.config.write_text(json.dumps(replies))
        if self.log.exists():
            self.log.unlink()
        sock = socket.socket(socket.AF_UNIX)
        try:
            if socket_present:
                path = self.root / "s/contract_version_1" / TARGET
                path.parent.mkdir(parents=True, exist_ok=True)
                sock.bind(str(path))
            self.run_bash('source "$1"; root=$2; n=1; parallel=1; '
                          'sessions=(TARGET); worker 0')
            calls = [json.loads(line) for line in self.log.read_text().splitlines()]
            self.assertEqual(((self.root / "count.0").read_text(), calls),
                             (" ".join(map(str, expected)) + "\n",
                              [reply["argv"] for reply in replies]))
        finally:
            sock.close()
            if socket_present:
                path.unlink()

    def test_source_has_no_side_effects(self):
        before = set(self.root.iterdir())
        result = self.run_bash('flags=$-; old_pwd=$PWD; old_traps=$(trap -p); '
                               'source "$1"; [[ $flags == "$-" && $old_pwd == "$PWD" '
                               '&& $old_traps == "$(trap -p)" ]]')
        self.assertEqual(result.stdout, "")
        self.assertEqual(result.stderr, "")
        self.assertEqual(set(self.root.iterdir()), before)

    def test_exact_exit_zero_with_ansi_target_listing(self):
        self.check_worker(stdout="\x1b[32mTARGET\x1b[0m [Created now]\n",
                          stderr=MISS, rc=0, expected=(1, 0, 0, 0))

    def test_exit_zero_rejects_extra_stderr(self):
        for stderr in ("diagnostic\n" + MISS, MISS + "diagnostic\n", MISS + "\n", MISS.rstrip("\n")):
            with self.subTest(stderr=repr(stderr)):
                self.check_worker(stdout="TARGET [Created now]\n", stderr=stderr, rc=0)

    def test_exit_zero_requires_exact_target(self):
        for stdout, stderr in (("OTHER [Created now]\n", MISS),
                               ("TARGET-extra [Created now]\n", MISS),
                               ("TARGET [Created now]\n", MISS.replace("TARGET", "OTHER"))):
            with self.subTest(stdout=stdout, stderr=stderr):
                self.check_worker(stdout=stdout, stderr=stderr, rc=0)

    def test_exact_exit_one_confirmed_alive(self):
        self.check_worker(alive=("layout { pane; }\n", "", 0), expected=(0, 1, 0, 0))

    def test_exact_exit_one_liveness_command_fails(self):
        self.check_worker(alive=("", "cannot connect\n", 1), expected=(0, 0, 1, 0))

    def test_exact_exit_one_absent_socket_skips_liveness(self):
        self.check_worker(socket_present=False, expected=(0, 0, 1, 0))

    def test_exact_exit_one_malformed_layout_is_unconfirmed(self):
        self.check_worker(alive=("unexpected reply\n", "", 0), expected=(0, 0, 1, 0))

    def test_exit_one_rejects_extra_stderr_without_liveness(self):
        for stderr in ("diagnostic\n" + ABSENT, ABSENT + "diagnostic\n", ABSENT + "\n", ABSENT.rstrip("\n")):
            with self.subTest(stderr=repr(stderr)):
                self.check_worker(stderr=stderr, expected=(0, 0, 0, 1))

    def test_exit_one_nonempty_stdout_skips_liveness(self):
        self.check_worker(stdout="unexpected output\n", expected=(0, 0, 0, 1))


if __name__ == "__main__":
    unittest.main()
