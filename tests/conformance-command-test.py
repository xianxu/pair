"""Stateful VM-command fixture: literal argv, one invocation, exact failure."""
import importlib.util
import json
import os
import re
import shlex
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('conformance_vm', Path(__file__).with_name('conformance-vm.py'))
vm = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(vm)


class CommandTest(unittest.TestCase):
    def test_host_invokes_generic_train_runner_once_and_preserves_exit(self):
        for code in (0, 37):
            with self.subTest(code=code), tempfile.TemporaryDirectory() as scratch:
                root = Path(scratch) / 'checkout ; literal'
                root.mkdir()
                binary = Path(scratch) / 'pair fixture'
                record = Path(scratch) / 'calls'
                binary.write_text('#!/usr/bin/env python3\nimport json,sys\nfrom pathlib import Path\np=Path(' + repr(str(record)) + ')\nwith p.open("a") as f: f.write(json.dumps(sys.argv[1:])+"\\n")\nsys.exit(' + str(code) + ')\n')
                binary.chmod(0o700)
                artifacts = Path(scratch) / 'nightly ; result'
                env = dict(os.environ, PAIR_VM_ARTIFACTS_DIR=str(artifacts), OPENAI_API_KEY='host-sentinel')
                self.assertEqual(vm.host_main(root, binary, env), code)
                calls = [json.loads(line) for line in record.read_text().splitlines()]
                self.assertEqual(calls, [['vm', 'run', '--checkout', str(root), '--lane', 'train', '--timeout', '20m', '--artifacts', str(artifacts), '--', 'python3', 'tests/conformance-vm.py', 'guest']])
                self.assertFalse(artifacts.exists(), 'wrapper must leave destination creation to the runner')

    def test_bad_artifact_path_refuses_before_child(self):
        with tempfile.TemporaryDirectory() as scratch:
            with self.assertRaises(ValueError):
                vm.host_main(Path(scratch), Path(scratch) / 'does-not-exist', {'PAIR_VM_ARTIFACTS_DIR':'relative'})

def hosted_selection(source):
    block = re.search(r'^test-couch-zellij-live:\n((?:\t[^\n]*\n)+)', source, re.M)
    if block is None:
        raise AssertionError('missing hosted lifecycle target')
    selected, excluded = {}, {}
    for row in block[1].splitlines():
        args = shlex.split(row.replace('$$', '$'))
        packages = [arg for arg in args if arg.startswith('./')]
        if args.count('-run') != 1 or args.count('-skip') > 1:
            raise AssertionError('ambiguous hosted Go flags')
        pattern = args[args.index('-run') + 1]
        union = re.fullmatch(r'\^\(([^()]+)\)\$', pattern) or re.fullmatch(r'\^([^()|]+)\$', pattern)
        if union is None:
            raise AssertionError('hosted selection must enumerate literal test names')
        names = union[1].split('|')
        if len(set(names)) != len(names) or any(not re.fullmatch(r'Test[A-Za-z0-9_]+', name) for name in names):
            raise AssertionError('ambiguous hosted selection')
        skip = args[args.index('-skip') + 1] if '-skip' in args else None
        for package in packages:
            key = 'github.com/xianxu/pair/' + package[2:]
            if key in selected:
                raise AssertionError('duplicate hosted lifecycle package')
            selected[key] = set(names)
            excluded[key] = skip
    return selected, excluded


def assert_same_selection(source, required, skip):
    hosted, excluded = hosted_selection(source)
    expected = {package: {name for name in names if '/' not in name} for package, names in required.items()}
    skip_parent = skip.split('/', 1)[0]
    expected_exclusions = {package: skip if any(re.fullmatch(skip_parent, name) for name in names) else None for package, names in expected.items()}
    if hosted != expected or excluded != expected_exclusions:
        raise AssertionError('hosted and VM lifecycle selection/exclusion drift')


class SelectionTest(unittest.TestCase):
    def source(self):
        return (Path(__file__).resolve().parents[1] / 'Makefile.local').read_text()

    def test_hosted_consumes_the_vm_case_and_exclusion_catalog(self):
        assert_same_selection(self.source(), vm.LIFECYCLE_REQUIRED, vm.LIFECYCLE_SKIP)

    def test_hosted_independent_case_package_and_exclusion_drift_refuse(self):
        for old, new in [('TestSessionQuiescenceLive', 'TestNewLifecycle'),
                         ('./cmd/internal/launcher -run', './cmd/internal/other -run'),
                         ('/^checkpoint$$', '/^other$$')]:
            with self.subTest(old=old), self.assertRaises(AssertionError):
                assert_same_selection(self.source().replace(old, new), vm.LIFECYCLE_REQUIRED, vm.LIFECYCLE_SKIP)

    def test_exclusion_moved_to_another_package_refuses(self):
        skip_arg = "-skip '" + vm.LIFECYCLE_SKIP.replace('$', '$$') + "' "
        changed = self.source().replace(skip_arg, '').replace('./cmd/internal/launcher -run', skip_arg + './cmd/internal/launcher -run')
        with self.assertRaises(AssertionError):
            assert_same_selection(changed, vm.LIFECYCLE_REQUIRED, vm.LIFECYCLE_SKIP)

    def test_vm_independent_addition_removal_and_exclusion_drift_refuse(self):
        for name, add in [('TestSessionQuiescenceLive', False), ('TestFutureLifecycle', True)]:
            required = {p: set(names) for p, names in vm.LIFECYCLE_REQUIRED.items()}
            names = required['github.com/xianxu/pair/cmd/internal/launcher']
            names.add(name) if add else names.remove(name)
            with self.subTest(name=name), self.assertRaises(AssertionError):
                assert_same_selection(self.source(), required, vm.LIFECYCLE_SKIP)
        with self.assertRaises(AssertionError):
            assert_same_selection(self.source(), vm.LIFECYCLE_REQUIRED, '^Other$')

    def test_both_population_patterns_derive_only_catalog_top_levels(self):
        for required in (vm.NATIVE_REQUIRED, vm.LIFECYCLE_REQUIRED):
            pattern = vm.test_pattern(required)
            for name in {n for names in required.values() for n in names}:
                self.assertEqual(bool(re.fullmatch(pattern, name)), '/' not in name)
            self.assertIsNone(re.fullmatch(pattern, 'TestUnselected'))


if __name__ == '__main__':
    unittest.main()
