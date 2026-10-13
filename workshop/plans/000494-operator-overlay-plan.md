# Operator annotation in the cast Chrome implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent tasks, or superpowers-executing-plans for session-warm integration. Steps use checkboxes for tracking.

**Goal:** Let the operator annotate the actual cast Chrome, with annotations recorded through the existing Board and all Couch UI excluded from page captures.

**Architecture:** Reuse the existing isolated world and excluded control host. A bounded operator binding adapter connects shared annotation UI to Board through the recorder's document identity; the existing recorder remains the only record writer. A document-local suppression owner protects captures and remote input. Annotation is unavailable while a viewer drives.

**Tech stack:** Go, CDP, embedded JavaScript/CSS, existing Node tests and headless Chrome harnesses. No new service, browser extension or external JS dependency.

**Approved scope:** Issue #494 Revisions; the operator correction supersedes concurrent annotation/driving and all keyboard-idle delays. One atomic delivery and one close review; the operator removed M2 before implementation. The Chrome menu is deferred until #495 then #506 land and the hub releases it.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| OperatorRequest / OperatorReply | `cmd/internal/webcast/board/operator.go` | new |
| OperatorView | `cmd/internal/webcast/board/operator_view.go` | new |
| annotationPolicy | `cmd/internal/webcast/web/annotation-policy.js` | new |
| suppressionPolicy | `cmd/internal/webcast/operator-policy.js` | new |
| recorded node path helper | `cmd/internal/webcast/recorder/observer.js` | modified |

- OperatorRequest is a bounded tagged union: prepare-note, save-note, stroke, resolve,
  cancel-draft. Broadcast/driving authority stays in Couch. Transport
  identity (session/context/document) comes from CDP, never a claimed author string.
  Parsing reuses Board shape predicates. Operator authorship is assigned in Go;
  HTTP parsers keep rejecting `operator` (ARCH-SECURE, ARCH-DRY).
- OperatorView projects the Board's current document, segment, pause/end, open notes
  and bounded recent strokes. It does not copy the DOM or assets. A monotonically
  increasing view revision and per-chunk sequence avoid duplicate stroke painting.
  It derives from Board's reducer; this is not another recording reducer.
- annotationPolicy is the shared pure mode/gesture decision function used by the
  viewer mount and operator adapter. Operator availability adds driving and cast
  readiness; viewer annotation rules do not gain the operator's driving restriction.
- suppressionPolicy owns document-local token acquisition/release. Its result says
  whether Couch UI is visible/hittable. A release removes only its own token.
  Capture and confirmed remote-dispatch tokens compose; unknown input promotes its
  guard to target lifetime, inherited by replacement documents until target teardown.
- The observer's existing path function becomes callable inside the same isolated
  world. The operator adapter uses that function; it does not invent selectors or
  another child-index grammar. Recorder nodeAt resolves it in event order.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Board operator entry points / view subscription | `cmd/internal/webcast/board/operator*.go` | new | validated commands, reducer projection and existing Emit |
| Recorder operator binding adapter | `cmd/internal/webcast/recorder/operator.go` | new | ordered CDP bindings and mirror identities |
| Operator bridge | `cmd/internal/webcast/chrome_operator.go` | new | bounded view delivery/replies to the active isolated world |
| Shared annotation mount | `cmd/internal/webcast/web/annotate.js` | modified | viewer/operator DOM and transport adapters |
| Operator injection / script assembly | `cmd/internal/webcast/operator.js`, `operator_script.go` | new | excluded shadow host and trusted embedded JS |
| Capture/input suppression | `cmd/internal/webcast/recorder/ui_capture.go` | new | exact-context Runtime.evaluate and Page.captureScreenshot |
| Existing Chrome setup and Cast | `cmd/internal/webcast/chrome.go`, `cast.go`, `inject.js` | modified | installation, ownership and shutdown |
| Couch annotation entry point | `cmd/internal/couchtty/console_webcast.go` and menu owner files after main merge | modified | shared operator mode command |

Board lives for one Cast. RecordOptions supplies its operator seam when recording is
configured; no recorder means disabled annotation with a reason. Recorder resolves
bindings on its queue; Board's Emit re-enqueues without waiting and is safe from
reentrancy. Never call a blocking Recorder.ask from that queue. Chrome owns one
bounded delivery worker and cancels/joins it at finish. Reuse recorder's scripted
Caller/peer, the real Board with its stateful reference Writer, FakeDriven and the
real temporary-profile headless Chrome as the external conformance check (ARCH-MOCK).

## Behavior and ordering

| Current state / event | Next state and effects |
|---|---|
| new document or new active tab | Off; cancel old gestures/draft handles/replies; obtain the current projection |
| live, no driver / select Pen or Note | selected mode, shared gesture engine enabled |
| any mode / Off or Escape | Off; cancel unfinished gesture; close editor on explicit Escape |
| active gesture / pause | Off; discard unsent points, release capture; retain accepted chunks |
| editor / pause | hold draft; no Save/Resolve; resume only on same document |
| any annotation / driver appears | Off; release pointer capture, hide/unfocus editor and pins; keep draft; show driving hint |
| driving / driver leaves | controls available; Off; offer held draft again without stealing focus |
| draft / scroll | keep original page box/node; move its visible pin relative to viewport |
| draft / Save accepted, reply lost | existing operation remains pending/accepted; reconcile same operation ID; do not send a second note |
| request / stale context, navigation or ended cast | refuse or ignore old reply; never assign to the current document |
| capture or driver dispatch / acquire token | hide all Couch UI, confirm exact-context suppression before the external operation |
| any suppression / release one token | restore only if no tokens remain; never target a replacement context |

The pure policy function owns state transitions; DOM handlers execute its effects.
Tests cover every row above; no new browser driving-control transition is introduced.
No two-keyboard arbitration, typing wait, or editor-caused delay to remote input.
The remaining operator authority buttons must not receive CDP-driven clicks.

### Request, acknowledgment and identity

1. Chrome accepts only the binding installed in its known couchcast top-frame context
   for the active recorded tab. Unknown/stale contexts and oversized/unknown payloads
   are rejected before any Board call. No URLs, cookies or viewer secrets enter the page.
2. A note gesture captures page-space region, the actual element's box and its path
   via the observer helper. prepare-note resolves once on the recorder queue, checking
   the reporting context's current document. Keep one prepared draft per context.
   Reply with an opaque draft handle bound to node + document + region + box.
3. Save names that handle and a request ID. Board validates color/text and the current
   live/non-driving operator predicate, then uses its existing Note/Emit path. Save
   never re-resolves the path. Resolve and stroke use the same operator admission.
4. Board's OperatorOperation owns one outstanding mutating note operation per context
   and its result until the UI acknowledges it or that context dies. Its pure
   nextOperatorOperation transition enumerates pending, accepted, refused and unknown.
   Add a completion seam to recorder.Emit: a nonblocking settlement callback runs
   exactly once after ordering-time admission and the writer's sink result. Enqueueing
   yields pending, never accepted. OnDoc/driver refusal settles refused and releases
   Board's pending note reservation; successful sink delivery settles accepted. A sink
   error after a possibly partial write settles unknown, disables resubmission, reports
   the recording fault and reconciles by the operation's stable note ID against the
   writer/Board's observed note (never by issuing a fresh write). If evidence remains
   inconclusive, return a visible unknown result requiring cast recovery; do not leave
   an indefinitely spinning pending operation. Chrome shutdown settles pending callers
   with an ended/unknown outcome according to whether ordering began. The bridge owns
   a five-second response budget, which reports pending/unknown but never retries a
   write under a new ID. The recorder owns actual settlement; a late completion may
   update the retained same-ID outcome but cannot revive old-document UI.
   Reservation removal is idempotent and keyed to the reserved note ID, not color.
   Both operator Note and NoteState use this completion seam; stroke chunks need no
   retry but ordering-time refusal must still surface as a visible failure. Repeating its ID returns its prior result;
   an unknown reply leaves the UI pending and reconciliation uses the same ID. A
   later note requires a new ID. No automatic duplicate save after timeout. A note
   accepted on an old document stays in that document's recording; its UI reply is
   ignored after navigation. The recorder rechecks live/document/driving at ordering
   of operator writes, closing admission-versus-pause/grant races.
5. Board view updates contain only the current open notes (existing per-color limit),
   plus up to 128 recent stroke chunks, at most 64 points per chunk. Prune by the
   shared fade duration, record time and capacity. Use a latest-value subscriber slot;
   per-chunk sequence numbers let the adapter apply a snapshot without duplicating
   chunks. A gap drops transient stroke history but refreshes all current note state.

A top-page note over an inaccessible cross-origin frame anchors to the recorded
iframe element, preserving the box. Open shadow trees use the existing path grammar;
closed page shadow interiors use their host. No guess at an inaccessible child.

### Shared UI and packaging

Refactor mountAnnotate into a small viewer bootstrap plus a shared mount accepting:
root/document/window, palette/identity, transport, projection subscription, coordinate
mapping, and an async prepareDraft adapter. Existing viewer nodeAt/boxOf is adapted to
that asynchronous interface without changing visible behavior. Injected identity is
constant; it uses no page localStorage. Off-mode annotation surfaces and pins do not
consume page pointer input; controls remain explicit targets. In Pen/Note, gestures
and pins work; while driving no annotation hit targets or editor focus remain.

Reuse collab.js's swatches/dialog/gestureBox and annotate.js's chunker, pinLayer and
stroke painting. Allow sayer/dialog mount targets to be inside the excluded shadow
root, never document.body. Reuse annotation CSS under that root. Test all supported
consumers: terminal viewer, live web viewer, replay, operator injection.

Compose trusted embedded collab.js, annotation-policy.js, annotate.js and operator.js
into closed module scopes for classic isolated-world injection. A narrow assembler
recognizes only these sources' explicit named imports and exported declarations,
resolves imports from an allowlisted module table, and rejects unsupported module
syntax. It is not a general bundler. Unit tests compile/evaluate the exact assembled
bytes with Node and compare exported helper behavior with the ES modules; every
import must resolve. No runtime network imports or new build tool dependency.

### Suppression and screenshots

The isolated control owner registers all Couch UI under excluded hosts, and implements
acquire(token, reason) / release(token) with per-document ownership. Capture uses
opacity/visibility that actually removes pixels; input suppression removes nodes
from hit testing too (`ignorePointerEventsNone:true` makes pointer-events alone
insufficient). Focus is already outside the held editor while driving. Hide all
operator buttons around the existing recorder driver hit-test/dispatch bracket.

The capture helper captures the original session/context/document, acquires and
confirms a token, performs Page.captureScreenshot, and releases in defer against the
same context. A failed/unconfirmed acquire skips the frame and reports the reason;
never save potentially contaminated bytes. Capture release failure leaves a bounded expiry owned by the original document;
navigation destroys that capture owner. Tokens compose with remote dispatch and other
captures. Capture expiry is longer than its CDP call deadline; if capture times out or
expires, do not publish its bytes. Input tokens do not use this expiry: an input effect
has no discard operation, and the rule below owns its uncertain lifetime. The screenshot worker carries
this identity with shotJob; its late result is checked again before PutFrame.

Use this helper for every production cast-page capture. Current source inspection
finds recorder/side.go's fidelity capture; submission #483 currently consumes stored
frames, and any direct capture landed before this issue closes must use this helper.
Replay/fidelity-renderer captures of a separate replay browser do not contain injected
operator UI and are not routed through cast suppression.

## Operating envelope and trust

- **ARCH-CONSTRAINTS:** one cast, one active operator context, ordinary human gestures,
  10–20 viewers per project. Preserve 50 ms stroke chunk cadence and 64-point cap;
  no render loop while no stroke needs fading. Notes retain existing text/color limits.
  One view-delivery worker, latest-value state slot and bounded transient chunks; no
  worker per event. Board writes and CDP payload parsing keep existing size budgets.
  Excess input receives a visible busy/refusal; no unbounded queue. Idle overlay must
  not add periodic DOM scans. Targeted tests run sequentially with `-p 1`; no full
  suites or foreground Chrome without hub coordination.
- **ARCH-PURE:** policy/projection/decoding are pure and tested without browser mocks;
  CDP, clocks, event listeners and worker lifetimes are injected boundary effects.
- **ARCH-DRY:** Board is validation/anchor authority; shared UI owns gesture geometry;
  observer owns path grammar; Controller owns drive decisions; one capture helper.
- **ARCH-PURPOSE:** M1 proves operator gestures reach live viewers, recording, replay
  and query. The delayed Couch entry point uses the existing menu after #506 lands.
- **ARCH-SECURE:** page scripts may remove a shared host, but cannot invoke isolated
  binding state. Treat all binding payloads as bounded untrusted values. Note/name
  text uses textContent. No operator identity on HTTP and no operator control via
  synthetic remote input. Tests use temporary profiles, local fixture servers/files.
- **ARCH-ORDER:** exact context/document/draft/request identities and production pure
  policy own the transitions above. Inject ordering failures in tests rather than
  relying on timing luck. Late view/reply delivery cannot re-enable an old document.
- **ARCH-FUNERAL:** creates no new persistent family: existing notes/strokes use the
  recording's retention policy. Draft/results/token sets die with their context;
  temporary profiles with test cleanup; worker and subscriptions with Cast/Chrome.
  Stroke cache bounded to 128 chunks and fade lifetime; draft operation state is one
  per context; capture tokens have deadlines; unknown-input suppression ends only when its CDP target is destroyed (see below).
- **ARCH-MOCK:** stateful Board Writer, recorder Caller/peer and FakeDriven cover
  event ordering, context replacement, hide failure and shutdown. Headless conformance
  runs at the single delivery boundary and after changes to the relevant external assumptions.

## Annotation and recording

### Task 1: operator protocol and Board authority

Files: create `board/operator.go`, `board/operator_view.go` and colocated tests;
modify `board/wire.go`, `board/board.go` to share validation/projection hooks.

- [x] Write failing tests for ParseOperatorRequest, nextOperatorOperation and
      projectOperatorView using the strategies below; run
      `go test -p 1 ./cmd/internal/webcast/board -run 'Operator' -count=1` for RED.
- [x] Implement operator-only entry points, shared validation, operation settlement
      and bounded projection; retain the HTTP parser's reserved-author rule.
- [x] Run the Board package and new tests with `-race`; commit `#494 M1: ...`.

### Task 2: ordered recorder bridge and capture suppression

Files: create `recorder/operator.go`, `recorder/ui_capture.go` and tests;
modify `recorder/observer.js`, `recorder/recorder.go`, `recorder/drive.go`,
`recorder/side.go`, `chrome.go`, `cast.go`; create `chrome_operator.go`.

- [x] Write failing stateful tests for handleOperator, settleOperatorWrite,
      captureWithoutUI and withInputSuppressed. Run
      `go test -p 1 ./cmd/internal/webcast/recorder ./cmd/internal/webcast -run 'Operator|Suppression' -count=1` for RED.
- [x] Implement ordered binding resolution, completion callbacks, bounded delivery
      and capture helper with the confirmed/unknown effect rules below.
- [x] Run affected recorder/Chrome lifecycle and race tests; commit the bridge.

### Task 3: shared annotation mount and injected operator adapter

Files: modify `web/annotate.js`, `web/annotate.css`,
`cmd/internal/broadcast/web/collab.js`, `inject.js`; create `web/annotation-policy.js`,
`operator-policy.js`, `operator.js`, `operator_script.go` and tests. Update embedded
asset tables and `cmd/internal/artifactpath/manifest.go` for new production Go files.

- [x] Write failing tests for annotationPolicy, suppressionPolicy, mountAnnotation
      and assembleOperatorScript. Run
      `node --test tests/webcast/annotate.test.mjs tests/webcast/operator.test.mjs` for RED.
- [x] Implement shared mount and operator adapter; keep terminal/live/replay/operator
      consumers on the shared helpers and serve every imported module.
- [x] Run Node/module closure tests; update README/atlas and commit the shared UI.

### Task 4: headless proof

Files: create `cmd/internal/webcast/live_operator_test.go`; extend existing
`cast_board_test.go`, recorder mirror/capture tests and `tests/webcast` cases.

- [x] Implement TestLiveOperator using the acceptance strategy below. Run
      `WEBCAST_LIVE_CHROME=1 go test -p 1 ./cmd/internal/webcast -run '^TestLiveOperator' -count=1 -v`.
      Inspect a saved headless UI screenshot; no foreground activation.
- [x] Run relevant Node tests, affected Go packages with `-p 1`, artifactpath inventory
      check and `git diff --check`; mutation-check critical rejection/suppression wiring.
- [ ] Record evidence in issue Log, update README/atlas and required project references
      through the gate's authorized mechanism (do not hand-edit ops).
### Task 5: deferred Couch menu, smoke, close and land

- [x] Wait for hub confirmation that #506 has landed, then merge fresh main before
      touching menu files. Read #495/#506's shared menu model and add annotation rows
      using it: left acts, right copies, Alt menu only LIVE/Chrome. Targeted menu tests.
- [ ] Arrange operator smoke through hub; stage/build in pair:0 only on explicit
      instruction. Fresh session for the binary; preserve selected HEAD.
- [ ] Record operator acceptance; run affected checks after the last change and
      `sdlc close --issue 494 --verified '<actual recorded evidence>'`. This runs the
      one mandatory boundary review; fix Important/Critical findings. Actuals are
      measured by the binary. Do not fabricate smoke or bypass checks.
- [ ] Merge fresh origin/main, resolve conflicts, rerun affected checks, `sdlc pr`.
      On SHIP run `sdlc merge --yes` (never --no-validate), report PR to hub and stop.

## Function-level verification strategies

| Function | Adversarial class and mechanical guard |
|---|---|
| ParseOperatorRequest | Fuzz closed tagged JSON over arbitrary bytes/bounds; malformed input cannot produce an admitted typed command or claimed author. |
| nextOperatorOperation / settleOperatorWrite | Model event sequences with delayed/lost completion and sink failure; assert at most one write per operation and accepted iff writer evidence confirms it; no leaked reservation after refusal. |
| projectOperatorView | Generate document/control histories and burst strokes; projection always matches Board's authoritative open-note set and stays within its chunk/point bounds. |
| handleOperator | Script CDP context/path replacement order; exact provenance must resolve the original recorded node or refuse without any write. |
| annotationPolicy | Sequence-test production transitions with interruptions/late replies; unavailable state has no annotation effect, preserved drafts never cross documents. |
| suppressionPolicy | Permute acquisitions, releases and deadlines; no UI visible while any live guard exists, no old guard changes a replacement context. |
| captureWithoutUI | Stateful Caller delays/fails each phase and changes documents; FrameWriter receives bytes only after confirmed suppression and matching capture identity. |
| withInputSuppressed | Stateful Caller returns dialog/timeout before delayed dispatch completion; authority controls never return while the effect is unknown. |
| assembleOperatorScript | Parse/evaluate exact assembled embedded bytes and import closure; unsupported syntax fails construction, shared exported functions match the ESM consumers. |
| mountAnnotation | Run shared mount with controlled DOM/event source; handlers execute policy effects and never create UI outside the supplied excluded root. |
| TestLiveOperator | Real headless user gestures through Cast and a viewer, then disk/replay/query plus screenshots; independently assert operator authorship, page fidelity, exclusion and off/held-draft input behavior across lifecycle changes. |

## Unknown remote-input outcomes (PQ-2)

withInputSuppressed owns the real input call's completion, not merely inputCall's
caller return. Add a completion result that distinguishes a CDP reply from caller
cancellation/dialog wakeup. Confirmed replies permit release (including a confirmed
CDP refusal); a timeout, lost connection, or early dialog return does not prove the
command stopped. Promote such a token to a target-scoped quarantine immediately:
all Couch controls in that CDP target remain hidden/non-interactive, and replacement
contexts inherit the guard before their setup is released. Do not expire this guard.
Stop further remote input to that target, report the uncertainty through Couch, and
instruct closing/reopening the cast Chrome. The ordinary Couch revoke/end controls
remain available. Target/process teardown is the bounded escalation and definitive
end of any late command; no background polling or automatic restart/focus stealing.
If the original pending call still produces a positive completion before teardown,
its exact operation can release its guard; an abandoned response cannot be fabricated
by a later ping. Capture tokens retain their separate safe-to-discard expiry rule.

## Non-goals

No simultaneous operator annotation and remote driving, no editor typing-delay queue,
no extension/SDK, no new recording format, no page-origin network or credentials,
no browser shield or new Stop/codes/Grant/Deny/Take-back controls,
no new replay interaction mode, no restyling of unrelated cast controls, and no
menu edits until the hub releases #506. Existing recording retention remains intact.

## Revisions

- 2026-10-12: Initial plan uses the operator's corrected driving rule. The earlier
  simultaneous-typing proposal is intentionally absent. M1 remains independent of
  the delayed menu and M2 controls; no implementation has started.

- 2026-10-12 (PQ-1..3): Added authoritative async write settlement and reservation
  cleanup; separated discardable capture expiry from unknown remote-input lifetime;
  compressed test inventories into named-function adversarial strategies. These
  refine implementation contracts without changing the approved operator experience.

- 2026-10-12: Operator removed M2 entirely (TL receipt 872f2784). Trimmed active
  mappings/tasks/acceptance to annotation only, preserving deferred Couch menu rows.
  The transient shield/Chrome authority request (fd133ced) was never implemented.
  One atomic delivery now closes with one review, full-flow retained. The approved
  annotation design is unchanged; no milestone-close or M2 code is to be produced.

### 2026-10-12 — implementation checkpoint and capture integration order

The annotation-only tasks 1–3 and headless proof are implemented. Actual entry
points are OperatorSession.run/Status, Recorder.handleOperator/writeOperator,
recorder.CaptureScreenshot, Recorder.withTab and TestOperatorLiveAnnotation in
operator_live_test.go. A same-loader/document tab return may acquire a fresh
recording DocSeg; old draft handles and old-segment requests remain invalid.
Headless screenshot and pixel witness inspected; missing-confirmation mutation
was caught and restored. Test evidence is recorded in the issue.

Per TL receipt 057ac00b, #494 owns CaptureScreenshot and whichever issue lands
second wires #483 submission capture through it. #506 still owns the Chrome menu;
no menu edits or pair:0 staging have occurred. Remaining steps: menu release/main
merge, submission integration if #483 precedes us, operator smoke, single close
review and SHIP-authorized merge. Project progress goes through hub/gate, not manual
ops edits.

### 2026-10-12 — operator rail matches viewer icons

Per TL receipt 6804f5e6, reuse the viewer's ✏️/💭 icon buttons and existing
tooltips exactly. Off is represented by neither icon selected; pressing an active
icon or Escape returns Off. Removed operator-only word labels and sizing override.
The annotation states and deferred Couch menu scope are unchanged.

### 2026-10-12 — annotation capability input for #523

TL receipt 39206f41 requests one overlay input for the upcoming 👆 capability
switch. Added isolated-world `__couchcastAnnotationAllowed(boolean)` over shared
`setAnnotationAllowed`, default true. False cancels gestures, forces Off, disables
annotation actions and hides/holds a draft; true restores availability with the
pen Off, subject to driving/pause. Projection updates preserve the input. #523
owns the switch, feeding this input on replacement documents and server admission;
none of those are built here. Targeted policy/mount tests demonstrated RED→GREEN.

### 2026-10-12 — menu released and capture integration complete

Merged #483 and #506 after hub release. Submission crops now use CaptureScreenshot;
the real headless pixel test proves exclusion and restoration through that call.
Shared viewer mount retains #506’s view-only flag. Chrome’s existing menu now gains
`Annotation · Off · Pen · Note` through its same action layout, preserving all status
click rules and avoiding Focus. Targeted tests and real Chrome checks passed as logged
in the issue; the machine-gated full suite and operator smoke precede close.

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
