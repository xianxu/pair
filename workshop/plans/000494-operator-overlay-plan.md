# Operator annotation in the cast Chrome implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent tasks, or superpowers-executing-plans for session-warm integration. Steps use checkboxes for tracking.

**Goal:** Let the operator annotate the actual cast Chrome and manage driving there, with annotations recorded through the existing Board and all Couch UI excluded from page captures.

**Architecture:** Reuse the existing isolated world and excluded control host. A bounded operator binding adapter connects shared annotation UI to Board through the recorder's document identity; the existing recorder remains the only record writer. A document-local suppression owner protects captures and remote input. Annotation is unavailable while a viewer drives.

**Tech stack:** Go, CDP, embedded JavaScript/CSS, existing Node tests and headless Chrome harnesses. No new service, browser extension or external JS dependency.

**Approved scope:** Issue #494 Revisions; the operator correction supersedes concurrent annotation/driving and all keyboard-idle delays. M1 and M2 are separate review boundaries. The Chrome menu is deferred until #495 then #506 land and the hub releases it.

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
  cancel-draft; M2 adds grant/deny/revoke through its own control adapter. Transport
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
  Capture and remote-dispatch tokens compose; old-document replies affect nothing.
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
| Browser driving controls | `cmd/internal/webcast/chrome_operator_drive.go` | new in M2 | existing drive.Controller |
| Couch annotation entry point | `cmd/internal/couchtty/console_webcast.go` and menu owner files after main merge | modified in M2 | shared operator mode command |

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
| M2 pending request / button press | verify request generation and viewer ID at Controller boundary before applying |

The pure policy function owns state transitions; DOM handlers execute its effects.
M1 tests cover every row applicable to M1. M2 tests add request/driver transitions.
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
4. Keep one outstanding mutating note operation per context and its result until the
   UI acknowledges it or that context dies. Repeating its ID returns its prior result;
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
never save potentially contaminated bytes. Release failure leaves a bounded expiry
owned by the original document; navigation destroys that owner. Tokens compose with
remote dispatch and other captures. Expiry is longer than the CDP call deadline; if
capture times out or expires, do not publish its bytes. The screenshot worker carries
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
  and query. M2 delivers the browser authority controls and delayed Couch entry point.
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
  per context; suppression tokens have deadlines and end on navigation/shutdown.
- **ARCH-MOCK:** stateful Board Writer, recorder Caller/peer and FakeDriven cover
  event ordering, context replacement, hide failure and shutdown. Headless conformance
  runs once per milestone and after changes to the relevant external assumptions.

## M1 — annotation and recording

### Task 1: operator protocol and Board authority

Files: create `board/operator.go`, `board/operator_view.go` and colocated tests;
modify `board/wire.go`, `board/board.go` only to share validation/projection hooks.

- [ ] Write table/sequence tests for reserved author, malformed request, unsafe color,
      exact text/point bounds, live/paused/driving, duplicate Save, bounded view cache,
      same-document resume, navigation and closed Board. Assert no emission on refusal.
- [ ] Run `go test -p 1 ./cmd/internal/webcast/board -run 'Operator' -count=1` and confirm RED.
- [ ] Implement typed parsing and operator-only entry points reusing existing validators;
      preserve HTTP rejection tests. Implement projection and one pending operation.
- [ ] Run the entire Board package, then new tests with `-race`; commit `#494 M1: ...`.

### Task 2: ordered recorder bridge and capture suppression

Files: create `recorder/operator.go`, `recorder/ui_capture.go` and tests;
modify `recorder/observer.js`, `recorder/recorder.go`, `recorder/drive.go`,
`recorder/side.go`, `chrome.go`, `cast.go`; create `chrome_operator.go`.

- [ ] First add scripted CDP/Writer tests: old context cannot prepare/save; path reused
      after prepare does not retarget; pause/grant between admission and emit drops it;
      Save reply lost does not duplicate; shutdown joins delivery and clears subscribers.
- [ ] Add suppression overlap tests with two captures and remote dispatch in both
      completion orders; failed hide skips capture; failed restore expires; context
      replacement cannot be modified by an old release/result. Assert no contaminated
      frame reaches FrameWriter and old-document accepted notes remain recorded.
- [ ] Run `go test -p 1 ./cmd/internal/webcast/recorder ./cmd/internal/webcast -run 'Operator|Suppression' -count=1` for RED.
- [ ] Implement exact-context routing, bounded bridge and shared capture helper. Keep
      driver operations behind suppression confirmation and normal drive admission.
- [ ] Run targeted recorder/Chrome lifecycle tests and race tests; commit the bridge.

### Task 3: shared annotation mount and injected operator adapter

Files: modify `web/annotate.js`, `web/annotate.css`,
`cmd/internal/broadcast/web/collab.js`, `inject.js`; create `web/annotation-policy.js`,
`operator-policy.js`, `operator.js`, `operator_script.go` and tests. Update embedded
asset tables and `cmd/internal/artifactpath/manifest.go` for new production Go files.

- [ ] Add Node policy tests for the state table, held draft, ignored old replies,
      scrolling, Off-mode hit targets, driver start during a stroke, duplicate view
      snapshots and nested suppression tokens. Add shared-mount fake DOM tests.
- [ ] Run `node --test tests/webcast/annotate.test.mjs tests/webcast/operator.test.mjs`
      for RED; implement the shared mount and operator adapter to turn GREEN.
- [ ] Make module assembly fail on missing/unsupported imports; test exact injected
      bytes, then existing live/replay/terminal module closure tests.
- [ ] Verify UI handling uses shadow-root mount targets and all its nodes are excluded;
      update README and atlas/webcast.md plus index as needed; commit shared UI.

### Task 4: headless proof and M1 boundary

Files: create `cmd/internal/webcast/live_operator_test.go`; extend existing
`cast_board_test.go`, recorder mirror/capture tests and `tests/webcast` cases.

- [ ] Add one temporary-profile real-Chrome scenario with a second headless viewer:
      off-mode page click/input; Pen stroke; Note drag and default-box click; Operator
      attribution live and on disk before stopping; replay/query output; no injected
      DOM/style/editor text in events; clean page screenshot pixels with UI visible.
- [ ] Exercise scroll, zoom/DPR, open shadow and iframe-host anchoring; pause/resume;
      navigation with draft/reply in flight; start driving with editor open, preserved
      draft after release, remote mouse/wheel/keys reach page and never operator UI.
- [ ] Run `WEBCAST_LIVE_CHROME=1 go test -p 1 ./cmd/internal/webcast -run '^TestLiveOperator' -count=1 -v`.
      Inspect a saved headless UI screenshot for layout; no foreground activation.
- [ ] Run relevant Node tests, affected Go packages with `-p 1`, artifactpath inventory
      check, `git diff --check`; mutation-check the key rejection/suppression wiring.
- [ ] Record evidence in the issue Log, update README/atlas and required project
      references through the gate's authorized mechanism (do not hand-edit ops).
- [ ] Commit and run `sdlc milestone-close --issue 494 --milestone M1` with required
      evidence per help. Fix Important/Critical findings; this is the one boundary
      review. Report M1 ready to hub so M2 does not delay annotation smoke readiness.

## M2 — browser driving authority and deferred Couch entry point

### Task 5: Grant / Deny / Take back

Files: create `chrome_operator_drive.go` and tests; modify `cast_drive.go`,
`chrome_operator.go`, `operator.js`, `inject.js`, `drive/controller.go` and its memory
implementation only as needed for atomic stale-request checks.

- [ ] Write tests where a request is withdrawn/recreated for the same viewer before
      a click applies: the old button must not grant the new request. Test stale
      driver state, pause, denied/granted result, cast end and synthetic remote click.
- [ ] Extend Controller with an atomic expected-state operation if its existing API
      cannot express that check; do not check then call Grant outside its lock.
- [ ] Subscribe to Controller state and project request buttons/driver name. Route
      isolated commands through bounded exact-context bridge to Controller; show
      fixed errors. Revoke addresses the visible generation; no optimistic success.
- [ ] Run Controller and Chrome targeted/race tests; headless Grant/Deny/Take-back
      scenario while annotations are disabled; update README/atlas and commit.

### Task 6: integrate latest menu, smoke, close and land

- [ ] Wait for hub confirmation that #506 has landed, then merge fresh main before
      touching menu files. Read #495/#506's actual shared menu model and append
      annotation rows using it. Left acts, right copies link, Alt menu only LIVE/Chrome.
      Do not create a competing menu. Targeted tests prove the existing link/code rows.
- [ ] Arrange operator smoke through hub; stage/build in pair:0 only on explicit
      instruction. Fresh session needed for a new binary. Preserve selected HEAD.
- [ ] Record operator acceptance; run M2 checks after last changes and invoke
      `sdlc milestone-close --issue 494 --milestone M2` with evidence. Fix findings.
- [ ] Run `sdlc close --issue 494 --verified '<actual recorded evidence>'`; measured
      actuals are adopted by the binary. Do not fabricate smoke or bypass checks.
- [ ] Merge fresh origin/main, resolve conflicts, rerun affected checks, `sdlc pr`.
      On SHIP run `sdlc merge --yes` (never --no-validate), report PR to hub and stop.

## Revisions

- 2026-10-12: Initial plan uses the operator's corrected driving rule. The earlier
  simultaneous-typing proposal is intentionally absent. M1 remains independent of
  the delayed menu and M2 controls; no implementation has started.
