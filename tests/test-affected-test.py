#!/usr/bin/env python3
"""Exercise the selected runner through Make and stateful child executables."""
import importlib.util
import fnmatch
import json
import os
import re
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
PREFIX = "github.com/xianxu/pair/"
RUNTIME = PREFIX + "cmd/internal/runtimebundle"


def runner():
    spec = importlib.util.spec_from_file_location("affected_runner", ROOT / "scripts/test-affected.py")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class PureSelection(unittest.TestCase):
    def test_exact_argv_and_lua_once(self):
        m = runner()
        for packages in ((PREFIX + "cmd/internal/model",),
                         (PREFIX + "cmd/internal/model", PREFIX + "cmd/internal/slugcmd")):
            self.assertEqual(m.go_test_argv(m.parse_selection(" \n".join(packages), "")),
                             ["go", "test", *packages, "-count=1", "-timeout", "20m"])
        for text in ("", RUNTIME):
            self.assertEqual(m.go_test_argv(m.parse_selection(text, "lua")),
                             ["go", "test", RUNTIME, "-count=1", "-timeout", "20m"])
        self.assertEqual(m.go_test_argv(m.parse_selection("", "build")), [])

    def test_invalid_selection(self):
        m = runner()
        for packages, groups in (("", ""), ("./...", ""), ("-run", ""),
                                 ("example.org/foreign", ""), (PREFIX + "../escape", ""),
                                 (PREFIX + "cmd;touch", ""), (PREFIX + "cmd//model", ""),
                                 (PREFIX + "cmd/anything...", ""),
                                 (PREFIX + "cmd " + PREFIX + "cmd", ""),
                                 ("", "lua lua"), ("", "unknown"), ("", "$(touch x)")):
            with self.subTest(packages=packages, groups=groups):
                with self.assertRaises(ValueError):
                    m.parse_selection(packages, groups)


RECORDER = '''#!/usr/bin/env python3
import json, os, pathlib, sys
kind = pathlib.Path(sys.argv[0]).name
row = {"kind": kind, "args": sys.argv[1:], "binary": os.environ.get("PAIR_TEST_BINARY", ""),
       "identity": os.environ.get("PAIR_SESSION_ID"), "knob": os.environ.get("PAIR_TEST_SENTINEL")}
with open(os.environ["AFFECTED_RECORD"], "a") as f:
    f.write(json.dumps(row) + "\\n")
print("child-output:" + kind, flush=True)
if os.environ.get("AFFECTED_FAIL") == kind + ":" + (sys.argv[1] if len(sys.argv) > 1 else ""):
    sys.exit(17)
if kind == "go" and len(sys.argv) > 1 and sys.argv[1] == "build":
    dest = pathlib.Path(sys.argv[sys.argv.index("-o")+1])
    dest.write_text("#!/bin/sh\\nexit 0\\n")
    dest.chmod(0o755)
'''


class MakeEntrypoint(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="pair-affected-test-")
        self.addCleanup(self.scratch.cleanup)
        self.directory = Path(self.scratch.name)
        self.record = self.directory / "record.jsonl"
        for name in ("go", "make"):
            path = self.directory / name
            path.write_text(RECORDER)
            path.chmod(0o755)

    def run_selection(self, packages="", groups="", fail=""):
        env = dict(os.environ, PATH=str(self.directory) + os.pathsep + os.environ["PATH"],
                   GOFLAGS="", PAIR_TEST_BINARY="", PAIR_SESSION_ID="poison-session",
                   PAIR_TEST_SENTINEL="preserved", SDLC_AFFECTED_GO_PKGS=packages,
                   SDLC_AFFECTED_GROUPS=groups, AFFECTED_RECORD=str(self.record), AFFECTED_FAIL=fail)
        for name in ("MAKEFLAGS", "MFLAGS", "MAKEOVERRIDES", "MAKELEVEL"):
            env.pop(name, None)
        # Absolute native Make drives the real recipe; only its children are doubled.
        import shutil
        native_make = shutil.which("make")
        result = subprocess.run([native_make, "--no-print-directory", "-f", "Makefile.local",
                                 "test-affected", "MAKE=" + str(self.directory / "make")],
                                cwd=ROOT, env=env, text=True, capture_output=True)
        rows = [json.loads(line) for line in self.record.read_text().splitlines()] if self.record.exists() else []
        return result, rows

    def test_exact_packages_no_private_build(self):
        for n in (1, 2):
            with self.subTest(n=n):
                self.record.unlink(missing_ok=True)
                packages = [PREFIX + "cmd/internal/model", PREFIX + "cmd/internal/slugcmd"][:n]
                result, rows = self.run_selection(" ".join(packages))
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                go = [r for r in rows if r["kind"] == "go"]
                self.assertEqual([r["args"] for r in go],
                                 [["test", *packages, "-count=1", "-timeout", "20m"]])
                self.assertEqual([r["args"] for r in rows if r["kind"] == "make"], [["test-make-env"], ["runtimebundle-generate"]])
                self.assertTrue(all(r["identity"] is None and r["knob"] == "preserved" for r in rows))

    def test_groups_share_one_private_binary(self):
        for groups in ("build", "build shell"):
            with self.subTest(groups=groups):
                self.record.unlink(missing_ok=True)
                result, rows = self.run_selection(groups=groups)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertEqual([r["args"][0] for r in rows if r["kind"] == "go"], ["build"])
                group_calls = [r for r in rows if r["kind"] == "make" and r["binary"]]
                self.assertEqual(len(group_calls), 1)
                self.assertEqual(group_calls[0]["args"], ["test-affected-" + g for g in groups.split()])
                self.assertFalse(Path(group_calls[0]["binary"]).exists(), "private binary leaked")

    def test_lua_regenerates_and_tests_runtimebundle_once(self):
        result, rows = self.run_selection(RUNTIME, "lua")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        go_tests = [r for r in rows if r["kind"] == "go" and r["args"][0] == "test"]
        self.assertEqual([r["args"] for r in go_tests], [["test", RUNTIME, "-count=1", "-timeout", "20m"]])
        self.assertEqual(rows[0]["args"], ["test-make-env"])
        self.assertEqual(rows[1]["args"], ["runtimebundle-generate"])

    def test_invalid_before_child_effects(self):
        for packages, groups in (("", ""), ("./...", "lua"), ("", "unknown"), ("", "lua lua")):
            with self.subTest(packages=packages, groups=groups):
                result, rows = self.run_selection(packages, groups)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("invalid affected selection", result.stderr)
                self.assertEqual(rows, [])

    def test_child_failure_stops_and_preserves_output(self):
        for failure in ("make:runtimebundle-generate", "go:test", "go:build", "make:test-affected-build"):
            with self.subTest(failure=failure):
                self.record.unlink(missing_ok=True)
                result, rows = self.run_selection(PREFIX + "cmd/internal/model", "build", failure)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertIn("child-output:", result.stdout)
                self.assertEqual(rows[-1]["kind"] + ":" + rows[-1]["args"][0], failure)

    def test_recipe_ownership(self):
        result = subprocess.run(["make", "-prRn", "-f", "Makefile.local", "test", "PAIR_TEST_BINARY=/unused", "GOFLAGS="],
                                cwd=ROOT, text=True, capture_output=True)
        self.assertIn(result.returncode, (0, 1), result.stderr)
        targets = {}
        for line in result.stdout.splitlines():
            if line.startswith("test:") or line.startswith("test-affected-") and ":" in line:
                name, prerequisites = line.split(":", 1)
                targets[name] = prerequisites.split()
        groups = {}
        for name in ("lua", "review", "doctor", "build", "shell"):
            self.assertIn("test-affected-" + name, targets)
            groups[name] = targets["test-affected-" + name]
        flat = sum(groups.values(), [])
        self.assertEqual(len(flat), len(set(flat)), "a recipe belongs to two groups")
        expected = set(targets["test"]) - {"runtimebundle-generate", "test-pair-title", "test-affected-runner", "test-affected-config"}
        self.assertEqual(set(flat), expected)
        config = json.loads((ROOT / ".sdlc/affected.json").read_text())
        self.assertEqual({g["name"] for g in config["groups"]}, set(groups))
        consumers = json.loads((ROOT / "tests/affected-consumers.json").read_text())
        self.assertEqual(set(consumers), set(groups))
        for name, recipes in groups.items():
            self.assertEqual(set(consumers[name]), set(recipes), f"{name} command audit is stale")
        source = (ROOT / "Makefile.local").read_text()
        for group in config["groups"]:
            for target in groups[group["name"]]:
                recipe = re.search(r"^" + target + r":[^\n]*\n((?:\t[^\n]*\n)+)", source, re.M)
                self.assertIsNotNone(recipe, target)
                files = re.findall(r"(?:bash|sh|python3(?: -B)?|nvim -l) ((?:tests|doctor|nvim)/[^\s]+)", recipe[1])
                for path in files:
                    self.assertTrue(any(fnmatch.fnmatchcase(path, pattern) for pattern in group["paths"]),
                                    f"{group['name']} does not claim {target}'s {path}")


if __name__ == "__main__":
    unittest.main()
