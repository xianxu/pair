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

- [ ] Revise design for realistic startup bursts, prompt in-session failure visibility, and bounded long-running retention; carry it through the #404 change-code gate before new implementation.
- [ ] Add regression coverage from the observed workload and implement the approved reliability changes.
- [ ] Verify regular-session capture survives startup and remains usable for waiting on #379; document limits and evidence gaps honestly.
- [ ] Complete independent review and ship #404 without closing #379.
