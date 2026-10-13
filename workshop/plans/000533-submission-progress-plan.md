# Submission Progress Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** Make DOM submission visibly active and hold its draft read-only only for one bounded operation, always releasing it on failure.

**Architecture:** Reuse the existing submission state reducer, status RPC, preparation worker, and draft statusline. Add informational typed progress to the preparation callback chain. The reducer owns active operation identity and terminal outcome; the Neovim adapter owns buffer options, redraw, timers and bounded subprocess execution.

**Tech Stack:** Go, Lua/Neovim, CDP, existing headless and stateful terminal fixtures.

## Core concepts

| Entity | Kind | Location | Change |
|---|---|---|---|
| SubmissionProgress | PURE value | cmd/internal/submission/model.go | new phase/current/total value; not durable delivery state |
| SubmissionStatus | PURE value | cmd/internal/couchmessage/protocol.go | modified optional progress and slot display label |
| DOM operation transitions | PURE | nvim/dom_submission_state.lua | modified active-send admission, progress and terminal outcomes |
| Preparation worker | INTEGRATION | cmd/internal/couchtty/console_submission.go | modified progress snapshot protected by existing mutex |
| Evidence preparation/capture | INTEGRATION | cmd/internal/webcast/cast_submission.go, chrome_submission.go, recorder/submission.go | modified explicit progress callback chain |
| Submission RPC handler | INTEGRATION | cmd/internal/couchcmd/submission.go | modified authoritative caller slot display label |
| DOM effect adapter | INTEGRATION | nvim/dom_submission.lua | modified deadline, error cleanup and progress rendering effects |
| DOM transport | INTEGRATION | nvim/dom_transport.lua | modified operation budget for pin/validate/RPC |
| Fresh sender | INTEGRATION | nvim/submission.lua | modified optional DOM operation budget and error result |
| Draft view integration | INTEGRATION | nvim/init.lua | modified read-only buffer owner and priority statusline rendering |
| Zellij action runner | INTEGRATION | nvim/zellij_trace.lua | modified optional timeout preserving redacted tracing |

Existing injected recorder CDP fixture, submission manager fake cast, and headless Zellij/session-log fakes model the dependencies. Their seams remain shared with production. New progress is one volatile snapshot per existing job, not a stream or durable event family.

## Design and choices

Use polling already in place instead of adding an event channel: updates arrive with the next 100ms status poll, and initial local progress renders immediately. A static spinner alone cannot show which crop is in progress. A new push channel would duplicate existing authority and complicate teardown (ARCH-DRY).

Pass an explicit optional callback through PrepareSubmission/CaptureCrops. Report replay, reading/context, settling crop i/N, capturing crop i/N, and publishing. The manager replaces the snapshot under its mutex and guards updates by the current job identity. The callback never holds that mutex while invoking browser or disk IO. Progress does not change stored manifests, delivery transitions, supersession or acknowledgment semantics.

The authenticated broker caller supplies the slot label (caller.Slot), not opaque thread tags or guessed environment values. Older servers without progress get a generic preparing message. Treat malformed progress as unavailable; restrict phase names and count ranges before formatting. Escape statusline percent characters and controls in external text.

The operation uses an absolute 36-second client deadline (the existing nominal 360 × 100ms polling envelope), retaining the server's existing 30-second preparation context and 4-second RPC limit. Each DOM subprocess receives min(remaining budget, applicable existing limit); exhausted budgets refuse before launch. The deadline is not renewed by progress or another chord press. Existing ordinary submission behavior remains unchanged.

Start owns the initiating buffer and its previous modifiable setting, renders preparing synchronously, and arms its deadline before pinning. Set nomodifiable only for that buffer. Completion/error restores that exact buffer before invoking today's clear callback. Invalid/deleted buffers are harmless cleanup targets. Final sent/failure status remains visible until the next submission or draft edit; operation status is checked at the start of PairStatusline, before minimized/review/normal-mode branches, so it takes priority over transient notices/review hints. Do not disable unrelated buffers.

Use optional operation context arguments through fresh append/commit/send and transport to propagate remaining time. Bounded vim.system waits replace unbounded vim.fn.system only where this DOM budget is supplied. The trace wrapper preserves argument redaction. Force a redraw before synchronous pin/send work. Before each send phase check the remaining budget, including the existing 100ms paste-settle delay. A timeout never claims that a partially written/submitted body was not delivered.

## Ordering and invariants

- Idle + send: own new epoch/buffer, lock/render/arm deadline, pin and prepare.
- Active + send: retain current operation; no replacement, deadline renewal or new bundle.
- Preparing + valid progress: update display, preserving operation identity/deadline.
- Prepared + matching current draft/target: display sending to slot, then begin/send through existing guards.
- Any active phase + failure/exception/deadline: retire the operation, release buffer, preserve text and display reason. In-flight callbacks are fenced by epoch; subprocesses are deadline-bounded.
- Confirmed send: release buffer, display sent, perform existing clear only if the original context is current. Retain independent acknowledgment receipts exactly as #524; receipt recovery never owns an editing lock.
- No notes: use the existing fresh ordinary fallback under the same budget and lock; empty text releases with a clear no-content result.
- Late/duplicate callbacks: cannot deliver, mutate progress, clear, or relock a newer draft. A genuine confirmed-send receipt retains its original ID for note resolution, never becomes evidence for a successor.

ARCH-ORDER: reducer transitions decide ownership; adapters execute effects and return events. Timeout/poll/send failure are terminal for UI ownership, not proof of external non-delivery. Process death removes volatile locks with Neovim; no persistent migration is introduced.

## Operating envelope and architecture

- ARCH-CONSTRAINTS: one active DOM operation per draft, one existing server preparation worker, 36s absolute client budget/30s server context, 4s RPC cap. Current note/bundle limits remain. Progress stores a constant-size snapshot, no per-poll history.
- ARCH-SECURE: slot/progress/error values cross process boundaries; validate and render literally. No authored text, page URL or credentials enter new telemetry. Measurements use isolated fixture data, never live operator panes.
- ARCH-PURE: state/outcome formatting resides in value-only model helpers; clocks, buffer options and subprocesses stay injected in adapters.
- ARCH-MOCK: controlled CDP, manager and pane fixtures model held completions, partial send, timeouts and late replies. Existing real-Chrome conformance checks browser behavior; a headless Neovim fixture verifies actual modifiable/statusline behavior.
- ARCH-PURPOSE: cover the entire chord, including pin, no-notes fallback, append, target validation, paste and commit—not only preparation polling.
- ARCH-FUNERAL: status and buffer ownership die on completion/timeout/process exit; stop/close owned timers. No new durable artifacts or telemetry files in product runtime.

## Tasks and verification

### 1. Measure and finalize

- [x] Run isolated instrumented existing fixtures for settlement, screenshot, bundle and pane-send phases. Record samples/commands, separate fixture delays from real external latency, and identify dominant costs without claiming operator-page timings.
- [x] Publish filled issue details and checkpoint this plan. Send concrete design and measurements to hub for required design approval before implementation.

### 2. Progress through preparation

- [x] Add failing tests at submissionManager.request/prepare and Recorder.CaptureCrops using controlled barriers to inspect progress while work is held. Verify job identity and no callback-under-lock deadlock. Run focused tests to establish RED.
- [x] Add typed progress and explicit callbacks across current interfaces, update all implementers/test doubles, and annotate actual phase boundaries. Carry caller.Slot in RPC responses. Keep durable manifest state untouched.
- [x] Run affected Go tests/race checks; commit with issue reference.

### 3. Bounded draft operation UI

- [x] Extend pure reducer tests over active admission, terminal paths, stale epochs and progress validation. Extend adapter fixtures with controllable clock/defer/completions, errors thrown by effects, and budget exhaustion. Establish RED before implementation.
- [x] Implement shared terminal handling, deadline/budget propagation, and buffer ownership/rendering. Return specific failure reason through the fresh sender; preserve confirmed dispatch despite later UI/marker warnings.
- [x] Extend headless production chord tests to hold preparation, inspect actual statusline and nomodifiable, release into failure/success, exercise timed-out subprocess and late callback interleavings, and demonstrate editable retry with new text. Use fixture state assertions rather than production-call counts alone.
- [x] Run Lua state/adapter/transport/sender tests and full headless submission transaction script; commit.

### 4. Integration and close

- [x] Update atlas/webcast-submission.md and operator-facing key documentation for operation-only read-only behavior.
- [x] Run make runtimebundle-generate, affected Go packages with GOMAXPROCS=2 and -p=1, applicable headless suites, real-Chrome crop readiness conformance, and full make test against stable committed sources. Record any unrelated failure precisely.
- [ ] Run sdlc close with actual verification evidence, address findings, publish PR, wait for CI, land on SHIP as authorized, report hub.

Test strategy: production reducer sequence tests establish ownership independently of IO; adapter tests hold and reorder real effect results; headless tests inspect actual buffer options and evaluated statusline before releasing work. Backend progress tests inspect observable status while a fake CDP/cast blocks, then ensure terminal delivery states remain unchanged. A deliberately blocked child proves timeout rather than a mock merely returning a timeout string. No prose duplication of executable case inventories.

## Revisions

### 2026-10-13 — approved design and display details

Ops:0 approved plan 84e3ca051 and the absolute36-second budget (verified receipt e5054a62-e77f-434b-a431-597d7c6d9c3d). On active repeat chord retain operation identity/deadline and display “already sending…” rather than remaining silent. Render settling as “waiting for the page to settle, crop i/N”. Keep #524 settlement unchanged. This supersedes only the initial wording of active-send display.

### 2026-10-13 — plan-review advisory PQ-1

Name the Lua risk targets explicitly: `dom_submission_state.step` uses deterministic reordered event sequences and asserts ownership/terminal invariants; `progress_text` rejects arbitrary malformed progress values and formats only recognized bounded fields. `dom_submission.new`/`send` use a controlled clock and delayed/throwing dependencies to prove release and epoch fencing. `dom_transport.new`'s rpc/pin/validate receive an expiring budget and stateful identity changes to prove no post-deadline external effect. `submit_fresh_operator_text` preserves confirmed dispatch under post-send errors and forwards the same budget to append/commit/send. `zellij_trace.action` runs a blocked child to prove enforced timeout with trace redaction. `PairSendDOM`/`PairStatusline` are exercised in real headless/embedded Neovim to establish buffer option ownership and visible status before held work completes.

### 2026-10-13 — verification scheduling

The corrected full suite is launched and queued behind machine-wide holders. Close review may run concurrently against committed sources; its verification string must say the full result is pending. Inspect and record that result before landing. No source edits occur during either run unless a concrete finding requires them.

### 2026-10-13 — BR-1: operation ownership includes draft actions

The close review reproduced partial navigation under nomodifiable: navigation changed slot identity before its buffer write failed, and a later autosave overwrote queued text. ARCH-ORDER/ARCH-PURPOSE require guarding the whole owned-draft action before any state/file mutation. While an operation owns its buffer, block navigation, queue insertion, deletion, reset, and alternate send/clear actions with “already sending…”. Keep the existing repeat-DOM behavior and absolute deadline. Release restores these actions on success, failure, and timeout. Enumerate production handlers and add headless regressions asserting selected slot identity and persisted draft/queue contents before and after failure/timeout, plus action recovery after release. This implements the approved operation-only ownership policy without introducing a persistent freeze or changing capture settlement.

### 2026-10-13 — BR-1 round2: asynchronous focus mutators

Round2 confirmed the navigation fix with mutation testing but reproduced FocusLost padding attempting to write under the lock, opening a hit-enter prompt. The action inventory was incomplete. Enumerate every direct buffer write and indirect draft mutator in init.lua; include scheduled focus pad creation/removal and timer callbacks, preserving cleanup ownership when deferred. Add real embedded insert-mode/EOL focus-loss/gain sequences through preparation/failure/timeout and ensure no blocking prompt or text corruption. Existing guarded synchronous navigation/send/queue behavior remains. Stop queued suite3 before this correction and restart only on committed stable sources.

Write-site inventory for BR-1: focus pad insertion (scheduled focus/tick) and removal (focus gain/unlock) require ownership-safe deferral with retained {buffer,row}; set_buffer_text is reached by guarded navigation/ship/dirty-history/delete; image placeholder is synchronous under attach_image and replacement is asynchronous under poll_capture before evidence deletion; quote block/inline writes are under PairPasteQuote; finish, dirty-history send/reset, forget-all and queue writes are guarded at entry; pending scrollback is guarded before reading/deleting files; Doctor's completion probe writes only its private scratch buffer, while its late draft clear uses buffer-scoped ownership. Indirect writes: slug apply defers with slug_pending, completion debounce refuses owned buffers, scheduled spelling completion rechecks captured buffer and ownership. Native edit/autopair/completion acceptance has no prior file/slot effects and is rejected atomically by nomodifiable. Cursor/extmark/highlight-only calls do not change authored text. Persistent autosave, navigation, prompt/send, queue/delete/reset and direct-submit/finish paths retain the previous entry guards.

### 2026-10-13 — inherited #531 artifact-inventory integration failure

After SHIP, the full suite reached Go and found seven #531 production files absent from the exhaustive source inventory. Ops:0 authorized fixing this in #533 (receipt ccd0c91d-912e-4875-977b-9caff9ee08d7). Inspect each file; classify launch diagnostics model/capture/show/store/fake, output capture and command renderer as NonArtifactSources because none constructs or consumes the checked Pair artifact filename families. The store owns namespace-local launch-diagnostics/failures.json, outside those families. Do not add known-violation exceptions. Existing exhaustive classification test is the RED regression; rerun artifactpath after adding explicit entries and re-close the changed source. Allow the in-progress full suite to finish so other failures remain visible. If #514 timing fails in a future suite, use make -k to run remaining targets rather than treating that first stop as full-suite evidence.
