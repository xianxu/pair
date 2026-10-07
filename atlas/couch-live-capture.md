# Opt-in Couch live capture

For #379, capture the two display boundaries of one explicitly opted-in Couch
process: Zellij output before Couch parses it, and Couch output accepted by the
host terminal. Ordinary Couch sessions do not record these streams.

## Start a capture on regular Couch

Exit the Couch UI normally, leaving Pair/Zellij threads available to reattach.
Then launch the new binary from a terminal outside Couch, with the capture setting
applied only to that command:

```sh
COUCH_CAPTURE_DIR="$HOME/.local/share/pair/captures" \
  /Users/xianxu/workspace/worktree/pair-slot2/pair/bin/couch "$HOME/workspace/pair"
```

Use your usual repository argument instead of `$HOME/workspace/pair` if different.
For another installation, build this branch and use its Couch binary. Capture is
read at process startup; setting it inside an already-running Couch has no effect.
No isolated HOME, scratch clone or new provider login is required. Existing Pair
threads can be resumed/reattached inside the traced Couch; no per-thread setting
or agent restart is required. Only output arriving after attachment is recorded.

`COUCH_CAPTURE_DIR` must be an absolute directory. It is off when unset or empty;
invalid/unopenable destinations fail startup. Activation is cleared from launched
child environments, so this setting belongs to the explicitly launched Couch.
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

Recording uses one asynchronous writer, at most 128 queued/in-flight records and
8 MiB retained record data. The file is capped at 256 MiB. Exceeding a limit stops
capture permanently and writes an incomplete ending when possible; it does not
block terminal rendering or silently resume after a gap. Disk errors are reported
when the owning command exits, with nonzero status. Keep sessions short and start
a fresh one if a capture fills. Copying and timestamps still perturb timing.

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
# Validate before producing replay files; never treat truncation as silence.
rows = [json.loads(line) for line in source.open()]
assert rows and rows[0]['kind'] == 'capture-start'
assert all(r['version'] == 1 and r['seq'] == i + 1 for i, r in enumerate(rows))
assert rows[-1]['kind'] == 'capture-end' and rows[-1]['status'] == 'complete'
streams = {}
for r in rows:
    data = base64.b64decode(r.get('data', ''), validate=True)
    if r['kind'] == 'host-write':
        assert r['requested'] == len(data) and 0 <= r['accepted'] <= len(data)
        streams.setdefault('host', bytearray()).extend(data[:r['accepted']])
    elif r['kind'] == 'endpoint-feed':
        # Encode the ID for use as a filename rather than trusting path text.
        name = 'endpoint-' + r['endpoint_id'].encode().hex()
        streams.setdefault(name, bytearray()).extend(data)
for name, data in streams.items():
    target = source.parent / (name + '.raw')
    with os.fdopen(os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'wb') as f:
        f.write(data)
    print(target)
PY
```

Raw concatenation alone discards geometry and chunk timing. For a faithful replay,
use the corresponding endpoint open/resize records and feed chunks at their
recorded offsets/times; compare host accepted writes around the reported flash.
This instrumentation helps locate the faulty layer; it does not itself fix #379.
