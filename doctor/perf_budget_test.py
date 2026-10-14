#!/usr/bin/env python3
"""Behavioral timing-policy tests; no host stopwatch assertions or source parsing."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent

# Every fake process observes the same logical clock. Collectors advance it when
# they run; date only reads it, and sleep advances it without sleeping.
TOOL = r'''
import json, os, sys
from pathlib import Path
name = Path(sys.argv[0]).name
args = sys.argv[1:]
clock = Path(os.environ["FAKE_CLOCK"])
before = int(clock.read_text())
key = name
if name == "ps":
    key = {"pid=":"process_count", "comm=":"process_names", "pid=,time=":"cputimes", "pid=,etime=,rss=,comm=":"sample"}.get(args[-1], "ps")
if name == "sysctl":
    key = args[-1]
cost = int(json.loads(os.environ["FAKE_COSTS"]).get(key, 0))
if name == "sleep":
    cost = int(args[0])
clock.write_text(str(before + cost))
with open(os.environ["FAKE_CALLS"], "a") as log:
    log.write(json.dumps({"tool":name,"key":key,"args":args,"before":before,"after":before+cost})+"\n")
if name == "date":
    print(before if args == ["+%s"] else "2026-01-01T00:00:00+0000")
elif name == "sysctl":
    print({"hw.ncpu":"8", "vm.loadavg":"{ 1.0 2.0 3.0 }", "kern.memorystatus_vm_pressure_level":"1"}[key])
elif name == "top":
    print("CPU usage: 10.00% user, 5.00% sys, 85.00% idle\nWindowServer 3.1\nCPU usage: 70.00% user, 17.60% sys, 12.40% idle\nWindowServer 45.2")
elif name == "ps":
    print({"process_count":"101\n102", "process_names":"/usr/bin/pair\n/usr/bin/go", "cputimes":"101 00:01.00\n102 00:02.00", "sample":"101 01:00 1200 /usr/bin/pair\n102 01:00 1500 /usr/bin/go"}[key])
elif name == "vm_stat":
    print("Pageins: 1000.\nSwapins: 200.\nSwapouts: 300.")
elif name == "iostat":
    print("disk0\nKB/t tps MB/s\n19.53 12 0.23\n31.25 44 1.34")
elif name == "pair":
    assert args and args[0] == "hoprtt", args
    print("0.01 0.02 0.03 30")
elif name not in ("sleep", "zellij"):
    raise SystemExit("unexpected tool " + name)
'''

REPORT = '''#!/bin/sh
cat <<EOF
# pair perf capture
captured_at=2026-01-01T00:00:00+0000
window_seconds=2
load=1/2/3
cpu_idle_pct=12.40
pair_family_procs=1
build_procs=1
windowserver_cpu_pct=45.2
elapsed_seconds=${TEST_ELAPSED}
budget_seconds=${TEST_BUDGET}
## sample_a
### cputime
101\t00:01.00
### procs
101\t01:00\t1200\tpair
# end
EOF
'''


def run(script, env, *args):
    return subprocess.run(["/bin/sh", str(script), *args], env=env,
                          text=True, capture_output=True, timeout=60)


class PerfBudgetTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="pair-perf-policy-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = os.environ.copy()
        # Ambient opt-ins and collector overrides cannot change a test's policy.
        for key in list(self.env):
            if key.startswith("PAIR_PERF_"):
                del self.env[key]

    def gate(self, elapsed="7", budget="6", isolated=False, report=REPORT):
        doctor = self.root / "repo" / "doctor"
        doctor.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(HERE / "perf_test.sh", doctor / "perf_test.sh")
        (doctor / "perf.sh").write_text(report)
        env = dict(self.env, TEST_ELAPSED=elapsed, TEST_BUDGET=budget)
        if isolated:
            env["PAIR_PERF_BUDGETS"] = "1"
        return run(doctor / "perf_test.sh", env, "--budget-only")

    def test_default_accepts_controlled_overrun_and_logs_timing(self):
        result = self.gate()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("elapsed_seconds=7", result.stdout + result.stderr)
        self.assertIn("budget_seconds=6", result.stdout + result.stderr)

    def test_isolated_rejects_same_controlled_overrun(self):
        result = self.gate(isolated=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("elapsed_seconds=7", result.stdout + result.stderr)
        self.assertIn("budget_seconds=6", result.stdout + result.stderr)

    def test_isolated_accepts_boundary_and_under_budget(self):
        for elapsed in ("6", "5"):
            with self.subTest(elapsed=elapsed):
                result = self.gate(elapsed=elapsed, isolated=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertIn("elapsed_seconds=" + elapsed, result.stdout + result.stderr)
                self.assertIn("budget_seconds=6", result.stdout + result.stderr)

    def test_malformed_timing_rejected_in_both_lanes(self):
        for isolated in (False, True):
            for elapsed, budget in (("", "6"), ("oops", "6"), ("1", ""), ("1", "no"), ("-1", "6"), ("1", "-1")):
                with self.subTest(isolated=isolated, elapsed=elapsed, budget=budget):
                    result = self.gate(elapsed, budget, isolated)
                    self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_budget_only_still_rejects_malformed_report_shape(self):
        for report in (REPORT.replace("load=1/2/3\n", ""),
                       REPORT.replace("# end", "0\n# end")):
            with self.subTest(report=report):
                result = self.gate(elapsed="1", report=report)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def capture(self, costs, budget=6, reserve=2, window=2):
        tools = self.root / "tools"
        tools.mkdir()
        for name in ("date", "sysctl", "top", "ps", "vm_stat", "iostat", "sleep", "pair", "zellij"):
            tool = tools / name
            tool.write_text("#!" + sys.executable + "\n" + TOOL)
            tool.chmod(0o755)
        clock, calls = self.root / "clock", self.root / "calls"
        clock.write_text("100")
        calls.write_text("")
        env = dict(self.env, PATH=str(tools) + os.pathsep + os.defpath,
                   PAIR_HOME=str(self.root), FAKE_CLOCK=str(clock), FAKE_CALLS=str(calls),
                   FAKE_COSTS=json.dumps(costs), PAIR_PERF_BUDGET=str(budget),
                   PAIR_PERF_PROBE_RESERVE=str(reserve), PAIR_PERF_WINDOW=str(window))
        result = run(HERE / "perf.sh", env)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        rows = [json.loads(line) for line in calls.read_text().splitlines()]
        fields = dict(line.split("=", 1) for line in result.stdout.splitlines() if "=" in line)
        self.assertEqual(fields["budget_seconds"], str(budget))
        self.assertEqual(fields["elapsed_seconds"], str(int(clock.read_text()) - 100))
        self.assertIn("### cputime", result.stdout)
        self.assertIn("101\t01:00\t1200\tpair", result.stdout)
        return result.stdout, fields, rows

    def assert_probes(self, fields, calls, present):
        probes = [row for row in calls if row["tool"] == "pair"]
        self.assertEqual(len(probes), 3 if present else 0, probes)
        if present:
            self.assertEqual([p["args"] for p in probes], [
                ["hoprtt"], ["hoprtt", "-spawn", "30", "--", "/usr/bin/true"],
                ["hoprtt", "-spawn", "5", "--", "zellij", "action", "query-tab-names"]])
        for key in ("pipe_hop_ms", "fork_exec_ms", "zellij_action_ms"):
            self.assertEqual(fields[key], "0.01" if present else "n/a (budget exceeded, probe skipped)")

    def test_generous_budget_collects_once_and_keeps_all_probes(self):
        output, fields, calls = self.capture({"top": 2, "iostat": 1}, budget=20)
        names = [row["tool"] for row in calls]
        self.assertEqual(names.count("top"), 1)
        self.assertEqual(names.count("iostat"), 1)
        self.assertEqual(names.count("sleep"), 1)
        self.assertEqual(names.count("vm_stat"), 2)
        self.assertEqual(sum(row["key"] == "sample" for row in calls), 2)
        self.assertIn("## sample_b\nat_s=104", output)
        self.assertEqual(fields["cpu_idle_pct"], "12.40")
        self.assertEqual(fields["windowserver_cpu_pct"], "45.2")
        self.assertEqual(fields["mb_per_s"], "1.34")
        self.assert_probes(fields, calls, True)

    def test_reserve_boundary_sheds_window_and_disk_but_keeps_probes(self):
        output, fields, calls = self.capture({"top": 4})
        names = [row["tool"] for row in calls]
        self.assertEqual(names.count("top"), 1)
        self.assertNotIn("sleep", names)
        self.assertNotIn("iostat", names)
        self.assertEqual(sum(row["key"] == "sample" for row in calls), 1)
        self.assertIn("## sample_b\nskipped=budget reserved for probes", output)
        self.assertIn("sample window was shed", fields["swapins_per_s"])
        self.assertIn("iostat skipped", fields["mb_per_s"])
        self.assert_probes(fields, calls, True)

    def test_reserve_before_top_skips_top_and_still_probes(self):
        _, fields, calls = self.capture({"process_count": 4})
        names = [row["tool"] for row in calls]
        self.assertNotIn("top", names)
        self.assertNotIn("sleep", names)
        self.assertNotIn("iostat", names)
        self.assertIn("top skipped", fields["cpu_idle_pct"])
        self.assert_probes(fields, calls, True)

    def test_exhaustion_skips_probes_and_lower_value_work(self):
        _, fields, calls = self.capture({"top": 6})
        names = [row["tool"] for row in calls]
        self.assertEqual(names.count("top"), 1)
        self.assertNotIn("sleep", names)
        self.assertNotIn("iostat", names)
        self.assertEqual(fields["elapsed_seconds"], "6")
        self.assert_probes(fields, calls, False)


if __name__ == "__main__":
    unittest.main()
