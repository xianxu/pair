#!/usr/bin/env python3
"""Exercise the standalone repro's actual worker without a live Zellij server."""

import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import textwrap
import unittest
from collections import Counter


SCRIPT = Path(__file__).resolve().parents[1] / "scripts/repro/zellij-false-miss.sh"
TARGET = "TARGET"
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
            "sys.path.insert(0, os.environ['ZELLIJ_FIXTURE_LIB'])\n"
            "from zellij_fixture_replies import emit_case\n"
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
            "emit_case(reply['case'])\n"
        )
        fake.chmod(0o755)
        self.env = dict(os.environ, FAKE_ROOT=str(self.root),
                        ZELLIJ_FIXTURE_LIB=str(SCRIPT.parents[2] / "tests/lib"),
                        PYTHONDONTWRITEBYTECODE="1",
                        PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
                        ZELLIJ_SOCKET_DIR=str(self.root / "s"))

    def run_bash(self, code):
        result = subprocess.run(["bash", "-c", code, "test", str(SCRIPT), str(self.root)],
                                env=self.env, capture_output=True, text=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)
        return result

    def check_worker(self, *, case="no-session", socket_present=True,
                     alive=None, expected=(0, 0, 0, 0)):
        replies = [dict(argv=QUERY, case="zfm-worker-" + case)]
        if alive is not None:
            replies.append(dict(argv=ALIVE, case="zfm-worker-" + alive))
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
        self.check_worker(case="ansi-target", expected=(1, 0, 0, 0))

    def test_exit_zero_rejects_extra_stderr(self):
        for variant in ("prefix", "suffix", "extra-newline", "missing-newline"):
            with self.subTest(variant=variant):
                self.check_worker(case="miss-" + variant)

    def test_exit_zero_requires_exact_target(self):
        for case in ("other-target", "longer-target", "other-diagnostic"):
            with self.subTest(case=case):
                self.check_worker(case=case)

    def test_exact_exit_one_confirmed_alive(self):
        self.check_worker(alive="layout", expected=(0, 1, 0, 0))

    def test_exact_exit_one_liveness_command_fails(self):
        self.check_worker(alive="layout-failure", expected=(0, 0, 1, 0))

    def test_exact_exit_one_absent_socket_skips_liveness(self):
        self.check_worker(socket_present=False, expected=(0, 0, 1, 0))

    def test_exact_exit_one_malformed_layout_is_unconfirmed(self):
        self.check_worker(alive="layout-malformed", expected=(0, 0, 1, 0))

    def test_exit_one_rejects_extra_stderr_without_liveness(self):
        for variant in ("prefix", "suffix", "extra-newline", "missing-newline"):
            with self.subTest(variant=variant):
                self.check_worker(case="absent-" + variant, expected=(0, 0, 0, 1))

    def test_exit_one_nonempty_stdout_skips_liveness(self):
        self.check_worker(case="partial-stdout", expected=(0, 0, 0, 1))


class MainTest(unittest.TestCase):
    def setUp(self):
        WorkerTest.setUp(self)
        self.cli_tmp = self.root / "cli"
        if len(os.fsencode(str(self.cli_tmp / "zfm.XXXXXX/s/contract_version_1/zfm-123456-0"))) >= 104:
            short = tempfile.TemporaryDirectory(prefix="zfc-", dir="/tmp")
            self.addCleanup(short.cleanup)
            self.cli_tmp = Path(short.name)
        self.cli_tmp.mkdir(exist_ok=True)
        self.env["TMPDIR"] = str(self.cli_tmp)
        (self.bin / "zellij").write_text(f"#!{sys.executable}\n" + textwrap.dedent('''\
            import fcntl, json, os, pathlib, socket, sys
            sys.path.insert(0, os.environ['ZELLIJ_FIXTURE_LIB'])
            from zellij_fixture_replies import case_payload, emit, emit_case
            root = pathlib.Path(os.environ['FAKE_ROOT'])
            argv = sys.argv[1:]
            event = {'argv': argv}
            reply_case = 'zfm-main-mutation-success'
            kind = payload = None
            with (root / 'lock').open('w') as lock:
                fcntl.flock(lock, fcntl.LOCK_EX)
                if os.environ.get('REFUSE_CALLS'):
                    event['unexpected'] = True
                    reply_case = 'zfm-main-refused-call'
                elif argv == ['--version']:
                    reply_case = 'zfm-main-version'
                else:
                    sockets = pathlib.Path(os.environ['ZELLIJ_SOCKET_DIR']) / 'contract_version_1'
                    private = sockets.parent.parent
                    event['root'] = str(private)
                    if len(argv) == 5 and argv[:4] == ['--layout', str(private / 'layout.kdl'), 'attach', '--create-background']:
                        assert (private / 'layout.kdl').is_file()
                        sockets.mkdir(parents=True, exist_ok=True)
                        with socket.socket(socket.AF_UNIX) as sock:
                            sock.bind(str(sockets / argv[4]))
                    elif argv == ['kill-all-sessions', '--yes']:
                        event['counts'] = {p.name: p.read_text() for p in private.glob('count.*')}
                        for path in sockets.iterdir():
                            path.unlink()
                    else:
                        assert len(list(sockets.iterdir())) == 3
                        if argv == ['list-sessions', '--no-formatting']:
                            kind, payload = 'sessions-full', sorted(p.name for p in sockets.iterdir())
                        else:
                            assert len(argv) >= 4 and argv[0] == '--session'
                            assert (sockets / argv[1]).is_socket()
                            assert argv[2:] in (['action', 'list-clients'],
                                ['action', 'list-panes', '--json', '--command'],
                                ['action', 'list-panes', '--json', '--command', '--state'])
                            kind = 'clients' if argv[3] == 'list-clients' else 'panes'
                            payload = case_payload('zfm-main-' + kind)
                with (root / 'requests.jsonl').open('a') as log:
                    log.write(json.dumps(event) + '\\n')
            if kind is not None:
                emit(kind, payload)
            emit_case(reply_case)
            '''))

    def run_cli(self, args, *, refuse_calls=False):
        if self.log.exists():
            self.log.unlink()
        env = dict(self.env)
        if refuse_calls:
            env['REFUSE_CALLS'] = '1'
        result = subprocess.run(['bash', str(SCRIPT), *args], env=env,
                                capture_output=True, text=True, timeout=10)
        calls = [json.loads(line) for line in self.log.read_text().splitlines()] if self.log.exists() else []
        return result, calls

    def check_cli_workload(self, args, queries, workers):
        result, calls = self.run_cli(args)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(calls[0]['argv'], ['--version'])
        starts = calls[1:4]
        self.assertTrue(all(call['argv'][2:4] == ['attach', '--create-background'] for call in starts))
        targets = [call['argv'][4] for call in starts]
        expected = []
        for i in range(queries):
            target = targets[(i // 4) % 3]
            expected.append([
                ['--session', target, 'action', 'list-panes', '--json', '--command'],
                ['--session', target, 'action', 'list-panes', '--json', '--command', '--state'],
                ['--session', target, 'action', 'list-clients'],
                ['list-sessions', '--no-formatting'],
            ][i % 4])
        self.assertEqual(Counter(tuple(call['argv']) for call in calls[4:-1]),
                         Counter(map(tuple, expected)))
        self.assertEqual(calls[-1]['argv'], ['kill-all-sessions', '--yes'])
        self.assertEqual(calls[-1]['counts'], {f'count.{w}': '0 0 0 0\n' for w in range(workers)})
        self.assertIn(f'queries={queries} workers={workers}\n', result.stdout)
        self.assertIn('cleanup=pass', result.stdout)
        self.assertFalse(Path(calls[-1]['root']).exists())

        return len(calls) - 5, len(calls)

    def test_cli_normalizes_decimal_queries_and_workers(self):
        for args, workers in ((['010', '01'], 1), (['10', '010'], 10)):
            with self.subTest(args=args):
                self.check_cli_workload(args, 10, workers)

    def test_cli_doubling_queries_doubles_measured_requests(self):
        measured, total = self.check_cli_workload(['10', '2'], 10, 2)
        doubled, doubled_total = self.check_cli_workload(['20', '2'], 20, 2)
        self.assertEqual((measured, doubled), (10, 2 * measured))
        self.assertEqual((total - measured, doubled_total - doubled), (5, 5))

    def test_cli_invalid_numbers_refuse_before_zellij_or_temp_root(self):
        for position, values in ((0, ['100001', '000100001', '18446744073709551617']),
                                 (1, ['17', '00017', '18446744073709551617'])):
            for value in values + ['', '0', '000', '-1', '1x']:
                args = ['1', '1']
                args[position] = value
                with self.subTest(args=args):
                    result, calls = self.run_cli(args, refuse_calls=True)
                    self.assertEqual((result.returncode, calls, list(self.cli_tmp.iterdir())), (2, [], []))

    def test_cli_rejects_extra_arguments_before_any_effect(self):
        result, calls = self.run_cli(['1', '1', 'extra'], refuse_calls=True)
        self.assertEqual((result.returncode, calls, list(self.cli_tmp.iterdir())), (2, [], []))

    def test_bounded_decimal_normalizes_and_checks_bounds(self):
        for maximum in (100000, 16):
            valid = [('1', '1'), (str(maximum), str(maximum)),
                     ('000' + str(maximum), str(maximum)), ('000010', '10')]
            invalid = ['', '0', '000', '-1', '+1', '1x', '1 0', '1\n',
                       str(maximum + 1), '000' + str(maximum + 1), '18446744073709551617']
            for value, expected in valid + [(value, None) for value in invalid]:
                with self.subTest(maximum=maximum, value=value):
                    result = subprocess.run(['bash', '-c', 'source "$1"; bounded_decimal "$2" "$3"',
                                             'test', str(SCRIPT), value, str(maximum)],
                                            env=self.env, capture_output=True, text=True, timeout=10)
                    if expected is None:
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(result.stdout, '')
                    else:
                        self.assertEqual((result.returncode, result.stdout), (0, expected + '\n'), result.stderr)


if __name__ == "__main__":
    unittest.main()
