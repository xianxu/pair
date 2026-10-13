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
| Board annotation admission | cmd/internal/webcast/board/board.go, capability.go | modified | request/emit ordering and recording sink |
| Recorder annotation commit | cmd/internal/webcast/recorder/recorder.go | modified | ordered stream writes and clear acknowledgment |
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

- [ ] Write failing transition-table tests for enable/disable/cleanup/retry/end,
  generations and all four independent capability combinations.
- [ ] Write Go/JS clear and CtlReset fixtures: notes+strokes+pending+driver,
  clear while paused, repeated reset, end refusal, catch-up after resume and
  replay before/after clear. Compare canonical state, not only raw records.
- [ ] Run `go test ./cmd/internal/webcast/capability ./cmd/internal/webcast/recording -count=1`
  and `node --test tests/webcast/model.test.mjs`; confirm expected red.
- [ ] Implement the pure types/transitions and register additive format kinds;
  include clear in catch-up context and enforce reset operator attribution.
- [ ] Run the same tests green; commit `#523 M1: model capability transitions and ordered clears`.

### Task 2: Admission and bounded cleanup

Files: board/capability.go, board/board.go, board/writer.go; recorder/recorder.go;
drive/controller.go, controller_mem.go, controller_contract.go, tab.go;
cast_capabilities.go, cast.go; corresponding tests and existing fake seams.

- [ ] Add failing direct and stateful tests for annotation/refusal across all
  Board mutations, clears and pending notes, plus delayed enqueue/clear ordering.
- [ ] Add failing controller tests: active+multiple pending reset, blocked
  Request/Grant/Input, stale batch after off/on to same viewer, held input release,
  sink/cleanup failure and retry, stop during cleanup.
- [ ] Implement generation-bound admission and cleanup acknowledgments as above;
  preserve Board leaf-lock invariant and recorder's single writer.
- [ ] Wire Cast defaults OFF before exposing routes, supported capability snapshots,
  latest-value subscriptions and cancellation/close. Fakes use the same transitions.
- [ ] Run `go test ./cmd/internal/webcast/board ./cmd/internal/webcast/drive ./cmd/internal/webcast/recorder ./cmd/internal/webcast -count=1`;
  mutation-check delayed-operation tests by removing the final generation check.
- [ ] Commit `#523 M1: enforce capability switches at mutation boundaries`.

### Task 3: HTTP and SSE contract

Files: server_board.go, server_drive.go, server.go, stream.go, link.go;
server_board_test.go, server_drive_test.go, server_test.go, stream tests.

- [ ] Add failing real HTTP/SSE tests for each link × switch combination;
  refuse disabled POST before body read and recheck after a delayed body.
- [ ] Test connected viewers and reconnects get latest effective state; seats
  remain available while driving is off; unsupported casts remain unsupported.
- [ ] Implement fixed disabled error messages, typed generation handling and
  bounded latest-value SSE updates derived from authorities.
- [ ] Run the webcast package tests and relevant race tests; update
  atlas/webcast.md and atlas/webcast-format.md for the new surface.
- [ ] Commit; run `sdlc milestone-close --issue 523 --milestone M1` with verified
  behavior evidence and measured actual as the gate requests. Resolve findings.

## M2 — Controls, viewers and integration

### Task 4: One click rule and truthful controls

Files: broadcast/pointer.go, session.go and tests; couchtty/cast_control.go,
console_pointer.go, console_webcast.go, terminal_input.go, reserve.go, terminal.go,
console_castcodes.go only where token readiness changes; associated tests.

- [ ] Test right-copy of unminted terminal pointer makes token+code atomically
  but never enables, preserves notes/marks state, and retries mint failure safely.
- [ ] Test one pure click-intent matrix plus real dispatch on all three buttons;
  stable links/code values, repeat enable copies, copy failure not fake rollback.
- [ ] Implement EnsurePointerLink and shared intent; web click enqueues bounded
  cast-owned work and reports its result, never blocks Couch on CDP.
- [ ] Render amber only when enabled, dim when off; preserve hit spans and widths.
  Existing code hover and Alt-click remain unchanged.
- [ ] Run broadcast/couchtty tests, including hidden terminal marker enforcement,
  clipped spans, delayed cast stop and stale updates; commit.

### Task 5: Viewer availability and overlay handshake

Files: web/viewer.js, annotate.js, drive.js; tests/webcast/*.test.mjs;
Chrome/operator seams listed above if #494 is on main, plus scripted/live tests.

- [ ] Add failing browser tests for off/on with an open draft, active stroke,
  held drive input, dynamic module mount after toggle, reconnect after clear,
  and delayed old events. View-only links never gain permission.
- [ ] Implement replay-latest capability consumption, disabled messages, no-flush
  gesture cancellation, held draft and fresh consent on enable.
- [ ] Integrate #494 according to landing order; record exact API ownership and
  run delayed-context/permission plus new-document tests. Do not open a window.
- [ ] Run `node --test tests/webcast/*.test.mjs tests/broadcast-viewer/*.test.mjs`
  and `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run Live -count=1 -v`
  using the existing headless temporary-profile harness; commit.

### Task 6: Acceptance, documentation and landing

- [ ] Update README, atlas/broadcast.md, atlas/webcast.md, atlas/webcast-format.md
  and atlas/index.md only if new atlas files are added. State defaults, click
  table, stable links, clear semantics, error behavior and overlay handoff.
- [ ] Run `make test` and targeted race tests for capability owners. Check full
  logs and exit codes. Fix failures without widening unrelated scope.
- [ ] Send TL readiness; stage to pair:0 and build only when instructed. Operator
  checks annotation OFF/ON and drive OFF with active/pending viewers, right-copy
  while OFF, terminal pointer behavior and new-page overlay availability.
- [ ] Close M2 with verified evidence and binary-owned fresh-context review;
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
