"""The CI gate must reject successful go test commands that ran no conformance."""
import importlib.util
from pathlib import Path
import unittest
import os
import sys
import tempfile

spec = importlib.util.spec_from_file_location("native_ci", Path(__file__).with_name("native-terminal-ci.py"))
ci = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ci)


class CoverageGateTest(unittest.TestCase):
    def complete(self):
        return [{"Action": "pass", "Test": name} for name in ci.REQUIRED]

    def test_malformed_output_joins_child_before_return(self):
        with tempfile.TemporaryDirectory() as scratch:
            pidfile = Path(scratch) / "pid"
            program = "import os,time; from pathlib import Path; Path(%r).write_text(str(os.getpid())); print('invalid-json',flush=True); time.sleep(60)" % str(pidfile)
            with self.assertRaises(ValueError):
                ci.run_tests([sys.executable, "-c", program], os.environ.copy(), Path(scratch) / "events")
            with self.assertRaises(ProcessLookupError):
                os.kill(int(pidfile.read_text()), 0)

    def test_executed_required_tests(self):
        ci.require_execution(self.complete())

    def test_empty_success_is_failure(self):
        with self.assertRaises(RuntimeError):
            ci.require_execution([])

    def test_missing_native_subtest_is_failure(self):
        with self.assertRaises(RuntimeError):
            ci.require_execution(self.complete()[1:])

    def test_skipped_or_failed_test_is_failure(self):
        for action in ("skip", "fail"):
            with self.subTest(action=action), self.assertRaises(RuntimeError):
                ci.require_execution(self.complete() + [{"Action": action, "Test": "unexpected"}])


if __name__ == "__main__":
    unittest.main()
