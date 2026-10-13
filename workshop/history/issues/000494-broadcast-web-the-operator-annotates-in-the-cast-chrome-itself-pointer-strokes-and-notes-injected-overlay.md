---
id: 000494
status: done
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 4.775
card_mirror: 'efd2cb8de6570da0a0db378d27574b3fb9c15de2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T20:05:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 3.40
---

# broadcast-web: the operator annotates in the cast Chrome itself (pointer strokes and notes, injected overlay)

## Problem

In the web cast, viewers point and leave notes (pair#470), but the operator, who is using the page in the cast Chrome, can't: they'd need a separate viewer window. The operator (10-12): the Chrome window Couch spins up should have the pointer and annotation facility itself, so the operator can use that screen directly and mark what they're pointing at for everyone.

## Spec

- **Injected overlay:** reuse the viewer's annotation pieces (`collab.js`, the swatch rail, pen strokes, notes, and #488's boxes once landed) in the cast tab, injected through CDP into the same isolated world as the Go-live button (#454), so the page's scripts can't see or affect it.
- **Never recorded as page content:** the overlay's nodes must not enter the recording as DOM (the recorder filters Couch's own injected nodes, as for the Go-live button); its strokes and notes enter as `stroke`/`note` records through the Board, authored as the operator.
- **Modes:** off by default; a toggle (in the overlay and a Couch key or the Chrome cell's menu) turns on pen or note mode, during which pointer input goes to the overlay, not the page; off returns input to the page. The operator's plain pointer is already streamed (`ptr`, `hover`).
- **While a viewer drives** (#467), the overlay stays available to the operator and never steals the driver's input.
- Depends on pair#493 (notes and strokes reach the recording) for replay and queries.

## Done when

- The operator draws and leaves notes in the cast Chrome; viewers see them live, attributed to the operator; the page's DOM in the recording has no overlay nodes; replay and queries show the operator's notes (after #493).
- Tests; the operator smoke-tested it.
- Fidelity screenshots and submission page captures contain no Couch UI; failed or unconfirmed suppression never publishes a contaminated capture.
- Annotation is off and disabled while a viewer drives, with an open draft held until control returns; driving and broadcast authority stays in Couch.
- Navigation, pause, return of control and lost replies follow the document-scoped behavior in Revisions; remote input cannot activate operator controls.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. sdlc reports the calibration stale; provisional.
The rows below are, in order: spec/approval; Board protocol; recorder binding;
capture/input suppression; operator UI policy; shared-mount refactor; embedded-module
assembly; deferred menu rows; real Chrome conformance;
README/atlas; one close review and shipping; one operator UX iteration. Implementation values
are 40% of the v2/v2.1 primitive ranges. Existing Board/CDP/collab/Controller cover
the library-availability check; no new library assumed. Remaining design uses the
thorough-spec discount (0.2) on the selected primitive bases, except spec authoring
and the anticipated operator UX iteration. Familiarity 1.0 uses existing repo seams;
15% design buffer because the plan and gate resolved the main ordering decisions.
Headless proof is in the integration/UI rows plus the explicit real-API discovery
row; no invented concurrent-work discount. Smoke scheduling waits are not active time.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.6 impl=0.08
item: smaller-go-module design=0.04 impl=0.16
item: api-integration design=0.3 impl=0.4
item: api-integration design=0.3 impl=0.4
item: tui-screen design=0.25 impl=0.32
item: cross-cutting-refactor design=0.12 impl=0.16
item: smaller-go-module design=0.04 impl=0.12
item: tui-screen design=0.1 impl=0.12
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.05 impl=0.08
item: milestone-review design=0 impl=0.2
item: ux-rename-iteration design=0.3 impl=0.08
design-buffer: 0.15
total: 4.775
```

Design 2.10 × 1.15 + implementation 2.36 = **4.775 focused ship-hours**.

## Plan

- [x] Write and approve durable plan; clear change-code's full-flow gate.
- [x] Implement operator Board protocol and bounded current annotation projection.
- [x] Implement shared annotation mount, isolated bridge and capture exclusion.
- [x] Verify off/paused/driving/document lifecycles headlessly and update README/atlas.
- [x] After #506 and hub release, merge main and add Couch annotation menu rows.
- [x] Obtain operator smoke acceptance through hub.

Close, PR and SHIP-authorized landing follow this acceptance checkpoint.

## Log

### 2026-10-12
- 2026-10-12: closed — Operator smoke PASS (799f7b20). BR-1..5 addressed with mutation-sensitive regressions: 215 browser tests, real headless operator/module conformance (0.94s), recorder tab-return ledger/fresh-save race tests passed. Earlier full make test passed; final full rerun failed only two unrelated couchcore host-creation-busy tests, both passed unchanged isolated rerun (21.628s), recorded with exact logs. All webcast packages and menu checks pass. Pair:0 restored to main; #523 wires hook second.; review verdict: SHIP

- Claimed in pair:1 and entered planning from fresh main. start-plan refreshed the
  ariadne layer and required a second invocation; no implementation changes made.
- Read the issue, project decisions, relevant lessons, control/Board/recorder seams.
  The isolated world protects JavaScript, not the shared host DOM. The recorder's
  DOM exclusion already exists; its screenshot worker has no UI-hiding bracket.

## Revisions

### 2026-10-12 — design proposal from the assigned slot

Reason: the project adds browser-side driving controls, and inspection exposes input,
recording and document-lifetime constraints. This elaborates the original Spec; it does
not retire its acceptance criteria. Pending hub approval.

**Experience.** The cast Chrome's existing Go-live control gains Off, Pen and Note
controls with the viewer's color rail. Annotation is off at launch and in each new
document. Pen draws transient strokes; Note draws a box (a click uses the shared
one-tenth-viewport default) and opens the shared note editor. Identity is fixed to
`operator` / `Operator`, never requested from or stored in the page. The ordinary
pointer continues through the existing observer. Turning annotation off immediately
returns page input; Escape cancels the current gesture/editor and turns it off.
Accepted notes and viewers' notes appear as the same boxes/pins on the cast page.

**Driving.** Pending driver requests appear beside Go live with named Grant / Deny
buttons. While someone drives, show their name and Take back control. These invoke
the existing Controller, as the Couch controls do. They remain usable while paused;
annotation writes do not. An operator can annotate while a viewer drives. Each
remote-input batch temporarily removes the overlay from hit testing, releases its
pointer capture and directs remote keys to the page rather than the note editor.
An interrupted operator gesture is cancelled, not completed by the remote pointer.
The editor retains its draft. Cleanup runs on success, error, expiry and navigation.

**Recording.** Reuse Board validation, anchoring, limits and the recorder's single
writer. Notes and strokes carry operator authorship and reach current live viewers,
disk, replay and queries through the existing record types. HTTP viewer requests must
continue to reject the reserved operator identity. No overlay nodes, styles, note
editor text, or control interactions enter recorded page content. Hide the entire
Couch UI during fidelity screenshots and submission captures, restoring it even when
capture fails, so its pixels do not contaminate the page image. No new file format.

**Document identity.** An annotation request is scoped to its CDP session, isolated
execution context and recorded document. Resolve a draft's anchor when the box is
completed, and retain that node/document identity through Save; never reinterpret an
old path on a replacement node. Navigation, tab changes and cast end cancel pending
input and stale replies. Pause cancels gestures, disables annotation and holds a draft;
resume on the same document may resume editing, but does not auto-enable the pen.
Errors use the Board's existing fixed refusal messages in the overlay.

**Reuse and delivery.** Reuse collab.js's dialog, swatches and box geometry, and
annotate.js's stroke/pin rendering and gesture rules through an injected adapter.
Keep one implementation for viewer and operator gestures. The operator adapter uses
CDP, never page-origin HTTP, injected credentials, localStorage or a public viewer URL.
The host and its closed shadow tree use the existing data-couchcast-ui exclusion.
Isolation protects bindings and closure state; the host remains in the shared DOM.

**Couch entry point.** Add annotation choices to the Chrome cell's Alt+left-click
menu, sharing its state with the browser controls. Preserve the dispatched cast-row
rule: left click acts, right click copies a link, Alt+click menus only on LIVE/Chrome.
Coordinate with the current cast-menu owner before touching this integration. Do not
stage or build in pair:0 until the hub requests it.

**Alternatives considered.** A separate operator viewer avoids injection complexity
but fails the operator's requirement to work in the cast Chrome. A standalone copied
annotation implementation avoids refactoring but duplicates gesture, pause and box
rules. The recommended adapter keeps the existing Board and shared UI as authorities
(ARCH-DRY, ARCH-PURPOSE).

**Verification contract.** Targeted pure tests and stateful CDP tests cover stale
contexts/anchors, malformed bindings, bounds, author reservation, pause, navigation,
shutdown, driver failure and capture restoration. A headless temporary-profile Chrome
run proves the real operator gestures reach a viewer and mid-cast recording, notes
survive replay/query, overlay DOM and pixels are excluded, page input works when off,
and remote mouse/keyboard input bypasses the overlay even with a note editor open.
Use no foreground activation and no broad concurrent suites. The operator's live
smoke test remains required; arrange it through the hub before declaring Done.

### 2026-10-12 — TL approval and milestone order

The TL approved e4e1280f3, including exclusion of all Couch UI from captures with
restoration on failure. M1 is annotation plus recording/capture/document correctness;
M2 adds Grant / Deny / Take back and must not hold M1's review/readiness. The Chrome
Alt-click menu is owned by pair:4 (#495), then pair:5 (#506). Do not edit it before
#506 lands and the hub says to merge main and proceed. No staging in pair:0 without
an explicit hub request.

### 2026-10-12 — operator correction: annotation is unavailable while driving

This supersedes both the original Spec's “while a viewer drives” behavior and the
proposal's concurrent-input arbitration. The TL clarified that simultaneous operator
annotation was the TL's requirement, not the operator's. While a viewer drives,
annotation turns off; Pen/Note and pins are disabled with “X is driving”. A note draft
is held, hidden and unfocused, never lost merely because driving starts. After control
returns on the same document, editing is available again and the pen stays off.

The earlier remote-focus handoff and 300 ms typing-idle / 500 ms maximum hold proposal
is withdrawn in full: implement no typing delay or concurrent editor arbitration.
Remote input remains unblocked. A small document-scoped UI suppression bracket around
remote hit testing/dispatch protects the remaining Go-live and (M2) authority buttons
from synthetic input; it is not a concurrency mode for annotation. Lost suppression
confirmation refuses that input safely rather than risking an operator action.

Spec-review findings incorporated into the plan: capture and input suppression share
token ownership so an overlapping release cannot expose UI early; bindings and drafts
carry exact document identities; accepted writes are not rolled back when their reply
is lost; retries reconcile a bounded operation identity. Already accepted stroke chunks
remain recorded when a gesture is cancelled. Correct the original isolation wording:
page scripts cannot access the isolated bindings, but can see/change the shared host
DOM. All page coordinates remain attached to the draft's original box after scrolling.

### 2026-10-12 — plan-quality round 1

PQ-1..3 identified enqueue-versus-recorded settlement, input calls outliving their
caller and over-enumerated test prose. The plan now requires explicit completion and
reservation cleanup, target-lifetime protection on unconfirmed remote input, and
named-function adversarial strategies. No code has started. Full flow inferred from
M1/M2; approval for execution received from TL on cefcb28c2.

### 2026-10-12 — plan-quality clear

Second plan-quality round disposed PQ-1..3 as addressed (CLEAN). Derived the estimate
after that verdict. Baselines before implementation: Board package passes, and the
nine existing annotate Node tests pass. No production changes yet.

### 2026-10-12 — operator removes M2; terminal remains control authority

This supersedes all earlier browser Grant/Deny/Take back requirements and the briefly
requested input shield / Stop / codes controls. The operator finds broadcast control
unambiguous in Couch's cast row, not in cast/viewer browser windows. DROP M2 entirely.
#494 now delivers only operator Off/Pen/Note annotations, off while driven with draft
held, Board-backed recording and screenshot exclusion. Couch annotation menu rows
remain deferred until #506 and hub release. No new browser driving-authority controls,
no shield, no typing arbitration. Existing Go-live control is not expanded by this
issue. With one atomic delivery, remove milestone tags and use one sdlc close review;
the issue remains full-flow because its implementation exceeds the quick shell.

The approved M1 mechanisms and tests remain unchanged; this is a scope reduction after
change-code passed (plan-quality CLEAN, estimate-quality INFO). The new M2 request was
never implemented. Estimate revision removes M2 UI/control rows and one milestone
review; terminal integration and final close/ship stay included. Atlas design is an
undiscounted 0.05 h exception. Conformance implementation is included in recorder,
capture and shared-UI rows; real-api-discovery covers probing external behavior only.

- 2026-10-12 implementation checkpoint: shared annotation mount now accepts an excluded root and pure availability policy. Driving hides/blurs but retains a draft; return stays Off; navigation discards it. Node targeted annotation/operator tests: 12 passed. Trusted module assembly executes shared modules under Node through targeted Go tests. Board adapter has bounded typed requests, fixed operator identity, settlement outcomes and bounded projection; Board package/race passed. Recorder/Chrome integration, capture/input suppression, menu (awaiting #506 release), and live verification remain outstanding.

### 2026-10-12 — capture integration ownership

TL receipt 057ac00b confirms #494 owns `recorder.CaptureScreenshot`; whichever of
#483 and #494 lands second wires submission crop capture through this helper.
The helper requires the exact couchcast top context and discards bytes if either
suppression confirmation is missing. #506 remains unlanded; menu is untouched.

### 2026-10-12 — integration evidence

The actual shared UI prepared and saved an operator note in a temporary headless
Chrome (TestOperatorLiveAnnotation, 2.22s), through recorder ordering and Board,
with fixed operator attribution. Recorded DOM contains no Couch host/marker.
Capture suppression confirmed and restored the exact document host. Targeted
shared annotation/injection tests: 22 passed. Recorder operator race tests pass.
Existing screenshot fake tests need the now-mandatory isolated context and
suppression protocol; extending those fakes without weakening fail-closed capture.

### 2026-10-12 — annotation implementation verified; menu dependency remains

- Shared operator UI/Board/recorder integration is implemented. The remaining
  menu work is explicitly blocked until the hub releases #506; no menu rows changed.
  The Cast/Chrome mode seam is ready and never calls Focus.
- `TestOperatorLiveAnnotation` passed in a temporary headless Chrome (latest 2.55s):
  real Note and Pen gestures are recorded, fixed operator attribution, driver grant
  holds/hides/unfocuses a draft, remote hit-testing and text reach a page field under
  the old Couch button, release restores the draft with the pen Off. A magenta pixel
  witness proves the entire host is absent from suppressed captures and restored.
  Recorder output contains no Couch host or marker. Saved and inspected
  `/tmp/pair494-operator.png`; operator labels widened after visual inspection.
- Targeted terminal/shared/viewer/replay Node tests: 80 passed; later injection
  sequencing test also passed. Chrome lifecycle/operator tests and Couch webcast
  tests passed. Artifactpath production inventory check passed. Board and recorder
  operator/input/capture race subsets passed, including A→B→A same-Chrome-document
  return, rejection of old drafts/segments, stale capture discard and retained
  guards after uncertain input completes late. No full-machine suite or foreground
  Chrome was run.
- Mutation check: removing the capture acquire confirmation made all three capture
  protocol tests fail (including a screenshot issued without confirmation). Restored
  production helper; the same targeted test passed. Diff whitespace check clean.
- Filename/name mapping: planned live_operator_test.go/TestLiveOperator is
  implemented as operator_live_test.go/TestOperatorLiveAnnotation. Board's bounded
  operation transition lives in OperatorSession.run/Status; shared UI policy and
  suppression policy are pure JavaScript modules used by production injection.

- Final settlement audit: Status reconciles an unknown note only from the exact
  stable operator note ID actually observed by Board, without another write.
  Unknown resolve stays conservative because its record has no operation-specific
  ID; matching target state alone cannot prove this operation committed. Targeted
  Operator race tests passed after a RED→GREEN regression.

### 2026-10-12 — shared icon rail visual correction

TL receipt 6804f5e6 requests the viewer's own pen ✏️ and note 💭 buttons and
tooltips after inspecting the clipped word labels. Removed the operator-only
Off/Pen/Note label branch and width/font override: both mounts now use the exact
same rail. Off remains the default state; clicking the selected icon toggles Off,
as in the viewer, and Escape also turns annotation off. #506 menu release is
still pending; no menu changes.

Validation: 13 targeted annotation/operator Node tests passed; the real headless
Chrome annotation test passed (1.11s). Refreshed and inspected
`/tmp/pair494-operator.png`: both icons fit the shared pills without clipped labels.
`git diff --check` passed.

### 2026-10-12 — annotation capability input for #523

TL receipt 39206f41 requests one overlay input for the upcoming 👆 capability
switch. Added isolated-world `__couchcastAnnotationAllowed(boolean)` over shared
`setAnnotationAllowed`, default true. False cancels gestures, forces Off, disables
annotation actions and hides/holds a draft; true restores availability with the
pen Off, subject to driving/pause. Projection updates preserve the input. #523
owns the switch, feeding this input on replacement documents and server admission;
none of those are built here. Targeted policy/mount tests demonstrated RED→GREEN.

Verification: 14 targeted annotation/operator Node tests, shared-module assembly
checks, and the bounded headless Chrome conformance test passed (1.05s), including
disabling through the exposed isolated-world input and refusing Note until
re-enabled. `git diff --check` passed.

### 2026-10-12 — integrate landed #483 and #506

TL released the Chrome menu after #506/PR #302 and confirmed #483/PR #301 landed
first. Merged main and routed `Recorder.CaptureCrops` through the shared exact-document
capture helper, retaining its navigation barriers and box-text identity checks.
Extended the trusted module assembler only for local named export lists now used
by the shared viewer module. Preserved #506's view-only annotation flag in the
shared mount and kept it separate from the operator capability input.

Verification: crop/suppression/recorder-frame tests and shared-module/Chrome tests
passed. The real headless operator test now checks both fidelity and actual submission
crop pixels with the bright excluded-host witness, and host restoration after each
(latest 0.97s). All 196 webcast/broadcast-viewer Node tests passed. Pair:7 received
the hook installation, replacement-context delivery, Board and recorder admission
seams for #523; the second landing still owns wiring its authority.

### 2026-10-12 — menu ready and integrated acceptance checks

Chrome’s menu now has `Annotation · Off · Pen · Note` using the code rows’ existing
layout/hit-testing. It invokes the same mode command without focusing Chrome and
shows refusals in Couch. LIVE’s menu and cast-row clicks remain unchanged. Targeted
menu/click tests passed under the race detector (4.736s); tests prove no browser-control
or code-rotation side effect. The full `make test` is running through the machine gate.
All webcast Go packages passed; crop/lock-page real headless conformance passed.
The operator headless test additionally proves its note reaches a live subscriber
and the still-open on-disk query (2.17s). View-only shared-mount regression passed.
Operator smoke and single close review remain pending.

### 2026-10-12 — full suite passed; staging authorized

`make test` passed (exit 0): all shell/Lua checks and uncached `go test ./...`,
including couchcore (225.854s), couchtty (35.780s), artifact inventory and all webcast
packages. Log: `/tmp/pair494-full.log`. Targeted Board/recorder race checks also passed.
TL receipt 1dbbe706 releases pair:0 and authorizes merge-main, move and build for
operator smoke. Fresh main merge contained only #526’s issue document; no code conflict.
The source checkout stayed fixed until the suite finished; operator acceptance remains
pending after staging.

### 2026-10-12 — operator smoke: defer note-box preview until intent is known

TL receipts 959a49f8 and aa0d115c report that both the viewer and cast Chrome flashed
the default click box on mouse-down before switching to a drag box. The shared mount
now draws nothing on down or sub-threshold wobble (existing 8px mouse / 16px touch
threshold). Crossing the threshold previews the drag box and keeps drag intent even
if the pointer returns near its start. Release without a drag places the default
box; the completed box remains visible while its note draft is edited, and clears
when the dialog closes. Both surfaces use the same implementation (ARCH-DRY).

Six surface/gesture regressions demonstrated RED then GREEN for viewer and operator
click, drag and drag-back; they check preview timing and the submitted region. All
203 webcast/broadcast-viewer Node tests and real headless operator/shared-module
checks passed. Rebuilding the staged pair:0 candidate for operator re-test; no close
or landing acceptance is inferred from this smoke finding.

### 2026-10-12 — operator smoke accepted; close and land authorized

TL receipt 799f7b20 conveys the operator’s smoke PASS on overlay and shared gesture
fix at 92a1c4258b3bc38691b337647ee8d683721f426d: no box on press, drag box on drag,
default box on click-release on both viewer and cast Chrome. TL authorized closing
and landing from pair:1, with pair:0 returned to main. Moved back through `sdlc move :1`;
pair:0 is on main. Fresh origin/main merged cleanly with issue-document changes only.
#523/pair:7 will wire the annotation capability hook when it lands second.

Verification for close: full `make test` passed before the gesture correction;
after it, all 203 Node tests plus headless operator/shared-module tests passed,
including the actual pointer-down/up visual assertions (latest Chrome 1.25s).
Targeted Board/recorder/menu race checks passed. A final full suite is rerunning
on the accepted code. Close owns the mandatory fresh-context review and project
publication; no operator smoke or gate result has been inferred.

### 2026-10-12 — close round 1 fixes (BR-1, BR-2, BR-3)

- BR-1 / operation-result-identity: a reconciled result completes only the exact
  originating action and body. A changed draft or resolve→save sequence keeps its
  editor and returns a refusal explaining that the current change was not submitted.
  Settlement clears only its own in-flight identity. Tests reproduce old-draft
  cancellation/new-draft save and resolve/save sequences, including a fresh retry.
- BR-2 / keyboard-event-boundary: the operator listens for Escape at the window
  capture boundary because menu-selected modes preserve page focus. Off/unavailable
  annotation leaves page keys alone; remote Escape cannot discard a held draft.
  Policy/event regression plus real headless page-focused Pen and Note checks pass.
- BR-3 / core-concept-purity-classification: moved the observer path helper from
  PURE to INTEGRATION; it reads live DOM ancestry and global document identity.
  Its shared grammar remains unchanged (ARCH-DRY, ARCH-PURE).

All new regressions demonstrated RED→GREEN. All 206 browser JS tests passed; real
headless operator/shared-module checks passed (operator 4.72s). The close verdict
was REWORK, not final; a new close review will dispose of these findings. The
reviewer's isolated checkout lacked the generated runtime and its Chrome could not
start; our actual repository headless run and earlier full suite supply those
independent checks without waiving a gate.

### 2026-10-12 — final full-suite rerun result

The second `make test` completed with two couchcore failures, both reporting
“workspace host creation is busy”: `TestReconcileAgreesWithRealGit/upstream=broken/conflict`
and `TestSeparateGitDirectoryEnrollmentPreservesStorageAuthority`. All other packages
passed, including couchtty and every webcast package. No #494 change touches their
workspace-provisioning code. Both tests then passed together in an isolated uncached
rerun with no code changes (21.628s). Preserve the failed full result rather than
calling it a green suite: `/tmp/pair494-final-full.log`; isolated rerun:
`/tmp/pair494-core-rerun.log`. The earlier complete full run remains green.

### 2026-10-12 — close round 2 fixes (BR-4, BR-5)

Round 2 independently mutation-tested and accepted BR-1..3, then found transport
scope retirement and swatch identity gaps. BR-4: transport operation ownership
now follows recording document lifetime as well as action/body. Accepted projection
changes retire waiters/timers/unsettled state before updating UI; inactive tab,
replacement document and cast end retire it, while same-document pause/driving do
not. Request IDs stay monotonic across retirement; a generation check rejects even
an already-resolved old continuation without clearing a newer operation. Retirement
never resubmits the old write and never invents its success. New adapter/transport
sequence tests reproduce lost reply, switch-return, late reply and end races.

BR-5: shared swatches only rebuild when the selected color actually changes.
Unrelated scroll/pointer/focus/projection records retain the exact pressed element.
Both viewer/operator interleaved-click tests demonstrated RED→GREEN. Audited other
interactive nodes: mode/dialog controls stay mounted, and pinLayer already keys
chips by note ID. All 215 browser tests and bounded headless operator/shared-module
checks pass (latest operator 0.94s). No new scope or UI control was added.
