#!/bin/bash
# One invocation; clock/due/missed policy belongs to the shared Couch scheduler.
set -euo pipefail
exec python3 - "$@" <<'PYTHON'
import argparse
import contextlib
import datetime
import fcntl
import json
import os
from pathlib import Path
import pwd
import re
import signal
import stat
import subprocess
import sys

NIGHT_LIMIT = 16
BYTE_LIMIT = 4 << 30
# Generic runner logs plus its separately bounded result and this manifest.
RUN_RESERVATION = (256 << 20) + 65536 + 4096
NATIVE_HOME = Path(pwd.getpwuid(os.getuid()).pw_dir)


def retained_bytes(root):
    total = 0
    entries = 0
    for parent, directories, files in os.walk(root, followlinks=False):
        for name in directories + files:
            entries += 1
            if entries > 1600000:
                raise ValueError('nightly entry bound reached; inspect/archive evidence')
            info = (Path(parent) / name).lstat()
            if not (stat.S_ISDIR(info.st_mode) or stat.S_ISREG(info.st_mode)):
                raise ValueError('nightly evidence contains a symlink or special file; inspect/archive')
            if stat.S_ISREG(info.st_mode):
                total += info.st_size
    return total


def reserved_night_bytes(night):
    size = retained_bytes(night)
    try:
        with (night / 'nightly.json').open('rb') as stream:
            raw = stream.read(4097)
        manifest = json.loads(raw) if len(raw) <= 4096 else None
        code = manifest.get('exit_code') if isinstance(manifest, dict) else None
    except (OSError, ValueError):
        code = None
    # An abruptly dead wrapper can leave SDLC writing under its own lifetime.
    # Charge its full possible bundle until a joined completion is recorded.
    return size if type(code) is int and 0 <= code <= 255 else max(size, RUN_RESERVATION)


@contextlib.contextmanager
def reserve_nightly_artifacts(root, date):
    if not root.is_absolute() or root.is_symlink():
        raise ValueError('nightly artifact root must be absolute and not a symlink')
    root = root.resolve()
    managed = (NATIVE_HOME / '.tart/pair-runner').resolve()
    if root == managed or root in managed.parents or managed in root.parents:
        raise ValueError('nightly artifact root overlaps managed runner storage')
    root.mkdir(mode=0o700, parents=True, exist_ok=True)
    descriptor = os.open(root / '.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, 'r+') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise ValueError('another nightly invocation owns evidence admission') from None
        nights = []
        for path in root.iterdir():
            if path.name == '.lock':
                continue
            if not re.fullmatch(r'\d{4}-\d{2}-\d{2}', path.name) or path.is_symlink() or not path.is_dir():
                raise ValueError('unexpected nightly evidence; inspect/archive before running')
            nights.append(path)
        destination = root / date
        if destination.exists():
            raise ValueError(f'nightly date already reserved: {destination}; inspect existing evidence')
        if len(nights) >= NIGHT_LIMIT or sum(reserved_night_bytes(night) for night in nights) + RUN_RESERVATION > BYTE_LIMIT:
            raise ValueError('nightly retention saturated; inspect/archive evidence before running')
        destination.mkdir(mode=0o700)  # failed/partial nights keep their reservation
        yield destination


def save_manifest(destination, source, code):
    raw = json.dumps({'version':1, 'source':source, 'exit_code':code,
                      'artifacts':str(destination / 'vm')}, sort_keys=True) + '\n'
    if len(raw.encode()) > 4096:
        raise ValueError('nightly manifest exceeds 4 KiB')
    temporary = destination / 'nightly.json.tmp'
    with temporary.open('x') as stream:
        stream.write(raw)
    temporary.replace(destination / 'nightly.json')


def nightly_main(checkout, root, date):
    with reserve_nightly_artifacts(root, date) as destination:
        print(f'nightly evidence: {destination}', flush=True)
        source = None
        child = None
        cancelled = 0

        def cancel(signum, frame):
            nonlocal cancelled
            if not cancelled:
                cancelled = signum
                if child is not None and child.poll() is None:
                    try:
                        child.send_signal(signum)
                    except ProcessLookupError:
                        pass

        previous = {sig: signal.signal(sig, cancel) for sig in (signal.SIGTERM, signal.SIGINT)}
        code = 125
        save_manifest(destination, source, None)
        try:
            # Git never checks out main or changes the caller's tracked/untracked files.
            child = subprocess.Popen(['git', 'fetch', '--no-tags', 'origin',
                                      '+refs/heads/main:refs/remotes/origin/main'], cwd=checkout)
            if cancelled:
                child.send_signal(cancelled)
            fetched = child.wait()
            if cancelled:
                return 128 + cancelled
            if fetched:
                code = fetched if fetched > 0 else 128 - fetched
                return code
            source = subprocess.check_output(['git', 'rev-parse', '--verify',
                                               'refs/remotes/origin/main^{commit}'], cwd=checkout, text=True).strip()
            if not re.fullmatch(r'[0-9a-f]{40}|[0-9a-f]{64}', source):
                raise ValueError('origin/main did not resolve to an exact commit')
            save_manifest(destination, source, None)
            if cancelled:
                return 128 + cancelled
            env = dict(os.environ, PAIR_VM_ARTIFACTS_DIR=str(destination / 'vm'))
            child = subprocess.Popen(['sdlc', 'local-check', 'run', '--name', 'conformance', '--rev', source],
                                     cwd=checkout, env=env)
            if cancelled and child.poll() is None:
                child.send_signal(cancelled)
            code = child.wait()  # SDLC owns command-group cleanup, grace and exact-head status.
            code = 128 + cancelled if cancelled else (code if code >= 0 else 128 - code)
            return code
        finally:
            # Do not publish completion before the owned check has joined.
            if child is not None and child.poll() is None:
                child.wait()
            if cancelled:
                code = 128 + cancelled
            save_manifest(destination, source, code)
            for sig, handler in previous.items():
                signal.signal(sig, handler)


def cli():
    parser = argparse.ArgumentParser(description='Run one exact origin/main conformance check; never retry or evict nights.')
    parser.add_argument('--checkout', type=Path, required=True)
    parser.add_argument('--artifacts-root', type=Path, default=NATIVE_HOME / '.local/share/pair/conformance-nightly')
    parser.add_argument('--date', default=datetime.date.today().isoformat(), help='evidence date YYYY-MM-DD (default local date)')
    args = parser.parse_args()
    if datetime.date.fromisoformat(args.date).isoformat() != args.date:
        parser.error('date must be YYYY-MM-DD')
    if not args.checkout.is_absolute():
        parser.error('checkout must be absolute')
    return nightly_main(args.checkout.resolve(), args.artifacts_root, args.date)


if __name__ == '__main__':
    try:
        sys.exit(cli())
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f'nightly: {error}', file=sys.stderr)
        sys.exit(125)
PYTHON
