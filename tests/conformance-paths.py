"""Check the canonical conformance path list against both hosted triggers.

The workflow's fixed paths-list subset is decoded without a YAML dependency.
Any unsupported list spelling refuses rather than silently losing a trigger.
"""
import json
from pathlib import Path
import re


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f'duplicate JSON key: {key}')
        result[key] = value
    return result


def validate_paths(catalog_text, workflow_text):
    catalog = json.loads(catalog_text, object_pairs_hook=unique_object)
    if not isinstance(catalog, dict) or type(catalog.get('version')) is not int or catalog['version'] != 1:
        raise RuntimeError('invalid conformance paths catalog version')
    expected = catalog.get('paths')
    if not isinstance(expected, list) or not expected or any(not isinstance(path, str) or not path for path in expected):
        raise RuntimeError('invalid conformance paths catalog')
    if len(set(expected)) != len(expected):
        raise RuntimeError('duplicate conformance paths')
    consumers = {}
    event, reading = None, False
    for line in workflow_text.splitlines():
        match = re.fullmatch(r'  (pull_request|push):\s*', line)
        if match:
            event, reading = match[1], False
            if event in consumers:
                raise RuntimeError(f'duplicate workflow paths consumer: {event}')
            consumers[event] = None
        elif line and not line.startswith('    ') and not line.lstrip().startswith('#'):
            event, reading = None, False
        elif event and re.fullmatch(r'    paths:\s*', line):
            if consumers[event] is not None:
                raise RuntimeError(f'duplicate workflow paths: {event}')
            consumers[event], reading = [], True
        elif event and reading:
            match = re.fullmatch(r'      - (".*")\s*', line)
            if match:
                value = json.loads(match[1])
                if not isinstance(value, str):
                    raise RuntimeError('invalid workflow paths entry')
                consumers[event].append(value)
            elif line.strip() and not line.lstrip().startswith('#'):
                if line.startswith('      '):
                    raise RuntimeError(f'unsupported workflow paths entry: {line}')
                reading = False
    for event in ('pull_request', 'push'):
        if consumers.get(event) != expected:
            raise RuntimeError(f'{event} workflow paths differ from tests/conformance-paths.json')
    return expected


def main():
    root = Path(__file__).resolve().parents[1]
    validate_paths((root / 'tests/conformance-paths.json').read_text(), (root / '.github/workflows/couch-zellij-conformance.yml').read_text())


if __name__ == '__main__':
    main()
