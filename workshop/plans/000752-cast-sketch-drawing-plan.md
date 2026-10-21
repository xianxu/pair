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
| Strict sketch command union, normalized digest and replies | `board/sketch_wire.go` | new |
| Strict Unicode byte validation | `../strictjson/unicode.go` | new |
| Drawing authority and permit identity | `drive/drawing.go` | new |
| Owned receipt/metadata/top queries and cancellation semantics | `recording/sketch_state.go`, `recording/sketch.go`, `recording/sketch_codec.go`, `recording/sketch_snapshot.go` | modified |
| Source-document anchor index | `recording/sketch_anchors.go` | new |
| Bounded source-prefix candidate and state byte accounting | `recording/sketch_source.go` | new |
| Browser interaction/recovery policy | `web/sketch-policy.js` | new |
| Render-only patch projection and incremental painter | `web/sketch-render.js` | new |
| Browser reducer queries and anchor index | `web/sketch-model.js`, `web/model.js` | modified |
| Operator journal/cursor and transfer receiver | `board/sketch_delivery.go`, `web/sketch-delivery.js` | new |

The command union owns ingress shape, not persisted shape: the client supplies node identity and observed box, never trusted author, owner, selector/path/text, source time or digest. Digest the normalized strict command (including operation, sequence and origin epoch); store that digest in the event. #751 independently compares actual final event semantics on reducer retry. A begin retry resolves its original server-assigned sketch/source before any current-source reconstruction.

Drawing authority is a projection of existing driving ownership, not another grant machine. A permit captures one owner/epoch and a controller-owned revocation witness. Metadata label/status credentials are distinct from drawing permission; they cannot mutate geometry. Narrow receipt and stack queries return owned small values in O(1), never clone all points.

Render patches contain already accepted facts: new point chunks and authoritative per-stroke visibility/sealed/truncated flags, sketch phase/label and metadata. The display projector applies those facts by identity; it does not reproduce Undo/Redo legality or maintain another authority FSM. Compare its output with the reducer projection at every test prefix.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Controller witness and capability publication | `drive/controller.go`, `drive/controller_mem.go`, `drive/capability.go` | modified | driving holder, acknowledged ctl/reset, input release and seat lifetime; new witness in `drive/drawing.go` |
| Board sketch authority/service and source lease | `board/sketch.go` | new | annotation proof, authenticated principal, frozen State and writer |
| Board lifecycle, sketch journal and anchor projection hooks | `board/board.go`, `board/operator_view.go` | modified | existing reducer and serialized Board |
| Settled reference writer | `board/writer_sketch.go` (new), `board/writer.go` (modified) | new / modified | existing stateful Writer, sink outcomes and barriers |
| Settled recording writer and ingress queue | `recorder/sketch.go` (new), `recorder/recorder.go` (modified) | new / modified | actor preflight, mirror, sink and barrier |
| Cast source/provider and scoped asset reader | `cast_sketch.go` (new), `cast.go`, `chrome.go` (modified) | new / modified | existing Hub and cast-owned fallback Hub; existing recording asset files |
| Sketch HTTP and capability routes | `server_sketch.go` (new), `server.go`, `server_capabilities.go` (modified) | new / modified | seat authentication, castlock and bounded bodies/assets |
| Shared tool and held browser scene | `web/sketch.js`, `web/sketch-scene.js`, `web/sketch.css` | new | Pointer Events, independent source Model/Renderer and label transitions |
| Viewer/coordinator and recorded geometry adapters | `web/viewer.js`, `web/drive.js`, `web/annotate.js`, `web/render.js`, `web/index.html`, `web/viewer.css` | modified | live connection, mode suspension and existing placement helper |
| Console transport and serialized runtime | `operator-sketch.js`, `chrome_operator_sketch.go`, `chrome_operator_sketch_binding.go`, `chrome_operator_sketch_wiring.go` | new | trusted context, async settled writes, metadata and private scene/journal delivery |
| Existing console assembly and view integration | `operator.js`, `operator_script.go`, `chrome_operator.go` | modified | shared UI modules and existing worker |
| Trusted original-node binding | `chrome_operator_original.go`, `recorder/operator_original.go` | modified | current-document note/sketch eligibility and exact debugger objects |
| Opaque private scene | `chrome_sketch_scene.go`, `sketch_scene_script.go` | new | child isolated world, frozen Renderer, loader/context identity and transfer handles |
| Scene recording exclusion | `chrome_sketch_exclusion.go`, `recorder/exclusion.go` (new), `recorder/drive.go`, `recorder/observer.js`, `recorder/recorder.go`, `chrome.go` (modified) | new / modified | frame admission, private asset URLs and existing recorder/drive lifetime |
| Existing browser/CDP stateful doubles | existing `fake.go`, `fake_drive.go` and package test harness | reused unchanged | existing interfaces; new stateful writer/source behavior is in the rows above |

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

Each task uses the named function-verification strategy above. A checked task records its implementation checkpoint; boundary findings reopen the affected behavior until the review accepts its disposition.

### Task 1 — protocol and reducer queries

- [x] Use the parser/digest and owned-query strategy: establish failing tests, implement strict ingress/replies and constant-size queries, then pass `go test ./cmd/internal/webcast/board ./cmd/internal/webcast/recording -run 'SketchCommand|SketchReceiptQueries' -count=1` and parser fuzz.
- [x] Extend cancellation through the shared Go/JS corpus, codec, semantic retry and snapshot receipt replay. Preserve absent/false identity and terminal reserve; pass focused corpus/policy tests and commit.

### Task 2 — driving witness and authority feed

- [x] Apply the driving-witness held-transition strategy to operator-local and granted-driver origins. Prove no old permit revives after A→B→A and no new permit precedes acknowledged release/control.
- [x] Pass `go test ./cmd/internal/webcast/drive ./cmd/internal/webcast -run 'DrawingPermit|DrawingReset|SketchAuthority' -count=1`, including normal driving regressions, then commit.

### Task 3 — source, acknowledged writers and lease

- [x] Use the source preflight, settled-writer and expiry strategies through both reference Writer and Recorder. Compare persisted replay and exact physical-write counts; refused preparation changes no clock/state/budget and unknown sinks stay unknown.
- [x] Pass `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder -run 'SketchSource|SketchWrite|SketchLease|SketchHandoff' -count=1`, N/2N counters and existing note/control ordering, then commit.

### Task 4 — routes and private delivery

- [x] Use ingress and journal strategies with the real Server/Writer/Hub and trusted-context CDP double. Compare every accepted render prefix to reducer output; ingress refusal precedes body effects and context replacement fences late work.
- [x] Pass `go test ./cmd/internal/webcast/... -run 'SketchRoute|OperatorSketch|SketchDelivery|OperatorScript' -count=1`, `node --test tests/webcast/sketch-delivery.test.mjs` and existing operator/castlock checks, then commit.

### Task 5 — shared tool, held scene and painter

- [x] Use pointer/painter, FIFO-policy and independent-scene strategies; mount the same tool in viewer and console. Required-label transitions and source-document identity are enforced on both adapters. Count new samples/segments, not wall-clock speed.
- [x] Pass `node --test tests/webcast/sketch-*.test.mjs tests/webcast/viewer.test.mjs tests/webcast/operator*.test.mjs tests/webcast/annotation-placement.test.mjs`, Go assembly tests, N/2N guards and owned browser fixtures, then commit.
- [x] Resolve round1 BR-2/BR-3 with red/green regressions for authoritative console lifecycle metadata and queue-full terminal recovery, covering move and pointer-up.

### Task 6 — conformance, review and landing

- [x] Run `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'SketchLive|OperatorSketchLive' -count=1 -v`: compare partial pressure/Undo/Redo/label/replay and assert actual console/private-scene behavior, including named-frame negative controls.
- [ ] Rerun host conformance on the corrected implementation and supply exact evidence for BR-5; retain interrupted-contact and moved-original/replacement regressions.
- [x] Update README, webcast/format atlas and index, including the invariant that the casting page never reaches the recorded scene, bindings or assets. Add only measured lessons.
- [x] Audit production artifact dispositions and source-reader inputs; pass `make test-affected-inventory`, artifactpath/repolint and targeted webcast/Node checks.
- [x] Open PR512 and pass the recorded local-check full fallback on the initial implementation.
- [ ] Push corrected code, rerun the recorded check and `make test-affected-inventory`, then close with evidence and `--no-project` because TL owns ops project. The close gate owns the mandatory fresh review; resolve its findings.
- [ ] After TL staging go, stage/build and send verified revisions for the real console smoke. No landing until smoke succeeds; #754 retains wider iPad/echo acceptance.
- [ ] On SHIP and release of the smoke hold, run `sdlc merge --yes`, retain its runner, report the landed PR to the hub and continue #756→#753→#754.

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

### 2026-10-21 — boundary round1 corrections

BR-4: Cross-checked every Core concepts path/status against the pinned branch patch. Corrected planned nonexistent splits (`board/sketch_authority.go`) and unchanged existing adapters; named actual `recording/sketch_source.go`, `cast_sketch.go`, `board/writer_sketch.go`, runtime, exclusion and anchor-index owners. Existing Hub, browser interfaces, placement helper and browser doubles are reused, not modified.

BR-2/BR-3: Reopened the UI verification class: console journal metadata must drive shared policy through open-contact handoff/pause/expiry/clear/end; bounded point queues must preserve terminal sealing/reconciliation at saturation, including pointer-up synchronous cancellation. Add failing regressions on these paths before fixes and retain exact-operation identity/queue bounds.

BR-5: The disposable review checkout could not start Chrome (GPU/sandbox initialization, CDP EOF); successful host conformance must be rerun on the corrected committed implementation and supplied to the next review. The operator console smoke remains a separate pre-landing hold.

BR-1: Removed repeated case inventories from task prompts; tasks now reference the single function-strategy table and retain runnable commands plus independent invariants. New boundary work is explicitly unchecked until verified.

### 2026-10-21 — lifecycle and terminal recovery implementation

BR-2: Both adapters now supply read-only accepted sketch metadata to the shared tool. Stable stroke identity preserves sealed facts across synthetic views and stale replies; authenticated status supplies current sequence/tops before late labels, including expiry's contextual operation. Geometry retirement and original-owner label-draft retirement have separate lifetimes. Clear/end or accepted labels retire drafts; interruption preserves them. Asynchronous status is fenced by target, text, retirement and observed revision.

BR-3: A full16-chunk queue reserves exactly one bounded truncated terminal behind admitted data, stops capture visibly and enters reconciliation. Pointer-up detaches its captured contact before synchronous effects and releases it in a finalizer. Unknown retries and definite-refusal continuation preserve accepted evidence and emit no duplicate seal.

Residency sweep removes receiver tombstones and prunes per-sketch metadata/drafts with UI slot lifetimes; settled cleared slots must not exhaust the per-cast active bound. Unknown in-flight identities survive retirement until settlement without resurrecting drawing.

Final focused rework checks passed: all targeted webcast Go packages, shared Node/assembly, and audited reader inventory. Settled-retired slot release and all per-slot cache pruning now pass512 clear/restart cycles; unknown cleared begins keep their immutable receipt lookup until definite settlement and cannot emit begin-accepted afterward. Host conformance and recorded suite will now run on the committed fix.
