#!/usr/bin/env python3
"""Fail closed when a Go test's audited repository-path reading scope changes.

This independent lexical tripwire deliberately includes fixture strings and
scratch paths. File hashes keep the audit small while requiring a new review
when an existing reader adds another path, alias or helper call. It is not a
Go dataflow analyzer and does not infer runtime dependency coverage.
"""
import hashlib
import json
from pathlib import Path
import re
import runpy
import os
import shutil
import subprocess
import tempfile
import unittest
import argparse
import shlex
import sys


ROOT = Path(__file__).resolve().parents[1]
INVENTORY = ROOT / "tests/affected-go-readers.json"
READER = re.compile(
    r'\bls-files\b|'
    r'(?:nvim|zellij|doctor|scripts|bin|tests|testdata|terminfo|atlas|construct|workshop)/'
    r'|"(?:nvim|zellij|doctor|scripts|bin|tests|testdata|terminfo|atlas|construct|workshop)"'
    r'|(?:filepath\.(?:Walk|WalkDir|Glob)|parser\.(?:ParseFile|ParseDir))\s*\('
    r'|(?:reporoot\.|\b(?:RepoRoot|repoRoot|repositoryRoot)\b)'
    r'|"\.\.(?:/|"\s*,)'
    r'|(?:os|ioutil)\.(?:ReadFile|Open|ReadDir)\s*\(\s*"(?:[^"\n]*\.(?:go|lua|sh|kdl|py)|\.)"'
)
# These APIs compile/read the real checkout and runtime assets even when the
# calling test contains no literal repository path. They are not source scanners.
NATIVE_HELPER_READER = re.compile(r'\b(?:testcouch\.(?:Build|New)|runtimeEnvironment)\s*\(')
TEST = re.compile(r'^func (Test\w+)\(t \*testing\.T\)', re.MULTILINE)
DISPOSITIONS = {"fixed_scanner", "non_go_full_fallback", "local_or_literal_fixture"}


def tracked_readers(root):
    paths = subprocess.check_output(["git", "ls-files", "-z", "--cached"], cwd=root)
    readers = {}
    for encoded in paths.split(b"\0"):
        name = encoded.decode("utf-8", errors="surrogateescape")
        if not name.endswith("_test.go"):
            continue
        path = root / name
        if not path.is_file():
            continue
        body = path.read_bytes()
        if READER.search(body.decode("utf-8")) or NATIVE_HELPER_READER.search(body.decode("utf-8")):
            readers[name] = body
    return readers


def repository_source_scan(body):
    # A high-signal rule independent of the audited disposition. Even an
    # accidental local-fixture classification cannot waive source walkers.
    outside = re.search(r'"\.\.(?:/|"\s*,)|\b(?:repoRoot|repositoryRoot|moduleRoot|reporoot|RepoRoot)\b', body)
    go_source = re.search(r'\.go"', body)
    reader = re.search(r'(?:filepath\.(?:Walk|WalkDir|Glob)|parser\.(?:ParseFile|ParseDir)|os\.(?:ReadFile|ReadDir|Open))\s*\(', body)
    direct_parent_read = re.search(
        r'os\.(?:ReadFile|ReadDir|Open)\s*\(\s*(?:filepath.Join\(\s*)?"\.\.(?:/|"\s*,)', body)
    return 'ls-files' in body or bool(direct_parent_read) or bool(outside and go_source and reader)


def audit(root, inventory, fixed_tests, actual=None):
    errors = []
    if inventory.get("version") != 1 or not isinstance(inventory.get("readers"), dict):
        return ["invalid Go reader inventory schema"]
    expected = {}
    for disposition, entries in inventory["readers"].items():
        if disposition not in DISPOSITIONS or not isinstance(entries, dict):
            errors.append("invalid reader disposition: " + disposition)
            continue
        for name, digest in entries.items():
            if name in expected:
                errors.append("duplicate reader: " + name)
            if not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest):
                errors.append("invalid reader digest: " + name)
            expected[name] = (disposition, digest)
    actual = tracked_readers(root) if actual is None else actual
    for name in sorted(expected.keys() - actual.keys()):
        errors.append("stale reader: " + name)
    for name, body in actual.items():
        if name not in expected:
            errors.append("unmapped reader: " + name + "; audit repository inputs before admitting its hash")
            continue
        disposition, digest = expected[name]
        if hashlib.sha256(body).hexdigest() != digest:
            errors.append("changed reader: " + name + "; re-audit inputs and preflight/fallback ownership")
        if NATIVE_HELPER_READER.search(body.decode("utf-8")) and disposition != "non_go_full_fallback":
            errors.append("native helper API requires repository-input fallback classification: " + name)
        if disposition == "fixed_scanner" or repository_source_scan(body.decode("utf-8")):
            tests = set(TEST.findall(body.decode("utf-8")))
            registered = set(fixed_tests.get(str(Path(name).parent), ()))
            if not registered or not tests <= registered:
                errors.append("fixed preflight omits tests from " + name + ": " + ", ".join(sorted(tests - registered)))
    return errors


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate inventory key: " + key)
        result[key] = value
    return result


def command_line():
    parser = argparse.ArgumentParser(description="Audit reader inputs before recording one explicit disposition.")
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--record", metavar="FILE")
    mode.add_argument("--remove", metavar="FILE")
    parser.add_argument("--disposition", choices=sorted(DISPOSITIONS))
    args = parser.parse_args()
    try:
        inventory = json.loads(INVENTORY.read_text(), object_pairs_hook=unique_object)
        if inventory.get("version") != 1 or not isinstance(inventory.get("readers"), dict):
            raise ValueError("invalid Go reader inventory schema")
        fixed = runpy.run_path(str(ROOT / "scripts/test-affected.py"))["SOURCE_GUARDS"]
        actual = tracked_readers(ROOT)
        known = {name: disposition for disposition in DISPOSITIONS
                 for name in inventory["readers"].get(disposition, {})}
        if args.record:
            if args.record not in actual or not args.disposition:
                raise ValueError("record requires a tracked reader and explicit --disposition")
            body = actual[args.record].decode("utf-8")
            source = repository_source_scan(body)
            registered = set(fixed.get(str(Path(args.record).parent), ()))
            if source and args.disposition != "fixed_scanner":
                raise ValueError("repository source reader requires fixed_scanner")
            if (source or args.disposition == "fixed_scanner") and (
                    not registered or not set(TEST.findall(body)) <= registered):
                raise ValueError("register every reader test in SOURCE_GUARDS before recording")
            for disposition in DISPOSITIONS:
                inventory["readers"].get(disposition, {}).pop(args.record, None)
            inventory["readers"].setdefault(args.disposition, {})[args.record] = hashlib.sha256(actual[args.record]).hexdigest()
        elif args.remove:
            if args.remove not in known or args.remove in actual:
                raise ValueError("remove requires an audited reader that is no longer present")
            inventory["readers"][known[args.remove]].pop(args.remove)
        else:
            errors = audit(ROOT, inventory, fixed, actual)
            for error in errors:
                print(error, file=sys.stderr)
                kind, _, rest = error.partition(": ")
                name = rest.split(";", 1)[0]
                command = "python3 -B tests/affected-go-readers-test.py "
                if kind == "stale reader":
                    print("After auditing removal: " + command + "--remove " + shlex.quote(name), file=sys.stderr)
                elif kind in ("changed reader", "unmapped reader"):
                    disposition = "fixed_scanner" if repository_source_scan(actual[name].decode()) else known.get(name, "non_go_full_fallback")
                    print("After auditing inputs: " + command + "--record " + shlex.quote(name)
                          + " --disposition " + disposition, file=sys.stderr)
            return bool(errors)
        INVENTORY.write_text(json.dumps(inventory, indent=2) + "\n")
        return 0
    except (OSError, ValueError, KeyError, TypeError) as error:
        print("reader inventory: " + str(error), file=sys.stderr)
        return 1


class FastInventoryCLI(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="pair-fast-inventory-")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        (self.root / "tests").mkdir()
        (self.root / "scripts").mkdir()
        shutil.copyfile(__file__, self.root / "tests/affected-go-readers-test.py")
        (self.root / "scripts/test-affected.py").write_text("SOURCE_GUARDS = {}\n")
        subprocess.run(["git", "init", "-q", str(self.root)], check=True)
        self.name = "cmd/leaf/read_test.go"
        self.reader = self.root / self.name
        self.reader.parent.mkdir(parents=True)
        self.reader.write_text('os.ReadFile("own.go")')
        subprocess.run(["git", "add", self.name], cwd=self.root, check=True)
        self.inventory = self.root / "tests/affected-go-readers.json"
        self.extra = {"future": {"integer": 123456789012345678901234567890}}
        self.inventory.write_text(json.dumps(dict(self.extra, version=1, readers={
            "local_or_literal_fixture": {self.name: hashlib.sha256(self.reader.read_bytes()).hexdigest()}
        })))
        tools = self.root / "tools"
        tools.mkdir()
        self.tool_record = self.root / "unexpected-tool"
        for name in ("go", "make"):
            tool = tools / name
            tool.write_text('#!/bin/sh\nprintf called >> "$INVENTORY_TOOL_RECORD"\nexit 89\n')
            tool.chmod(0o755)
        self.env = dict(os.environ, PATH=str(tools) + os.pathsep + os.environ["PATH"],
                        INVENTORY_TOOL_RECORD=str(self.tool_record))

    def cli(self, *args):
        return subprocess.run([os.sys.executable, "-B", "tests/affected-go-readers-test.py", *args],
                              cwd=self.root, env=self.env, text=True, capture_output=True)

    def test_fast_check_names_exact_repair_and_preserves_unknown_fields(self):
        self.assertEqual(self.cli("--check").returncode, 0)
        self.reader.write_text(self.reader.read_text() + "\n// audited change\n")
        result = self.cli("--check")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("changed reader: " + self.name, result.stderr)
        self.assertIn("python3 -B tests/affected-go-readers-test.py --record " + self.name
                      + " --disposition local_or_literal_fixture", result.stderr)
        result = self.cli("--record", self.name, "--disposition", "local_or_literal_fixture")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.cli("--check").returncode, 0)
        self.assertEqual(json.loads(self.inventory.read_text())["future"], self.extra["future"])
        self.assertFalse(self.tool_record.exists(), "fast inventory mode invoked Go or Make")

    def test_record_refuses_untracked_and_scanner_waivers_without_writing(self):
        before = self.inventory.read_bytes()
        result = self.cli("--record", "cmd/untracked_test.go", "--disposition", "fixed_scanner")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("tracked reader", result.stderr)
        self.reader.write_text('func TestScan(t *testing.T) { os.ReadFile("../source.go") }')
        for disposition in ("local_or_literal_fixture", "non_go_full_fallback", "fixed_scanner"):
            result = self.cli("--record", self.name, "--disposition", disposition)
            self.assertNotEqual(result.returncode, 0, disposition)
            self.assertIn("SOURCE_GUARDS" if disposition == "fixed_scanner" else "requires fixed_scanner", result.stderr)
            self.assertEqual(self.inventory.read_bytes(), before)

    def test_removed_reader_has_exact_removal_command(self):
        self.assertNotEqual(self.cli("--remove", self.name).returncode, 0)
        self.reader.unlink()
        result = self.cli("--check")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("--remove " + self.name, result.stderr)
        self.assertEqual(self.cli("--remove", self.name).returncode, 0)
        self.assertEqual(self.cli("--check").returncode, 0)

    def test_duplicate_keys_refuse_without_rewrite(self):
        self.inventory.write_text('{"version":1,"version":1,"readers":{}}')
        before = self.inventory.read_bytes()
        result = self.cli("--record", self.name, "--disposition", "local_or_literal_fixture")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate inventory key", result.stderr)
        self.assertEqual(self.inventory.read_bytes(), before)

    def test_changed_source_reader_prints_executable_required_disposition(self):
        old_digest = hashlib.sha256(self.reader.read_bytes()).hexdigest()
        (self.root / "scripts/test-affected.py").write_text("SOURCE_GUARDS = {'cmd/leaf': ('TestScan',)}\n")
        self.reader.write_text('func TestScan(t *testing.T) { os.ReadFile("../source.go") }')
        for previous in ("local_or_literal_fixture", "non_go_full_fallback"):
            with self.subTest(previous=previous):
                self.inventory.write_text(json.dumps({"version": 1, "readers": {
                    previous: {self.name: old_digest}
                }}))
                result = self.cli("--check")
                self.assertNotEqual(result.returncode, 0)
                command = next(line.removeprefix("After auditing inputs: ")
                               for line in result.stderr.splitlines() if line.startswith("After auditing inputs: "))
                self.assertIn("--disposition fixed_scanner", command)
                recorded = self.cli(*shlex.split(command)[3:])
                self.assertEqual(recorded.returncode, 0, recorded.stderr)
                self.assertEqual(self.cli("--check").returncode, 0)


class ReaderInventoryTest(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="pair-go-reader-guard-")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        subprocess.run(["git", "init", "-q", str(self.root)], check=True)

    def write(self, name, body, tracked=True):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(body)
        if tracked:
            subprocess.run(["git", "add", "--", name], cwd=self.root, check=True)

    def inventory(self, disposition="non_go_full_fallback"):
        return {"version": 1, "readers": {disposition: {
            name: hashlib.sha256(body).hexdigest()
            for name, body in tracked_readers(self.root).items()
        }}}

    def test_new_relative_and_joined_readers_refuse(self):
        for body in (
            'os.ReadFile("../../../nvim/new.lua")',
            'os.ReadFile(filepath.Join(root, "nvim", "new.lua"))',
            'parser.ParseFile(fset, filepath.Join("..", sibling), nil, 0)',
            'filepath.Glob("*.go")',
            'os.ReadFile("own.go")',
            'root := reporoot.Current(); os.ReadFile(path)',
            'root := repoRoot(t); os.ReadFile(path)',
            'root := repositoryRoot(t); os.ReadFile(path)',
            'root := fixture.RepoRoot; os.ReadFile(path)',
        ):
            with self.subTest(body=body):
                self.write("cmd/leaf/new_test.go", body)
                errors = audit(self.root, {"version": 1, "readers": {}}, {})
                self.assertTrue(any("unmapped reader" in error for error in errors), errors)

    def test_changed_and_deleted_audited_file_refuse(self):
        self.write("cmd/leaf/read_test.go", 'os.ReadFile(filepath.Join(root, "nvim", "init.lua"))')
        inventory = self.inventory()
        self.assertEqual(audit(self.root, inventory, {}), [])
        self.write("cmd/leaf/read_test.go", 'os.ReadFile(filepath.Join(root, "zellij", "config.kdl"))')
        self.assertTrue(any("changed reader" in error for error in audit(self.root, inventory, {})))
        (self.root / "cmd/leaf/read_test.go").unlink()
        self.assertTrue(any("stale reader" in error for error in audit(self.root, inventory, {})))

    def test_untracked_and_nonreader_files_are_outside_inventory(self):
        self.write("cmd/leaf/untracked_test.go", 'os.ReadFile(filepath.Join(root, "nvim", "init.lua"))', False)
        self.write("cmd/leaf/pure_test.go", 'package leaf\nfunc TestPure() {}')
        self.assertEqual(audit(self.root, {"version": 1, "readers": {}}, {}), [])

    def test_repository_source_walk_cannot_be_waived_as_local(self):
        for source in (
            'root := filepath.Join("..", "..", ".."); filepath.WalkDir(root, scan); _ = ".go"',
            'filepath.Walk("../../../cmd", scan); _ = ".go"',
            'os.ReadDir(filepath.Join("..", "sibling"))',
            'os.ReadFile("../sibling/source.go")',
            'exec.Command("git", "ls-files").Output()',
        ):
            with self.subTest(source=source):
                self.write("cmd/scan/read_test.go", 'package scan\nfunc TestScan(t *testing.T) { ' + source + ' }\n')
                inventory = self.inventory("local_or_literal_fixture")
                self.assertTrue(any("fixed preflight" in error for error in audit(self.root, inventory, {})))
                self.assertEqual(audit(self.root, inventory, {"cmd/scan": ("TestScan",)}), [])

    def test_fixed_scanner_requires_every_registered_test(self):
        self.write("cmd/scan/read_test.go", 'package scan\nfunc TestScan(t *testing.T) { filepath.Glob("*.go") }\n')
        inventory = self.inventory("fixed_scanner")
        self.assertTrue(any("fixed preflight" in error for error in audit(self.root, inventory, {})))
        self.assertEqual(audit(self.root, inventory, {"cmd/scan": ("TestScan",)}), [])

    def test_duplicate_or_invalid_disposition_refuses(self):
        self.write("cmd/leaf/read_test.go", 'os.ReadFile("own.go")')
        inventory = self.inventory()
        inventory["readers"]["unknown"] = dict(inventory["readers"]["non_go_full_fallback"])
        self.assertTrue(audit(self.root, inventory, {}))
        inventory = self.inventory()
        inventory["readers"]["local_or_literal_fixture"] = dict(inventory["readers"]["non_go_full_fallback"])
        self.assertTrue(any("duplicate reader" in error for error in audit(self.root, inventory, {})))

    def test_seeded_cross_package_violation_stops_actual_affected_run(self):
        native = self.root / "native"
        subprocess.run(["git", "worktree", "add", "--detach", "--quiet", str(native), "HEAD"],
                       cwd=ROOT, check=True, capture_output=True)
        self.addCleanup(subprocess.run, ["git", "worktree", "remove", "--force", str(native)],
                        cwd=ROOT, check=True, capture_output=True)
        # The production runner is copied so this regression also works before
        # its correction has been committed during the red/green cycle.
        shutil.copyfile(ROOT / "scripts/test-affected.py", native / "scripts/test-affected.py")
        source = native / "cmd/internal/slugcmd/slug.go"
        source.write_text(source.read_text() + "\n// TreeSummary is the active session model.\n// TestOnlyTeardownProof( must stay test-only.\n")
        env = dict(os.environ, SDLC_AFFECTED_GO_PKGS="github.com/xianxu/pair/cmd/internal/slugcmd",
                   SDLC_AFFECTED_GROUPS="build", PAIR_TEST_BINARY="")
        result = subprocess.run(["make", "test-affected"], cwd=native, env=env,
                                text=True, capture_output=True)
        output = result.stdout + result.stderr
        self.assertNotEqual(result.returncode, 0, "seeded cross-package violation escaped affected execution")
        self.assertIn("--- FAIL: TestDeletedVocabularyIsNotDescribedAsLive", output)
        self.assertIn("--- FAIL: TestOnlyTeardownProofHasNoProductionCaller", output)
        self.assertIn("cmd/internal/slugcmd/slug.go", output)
        self.assertNotIn("python3 -B tests/build-go-atomic-test.py", output,
                         "selected groups ran after the failed preflight")

    def test_native_helper_api_requires_repository_input_classification(self):
        for helper in ("testcouch.Build", "testcouch.New", "runtimeEnvironment"):
            with self.subTest(helper=helper):
                self.write("cmd/scan/scan_test.go",
                    "package scan\nfunc TestScan(t *testing.T) { " + helper + "(t) }\n")
                self.assertTrue(any("unmapped reader" in error
                                    for error in audit(self.root, {"version": 1, "readers": {}}, {})))
                self.assertEqual(audit(self.root, self.inventory(), {}), [])
                errors = audit(self.root, self.inventory("local_or_literal_fixture"), {})
                self.assertTrue(any("native helper API" in error for error in errors), errors)

    def test_repository_inventory_and_fixed_preflight(self):
        inventory = json.loads(INVENTORY.read_text())
        runner = runpy.run_path(str(ROOT / "scripts/test-affected.py"))
        self.assertIn("SOURCE_GUARDS", runner, "wire the fixed scanner registry before enabling this guard")
        errors = audit(ROOT, inventory, runner["SOURCE_GUARDS"])
        self.assertEqual(errors, [], "\n".join(errors))
        weakened = dict(runner["SOURCE_GUARDS"])
        weakened.pop("cmd/internal/termcmd", None)
        self.assertTrue(any(
            "fixed preflight omits tests from cmd/internal/termcmd/doccomment_test.go" in error
            for error in audit(ROOT, inventory, weakened)
        ), "removing the actual sibling-source scanner must refuse")


if __name__ == "__main__":
    if any(arg in ("--check", "--record", "--remove") for arg in sys.argv[1:]):
        sys.exit(command_line())
    unittest.main(verbosity=2)
