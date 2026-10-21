"""Stateful VM-command fixture: literal argv, one invocation, exact failure."""
import importlib.util
import json
import os
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


if __name__ == '__main__':
    unittest.main()
