#!/usr/bin/env python3
"""Bounded offline ten-minute write labels from supervision fact JSONL."""

import argparse
from bisect import bisect_right
from collections import defaultdict
from datetime import datetime
import json
import re
import sys

MAX_LINE = 32 * 1024
MAX_BYTES = 64 * 1024 * 1024
MAX_ROWS = 100000
HORIZON_MS = 600000
MAX_UINT64 = 2**64 - 1
MAX_INT64 = 2**63 - 1
IDENTITY = re.compile(r'[0-9a-f]{32}\Z')
UTC = re.compile(r'\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z\Z')
COMMON = {'version', 'generation', 'seq', 'at', 'elapsed_ms', 'dropped', 'event'}
OPTIONAL = {'subject', 'incarnation', 'sender', 'message', 'hint', 'outcome',
            'generated', 'submission', 'connected', 'settled', 'picker', 'slots'}
SLOT_FIELDS = {'subject', 'incarnation', 'known', 'resting', 'settled', 'picker',
               'pending', 'quiet_ms', 'submission'}
REQUIRED = {
    'hub_delivery': {'subject', 'sender', 'message', 'hint', 'outcome', 'generated'},
    'session_write': {'subject', 'incarnation'},
    'operator_submit': {'subject', 'incarnation', 'submission'},
    'slot_state': {'subject', 'incarnation', 'connected', 'settled', 'picker'},
    'snapshot': {'slots'},
}


def nonnegative(value, maximum=MAX_UINT64):
    return type(value) is int and 0 <= value <= maximum


def identity(value):
    return isinstance(value, str) and IDENTITY.fullmatch(value) is not None


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError('duplicate JSON key')
        result[key] = value
    return result


def parse_fact(line):
    """Validate closed v1 fields without retaining unrecognized content."""
    row = json.loads(line, object_pairs_hook=unique_object)
    if not isinstance(row, dict) or not COMMON <= row.keys() or row.keys() - (COMMON | OPTIONAL):
        raise ValueError('invalid fact fields')
    if type(row['version']) is not int or row['version'] != 1:
        raise ValueError('unsupported version')
    event = row['event']
    if not isinstance(event, str) or event not in REQUIRED or not REQUIRED[event] <= row.keys():
        raise ValueError('invalid event')
    if (not all(nonnegative(row[key]) for key in ('seq', 'dropped'))
            or not nonnegative(row['elapsed_ms'], MAX_INT64) or row['seq'] == 0):
        raise ValueError('invalid sequence or time')
    for key in ('generation', 'subject', 'incarnation', 'sender', 'message'):
        if key in row and not identity(row[key]):
            raise ValueError('invalid identity')
    if not isinstance(row['at'], str) or not UTC.fullmatch(row['at']):
        raise ValueError('invalid UTC time')
    datetime.fromisoformat(row['at'].replace('Z', '+00:00'))
    for key, allowed in [('hint', ('landed', 'waiting-for-operator', 'progress', 'unknown')),
                         ('outcome', ('confirmed', 'uncertain', 'cancelled'))]:
        if key in row and row[key] not in allowed:
            raise ValueError('invalid enum')
    for key in ('generated', 'connected', 'settled', 'picker'):
        if key in row and not (type(row[key]) is bool or (key in ('settled', 'picker') and row[key] is None)):
            raise ValueError('invalid boolean')
    if 'submission' in row and not nonnegative(row['submission']):
        raise ValueError('invalid submission')
    if 'slots' in row:
        if not isinstance(row['slots'], list) or len(row['slots']) > 128:
            raise ValueError('invalid snapshot size')
        for slot in row['slots']:
            if not isinstance(slot, dict) or set(slot) != SLOT_FIELDS:
                raise ValueError('invalid snapshot fields')
            if not identity(slot['subject']) or not identity(slot['incarnation']) or type(slot['known']) is not bool:
                raise ValueError('invalid snapshot identity')
            for key in ('resting', 'settled', 'picker', 'pending'):
                if slot[key] is not None and type(slot[key]) is not bool:
                    raise ValueError('invalid snapshot boolean')
            for key in ('quiet_ms', 'submission'):
                maximum = MAX_INT64 if key == 'quiet_ms' else MAX_UINT64
                if slot[key] is not None and not nonnegative(slot[key], maximum):
                    raise ValueError('invalid snapshot counter')
    return row


def read_facts(paths):
    """Read a bounded batch; malformed rows suppress all negative labels."""
    records, invalid, total, count = [], 0, 0, 0
    for path in paths:
        with open(path, 'rb') as stream:
            while True:
                line = stream.readline(MAX_LINE + 1)
                if not line:
                    break
                count += 1
                total += len(line)
                if len(line) > MAX_LINE or total > MAX_BYTES or count > MAX_ROWS:
                    raise ValueError('input limit exceeded; select fewer or shorter log segments')
                try:
                    records.append(parse_fact(line))
                except (ValueError, TypeError, RecursionError, OverflowError):
                    invalid += 1
    if invalid:
        print(f'Skipped {invalid} invalid or unsupported rows; negative labels disabled.', file=sys.stderr)
    return records, bool(invalid)


class WriteIndex:
    """Range maximum sequence over write times, including damaged clocks."""

    def __init__(self, rows):
        rows = sorted((row['elapsed_ms'], row['seq']) for row in rows)
        self.times = [time for time, _ in rows]
        self.n = len(rows)
        self.tree = [0] * self.n + [seq for _, seq in rows]
        for i in range(self.n - 1, 0, -1):
            self.tree[i] = max(self.tree[i * 2], self.tree[i * 2 + 1])

    def matches(self, start, seq):
        left = bisect_right(self.times, start) + self.n
        right = bisect_right(self.times, start + HORIZON_MS) + self.n
        maximum = 0
        while left < right:
            if left % 2:
                maximum = max(maximum, self.tree[left])
                left += 1
            if right % 2:
                right -= 1
                maximum = max(maximum, self.tree[right])
            left //= 2
            right //= 2
        return maximum > seq


def derive_labels(records, gaps=False):
    """Yield deterministic labels from validated rows; no IO or online state."""
    generations = defaultdict(dict)
    conflicted = set()
    for row in records:
        bucket = generations[row['generation']]
        old = bucket.get(row['seq'])
        if old is not None and old != row:
            conflicted.add(row['generation'])
            # Neither version is reliable positive evidence. Remove the key below.
            bucket[row['seq']] = None
        elif row['seq'] not in bucket:
            bucket[row['seq']] = row
    for generation, bucket in sorted(generations.items()):
        rows = [row for _, row in sorted(bucket.items()) if row is not None]
        times = [row['elapsed_ms'] for row in rows]
        broken = generation in conflicted or any(a > b for a, b in zip(times, times[1:]))
        dropped = any(row['dropped'] for row in rows)
        writes = defaultdict(list)
        for row in rows:
            if row['event'] == 'session_write':
                writes[row['subject']].append(row)
        indexes = {subject: WriteIndex(items) for subject, items in writes.items()}
        # First discontinuity after each position gives constant-time coverage checks.
        ends = [len(rows) - 1] * len(rows)
        for i in range(len(rows) - 2, -1, -1):
            ends[i] = ends[i + 1] if rows[i + 1]['seq'] == rows[i]['seq'] + 1 else i
        for i, row in enumerate(rows):
            if row['event'] != 'hub_delivery' or row['generated']:
                continue
            index = indexes.get(row['subject'])
            label, reason = None, 'incomplete_horizon'
            if index is not None and index.matches(row['elapsed_ms'], row['seq']):
                label, reason = True, 'observed_write'
            elif gaps:
                reason = 'invalid_input'
            elif broken:
                reason = 'conflict_or_backwards_clock'
            elif dropped:
                reason = 'dropped_facts'
            # Equal timestamps may still precede a qualifying write.
            elif times[ends[i]] > row['elapsed_ms'] + HORIZON_MS:
                label, reason = False, 'complete_horizon'
            elif ends[i] < len(rows) - 1:
                reason = 'sequence_gap'
            yield {key: row[key] for key in ('generation', 'seq', 'at', 'elapsed_ms',
                                             'subject', 'sender', 'message', 'hint', 'outcome')} | {
                'label': label, 'reason': reason}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('logs', metavar='LOG', nargs='+')
    args = parser.parse_args(argv)
    try:
        rows, gaps = read_facts(args.logs)
    except (OSError, ValueError):
        print('Cannot read bounded input; check files and select fewer or shorter log segments.', file=sys.stderr)
        return 1
    for label in derive_labels(rows, gaps):
        print(json.dumps(label, separators=(',', ':')))
    return 0


if __name__ == '__main__':
    sys.exit(main())
