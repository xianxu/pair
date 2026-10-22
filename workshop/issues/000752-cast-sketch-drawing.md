---
id: 000752
status: codecomplete
deps: [pair#751]
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 10.63
collaboration-mode: collaboration
card_mirror: 'b812206c9ab49db6a75e91e96dc2d5852d506ea9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T00:30:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 5.41
---

# cast-sketch: live console and driver drawing with pen pressure

## Problem

Deliver the approved live drawing tool on both required surfaces: operator inside the casting Chrome and the currently granted drive-link viewer, including iPad/Apple Pencil.

## Spec

Approved design: #750, workshop/plans/000750-cast-sketch-plan.md, especially its 2026-10-20 operator approval revision. This issue implements only its assigned slice. Start from main after dependencies land; write a slice-specific execution plan and pass the SDLC gates. No stacked branches.

Drawing follows driving, exactly one drawer cast-wide: console user when control is local; current granted drive-link holder when remote. A drive link without the current seat grants no drawing. Pointer-link and view-only viewers watch; pointer-link users retain pointing. Use current driving ownership/generation as authority, not an independent multi-drawer grant system or attribution ID.

Implement acknowledged begin/chunk/end/Undo/Redo/label/status operations, exact retries, writer admission checks and a held source-moment scene on BOTH casting Chrome and remote drawer. Label is required on leaving the tool, Undo/Redo affect the current sketch only, colour is fixed per sketch, one anchor per stroke. Old-author metadata-only label finalization may survive a handoff but never grants drawing or blocks the new holder. No takeover after expired link/session.

Casting Chrome controls use the privileged excluded isolated-world host and recorder binding, with bounded private snapshot/asset transport; no page script receives credentials/bindings. Reuse the shared sketch UI/renderer and existing snapshot/asset seams; this slice does not depend on the later headless CLI exporter.

Support pen Pointer Events and normalized pressure, coalesced samples, pointer capture, hover exclusion, release/cancel handling and deterministic mouse/touch fallback. Stream partial strokes to observers before pointer-up. Grant/revoke/release/leave/reset must seal prior accepted work and fence old-generation queued writes before enabling the next drawer. Pause/end/reconnect/expiry, backpressure, limits and failure UI follow the approved state model.

Primary surfaces: board/{sketch.go,sketch_wire.go,sketch_delivery.go,writer_sketch.go,operator_view.go}, recorder/{sketch.go,exclusion.go,operator_original.go}, recording/{sketch_source.go,sketch_anchors.go}, server_sketch.go, cast_sketch.go, cast.go, chrome_operator_sketch*.go, chrome_sketch_scene.go, operator-sketch.js, operator.js, operator_script.go and web/{sketch.js,sketch-policy.js,sketch-scene.js,sketch.css,viewer.js,annotate.js,render.js}. Existing browser and placement interfaces are reused.

## Done when

- Two-viewer test observes first incomplete stroke before drawer pointer-up; Undo/Redo change everyone’s overlay and preserve replay history.
- Real casting-Chrome operator can draw in its held scene, label on leave and Undo/Redo; currently granted drive-link drawer can do the same.
- Pointer/view links and ungranted drive links cannot draw; pointer-link pointing still works.
- Held-write tests force every driving handoff across admission/queue/sink/reply and prove one drawer, old-generation refusal, retained partial evidence and independent pending-label completion.
- Pen pressure survives live transport; coalesced samples, hover/cancel/release and mouse/touch fallbacks are covered. Budget, source synchronization, exact retries and N/2N counters pass.
- README/atlas map the controls and recovery. Required checks/review pass, and TL-arranged real console input smoke succeeds before landing.

## Plan

- [x] Claim/start-plan after dependencies land; write the slice-specific execution plan at [000752-cast-sketch-drawing-plan.md](../plans/000752-cast-sketch-drawing-plan.md).
- [x] Implement both surfaces and verify protocol, pressure, recovery, privacy and placement through automated checks.
- [x] Complete automated implementation verification and the recorded check; run the close review at the acceptance gate. TL console smoke remains a separate pre-landing hold.

## Log

- 2026-10-21: Authenticated TL release1a61c2c4-9b66-4b57-bc40-416fef0b2cd0 authorizes stage --build on pair:0 as soon as the shared cache is cleared; TL owns the real console smoke. Returned to preserved branch7beb52b43120b40879a6d9cae1a32245f6ec7c76. #767 landed PR520 and retired its registration. #771 is codecomplete/PR527, root fix and narrow review complete, but its integrated suite failed before tests because AgentScratch/go-build was full; branch and evidence retained. Gracefully stopped only its verified waiting merge (exit130) before this branch switch; no suite or peer runner was canceled. Operator is reclaiming shared cache; no stage/build attempted while disk remains full. Existing final-source suite10m2s remains valid for this branch's unchanged executable code.

- 2026-10-21: Final-code recorded full suite PASSED on61e43e35e7794260aa860c3b3ff269e687351e14: exit0 in10m2s after1372s queue, attempt735922b215e89a192d144d44af5f4a35, log .git/sdlc/local-checks/61e43e35e7794260aa860c3b3ff269e687351e14/suite.log, SHA256 prefixc0f339cf4f5a. GitHub confirms sdlc-local/suite success. Every webcast package, artifactpath, repolint and couchcore passed; no flake rerun needed. Automated acceptance and finalized local close are complete. Waiting for TL restart/staging release and real console smoke before landing PR512; no stage or merge performed. Subsequent evidence commits are prose only and do not require replacing this source-equivalent pass.

- 2026-10-21: Continuation recovery found the retained final-source check had not survived the context restart: exec session72612 unknown; lease-2496662231, inflight log, detached checkout and saved attempt absent; no recorded suite status on607f31bf. Started replacement on pushed prose-only head61e43e35e7794260aa860c3b3ff269e687351e14 (Git confirms only issue/plan/continuation differ from accepted source). Retain exec session31948, lease-3343331055, attempt735922b215e89a192d144d44af5f4a35, checkout /Volumes/TestScratch/local-checks/sdlc-local-2988573213, log .git/sdlc/test-stats/inflight/735922b215e89a192d144d44af5f4a35/run.log. Full suite fallback, queued sixth. No final recorded pass yet. Codecomplete remains finalized; stage, real console smoke and landing remain held. Hub recovery report receipt c68e8d5a-5fc4-4a6c-8da7-47e1ce4caacf submitted.

- 2026-10-21: FIX-THEN-SHIP correction committedb8c373efdb7437a0d12768b38a7c40852ce1a5a0. SDLC recovery reconcile completed/pushed the close: evidencead61650fb44bab6a4ac0c98a1646bf79697f385b, mirror607f31bf0648aed1a934cf5e80316c606fce9f8e, cardcodecomplete confirmed. Root final Node/operator1.915s; live Chrome4 passed7.461s after Enter fix (privacy1.54s, drawing2.06s, handoff1.56s, two viewers2.14s). Canceled only obsolete own lease-3562366554 after required executable fix; retain final source run lease-2496662231 on607f31bf / attemptec72f1b3a2638dba7d7b0e1c8c652d38 / worktree sdlc-local-155639581, queued6, full fallback. No final recorded pass claimed. All review criticals disposed; BR-10 was fixed under the explicit post-verdict protocol, no redundant review needed. Stage/real TL console smoke and landing remain held.

- 2026-10-21: Close on320d01886adaa2859c3bfd1c51306826dc38e32b returned FIX-THEN-SHIP: all BR-1 through BR-9 disposed (BR-5 withdrawn), no criticals; BR-10 Important missing approved Enter-to-save. Verified #750 interaction and fixed by delegating Enter to existing saveLabel, preserving composition and settlement guards. Both-adapter tests RED2/2 then GREEN2/2 cover blank/composing/repeated Enter, held label ACK (no tool/source release), accepted requested-tool transition exactly once and no later duplicate. Root final TestNode/OperatorScript passes1.915s, inventory/diff pass. Bundle fixes and review artifacts once, then reconcile the pending close as instructed; do not dispatch redundant close review. Final recorded suite and real console smoke remain open and held.

- 2026-10-21: BR-9 source committed72b928d3465a5214b70b847aed8c6c34bb83f346; measurement fixture/evidence committed7a080ecf49b83cf6b388f14ed3a46d635d108502 and pushed PR512. Final inventory passes; artifactpath9.376s and repolint9.750s pass. Canceled only obsolete own lease-2755259726 after the executable correction; replacement final-head run lease-3562366554 / attempt9d3757397da36d5f0da3079819986735 / worktree sdlc-local-586396399 is retained at queue position6, full fallback for the existing graph/embed gap. Local close may overlap this queue, with the final recorded pass and TL-held console smoke still mandatory before landing. No check bypass or staging.

- 2026-10-21: Final direct-token latency run passed108.30s with the actual createSketchGesture physical-release→anchor→terminal-drain path. Root recomputed raw /Volumes/TestScratch/tmp/sketch-anchor-latency-3872495943/samples.json: local viewer n500 p95/max0.1/0.5ms; tunnel-served viewer n500 0.1/0.1ms; actual console n300 0.5/2.1ms; console with real60Hz pressure/journal n300 0.6/4.5ms, journal cursor advanced3. Zero of1600 token samples exceeded150ms. Distinct tunnel begin/terminal ACK n30 each p50 28.4/27.3ms, p95 34.1/31.7ms, max49.6/41.4ms; source preparation4.9/79.7ms local/tunnel, n1 each. Synthetic small desktop fixture only; mobile/large-DOM/echo remains unqualified. Retain no-successor-queue design. Repro fixture TestSketchAnchorLatencyLive is opt-in with WEBCAST_MEASURE_SKETCH_ANCHOR=1 and WEBCAST_LIVE_CHROME=1, owns anonymous synthetic tunnel/Chrome and raw scratch output, bounded2min. Final fixture cleanup was tightened and compile/skip checked0.223s after measurement; measurement logic unchanged.

- 2026-10-21: BR-9 implemented from the complete gesture/event table: sole accumulator-token admission before allocation, consumed successor tails/fresh-down requirement, transactional coordinate/capture setup, five-second ordinary anchor observer, logical-owner Undo/Redo admission, and exact token+generation guards on success/failure. Unknown prior-terminal responses retire unanchored current gestures whether physical contact remains open or already released; unsent optimistic ink is removed and no nonexistent terminal is invented. Both-adapter ending, same/different pointer, N/2N allocation, timeout,4096 FIFO, setup, Undo/Redo and stale-completion tests pass. Additional open/released×oldreject/timeout×both-adapter test was RED4/8 then GREEN8/8. Root Node/operator checks pass1.435s; host live4 pass5.175s; reader inventory/diff pass.
- TL latency condition: first real owned headless local/anonymous-tunnel run passed111.81s. Root independently recomputed raw samples at /Volumes/TestScratch/tmp/sketch-anchor-latency-2051644339/samples.json: local ready viewer n500 p95/max0.1/0.2ms; tunnel-served viewer n500 p95/max0/0.1ms (timer resolution); actual console parent/worker/private-CDP n300 p95/max0.5/1.2ms; console with live60Hz pressure/journal n300 p95/max0.5/1.0ms. All1600 anchor samples zero above150ms. Distinct ACK series n30 each: local begin/end p95 2.2/1.2ms, tunnel37/40.4ms (max64.8/71.7ms). Source preparation local4.7ms/tunnel90.3ms, n1 each. ACK is not accumulator ownership after terminal production. Desktop synthetic samples support no successor queue; device/echo remains #754. Second pass adds accumulator-drain timing; no staging authorized.

- 2026-10-21: TL approved BR-9 (50744457-9fa4-450d-9421-52f62ea94a44), requiring one complete gesture/event table and measured local/tunnel anchor latency. Canonical table committed11020253b covers idle/drawing/anchoring/finishing/retired on both surfaces. Browser owner established eight controlled RED cases across pointer-up/cancel/lost-capture/1024-buffer ending on both adapters (all fail160.833ms), then implementation authorized from table. Wire owner qualifies real headless local/tunnel fixture latency via existing anonymous Cloudflared seam; no staging or machine configuration change.

- 2026-10-21: Next close verified BR-8 with all 16 regressions passing at head and failing without the fix, then returned REWORK BR-9 (Critical, third bounded-input-retains-terminal-work instance). A successor pointerdown after physical pointerup but before async anchor resolution mutates contact before the still-owned gesture rejects start; next pointerup crashes. Stopped for a complete logical-gesture ownership/admission re-plan across every physical ending and successor transition. Existing final-head runner lease-2755259726 retained; no new production edits yet. Staging remains held.

- 2026-10-21: Final BR-8 executable head 4f30ab65ccb8909beaf8f4fc2e8d87ca5ea355e1 pushed to PR512. Host WEBCAST_LIVE_CHROME=1 conformance passed all four cases in 5.602s: scene privacy 0.98s, console drawing 1.68s, open-contact handoff label 0.91s, two viewers 1.89s (three open pressure points, one echo, Undo/Redo/label/replay agree). Root Node/operator checks passed 1.592s; inventory and diff checks pass. Canceled only obsolete own queued lease-3778018252 after executable fix; new final-head run lease-2755259726 / attempt 55220d956c975a5e7f260fc428d52b7f is retained, queued position5. Existing affected package-graph embed gap again selects the full suite. No final-head recorded pass claimed; review overlaps the queue under tl-worker contract. TL stage/real console smoke and landing remain held.

- 2026-10-21: BR-8 implemented with one receipt-owned finishing phase and shared viewer/console tool gate. Freeze capture before ending the gesture; admit only its existing stroke identity until anchor/FIFO/terminal receipts settle. First exit intent survives unknown and definite-refusal recovery; label Escape ends that intent. Shared five-second request observers release UI wait/deduplication ownership without changing writer operation identity; late transport replies cannot revive callbacks. Finishing visibly saves, then explicitly reports accepted-work-kept failure at deadline; unaccepted hung anchors are fenced, unknown writes remain queryable, all timers retire on disposal.
- BR-8 RED/GREEN: browser owner reproduced 16 delayed-anchor exit tests failing (Done/Escape/Pen/Note × open/released contact × both adapters), then passing. Added controlled queue saturation, invalid anchor, preparation, unknown-before-exit exact retry, hung HTTP/status/late response, refusal-prefix recovery, repeated exit and label Escape, handoff/pause/clear/end and timer-disposal coverage. Root final TestNode + TestOperatorScript passed 1.592s; inventory preflight and diff checks pass. Host live conformance, fresh final-head recorded check and close review still pending; stage remains held.

- 2026-10-21: TL approved BR-8 re-plan (ec1b8b37-c337-452a-a91e-45f37ba01996), requiring visible saving, existing-deadline failure with accepted history kept, and pending-anchor exit-control regressions on both surfaces. Browser owner reproduced all six initial cases read-only; independent audit found viewer reconciling gate, Escape and preparation exit intent must share the same owner. TDD implementation authorized; staging still held.

- 2026-10-21: Third close review on 636d1e1a accepted BR-6/BR-7 and returned REWORK for BR-8 (Critical, bounded-input-retains-terminal-work): Done/Escape/tool change before asynchronous anchor resolution enters label-required too early and rejects the buffered stroke. Stopped implementation for a whole voluntary-exit state re-plan and independent adapter audit. Recorded runner lease-3778018252 is retained; staging remains held by TL.

### 2026-10-20

- Filed from operator-approved #750 design via ops:0. Left unclaimed for TL dispatch.

### 2026-10-21 — execution design
- 2026-10-21: closed — Console smoke corrections: native capture focus-loss RED2.522s; fixed native console pressure/native toolbar hit-clicks/annotation-off admission, focus-caret-blur through repeated deferred captures and source updates, native AxB typing, capture resumption, private scene, two viewers and ordinary annotation PASS12.976s. Recorder PASS4.538s, Board5.065s, repolint11.387s, all484 Node PASS1.243s; inventory and diff PASS. Recorded final-head affected fallback retained and must pass before landing. TL owns project; restage and accepted real console smoke remain required. Follow-ups782/783/784 are unclaimed.; review verdict: FIX-THEN-SHIP
- 2026-10-21: closed — One canonical gesture/event table drives both-adapter controlled ownership tests: every ending, successor rejection, setup rollback, bounds, Undo/Redo, stale anchor success/failure, unknown/refusal and finishing recovery. Root Node/operator checks pass1.435s; live Chrome4 pass5.175s; artifactpath9.376s, repolint9.750s, inventory/diff pass. Actual local/tunnel accumulator-token measurement passed108.30s:1600 samples zero >150ms, console busy p95 0.6ms max4.5ms; raw evidence and fixture in Log. Earlier full suite passed; final executable-head full fallback lease-3562366554 is queued and must pass before landing. TL owns ops project; real console smoke/staging remains held.; review verdict: FIX-THEN-SHIP

- #751 landed as PR505 in train216/PR508, with branch full suite10m7s and integrated full suite11m57s; final CI passed. Its post-land local cleanup hit known ariadne#512. Per TL instruction preserved the landed branch, added evidence to that issue, returned to main-slot9 and fast-forwarded before claiming/starting this issue.
- Using writing-plans with read-only bounded browser/protocol/producer exploration. ARCH-ORDER: extend existing driving owner with a sketch-only revocation witness/epoch to fence A→B→A and release-in-progress; actor rechecks it without taking controller locks. ARCH-DRY: capture exact prospective source from Hub on the writer, reuse shared model/placement/tool, and deliver small operator render patches rather than copying full history per chunk.
- Draft plan records source/queue/delivery bounds, state/recovery mapping, generated/held-interleaving tests and required real console smoke. Checking actual console frame isolation before approval. No implementation or estimate yet.
- Disposable Chrome156 privacy probe found a named same-origin held iframe inside a closed shadow root can be read by page `window.open` targeting its known name, despite being absent from `window.frames`. Top isolated worlds (including universal access) cannot simply render an opaque-origin sandbox frame. Qualifying a concrete correction before approval; probe artifacts are private scratch, the durable result is recorded here and in plan Revisions.
- Qualified the actual shared Model/Renderer inside an opaque data shell's scoped child isolated world: nested frames, recorded name/CSS and private blob image rendered; parent named targeting was denied with SecurityError and binding/renderer globals stayed private. Chose this origin boundary over stripping recorded names, which changes source layout. Parent keeps the tool/label draft; child exposes only frozen rendering and read-only hit tests through trusted CDP. Plan also adds optional stroke-end truncation, needed to record pointercancel while continuing the same sketch.
- ops:0 approved the full execution plan (delivery96710fe7-a135-4b67-81e3-9189c71f5d63), including opaque scene and named-target regression. TL will do console smoke when staged; requested atlas invariant that the casting page never reaches the recorded scene, bindings or assets. Proceeding to plan-quality, then estimate, before implementation.


### 2026-10-21 — first implementation checkpoint

- Added strict ingress/replies, small owned reducer queries and additive truncated stroke-end semantics with Go/JS corpus coverage. Added a lock-free driving-turn witness and separate seat-lifetime metadata witness; A→B→A, pending-only transitions, held control/release and Close are covered.
- Frozen source preflight owns the exact fenced begin prefix and reduced scene, refuses independent 16 MiB prefix / 8 MiB state budgets without mutation, survives navigation compaction, and indexes only source-bound asset hashes. ARCH-ORDER/ARCH-CONSTRAINTS shaped this seam; node/sheet accounting visits each once.
- Verification returned green: focused recording/strictjson tests, race-enabled driving tests, and 87 Node sketch model/policy/render/delivery tests. Source received an independent review with no concrete ownership/order/accounting violation found. Shared Board/recorder/routes and actual UI integration are still in progress; this checkpoint does not claim end-to-end completion.
- Added the TL-required atlas invariant. Mandatory named-target Chrome regression and TL console smoke remain outstanding before landing.

### 2026-10-21 — ordered service and private delivery checkpoint

- Implemented Board ingress/origin receipts, original-owner late labels, a single frozen-source lease with one reusable cancelable timer, acknowledged reference/recorder writes, healthy observation barriers, scoped source/asset reads and strict drive-seat HTTP routes. Unknown persistence remains unknown. Added compact bounded operator journal/bootstrap with independent receiver verification.
- Fresh service review found epoch/sketch lookup collisions, lease deadline/held-reply ordering, last-refusal retention and late unknown-source resurrection gaps; fixed with focused regressions. Actor acceptance now renews the lease and ends the global begin reservation, independently of transport replies; old-label work stays separate. Sketch chunks bypass unrelated historical note projection scans.
- Returned verification: all Board, drive and recorder package tests pass; focused writer/service/journal race checks pass. HTTP race coverage includes pre-body role/seat/epoch refusal, pressure streaming, replay, old-author label during a new sketch, scoped reads and a held asset read retired by label/expiry. Cast fallback reuses the real Hub or adds one private source hub for absent/custom live factories.
- The mandatory actual Chrome privacy regression passed (unchanged shared Renderer, private SVG2×3, recorded nested frame name/CSS127 border-box, named-target SecurityError and private blob fetch TypeError). Console UI/observer exclusion integration, two-viewer live streaming and TL console smoke are still pending. The scene's asset transfer envelope is aligned to the existing recorder fallback capture32 MiB, separate from the16 MiB source prefix.

### 2026-10-21 — both surfaces and live proof

- Shared pressure tool, FIFO recovery, required label, Undo/Redo and incremental painter now run on both viewer and isolated console. Drawing suspends page drive input. Render-only journal delivery and anchor indexes avoid full path/point copies on normal samples; empty/undone anchors retain trusted current-document original-node placement, including common Save controls without an open note. Identical replacements receive no historical identity.
- Opaque child scene owns its renderer, bounded source/assets and CDP context lifetime. Recorder frame admission/exclusion precedes observer setup; exact private blob URL exclusion preserves ordinary sibling fonts. Trusted parent-context retirement fences late requests, including numeric context-ID reuse.
- Fresh actual Chrome run: TestOperatorSketchLiveScenePrivacy PASS2.10s, TestOperatorSketchLiveDrawing PASS1.57s, TestSketchLiveTwoViewers PASS2.77s (package6.597s). The tests exercised private DOM/named-frame/blob denial, actual StartCast console pen input, three pressure samples durable before pointer-up, single echo, Undo/Redo, required label and replay agreement. Main-world private hooks stayed unavailable. These owned fixtures do not replace TL console smoke.
- Focused fresh review found a coalesced clear→same-anchor bootstrap gap: the browser could prune its original proof while Go retained a bound acknowledgment. Ready anchor revisions now invalidate old/new proof attempts; incomplete journal delivery defers sketch-only retries. The failing regression now passes with the original-binding race suite (1.329s).
- Final consistency regression exposed durable width3 versus optimistic width4; the service now persists4 and its focused test fails before the fix and passes after it. Other targeted Go packages and Node/assembly checks passed; broader checks are being finalized. Audited18 production-file artifact dispositions and individually audited source-reader inputs; no new Pair artifact family.
- Additional actual StartCast regression PASS2.371s: completed ink remains on a common Save button with no ID and no note, follows that exact button by +200/+80 after leaving the held scene, and disappears for an identical replacement across worker retry turns. No fabricated binding or new production test hook. Shared Node tests214/214, original/assembly checks, targeted race checks, local make build, artifact classification and reader inventory pass.
- Staging is explicitly held by ops:0 delivery d053ca5e-cb78-44ee-b89c-d3db95f99841 until the coordinated Couch restart preserves the current main build. Ready-state report will precede staging, and TL go plus real console smoke remain mandatory before landing.

### 2026-10-21 — pre-smoke acceptance order

- PR512 is open on pushed implementation10e2d7b59465d1f955becbc6ceb8184e03a6ab3c; local make build returned exit0. Recorded affected selected the full suite because its fresh checkout lacks the generated runtime embed graph (known #738/ariadne#517 fallback); retaining its runner.
- ops:0 delivery b1d858cd-0dfc-445b-a623-c3e78b62a282 explicitly requests recorded check and close review now, with console smoke the sole remaining acceptance step afterward. Staging remains held until the operator restart; no landing before smoke and TL go. Close verification will identify this pending live acceptance rather than claim it happened.

### 2026-10-21 — recorded full suite

- Retained local-check runner acquired capacity after366s and completed the full fallback suite exit0 in11m11s. Recorded sdlc-local/suite on10e2d7b59465d1f955becbc6ceb8184e03a6ab3c; log SHA256 prefix c157a785c710. All Go packages and shell/editor checks passed. Later commits only document the acceptance hold, so this evidence covers unchanged implementation.
- Entering mandatory close review under the TL-directed pre-smoke order. Console smoke remains unperformed; stage and merge remain held until restart admission and that acceptance.

### 2026-10-21 — close round1 REWORK

- Mandatory review reproduced BR-2 (console interrupted open contact cannot label) and BR-3 (full point queue loses terminal sealing and may null-contact crash during pointer-up). Browser work owns red/green shared-policy and console-metadata fixes; actual console open-contact handoff regression is being added independently. Fix the lifecycle/terminal classes across both adapters.
- BR-4: Corrected the complete Core concepts map against the branch patch; removed nonexistent file splits and unchanged-interface modification claims. BR-1: Compressed task case inventories into the one function-strategy map, retaining commands and independent invariants. Both corrections are recorded in plan Revisions.
- BR-5 is an unverifiable host condition: review Chrome failed at GPU/sandbox startup and CDP EOF, while prior host runs passed. Rerun all required live tests on the corrected committed implementation and submit measured host evidence to the next review. TL smoke remains separate and unperformed; no stage or merge.

### 2026-10-21 — rework regressions

- BR-2 baseline actual StartCast handoff reproduced the permanent label block: final asynchronous-convergence oracle RED9.453s, exact message RED2.765s. The ordered actor had sealed/truncated the old contact and accepted the new holder begin before the label attempt.
- BR-3 Node baselines reproduced queue-full remaining in drawing and pointer-up null.pointer. The fix reserves one truncated terminal behind at most16 queued point chunks and visibly reconciles; focused tests cover movement/up, unknown retries and refusal→Continue with exactly one seal.
- BR-2 implementation connects journal metadata and stable sealed facts to shared policy, preserves interrupted drafts and obtains authoritative sequence before late label. New lifecycle tests cover empty/nonempty contacts through handoff, expiry, pause, clear and end, plus old/new drawer independence and held status races. Initial integrated rerun exposed repeated release-scene closing the label; fixed by separating scene and draft lifetime.
- Latest live console pair ran twice without diagnostic wrappers: all4 runs pass, package6.642s. Old label persisted and new holder subsequently submitted an accepted stroke using its original source/permit. Shared Node/assembly targeted run passed1.838s. A final retired-slot/cache residency sweep is being verified before committing and rerunning all host/recorded checks.

### 2026-10-21 — rework checkpoint

- Retired-slot cleanup also reproduced RED (settled clear retained a slot; unknown-clear late begin emitted begin-accepted). Both are fixed, including512 clear/restart cycles, original-operation retention until settlement, no resurrection, and pruning of all per-sketch caches.
- Final source-stable targeted run passes across all webcast packages, including shared Node/operator assembly (root5.101s). Focused policy/mount/operator tests70 pass; reader inventory and diff check pass. Committing corrected code before exact-head host conformance and the next recorded full check.

### 2026-10-21 — corrected-head host conformance

- On committed8be34c6fab5b1cc1c2431a7cadb1da915b2e793d, WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'SketchLive|OperatorSketchLive' -count=1 -v exited0 in9.541s. Scene privacy2.16s; actual console drawing2.51s; open-contact handoff label1.80s; two viewers2.91s. All assertions executed, including named-frame/blob denial, moved original/replacement, three pre-up pressure samples, replay/Undo/Redo and old-label/new-holder source continuity.
- Supplied this fresh host evidence to BR-5 through sdlc finding dispute; the next mandatory review must explicitly dispose of the unverifiable finding. The operator console smoke remains unperformed and staging/landing held.

### 2026-10-21 — classified fixture flake and independent close review

- Corrected-head full suite ran after1238s queue and finished10m22s. Every test passed except unchanged couchcore.TestProvisionFixtureIdentityUsesOneGitProcess (exit42, child shim could not read /dev/fd/3 before real Git). Attempt53424823b0e2ed3099f5dab5c5a415b2, raw log .git/sdlc/test-stats/failed/53424823b0e2ed3099f5dab5c5a415b2.log, SHA256 prefix48f54c92822d. All webcast and shared Node tests passed.
- Same exact ecef5f7af3515fead7c02d6464ae112b16498fe1: targeted fixture test passed (root0.533s; independent single0.13s and10/10 in3.062s). Registered unclaimed pair#767 with fail→pass evidence. Underlying FD cause is unproven; this is flake classification, not a fixture fix. Hub filing receipt submitted.
- Started SDLC recorded full rerun on the same head, retained lease-1901225109; queued sixth behind existing runs. This binary launches a fresh suite for this recovery. Close review can proceed independently with the classified same-head failure and targeted pass explicit; recorded status remains failed/pending until the rerun qualifies. No stage, landing or claim of a clean recorded pass yet.

### 2026-10-21 — close round2 contract correction

- Review disposed BR-1 through BR-4 as addressed and withdrew BR-5 based on exact executable host evidence. It independently reverted the browser fix and reproduced25 failures, validating the lifecycle/queue regressions. All414 Node tests and targeted Go/race checks passed in review.
- New BR-6 found my width alignment chose4 rather than the approved3 CSS px at #750 plan line200 and this plan's focused API. Restored producer and optimistic width3, corrected the actual-producer fixtures and format atlas, and retained generic historical width4 model fixtures. RED tests now pin contract pressure widths and mouse/touch3; this corrects the earlier consistency choice, not the approved contract.
- BR-7 repeats the Core concepts family: effectful canvas and random journal initialization were mislabeled pure. Applying a complete function/type/effect classification sweep, rather than moving only the named rows.
- Source changes invalidate the queued ecef5f7a recovery run; after committing the fix, replace only this checkout's obsolete lease-1901225109 with a recorded run on the corrected head. No other checkouts or suite holders are touched.

### 2026-10-21 — approved-width host conformance

- On committed bccbfa399b9a829f347ee21d22a1c45323fc5312, all4 live Chrome tests pass (package6.547s): scene privacy1.18s, console drawing2.38s, open-contact handoff label1.09s, two-viewer pressure/replay1.74s. Source is unchanged after this measurement. Targeted Go/Node/assembly and reader inventory also pass.
- Cancelled only obsolete ecef5f7a lease-1901225109 through SDLC after the executable width correction; no status was recorded for that queued cancellation. Started a fresh recorded affected/full fallback on bccbfa399. TL confirmed the targeted saved-failure recovery gap is open ariadne#463 and the full rerun is supported. Retain the corrected-head runner.
- Entering the next close review independently while that recorded check is pending, with the pending status explicit. Both new findings are corrected; stage, operator smoke and landing remain held.

### 2026-10-21 — final recorded suite and live operator acceptance

Final executable source 4f0f3228eec3ae76481f3bc120f22c216bc38f7e passed the recorded suite in 10m10s (attempt 0177f6151eb0fffb665ada109d4bbe2c, log hash e63ce55405ab); staged revision 95206a8d02110af2bd98dab1fd1d2d6c50117956 differs only in reviewed prose/evidence. Stage/build verified all four binaries at that revision with vcs.modified=false. After restarting Couch onto that build, the operator accepted the console flow: "ok, those steps work" — draw shape 1, click Note between Redo and Done during the active sketch, type and Save label to finish, then repeat for shape 2. Authenticated TL delivery 4a553a59-790a-4211-903c-828bec82e32d records this real-environment acceptance and authorizes landing. Editing a completed label remains the approved separate #783, not part of this acceptance. TL directed #771's already-boarded train254 to finish first; PR527 is now merged and its retained cleanup finished. Proceed with this accepted #752 landing under the renewed grant; hub owns the subsequent pair:0 rebuild and stage clear.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Derived after plan-quality round1 passed without blockers. Calibration is tagged stale by `sdlc estimate-source`, so this is provisional. Single repo; includes both UI surfaces, owned/live smoke, documentation and final review. Browser pure modules/screens use the nearest greenfield-module/TUI-screen primitives; this is an explicit analogy, not a new vocabulary item.

Each row below corresponds in order to an item in the block. Design uses ×0.2 for the approved concrete plan. Library availability: standard JSON/crypto/context/HTTP and existing NextSketch, Hub, controller, Writer and Model/Renderer cover the greenfield foundations, so those raw greenfield design picks are halved before the spec discount. API integration design picks are 1–1.5 hours before the spec discount. All implementation values are already ×0.40 of the v2/v2.1 table; familiarity1.0 because this extends the qualified existing stack. One external API discovery budget covers Chrome/CDP. No separate queue-wait estimate or double implementation scaling.

| Component | Design h | Scaled impl h |
| --- | --- | --- |
| slice specification/approved architecture | 0.20 | 0.08 |
| strict command union/digest | 0.10 | 0.20 |
| receipt queries and additive cancellation fact | 0.04 | 0.16 |
| drawing witness | 0.15 | 0.28 |
| source preflight | 0.15 | 0.28 |
| Board lease/ownership | 0.20 | 0.32 |
| settled recorder writer/barrier | 0.30 | 0.60 |
| HTTP owner routes | 0.20 | 0.40 |
| private source/asset transfer | 0.30 | 0.60 |
| operator journal/bootstrap | 0.15 | 0.28 |
| render patch projection/painter | 0.10 | 0.24 |
| pointer interaction/recovery policy (browser screen analogy) | 0.20 | 0.40 |
| shared controls/label coordinator (browser screen analogy) | 0.20 | 0.40 |
| held scene integration | 0.30 | 0.60 |
| opaque Chrome child scene | 0.30 | 0.60 |
| placement source identity | 0.04 | 0.16 |
| module assembly/subscriptions | 0.06 | 0.16 |
| two-viewer/console acceptance integration | 0.20 | 0.60 |
| Chrome CDP isolation discovery | 0.00 | 0.24 |
| controls/invariant documentation | 0.03 | 0.08 |
| one final boundary | 0.04 | 0.20 |

Design subtotal3.26h ×1.15 thorough-plan buffer + implementation6.88h ×1.0 =10.629h, rounded to **10.63h**.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.2 impl=0.08
item: greenfield-go-module design=0.1 impl=0.2
item: smaller-go-module design=0.04 impl=0.16
item: greenfield-go-module design=0.15 impl=0.28
item: greenfield-go-module design=0.15 impl=0.28
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.3 impl=0.6
item: api-integration design=0.2 impl=0.4
item: api-integration design=0.3 impl=0.6
item: greenfield-go-module design=0.15 impl=0.28
item: greenfield-go-module design=0.1 impl=0.24
item: tui-screen design=0.2 impl=0.4
item: tui-screen design=0.2 impl=0.4
item: api-integration design=0.3 impl=0.6
item: api-integration design=0.3 impl=0.6
item: smaller-go-module design=0.04 impl=0.16
item: cross-cutting-refactor design=0.06 impl=0.16
item: api-integration design=0.2 impl=0.6
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 10.63
```


## Revisions

### 2026-10-21 — TL-directed pre-smoke close review

Implementation and automated verification can close locally before the deferred real console smoke, per ops:0 delivery b1d858cd-0dfc-445b-a623-c3e78b62a282. The Done-when requirement for smoke before landing is unchanged. The operator restart owns staging admission; stage and landing remain held.

### 2026-10-21 — implementation-location correction

BR-4 also corrects the Spec entry-point list to actual implemented files and marks existing browser/placement interfaces as reused. This changes locations, not required behavior.

### 2026-10-21 — staging released after restart

TL delivery1a61c2c4-9b66-4b57-bc40-416fef0b2cd0 releases the restart/staging hold and prioritizes this item. Stage/build waits only for operator-owned Go-cache reclamation. TL must still perform and accept the real console smoke before landing; this revision changes admission timing, not behavior or acceptance scope.

### 2026-10-21 — operator-smoke disabled controls

Authenticated TL delivery d446a25b-56c0-471e-b371-92b2ee676110 adds visible recovery guidance for disabled Sketch and disables Done when no sketch/preparation can finish. Operator was blocked with annotation off; Chrome's 👆 is in Couch's bottom status row after Chrome's ↗, distinct from LIVE's earlier screen-broadcast 👆; it has no dedicated keyboard shortcut. Shared toolbar currently binds Sketch disabled but supplies no reason, and Done has no disabled binding. Derive both from existing authority/lifetime, with console-specific recovery text; no new authority or persistence state. The plan revision records regression and revalidation requirements. Existing staged revision stays available while this acceptance fix is prepared; prior smoke is not accepted.

### 2026-10-21 — active toolbar input routing

Operator drew two strokes but could not finish because toolbar buttons disappeared (TL delivery185e0754-d052-4f48-8e87-485c63a8537b). Escape followed by label Enter was relayed as the existing safe exit. Actual Chrome regression RED in1.762s, diagnostic RED2.570s: all Sketch/Undo/Redo/Done button center hit tests return sketch-layer; buttons are static/z-auto, under positioned canvas/held scene. Explicit toolbar stacking and native pointer-click regression replace the previous controller-only acceptance checks. Earlier availability correction passed482 Node tests and actual Chrome3.986s; full close refused the noon main-integration precondition before dispatching review. Integrate current main and reverify after both smoke blockers are corrected; no accepted smoke or landing yet.

### 2026-10-21 — console trust bounded approval

TL22603f68 andcbf33121 approve console drawing independent of viewer annotation On/Off, retaining the existing single-holder rule and generation/board/driver lifetime fences. Three seams must agree: CDP projection, cast ingress factory, and opaque admission Check used by service/scene-open. Ordinary viewer admission and annotation-off global cleanup remain unchanged. Concurrent lanes/new recording version/selective viewer clearing are deferred to an unclaimed follow-up; no concurrency implementation is authorized in this item. The durable plan revision owns regression scope. Toolbar native-click conformance passed2.277s before main integration; required integrated final verification remains pending.

### 2026-10-21 — focus and unfinished Note acceptance

TL9596d9a8 reports console flashing and label typing interruptions. Source audit found periodic5s screenshot privacy suppression hides the entire operator host, removing textarea focus without losing its value. TL72ca720e approves deferring optional background captures throughout sketch/label UI when capture without visible host toggling is too large; preserve suppression for captures actually taken and file full no-flicker work separately. Add native focus/caret/typing and idle/remote-update regressions, no-op identical control/authority refreshes, and a Note affordance for unfinished text. Completed edits are deferred to unclaimed #783; viewer annotation-based drawing is recorded in #782, outside this issue. Console trust targeted Board/HTTP tests pass0.201s/0.240s and actual Chrome drawing/handoff passes4.032s after waiting for both permission and live projection. No final smoke acceptance yet.

### 2026-10-21 — smoke corrections verified before final gates

Native capture RED2.522s demonstrated lost focus/caret under screenshot suppression. Background acquire now defers atomically while existing sketch/preparation or note/label editor owns UI; deferred work emits no pixels or failed-frame evidence, and admitted/explicit captures keep suppression/check/release. Native conformance passes12.976s across console drawing/handoff/private scene, two viewers and ordinary operator annotations: toolbar actions use real pointer clicks; annotation stays off for console; drawing and label defer captures, label caret/focus/blur survive repeated opportunities and source updates, native insertion yields AxB without refocus, and captures resume after accepted label. Recorder package passes4.538s; Board5.065s/repolint11.387s; all484 Node tests pass1.243s; inventory/diff checks pass. Initial portable fixtures lacked acquire-background/getAttribute; stateful fake/control DOM now model them and all checks pass. #782 annotation-viewer revision is published; #783 completed-text editing and #784 full no-flicker capture are unclaimed. Existing staged build is older; fresh review, recorded pass and restage precede accepted real smoke/landing.

### 2026-10-21 — close BR-11 prose correction

Full close on4f0f3228 returned FIX-THEN-SHIP: prior BR1–10 disposed, no critical findings; new Important BR-11 identifies stale README console annotation prerequisite, old Label unfinished sketch name, missing Note affordance and blanket screenshot statement. Swept the sketch usage/capture passage against current source and updated it with human-review markers; no executable change. Follow binary fix-then-reconcile protocol, without a duplicate close. Retained recorded suite lease-3430025668 remains on the same final executable source. Restage follows reconciliation; real operator smoke still precedes landing.
