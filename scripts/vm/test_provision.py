"""Portable, no-install tests for the guest provisioning trust boundary."""
import hashlib
import io
import json
import pathlib
import tarfile
import tempfile
import unittest
import contextlib
import subprocess
import os
import stat
import sys

SCRIPT = pathlib.Path(__file__).with_name('provision.sh')


class ProvisionTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.api = {'__name__': 'provision_test'}
        source = SCRIPT.read_text().split("<<'PYTHON'\n", 1)[1].rsplit('\nPYTHON', 1)[0]
        exec(compile(source, str(SCRIPT), 'exec'), cls.api)

    def test_lock_requires_complete_pinned_tools(self):
        lock = json.loads(SCRIPT.with_name('image.lock.json').read_text())
        self.api['validate_lock'](lock)
        for change in ('checksum', 'base', 'missing'):
            bad = json.loads(json.dumps(lock))
            if change == 'checksum':
                bad['tools']['go']['sha256'] = 'latest'
            elif change == 'base':
                bad['base'] = 'ghcr.io/cirruslabs/macos-tahoe-base:latest'
            else:
                del bad['tools']['zellij']
            with self.assertRaises(ValueError):
                self.api['validate_lock'](bad)

    def test_archive_rejects_traversal_symlinks_and_size_before_extract(self):
        for name, link, size in [('../escape', None, 0), ('safe/link', '../../escape', 0), ('large', None, 1025)]:
            with self.subTest(name=name), tempfile.TemporaryDirectory() as d:
                path = pathlib.Path(d) / 'archive.tar.gz'
                with tarfile.open(path, 'w:gz') as tf:
                    member = tarfile.TarInfo(name)
                    if link:
                        member.type = tarfile.SYMTYPE
                        member.linkname = link
                    member.size = size
                    tf.addfile(member, io.BytesIO(bytes(size)))
                with self.assertRaises(ValueError):
                    self.api['validate_archive'](path, max_bytes=1024)
                self.assertEqual(sorted(p.name for p in pathlib.Path(d).iterdir()), ['archive.tar.gz'])

    def test_archive_rejects_indirect_symlink_escape(self):
        with tempfile.TemporaryDirectory() as d:
            path = pathlib.Path(d) / 'archive.tar.gz'
            with tarfile.open(path, 'w:gz') as tf:
                for name, target in [('inside', '.'), ('escape', 'inside/..')]:
                    member = tarfile.TarInfo(name)
                    member.type = tarfile.SYMTYPE
                    member.linkname = target
                    tf.addfile(member)
            with self.assertRaises(ValueError):
                self.api['validate_archive'](path)

    def test_checksum_compares_download_bytes(self):
        with tempfile.TemporaryDirectory() as d:
            path = pathlib.Path(d) / 'asset'
            path.write_bytes(b'vendor binary')
            expected = hashlib.sha256(path.read_bytes()).hexdigest()
            self.api['verify_checksum'](path, expected)
            path.write_bytes(b'corruption')
            with self.assertRaises(ValueError):
                self.api['verify_checksum'](path, expected)

    def test_go_layout_reports_real_tree_and_missing_tool_directory(self):
        self.assertTrue('describe_go_layout' in self.api)
        with tempfile.TemporaryDirectory() as d:
            root = pathlib.Path(d)
            binary = root / 'store/go-1.26.3/go/bin/go'
            binary.parent.mkdir(parents=True)
            binary.write_bytes(b'fixture')
            entry = root / 'bin/go'
            entry.parent.mkdir()
            entry.symlink_to(binary)
            describe = self.api['describe_go_layout']
            missing = describe(binary, entry)
            self.assertEqual(missing['entrypoint_resolved'], str(binary))
            self.assertEqual(missing['installed_root'], str(binary.parent.parent))
            self.assertFalse(missing['pkg_tool']['exists'])
            tools = binary.parent.parent / 'pkg/tool'
            tools.mkdir(parents=True)
            tools.chmod(0o755)
            present = describe(binary, entry)
            self.assertTrue(present['pkg_tool']['exists'])
            self.assertEqual(present['pkg_tool']['mode'], '0755')

    def test_direct_go_probe_cannot_turn_failed_entrypoint_into_success(self):
        calls = []
        original_run = self.api['run']
        failure = subprocess.CalledProcessError(2, ['/fixture/bin/go', 'version'])
        def command(argv, **kwargs):
            calls.append(argv)
            if len(calls) == 1:
                raise failure
            return 'go version go1.26.3 darwin/arm64'
        self.api['run'] = command
        try:
            with contextlib.redirect_stderr(io.StringIO()) as diagnostic:
                with self.assertRaises(subprocess.CalledProcessError) as caught:
                    self.api['go_version'](pathlib.Path('/fixture/store/go/bin/go'), pathlib.Path('/fixture/bin/go'))
            self.assertIs(caught.exception, failure)
            self.assertEqual(calls, [['/fixture/bin/go', 'version'], ['/fixture/store/go/bin/go', 'version']])
            self.assertIn('go direct-binary probe succeeded', diagnostic.getvalue())
            self.assertLess(len(diagnostic.getvalue()), 4096)
        finally:
            self.api['run'] = original_run

    @unittest.skipUnless(sys.platform == 'darwin', 'Darwin symlink permission semantics')
    def test_installed_links_remain_readable_under_private_umask(self):
        original_run, original_prefix = self.api['run'], self.api['PREFIX']
        def command(argv, **kwargs):
            # Same filesystem operations as guest installation, without sudo:
            # this test owns both the scratch root and all its links.
            subprocess.run(argv, check=True)
        try:
            with tempfile.TemporaryDirectory() as scratch:
                root = pathlib.Path(scratch)
                (root / 'bin').mkdir()
                target = root / 'tool'
                target.write_bytes(b'fixture')
                self.api['PREFIX'], self.api['run'] = root, command
                previous_umask = os.umask(0o077)
                try:
                    self.api['link']('tool', target)
                finally:
                    os.umask(previous_umask)
                entry = root / 'bin/tool'
                self.assertEqual(stat.S_IMODE(entry.lstat().st_mode), 0o755)
                self.assertEqual(os.readlink(entry), str(target))
        finally:
            self.api['run'], self.api['PREFIX'] = original_run, original_prefix

    @unittest.skipUnless(sys.platform == 'darwin', 'Darwin symlink permission semantics')
    def test_go_diagnostics_survive_unreadable_symlink(self):
        with tempfile.TemporaryDirectory() as scratch:
            root = pathlib.Path(scratch)
            binary = root / 'go/bin/go'
            binary.parent.mkdir(parents=True)
            binary.write_bytes(b'fixture')
            entry = root / 'entry'
            entry.symlink_to(binary)
            os.lchmod(entry, 0)
            try:
                info = self.api['describe_go_layout'](binary, entry)
                self.assertEqual(info['entrypoint_lstat']['mode'], '0000')
                self.assertEqual(info['entrypoint_resolve_errno'], 13)
                self.assertEqual(info['entrypoint_readlink_errno'], 13)
            finally:
                os.lchmod(entry, 0o755)

    def test_version_requires_exact_token(self):
        self.api['verify_version']('zellij 0.45.1', '0.45.1')
        with self.assertRaises(ValueError):
            self.api['verify_version']('zellij 0.45.10', '0.45.1')


if __name__ == '__main__':
    unittest.main()
