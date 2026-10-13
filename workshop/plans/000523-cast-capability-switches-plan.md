# Cast capability switches implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make 👆 control annotation availability and 👽 control outside driving, with server enforcement, stable share links, and one left-toggle/right-copy rule.

**Architecture:** Board owns annotation admission; the drive controller owns outside-driving admission. Cast publishes their state to Couch and SSE viewers. Ordered recording operations clear annotations and control without resurrecting old work; the existing terminal pointer retains its own mature safety model.

**Tech Stack:** Go, existing CDP/recorder queue, HTTP/SSE, browser JavaScript, terminal cell renderer, Go and Node tests, headless Chrome conformance.

## Scope and approved behavior

#506 is on main (PR #302). Start this issue from main; never import #494's branch.
Web annotation and drive both start OFF. Left click enables/disables; enabling
copies the stable link, as terminal pointing already does. Right click copies
regardless of state; a missing terminal pointer token/code is made without
switching on. Preserve terminal marker visibility enforcement, rate limits,
geometry checks, code locks, hover/Alt-click and control-click meanings.
There is no terminal 👽 or terminal driving implementation in this issue.

Annotation OFF covers both pointer and drive links and the operator's overlay.
It clears current strokes and notes while preserving earlier replay history.
Drive OFF clears active and pending control, releases held input and refuses
new requests/grants/input. Seats remain so reconnect does not require a new
identity; re-enable requires a new request and operator grant.

Two independent capability bits are intentional: all four combinations are
legal (watch only; annotate; drive without annotations; annotate and drive).
Support/absence of a capability and link creation are separate facts.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| CapabilityState / NextCapability | cmd/internal/webcast/capability/state.go | new |
| AnnotationClear | cmd/internal/webcast/recording/annotations.go | new |
| State.Apply / CatchUp | cmd/internal/webcast/recording/reduce.go, catchup.go | modified |
| NextControl / CtlReset | cmd/internal/webcast/recording/control.go | modified |
| castControlIntent | cmd/internal/couchtty/cast_control.go | new |
| StatusModel capability projection | cmd/internal/couchtty/reserve.go | modified |
| JS recording Model | cmd/internal/webcast/web/model.js | modified |

CapabilityState is one capability's admission generation and lifecycle, reused
by Board and Controller without making either read Couch booleans. Legal phases:
Off, On, Disabling (admission closed, cleanup pending), Ended. The generation
invalidates previously admitted work. Each owner owns exactly one such state;
Cast and viewers consume snapshots. This centralizes disable/retry ordering,
not the unlike terminal/web storage models. Future behavior widens this model
only when a concrete new lifecycle event is needed.

Planned transition API shape:

```go
type Phase uint8 // Off, On, Disabling, Ended
type CapabilityState struct { Phase Phase; Generation uint64 }
type Event uint8 // Enable, Disable, Cleaned, End
func NextCapability(s CapabilityState, e Event) (CapabilityState, error)
```

AnnotationClear is an additive recording operation (`annotation-clear`) that
removes current annotation state at its ordered position, including during a
pause. No fake navigation or UI-only hiding. Existing recordings need no
migration; old readers already preserve/ignore unknown kinds but cannot
reproduce the new clear behavior. The current Go/JS readers, canonical state,
replay and catch-up must agree. Earlier moments in a recording remain intact.

CtlReset is one operator-authored control transition that produces no driver
and no pending requests. It avoids separately observable partial deny/revoke
sequences. Existing request/grant/release records and old recordings keep their
meaning. Record validation names the operator explicitly for reset.

castControlIntent maps right-click to Copy, left+on to Disable and left+off to
EnableAndCopy. Unsupported controls remain handled by their existing notices.
Use it for terminal 👆 and web 👆/👽; code menus remain copy-only.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Board annotation admission | cmd/internal/webcast/board/board.go, annotation.go | modified/new | request/emit ordering and recording sink |
| Recorder acknowledged writes | cmd/internal/webcast/recorder/recorder.go, capability.go, control.go, drive.go | modified/new | ordered stream writes, control commitment and clear acknowledgment |
| MemController capability | cmd/internal/webcast/drive/controller_mem.go, controller.go | modified | control sink and input injection |
| Cast capabilities | cmd/internal/webcast/cast_capabilities.go, cast.go | new/modified | Board/controller state and lifecycle |
| Capability HTTP/SSE | cmd/internal/webcast/server_board.go, server_drive.go, server.go, stream.go | modified | untrusted HTTP requests and live viewer state |
| Browser capability consumers | cmd/internal/webcast/web/viewer.js, annotate.js, drive.js | modified | SSE, delayed module mount, gestures and drafts |
| EnsurePointerLink | cmd/internal/broadcast/pointer.go, session.go | new | token/code minting without enabling |
| Couch click dispatch | cmd/internal/couchtty/console_pointer.go, console_webcast.go, terminal_input.go, terminal.go | modified | clicks, copy and state rendering |
| Operator hook adapter (landing-order integration) | cmd/internal/webcast/chrome_operator.go, board/operator.go, recorder/operator.go | modified after #494 | exact isolated-world CDP context and operator mutation admission |

Reuse Board.Writer, FakeBrowser/FakeDriven, drive.FakeInjector, the real HTTP
fixtures and Chrome scripted peer. These retain state across requests; do not
replace them with call-count-only mocks. Extend the Chrome peer to track applied
permission per context and allow delayed/reordered responses. Live conformance
is headless, with temporary profiles and local pages, never the operator's real
Chrome profile.

## Ordering, errors and ownership

| State/event | Transition and effects |
| --- | --- |
| Start supported cast | Both capabilities Off before any route or overlay admits work |
| Off + enable | On; generation advances; publish enabled; copy stable link after successful enable |
| On + disable | Disabling immediately closes admission and invalidates old generation; enqueue clear/reset |
| Disabling + cleanup acknowledged | Off; publish completed state |
| Cleanup failure | Remain Disabling/off for admission; surface error; retry same cleanup, never silently re-enable |
| Off + duplicate disable | No new clear or secret; idempotent |
| Disabling + enable | Refuse until cleanup is confirmed, preventing resurrection |
| Any + end | Ended; invalidate permits, cancel/join owned work, close subscriptions |
| Right copy | State unchanged; ensure missing token/code atomically, then copy |
| HTTP admitted before toggle, body delayed | Captured generation must still match at Board/controller admission |
| Annotation validated before toggle, emission delayed | Serialized mutation enqueue must precede clear, or stale permit refuses it |
| Driver batch admitted before disable | Recorder dispatch rechecks capability generation/control; no stale input after reset |
| Document/context replaced | Ignore old context completions; apply current capability to new exact context before interaction |

Keep Board.mu a leaf: never emit or wait for recorder work under it. Add a
separate command serialization boundary for annotation validation/enqueue and
clear enqueue. Board.Put follows stream state without acquiring that command
lock. Capture the generation at HTTP admission and recheck after body read;
operator preparation/ordered sink uses the same admission generation. The
recorder must acknowledge ordered cleanup, not merely acknowledge enqueue.
Before the first live document there are no marks to clear; close admission
without requiring an unopened recorder. Paused cleanup is an explicit operator
control operation and must not be dropped by the ordinary live-document guard.

Drive disable closes controller admission before reset, under the controller's
existing ownership lock. Queue one CtlReset and confirm its ordered outcome;
new requests/grants are refused while Disabling. A reset releases held input
through the existing Releaser. A queued input must recheck the permit at actual
dispatch, including after a quick off/on/new grant to the same seat. Do not
hold controller locks across CDP waits. Preserve error evidence and remain
closed if the recorder or CDP fails; bounded cleanup follows existing cast
cancellation and release timeouts.

Viewer SSE snapshots distinguish link permission from cast permission. Effective
annotation is link.annotates AND annotation On; effective driving is link.drives
AND drive On. A latest-value snapshot is available synchronously to consumers
and replayed when they subscribe, so a toggle during dynamic import is not lost.
Incoming older generations never override a newer state. Off cancels gestures
without flushing a final stroke, holds the unsent draft, sets watch mode, clears
queued drive input and shows "Annotation is off" / "Driving is off". Turning
on never resumes a prior gesture or automatic drive request.

## #494 coordination contract

Peer-confirmed seam: operator.js installs __couchcastAnnotationAllowed(bool) in
the `couchcast` isolated world; false cancels/holds/hides draft, true leaves pen
Off. #494 currently defaults true, so integration must make startup unavailable
until current permission is applied. Use chrome_operator.go startOperator and
refreshOperator to target the exact session/contextId from loopState.contexts.
Serialize/version permission application so a late ON cannot win after OFF;
apply after every replacement-document hook installation, not just once at cast
startup. JS is availability only: gate board/operator.go annotationPageLocked
and recorder/operator.go operatorAdmission at preparation and ordered write.

Whichever issue lands second implements and verifies these connections. Before
closing this issue, inspect #494's landing state. If it is on main, merge main
and wire here; otherwise record the API/seam and tests in the handoff to pair:1,
which explicitly accepted second-landing ownership. Do not claim end-to-end
overlay acceptance until that integration has actually run.

## Architectural checks and operating envelope

- ARCH-DRY: one click-intent function; one capability transition model; existing
  Board/controller own permission, while all presentation derives from them.
- ARCH-PURE: transition, recording reset/clear and click intent are pure unit
  surfaces. IO and mutex ownership stay in the named integration points.
- ARCH-PURPOSE: cover drive-link annotation, operator paths, body-read races,
  queued writes/input, reconnect and replacement documents, not only controls.
- ARCH-MOCK: extend the existing stateful writer/controller/CDP doubles and run
  headless conformance for changed CDP behavior.
- ARCH-CONSTRAINTS: click/admission work is O(1), without network/CDP on Couch's
  input loop. Existing request size/rate/in-flight budgets remain. Queue cleanup
  work on cast-owned workers, bounded by existing 5s CDP / 10s release limits;
  overload or timeout remains visibly disabled. Latest-value subscribers hold
  one snapshot, not an unbounded queue. No new periodic polling.
- ARCH-SECURE: HTTP identity, body and generations are untrusted; parse typed
  fields and retain existing code/seat authentication. Capability only narrows
  link roles, never upgrades a view-only link. Tokens/codes are minted solely
  through explicit enable/copy actions or existing cast startup; never logged
  or shown in broadcast notices. CDP calls use structured arguments and exact
  context identity, with boolean permission only.
- ARCH-ORDER: table above covers delayed body/emit/input, reset failure, rapid
  toggle, replacement context, disconnect and cast end. Tests control these
  interleavings with barriers and stateful queues rather than sleeps.
- ARCH-FUNERAL: no new sidecar or permanent cache. Switch state and subscriptions
  die with the cast; workers are cancelled/joined. New clear/reset records add
  O(1) bytes per toggle to the existing operator-owned recording lifecycle.

## M1 — Authority, recording and server behavior

### Task 1: Pure state and ordered recording operations

Files: new capability/state.go and state_test.go; recording/annotations.go,
recording/control.go, record.go registry, reduce.go, codec/validation/canonical,
catchup.go and colocated tests; web/model.js and tests/webcast/model.test.mjs.

- [x] Write failing tests using the Task 1 function strategies below.
- [x] Run `go test ./cmd/internal/webcast/capability ./cmd/internal/webcast/recording -count=1`
  and `node --test tests/webcast/model.test.mjs`; confirm expected red.
- [x] Implement the pure types/transitions and register additive format kinds;
  include clear in catch-up context and enforce reset operator attribution.
- [x] Run the same tests green; commit `#523 M1: model capability transitions and ordered clears`.

### Task 2: Admission and bounded cleanup

Files: board/annotation.go, board/board.go, board/writer.go; recorder/recorder.go;
drive/controller.go, controller_mem.go, controller_contract.go, tab.go;
cast_capabilities.go, cast.go; corresponding tests and existing fake seams.

- [x] Write failing tests using the Task 2 function strategies below.
- [x] Implement generation-bound admission and cleanup acknowledgments as above;
  preserve Board leaf-lock invariant and recorder's single writer.
- [x] Wire Cast defaults OFF before exposing routes, supported capability snapshots,
  latest-value subscriptions and cancellation/close. Fakes use the same transitions.
- [x] Run `go test ./cmd/internal/webcast/board ./cmd/internal/webcast/drive ./cmd/internal/webcast/recorder ./cmd/internal/webcast -count=1`;
  mutation-check delayed-operation tests by removing the final generation check.
- [x] Commit `#523 M1: enforce capability switches at mutation boundaries`.

### Task 3: HTTP and SSE contract

Files: server_board.go, server_drive.go, server.go, stream.go, link.go;
server_board_test.go, server_drive_test.go, server_test.go, stream tests.

- [x] Write failing tests using the Task 3 function strategies below.
- [x] Implement fixed disabled error messages, typed generation handling and
  bounded latest-value SSE updates derived from authorities.
- [x] Run the webcast package tests and relevant race tests; update
  atlas/webcast.md and atlas/webcast-format.md for the new surface.
- [x] Commit; run `sdlc milestone-close --issue 523 --milestone M1` with verified
  behavior evidence and measured actual as the gate requests. Resolve findings.

## M2 — Controls, viewers and integration

### Task 4: One click rule and truthful controls

Files: broadcast/pointer.go, session.go and tests; couchtty/cast_control.go,
console_pointer.go, console_webcast.go, terminal_input.go, reserve.go, terminal.go,
console_castcodes.go only where token readiness changes; associated tests.

- [x] Write failing tests using the Task 4 function strategies below.
- [x] Implement EnsurePointerLink and shared intent; web click enqueues bounded
  cast-owned work and reports its result, never blocks Couch on CDP.
- [x] Render amber only when enabled, dim when off; preserve hit spans and widths.
  Existing code hover and Alt-click remain unchanged.
- [x] Run broadcast/couchtty tests, including hidden terminal marker enforcement,
  clipped spans, delayed cast stop and stale updates; commit.

### Task 5: Viewer availability and overlay handshake

Files: web/viewer.js, annotate.js, drive.js; tests/webcast/*.test.mjs;
Chrome/operator seams listed above if #494 is on main, plus scripted/live tests.

- [x] Write failing tests using the Task 5 function strategies below.
- [x] Implement replay-latest capability consumption, disabled messages, no-flush
  gesture cancellation, held draft and fresh consent on enable.
- [x] Integrate #494 according to landing order; record exact API ownership and
  run delayed-context/permission plus new-document tests. Do not open a window.
- [x] Run `node --test tests/webcast/*.test.mjs tests/broadcast-viewer/*.test.mjs`
  and `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run Live -count=1 -v`
  using the existing headless temporary-profile harness; commit.

### Task 6: Acceptance, documentation and landing

- [x] Update README, atlas/broadcast.md, atlas/webcast.md, atlas/webcast-format.md
  and atlas/index.md only if new atlas files are added. State defaults, click
  table, stable links, clear semantics, error behavior and overlay handoff.
- [x] Run `make test` and targeted race tests for capability owners. Check full
  logs and exit codes. Fix failures without widening unrelated scope.
- [x] Send TL readiness; stage to pair:0 and build only when instructed. Operator
  checks annotation OFF/ON and drive OFF with active/pending viewers, right-copy
  while OFF, terminal pointer behavior and new-page overlay availability.
- [x] Close M2 with verified evidence and binary-owned fresh-context review;
  update issue/plan checkboxes and lessons for any review findings.
- [ ] Merge current main, resolve/test, close issue, open PR, wait for CI and
  SHIP, then `sdlc merge --yes` with all gates enabled; report outcome to hub.

## Alternatives considered

1. **UI-only switches:** smallest diff but cannot refuse a stale or scripted
   client and cannot disable operator admission. Rejected.
2. **Delete/recreate share links on toggles:** defeats stable URLs/codes and
   invalidates viewer identities. Rejected.
3. **Existing authorities + ordered clear/reset (chosen):** larger than a UI
   patch, but preserves one truth per capability and recorder consistency.
   Multiple note-resolution/deny records were considered; one clear/reset avoids
   partial progress being presented as an atomic switch and stale catch-up.

## Approval and execution boundary

Interaction semantics are approved by TL receipt
9e080cee-0e91-4054-ab52-d84a5fd86bd4. Send this design/plan to the hub before
code. Run the full-flow `sdlc change-code` plan-quality gate after design
approval; derive the estimate only after that gate requests it. No
implementation or tests have been changed while authoring this plan.

## Revisions

### 2026-10-12 23:08 PDT — Visible recovery for every capability phase

TL approved the design at 2ddb90a9f (receipt
6f882976-021f-41f3-833c-5a2e0c30aba4), requiring a small user-facing state
projection with one recovery action. This revision supersedes the earlier
unspecified cleanup retry and extends castControlIntent beyond an on/off bit.
No hidden automatic retry loop is permitted. Apply this rule to both annotation
clear and drive reset/held-input release, including their overlay integration.

| Internal phase / cleanup result | Operator cast row and notice | Viewer projection | One operator action |
| --- | --- | --- | --- |
| Off | Dim control, OFF | "Annotation is off" / "Driving is off" | Left-click that control to enable and copy |
| On | Amber control, ON | Relevant tools available, subject to link role | Left-click that control to turn off |
| Disabling / operation in flight | Dim control, "Turning annotation/driving off…" | Capability off; gestures/input canceled | Wait for this bounded operation; repeated clicks do not enqueue duplicates |
| Disabling / retryable cleanup failure | Dim control; "Couldn't clear marks: <reason>; click 👆 to retry" or "Couldn't release control: <reason>; click 👽 to retry" | "Annotation/Driving is off; waiting for the operator to retry" | Left-click that same control to retry cleanup |
| Disabling / terminal recorder or CDP loss, or cast fatal failure | Both controls unavailable; "Cast broken: <reason>; end the cast" | "Cast unavailable; waiting for the operator to end it" | End the cast using its existing end action |
| Ended | Existing ended/removed cast row | "Cast ended" | No recovery in this cast; start a new cast through the existing start action |

The in-flight row is bounded by existing operation timeouts and must settle to
Off, retryable failure or broken. The failure notice remains discoverable with
the control's status until resolved; a transient notice alone is insufficient.
Detailed errors accompany the action and never replace it. Read-only viewers
cannot perform these operator actions; their messages name who must act.

Cleanup outcome is owned by the authority/Cast, not reconstructed from a UI
boolean or error string: typed retryable versus terminal results drive the
projection. Cancellation because the cast ended projects Ended; a closed
recorder, detached/unrecoverable CDP session or a fatal cast error projects
broken. A timeout while the dependencies remain alive is retryable. Each attempt
is single-flight. A retry reconciles ordered state idempotently, including an
earlier attempt whose acknowledgement was lost; it cannot reopen admission or
duplicate releases. Successful retry finishes OFF, clears the failure notice
and offers the normal enable action. It does not enable as a side effect.

Right-click remains copy-only in all supported active-cast states, including
cleanup failure; it never starts a retry or changes admission. Broken and
in-flight left-clicks show their existing action/status rather than accidentally
enabling. Terminal pointing keeps its existing lifecycle and gains only the
approved ensure-link/copy behavior.

M1 and M2 verify this projection through the function strategies below.
Include recovery in the operator acceptance script and README/atlas docs.

### 2026-10-12 23:12 PDT — Plan gate PQ-1/PQ-2 dispositions

PQ-1: ARCH-ORDER uncertain-effect reconciliation applies to every input release
path, not only the capability button. Existing controller_mem.go changeLocked
launches an unowned goroutine and discards release errors; drive/tab.go release
clears held input before dispatch. Neither behavior can implement the recovery
contract above. The following replaces that reuse assumption and narrows the
previous blanket timeout-is-retryable classification.

**One owned release path.** MemController owns a bounded cleanup operation with
cast context, cancellation and join, replacing detached release goroutines for
all control changes. Capability cleanup closes admission first, confirms the
ordered CtlReset, then awaits the same Releaser path. Other control transitions
also use this path so granting a new driver cannot race unresolved release.
No controller mutex is held while waiting for recorder/CDP. Controller state
retains the outgoing driver, operation generation and cleanup progress until
completion; new driving remains gated while release is pending or failed.

TabInjector retains each held key/button until its release is acknowledged.
Successful releases remove only their own entry; a definitely unsubmitted or
explicitly rejected operation retains the entry for retry. The same guarded
release helper serves control change, navigation and refused-batch cleanup.
The recorder queue serializes this with input, and the capability generation is
rechecked at actual input dispatch. No newer driver can press keys while the
old driver's cleanup is unresolved. Close cancels and joins owned workers;
CDP session destruction terminates their relevance without claiming a successful
release in a still-live session.

**Uncertainty is terminal, not a false retry.** A timeout/cancellation before
queue submission has no external effect and may offer Retry. Once a CDP release
command has been submitted, absence of a conclusive acknowledgement is an
uncertain external effect: mark the cast broken and offer End cast. Do not
blindly repeat it or infer success from an empty local hold set. A transport
result carries a typed outcome (confirmed, definitely-no-effect, uncertain,
terminal); it is not classified by matching message text. The CDP adapter must
preserve submission evidence. Successful reply is completion evidence; detached
session/recorder is terminal. Late replies may settle diagnostics for that
attempt but cannot reopen a broken cast or clear a newer failure.

Ordered clear/reset are different: they are idempotent local recording effects.
A cleanup generation has a stable operation ID; the recorder caches its last
completed ID/result until that cast ends. Retry queries/enqueues that same ID,
so an acknowledgement lost after commit cannot append a duplicate reset/clear.
A single latest completion per capability suffices because attempts are serial
and stale generations are rejected. Release resumes only definitely unperformed
entries; uncertain CDP delivery never reaches this retry path.

**Pure transition completion.** Extend the proposed state with Attempt uint64
and a tagged cleanup outcome (Running, Retryable, Broken), present only during
Disabling. Event becomes a tagged value: Enable, Disable, Retry,
CleanupSucceeded(attempt), CleanupFailed(attempt, kind), End. Disable and Retry
allocate an attempt; Retry is legal only for Retryable. Cleanup events from
another attempt are ignored. A Running attempt cannot be retried or enabled;
Retryable allows only Retry; Broken allows only End. Success transitions to Off,
never On. Generation advances on admission-changing transitions; attempt changes
fence worker completion independently. The projection function maps this model
to the previously approved visible states and actions. No UI owns duplicate
retry state.

PQ-2: the test-case inventories above are compressed into the following named
function strategies. Acceptance semantics remain in the scope/projection tables;
executable cases belong in tests. Proposed function names are marked new.

| Task | Production function(s) | Adversarial class and mechanical guard |
| --- | --- | --- |
| 1 | NextCapability (new) | Arbitrary tagged event sequences; table/property tests prove admission and attempt invariants. |
| 1 | State.Apply, NextControl, CatchUp; JS Model.apply | Generated valid timelines plus malformed clear/reset records; Go/JS canonical equivalence and replay-prefix comparison. |
| 2 | Board.Stroke, Note, SetState, SetAnnotationAllowed (new) | Concurrent mutation and cleanup schedules; barriers prove generation admission and ordered emission. |
| 2 | Writer.Emit, recorder.Emit and ApplyCapabilityCleanup (new) | Delayed/lost acknowledgements and lifecycle changes; stateful writer proves deduplication and final authority checks. |
| 2 | MemController.Request, Grant, Input, SetDriveAllowed (new) | Arbitrary control/cleanup interleavings; stateful sink/injector verifies no driving while cleanup is unresolved. |
| 2 | TabInjector.Inject, Release, release | Partial and uncertain CDP outcomes; stateful held-input fake verifies retained evidence and typed terminal/retry decisions. |
| 2 | Cast.SetCapability, SubscribeCapabilities (new) | Cancellation and stale worker completions; controlled worker barriers prove bounded lifetime and latest-generation projection. |
| 3 | Server.boardRoute, driveRoute, stream capability event handler (new) | Untrusted roles and delayed HTTP bodies; real HTTP fixtures enforce role intersection and final permit validation. |
| 4 | castControlIntent, capabilityProjection (new) | Exhaustive phase/outcome/input domain; pure table tests enforce one visible recovery action. |
| 4 | Session.EnsurePointerLink (new), terminal/web click dispatch | Failed link/copy operations and repeated clicks; stateful session/clipboard fixtures prove stable secrets and unchanged copy-only admission. |
| 5 | mountAnnotate, mountDrive, capability subscriber (new) | Reordered stream/module/gesture events; browser stateful fixtures prove cancellation and replay-latest permission. |
| 5 | startOperator, refreshOperator, annotationPageLocked, operatorAdmission (#494) | Replaced contexts and delayed permissions; scripted CDP peer plus isolated headless conformance proves exact-context fail-closed admission. |

The full test commands and the two review boundaries above remain unchanged.

### 2026-10-12 23:23 PDT — Persisted ordering of annotation clears

Task 2 must preserve the clear barrier across Split/Merge, not only the live
queue. Existing recording/files.go Merge orders equal timestamps by events,
notes, pointer; therefore recorder mirror.emit and reference Writer cleanup
stamp AnnotationClear at max(clock, previous record T+1). All subsequent emits
retain the existing monotonic clamp. Add a stateful equal-clock split/replay
strategy to their tests. No file rank or old recording ordering is changed.

### 2026-10-13 00:26 PDT — M1 boundary review corrections

BR-1 (ARCH-PURPOSE): the recording consumer inventory is Go State.Apply,
CatchUp, Split/Merge, JS Model.apply, replay.js catchUp and Player.seek.
Replay catch-up must retain annotation-clear across resumes and reset its kept
note IDs. Shared capability-events.jsonl prefix and arbitrary-order seek tests
compare current notes AND strokes with sequential reduction; both reproduced
resurrection before the fix and pass afterward.

BR-2 (ARCH-ORDER, ARCH-MOCK): ordinary control records need the same production
commit evidence as cleanup. Add an acknowledged write through liveGate, Chrome
and the ordered Recorder sink; a successful enqueue is insufficient. Stateful
production-adapter tests delay completion, close the recorder and fail its sink.
Preserve started-operation ownership when a caller cancels.

BR-3: Board annotation admission lives in new board/annotation.go alongside
modified board/board.go; the concept table and task paths now name those files.
BR-4: README now documents default-OFF admission and distinguishes persistent
link permissions from capability availability; M2's visible switches remain
explicitly pending until implemented.

### 2026-10-13 00:34 PDT — Landed overlay seam integration

#494 landed in PR303 (889fafd05); #523 lands second and owns its wiring.
Keep the approved held-draft behavior: an unsent prepared draft remains usable
on a fresh save after re-enable in the same document. Capture authority at
binding ingress, then recheck the captured permit at prepare completion and
ordered mutation dispatch. This fences already queued work across Off/On without
invalidating the operator's unsent text or requiring a replacement draft.
Status/cancel remain available for reconciliation while Off.

Recorder operator processing must not acquire Board.command: disable holds
that serialization lock while awaiting its ordered recorder clear. Operator
admission therefore uses the leaf Board.mu check and recorder ordering for its
final write, while viewer mutations retain command serialization.

New isolated contexts construct annotation UI disabled. The injected script's
initial hello may precede DOMContentLoaded and hook installation; report readiness
again after mount so the existing refresh path applies current capability to
that exact context. Missing hooks do not count as successful delivery. Permission
updates must not reset pen mode on unrelated view refreshes, and delayed older
permission must not overwrite a newer one. Scripted-context and headless tests
cover readiness, replacement, authority changes and held-draft restoration.

### 2026-10-13 01:12 PDT — M2 asynchronous UI evidence

ARCH-ORDER: capability and control SSE channels have no relative delivery order.
Keep the latest control snapshot until the initial authoritative capability
arrives, while a new generation discards old consent. Mounted viewer fixtures
exercise both orders, reconnect and delayed module initialization.

Cast-row completion captures generation and attempt and rechecks them on the
terminal command loop before copying. A deterministic completion barrier proves
that a newer disable/end cannot be followed by stale enable-copy; removing the
guard makes the regression fail. Annotation-clear and capability changes also
clear transient viewer/operator stroke caches, not only the recording reducer.

The full headless run exposed two old assumptions in live fixtures: Board now
starts OFF, and an uncertain CDP dialog result now requires end-cast. Fixtures
explicitly enable annotations and assert the approved Broken projection after
uncertain input. Both affected tests pass, as do the operator/annotation/drive
headless subset. A fresh complete headless run and make test are underway.

### 2026-10-13 01:18 PDT — M2 verification complete, acceptance pending

Fresh make test exited 0, including shell checks and go test ./... -count=1.
All webcast packages passed under -race; all 229 browser tests passed; the full
headless Live suite passed in 159.587s. Logs: /tmp/pair-523-m2-suite.log,
/tmp/pair-523-m2-race.log, /tmp/pair-523-m2-node-final.log and
/tmp/pair-523-m2-live-final.log. Hub readiness was sent; pair:0 staging and
operator acceptance remain pending. No visible window was opened.

### 2026-10-13 08:56 PDT — Smoke error must remain readable

TL reported a tunnel startup error hidden by cast-row clipping. Investigation
stopped at TL instruction: operator is traveling with network trouble; no
cloudflared reproduction or live Couch changes. The transport cause is unproven.

Full text already lives in the bounded Feed.Messages snapshot, including after
row expiry. Reuse it (ARCH-DRY) with a switcher-only Ctrl+Y copy-last-notice action
and visible hint. Copy via Presenter.Copy outside the console mutex; confirmation
belongs to the menu so it cannot replace the source notice. Tests must verify
clipped/expired full text, repeat copying, empty feed, and actor-focused Ctrl+Y
passthrough. This addresses the operator-approved copyable-line recovery without
adding a second history store or changing tunnel behavior.

Local-only acceptance uses existing COUCH_BROADCAST_TUNNEL=off with
COUCH_BROADCAST_HOSTNAME unset/empty on the operator's next launch. Both casts use
LocalOnly loopback links. All #523 capability checks can run with local viewer
identities; this does not validate remote tunnel transport. No live restart is
performed by this agent.

### 2026-10-13 — Operator acceptance and driver chip refinement

Operator smoke passed: right-copy, annotation clearing/held draft/pen Off, drive
revocation with pending-request removal and fresh consent. Receipt
762b6600-ba7b-4038-b46e-d34383665f67 approves one display refinement: OFF plain
👽; ON idle amber 👽; ON occupied amber 👽 plus driver name in the same clickable
capability span. Remove the separate active-driver chip; retain the hand-raise
request span and its popup. Whole driver chip left toggles OFF and right copies.
Test the three renderings, clipped hit spans and click semantics; build on pair:0
and wait for the operator's quick look before landing.

### 2026-10-13 — M2 BR-5 generation provenance and BR-6 retry envelope

ARCH-ORDER/ARCH-SECURE: ingress-time permits alone do not fence requests that
were created before OFF but arrive after OFF/ON. Carry the capability generation
observed at browser request creation to the server: operator permission delivery
sets the generation used by binding envelopes, and viewer mutation requests send
a decimal generation header. Missing, malformed or retired generations fail
closed at ingress. Capture a Board permit for that exact generation and retain
its existing ordered-dispatch check. Status/cancel reconciliation stays available;
an unsent held draft uses a fresh generation on a fresh save after enable.

Annotation-clear cancels the shared UI gesture and puts the pen in watch even
if its separate capability event is delayed. Tests hold operator/viewer payloads
before ingress across OFF/ON, verify all mutation routes, and deliver clear while
a stroke is held. Audit drive request/input provenance for the same class.

The operator permission worker is the explicit exception to the original
no-periodic-polling envelope: one cast-owned 250ms ticker drives retries only
while delivery is unresolved (nil retry channel otherwise), never an enable or
cleanup retry. The worker stops its ticker/subscriptions on cast shutdown; exact
context state is discarded with replaced contexts. Each CDP delivery is bounded
by its call deadline. Readiness events remain the primary trigger; this bounded
retry covers an unavailable hook or unacknowledged delivery, without treating
an uncertain ON as permission to mutate.

BR-5 drive audit confirmed the same delayed-before-ingress risk for request,
release and input. All three now carry their originating drive generation;
input batches freeze it when created. Tests reject missing/retired generations
after off/on even after a fresh grant, while current-generation requests succeed.
BR-6 clarification: normal notifications can reconcile sooner than 250ms; the
single worker's timer only supplies unresolved-delivery retries. An acknowledged
delivery removes the timer from selection, and cast shutdown stops it.
