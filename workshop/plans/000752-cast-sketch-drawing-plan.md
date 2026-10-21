---
collaboration-mode: discussion
---

# Cast sketch live drawing implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-subagent-driven-development for bounded parser, authority and browser tasks; keep writer/source integration in the coordinating session. Steps use checkboxes. This plan has one final SDLC review boundary; task numbers are not milestones.

**Goal:** Let the casting-Chrome operator or current granted driver sketch one held recorded moment, streaming pressure-aware paths to observers, with acknowledged recovery and required labels.

**Architecture:** The existing driving owner supplies a revocable drawing permit. The ordered recorder remains the sole transition/persistence authority, using #751's reducer, receipts and byte reserve. Both surfaces mount the same browser tool and renderer; a separate acknowledged source scene supplies frozen geometry while live state continues advancing.

**Tech stack:** Go recording/board/recorder/CDP, browser ES modules, Node tests, owned Chrome fixtures.

## Scope and existing evidence

- Approved requirements are #752 and the operator approval in [#750](../history/plans/000750-cast-sketch-plan.md). #751 landed as PR505 with the recording core, preserving codec, snapshot catch-up, terminal reserve and geometry. Keep its strict ctl validation and unchanged non-sketch catch-up prerequisites.
- Deliver BOTH console Chrome and granted driver; pointer/view links watch, pointer links retain pointing. No image-export CLI, mock-version scoping or device acceptance claims belonging to #753/#756/#754.
- The real console input smoke in this issue is mandatory. Headless owned-page conformance does not replace the TL-arranged live check.
- Current `Emit` is enqueue-only. Reuse `WritePrepared`/`ObserveNote`, `WriteControl` and `acknowledged` patterns for settled writes and barriers. Never interpret reducer acceptance as sink confirmation.
- Current drive capability generation changes on capability toggles, not A→B→A holder turns. Current `OperatorView` coalesces snapshots; it cannot carry full sketch history per point. Current module assembler rejects class exports and operator `onRecord` has one callback. These are enumerated integration work, not reasons to omit a surface.

## Core concepts

Paths below are relative to `cmd/internal/webcast/`.

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| Strict sketch command union, normalized command digest, reply values | `board/sketch_wire.go` | new |
| Drawing authority projection and permit identity values | `drive/drawing.go` | new |
| Narrow immutable sketch receipt/metadata/top queries | `recording/sketch_state.go` | modified |
| Bounded source-prefix candidate and scene byte accounting | `recording/sketch_source.go` | new |
| Browser interaction and recovery policy | `web/sketch-policy.js` | new |
| Pressure path painter and render-only patch projection | `web/sketch-render.js` | new |
| Operator delta cursor/transfer protocol | `board/sketch_delivery.go`, `web/sketch-delivery.js` | new |

The command union owns ingress shape, not persisted shape: the client supplies node identity and observed box, never trusted author, owner, selector/path/text, source time or digest. Digest the normalized strict command (including operation, sequence and origin epoch); store that digest in the event. #751 independently compares actual final event semantics on reducer retry. A begin retry resolves its original server-assigned sketch/source before any current-source reconstruction.

Drawing authority is a projection of existing driving ownership, not another grant machine. A permit captures one owner/epoch and a controller-owned revocation witness. Metadata label/status credentials are distinct from drawing permission; they cannot mutate geometry. Narrow receipt and stack queries return owned small values in O(1), never clone all points.

Render patches contain already accepted facts: new point chunks and authoritative per-stroke visibility/sealed/truncated flags, sketch phase/label and metadata. The display projector applies those facts by identity; it does not reproduce Undo/Redo legality or maintain another authority FSM. Compare its output with the reducer projection at every test prefix.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Controller drawing witness and capability publication | `drive/drawing.go`, `drive/controller.go`, `drive/controller_mem.go`, `drive/capability.go` | new / modified | driving holder, acknowledged ctl/reset, held-input release, seat lifetime |
| Board sketch service and source lease | `board/sketch.go`, `board/sketch_authority.go`, `board/board.go` | new / modified | annotation proof, authenticated principal, frozen State, ordered writer |
| Settled sketch writer/reference fake | `recorder/sketch.go`, `recorder/recorder.go`, `board/writer.go` | new / modified | actor queue, mirror, sink, barrier and expiry timer |
| Source-prefix provider | `recording/hub.go`, `cast.go`, `browser.go` | modified | accepted Hub log and cast-owned fallback Hub |
| Sketch HTTP routes | `server_sketch.go`, `server.go`, `server_drive.go`, `server_capabilities.go` | new / modified | castlock, seat authentication, bounded body/asset exchange |
| Shared tool, held scene and viewer adapter | `web/sketch.js`, `web/sketch-scene.js`, `web/sketch.css`, `web/viewer.js`, `web/drive.js`, `web/annotate.js` | new / modified | Pointer Events, renderer, live connection and tool transitions |
| Shared anchor placement | `web/annotation-placement.js`, `web/render.js` | modified | current box/clip and recorded source-document identity |
| Privileged console adapter | `operator.js`, `operator_script.go`, `chrome_operator.go`, `recorder/operator.go`, `board/operator.go`, `board/operator_view.go` | modified | isolated binding, excluded host, private scene/assets, bounded delivery |
| Opaque private scene session | `chrome_sketch_scene.go`, `sketch_scene_script.go` | new | scoped child CDP world, frozen Renderer, loader/context identity and transfer handles |
| Existing owned browser doubles | `fake.go`, `fake_drive.go`, recorder test harness | modified | same new sketch/source writer interfaces and held sink outcomes |

Use the existing stateful `board.Writer` and fake browser/CDP harness to model queue start, persisted records, uncertainty and late replies. HTTP integration boots the real owned Server with Hub/Writer and fake browser. Live conformance uses disposable Chrome profiles and recorded assets. Storage stays in the existing portable recording directory; no new sidecar or user-profile dependency.

## Authority, commands and ordering

### One lifetime resource

`drive.DrawingAuthority{Epoch uint64, Holder string, Ready bool}` is ephemeral capability data. Add `DrawingAuthority()`, `AdmitDrawing(expectedEpoch, seatKey)` and `AdmitOperatorDrawing(expectedEpoch)`; opaque `DrawingPermit.Check()` reads only its revocation witness, without taking controller locks.

Close the witness/increment epoch before a holder-changing ctl or reset starts. Renew it only after ctl persistence AND held-input release succeed. Unknown/failed cleanup remains unavailable. Pending-only request/deny/withdrawal keeps the epoch/witness and does not interrupt sketches. Close also revokes the witness. Normal drive input semantics remain unchanged.

Publish this projection through the existing ephemeral capabilities feed and isolated operator permission delivery, waking the existing subscription when ready/epoch changes. A client captures the epoch when it creates an operation; retries never refresh it. Reject stale-origin requests even when A has become holder again before ingress. Ingress checks current credential/epoch before body reads; final actor checks witness, original annotation generation and actor control holder. No Board command lock or controller lock is acquired from the actor.

### Focused request and settlement API

`board/sketch_wire.go` defines operation-specific parsers for begin, stroke-begin, points, stroke-end, Undo, Redo, label, status and source/asset reads. `SketchCommand` holds operation ID, safe sequence number and origin epoch; begin sequence is zero. Reuse #751 numeric/Unicode invariants through a shared focused validator rather than copying escaped-surrogate logic.

- Begin: operation/sequence/epoch, current segment and palette index. Server chooses sketch ID, author, owner, source time/tab/viewport.
- Stroke-begin: sketch/stroke IDs, frozen node ID and observed positive box. Server anchors against retained source State with `notes.Structural.Anchor`, fixed base width 3; it accepts no selector evidence.
- Points/end/Undo/Redo/label: only their strict owned fields. Enforce #751 chunk, point, label and sequence limits; reject unknown keys, duplicate keys, malformed Unicode and server-owned fields.
- `SketchReply` has operation/sketch/nextSequence, outcome `accepted|refused|unknown|not-observed`, a bounded stable error code, and optional source descriptor. HTTP and isolated binding return the same semantic reply.
- `WriteSketch(ctx, command, admission)` returns only after the started actor operation settles and `r.write` confirms the sink. Cancellation may cancel queued work before start; a started effect owns and records its bounded outcome even if the caller leaves.
- Status enqueues a healthy writer barrier. `not-observed` is nonfinal while an older command could still run. Only barrier-proven refusal lets the client release/reuse its operation slot. Unknown sink results remain unknown on exact retry and never append twice.
- Retain accepted receipts through the existing bounded reducer; retain only one active/last refused slot per sketch plus bounded begin slots. Cap outstanding begin/transport slots at 64; capacity refusal happens before reserving work. Old labels have their own sketch slot and do not block the current drawer.

### Exact source preparation

Add `Hub.CandidateSketchSource(begin, prefixLimit, stateLimit)` and inject that source interface into the writer. Reuse a configured Hub. If `NewLive` is absent/custom and lacks the source interface, the cast creates one private `NewHub(0)` in the existing Tee lifetime. Do not add a second log for the normal Hub path or another retained-history implementation.

On the recorder actor: flush pending earlier records, verify writer health/admission, construct the prospective strictly timestamp-fenced begin, take the current Hub prefix and append that SAME prospective event privately. Encode/validate the exact candidate before committing begin. `ReduceCatchUp` runs once to create independent source State. If size/shape fails, emit nothing and advance neither clock nor budget/lease. After actual sink success, acknowledge this exact source; do not re-read Hub after ack, where a later segment could already have compacted it.

The 16 MiB source prefix includes existing CatchUp notes/sketch/bootstrap content; never omit history to squeeze the limit. The separate 8 MiB source DOM/state budget measures a compact scene accounting projection in one traversal of nodes, styles and current auxiliary state (not Canonical's node×sheet traversal). Source assets are immutable recorded hashes and read via the existing cast asset store. A source handle is owner/cast/sketch-bound; it is never authorization by itself.

Keep one active source/lease cast-wide. Accepted mutation updates the two-minute idle deadline; status/read traffic does not extend it. Expiry enqueues an identified `sketch-interrupt` through the same writer and reserve. Recheck deadline on the actor to resolve a racing chunk. Handoff/navigation/end/clear retire source; metadata receipts remain. Same-document pause keeps source and reconciliation, seals the accepted partial stroke, and permits drawing again only after resume/reconciliation. Timer creation/stop/join belongs to cast/recorder lifetime.

## Browser and privileged delivery

`mountSketch(doc, win, view, options)` is shared by viewer and casting Chrome. `createSketchPolicy` owns `off|preparing|drawing|label-required|reconciling|paused|interrupted` tagged states. It permits one command in flight, FIFO chunks at 50 ms, at most 64 points/chunk and 16 queued chunks. At capacity stop the gesture visibly, retain accepted evidence and reconcile/seal; never silently discard samples or send them under a new operation identity.

The tool appears for eligible console/current driver. Enter prepares and awaits scene readiness before accepting input. It suppresses normal remote page input while retaining the driving seat. A common tool coordinator intercepts transitions away from sketch, ends any open stroke, requests a nonblank label, and completes the transition only on confirmation. Escape returns to the same sketch. Undo/Redo wait for stroke-end acknowledgment and use only the current unfinished sketch. A handoff releases the drawing mode promptly and preserves a separately reachable old-author label action.

Pointer handling captures only the contact pointer, ignores pen hover, consumes bounded coalesced samples and preserves normalized pressure. Mouse/touch use pressure 1; pointer-up retains the last contact pressure; cancel seals accepted partial work with truncation. The existing stroke-end cannot express this fact: add optional `truncated: true` to `SketchStrokeEnd`, using `omitempty` so absent/false retain their existing wire/digest. First sealing uses the flag; already sealed retries never erase interruption evidence. The sketch remains usable for another stroke, Undo/Redo and label. Update both codecs, semantic identity, core seal and snapshot receipt replay; preserve this new unknown member in a frozen #751 reader. Optimistic samples are keyed by stroke/chunk and replaced by echo, never doubled. Use shared `projectPoint`/`pressureWidth` and round joins/dots; keep fading pointer trails separate.

`createSketchScene` owns an independent Model/Renderer, iframe, recorded asset resolver and readiness/disposal. Initial and reconnect scenes reduce the acknowledged begin prefix. Freeze source scroll, viewport and animations, including nested frames/scrollers; viewport fitting changes only the display transform. Live connection continues applying DOM/control/sketch records, suspending only its page paint while the held scene is active. On exit destroy scene/blobs and request ordinary fresh catch-up. Placement shares the existing confidence/clip traversal and generalizes original-node proof to sketch source document identity; absent/ambiguous anchors keep their historical list item.

The operator reaches the same Board service through the excluded isolated world. Keep sketch request correlation separate from annotation's document-scoped ledger so metadata-only pending labels survive a handoff/navigation while credentials live. Page scripts receive no binding, seat key, private handle or recorded asset bytes. Reuse the existing Chrome delivery worker/timer and extend its module assembler narrowly to support the actual shared Model/Renderer export forms, with execution tests; do not import the entire replay UI.

For the console held page, use an opaque `data:` shell with `sandbox="allow-same-origin"` and no scripts. Chrome's `SceneSession` finds its frame ID through `DOM.describeNode`, uses the owning Page session (attaching an OOPIF only when needed), and creates one scoped child isolated world there. Inject the actual shared Model/Renderer into that world through bounded CDP transfers; do not place history/assets/bindings in the shell URL or page messages. Its nested about:blank frames share its opaque origin, so recorded names and CSS remain intact while the casting page cannot read their DOM/assets even through named targeting.

The parent isolated world retains toolbar, label draft, gesture controller and overlay. The child owns only frozen Model/Renderer plus read-only hit-test/placement methods. The shared scene view accepts async hit-testing; collect the bounded current gesture while its first anchor is acknowledged. Parent↔child calls go through the existing trusted CDP worker, not public `postMessage`. `SceneSession` binds parent context, frame/session, loader, child context and source lease; context/loader retirement invalidates all responses. Never reinstall a trusted binding after page-caused navigation. Exclude scene targets from both recording observation and normal drive input. Preflight shell/module readiness before committing console begin; a CSP block is an explicit source-preparation failure, with no page-policy bypass.

`OperatorView` adds only drawing authority and journal cursor. Board appends accepted render patches/contextual changes to one 256-entry/1 MiB journal, encoding new data once. Chrome drains at most 32 records/64 KiB per step from its ACKed cursor. New context/gap uses a render-only owned bootstrap captured atomically with high-water; it contains paths/flags/labels, not reducer receipts. Its 36 MiB cap follows `admitted history bytes + 128×stroke count + 256×sketch count +1024 <35 MiB` at 16384 strokes/256 sketches, rounded upward. Each header/point/label occurs once, variable IDs are not duplicated outside their headers; test the actual encoding against that inequality. No accepted item may be silently dropped. Receiver ignores exact duplicate batches and rejects a gap before mutation.

Private source/bootstrap/assets use 32 KiB raw chunks (about 44 KiB base64) with exact handle/offset/total, one outstanding transfer per context and at most four contexts. Only hashes named by that source may be read. Process one bounded transfer step per worker turn so permission/reply delivery cannot starve. Context retirement, clear/end, lease expiry and scene exit release handles/blobs; late chunks cannot recreate them.

## State and recovery contract

Actions: continue drawing, retry exact operation, reconnect, resume cast, enter label, start new sketch, Undo, Redo, inspect recording.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Begin source is being admitted/built | Preparing held scene, input disabled | Retry exact operation | No accepted work |
| Command started, reply lost or sink uncertain | Confirmation pending, gesture stopped | Retry exact operation | Unsent local samples if tab closes; accepted history retained |
| Writer barrier proves refusal / queue at bound | Interrupted gesture with accepted prefix visible | Undo then continue drawing | Rejected/unsent samples only |
| Same-document pause | Paused, scene and pending command retained | Resume cast | No accepted history |
| Disconnect within exact holder/session lifetime | Reconnecting | Reconnect and retry exact operation | Unsent samples if client closes |
| Holder changes, reset, expiry or source replacement | Old sketch interrupted; pending label remains separately reachable | Enter label; start new sketch as eligible holder | Unsent samples and unentered label |
| Session/cast ends before label | Explicit unfinished intent in history | Inspect recording | Unentered label; no ownership takeover |
| Blank label / label acknowledgment lost | Label required / confirmation pending | Enter label / retry exact operation | None within retained client draft |
| Source/scene limit or asset/readiness failure | Preparation unavailable with reason | Retry exact operation after source changes | No begin emitted for preflight failure |
| Anchor absent/ambiguous on live view | Placement unavailable, historical item retained | Inspect recording | No stored geometry |
| Operator journal gap/context replacement | Resynchronizing overlay; drawing disabled | Reconnect/bootstrap | No accepted history; local unsent samples may be lost |
| Annotation clear/revocation | Current overlays and source retired | Inspect recording before clear | Local unsent work; recorded earlier timeline remains |

The most dangerous interleaving is a holder handoff while an old chunk awaits the actor or sink: the witness must fence old work before enabling the next drawer, while an old label must not hold that transition hostage. Force admission→queue→sink→reply rendezvous in tests. No callback survives cast Close; recorder started effects are joined, delivery is single-worker, and all timers/scene requests have cancellation/retirement owners.

## Operating envelope and counted verification

| Growth driver / interaction | Bound and basis | Required exceeded behavior / cost |
| --- | --- | --- |
| Pointer transport | 50 ms cadence, 64 points/chunk, 16 queued chunks; approved cadence plus explicit bounded backpressure | Stop gesture visibly, reconcile same operation; O(new samples), no full history scans |
| Request body/concurrency | 16 KiB strict mutation body; existing `admitBody` deadline/rate/in-flight controls, 64 outstanding service slots | Refuse before allocation/effect; labels retain reserved terminal capacity |
| Sketch history | #751: 4096 pts/stroke,16384/sketch,64 strokes,256 sketches,32 MiB history | Existing reducer and writer reject growth; exact retries charge nothing |
| Source preparation | 8 MiB DOM/state,16 MiB full prefix; approved bounds | Refuse before begin commit; one prefix copy/reduction per begin |
| Authority/status | One current driving witness; O(1) receipt/top/permit queries | Epoch mismatch refuses; polling never clones geometry |
| Operator delivery | Journal256/1 MiB; batch32/64 KiB; raw transfer32 KiB | Gap bootstrap; bounded worker step, no per-point full snapshot |
| Scene/asset residency | One held source cast-wide; four transfer contexts; two-minute mutation-idle upper bound | Retire on owning lifecycle; assets remain existing recording files |

Count N/2N samples/chunks at 8/16 and 64/128; assert proportional visits, point appends and encoded journal bytes, zero source preparation/full-history snapshots on the chunk/status hot path. Count anchor candidate scans using existing placement instrumentation. Scene preparation may be linear in retained prefix/source bytes; it must not replay that prefix per chunk. Disk throughput and GPU performance are measured in the owned Chrome fixture; there is no new external service or new durable storage family.

### Function verification strategy

| Production function / family | Risk class | Mechanical guard and independent oracle |
| --- | --- | --- |
| `ParseSketch*`, `SketchCommandDigest` | Hostile JSON/Unicode/numeric types and request identity | Fuzz each closed union; normalized typed digest corpus; duplicate/server-owned-key negative controls |
| `SketchState.Receipt`, metadata/top queries | Mutable authority leaks / accidental history scan | Mutate returned small values, then compare reducer; counted map/lookups at N/2N |
| `AdmitDrawing`, `AdmitOperatorDrawing`, permit `Check` | ABA/stale origin and release-in-progress | Held acknowledged ctl/release in real controller; old permit remains closed after A→B→A, pending-only negative control |
| `Hub.CandidateSketchSource`, scene-size accounting | Wrong source/time, size preflight mutates state, quadratic work | Generated same-clock record prefixes; compare candidate reduction with accepted begin prefix; exact limit boundary and counted one-pass visits |
| `WriteSketch`, `ObserveSketch` | Cancel/retry/unknown effects and stale authority | Stateful Writer/Recorder contract with explicit queue/sink/reply gates; count physical writes and compare original recording replay |
| lease expiry callback | Timer loses a last mutation / resurrects retired source | Inject clock and rendezvous before actor execution; generated mutation→expiry orderings; one terminal receipt and zero surviving timers after Close |
| `SketchStrokeEnd` seal and snapshot receipt replay | Cancellation evidence lost or Undo stack duplicated | Shared Go/JS event-sequence corpus, absent/false compatibility and conflicting exact-retry controls |
| `OperatorSketchSince`, bootstrap / receiver | Coalescing, duplicate ACK and gap corruption | Generated cursor/overflow/ACK sequences; compare render projection at every accepted prefix; bounded bytes and zero bootstrap on normal points |
| `createSketchPolicy`, FIFO sender | New identity after unknown / terminal input races | Generated user/network/ctl sequences; one active operation and same-origin retry invariants; bounded queue negative controls |
| shared pointer sampler/painter | Pen pressure loss, hover/palm input, duplicate optimistic echo | Recorded synthetic PointerEvent/coalesced corpus and shared geometry oracle; count new samples/painted segments at N/2N |
| `createSketchScene`, scene/asset adapters | Display differs from acknowledged moment / leaked or late handles | Two independent Models with deliberately divergent live source; owned Chrome render/scroll/animation checks; late chunk/dispose and adversarial-page controls |
| sketch HTTP / isolated binding adapters | Credential/body/epoch confusion | Real Server with stateful fake writer, body-read spy and cross-owner negative requests; actual isolated Chrome binding/frame probes |

The counters are test seams around named traversals/encodes, not wall-clock assertions. Public-interface tests replay the recorded result; they do not merely assert that a method was called.

## Implementation tasks

### Task 1 — strict protocol and narrow model queries

**Files:** create `board/sketch_wire.go`, `board/sketch_wire_test.go`; extend `recording/sketch_state.go`/tests and focused Unicode validation helper; add `web/sketch-policy.js`/tests.

- [x] Write `TestSketchCommandUnionRejectsOwnedFields`, `TestSketchCommandUnicodeAndBounds`, `TestSketchReceiptQueriesOwnedAndConstantWork`; seed parser fuzz with all commands, duplicate keys, unsafe integers, nonfinite triples, lone surrogates and valid pairs.
- [x] Run `go test ./cmd/internal/webcast/board ./cmd/internal/webcast/recording -run 'SketchCommand|SketchReceiptQueries' -count=1`; observe missing parser/query failures before implementation.
- [x] Implement strict command values/digest/replies and O(1) immutable receipt, metadata and stack-top queries. Keep existing persisted event behavior except the explicit additive truncation fact below.
- [x] Add optional stroke-end truncation with red tests for cancel→next stroke/Undo/Redo/label, absent/false/true/null/type validation, same-operation conflict, old-reader roundtrip, snapshot receipt replay and the terminal reserve. Extend/regenerate the shared Go/JS corpus with cancellation. This is an additive fact required by the approved pointercancel behavior.
- [x] Add generated browser policy sequences for unknown/refused/exact retry, label leave/Escape, pause and handoff. Run focused Go/Node tests, then commit this bounded task.

### Task 2 — driving permit resource and UI authority feed

**Files:** create `drive/drawing.go`/tests and `board/sketch_authority.go`; extend controller/capability, wire capabilities and server tests.

- [x] Write held-transition tests `TestDrawingPermitABARetiresOldOrigin`, `TestDrawingPermitWaitsForControlAndRelease`, `TestPendingControlKeepsDrawingPermit`, `TestDrawingResetAndCloseFence`, `TestSketchAuthorityArrivesBeforeOrAfterCtl`.
- [x] Run `go test ./cmd/internal/webcast/drive ./cmd/internal/webcast -run 'DrawingPermit|DrawingReset|SketchAuthority' -count=1`; verify red, implement the sketch-only witness and existing-feed projection, then rerun.
- [x] Verify console-local availability with remote drive disabled, failure-frozen release, view/pointer denial and current-seat authentication. Preserve normal driving regression tests. Commit.

### Task 3 — acknowledged source/writer/Board service

**Files:** create `recording/sketch_source.go`/tests, `board/sketch.go`/tests, `recorder/sketch.go`/tests; extend Hub, reference Writer, cast/browser/fakes, board lifecycle.

- [x] Write source tests for exact begin fencing, following same-clock mutation, later navigation compaction, independent8/16 MiB boundaries, zero-mutation refusal and absent/custom live factory fallback.
- [x] Write generated operation sequences and held queue/sink/reply tests through BOTH reference Writer and Recorder: accepted/refused/unknown retries, same-ID conflicts, handoff variants, old-owner metadata label, clear/pause/resume/end, expiry racing last mutation and Close joining work.
- [x] Run `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder -run 'SketchSource|SketchWrite|SketchLease|SketchHandoff' -count=1`; prove red before adding source/settlement service.
- [x] Implement writer preflight→commit→flush→receipt, exact source retention/lease and healthy-barrier status. Anchor only against frozen State. Run N/2N counters and existing note/control ordering tests; commit.

### Task 4 — routes and private operator delivery

**Files:** create `server_sketch.go`/tests, `board/sketch_delivery.go`/tests, `web/sketch-delivery.js`/tests; extend cast locks/routes, recorder/operator, chrome_operator and operator assembler.

- [x] Write HTTP tests proving link/seat/epoch refusal before body reads, per-owner status/source isolation, bounded source/asset chunk offsets and terminal error mapping. Use stateful Server+Writer+Hub stack.
- [x] Write operator tests for trusted-context admission, new-context late replies, metadata label after document retirement, coalesced journal drain, duplicate ACK, gap bootstrap, worker fairness, exact maximum bootstrap and handle cleanup.
- [x] Run `go test ./cmd/internal/webcast/... -run 'SketchRoute|OperatorSketch|SketchDelivery|OperatorScript' -count=1` and `node --test tests/webcast/sketch-delivery.test.mjs`; observe red, implement, rerun plus existing operator/castlock suites, commit.

### Task 5 — shared held scene, painter and tool on both surfaces

**Files:** create `web/sketch.js`, `web/sketch-scene.js`, `web/sketch-render.js`, `web/sketch.css` and focused tests; extend viewer/drive/annotate/placement/render/operator and embedded asset registry.

- [x] Write pointer policy tests for coalesced pressure, hover, capture, cancel/up, mouse/touch fallback, optimistic echo, bounded backpressure and current-sketch Undo/Redo. Test renderer output against #751 shared geometry corpus.
- [x] Write held-scene tests proving source/live independence at initial prepare AND reconnect, nested scroll/viewport/animation hold, asset readiness failure/disposal and restoration to fresh live catch-up. Generalize source identity proof with negative cross-document placement controls. Add opaque SceneSession loader/context fences, CSP refusal, nested-frame names/CSS fidelity and parent named-target DOM/blob denial.
- [x] Mount shared UI and tool coordinator in viewer and isolated console; replace single operator callback with subscriptions. Add label-required leave/Escape and independent interrupted-label affordance; never expose transport internals in product UI.
- [x] Run `node --test tests/webcast/sketch-*.test.mjs tests/webcast/viewer.test.mjs tests/webcast/operator*.test.mjs tests/webcast/annotation-placement.test.mjs`; run Go module-assembly tests and browser fixture. Compare render patches with reducer projections at every prefix, run N/2N work counters, commit.

### Task 6 — end-to-end proof, documentation and landing

- [x] Owned headless Chrome tests `TestSketchLiveTwoViewers` and `TestOperatorSketchLiveScenePrivacy`: real Server/recording sees an incomplete pressure stroke before pointer-up; echo once; Undo/Redo/label and historical replay agree. Exercise privileged console scene with the same renderer; adversarial page sees no binding/credentials/private assets and receives no drawing input. Run `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'SketchLive|OperatorSketchLive' -count=1 -v` against a disposable profile, including named-frame attacks as a real negative control.
- [ ] Obtain TL-arranged live console input smoke before landing; send reviewable commands/URLs and exact built revision. Include current-driver and pointer/view behavior; #754 retains wider real iPad/echo acceptance.
- [x] Update README controls/recovery, `atlas/webcast.md`, `atlas/webcast-format.md` and index as needed. Add the TL-required invariant to webcast atlas: the casting page never reaches the recorded scene, bindings or assets. Document ephemeral authority/delivery separately from append-only persisted events. Add lessons only for actual discoveries.
- [x] Audit new production-file dispositions and each changed source-reader input; run `make test-affected-inventory`, artifactpath/repolint checks and all targeted webcast/Node tests.
- [x] Push PR, run recorded `sdlc local-check run --name affected` (full-suite fallback if graph still unavailable), close with evidence and `--no-project` because TL owns ops project. Mandatory fresh review is the close gate; fix all blocking findings and reverify final code.
- [ ] On SHIP promptly `sdlc merge --yes`; retain its runner through admission/integration/CI. Report landed PR and evidence once to hub, then proceed #756→#753→#754 as assigned. Do not claim UI/device coverage that did not run.

## Architectural check

- **ARCH-DRY:** reuse NextSketch, existing driving seat, admission/body bounds, writer acknowledgment/barrier, CatchUp/Hub, anchor traversal and renderer; one shared tool for both surfaces. Server-derived render facts do not create another legal-transition owner.
- **ARCH-PURE:** strict commands/digests, policy, geometry and cursor patches are pure; clocks, actor queues, HTTP/CDP and DOM remain injected integration seams.
- **ARCH-PURPOSE:** privileged console and granted driver are both required; no fallback to remote-only drawing or freezing an arbitrary displayed page.
- **ARCH-MOCK:** stateful Writer/Hub/fake browser/CDP share production interfaces and portable recording storage; owned Chrome conformance plus real console smoke qualify actual seams.
- **ARCH-CONSTRAINTS:** explicit body/history/source/queue/transfer budgets and counted growth tests above; no full-history clone/status scan per chunk.
- **ARCH-SECURE:** current driving seat and isolated operator context establish identity; public epoch/attribution/handles do not. Authenticate before body reads, allow only source-named recorded asset hashes, and verify original owner for late labels. Page DOM/scripts and command payloads are untrusted; original frozen node IDs are evidence only within their source document. Opaque scene origin plus a scoped child CDP world separates page script from source assets and binding authority; actual Chrome qualified the real shared Renderer, nested names/CSS and blob denial. Loader/context retirement fences stale calls.
- **ARCH-ORDER:** existing driving witness lifetime proves exclusivity; writer queue owns ordering and settlement, started effects retain their outcomes, exact origin epochs fence A→B→A, and source capture precedes commit.
- **ARCH-FUNERAL:** source/lease dies on completion/interruption/clear/end; actor timer and Chrome worker join at cast end; transfer handles/blobs die with context/scene; journal has byte/count bounds. Recording events use #751's bounded writer and existing whole-recording lifecycle.
- **ARCH-EVOLVE:** optional stroke-end truncation is additive v1; absent/false preserve old semantics and old readers preserve the unknown flag. New command/capability/delivery values are ephemeral strict service protocols. No changes to existing ctl interpretation or pointer trails.

## Approval and checkpoint

Approved by ops:0 on 2026-10-21 (delivery96710fe7-a135-4b67-81e3-9189c71f5d63), including the opaque scene, authority/receipt/delivery contracts and mandatory named-target leak regression. TL will perform console smoke when staged. Run `sdlc change-code --issue 752 --flow full --worktree=no`, answer plan-quality findings, then derive the estimate from the accepted plan. No implementation or estimate preceded approval.

## Revisions

### 2026-10-21 — hostile-page scene qualification

- Actual disposable Chrome156 proved that a named same-origin iframe inside a closed shadow root is still discoverable with `window.open('', knownName)` and its recorded DOM is readable by the page. `window.frames` hiding and isolated globals alone are insufficient evidence. Ordinary and universal-access top isolated worlds cannot directly read an opaque-origin sandbox frame either. Finalize and qualify the smallest concrete scene isolation mechanism before requesting approval; no implementation began on the disproven assumption.
- Qualified alternatives with actual shared Model/Renderer in Chrome156.0.8078.47: making all contexts unnamed from birth prevents named targeting but changes recorded name/CSS semantics; removing names after loading still leaks. Chosen opaque `data:` shell plus child isolated world renders normal/nested recorded frames, name-based CSS and private blob images unchanged. Parent named targeting receives SecurityError and cannot see binding/renderer globals. This is the bounded revised scene plan; no further incremental privacy mechanism is assumed. Reproduce that evidence as the mandatory live regression rather than depending on scratch artifacts.

### 2026-10-21 — verified implementation checkpoint

Tasks1–5 are implemented and covered by focused Go/Node, stateful writer/controller tests, generated corpus/fuzz checks and counted N/2N guards. The function-strategy table above is the verification map; task case lists are implementation prompts, not additional duplicated test inventories. Actual evidence is in the issue Log. Shared completed-path placement additionally needs an indexed source-document anchor set, including empty/undone strokes; the trusted original-node binding admits only current-document anchors and never identifies an identical replacement by text. The parent console receiver waits for journal synchronization before original-node binding, avoiding exhausted retries during a bounded bootstrap.

The opaque scene required automatic observer exclusion before bootstrap plus actor-acknowledged frame admission before normal recording setup. An exact private-asset URL registry excludes its blobs while retaining unrelated session fonts. These are lifecycle owners for the approved privacy invariant. The original console live fixture initially captured about:blank before its requested URL committed; its readiness now proves the committed fixture loader before creating the scoped scene. Production parent/context retirement remains strict.

TL delivery d053ca5e-cb78-44ee-b89c-d3db95f99841 explicitly holds staging until the coordinated main-build restart. Prepare and check this branch, report readiness, and await TL go before sdlc stage. Required TL console smoke and recorded local check remain open.

### 2026-10-21 — acceptance sequencing

TL delivery b1d858cd-0dfc-445b-a623-c3e78b62a282 directs finishing the recorded check and close review while the operator restart holds staging. Close may record local code acceptance with smoke explicitly pending; the required real console smoke still precedes landing. No stage or merge is authorized until that hold clears.

### 2026-10-21 — recorded verification

The retained full fallback suite passed on10e2d7b59465d1f955becbc6ceb8184e03a6ab3c (11m11s, log SHA256 prefix c157a785c710). PR512 is open. The close gate now owns its mandatory review; that task remains subject to resolving its findings. TL console smoke and landing remain unchecked and held.
