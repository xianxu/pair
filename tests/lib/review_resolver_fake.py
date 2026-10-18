"""Owned observation-fixture identities, not a replacement Git interpreter.

The shared catalog names two known commits. Actual loose HEAD/ref files choose
the current one, so even a checkout performed inside the editor is observed.
The Lua transport reads the same catalog at the vim.system boundary.
"""
import copy
import json
from pathlib import Path
import subprocess


class ReviewResolverFake:
    LIMIT = 8 << 20

    def __init__(self, model_path):
        self.model = json.loads(self._read(Path(model_path)))
        self.repo = Path(self.model['repo'])
        self.calls = 0

    @classmethod
    def _read(cls, path):
        with path.open('rb') as source:
            value = source.read(cls.LIMIT + 1)
        if len(value) > cls.LIMIT:
            raise ValueError('fixture resolver input exceeds 8 MiB')
        return value.decode('utf-8')

    def resolve(self, repo, snapshot=False):
        try:
            if str(repo) != str(self.repo):
                raise ValueError('unknown fixture repository')
            head = self._read(self.repo / '.git' / 'HEAD').strip()
            if not head.startswith('ref: '):
                raise ValueError('unknown fixture HEAD')
            ref = head[5:]
            identity = copy.deepcopy(self.model['by_ref'].get(ref))
            if identity is None:
                raise ValueError('unknown fixture branch')
            if self._read(self.repo / '.git' / ref).strip() != identity['head']:
                raise ValueError('fixture HEAD changed')
            if snapshot:
                identity['snapshot'] = self._read(self.repo / identity['file'])
            return identity
        except (OSError, UnicodeError, ValueError) as error:
            return dict(status='invalid', repo=str(repo), diagnostic=str(error))

    def run(self, command, **_options):
        self.calls += 1
        if command[1:4] != ['review', 'readiness', '--resolve'] or len(command) < 5:
            raise AssertionError('unexpected fixture resolver command')
        value = self.resolve(command[4], snapshot='--snapshot' in command[5:])
        return subprocess.CompletedProcess(command, 0, json.dumps(value), '')
