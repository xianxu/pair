---
id: 000494
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 5.244
card_mirror: 'f31b7cdfedff1ad106595fc4868d40783a7b6441' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T20:05:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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
- Driver requests can be granted/denied and control taken back in the cast Chrome (M2); annotation is off and disabled while a viewer drives, with an open draft held until control returns.
- Navigation, pause, return of control and lost replies follow the document-scoped behavior in Revisions; remote input cannot activate operator controls.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. sdlc reports the calibration stale; provisional.
The rows below are, in order: spec/approval; Board protocol; recorder binding;
capture/input suppression; operator UI policy; shared-mount refactor; embedded-module
assembly; M2 Controller adapter; deferred menu rows; real Chrome conformance;
README/atlas; two milestone reviews; one operator UX iteration. Implementation values
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
item: smaller-go-module design=0.06 impl=0.2
item: tui-screen design=0.1 impl=0.12
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.05 impl=0.08
item: milestone-review design=0 impl=0.2
item: milestone-review design=0 impl=0.2
item: ux-rename-iteration design=0.3 impl=0.08
design-buffer: 0.15
total: 5.244
```

Design 2.16 × 1.15 + implementation 2.76 = **5.244 focused ship-hours**.

## Plan

- [ ] Write and approve the durable implementation plan at `workshop/plans/000494-operator-overlay-plan.md`.
- [ ] M1 — Operator annotation: shared Off/Pen/Note UI, Board/recorder bridge, capture exclusion, document identity and driving-disable behavior; targeted/headless proof and milestone review.
- [ ] M2 — Grant / Deny / Take back in cast Chrome; deferred Couch menu integration after #506 and hub release; operator smoke acceptance and milestone review.
- [ ] Close, merge fresh main, publish the PR and land on SHIP.

## Log

### 2026-10-12

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
