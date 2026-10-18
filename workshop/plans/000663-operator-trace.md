# Operator trace candidate for pair#663

Not executed. Use only if the approved observer comparison leaves attribution open and the hub arranges an authorized final-Go run. This command starts no suite. Run it when the final recipe prints `go test ./... -count=1 -timeout 20m`.

The proposed scope is filesystem, disk I/O and exec events for **all host writers during 60 seconds**. This scope needs the operator's agreement: the missing writer is unknown, and fixed PID filters omit short-lived compiler/Git children and unrelated producers. A narrower command-name filter would answer only whether those named processes write. Keep the raw trace local; it can contain unrelated file paths. No whole-host census is built.

The native timeout bounds tracing. The wrapper also stops at 60 seconds or 128 MiB, retains output in memory until tracing stops, and reaps its own tracer. A forced cleanup can add two seconds. An output cap or early exit makes the trace partial. There is no automatic retry.

```sh
capture_dir=$(mktemp -d "${TMPDIR:-/tmp}/pair663-fsusage.XXXXXX") || exit 1
sudo /usr/bin/python3 - "${capture_dir:?}" <<'PY'
import json, os, selectors, signal, subprocess, sys, time
from pathlib import Path

out = Path(sys.argv[1])
p = None
chunks, size = [], 0
limit = 128 * 1024 * 1024
receipt = {"start_epoch": time.time(), "limit_bytes": limit,
           "scope": "all host processes; filesys, diskio, exec; 60 seconds"}
reason = "60-second deadline"

def interrupted(sig, frame):
    raise KeyboardInterrupt("signal " + str(sig))

for sig in (signal.SIGINT, signal.SIGTERM, signal.SIGHUP):
    signal.signal(sig, interrupted)
try:
    p = subprocess.Popen(
        ["/usr/bin/fs_usage", "-w", "-f", "filesys", "-f", "diskio",
         "-f", "exec", "-t", "60"],
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, start_new_session=True)
    sel = selectors.DefaultSelector()
    sel.register(p.stdout, selectors.EVENT_READ)
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        if not sel.select(min(.2, max(0, deadline - time.monotonic()))):
            continue
        block = os.read(p.stdout.fileno(), min(65536, limit - size))
        if not block:
            reason = "tracer output ended; inspect exit and diagnostic lines"
            break
        chunks.append(block)
        size += len(block)
        if size == limit:
            reason = "128 MiB output cap; partial capture"
            break
except BaseException as exc:
    reason = repr(exc)
finally:
    for sig in (signal.SIGINT, signal.SIGTERM, signal.SIGHUP):
        signal.signal(sig, signal.SIG_IGN)
    if p is not None:
        if p.poll() is None:
            try:
                os.killpg(p.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
            try:
                p.wait(timeout=2)
            except subprocess.TimeoutExpired:
                os.killpg(p.pid, signal.SIGKILL)
                p.wait()
        receipt["exit"] = p.returncode
    receipt.update(end_epoch=time.time(), captured_bytes=size, reason=reason)
    (out / "fs_usage.log").write_bytes(b"".join(chunks))
    (out / "receipt.json").write_text(json.dumps(receipt, indent=2))
    if "SUDO_UID" in os.environ:
        for path in out.iterdir():
            os.chown(path, int(os.environ["SUDO_UID"]), int(os.environ["SUDO_GID"]))
    print(str(out))
    print(json.dumps(receipt))
PY
```

The local fs_usage manual documents multiple `-f` filters, command/PID filters and `-t`. This command has been checked against that manual, not run with root privileges. SIGKILL of the wrapper cannot run cleanup; fs_usage still has its native timeout. Inspect trace-loss notices and process/path coverage before interpreting totals. Syscall requested byte counts, APFS counters and device write bytes are different views and cannot simply be added. The trace can identify write/metadata/flush activity, but does not guarantee full ancestry or exact physical-byte attribution to a process.
