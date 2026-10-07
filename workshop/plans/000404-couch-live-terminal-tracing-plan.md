# Opt-in Couch live capture implementation plan (#404)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use bounded subagents for the recorder and terminal observation seam; keep integration in the main session. Steps use checkbox syntax for tracking.

**Goal:** Deliver the operator-requested isolated live capture needed to reproduce #379, without claiming the underlying display bug fixed.

**Architecture:** One explicitly opened recorder owns a bounded asynchronous ordered JSONL stream. Endpoint observations preserve parser/geometry ordering; a host writer decorator records actual accepted writes. Composition root alone reads activation environment and enforces isolation.

**Tech Stack:** Go, existing terminal Endpoint/Presenter, PtyRunner, hostty, isolated Couch runtime.

## Approval and scope

The operator approved the previously proposed two-boundary capture on 2026-10-03: “create a mode of this live capture ... add code and guard it ... only run in some isolated session.” That approval authorizes implementation of this capture, not a speculative rendering fix. #379 remains open pending a captured occurrence and diagnosis.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Observation | cmd/internal/terminal/observation.go | new |
| Capture record | cmd/internal/terminalcapture/record.go | new |

Observation carries kind, endpoint ID, geometry and borrowed input bytes. It is emitted synchronously under Endpoint's existing mutex; observers must copy bytes and never re-enter the endpoint. Capture records add schema version, sequence, wall timestamp, monotonic elapsed nanoseconds, boundary, exact bytes (JSON base64), and host write requested/accepted/error information. One record format is the authority for decoding; no shell-escaped byte strings.

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| Recorder | cmd/internal/terminalcapture/recorder.go | new | private session directory + append-only JSONL |
| Endpoint observer | cmd/internal/terminal/endpoint.go | modified | ordered Feed/Resize transitions |
| Observation injection | cmd/internal/ptychild/{child,terminal}.go and couchcore/ptyrunner.go | modified | per-child launch before first output |
| Captured host | cmd/internal/couchcmd/capture.go | new | actual host WriteContext/Size/Close |
| Activation | cmd/internal/couchcmd/{run,singleton}.go | modified | isolated runtime + env |
| Thread binding | cmd/internal/couchtty/console.go | modified | handle→thread identity |

## Design decisions

- Activation: `COUCH_CAPTURE_DIR=/absolute/isolated/root/captures` plus existing `COUCH_ISOLATED_ROOT=/absolute/isolated/root`. Disabled by default. Reject missing isolation, relative/escaping/symlink-escaping destinations before child launch. A unique mode-0700 subdirectory is created per run; files mode0600. No inherited default, global config change, or collection from another Couch process.
- Use a dedicated finite capture instead of diagnosticlog rotation: replay needs one ordered session with an explicit end/incomplete marker, while ordinary traces can rotate/drop independently. Reuse existing isolated-root validation and terminal transport boundaries (ARCH-DRY).
- `events.jsonl`: schema/start metadata, endpoint-open/feed/resize/end, host-geometry, host-write, endpoint/thread binding, optional selection context, and final status. Initial metadata includes PID/build information and wall start; timestamps correlate existing wrapper logs. Sequence orders admission, not proof of physical host paint. Host writes include duration and requested/accepted bytes; replay only accepted prefixes. Raw host input is not collected.
- Endpoint records occur inside its mutex immediately before parsing and after acknowledged geometry changes, never from delayed Console delivery. PTY input chunks and resizes therefore have a reliable relative order (ARCH-ORDER). Failed PTY/backend application is not reported as applied. Record geometry immediately after e.geometry changes, before commitReplies: a later reply failure does not erase an already-applied resize. Test this distinction. Observe before pump startup.
- Host decorator preserves optional contextual read and termination capabilities by embedding the concrete OSHost; Write delegates to WriteContext. Record every underlying write result, including partial/error writes. Host geometry comes from Size observations. Thread-binding metadata joins endpoint ID to repo-scope/tag/actor; do not log complete child environments or argv.
- Workload: optional diagnostic terminal hot path. No file IO or goroutine-per-event on capture calls. A bounded queue (8 MiB total payload, bounded record count) feeds one writer. On overflow or a disk cap (256 MiB/session), stop capture and mark incomplete; never silently drop then resume. Continue terminal operation. Disk failure is reported at teardown; successful capture ends with an explicit completion record. Missing final status means incomplete (including process crash). Document that queue copying/timestamping still perturbs timing.
- Open/configuration failure is a startup error when capture was explicitly requested; never silently run without evidence. Close is idempotent: active→draining rejects new records, admitted records drain, then closed; overflow/disk errors enter failed and cannot resume. Drain waits at most two seconds and reports timeout. Regular-file IO cannot reliably be canceled by Go; the single existing worker remains recorder-owned until that IO returns or the isolated process exits, with no new workers/admission. Terminal restoration precedes this bounded wait. Test blocked-write timeout, subsequent unblock/worker exit, and concurrent Close/admission. A completion marker describes persisted admitted records; missing/truncated marker means incomplete. Capture is closed on early launch errors as well as normal teardown. Capture directories are user-managed and never auto-uploaded.
- ARCH-PURE: record shape and validation separated from file/clock seams. ARCH-MOCK: tests use real temp files and existing stateful fake Host/Child; no new external binary dependency. ARCH-SECURE: output may contain visible private content; explicit isolated opt-in and private files, parser consumers must reject unknown schema or truncated final records. ARCH-CONSTRAINTS: finite memory/disk, explicit incomplete state. ARCH-PURPOSE: both requested boundaries plus usable activation/runbook; no claim that capture fixes #379. ARCH-FUNERAL: command ownership closes capture on every exit, after terminal restoration; private session artifacts are retained for operator-managed deletion. Configuration and observers are injected explicitly, without low-level ambient env reads.

## Chunk 1: Capture implementation and verification

### Task 1 — Recorder contract

Files: new `cmd/internal/terminalcapture/{record,recorder,recorder_test}.go`.

- [x] Write failing tests for exact arbitrary bytes roundtrip, sequences/timestamps, private unique sessions, concurrent producers, successful close, bounded queue/disk cap and disk failure producing an incomplete capture.
- [x] Run `go test ./cmd/internal/terminalcapture -count=1`; confirm the missing implementation fails.
- [x] Implement one bounded asynchronous recorder with explicit lifecycle and nil-safe off path. Keep IO injectable for deterministic blocked-writer/error tests; production uses ordinary files.
- [x] Run package tests and race tests; verify admitted byte ownership survives caller mutation.

### Task 2 — Ordered terminal observation

Files: new `cmd/internal/terminal/observation.go`; modify `terminal/endpoint.go`, `ptychild/{child,terminal}.go`, `couchcore/ptyrunner.go`; colocated tests.

- [x] Write failing tests observing open, byte-split Feed, successful/failed resize, EOF; ensure original output and query replies unchanged.
- [x] Add optional observer to Endpoint constructor and pass it from PtyRunner via ptychild.Options before pump starts. Default nil changes no environment/file behavior.
- [x] Run `go test ./cmd/internal/terminal ./cmd/internal/ptychild ./cmd/internal/couchcore -count=1`; assert observer initial geometry precedes first feed.

### Task 3 — Isolated activation and host capture

Files: new `cmd/internal/couchcmd/capture.go`, tests; modify `couchcmd/{run,singleton}.go`, `couchtty/console.go` for identity metadata if needed.

- [x] Write failing tests: disabled activation produces no file; opt-in without isolation rejected; escaping path/symlink rejected; explicit invalid destination fails startup; valid isolated run records both seams.
- [x] Decorate host preserving optional host interfaces; record exact write receipts, partial/error results, and geometry. Ensure close runs on early errors and after normal Presenter/child release.
- [x] Wire recorder observer into PtyRunner; bind IDs to thread metadata. Keep diagnostic failures separate from terminal IO errors and surface incomplete status.
- [x] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty -count=1`; use an isolated PTY integration fixture to prove real ingress plus host output capture and teardown.

### Task 4 — Operator runbook and review

Files: `atlas/` existing Couch diagnostic documentation (linked from atlas/index.md), issue Log.

- [x] Document exact isolated launch command, environment overrides to clear, capture file schema, limits/incomplete detection, how to retain timestamp of a flash, and how to reconstruct each stream from JSON base64. Note auth/setup follows existing isolated-runtime behavior.
- [x] Run focused tests + `go test -race` for changed concurrent components; `git diff --check`; build binary. Demonstrate an isolated capture with no production runtime effects.
- [x] Checkpoint implementation and request fresh-eyes review using the SDLC checkpoint appropriate to partial issue delivery; do not close #379 before its occurrence-analysis acceptance criterion is met.

## Revisions

- 2026-10-03 — Operator authorized guarded live capture after complete-prefix replay failed to reproduce. This plan delivers instrumentation while preserving the original bug investigation's outstanding acceptance criterion.

- 2026-10-03 — Plan review clarified applied-resize ordering and bounded recorder shutdown. The explicit OS-file cancellation limitation is retained rather than claiming cancellable regular-file IO.

- 2026-10-03 — Implementation reviewed and verified. Command owns recorder closure after terminal restoration on normal exit and reports failures on early launch errors too. Five new production sources registered in the artifact inventory; remaining inventory failures reproduced unchanged in a generated baseline. #379 remains working pending a real captured occurrence.

- 2026-10-07 — Operator clarified that “isolated session” meant explicitly enabling capture for one regular Couch launch, not isolated storage/HOME. Supersedes the mandatory-isolation requirement above: allow an absolute COUCH_CAPTURE_DIR without COUCH_ISOLATED_ROOT; retain confinement when isolation is explicitly requested. Keep opt-in/default-off, private bounded capture, and child activation clearing. Extend real-PTY coverage to both regular and isolated activation and simplify runbook.

- 2026-10-07 — Operator separated implementation into #404 and retained #379 for diagnosis. This plan is transferred intact with historical decisions and verification preserved; the revision allowing regular-session opt-in supersedes original isolation requirements. Original code provenance: 9ca17478 and 0b384b45, preserved on archive/000379-before-tracing-split.
- 2026-10-07 — Real regular capture stopped after 6.88 seconds / 5.35 MiB with queue-limit failure. Earlier completed checkboxes describe the imported work, not readiness to ship. ARCH-CONSTRAINTS and ARCH-PURPOSE require realistic burst handling and useful long-running capture before delivery. Failure visibility and retention design remain pending; no new runtime behavior is implemented by this transfer.

## Remaining work before delivery

- [x] Revise design for realistic startup bursts, prompt in-session failure visibility, and bounded long-running retention; carry it through the #404 change-code gate before new implementation.
- [x] Add regression coverage from the observed workload and implement the approved reliability changes.
- [x] Verify regular-session capture survives startup and remains usable for waiting on #379; document limits and evidence gaps honestly.
- [x] Prepare #404 for its independent close review and publication; keep #379 open.


## 2026-10-07 reliability revision — proposed completion design

This revision supersedes the imported queue and failure-visibility decisions;
all previous verification remains historical. Use the existing full-prefix file
format; rolling arbitrary ANSI history would discard parser/screen state needed
for faithful reproduction. Keep a finite configurable disk budget and make its
usage and stop condition persistent in Couch. The implementation uses this recommendation under the operator’s authorization
to finish #404; an optional retention preference question received no correction.

### Evidence and bounded workload

The failed regular capture has 3,280 records and 3,722,498 raw data bytes over
6.88 seconds. 3,228 are endpoint-feed records; the largest payload is 18,851
bytes. The busiest observed 100 ms contains 279 records / 277,881 payload bytes.
A 128-record bound can fail with well under the 8 MiB memory budget. Replace it
with 8,192 records while retaining the 8 MiB admission-cost bound (including
in-flight data and string/record overhead). This holds the entire observed burst
with writer progress paused. It is a measured envelope, not a promise to keep up
with indefinitely blocked storage. No producer waits for disk. Test a sanitized
4,000-record / 1 KiB burst with a deliberately blocked writer, then verify exact
order and bytes after drain. Replay the actual capture locally, without checking
private contents into Git. No buffering optimization without measurement.

### Core concepts and integration

| Entity | Lives in | Status |
|---|---|---|
| Capture status snapshot (phase, persisted bytes, limit, failure) | terminalcapture/recorder.go | new |
| Admission and completion lifecycle | terminalcapture/recorder.go | refined |
| Capture status badge | couchtty/reserve.go | new |
| Capture configuration | couchcmd/capture.go | extended |

Recorder phase transitions: recording + admit -> recording; recording + close
-> draining; recording/draining + failure -> failed; draining + successful
writer close -> closed. Failed stays failed; no later data admission. Repeated
close keeps its existing idempotent receipt and bounded wait. No caller may
mutate phase. Record and worker transitions/status share the recorder mutex.
Persisted-byte counters advance from write receipts, not offered data; a complete
end still means admitted-stream completeness, not successful command shutdown.

Recorder exposes nil-safe Status() and a bounded coalesced Changes() notification
channel. Notify on phase/failure or whole-percent usage changes, not every record;
read the authoritative snapshot after notification. Never call Console from the
recorder. Console.Run selects Changes alongside existing events, then uses its
Presenter-owned repaint. Initial paint reads status even if failure preceded Run.
No new goroutine, stderr output, timer, or closed-channel busy loop. Badge precedes
actor chips, never owns a click target, is clipped normally, and remains visible
in actor and switcher views. Failures show a compact fixed reason (queue/full/IO),
not unsanitized filesystem/error text. Disabled mode remains unchanged.

Configuration: retain default 256 MiB complete capture; add strict positive
COUCH_CAPTURE_MAX_MIB override (bounded to a documented safe maximum), meaningful
only with explicit COUCH_CAPTURE_DIR. Validate before launch; clear both from
children. Expose configuration via Open options without breaking existing callers.
Status shows capture usage percentage and persistent stopped reason. Once stopped,
restart capture with a larger budget; no silent resumption or discarded prefix.
Runbook gives a 4 GiB opt-in example for a longer wait and explains duration depends
on recorded traffic. No guarantee that any finite cap covers an indefinite wait.

ARCH-DRY: use existing recorder, status row and Presenter rather than a second
terminal writer. ARCH-ORDER: explicit phases and coalesced wakeups; missed/coalesced
notifications cannot erase failure. ARCH-CONSTRAINTS: admitted record cost remains 8 MiB, record
count 8,192, disk finite/configurable, no new producer blocking. ARCH-PURPOSE:
full-prefix evidence and immediate failure visibility support #379 without claiming
a root cause. ARCH-PURE/SECURE: typed status and strict config parsing; private
files and existing exact-byte schema remain. ARCH-MOCK: existing stateful faultSink,
fake Console host and real-PTY integration. ARCH-FUNERAL: per-run files remain
operator-retained evidence within the selected finite cap; runbook states per-run
cost and explicit deletion responsibility rather than auto-deleting investigation
evidence. No new persistent artifact family or external service.

### Completion tasks (one close/review boundary)

- [x] Write failing blocked-startup-burst and status/wakeup lifecycle tests; implement
  queue correction and explicit recorder phase/status with receipt accounting.
- [x] Write invalid/valid disk-limit and child-environment tests; implement config
  at couchcmd composition root, preserving optional isolation and default-off.
- [x] Write status-row and idle failure repaint tests (actor and switcher), then wire
  recorder changes through Console.Run. Cover pre-Run failure and narrow rows.
- [x] Replay local measured workload; extend real-PTY regular/isolated fixture with
  sustained startup output and assert both exact boundaries survive capture.
- [x] Update runbook (limits, stopped state, larger-budget launch, retention and
  extraction), run relevant suites/race checks and build. Close review and ship
  follow as SDLC gates for #404 only.

- 2026-10-07 — Fresh-eyes plan review clarified memory accounting: 8 MiB bounds
  admitted record cost, not process RSS. The fixed 8,192-element channel backing
  array and one in-flight JSON encoding allocation are additional bounded costs;
  JSON escaping can expand string fields up to 6x (byte Data uses base64). No claim
  of 8 MiB total memory. Test a near-limit individual record with a blocked writer
  as well as the many-small-record workload. Existing capture copies/encoding
  perturb timing; this is diagnostic overhead, not a real-time guarantee.

- 2026-10-07 — Completion implementation maps the phase/status entity to
  `terminalcapture.Phase`, `Status`, `Recorder.Status`, `Recorder.Changes`; config
  to `terminalcapture.Config` and `couchcmd.captureSettings`; badge to
  `couchtty.captureBadge`, `StatusModel.Capture` and the Console.Run select arm.
  No new production source files were needed beyond the transferred inventory.
  Fresh-eyes plan review approved the revised accounting. TDD reproduced queue
  overflow, invalid/ignored cap settings, inherited child limit and missing idle
  status updates before their fixes. Removing the Console change-notification
  repaint arm makes both actor/switcher idle tests fail.
- 2026-10-07 — Actual private workload replay: 3,278 observations / 3,722,498 bytes,
  no-delay admission plus drain in 100.7 ms, exact field/payload comparison and
  complete end. Private payload was not committed; temporary replay probe removed.
  Real-PTY Console tests in regular and isolated configurations each captured
  3,800 KiB startup output exactly and painted the final host marker (5.51 s total).

- 2026-10-07 — Delivery checklist distinguishes completed implementation/verification
  from the subsequent close and merge gates. Full repository assertions that failed
  were reproduced on main (see issue Log); the cumulative couchcore timeout was
  completed by a separate passing run of its 126 remaining tests. No baseline bug
  is claimed fixed. Runbook large-capture extractor verified with disposable bytes.


## 2026-10-07 boundary-review revision

The first close review raised BR-1 (README), BR-2 (aggregate retention), and BR-3
(pure lifecycle authority). Address the classes without deleting evidence or
changing the operator-approved complete-prefix choice. This supersedes prior
per-session-only retention and the proposed 1 TiB configuration maximum.

- BR-1: document both activation variables, defaults/ranges, recording/stopped
  indicator and storage admission in README, linked to the detailed atlas runbook.
- BR-2: `terminalcapture/storage.go` owns admission under a persistent private
  parent lock (nonblocking flock; busy refuses startup). Reserve each session's
  full allowance in a private versioned budget.json before returning its file.
  Enforce 32 GiB aggregate reserved stream bytes and 64 session directories.
  Per-session configuration is consequently limited to 1 MiB–32 GiB. Reservations
  survive close/crash; removing or moving the particular evidence session frees
  admission. Never automatically delete recordings. Validate bounded metadata,
  reject symlink/corrupt/unknown sessions and files larger than their reservation.
  Legacy sessions lacking metadata are charged by actual size only when their
  final bounded JSONL tail proves a version-one complete/incomplete capture-end;
  otherwise refuse with a fresh-directory/preserve-evidence instruction. Lock and
  metadata overhead is bounded separately from recorded stream bytes. Test repeated
  and concurrent admissions, count/byte exhaustion, old/corrupt/unfinished evidence,
  permissions and no existing-data loss. No per-record disk scan or lock added.
- BR-3: `terminalcapture/lifecycle.go` encapsulates authoritative phase/error state;
  a pure state/event transition yields close-admission and notify effects. Recorder
  executes those effects; writer completion, failure and close timeout cannot
  mutate phase directly. Tests exercise normal/repeated close, queue/IO failure,
  timeout and late completion through this production model and concurrent recorder.
- Register both new source files in the artifact inventory. Rerun affected package,
  race and real-PTY tests; update all configuration range/readback tests and docs.

Operator explicitly confirmed “Complete capture with visible limit (recommended)”
during the first close review. Review refusal was not bypassed; re-close after fixes.
