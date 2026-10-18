# Operator trace for pair#663: final Go and no-suite background

Not executed. Hub approved this handoff in receipt 99719b95-da83-4047-9c8d-1f380df71cbb; the operator runs it. This starts no suite and changes no machine admission policy. Arrange an already-authorized suite and then a period with **no full suite running anywhere on the machine**. Keep other full suites out of both windows through normal coordination.

Paste once in an operator terminal before final Go. At the first prompt, type FINAL_GO when its recipe prints `go test ./... -count=1 -timeout 20m`. After that capture, wait for all suites to finish, then type NO_SUITES at the second prompt. These are operator attestations; this command does not verify them. Each prompt allows 30 minutes; timeout retains any completed evidence and stops. Ctrl+C cancels and retains partial evidence.

Each arm captures **all host writers**, filtered to filesystem, disk I/O and exec events, for at most 60 seconds and 128 MiB. Fixed PID filters would omit brief Go/Git children and unknown background writers. Keep traces local: they can contain unrelated file paths. Both arms retain bounded payload in memory (at most 256 MiB combined, plus Python overhead) until tracing ends; only then are logs written below TMPDIR, avoiding trace-log writes during either capture. The output cap makes a trace partial, not a pass. There is no automatic retry.

```sh
capture_dir=$(mktemp -d "${TMPDIR:?TMPDIR must be set}/pair663-fsusage.XXXXXX") || exit 1
sudo /usr/bin/python3 - "${capture_dir:?}" <<'PY'
import json, os, selectors, signal, subprocess, sys, time
from pathlib import Path

out = Path(sys.argv[1])
limit = 128 * 1024 * 1024
captures, receipts = {}, []
p = None
failure = None

def interrupted(sig, frame):
    raise KeyboardInterrupt("signal " + str(sig))

for sig in (signal.SIGINT, signal.SIGTERM, signal.SIGHUP):
    signal.signal(sig, interrupted)

def stop_tracer():
    if p is None:
        return
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

try:
    with open("/dev/tty", "r") as tty_in, open("/dev/tty", "w", buffering=1) as tty_out:
        for arm, token, condition in (
            ("final-go", "FINAL_GO", "suite final Go recipe has just started"),
            ("no-suites", "NO_SUITES", "no full suite is running anywhere on this machine")):
            tty_out.write("\nWhen " + condition + ", type " + token + " then Return (30 min bound): ")
            tty_out.flush()
            ready = selectors.DefaultSelector()
            ready.register(tty_in, selectors.EVENT_READ)
            if not ready.select(1800):
                raise TimeoutError("condition prompt expired: " + arm)
            answer = tty_in.readline().strip()
            ready.close()
            if answer != token:
                raise RuntimeError("condition not confirmed: " + arm)
            receipt = {"arm": arm, "attestation": answer, "start_epoch": time.time(),
                       "scope": "all host processes; filesys, diskio, exec",
                       "limit_bytes": limit, "limit_seconds": 60}
            receipts.append(receipt)
            chunks, size = [], 0
            captures[arm] = chunks
            reason = "60-second deadline"
            p = subprocess.Popen(
                ["/usr/bin/fs_usage", "-e", "-w", "-f", "filesys", "-f", "diskio",
                 "-f", "exec", "-t", "60"],
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, start_new_session=True)
            sel = selectors.DefaultSelector()
            sel.register(p.stdout, selectors.EVENT_READ)
            deadline = time.monotonic() + 60
            try:
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
                raise
            finally:
                stop_tracer()
                receipt.update(end_epoch=time.time(), captured_bytes=size,
                               exit=p.returncode, reason=reason)
                p.stdout.close()
                sel.close()
                p = None
            tty_out.write("\n" + arm + " capture stopped: " + reason + "\n")
            tty_out.flush()
except BaseException as exc:
    failure = repr(exc)
finally:
    for sig in (signal.SIGINT, signal.SIGTERM, signal.SIGHUP):
        signal.signal(sig, signal.SIG_IGN)
    stop_tracer()
    # Persist only after both traces stop, including partial evidence on cancellation.
    for arm, chunks in captures.items():
        with (out / (arm + ".log")).open("wb") as f:
            for block in chunks:
                f.write(block)
    (out / "receipt.json").write_text(json.dumps(
        {"arms": receipts, "failure": failure}, indent=2))
    if "SUDO_UID" in os.environ:
        for path in out.iterdir():
            os.chown(path, int(os.environ["SUDO_UID"]), int(os.environ["SUDO_GID"]))
    print(str(out))
    print(json.dumps({"arms": receipts, "failure": failure}))
if failure:
    sys.exit(1)
PY
```

Inspect fs_usage exits, trace-loss notices, elapsed intervals and output-cap reasons before interpreting either arm. Native `-t 60` and a separate wrapper deadline bound tracing; forced cleanup may add two seconds. SIGKILL of the wrapper cannot run cleanup, though the tracer keeps its native timeout. Condition waiting is bounded separately and is not a trace interval. Requested syscall bytes, APFS counters and physical device bytes cannot be added; the trace does not guarantee complete ancestry or exact physical-byte attribution. A no-suite window may still contain earlier writeback, which is part of the background question rather than proof of an unrelated application.

## Revisions

- 2026-10-18: Replaced the unexecuted single-arm candidate (commit 302dc60c9) with the hub-requested two-arm paste. Added explicit operator condition confirmations, 30-minute prompt bounds, separate 128 MiB arm caps, and delayed persistence of both logs under required TMPDIR. All-host event scope is approved for the operator handoff; this slot executes no privileged trace.

- 2026-10-18 pre-handoff review: use separate terminal read/write handles rather than a seek-requiring buffered update stream. Add fs_usage -e with no process list to request exclusion of only the tracer rather than the default shell/terminal exclusions. Retained-payload limits do not claim to bound Python overhead.
