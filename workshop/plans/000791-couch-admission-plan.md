---
collaboration-mode: discussion
---

# Couch idle admission and stalled delivery implementation plan

> **For agentic workers:** Follow AGENTS.md §3. The SDLC gates own boundary review. TL approval of this plan is required before production code.

**Goal:** Recognize demonstrated idle Claude/Codex screens safely, name the actual refusal condition, and automatically preserve evidence when an intermittent delivery stalls.

**Architecture:** Preserve the separate peer-delivery and restart admission policies. Share typed explanatory evidence with their existing owners; explanations never grant input or restart authority. Use the wrapper's existing pending-delivery clock and bounded diagnostic writer to capture a refused observation, then repair only a reproduced recognition defect.

**Tech stack:** Go, existing terminalModel/Ultraviolet cells, couchmessage private endpoints, diagnosticlog.BufferedWriter, captured native ANSI fixtures, shared testcouch harness.

**Active scope:** The final 2026-10-22 revision below supersedes Tasks 1–4 and their restart/root-cause acceptance. The original proposal is preserved as design history. Implement the diagnostic-only revised tasks.

## Evidence and limits

The operator reports recurring hub stalls before and after #735, perhaps twice daily. A session-file submit sometimes produces no immediate agent-bar message, or the hub unexpectedly goes quiet; a keystroke then flushes deliveries. Workers may also stall but that is unconfirmed. The update banner stays visible for hours while most delivery succeeds, so it cannot alone explain the symptom.

On 2026-10-22, scratch-only overlay replay of retained native bytes plus recorded resize offsets through current main produced PeerComposerEmpty for all three observations:

| Capture | Bytes replayed | Resize events | Result |
|---|---:|---:|---|
| ops:0, 1-ops-71, Claude bypass-permissions + update banner | 54,648,427 | 92 | complete, visible cursor (2,33), empty |
| echo:3's earlier 1-echo-87, Claude auto-mode + update banner, pre-exit cutoff | 80,031 | 5 | complete, visible cursor (2,34), empty |
| parley.nvim:3, 1-parley-nvim-83, Codex blank line + one-warning footer | 9,662,972 | 48 | complete, visible cursor (2,35), empty |

Run: `go test -overlay /Volumes/TestScratch/pair791-replay.v_daojgb/overlay.json ./cmd/internal/wrapcmd -run '^TestInvestigation791CaptureReplay$' -count=1 -v` — PASS, 4.596s package time. This proves those retained final screens are recognized by current main, not that the earlier stalled state was reproduced. Raw streams stay private; committed fixtures contain sanitized native composer/chrome bytes with provenance and truncation notes.

Claude's ruled-box recognizer does not inspect footer wording. Restart admission currently sees only Settled=false and prints a combined cause; peer admission does not require Settled or an inactive turn. Cursor visibility, incomplete synchronized frames, real resize/repaint, automatic-input state, incomplete input, picker and viewport observations remain possible triggers. Do not treat hypotheses as reproduced causes.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| SettleEvidence | cmd/internal/wrapcmd/peer_settle.go | new |
| SettleObservation | cmd/internal/couchmessage/settle_endpoint.go | new |
| PeerWaitCaptureState / advancePeerWaitCapture | cmd/internal/wrapcmd/peer_wait_capture.go | new |
| PeerWaitEvidence | cmd/internal/wrapcmd/peer_wait_capture.go | new |
| LiveRestartFacts / DecideLiveRestart | cmd/internal/couchcore/live_restart.go | modified |

SettleEvidence names the first actual unsettled condition with a fixed enum: turn, picker, unknown composer (including hidden cursor/incomplete frame/geometry), occupied composer, pending input/output/image, or quiet interval not elapsed. It derives the existing boolean rather than creating a second admission policy (ARCH-DRY/PURE). Observations carry binding/sequence and a reason without draft contents. No observation may overwrite a newer invalidation.

PeerWaitCaptureState owns only diagnostic eligibility: delivery ID, first-wait time and captured flag. PeerWaitEvidence holds the condition actually used by dispatchPeer, elapsed wait, receipt phase, pending-input/output counts, automatic-input state, turn/picker, paste mode, viewport result, render/terminal generations and the decision snapshot. Existing composer recognizers remain the source of composer authority; any later recognition change requires a failing native fixture.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| settle-observe endpoint | cmd/internal/couchmessage/settle_endpoint.go; endpoint.go; cmd/internal/wrapcmd/peer_runtime.go | new | read-only exact-bound wrapper observation |
| settleState / settleFired | cmd/internal/wrapcmd/peer_settle.go | modified | terminal, atomics, pending-I/O and quiet-generation fence |
| liveRestartProbe | cmd/internal/couchcmd/live_restart_probe.go | modified | existing connected-wrapper endpoint and restart effect-time recheck |
| dispatchPeer diagnostic emission | cmd/internal/wrapcmd/peer_delivery.go; peer_runtime.go; diagnostic_writers.go | modified | one refused observation and existing managed wrap-events writer |
| terminalModel | cmd/internal/wrapcmd/terminal_model.go | existing, fix only if replay proves defect | native bytes, resize and coherent cell snapshot |

Use a new read-only `settle-observe` request with a new optional response member, populated only for that operation. Existing operations retain their response shape; no session-frame dialect change. New clients receiving an old wrapper's unsupported operation report reason unavailable and keep refusal. New wrapper/old client compatibility is pinned by tests. A short bounded diagnostic query on restart refusal cannot admit or signal anything; match binding/session before attributing it, label disagreement with the cached settled claim explicitly. Apply the same explanation at effect-time refusal. No screen/draft goes over this explanation endpoint.

Tests reuse harnessSessionFake's terminal and real peer reducer, manual d.now/settle clock, stateful endpoint server, existing session-negotiation fixtures, and temp-root diagnosticlog. Native delivery acceptance uses shared cmd/internal/testcouch only, for Claude and Codex; Qoder parser/admission contracts remain covered without requiring credits. No live-fleet test sends or key injection.

## Operating envelope and ordering

Proposed threshold: 15 seconds of continuous pre-paste wait for the active head delivery, checked on the existing one-second queued poll. Capture once per delivery; a successful paste or terminal/cancel/release ends it. A new delivery starts fresh. Batch preparation uses the same head receipt, so Claude hub batches cannot bypass capture. Queue position >1 has no refused wrapper screen of its own: capture the active head with its batch IDs, not invented downstream refusals.

| Diagnostic state / event | Next state / effect |
|---|---|
| no active delivery / enqueue or prepare-batch | waiting, start timestamp |
| waiting / refused poll before threshold | waiting, no serialization or IO |
| waiting / refused poll at threshold | captured, copy that refused observation once |
| captured / further refusal, reason change, resize | captured, no repeated record |
| waiting or captured / paste begins, terminal receipt, release, replacement | reset; no input or receipt effects from diagnostics |
| capture sink failure / any state | preserve admission/receipt; report diagnostic failure, no retry storm |

Capture the exact snapshot used by composer classification, not a later peek. For early guards that do not read the screen, capture a coherent terminal snapshot with the already selected reason and label its generation. Do not claim a rendering generation proved an earlier pending-I/O condition. Include native cursor visibility, dimensions, completion/sync/resize uncertainty and style-preserving cell runs; rendered text alone cannot replay this class.

Cap the complete encoded record at diagnosticlog.BatchMaxRecordBytes (64 KiB), including envelope. Record a full screen when it fits; otherwise retain composer/footer region and explicit omitted/truncated metadata. The encoder stops at the budget; oversized text/cells cannot produce unbounded output. Do not log message bodies or argv. Local screen content is potentially private and stays in the local wrap-events diagnostic file, never automatic public uploads. Explain this in atlas.

Only the input owner chooses the capture and marks it emitted. Copy under its existing fences, then release admission/mailbox locks before encoding/enqueueing to the pre-opened BufferedWriter. Open/close that existing writer with the proxy, never lazily on a refused input path. Writer IO is asynchronous and joined by its current close owner; no extra polling process/goroutine or file family (ARCH-ORDER/FUNERAL). Reuse existing diagnostic generation caps, seven-day retention and GC; no unbounded per-delivery files. Disabled/unavailable diagnostics are visible and cannot change whether delivery is safe.

Per queued poll, before threshold: O(1) new bookkeeping; after threshold: at most one O(C) snapshot/encode bounded by terminal cells C and record bytes. No history, log or capture rescans per poll. Count 8/16 deliveries and 8/16 repeated polls: capture count <= deliveries, additional repeated-poll captures zero; cell visits linear in visited cells and serialized bytes <=64 KiB. No filesystem IO under input/mailbox locks. Network overhead applies only to explicit restart refusal, bounded to one diagnostic observation with a proposed 250ms child context; timeout leaves the original refusal.

## Visible state and recovery

Actions: wait, inspect receipt/capture, finish/clear one's own input, close picker, relaunch when safely idle, report capture to TL.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| known turn/draft/picker/recent input | busy: actual named condition | finish/clear own input or close picker | none; no automatic overwrite |
| incomplete/hidden/unrecognized frame | waiting: exact missing evidence | inspect receipt/capture | delivery stays queued; no replay after uncertain paste |
| past threshold | same wait + local diagnostic location | inspect capture/report to TL | a bounded screen may be truncated, explicitly marked |
| older wrapper / observation timeout | busy, reason unavailable | relaunch when safely idle | explanation unavailable, no weakened guard |
| optional sink disabled/full/fails | same wait, diagnostics unavailable | inspect receipt/report failure | one capture may be lost; no user input lost |
| uncertain paste/submit | existing indeterminate state | inspect before any resend | existing possible submission uncertainty, no diagnostic retry |

Coverage includes all enumerated peer guards and restart admission/effect-time refusal. Not covered: claiming every historical stall was the same fault, unconditional idle admission by banner, manufacturing terminal cursor visibility, automatic retry/resubmit, or fixing unrelated echo reboot ownership errors.

## Tasks

### 1. Preserve captured controls and isolate a failing transition

Files: cmd/internal/wrapcmd/testdata/peer/{claude,codex}/ (sanitized #791 captures with metadata), peer_composer_test.go, peer_resize_test.go, new peer_admission_capture_test.go.

- [ ] Extract native composer/chrome bytes from the retained captures, record source offsets/dimensions/version and source digest. Keep original raw streams private. Replay with native resize events, include exact observed footer text and Codex style/faint placeholder.
- [ ] Add passing controls for the three captured idle screens and banner-free variants. Occupied composer, slash/shell draft, picker, queued-message line and active-turn cases assert their existing intended policies.
- [ ] Exercise split synchronized frames, hidden/show cursor, changed/same-size/aborted resize, pending replies, end-of-turn and queued-line redraw sequences using existing controlled seams. Do not call a synthetic fault a reproduction of the reported live stall.
- [ ] Run `go test ./cmd/internal/wrapcmd -run 'Test(Peer|Terminal.*Resize|.*791)' -count=1`; save failures by exact condition and test. If no failing live transition exists yet, retain controls and proceed to diagnostic capture, leaving root-cause acceptance pending.

### 2. Explain restart and peer refusals from their real owners

Files: peer_settle.go and tests; peer_composer.go and tests; couchmessage/settle_endpoint.go and tests; endpoint.go; peer_runtime.go; couchcore/live_restart.go and tests; couchcmd/live_restart_probe.go and tests.

- [ ] Write failing tests for each distinct settled refusal, quiet invalidation generation, legacy wrapper unsupported response, malformed observation, timeout, disconnect/reconnect and effect-time change. Pin that neither explanations nor failed diagnostics grant admission.
- [ ] Implement SettleEvidence and narrow read-only settle-observe. Keep diagnostic-only response optional/operation-specific; transport still validates identity and known field types strictly.
- [ ] Feed the explanation into both initial restart refusal and effect-time refusal. Existing peer details derive from the decision and include cursor/frame/geometry causes where known. Keep active-turn peer delivery permitted when its composer/input guards permit it.
- [ ] Run `go test -race ./cmd/internal/couchmessage ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/wrapcmd -run 'Test(.*Settle|.*LiveRestart|.*Restart|.*Peer.*Wait|.*Endpoint|.*Negotiat|.*791)' -count=1` and commit.

### 3. Capture a delayed refused delivery without changing its outcome

Files: new wrapcmd/peer_wait_capture.go and peer_wait_capture_test.go; peer_delivery.go; peer_batch.go if its preparation bypasses the normal decision; peer_runtime.go; diagnostic_writers.go; peer_integration_test.go; diagnostic_turn_budget_test.go.

- [ ] Pure transition tests cover the table above, exact threshold boundary, backwards clock, new IDs/batches, cancellation/replacement, and exactly once despite repeated polls/reason changes.
- [ ] Integration tests pin condition and refused snapshot generation together, preserve cursor/style/frame metadata, replay encoded evidence, exercise every early guard and Claude batch preparation, and prove zero PTY writes while refused.
- [ ] Inject slow/full/failing diagnostic writers and show admission locks are released before optional emission, delivery state unchanged, no retry amplification, bounded serialization and existing shutdown join.
- [ ] Implement the transition and wire it to the existing pending-delivery tick and pre-opened managed wrap-events writer. Before the threshold, take no extra snapshot/encode. Use one bounded record labelled PEER-wait with slot/session/message identifiers, not message text.
- [ ] Run counted N/2N tests and `go test -race ./cmd/internal/wrapcmd ./cmd/internal/diagnosticlog -run 'Test(.*Peer.*Capture|.*791|.*Diagnostic|.*Buffered|PeerPolling)' -count=1`; commit.

### 4. Repair the demonstrated recognition defect and qualify the integrated behavior

Files: only the recognition/terminal/input source proved responsible by replay; corresponding colocated tests; atlas/couch.md and atlas/couch-testing.md; artifactpath/manifest.go and reader inventories if new sources require registration.

- [ ] Obtain a failing captured transition from retained streams or the threshold diagnostics; write the regression first and demonstrate the exact guard that refuses a genuinely idle composer. If it needs live diagnostic rollout first, send the evidence gap and stage request to TL; do not call the issue fixed or weaken unknown-state refusal.
- [ ] Fix the smallest responsible recognition/state transition, preserving invalidation ordering, complete-frame rules, operator draft ownership and uncertain-input handling. If the capture requires a different mechanism than this plan, bring the revised design to TL before code.
- [ ] Run fixture conformance, duplicate-resize #735 controls, complete affected Go packages, repolint/artifactpath/reader-inventory preflights. Document capture label/path, privacy, threshold, cap/retention, missing diagnostics and safe recovery in the existing atlas guides (existing index links remain sufficient).
- [ ] Run shared disposable-Couch acceptance for Claude and Codex: idle banner/warning frames, deliberate draft/picker hold, threshold capture, then native repaint recovery and exactly one submitted delivery. No test sends to real workers/hub.
- [ ] Push final head; `sdlc local-check run --name affected` (fixture edits may select full suite). Only after its pass and root-cause regression, use `sdlc close --issue 791 --verified '<exact evidence>'`; SDLC owns fresh-context review. No invented hours.
- [ ] Request TL-coordinated `sdlc stage --build`/operator restart only when authorized. The intermittent real-world capture/live check remains an acceptance obligation; report its exact evidence or uncertainty before landing through the normal line.

## Architectural accounting

ARCH-DRY/PURE: owner-derived evidence and pure diagnostic eligibility, no alternate recognition rules. ARCH-PURPOSE: both personalities, both refusal paths and intermittent-capture acceptance included. ARCH-MOCK: stateful existing terminal/endpoint/disposable-fleet seams, not call-count mocks. ARCH-CONSTRAINTS: one-second existing pending poll, 15s threshold, 64KiB record, counted growth, bounded reason query. ARCH-SECURE: authenticated exact binding, screen privacy, bounded encoding and no public raw transcripts. ARCH-ORDER: refusal snapshot and condition travel together; diagnostic completion never authorizes input; reconnect/quiet invalidation retain fences. ARCH-FUNERAL: existing bounded diagnostic lifecycle, no new files or workers. ARCH-EVOLVE: new endpoint response only for a new explicit request; existing frames and persisted formats unchanged; unsupported older wrappers visibly refuse without inferred reason.

## Revisions

### 2026-10-22 — TL approved diagnostic-only delivery now

Authenticated approval 4333f27a-2f1d-4e8a-b9dd-039d84a610c0: land capture as #791 now; recognition fix becomes a follow-up waiting for the first real failing capture. No restart endpoint, new settle evidence/session transport, or recognition changes are in this delivery. Existing peer refusal details already name their precise conditions; capture records that exact detail. This supersedes original Tasks 1–4, original Core concepts tables, and original root-cause/live acceptance gate above. The scratch replays remain evidence, not a new fixture family or recognition deliverable.

**Current Core concepts — pure:**

| Name | Lives in | Status |
|---|---|---|
| PeerWaitCaptureState / advancePeerWaitCapture | cmd/internal/wrapcmd/peer_wait_capture.go | new |
| PeerWaitEvidence / encodePeerWaitCapture | cmd/internal/wrapcmd/peer_wait_capture.go | new |

The transition receives active-head ID, pre-paste status, refusal condition and time; it returns at most one capture effect per head after 15s. Its evidence owns an immutable terminal snapshot, flags and identity metadata. Encoder retains styled cells under a 64KiB complete-record limit, explicit truncation if needed. State ends with the delivery; records use existing wrap-events generation retention. No additional durable state or worker.

**Current integration points:**

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| dispatchPeer | cmd/internal/wrapcmd/peer_delivery.go | modified | exact refused snapshot, existing clock and admission/mailbox locks |
| peerRuntime | cmd/internal/wrapcmd/peer_runtime.go | existing | wires the proxy's already-open diagnostic owner |
| wrap-events writer | cmd/internal/wrapcmd/diagnostic_writers.go; wrap.go | existing, reused | diagnosticlog.BufferedWriter and managed retention |

Capture the snapshot used for composer refusal; early non-screen guards get one coherent snapshot at the refusal and identify its generation. Writer enqueue and JSON encoding happen after both locks are released. Reuse the existing queued poll (including Claude batch preparation) and injected d.now. No extra polling clock. Before the threshold bookkeeping is O(1), no extra screen copy. Capture once and suppress repeated refusal polls, failed writes and reason changes; reset for next head. Disabled diagnostics do not copy a screen. Preserve the 15s/64KiB envelope, full state/visible recovery tables and existing diagnostic privacy/retention rules above, except no new restart-query behavior.

**Revised tasks (one close review boundary):**

- [x] File unclaimed follow-up for recognition, blocked on first failing capture and referencing #791. Update #791 Done-when with this approved narrowing.
- [x] Test first: force hidden-cursor refusal through production dispatchPeer. At 14.999s no record; at 15s exactly one PEER-wait record names cursor-hidden and includes the refused screen/cursor/complete/style evidence; repeated polls/reason changes do not duplicate it. Recover by native repaint, assert normal delivery and a fresh capture for a new head.
- [x] Unit-test advancePeerWaitCapture with generated event sequences asserting independent threshold/reset/once-per-head invariants; fuzz encodePeerWaitCapture over styled arbitrary cells, asserting valid JSON, explicit truncation, and complete-record byte cap. Stateful dispatch integration uses the existing terminal/peer seams across supported personalities and batch preparation, preserving input safety and proving both locks free at writer invocation.
- [x] Implement the bounded diagnostic state and encoding, hook capture selection into dispatchPeer without changing its safety decisions. Add counted 8/16 head and repeated-poll cases; screen copy/encode occurs once/head, zero filesystem IO under input locks. Existing BufferedWriter owns flush/close and residue bounds.
- [ ] Run targeted race tests, full wrapcmd and diagnosticlog packages, artifactpath/reader inventory/repolint preflights. Register new production source in the normal manifest, not quarantine. Update atlas/couch.md and couch-testing.md with label/path, privacy, threshold, truncation and unavailable diagnostics.
- [ ] Push final head, run recorded affected check (full suite if selected), close via SDLC mandatory review with exact forced-refusal/no-input/lock/cap evidence, then enter normal landing line. Native GUI staging only on TL authorization if the gates require it; no real-fleet sends or recognition acceptance claimed.

### 2026-10-22 — PQ-1 risky-function contract

Addressed the risky-function naming class in the active tables and task: advancePeerWaitCapture is the pure timing/reset transition; generated event sequences assert independent threshold and once-per-head invariants. encodePeerWaitCapture is the pure screen encoder; fuzz arbitrary styled cells with valid-JSON, explicit-truncation and whole-record-budget oracles. dispatchPeer remains the stateful integration boundary, with forced refusal and writer lock/failure assertions. Superseded historical case lists above are not a second executable test plan. Follow-up allocated as pair#810, unclaimed and waiting for first real capture.

### 2026-10-22 — concrete diagnostic implementation anchors

The active concepts materialized as PeerWaitCaptureState/advancePeerWaitCapture and PeerWaitEvidence/encodePeerWaitCapture in peer_wait_capture.go. The existing terminalModel adds SnapshotGeneration (terminal_model.go), returning the same copied snapshot with its generation under one terminal lock; Snapshot delegates to it without changing recognition. dispatchPeer reuses this snapshot for composer decisions and defers emitPeerWaitCapture until both input/mailbox locks are released. The encoder stores bottom-first per-row ANSI style runs, preserving screen coordinates and explicit truncation under the whole64KiB limit. There is no new peer-runtime worker or endpoint. Source inventory classification is NonArtifactSources. Tests are TestPeerWaitCapture* and FuzzEncodePeerWaitCapture.

### 2026-10-22 — BR-1 fitting-screen preservation

The row encoder now budgets exact escaped JSON bytes incrementally rather than reserving worst-case sixfold expansion for every raw byte. It bounds raw grapheme size before escaping and includes each row envelope/separator and the complete record/newline. TestPeerWaitCaptureFittingStyledRow reproduces the prior omission of a37,009-byte styled row, then passes with the correction. Preserve the existing full-screen-when-it-fits contract; no new mechanism or recognition policy. BR-2 host tracker proof is in the issue Log and dispute ledger.


### 2026-10-22 13:31 — approved resize evidence and visible-pane transition

TL approval d06e1506-63a0-4335-85b4-96087a40510b authorizes this diagnostic addition before landing. The operator observed a stall after tab-bar wrapping, always in the viewed slot. This is a lead, not a reproduced cause. The previous diagnostic-only architecture remains; recognition repair stays in #810 and can begin only after an actual failing transition is demonstrated and reported.

| Name | Lives in | Status |
|---|---|---|
| terminalResizeObservation | cmd/internal/wrapcmd/terminal_model.go | new pure value |
| PeerWaitEvidence | cmd/internal/wrapcmd/peer_wait_capture.go | modified; latest resize observation |

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| terminalModel.PrepareResize / terminalResize.resolve | cmd/internal/wrapcmd/terminal_model.go | modified | existing resize transaction and clock |
| terminalModel.SnapshotDiagnostic | cmd/internal/wrapcmd/terminal_model.go | new | one terminal mutex for copied screen, generation and resize evidence |
| dispatchPeer / selectPeerWaitCaptureLocked | cmd/internal/wrapcmd/peer_delivery.go; peer_wait_capture.go | modified | carry the decision's coherent diagnostic snapshot |

Keep one fixed-size observation: attempt time, completion time, requested cols/rows, changed flag and outcome (prepared/committed/aborted). A valid PrepareResize records the attempt under m.mu; resolution updates completion/outcome under the same mutex. No-op resize records its event without invalidating existing screen evidence. A rejected invalid size/closed call cannot overwrite an admitted resize observation. SnapshotDiagnostic copies the observation, cells and generation under m.mu. SnapshotGeneration and Snapshot retain their existing API and recognition behavior. No new worker, file family, timer or per-event history; existing diagnostic retention owns the additive JSON evidence. Each resize is O(1) metadata work; snapshot remains O(cells) once per capture.

Actions: inspect diagnostic; wait for repaint; request authorized refresh.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| No admitted resize observed | no latest resize | inspect diagnostic | earlier wrapper history unavailable |
| Prepared, native PTY update outstanding | prepared with attempt time; incomplete screen | wait for repaint | no input admitted from incomplete screen |
| Commit after changed geometry, cursor not repainted | committed times/geometry; precise hidden-cursor refusal | wait for repaint | queued delivery waits |
| Abort after failed native resize | aborted with attempt/completion times | request authorized refresh | geometry uncertain; no input authority added |
| Same-size commit | committed, changed=false; existing screen evidence retained | inspect diagnostic | no additional loss |
| Optional diagnostic disabled/fails | existing receipt wait remains | inspect diagnostic | screen/resize evidence unavailable |

Tasks (same single close boundary):

- [ ] Replay visible-pane one-row/two-row/one-row geometry using the real status-bar planner and retained native Claude composer/chrome; exercise in-flight resize, repaint and duplicate SIGWINCH, ordinary and batch admission. Label modeled transitions separately from captured native resize/repaint sequences. Inspect retained native events for an actual failed transition; report any reproduction to TL immediately, before landing.
- [ ] Write a behavioral red asserting the emitted forced-refusal record contains the most recent attempt/completion times and requested geometry/outcome, including no-op and aborted cases. Add same-generation coherence assertions under concurrent resize/feed/snapshot.
- [ ] Implement fixed-size resize observation and coherent SnapshotDiagnostic, preserving SnapshotGeneration's API and all existing admission decisions. Carry the exact decision metadata through dispatch; early guards obtain a coherent current observation. Tests assert no IO under input/mailbox locks and whole-record <=64KiB with the extra fields.
- [ ] Risky-function strategy: PrepareResize/resolve generated sequences cover changed/no-op/abort/retry/invalid/closed and idempotent resolution; independent invariants pin completion ordering and screen invalidation. SnapshotDiagnostic concurrent stress pins screen dimensions/generation and the matching latest transaction; encoder fuzz continues valid-JSON/explicit-truncation/full-record bounds. Count 8/16 repeated resize/capture heads to retain constant metadata and once/head diagnostic work.
- [ ] Run targeted race and complete changed packages plus preflights; update atlas evidence fields. If production code changes, cancel only the exact obsolete queued check through local-check, push and submit the replacement recorded check. Retain previous successful verification as history, not final-head proof. Re-close through SDLC and land only after the recorded pass and report of the resize investigation.
