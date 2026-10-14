import contextlib
import io
import itertools
import json
from pathlib import Path
import tempfile
import unittest
import subprocess
import sys
from unittest.mock import patch

import labels


A, B, G = 'a' * 32, 'b' * 32, 'c' * 32


def fact(seq, ms, event='snapshot', **fields):
    row = dict(version=1, generation=G, seq=seq, at='2026-10-13T00:00:00Z',
               elapsed_ms=ms, dropped=0, event=event)
    if event == 'snapshot':
        row['slots'] = []
    elif event == 'hub_delivery':
        row.update(subject=A, sender=B, message=B, hint='unknown',
                   outcome='uncertain', generated=False)
    elif event == 'session_write':
        row.update(subject=A, incarnation=B)
    row.update(fields)
    return row


class LabelsTest(unittest.TestCase):
    def label(self, rows, gaps=False):
        return list(labels.derive_labels(rows, gaps))[0]

    def test_positive_window_edges(self):
        for ms, expected in [(0, False), (1, True), (600000, True), (600001, False)]:
            with self.subTest(ms=ms):
                result = self.label([fact(1, 0, 'hub_delivery'),
                                     fact(2, ms, 'session_write'), fact(3, 600002)])
                self.assertIs(result['label'], expected)
                self.assertEqual(result['outcome'], 'uncertain')

    def test_write_must_be_later_by_sequence(self):
        self.assertIsNone(self.label([fact(2, 0, 'hub_delivery'),
                                     fact(1, 1, 'session_write'), fact(3, 600001)])['label'])

    def test_subject_generation_and_generated_boundaries(self):
        delivery = fact(1, 0, 'hub_delivery')
        for write in [fact(2, 2, 'session_write', subject=B),
                      fact(2, 2, 'session_write', generation=B)]:
            self.assertIsNone(self.label([delivery, write])['label'])
        self.assertEqual(list(labels.derive_labels([dict(delivery, generated=True)], False)), [])

    def test_permutations_identical_duplicates(self):
        rows = [fact(10, 0, 'hub_delivery'), fact(11, 600001), fact(11, 600001)]
        for perm in itertools.permutations(rows):
            self.assertIs(self.label(perm)['label'], False)

    def test_inclusive_boundary_tail_is_incomplete(self):
        rows = [fact(1, 0, 'hub_delivery'), fact(2, 600000)]
        self.assertIsNone(self.label(rows)['label'])
        # A later sequence can still carry a qualifying write at that timestamp.
        self.assertIs(self.label(rows + [fact(3, 600000, 'session_write')])['label'], True)
        self.assertIs(self.label(rows + [fact(3, 600001)])['label'], False)

    def test_inclusive_boundary_gap_is_incomplete(self):
        rows = [fact(1, 0, 'hub_delivery'), fact(2, 600000), fact(4, 600001)]
        for perm in itertools.permutations(rows):
            result = self.label(perm)
            self.assertIsNone(result['label'])
            self.assertEqual(result['reason'], 'sequence_gap')
        self.assertIs(self.label(rows + [fact(3, 600000, 'session_write')])['label'], True)
        self.assertIs(self.label(rows + [fact(3, 600000)])['label'], False)

    def test_coverage_corruption_never_proves_negative(self):
        delivery = fact(1, 0, 'hub_delivery')
        for tail in [[fact(3, 600000)], [fact(2, 599999)],
                     [fact(2, 600000, dropped=1)],
                     [fact(2, 600000), fact(2, 600001)],
                     [fact(2, 600000), fact(3, 500000)]]:
            row = self.label([delivery] + tail)
            self.assertIsNone(row['label'])
            self.assertTrue(row['reason'])
        self.assertIsNone(self.label([delivery, fact(2, 600000)], True)['label'])

    def test_valid_positive_survives_missing_evidence(self):
        self.assertIs(self.label([fact(1, 0, 'hub_delivery'),
                                  fact(5, 2, 'session_write', dropped=3)], True)['label'], True)

    def test_schema_rejects_unknown_and_bad_types(self):
        good = fact(1, 0, 'hub_delivery')
        self.assertEqual(labels.parse_fact(json.dumps(good)), good)
        for change in [dict(version=2), dict(seq=True), dict(seq=0), dict(elapsed_ms=-1),
                       dict(generation='private path'), dict(event='future'),
                       dict(hint='secret text'), dict(at='yesterday'), dict(generated=1),
                       dict(body='private text'), dict(outcome='future')]:
            with self.subTest(change=change), self.assertRaises(ValueError):
                labels.parse_fact(json.dumps(dict(good, **change)))
        for row in [fact(1, 0, slots=[{}]), fact(1, 0, slots=[{}] * 129)]:
            with self.assertRaises(ValueError):
                labels.parse_fact(json.dumps(row))

    def test_reader_invalid_rows_and_limits(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'facts.jsonl'
            path.write_bytes((json.dumps(fact(1, 0)) + '\nsecret malformed\n').encode())
            err = io.StringIO()
            with contextlib.redirect_stderr(err):
                rows, gaps = labels.read_facts([path])
            self.assertEqual(len(rows), 1)
            self.assertTrue(gaps)
            self.assertIn('1', err.getvalue())
            self.assertNotIn('secret', err.getvalue())
            for constant, limit in [('MAX_BYTES', 8), ('MAX_ROWS', 1), ('MAX_LINE', 8)]:
                with patch.object(labels, constant, limit), self.assertRaises(ValueError):
                    labels.read_facts([path])

    def test_snapshot_nullable_projection(self):
        slot = dict(subject=A, incarnation=B, known=False, resting=None, settled=None,
                    picker=None, pending=None, quiet_ms=None, submission=None)
        row = fact(1, 0, slots=[slot])
        self.assertEqual(labels.parse_fact(json.dumps(row)), row)
        for field, value in [('known', None), ('quiet_ms', True), ('pending', 0), ('submission', -1)]:
            with self.subTest(field=field), self.assertRaises(ValueError):
                labels.parse_fact(json.dumps(fact(1, 0, slots=[dict(slot, **{field: value})])))

    def test_conflicting_write_is_not_positive_evidence(self):
        rows = [fact(1, 0, 'hub_delivery'), fact(2, 1, 'session_write'),
                fact(2, 1), fact(3, 600000)]
        for perm in itertools.permutations(rows):
            self.assertIsNone(self.label(perm)['label'])

    def test_write_index_matches_brute_force_with_backwards_clocks(self):
        writes = [fact(i + 1, ms, 'session_write')
                  for i, ms in enumerate([0, 600001, 20, 600000, 700000, 2, 20])]
        index = labels.WriteIndex(writes)
        for start in [0, 1, 20, 600000, 900000]:
            for seq in range(9):
                expected = any(w['seq'] > seq and start < w['elapsed_ms'] <= start + 600000
                               for w in writes)
                self.assertEqual(index.matches(start, seq), expected)

    def test_cli_malformed_and_oversize_do_not_echo_content(self):
        script = Path(labels.__file__)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'facts.jsonl'
            path.write_text('\n'.join([json.dumps(fact(1, 0, 'hub_delivery')),
                                       'private text', json.dumps(fact(2, 600000))]) + '\n')
            result = subprocess.run([sys.executable, str(script), str(path)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0)
            self.assertIsNone(json.loads(result.stdout)['label'])
            self.assertNotIn('private text', result.stdout + result.stderr)
            path.write_bytes(b'x' * (labels.MAX_LINE + 1))
            result = subprocess.run([sys.executable, str(script), str(path)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, '')

    def test_duplicate_json_keys_rejected(self):
        text = json.dumps(fact(1, 0))
        with self.assertRaises(ValueError):
            labels.parse_fact(text[:-1] + ', "version": 1}')

    def test_numeric_ranges_match_go_envelope(self):
        for field, maximum in [('seq', 2**64 - 1), ('dropped', 2**64 - 1),
                               ('submission', 2**64 - 1), ('elapsed_ms', 2**63 - 1)]:
            with self.subTest(field=field):
                row = fact(1, 0, **{field: maximum}) if field != 'seq' else fact(maximum, 0)
                self.assertEqual(labels.parse_fact(json.dumps(row)), row)
                for invalid in (maximum + 1, True):
                    with self.assertRaises(ValueError):
                        labels.parse_fact(json.dumps(dict(row, **{field: invalid})))
        slot = dict(subject=A, incarnation=B, known=True, resting=None, settled=None,
                    picker=None, pending=None, quiet_ms=2**63 - 1, submission=2**64 - 1)
        row = fact(1, 0, slots=[slot])
        self.assertEqual(labels.parse_fact(json.dumps(row)), row)
        for field, invalid in [('quiet_ms', 2**63), ('submission', 2**64),
                               ('quiet_ms', True), ('submission', True)]:
            with self.subTest(field=field, invalid=invalid), self.assertRaises(ValueError):
                labels.parse_fact(json.dumps(fact(1, 0, slots=[dict(slot, **{field: invalid})])))


if __name__ == '__main__':
    unittest.main()
