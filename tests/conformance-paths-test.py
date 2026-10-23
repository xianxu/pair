"""The conformance path catalog must match both hosted workflow consumers."""
import importlib.util
import json
from pathlib import Path
import unittest

SPEC = importlib.util.spec_from_file_location("conformance_paths", Path(__file__).with_name("conformance-paths.py"))
paths = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(paths)


def workflow(entries):
    block = "\n".join('      - ' + json.dumps(entry) for entry in entries)
    return 'on:\n  pull_request:\n    paths:\n' + block + '\n  push:\n    branches: [main]\n    paths:\n' + block + '\njobs:\n  conformance:\n    runs-on: macos-latest\n'


class PathsTest(unittest.TestCase):
    def test_current_catalog_matches_both_consumers(self):
        root = Path(__file__).resolve().parents[1]
        paths.validate_paths((root / 'tests/conformance-paths.json').read_text(), (root / '.github/workflows/couch-zellij-conformance.yml').read_text())

    def test_addition_deletion_and_either_workflow_drift_refuse(self):
        catalog = json.dumps({'version': 1, 'paths': ['cmd/**', 'go.mod']})
        paths.validate_paths(catalog, workflow(['cmd/**', 'go.mod']))
        for text in [workflow(['cmd/**']), workflow(['cmd/**', 'go.mod', 'new/**']), workflow(['cmd/**', 'go.mod']).replace('"cmd/**"', '"other/**"', 1)]:
            with self.subTest(text=text), self.assertRaisesRegex(RuntimeError, 'paths'):
                paths.validate_paths(catalog, text)

    def test_malformed_or_ambiguous_catalog_refuses(self):
        for raw in ['[]', '{}', '{"version":2,"paths":["cmd/**"]}', '{"version":1,"paths":[]}', '{"version":1,"paths":["cmd/**","cmd/**"]}', '{"version":1,"paths":[1]}', '{"version":1,"version":1,"paths":["cmd/**"]}']:
            with self.subTest(raw=raw), self.assertRaises((RuntimeError, ValueError)):
                paths.validate_paths(raw, workflow(['cmd/**']))

    def test_missing_duplicate_or_empty_consumer_refuses(self):
        catalog = json.dumps({'version': 1, 'paths': ['cmd/**']})
        for text in [workflow(['cmd/**']).replace('  push:', '  other:'), workflow(['cmd/**']) + '  pull_request:\n    paths:\n      - "cmd/**"\n', workflow([]), workflow(['cmd/**', 'cmd/**'])]:
            with self.subTest(text=text), self.assertRaises(RuntimeError):
                paths.validate_paths(catalog, text)


if __name__ == '__main__':
    unittest.main()
