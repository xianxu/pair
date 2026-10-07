# Opt-in Couch live capture

For #379, capture the two display boundaries of one explicitly opted-in Couch
process: Zellij output before Couch parses it, and Couch output accepted by the
host terminal. Ordinary Couch sessions do not record these streams.

## Start a capture on regular Couch

Exit the Couch UI normally, leaving Pair/Zellij threads available to reattach.
Then launch the new binary from a terminal outside Couch, with the capture setting
applied only to that command:

```sh
COUCH_CAPTURE_DIR="$HOME/.local/share/pair/captures" COUCH_CAPTURE_MAX_MIB=4096 \
  /Users/xianxu/workspace/worktree/pair-slot2/pair/bin/couch "$HOME/workspace/pair"
```

Use your usual repository argument instead of `$HOME/workspace/pair` if different.
For another installation, build this branch and use its Couch binary. Capture is
read at process startup; setting it inside an already-running Couch has no effect.
No isolated HOME, scratch clone or new provider login is required. Existing Pair
threads can be resumed/reattached inside the traced Couch; no per-thread setting
or agent restart is required. Only output arriving after attachment is recorded.

`COUCH_CAPTURE_DIR` must be an absolute directory. It is off when unset or empty;
invalid/unopenable destinations fail startup. `COUCH_CAPTURE_MAX_MIB` optionally
sets the per-session disk budget in whole MiB (1–32768); the default is 256 MiB.
The example chooses 4 GiB for a longer wait. A limit alone does not enable capture.
When capture is enabled, invalid limits fail before any capture file is created.
Both capture settings are cleared from launched child environments, so activation
belongs to the explicitly launched Couch.
If you also set `COUCH_ISOLATED_ROOT`, the capture directory must remain inside
that root, including after symlink resolution. Isolation is optional.

When the flash occurs, note its wall-clock time and visible thread, then exit
Couch normally. Retain the whole `captures/session-*/` directory. Each run creates
a new mode-0700 directory with mode-0600 `events.jsonl`; it contains unredacted
display bytes. Nothing is uploaded or automatically removed. To disable capture,
launch Couch again without `COUCH_CAPTURE_DIR`.

## Capture contract

One JSON object per line, schema `version: 1`:

- `seq` and `elapsed_ns`: admission order and monotonic time since recorder start.
  `time` is UTC wall time for correlation with wrapper logs.
- `capture-start`: process ID and Go/build revision metadata.
- `endpoint-open`, `endpoint-feed`, `endpoint-resize`, `endpoint-end`:
  endpoint identity, dimensions, and exact pre-parser `data` (JSON base64).
  Input and applied resize observations share the Endpoint lock. An applied
  resize is recorded even if subsequently sending a terminal reply fails.
- `thread-bind`: joins `endpoint_id` to repository `scope`, thread `tag` and
  `actor`. It can follow initial endpoint bytes because children may write before
  attachment completes.
- `host-geometry`: dimensions returned by the host query, or an `error`.
- `host-write`: requested `data`, `requested`, `accepted`, optional `error`, and
  `duration_ns`. Its timestamp is after the underlying write returns. Replay
  only `data[:accepted]`; a failed write may still have accepted a prefix.
- `select-start`/`select-end` and `panel-start`/`panel-end`: brackets around
  presentation requests, including failures. They do not assert host paint time.
- `capture-end`: `status: complete` only when all admitted records were written;
  otherwise `incomplete` with an error. Missing/truncated end, unknown schema or
  a sequence gap must not be treated as a complete replay.

The capture includes all children attached to this Couch, so background
Claude completions can be distinguished. It records no host keystroke stream,
child environments or full command lines. Display output can still echo input.

The status row starts with `REC N%` while recording. Usage counts bytes accepted
by file writes, not an fsync guarantee. The badge updates on whole-percentage
changes; it stays visible in both actor and switcher views. `REC STOP:queue`,
`REC STOP:full`, or `REC STOP:IO` means recording has stopped and later incidents
will **not** be captured. The badge takes precedence over actor chips and is not
clickable; very narrow terminals can clip it. Full error details are reported
when the owning command exits, with nonzero status.

Recording uses one asynchronous writer, at most 8,192 queued/in-flight records
and 8 MiB of admitted record cost (payload, string data and per-record allowance).
The fixed channel and one in-flight JSON encoding add bounded memory overhead;
8 MiB is not a process-memory ceiling. The disk budget includes the ending marker.
Exceeding a queue/disk limit stops capture permanently and writes an incomplete
ending when possible; it does not wait for disk on the rendering path or silently
resume after a gap. Copying and timestamps still perturb timing.

Capture preserves the full prefix: it does not rotate away terminal parser and
screen history needed for replay. A finite budget cannot cover an indefinite
wait. Duration depends on traffic: 4 GiB lasts about 11.7 hours at 100 KiB/s or
68 minutes at 1 MiB/s of encoded capture growth. Startup bursts are not a steady
rate estimate. Watch the percentage; restart with a larger budget before it fills.
Each restart reserves its full configured allowance against a **32 GiB directory
budget**, with at most **64 sessions**. Reservations remain after close or crash;
unused reserved capacity is intentionally not reclaimed automatically. Admission
uses a private lock and versioned `budget.json` beside each `events.jsonl`. A busy
lock or exhausted budget refuses startup. To free a reservation, move a specific
saved session to your evidence archive or remove it when no longer needed; never
move or remove a session that is still recording. No recording is auto-deleted.
These limits bound recorder-created stream storage; separately extracted raw files
and copies belong to the operator's evidence archive.

Finished legacy recordings without budget metadata count at their actual file
size only when the ending record proves recording stopped. Unfinished, corrupt or
unrecognized recordings refuse admission rather than being assumed disposable;
retain them and use a fresh capture directory. Budget metadata must not be edited
to claim capacity: mismatches or unknown versions refuse admission. Each capture
contains unredacted display data; retain the whole session directory for incidents.

Shutdown restores the terminal before draining capture, with a two-second drain
limit. A filesystem write cannot reliably be canceled; after timeout the existing
writer may finish only when the OS returns or the process exits. The command
reports that failure. A late complete marker describes persisted admitted data,
not successful command shutdown. A killed process normally has no ending marker.

## Inspect and reconstruct

Do not print the decoded escape stream into a working terminal. Read JSON metadata
or save bytes for replay in a disposable terminal/emulator. For example, this
extracts a validated complete capture into separate boundary files while retaining
`events.jsonl` as the timing/resize authority:

```sh
python3 - "$HOME/.local/share/pair/captures/session-REPLACE/events.jsonl" <<'PY'
import base64, json, os, pathlib, sys
source = pathlib.Path(sys.argv[1])
# Two streaming passes: validate the whole capture before producing replay files.
# Stop Couch first so the file cannot grow between passes.
last = None
with source.open() as f:
    for seq, line in enumerate(f, 1):
        r = json.loads(line)
        assert r['version'] == 1 and r['seq'] == seq
        assert last is None or last['kind'] != 'capture-end'
        if seq == 1:
            assert r['kind'] == 'capture-start'
        data = base64.b64decode(r.get('data', ''), validate=True)
        if r['kind'] == 'host-write':
            assert r['requested'] == len(data) and 0 <= r['accepted'] <= len(data)
        last = r
assert last and last['kind'] == 'capture-end' and last['status'] == 'complete'
created = set()
with source.open() as f:
    for line in f:
        r = json.loads(line)
        data = base64.b64decode(r.get('data', ''), validate=True)
        if r['kind'] == 'host-write':
            name, data = 'host', data[:r['accepted']]
        elif r['kind'] == 'endpoint-feed':
            # Encode the ID rather than trusting it as a filesystem path.
            name = 'endpoint-' + r['endpoint_id'].encode().hex()
        else:
            continue
        target = source.parent / (name + '.raw')
        flags = os.O_WRONLY | (os.O_APPEND if name in created else os.O_CREAT | os.O_EXCL)
        with os.fdopen(os.open(target, flags, 0o600), 'ab') as out:
            out.write(data)
        if name not in created:
            created.add(name)
            print(target)

PY
```

Raw concatenation alone discards geometry and chunk timing. For a faithful replay,
use the corresponding endpoint open/resize records and feed chunks at their
recorded offsets/times; compare host accepted writes around the reported flash.
This instrumentation helps locate the faulty layer; it does not itself fix #379.
