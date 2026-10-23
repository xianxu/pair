"""The CI gate must reject successful go test commands that ran no conformance."""
import contextlib
import io
import json
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch
import os
import sys
import tempfile

spec = importlib.util.spec_from_file_location("native_ci", Path(__file__).with_name("native-terminal-ci.py"))
ci = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ci)


class CoverageGateTest(unittest.TestCase):
    def setUp(self):
        self.enterContext(contextlib.redirect_stdout(io.StringIO()))

    def complete(self):
        events = []
        for package, names in ci.REQUIRED_BY_PACKAGE.items():
            events.append({"Action": "start", "Package": package})
            for name in sorted(names):
                events.extend([{"Action": "run", "Package": package, "Test": name},
                               {"Action": "pass", "Package": package, "Test": name}])
            events.append({"Action": "pass", "Package": package})
        return events

    def test_malformed_output_joins_child_before_return(self):
        with tempfile.TemporaryDirectory() as scratch:
            pidfile = Path(scratch) / "pid"
            program = "import os,time; from pathlib import Path; Path(%r).write_text(str(os.getpid())); print('invalid-json',flush=True); time.sleep(60)" % str(pidfile)
            with self.assertRaises(ValueError):
                ci.run_tests([sys.executable, "-c", program], os.environ.copy(), Path(scratch) / "events")
            with self.assertRaises(ProcessLookupError):
                os.kill(int(pidfile.read_text()), 0)

    def test_failed_run_publishes_complete_artifacts_before_scratch_removal(self):
        captured = io.StringIO()
        with tempfile.TemporaryDirectory() as scratch:
            root = Path(scratch)
            artifacts = root / "artifacts" / "test" / "node-oracle-owned"
            artifacts.mkdir(parents=True)
            request = json.dumps({"Chunks": ["界" * 6000, "REQUEST-END"]})
            report = json.dumps({"header": {}, "tail": "REPORT-END"})
            for name, body in {"request.json": request, "node-report.json": report,
                               "stderr.txt": "snapshot", "stdout.raw": "partial"}.items():
                (artifacts / name).write_text(body)
            (root / "unrelated-secret").write_text("OUTSIDE-ARTIFACTS")
            program = "import sys; sys.stdout.write(%r); raise SystemExit(7)" % "".join(json.dumps(event) + "\n" for event in self.complete())
            with contextlib.redirect_stderr(captured), self.assertRaisesRegex(RuntimeError, "exited 7"):
                ci.qualify([sys.executable, "-c", program], os.environ.copy(), root)
        output = captured.getvalue()
        self.assertTrue(request in output, "complete request missing from failure log")
        self.assertIn(report, output)
        self.assertNotIn("OUTSIDE-ARTIFACTS", output)
        self.assertFalse(root.exists())

    def test_validation_failure_publishes_but_success_does_not(self):
        for events, fails in [([], True), (self.complete(), False)]:
            with self.subTest(fails=fails), tempfile.TemporaryDirectory() as scratch:
                root = Path(scratch)
                artifact = root / "artifacts" / "node-oracle-owned"
                artifact.mkdir(parents=True)
                (artifact / "request.json").write_text("RETAIN-ONLY-ON-FAILURE")
                command = [sys.executable, "-c", "import sys; sys.stdout.write(%r)" % "".join(json.dumps(event) + "\n" for event in events)]
                output = io.StringIO()
                with contextlib.redirect_stderr(output):
                    if fails:
                        with self.assertRaises(RuntimeError):
                            ci.qualify(command, os.environ.copy(), root)
                    else:
                        ci.qualify(command, os.environ.copy(), root)
                self.assertEqual("RETAIN-ONLY-ON-FAILURE" in output.getvalue(), fails)

    def test_pass_without_run_or_package_identity_is_failure(self):
        with self.assertRaises((ValueError, RuntimeError)):
            ci.require_execution([{"Action": "pass", "Test": name} for name in ci.REQUIRED])

    def test_production_stream_parses_each_event_once(self):
        events = self.complete()
        command = [sys.executable, "-c", "import sys; sys.stdout.write(%r)" % "".join(json.dumps(event) + "\n" for event in events)]
        with tempfile.TemporaryDirectory() as scratch:
            with patch.object(ci._validator.json, "loads", wraps=json.loads) as loads:
                ci.qualify(command, os.environ.copy(), Path(scratch))
            self.assertEqual(loads.call_count, len(events))

    def test_executed_required_tests(self):
        ci.require_execution(self.complete())

    def test_empty_success_is_failure(self):
        with self.assertRaises(RuntimeError):
            ci.require_execution([])

    def test_missing_native_subtest_is_failure(self):
        with self.assertRaises(RuntimeError):
            ci.require_execution(self.complete()[1:])

    def test_terminal_recreation_must_actually_execute(self):
        prior_tests = [e for e in self.complete() if e.get("Test") != "TestTerminalRecoveryZellijConformance"]
        with self.assertRaisesRegex(RuntimeError, "TestTerminalRecoveryZellijConformance"):
            ci.require_execution(prior_tests)

    def test_skipped_or_failed_test_is_failure(self):
        for action in ("skip", "fail"):
            with self.subTest(action=action), self.assertRaises(RuntimeError):
                ci.require_execution(self.complete() + [{"Action": action, "Test": "unexpected"}])


if __name__ == "__main__":
    unittest.main()
