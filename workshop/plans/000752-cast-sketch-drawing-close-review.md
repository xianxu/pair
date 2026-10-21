# Boundary Review — pair#752 (whole-issue close)

| field | value |
|-------|-------|
| issue | 752 — cast-sketch: live console and driver drawing with pen pressure |
| repo | pair |
| issue file | workshop/issues/000752-cast-sketch-drawing.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..60a400cb13be0e54e0469e36ce5d3805461cc07c |
| command | sdlc close --issue 752 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:55:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation delivers both drawing surfaces, strict ingress, ownership fencing, bounded delivery, and README/atlas documentation. Existing automated tests pass, but two reproduced UI failures block acceptance: interrupted console strokes can become impossible to label, and queue saturation can strand an open stroke or crash pointer-up. The plan’s entity map also needs correction. Live Chrome conformance could not run successfully in this environment.

```findings
findings:
  - id: new
    severity: Critical
    family: authoritative-state-reaches-ui
    title: |
      Console handoff leaves partial-stroke label saving permanently blocked
    detail: |
      cmd/internal/webcast/web/sketch.js:470 refuses label saving while cached status.open_stroke is populated, but its lifecycle refresh at :663 requires model.sketchMetadata and actual ctl/stop/sketch-interrupt records. The console model in operator.js:173 has no such query, publishes synthetic view records, and its sketch journal receiver only refreshes painting. Reproduced an accepted open stroke followed by handoff and authoritative sealing: labelRequests remained zero with “Wait for the stroke confirmation before saving its label.” ARCH-ORDER/ARCH-PURPOSE: feed authoritative sketch metadata and lifecycle transitions into the console tool, including handoff, pause, expiry, clear and end; add an interrupted-contact console regression.
  - id: new
    severity: Critical
    family: bounded-input-retains-terminal-work
    title: |
      Queue saturation cancels the gesture without sealing it and can crash pointer-up
    detail: |
      cmd/internal/webcast/web/sketch-policy.js:207 emits only stop-gesture when sixteen point chunks are queued; sketch.js handles that by canceling the gesture and clearing contact without retaining terminal work or entering visible recovery. Holding the first points reply and filling the queue reproduced a drained queue with the server stroke still open, zero stroke-end requests, and no status message; when saturation occurs inside finish() at sketch.js:322, line 324 throws “Cannot read properties of null (reading 'pointer')”. ARCH-CONSTRAINTS/ARCH-ORDER: preserve a truncated terminal operation behind accepted queued work, expose reconciliation, and make finish safe against synchronous cancellation.
  - id: new
    severity: Critical
    family: core-concept-map-matches-code
    title: |
      The Core concepts table names nonexistent and unchanged implementation locations
    detail: |
      workshop/plans/000752-cast-sketch-drawing-plan.md:50 names board/sketch_authority.go, which does not exist; SketchAuthority is in board/sketch.go:14. Rows 52–58 also claim modifications to recording/hub.go, browser.go, server_drive.go, web/annotation-placement.js, recorder/operator.go, board/operator.go, fake.go and fake_drive.go that are absent from the pinned diff. Per the required Core concepts cross-check, reconcile the complete table with actual files and statuses, including sketch_source.go, cast_sketch.go and writer_sketch.go, and record the correction under Revisions.
  - id: new
    severity: Important
    kind: unverifiable
    family: live-conformance-evidence
    title: |
      Required Chrome conformance assertions could not execute
    detail: |
      Attempted WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'SketchLive|OperatorSketchLive' -count=1 -v. Both operator tests failed before opening their page with GPU-process/sandbox initialization failures, and TestSketchLiveTwoViewers failed with CDP EOF. Supply successful host results for these tests on the reviewed implementation, or provide an environment where Chrome starts, to verify private-scene isolation and real two-viewer/console behavior.
```

1. **Strengths**

   - Driving permits fence stale turns before control settlement; held-write, release-failure and ABA tests exercise this behavior.
   - Source preparation captures an exact ordered prefix and refuses independent prefix/state limits before committing begin.
   - Additive truncation has codec, semantic-identity, snapshot, cancellation and legacy-preservation coverage.
   - README and atlas explain controls, recovery, limits and private-scene architecture.

2. **Critical findings**

   The three Critical findings above require two behavioral fixes with regression tests and a correction to the plan’s implementation map.

3. **Important findings**

   Live Chrome conformance needs executable host evidence. The separately documented TL console smoke remains pending before landing; this review does not treat it as completed.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/strictjson ./cmd/internal/webcast/...`
   - All 388 Node webcast tests.
   - Race-enabled sketch/drawing tests for Board, drive and recorder.
   - `git diff --check`.

   Read-only Node probes using the existing harness reproduced both behavioral findings. Current handoff coverage does not exercise a console contact interrupted while its stroke remains open; queue tests do not cover saturation during pointer-up.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared tool, renderer, protocol and writer seams are reused. |
   | ARCH-PURE | Pass: parser, policy and projection logic are separated from transport and DOM effects. |
   | ARCH-PURPOSE | Flag: interrupted console labeling fails a required workflow. |
   | ARCH-MOCK | Pass structurally: stateful writer/controller doubles share production seams; live conformance remains unverifiable here. |
   | ARCH-CONSTRAINTS | Flag: queue capacity is enforced without safe terminal recovery. |
   | ARCH-SECURE | Isolation structure inspected; actual Chrome privacy assertion remains unverifiable here. |
   | ARCH-ORDER | Flag: authoritative sealing does not reach console policy/status, and cancellation invalidates state during `finish()`. |
   | ARCH-FUNERAL | Pass: source leases, transfer handles, blobs and workers have retirement owners. |
   | ARCH-EVOLVE | Pass: optional truncation preserves absent/false semantics and unknown-field round trips. |

7. **Plan revision recommendations**

   Add `## Revisions` entries that:

   - Correct every Core concepts path/status against the pinned implementation.
   - Describe authoritative console metadata delivery and its interrupted-contact regression.
   - Specify queue-full terminal sealing and pointer-up recovery tests.
   - Record successful host conformance evidence while retaining the pre-landing console-smoke hold.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"49c4f4f9-2fc4-41fe-9ee7-c8d91eed8d50","repository":"github.com/xianxu/pair","issue_id":"000752","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c1af59ffccdebf8f","reviewed_head":"60a400cb13be0e54e0469e36ce5d3805461cc07c","slot":"pair:9","requested_at":"2026-10-21T09:50:14.707355Z"},"phase":"finished","pid":59176,"started_at":"2026-10-21T09:50:14.802301Z","finished_at":"2026-10-21T09:55:06.46759Z","session_id":"01a15dde-581e-7a91-ad27-a35d3046dd27","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.920169125s
head: "60a400cb13be0e54e0469e36ce5d3805461cc07c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T03:53:20-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 752 — cast-sketch: live console and driver drawing with pen pressure |
| repo | pair |
| issue file | workshop/issues/000752-cast-sketch-drawing.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..e9ea2bbf3ed082d870a2f67af932ce78dff28d93 |
| command | sdlc close --issue 752 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:53:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The prior console lifecycle and queue-saturation fixes have meaningful red/green regression evidence. File mappings, test-strategy prose, and documentation are improved. Two blockers remain: stroke width contradicts the approved contract, and the Core concepts table still classifies effectful components as PURE. Chrome cannot start in this environment, but the supplied successful host evidence covers unchanged executable code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Inspected the before/after plan diff: Tasks 1–5 now reference the Function verification strategy table while retaining commands and independent invariants. The correction is recorded under Revisions.
  - id: BR-2
    disposition: addressed
    note: |
      Console delivery now supplies authoritative metadata to shared reconciliation. Lifecycle, stale-status, draft-retention and held-status tests pass. Reverting the browser fixes in a disposable copy makes the interrupted-contact label regression fail.
  - id: BR-3
    disposition: addressed
    note: |
      Saturation reserves one truncated terminal and enters visible reconciliation; finish detaches contact before synchronous effects. Move, pointer-up, unknown and refused-operation regressions pass. Reverting the fixes reproduces both missing recovery and the null.pointer crash.
  - id: BR-4
    disposition: addressed
    note: |
      The revised integration table names board/sketch.go, recording/sketch_source.go, cast_sketch.go and board/writer_sketch.go correctly and identifies unchanged interfaces/doubles as reused. These path/status corrections match the pinned diff and are recorded under Revisions. A distinct PURE classification contradiction is reported below.
  - id: BR-5
    disposition: withdrawn
    note: |
      The supplied host run on 8be34c6fab5b1cc1c2431a7cadb1da915b2e793d executed all four required tests successfully. The diff from that commit to reviewed head changes only tracker/plan records, so the measurement covers this executable implementation. My rerun still fails before assertions with Chrome GPU/sandbox startup errors and CDP EOF; that does not contradict the supplied host results.
findings:
  - id: new
    severity: Critical
    family: approved-rendering-contract
    title: |
      Both live and persisted strokes violate the approved 3 px width contract
    detail: |
      board/sketch.go:305 persists Width: 4 and web/sketch.js:354 uses optimistic width 4. The approved design at workshop/history/plans/000750-cast-sketch-plan.md:200 and execution plan at workshop/plans/000752-cast-sketch-drawing-plan.md:85 require base width 3. This makes every pressure-derived width one-third thicker, including replay. A disposable Go test overlay changing the existing service assertion to the specified width fails with “durable width 4 differs from approved fixed sketch tool width 3.” ARCH-PURPOSE: restore 3 in both paths and pin the specified pressure widths in regression tests; the existing assertion at board/sketch_test.go:49 currently enshrines the incorrect value.
  - id: new
    severity: Critical
    family: core-concept-map-matches-code
    title: |
      The Core concepts table still labels effectful components as PURE
    detail: |
      workshop/plans/000752-cast-sketch-drawing-plan.md:38 classifies the painter as PURE, but web/sketch-render.js:343 mutates an injected canvas context, and tests/webcast/sketch-render.test.mjs:92 supplies a canvas double. Row 40 also classifies the journal as PURE although board/sketch_delivery.go:136 obtains a random epoch through randomID(). ARCH-PURE: distinguish the pure geometry/projection entities from the INTEGRATION canvas and random-initialization adapters, and record the correction under Revisions. This is the 2nd finding in family core-concept-map-matches-code. State and apply the rule across every table row: classification must follow actual effects and test dependencies, not merely file placement; sweep the complete table rather than correcting only these examples.
```

1. **Strengths**

   - Both surfaces reuse the shared tool, recovery policy, renderer and acknowledged service.
   - Ownership tests exercise stale epochs, ABA handoffs and held persistence/release boundaries.
   - Source preparation enforces independent prefix/state bounds before committing begin.
   - README and atlas document controls, recovery, privacy and lifecycle ownership.

2. **Critical findings**

   The two findings above require contract-aligned width behavior and a corrected PURE/INTEGRATION classification.

3. **Important findings**

   None newly raised. TL console smoke and the corrected-head recorded full-suite rerun remain explicitly pending; this review does not establish their completion.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - All **414** Node webcast tests.
   - `go test ./cmd/internal/webcast/... ./cmd/internal/strictjson ./cmd/internal/artifactpath`.
   - Race-enabled sketch/drawing tests for Board, drive and recorder.
   - `git diff --check`.

   Reverting three browser fix files in a disposable copy produced **25 failures**, including the original console-label and pointer-up defects. Temporary test copies and overlays were removed; the repository remains unchanged.

6. **Architectural notes for upcoming work**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: shared UI, geometry and writer seams are reused. |
   | ARCH-PURE | **Flag:** effectful painter/journal classification, detailed above. |
   | ARCH-PURPOSE | **Flag:** implemented width contradicts the approved rendering contract. |
   | ARCH-MOCK | Pass: stateful writer/controller/CDP doubles share production seams; host conformance evidence covers head. |
   | ARCH-CONSTRAINTS | Pass: bounded queues retain terminal work; source, transfer and journal limits have coverage. |
   | ARCH-SECURE | Pass on inspected structure and supplied conformance: authenticated ingress, scoped assets and opaque scene isolation. |
   | ARCH-ORDER | Pass: corrected lifecycle reconciliation and terminal recovery have controlled-order regressions. |
   | ARCH-FUNERAL | Pass: sources, timers, transfers, blobs and workers have retirement owners. |
   | ARCH-EVOLVE | Pass: optional truncation preserves absent/false semantics and has legacy unknown-field preservation coverage. |

7. **Plan revision recommendations**

   Add `## Revisions` entries recording:

   - Restoration of the approved 3 px base across optimistic and persisted strokes, with contract-based regression evidence.
   - A complete Core concepts classification sweep separating pure geometry/projections from canvas, randomness and other integration effects.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f33df438-785c-4978-be54-05bb60d101f3","repository":"github.com/xianxu/pair","issue_id":"000752","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/540ec201ce77f0f3","reviewed_head":"e9ea2bbf3ed082d870a2f67af932ce78dff28d93","slot":"pair:9","requested_at":"2026-10-21T10:49:10.097006Z"},"phase":"finished","pid":23898,"started_at":"2026-10-21T10:49:10.264146Z","finished_at":"2026-10-21T10:53:19.593029Z","session_id":"01a15e14-4afd-7230-8ed8-0e57f11ca24e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.454134334s
head: "e9ea2bbf3ed082d870a2f67af932ce78dff28d93"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T04:03:18-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 752 — cast-sketch: live console and driver drawing with pen pressure |
| repo | pair |
| issue file | workshop/issues/000752-cast-sketch-drawing.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..636d1e1a19a65fb3f5897cee9f3b15df2f51bd7a |
| command | sdlc close --issue 752 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T04:03:18-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-6 and BR-7 are addressed. The approved width regressions fail when their fixes are removed, and the revised Core concepts classifications match the inspected effects. One new blocking ordering defect remains: leaving while an anchor lookup is pending discards the buffered stroke and allows an empty sketch to complete.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The plan retains one function-verification strategy table; task checklists reference it instead of repeating case inventories.
  - id: BR-2
    disposition: addressed
    note: |
      Retain the accepted disposition. Authoritative metadata reaches shared policy; lifecycle and pending-label Node regressions pass.
  - id: BR-3
    disposition: addressed
    note: |
      Queue-saturation regressions pass for movement, pointer-up, unknown confirmation and explicit refusal recovery, retaining one truncated terminal.
  - id: BR-4
    disposition: addressed
    note: |
      The corrected Core concepts locations and modification statuses match the inspected pinned range.
  - id: BR-5
    disposition: withdrawn
    note: |
      Retain the prior evidence-based withdrawal. This environment again failed Chrome startup with GPU-process errors/CDP EOF; that repeats the disposed environment limitation.
  - id: BR-6
    disposition: addressed
    note: |
      Both producers use base width 3. In a disposable pinned-head copy, restoring width 4 independently failed TestSketchServiceFrozenAnchorAndExactRetry and the optimistic-contact pressure regression. Both pass at head.
  - id: BR-7
    disposition: addressed
    note: |
      Plan lines 29–44 restrict PURE to named deterministic entities; lines 59 and 65–68 explicitly classify random journal ownership, policy callbacks, canvas/scheduling and delivery callbacks as INTEGRATION. Supporting implementations include journal init(), paintSketchPaths() and createSketchDelivery(); Revisions records the complete classification sweep.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: bounded-input-retains-terminal-work
    title: |
      Leaving before the asynchronous anchor resolves discards the stroke
    detail: |
      cmd/internal/webcast/web/sketch.js:461–464 calls finish(), immediately dispatches leave, then awaits pending work. If scene.hitTest is unresolved, finish only buffers the terminal; the later anchor callback at lines 363–365 emits stroke-begin/points/end after sketch-policy.js:173 has entered label-required, so the guard at lines 190–201 rejects every command. Using the existing mount harness, all six combinations of Done/Escape/tool change and open/released contact completed with zero strokes and only sketch-begin plus sketch-label sent. ARCH-ORDER/ARCH-PURPOSE: introduce an explicit finishing transition that blocks new input and label submission while admitting and settling the existing gesture. This is the 2nd finding in family bounded-input-retains-terminal-work: state the preservation rule and sweep every voluntary exit across pending-anchor, queued and uncertain-work states on both adapters, with controlled-order regression tests.
```

1. **Strengths**

   - Approved pressure widths now have independent producer and optimistic-painter assertions, verified by removing each fix.
   - Writer seams distinguish confirmed persistence from unknown outcomes and retain exact operation identities.
   - README and atlas document controls, recovery, limits and private-scene ownership.
   - Core concepts now distinguish pure helpers from effectful wrappers in mixed files.

2. **Critical findings**

   The delayed-anchor exit defect above blocks shipping. Preserve the gesture through an explicit finishing state before allowing label completion.

3. **Important findings**

   None newly raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed all webcast Go packages, strictjson and artifactpath checks.
   - Passed all **415** browser tests.
   - Passed focused race checks for Board, driving and recorder sketch behavior.
   - Both BR-6 regression tests failed with their respective fixes removed.
   - The existing suite misses pending-anchor completion after voluntary exit; the controlled reproduction exposed it in all six tested combinations.
   - Chrome conformance failed before assertions during GPU startup/CDP connection. TL console smoke and the recorded-check hold remain pending as documented.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared tool, renderer and writer preflight.
   - **ARCH-PURE — pass:** revised entity classifications match inspected effects.
   - **ARCH-PURPOSE — flag:** ordinary exit can complete without the user’s buffered stroke.
   - **ARCH-MOCK — pass:** stateful writer/browser seams support controlled integration checks; live execution has the limitation above.
   - **ARCH-CONSTRAINTS — pass:** bounded queues, transfers, history and counted work checks.
   - **ARCH-SECURE — pass:** inspected ingress authenticates authority and scopes source/asset access.
   - **ARCH-ORDER — flag:** label-required is entered before pending gesture effects can drain.
   - **ARCH-FUNERAL — pass:** source, scene, transfer and timer ownership have retirement paths.
   - **ARCH-EVOLVE — pass:** optional truncation preserves absent/false semantics and unknown-field round trips.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the voluntary-exit invariant: stop new capture, preserve and settle the current gesture, then permit label completion. Enumerate Done, Escape and tool changes against pending-anchor, queued and uncertain work, and retain controlled-order tests for both adapters.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a750da2d-99ae-4c0b-a4c5-a2e26088d435","repository":"github.com/xianxu/pair","issue_id":"000752","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a980fba124a9a86a","reviewed_head":"636d1e1a19a65fb3f5897cee9f3b15df2f51bd7a","slot":"pair:9","requested_at":"2026-10-21T10:59:31.671278Z"},"phase":"finished","pid":52529,"started_at":"2026-10-21T10:59:31.765548Z","finished_at":"2026-10-21T11:03:17.495705Z","session_id":"01a15e1d-c635-7962-b2e9-6c2b7e72831a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.555164666s
head: "636d1e1a19a65fb3f5897cee9f3b15df2f51bd7a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T04:21:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 752 — cast-sketch: live console and driver drawing with pen pressure |
| repo | pair |
| issue file | workshop/issues/000752-cast-sketch-drawing.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..208edef0488a0171064d6892db65d9a878880288 |
| command | sdlc close --issue 752 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T04:21:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-8’s voluntary-exit correction is verified: all 16 delayed-anchor regressions pass at the pinned head and fail with the fix removed in a disposable copy. One adjacent blocking defect remains: starting another stroke after pointer-up but before anchor resolution crashes the shared controller on both surfaces. Focused Go/Node checks and the reader inventory pass; live Chrome checks remain limited by the previously documented environment failure.

1. **Strengths**

   - Both adapters reuse the same finishing policy, preserving pending geometry before requesting a label.
   - Writer acknowledgments distinguish durable acceptance from uncertainty, retaining immutable retry identities.
   - README and atlas document controls, recovery, privacy boundaries and limits. The revised Core concepts tables distinguish deterministic helpers from effectful wrappers.

2. **Critical findings and prior dispositions**

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retain prior disposition: implementation tasks reference the shared function-verification table instead of repeating case inventories.
  - id: BR-2
    disposition: addressed
    note: |
      Retain prior disposition: console authoritative lifecycle and late-label regressions pass in the pinned-head Node suite.
  - id: BR-3
    disposition: addressed
    note: |
      Retain prior disposition: queue-saturation terminal and pointer-up regressions pass; the finding below concerns a separate pending-anchor ordering.
  - id: BR-4
    disposition: addressed
    note: |
      Retain prior disposition: Core concepts paths and new/modified/reused classifications match the inspected implementation range.
  - id: BR-5
    disposition: withdrawn
    note: |
      Retain prior withdrawal. Local live checks again encounter GPU-process startup failure/CDP EOF; this does not reopen the previously disposed environment finding.
  - id: BR-6
    disposition: addressed
    note: |
      Retain prior disposition: producer and optimistic painter use width 3; durable-width and pressure regressions pass.
  - id: BR-7
    disposition: addressed
    note: |
      Retain prior disposition: PURE rows identify deterministic symbols, while callbacks, canvas, journal ownership and live adapters are classified INTEGRATION.
  - id: BR-8
    disposition: addressed
    note: |
      All 16 viewer/console delayed-anchor exit regressions pass at head and fail with the finishing fix removed in a temporary copy; the temporary copy was removed.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: bounded-input-retains-terminal-work
    title: |
      A second contact before the previous anchor resolves crashes drawing on both surfaces
    detail: |
      At cmd/internal/webcast/web/sketch.js:352–357, pointer-up clears contact while the logical gesture remains pending its anchor. The down guard at lines 367–376 admits another contact and mutates contact before createSketchGesture.start rejects it at sketch-policy.js:437. Controlled viewer and console harnesses reproduce “gesture already open”, followed by a TypeError reading contact.last at sketch.js:346 on the next pointer-up; the second stroke is absent. ARCH-ORDER/ARCH-PURPOSE: this is the 3rd finding in family bounded-input-retains-terminal-work. Earlier rounds fixed instances. Do NOT fix only this instance: state and enforce the rule that physical contact release does not retire logical gesture ownership. Enumerate pointer-up, pointercancel, lost capture and automatic buffer/point-limit endings, and prevent or explicitly queue successor gestures until their predecessor relinquishes ownership. Add controlled-order regressions on both adapters.
```

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed focused webcast Go packages, `TestNode`, operator assembly checks, `make test-affected-inventory`, and range whitespace checks.
   - Independently reproduced the new crash through both existing mount harnesses without modifying the checkout.
   - Existing pending-anchor coverage resolves the anchor before starting the next stroke ([sketch.test.mjs:324](/Volumes/TestScratch/review-checkouts/d48da7d342cc7fc3/tests/webcast/sketch.test.mjs:324)); it misses this ordering.
   - All four requested live Chrome cases failed before substantive assertions because Chrome’s GPU process could not initialize or CDP returned EOF. Real console smoke and the corrected-head recorded check remain explicit pre-landing holds.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared tool, renderer, authority and writer seams are reused.
   - **ARCH-PURE — pass:** inspected deterministic helpers match their narrowed classifications; effects are identified separately.
   - **ARCH-PURPOSE — flag:** ordinary successive strokes can fail on both required drawing surfaces.
   - **ARCH-MOCK — pass structurally:** stateful Writer/Hub/CDP harnesses exercise production seams; live verification has the limitation above.
   - **ARCH-CONSTRAINTS — pass:** explicit queue/source/transfer limits and counted-growth tests are present.
   - **ARCH-SECURE — pass on inspected paths:** ingress validates authority and shape; private reads remain source-bound.
   - **ARCH-ORDER — flag:** physical contact and logical gesture lifetimes disagree about when another contact is legal.
   - **ARCH-FUNERAL — pass:** scene, timer, transfer and journal resources have retirement paths and bounds.
   - **ARCH-EVOLVE — pass:** truncation is additive, absent/false compatibility is tested, and preserving codec seams remain intact.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining logical gesture ownership through delayed-anchor settlement. Enumerate every physical-contact ending and successor-input transition, then require controlled-order tests on both adapters. Reopen Task 5’s pointer verification until that rule is enforced.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"44f88c08-8708-45c0-b5ca-4c68de1d92c5","repository":"github.com/xianxu/pair","issue_id":"000752","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/d48da7d342cc7fc3","reviewed_head":"208edef0488a0171064d6892db65d9a878880288","slot":"pair:9","requested_at":"2026-10-21T11:18:13.540205Z"},"phase":"finished","pid":78378,"started_at":"2026-10-21T11:18:13.708684Z","finished_at":"2026-10-21T11:21:38.74408Z","session_id":"01a15e2e-e583-7853-82da-5bdbb6b61475","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.955410375s
head: "208edef0488a0171064d6892db65d9a878880288"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T04:42:24-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 752 — cast-sketch: live console and driver drawing with pen pressure |
| repo | pair |
| issue file | workshop/issues/000752-cast-sketch-drawing.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..320d01886adaa2859c3bfd1c51306826dc38e32b |
| command | sdlc close --issue 752 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T04:42:24-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

BR-9 is addressed with meaningful regression evidence: all eight controlled physical-ending cases pass at head and fail without the fix. One separate, inexpensive workflow gap remains: Enter does not save the required label on either surface, despite the approved design.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retain disposition; the plan uses the function verification strategy table.
  - id: BR-2
    disposition: addressed
    note: |
      Console handoff and old-author label regressions pass.
  - id: BR-3
    disposition: addressed
    note: |
      Queue saturation reserves a truncated terminal; move and pointer-up regressions pass.
  - id: BR-4
    disposition: addressed
    note: |
      Core concepts now identify existing implementation paths and distinguish new, modified and reused surfaces.
  - id: BR-5
    disposition: withdrawn
    note: |
      Retain withdrawal. This environment again failed during Chrome GPU/sandbox startup before assertions; no new conformance claim is made.
  - id: BR-6
    disposition: addressed
    note: |
      Producer and optimistic painter both use base width 3; relevant Go and Node tests pass.
  - id: BR-7
    disposition: addressed
    note: |
      Core concepts distinguish deterministic symbols from effectful wrappers and their integration tests.
  - id: BR-8
    disposition: addressed
    note: |
      Delayed-anchor exit regressions pass for both adapters, including open and released contacts.
  - id: BR-9
    disposition: addressed
    note: |
      Eight pointer-up/cancel/lost-capture/buffer-limit regressions pass at head and all fail with “gesture already open” when sketch.js is restored to its pre-fix version in a disposable checkout.
findings:
  - id: new
    severity: Important
    kind: defect
    family: approved-controls-match-behavior
    title: |
      Enter does not submit the required sketch label on either surface
    detail: |
      ARCH-PURPOSE: the approved interaction at workshop/history/plans/000750-cast-sketch-plan.md:36 says Enter saves a nonblank label and returns to the requested tool. The shared handler at cmd/internal/webcast/web/sketch.js:507 handles only Escape. Controlled viewer and console reproductions at pinned head leave zero sketch-label commands after Enter. Route Enter through saveLabel, preserving its settlement guards, and add both-adapter regressions proving the requested tool opens only after acceptance.
```

1. **Strengths**

   - Logical admission checks the accumulator token before allocating contact, capture, identity or hit-test.
   - Anchor success and failure callbacks check both generation and stroke identity.
   - README and atlas document controls, recovery, limits and private-scene boundaries.

2. **Critical findings:** None.

3. **Important findings:** Enter-to-save is missing at `cmd/internal/webcast/web/sketch.js:507`. Reuse the existing guarded `saveLabel()` path.

4. **Minor findings:** None.

5. **Test coverage notes**

   - All 113 viewer/console tests pass.
   - Webcast package suite, strictjson, artifactpath, inventory and diff checks pass.
   - Focused authority/writer race tests pass. The broader race run exceeded its 60-second limit in the large bootstrap fixture.
   - Live Chrome assertions could not execute because browser startup failed.
   - Disposable regression checkouts were removed; reviewed files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared tool, reducer, source and rendering seams.
   - **ARCH-PURE — pass:** deterministic entities distinguished from effectful adapters.
   - **ARCH-PURPOSE — flag:** approved Enter-to-save action is absent.
   - **ARCH-MOCK — pass:** stateful Writer/Hub/CDP seams exercised.
   - **ARCH-CONSTRAINTS — pass:** explicit queue, source, history and transfer bounds.
   - **ARCH-SECURE — pass:** authenticated authority, strict ingress and scoped private scenes.
   - **ARCH-ORDER — pass:** retained logical ownership and fenced late completions have controlled-order coverage.
   - **ARCH-FUNERAL — pass:** source, scene, timer, transfer and journal lifetimes are bounded.
   - **ARCH-EVOLVE — pass:** additive truncation retains absent/false compatibility and unknown-field preservation.

7. **Plan revision recommendation**

   Add a `## Revisions` entry recording Enter-to-save parity with #750 and both-adapter keyboard/acknowledgment coverage. Existing recorded-check and real-console-smoke holds remain separate pre-landing requirements.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"256d00bf-cf86-4b57-9c5b-5e91a5efea40","repository":"github.com/xianxu/pair","issue_id":"000752","recipe":"milestone-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/5f546d39033bfe5d","reviewed_head":"320d01886adaa2859c3bfd1c51306826dc38e32b","slot":"pair:9","requested_at":"2026-10-21T11:37:50.369516Z"},"phase":"finished","pid":54911,"started_at":"2026-10-21T11:37:50.470553Z","finished_at":"2026-10-21T11:42:23.283692Z","session_id":"01a15e40-d98d-7b72-9af9-318b4623b594","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.532948917s
head: "320d01886adaa2859c3bfd1c51306826dc38e32b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T13:34:03-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 752 — cast-sketch: live console and driver drawing with pen pressure |
| repo | pair |
| issue file | workshop/issues/000752-cast-sketch-drawing.md |
| boundary | whole-issue close |
| milestone | — |
| window | 65469fe6b2d13d83a736c11e948397711e1559a9..4f0f3228eec3ae76481f3bc120f22c216bc38f7e |
| command | sdlc close --issue 752 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T13:34:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

BR-10 is addressed with meaningful regressions on both surfaces. The shared implementation, authority fencing, bounded delivery and additive recording changes pass the available automated checks. One documentation correction remains: README describes controls and console prerequisites that changed in the latest revisions. Live Chrome verification remains limited by this environment’s previously reported GPU/CDP startup failure.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained: the plan uses a function/strategy verification map rather than repeated prose inventories.
  - id: BR-2
    disposition: addressed
    note: |
      Retained: authoritative console lifecycle reconciliation and interrupted-contact label regressions remain present and pass.
  - id: BR-3
    disposition: addressed
    note: |
      Retained: shared queue-saturation and pointer-up terminal-recovery regressions pass.
  - id: BR-4
    disposition: addressed
    note: |
      Retained: the Core concepts tables name actual implementation paths and distinguish new, modified and reused entities.
  - id: BR-5
    disposition: withdrawn
    note: |
      Retained from the prior disposition; this environment again fails before live assertions with Chrome GPU initialization errors and CDP EOF. This does not contradict the previously accepted host evidence.
  - id: BR-6
    disposition: addressed
    note: |
      Retained: board/sketch.go produces width 3; approved-width browser and persisted-pressure regressions pass.
  - id: BR-7
    disposition: addressed
    note: |
      Retained: the revised tables classify callbacks, DOM/canvas, journal initialization and Hub observation as INTEGRATION, separately from deterministic helpers.
  - id: BR-8
    disposition: addressed
    note: |
      Retained: finishing owns delayed-anchor settlement; both-adapter voluntary-exit and deadline regressions pass.
  - id: BR-9
    disposition: addressed
    note: |
      Retained: accumulator ownership guards successor admission, and both-adapter delayed-anchor/successor regressions pass.
  - id: BR-10
    disposition: addressed
    note: |
      sketch.js:526 routes Enter through saveLabel with composition guards. Tests at sketch.test.mjs:963 and operator-sketch.test.mjs:381 pass; removing only the Enter branch through a temporary loader makes both fail because zero label commands are sent.
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-docs-match-controls
    title: |
      README retains superseded console prerequisites and label-control instructions
    detail: |
      README.md:1572 says annotation must be enabled for console Sketch, although board/annotation.go:54 and operator-sketch.js now deliberately allow trusted console sketching with viewer annotation off. README.md:1593 also directs users to “Label unfinished sketch,” replaced by “Note” in web/sketch.js:147, and omits the new active-sketch Note action. ARCH-PURPOSE: update the complete sketch usage passage to match the approved revisions, including the background-capture deferral qualification to the blanket capture statement at README.md:1623; atlas/webcast.md already describes these changes.
```

1. **Strengths**

   - Both surfaces reuse the same tool and guarded label-save path.
   - Writer settlement distinguishes durable acceptance from unknown persistence; exact retries retain operation identity.
   - Console admission remains generation-bound and cannot authorize ordinary viewer annotations.
   - Cancellation’s optional `truncated` field preserves absent/false identity and unknown-field round trips.

2. **Critical findings**

   None.

3. **Important findings**

   Update [README.md:1572](/Volumes/TestScratch/review-checkouts/825d50d258c4180a/README.md:1572) as detailed above. This is a prose correction; wording-presence tests are unnecessary.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: all 484 browser tests.
   - Passed: `go test ./cmd/internal/webcast/... ./cmd/internal/strictjson ./cmd/internal/artifactpath`.
   - Passed: affected-reader inventory and pinned-range whitespace checks.
   - BR-10: both regressions fail without the Enter handler.
   - Live Chrome: all four selected fixtures failed during browser startup, before behavioral assertions.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared UI, renderer, preflight and Unicode validation.
   - **ARCH-PURE — pass:** deterministic helpers are separated from explicitly classified integration adapters.
   - **ARCH-PURPOSE — flag:** user instructions lag the approved controls; both required implementation surfaces are present.
   - **ARCH-MOCK — pass:** stateful writer/browser/CDP seams exist; live verification has the limitation above.
   - **ARCH-CONSTRAINTS — pass:** explicit queue, source, history and transfer bounds have focused coverage.
   - **ARCH-SECURE — pass:** authenticated ownership, generation checks and private-scene isolation are explicit.
   - **ARCH-ORDER — pass:** production policy and gesture ownership govern settlement and successor admission.
   - **ARCH-FUNERAL — pass:** sources, timers, transfers and scenes have bounded lifetimes and disposal paths.
   - **ARCH-EVOLVE — pass:** the persisted addition preserves compatibility and strict validation.

7. **Plan revision recommendations**

   No design revision is needed. Complete the README synchronization required by the existing documentation task; retain the explicit pre-landing console-smoke hold.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"78519034-962f-46c5-918b-1939ccee2b1f","repository":"github.com/xianxu/pair","issue_id":"000752","recipe":"milestone-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/825d50d258c4180a","reviewed_head":"4f0f3228eec3ae76481f3bc120f22c216bc38f7e","slot":"pair:9","requested_at":"2026-10-21T20:31:51.009069Z"},"phase":"finished","pid":99878,"started_at":"2026-10-21T20:31:51.131738Z","finished_at":"2026-10-21T20:34:02.496407Z","session_id":"01a16029-c076-79f3-865f-9c6f8f579202","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.887835625s
head: "4f0f3228eec3ae76481f3bc120f22c216bc38f7e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
