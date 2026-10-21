"""Portable, no-install tests for the guest provisioning trust boundary."""
import hashlib
import io
import json
import pathlib
import tarfile
import tempfile
import unittest

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

    def test_version_requires_exact_token(self):
        self.api['verify_version']('zellij 0.45.1', '0.45.1')
        with self.assertRaises(ValueError):
            self.api['verify_version']('zellij 0.45.10', '0.45.1')


if __name__ == '__main__':
    unittest.main()
