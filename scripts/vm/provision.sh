#!/bin/bash
# Guest-only image builder. Invoke from an isolated Tart guest checkout with the
# read-only lock as argv[1]. stdout is a bounded tools JSON map; diagnostics stderr.
set -euo pipefail
if [[ $# != 1 || ! -r "$1" ]]; then
  echo 'usage: provision.sh READ_ONLY_IMAGE_LOCK_JSON (inside fresh Tart guest)' >&2
  exit 2
fi
if [[ "$(uname -s)" != Darwin || "$(uname -m)" != arm64 ]]; then
  echo 'provisioning requires a macOS arm64 guest' >&2
  exit 2
fi
# Tart guests advertise VirtualMac; refuse a physical Mac before privilege use.
case "$(/usr/sbin/sysctl -n hw.model)" in
  VirtualMac*) ;;
  *) echo 'refusing provisioning outside a VirtualMac guest' >&2; exit 2 ;;
esac
/usr/bin/xcrun --find clang >/dev/null
/usr/bin/xcrun --find make >/dev/null
[[ -x /usr/bin/python3 ]] || { echo 'base requires CLT Python bootstrap' >&2; exit 2; }
exec /usr/bin/python3 - "$1" <<'PYTHON'
import hashlib
import json
import os
import pathlib
import posixpath
import re
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import time

TOOLS = ('go', 'node', 'python3', 'nvim', 'zellij', 'claude', 'codex', 'qodercli')
PREFIX = pathlib.Path('/opt/pair')
MAX_JSON = 65536
DEADLINE = 0


def validate_lock(lock):
    if lock.get('version') != 1 or not re.fullmatch(
            r'ghcr\.io/cirruslabs/macos-tahoe-base@sha256:[a-f0-9]{64}', lock.get('base', '')):
        raise ValueError('lock requires version 1 and immutable Tahoe base')
    if set(lock.get('tools', {})) != set(TOOLS):
        raise ValueError('lock requires the complete fixed guest tool catalog')
    for name, tool in lock['tools'].items():
        if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+', tool.get('version', '')):
            raise ValueError(name + ': exact release version required')
        for field in ('sha256', 'binary_sha256'):
            if field in tool and not re.fullmatch(r'[a-f0-9]{64}', tool[field]):
                raise ValueError(name + ': invalid ' + field)
        if 'sha256' not in tool or not tool.get('url', '').startswith('https://'):
            raise ValueError(name + ': HTTPS URL and checksum required')
        if not tool.get('checksum_source', '').startswith('https://'):
            raise ValueError(name + ': checksum provenance required')
    if lock['tools']['zellij'].get('binary_sha256') != 'ca5f9333735bdbc59a621f1d8ed8e24798845302a28ff175d253d4793d5a4a2c':
        raise ValueError('official Zellij 0.45.1 binary checksum required')
    if lock['tools']['zellij']['version'] != '0.45.1':
        raise ValueError('qualified Zellij version must be 0.45.1')


def verify_checksum(path, expected):
    digest = hashlib.sha256()
    with open(path, 'rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(block)
    if digest.hexdigest() != expected:
        raise ValueError(str(path.name) + ': SHA256 mismatch')
    return digest.hexdigest()


def validate_archive(path, max_bytes=1024 * 1024 * 1024):
    # Validate the complete pinned archive before allowing tar any writes.
    total = 0
    seen = set()
    links = {}
    with tarfile.open(path, 'r:gz') as archive:
        for count, member in enumerate(archive, 1):
            name = posixpath.normpath(member.name)
            if (count > 50000 or name.startswith('/') or name == '..'
                    or name.startswith('../') or name in seen):
                raise ValueError('unsafe or duplicate archive path')
            seen.add(name)
            if not (member.isfile() or member.isdir() or member.issym()):
                raise ValueError('unsupported archive member')
            total += member.size
            if total > max_bytes:
                raise ValueError('archive exceeds extracted byte budget')
            if member.issym():
                links[name] = member.linkname
                target = posixpath.normpath(posixpath.join(posixpath.dirname(name), member.linkname))
                if member.linkname.startswith('/') or target == '..' or target.startswith('../'):
                    raise ValueError('archive symlink escapes extraction root')
    # Resolve links as filesystem components, before normalizing '..': a -> '.'
    # followed by b -> 'a/..' escapes even though lexical normalization is '.'.
    for name in seen:
        parents = name.split('/')[:-1]
        if any('/'.join(parents[:i]) in links for i in range(1, len(parents) + 1)):
            raise ValueError('archive member has a symlink ancestor')
    for name in links:
        pending = name.split('/')
        resolved = []
        follows = 0
        while pending:
            part = pending.pop(0)
            if part in ('', '.'):
                continue
            if part == '..':
                if not resolved:
                    raise ValueError('archive symlink chain escapes extraction root')
                resolved.pop()
                continue
            resolved.append(part)
            target = links.get('/'.join(resolved))
            if target is not None:
                follows += 1
                if follows > 40:
                    raise ValueError('archive symlink cycle or excessive chain')
                resolved.pop()
                pending = target.split('/') + pending
    return total


def verify_version(output, expected):
    if not re.search(r'(?<![0-9.])' + re.escape(expected) + r'(?![0-9.])', output):
        raise ValueError('installed version differs from lock: ' + expected)


def run(argv, timeout=300, capture=False, root=False):
    remaining = DEADLINE - time.monotonic()
    if remaining <= 0:
        raise TimeoutError('guest provisioning exceeded twenty-minute budget')
    if root and os.geteuid() != 0:
        argv = ['/usr/bin/sudo', '-n'] + list(argv)
    # No caller environment or credential material is forwarded to installers.
    env = {'PATH': '/opt/pair/bin:/usr/bin:/bin:/usr/sbin:/sbin',
           'HOME': os.environ['HOME'], 'TMPDIR': os.environ.get('TMPDIR', '/tmp'),
           'LANG': 'en_US.UTF-8', 'DISABLE_AUTOUPDATER': '1',
           'CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC': '1'}
    result = subprocess.run(argv, env=env, stdin=subprocess.DEVNULL,
                            stdout=subprocess.PIPE if capture else sys.stderr,
                            stderr=sys.stderr, timeout=min(timeout, remaining), check=True)
    if capture:
        if len(result.stdout) > 16384:
            raise ValueError('version output exceeds 16 KiB')
        return result.stdout.decode('utf-8', errors='strict').strip()
    return ''


def link(name, target):
    path = PREFIX / 'bin' / name
    run(['/bin/ln', '-s', str(target), str(path)], root=True)
    # Darwin enforces symlink read permissions for readlink/EvalSymlinks even
    # when exec follows it. The guest's private umask must not make root-owned
    # links unreadable to the ordinary test user. Change the link, not its target.
    run(['/bin/chmod', '-h', '755', str(path)], root=True)


def describe_go_layout(binary, entrypoint):
    root = binary.parent.parent
    def describe(path, follow=True):
        try:
            info = path.stat() if follow else path.lstat()
            return {'exists': True, 'directory': stat.S_ISDIR(info.st_mode),
                    'mode': format(stat.S_IMODE(info.st_mode), '04o'),
                    'uid': info.st_uid, 'gid': info.st_gid}
        except OSError as error:
            return {'exists': False, 'errno': error.errno}
    result = {'entrypoint': str(entrypoint),
              'entrypoint_lstat': describe(entrypoint, follow=False),
              'installed_root': str(root),
              'root': describe(root), 'pkg': describe(root / 'pkg'),
              'pkg_tool': describe(root / 'pkg/tool'),
              'store': describe(root.parent.parent)}
    try:
        result['entrypoint_target'] = os.readlink(entrypoint)
    except OSError as error:
        result['entrypoint_readlink_errno'] = error.errno
        result['entrypoint_resolve_errno'] = error.errno
        return result
    try:
        result['entrypoint_resolved'] = os.path.realpath(entrypoint)
    except OSError as error:
        result['entrypoint_resolve_errno'] = error.errno
    return result


def go_version(binary, entrypoint):
    # Keep the original failure authoritative; direct execution is diagnostic
    # evidence to distinguish layout failure from executable-path discovery.
    layout = json.dumps(describe_go_layout(binary, entrypoint), sort_keys=True)
    print('go installation layout: ' + layout, file=sys.stderr)
    try:
        return run([str(entrypoint), 'version'], timeout=30, capture=True)
    except Exception:
        try:
            direct = run([str(binary), 'version'], timeout=30, capture=True)
            print('go direct-binary probe succeeded: ' + direct, file=sys.stderr)
        except Exception as probe_error:
            print('go direct-binary probe failed: ' + str(probe_error), file=sys.stderr)
        raise


def main():
    global DEADLINE
    DEADLINE = time.monotonic() + 1200
    with open(sys.argv[1], 'rb') as source:
        raw = source.read(MAX_JSON + 1)
    if len(raw) > MAX_JSON:
        raise ValueError('lock exceeds 64 KiB')
    lock = json.loads(raw)
    validate_lock(lock)
    # Base owns CLT and the Tart guest agent; never fetch a floating replacement.
    run(['/usr/bin/xcrun', '--find', 'clang'], capture=True)
    run(['/usr/bin/xcrun', '--find', 'make'], capture=True)
    run(['/usr/bin/git', '--version'], capture=True)
    for command in ('/usr/bin/tic', '/usr/bin/infocmp'):
        if not os.access(command, os.X_OK):
            raise ValueError('base missing terminal prerequisite: ' + command)
    if PREFIX.exists():
        raise ValueError('fresh image required: /opt/pair already exists')
    run(['/bin/mkdir', '-p', str(PREFIX / 'bin'), str(PREFIX / 'store')], root=True)
    run(['/bin/chmod', '755', str(PREFIX), str(PREFIX / 'bin'), str(PREFIX / 'store')], root=True)
    if shutil.disk_usage('/opt').free < 4 * 1024**3:
        raise ValueError('provisioning requires 4 GiB free guest storage')
    tools = {}
    # tempfile honors the guest-only TMPDIR supplied by the enclosing runner.
    with tempfile.TemporaryDirectory(prefix='pair-provision-') as scratch:
        scratch = pathlib.Path(scratch)
        for name in TOOLS:
            pin = lock['tools'][name]
            print('provisioning ' + name + ' ' + pin['version'], file=sys.stderr)
            asset = scratch / (name + ('.pkg' if name == 'python3' else '.download'))
            run(['/usr/bin/curl', '--fail', '--silent', '--show-error', '--location',
                 '--proto', '=https', '--proto-redir', '=https', '--connect-timeout', '30',
                 '--max-time', '300', '--max-filesize', str(384 * 1024**2),
                 '--output', str(asset), pin['url']])
            verify_checksum(asset, pin['sha256'])
            destination = PREFIX / 'store' / (name + '-' + pin['version'])
            stage = scratch / name
            stage.mkdir()
            if name == 'python3':
                run(['/usr/sbin/installer', '-pkg', str(asset), '-target', '/'], root=True)
                binary = pathlib.Path('/Library/Frameworks/Python.framework/Versions/3.12/bin/python3.12')
                link('python3', binary)
                link('python', binary)
                link('pip3', binary.parent / 'pip3.12')
            else:
                if name == 'claude':
                    shutil.copyfile(asset, stage / 'claude')
                    (stage / 'claude').chmod(0o755)
                    relative = 'claude'
                else:
                    validate_archive(asset)
                    run(['/usr/bin/tar', '-xzf', str(asset), '-C', str(stage)])
                    relative = {
                        'go': 'go/bin/go',
                        'node': 'node-v' + pin['version'] + '-darwin-arm64/bin/node',
                        'nvim': 'nvim-macos-arm64/bin/nvim',
                        'zellij': 'zellij',
                        'codex': 'codex-aarch64-apple-darwin',
                        'qodercli': 'qodercli',
                    }[name]
                staged_binary = stage / relative
                if not staged_binary.is_file() or staged_binary.is_symlink():
                    raise ValueError(name + ': expected regular vendor binary missing')
                if pin.get('binary_sha256'):
                    verify_checksum(staged_binary, pin['binary_sha256'])
                run(['/bin/mv', str(stage), str(destination)], root=True)
                run(['/usr/sbin/chown', '-R', 'root:wheel', str(destination)], root=True)
                run(['/bin/chmod', '-R', 'a+rX', str(destination)], root=True)
                binary = destination / relative
                link(name, binary)
                if name == 'go':
                    link('gofmt', binary.parent / 'gofmt')
                if name == 'node':
                    link('npm', binary.parent / 'npm')
                    link('npx', binary.parent / 'npx')
            if name == 'go':
                output = go_version(binary, PREFIX / 'bin/go')
            else:
                output = run([str(PREFIX / 'bin' / name), '--version'], timeout=30, capture=True)
            verify_version(output, pin['version'])
            # sha256 identifies the installed executable; archive digest is retained separately.
            actual = hashlib.sha256(binary.read_bytes()).hexdigest()
            tools[name] = dict(pin, sha256=actual, download_sha256=pin['sha256'])
            asset.unlink()
        run([str(PREFIX / 'bin/npm'), '--version'], timeout=30, capture=True)
        run([str(PREFIX / 'bin/nvim'), '--headless', '-u', 'NONE', '+qall'], timeout=30)
    encoded = json.dumps(tools, sort_keys=True, separators=(',', ':'))
    if len(encoded.encode()) > MAX_JSON:
        raise ValueError('tool manifest exceeds 64 KiB')
    print(encoded)


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        print('guest provisioning failed: ' + str(error), file=sys.stderr)
        sys.exit(1)
PYTHON
