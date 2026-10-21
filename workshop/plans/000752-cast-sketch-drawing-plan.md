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

Classify each named entity by its actual effects and test dependencies, not its file. PURE here names value types or deterministic transformations of explicitly supplied/owned data, tested without IO doubles. A mixed file can appear in both tables: only the symbols named in the PURE row have that classification; locks, revocation channels, random IDs, callback invocation, canvas/DOM and external execution belong to INTEGRATION.

| Name | Lives in | Status |
| --- | --- | --- |
| Strict command/reply value types, `ParseSketchCommand`, `ParseSketchReply`, `SketchCommandCanonical` and `SketchCommandDigest` | `board/sketch_wire.go` | new |
| `ValidateUnicode` byte validation | `../strictjson/unicode.go` | new |
| `DrawingAuthority` projection value type only | `drive/drawing.go` | new |
| Receipt/metadata/top value types, `NextSketch`, `Reconcile`, `Receipt`, `Metadata`, `StackTops` and deterministic codec/snapshot transforms | `recording/sketch_state.go`, `recording/sketch.go`, `recording/sketch_codec.go`, `recording/sketch_snapshot.go` | modified |
| Source-document anchor index and owned `HasAnchor`/`AnchorNodes` queries | `recording/sketch_anchors.go` | new |
| Owned `SketchSource` value/read helpers and local `sketchSceneSize` byte accounting only | `recording/sketch_source.go` | new |
| Deterministic gesture accumulator, `createSketchGesture` only | `web/sketch-policy.js` | new |
| Accepted display-data model `createSketchDisplay` and `displayFromSketchProjection` only | `web/sketch-render.js` | new |
| Affine point transformation `projectSketchPoint` only | `web/sketch-render.js` | new |
| Deterministic sketch reducer metadata, projection and anchor-index queries | `web/sketch-model.js`, `web/model.js` | modified |
| Cursor/display/batch value types and `sketchDisplayProjection` only | `board/sketch_delivery.go` | new |
| Owned bounded byte-buffer/offset model, `createSketchTransfer` only | `web/sketch-delivery.js` | new |

The command union owns ingress shape, not persisted shape: the client supplies node identity and observed box, never trusted author, owner, selector/path/text, source time or digest. Digest the normalized strict command (including operation, sequence and origin epoch); store that digest in the event. #751 independently compares actual final event semantics on reducer retry. A begin retry resolves its original server-assigned sketch/source before any current-source reconstruction.

Drawing authority is a projection of existing driving ownership, not another grant machine. A permit captures one owner/epoch and a controller-owned revocation witness. Metadata label/status credentials are distinct from drawing permission; they cannot mutate geometry. Narrow receipt and stack queries return owned small values in O(1), never clone all points.

Render patches contain already accepted facts: new point chunks and authoritative per-stroke visibility/sealed/truncated flags, sketch phase/label and metadata. The display projector applies those facts by identity; it does not reproduce Undo/Redo legality or maintain another authority FSM. Compare its output with the reducer projection at every test prefix.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Controller witness and capability publication | `drive/controller.go`, `drive/controller_mem.go`, `drive/capability.go` | modified | driving holder, acknowledged ctl/reset, input release and seat lifetime; new admission/permit methods in `drive/drawing.go` check channel/lock-owned state |
| Board sketch authority/service and source lease | `board/sketch.go` | new | annotation proof, authenticated principal, random sketch/source identities, lease clock and writer callbacks |
| Board lifecycle, sketch journal and anchor projection hooks | `board/board.go`, `board/operator_view.go` | modified | existing reducer, serialized Board and subscriber publication |
| Operator sketch journal lifecycle | `board/sketch_delivery.go` | new | `randomID()` epoch initialization/rotation, Board synchronization, retained entries and subscription channels; not the pure DTO/projection functions above |
| Settled reference writer | `board/writer_sketch.go` (new), `board/writer.go` (modified) | new / modified | stateful Writer, context/lock admission, `SketchWrite.Preflight` authority/source callbacks, sink outcomes and barriers |
| Settled recording writer and ingress queue | `recorder/sketch.go` (new), `recorder/recorder.go` (modified) | new / modified | actor preflight, cancellation, mirror, sink, accepted callbacks and barrier |
| Ordered Hub source candidate provider | `recording/sketch_source.go` | new | `Hub.CandidateSketchSource` locks and observes the existing live Hub before building an owned candidate; accounting/read helpers alone are pure |
| Cast source/provider and scoped asset reader | `cast_sketch.go` (new), `cast.go`, `chrome.go` (modified) | new / modified | existing Hub and cast-owned fallback Hub; existing recording asset file reads |
| Sketch HTTP and capability routes | `server_sketch.go` (new), `server.go`, `server_capabilities.go` (modified) | new / modified | request/context IO, seat authentication, castlock and bounded bodies/assets |
| Browser interaction/recovery policy driver | `web/sketch-policy.js` | new | `createSketchPolicy` invokes an injected operation-ID provider (production uses UUID generation) and emits transport effects; deterministic gesture logic is named separately above |
| Canvas painter and scheduled paint adapter | `web/sketch-render.js` | new | `paintSketchPaths` mutates the supplied canvas context and invokes coordinate callbacks; `createSketchPaintScheduler` invokes schedule/paint callbacks |
| Accepted-record display adapter | `web/sketch-render.js` | new | `createSketchRecordDisplay` observes a supplied live Model through projection/metadata queries; the underlying data-only display transforms are named separately above |
| Browser delivery/ACK callback adapter | `web/sketch-delivery.js` | new | `createSketchDelivery` invokes `installBootstrap`/`applyBatch` and acknowledges only their successful completion; owned buffer/offset logic is named separately above |
| Shared tool and held browser scene | `web/sketch.js`, `web/sketch-scene.js`, `web/sketch.css` | new | Pointer Events, independent source Model/Renderer, timers/animation scheduling, canvas/DOM and label transitions |
| Viewer/coordinator and recorded geometry adapters | `web/viewer.js`, `web/drive.js`, `web/annotate.js`, `web/render.js`, `web/index.html`, `web/viewer.css` | modified | live connection, mode suspension, DOM/canvas layout and existing placement helper |
| Console transport and serialized runtime | `operator-sketch.js`, `chrome_operator_sketch.go`, `chrome_operator_sketch_binding.go`, `chrome_operator_sketch_wiring.go` | new | trusted context, async settled writes, permission/metadata callbacks and private scene/journal delivery |
| Existing console assembly and view integration | `operator.js`, `operator_script.go`, `chrome_operator.go` | modified | embedded shared UI assembly, DOM event subscriptions and existing delivery worker |
| Trusted original-node binding | `chrome_operator_original.go`, `recorder/operator_original.go` | modified | current-document note/sketch eligibility, recorder actor, exact debugger objects and retry lifecycle |
| Opaque private scene | `chrome_sketch_scene.go`, `sketch_scene_script.go` | new | embedded module reads/assembly, child CDP isolated world, frozen DOM Renderer, loader/context identity and transfer handles |
| Scene recording exclusion | `chrome_sketch_exclusion.go`, `recorder/exclusion.go` (new), `recorder/drive.go`, `recorder/observer.js`, `recorder/recorder.go`, `chrome.go` (modified) | new / modified | frame admission barriers, private asset URLs, observer bootstrap and existing recorder/drive lifetime |
| Existing browser/CDP stateful doubles | existing `fake.go`, `fake_drive.go` and package test harness | reused unchanged | stateful test substitutes for actual browser/CDP interfaces; integration verification, not PURE entities |

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

`mountSketch(doc, win, view, options)` is shared by viewer and casting Chrome. `createSketchPolicy` owns `off|preparing|drawing|finishing|label-required|reconciling|paused|interrupted` tagged states. It permits one command in flight, FIFO chunks at 50 ms, at most 64 points/chunk and 16 queued chunks. At capacity stop the gesture visibly, retain accepted evidence and reconcile/seal; never silently discard samples or send them under a new operation identity.

The tool appears for eligible console/current driver. Enter prepares and awaits scene readiness before accepting input. It suppresses normal remote page input while retaining the driving seat. A common tool coordinator intercepts transitions away from sketch, ends any open stroke, requests a nonblank label, and completes the transition only on confirmation. Escape returns to the same sketch. Undo/Redo wait for stroke-end acknowledgment and use only the current unfinished sketch. A handoff releases the drawing mode promptly and preserves a separately reachable old-author label action.

Pointer handling captures only the contact pointer, ignores pen hover, consumes bounded coalesced samples and preserves normalized pressure. Mouse/touch use pressure 1; pointer-up retains the last contact pressure; cancel seals accepted partial work with truncation. The existing stroke-end cannot express this fact: add optional `truncated: true` to `SketchStrokeEnd`, using `omitempty` so absent/false retain their existing wire/digest. First sealing uses the flag; already sealed retries never erase interruption evidence. The sketch remains usable for another stroke, Undo/Redo and label. Update both codecs, semantic identity, core seal and snapshot receipt replay; preserve this new unknown member in a frozen #751 reader. Optimistic samples are keyed by stroke/chunk and replaced by echo, never doubled. Use shared `projectPoint`/`pressureWidth` and round joins/dots; keep fading pointer trails separate.

`createSketchScene` owns an independent Model/Renderer, iframe, recorded asset resolver and readiness/disposal. Initial and reconnect scenes reduce the acknowledged begin prefix. Freeze source scroll, viewport and animations, including nested frames/scrollers; viewport fitting changes only the display transform. Live connection continues applying DOM/control/sketch records, suspending only its page paint while the held scene is active. On exit destroy scene/blobs and request ordinary fresh catch-up. Placement shares the existing confidence/clip traversal and generalizes original-node proof to sketch source document identity; absent/ambiguous anchors keep their historical list item.

The operator reaches the same Board service through the excluded isolated world. Keep sketch request correlation separate from annotation's document-scoped ledger so metadata-only pending labels survive a handoff/navigation while credentials live. Page scripts receive no binding, seat key, private handle or recorded asset bytes. Reuse the existing Chrome delivery worker/timer and extend its module assembler narrowly to support the actual shared Model/Renderer export forms, with execution tests; do not import the entire replay UI.

For the console held page, use an opaque `data:` shell with `sandbox="allow-same-origin"` and no scripts. Chrome's `SceneSession` finds its frame ID through `DOM.describeNode`, uses the owning Page session (attaching an OOPIF only when needed), and creates one scoped child isolated world there. Inject the actual shared Model/Renderer into that world through bounded CDP transfers; do not place history/assets/bindings in the shell URL or page messages. Its nested about:blank frames share its opaque origin, so recorded names and CSS remain intact while the casting page cannot read their DOM/assets even through named targeting.

The parent isolated world retains toolbar, label draft, gesture controller and overlay. The child owns only frozen Model/Renderer plus read-only hit-test/placement methods. The shared scene view accepts async hit-testing; collect the bounded current gesture while its first anchor is acknowledged. Parent↔child calls go through the existing trusted CDP worker, not public `postMessage`. `SceneSession` binds parent context, frame/session, loader, child context and source lease; context/loader retirement invalidates all responses. Never reinstall a trusted binding after page-caused navigation. Exclude scene targets from both recording observation and normal drive input. Preflight shell/module readiness before committing console begin; a CSP block is an explicit source-preparation failure, with no page-policy bypass.

`OperatorView` adds only drawing authority and journal cursor. Board appends accepted render patches/contextual changes to one 256-entry/1 MiB journal, encoding new data once. Chrome drains at most 32 records/64 KiB per step from its ACKed cursor. New context/gap uses a render-only owned bootstrap captured atomically with high-water; it contains paths/flags/labels, not reducer receipts. Its 36 MiB cap follows `admitted history bytes + 128×stroke count + 256×sketch count +1024 <35 MiB` at 16384 strokes/256 sketches, rounded upward. Each header/point/label occurs once, variable IDs are not duplicated outside their headers; test the actual encoding against that inequality. No accepted item may be silently dropped. Receiver ignores exact duplicate batches and rejects a gap before mutation.

Private source/bootstrap/assets use 32 KiB raw chunks (about 44 KiB base64) with exact handle/offset/total, one outstanding transfer per context and at most four contexts. Only hashes named by that source may be read. Process one bounded transfer step per worker turn so permission/reply delivery cannot starve. Context retirement, clear/end, lease expiry and scene exit release handles/blobs; late chunks cannot recreate them.

## Canonical gesture lifetime and event contract

This is the single gesture transition/test table for BOTH viewer and console (BR-9, TL delivery50744457-9fa4-450d-9421-52f62ea94a44). It describes logical ownership, independently of transport receipt state. `idle` means no accumulator token/contact; `drawing` means one token with physical contact, before or after anchor resolution; `anchoring` means physical contact ended but the token still awaits anchor/terminal production; `finishing` means voluntary exit owns settlement/required label and freezes new capture; `retired` means a gesture generation was canceled and all its late callbacks are fenced. Ordinary terminal production transfers work to the existing FIFO and releases the accumulator token. A pending/unknown FIFO receipt remains governed by the existing policy; it does not require an extra gesture owner.

Admission predicate: current policy permits drawing AND no physical contact AND no accumulator token. Check before ID/optimistic/capture/hit-test/timer allocation; validate a complete candidate before publishing contact. A rejected physical contact never becomes admitted by later predecessor completion: require a fresh down. Sketch-layer events are consumed while the tool owns input, including rejected successor tails. No successor buffer is created. The table is the verification requirement, not a claim that every cell has already passed. Cover reachable transitions on both adapters; document unavailable combinations and check no-op boundaries. Column precedence is retired callback generation, then finishing/required-label exit intent, then drawing/anchoring according to physical contact, then idle. These are derived ownership views, not another production FSM.

| Input / completion event | idle | drawing | anchoring | finishing | retired |
| --- | --- | --- | --- | --- | --- |
| Primary contact down, valid button | If policy ready, validate then acquire one token/contact and capture; otherwise consume/refuse | Consume other down, no mutation/allocation | Consume visibly: finishing previous stroke; fresh down required after release | Consume, no new capture | Ignore old lifetime; new lifetime must pass ready/idle admission |
| Hover, nonprimary or invalid-button down | No gesture | No mutation | No mutation | No mutation | No mutation |
| Matching move / coalesced samples | Consume rejected-contact tail; never synthesize down | Append bounded original pressure samples; start bounded batch timer; auto-end at limits | Consume rejected-contact tail; preserve predecessor | No new samples after exit closes contact | Ignore old events |
| Matching pointer-up | Consume rejected/duplicate tail | Final sample then exactly one end; release physical capture; anchored token drains→idle, pending anchor→anchoring | Duplicate/rejected tail no-op; cannot end predecessor twice | Existing matching contact ends once; settlement continues | No-op |
| Matching pointercancel / lost capture | Consume rejected/duplicate tail | End once with truncation; release physical capture; drain→idle or retain→anchoring | No-op to predecessor | Same one-end rule; do not cancel retained exit work | No-op |
| Foreign pointer move/up/cancel/lost capture | Consume/no-op | Consume/no-op; preserve matching owner | Consume/no-op; preserve predecessor | Consume/no-op | No-op |
| Candidate setup / capture failure | Allocate nothing on invalid coordinates; rollback only newly acquired token/capture on setup throw; explicit failure, remain idle | No successor admission | No successor admission | No successor admission | No admission |
| Undo / Redo | Permit only existing policy/receipt/top readiness | Refuse/disable while token held | Refuse/disable while token held even with no contact/FIFO | Refuse while exit settlement pending | No geometry action |
| 50 ms sample flush | No-op | Flush bounded chunk to accumulator/FIFO | Preserve existing bounded buffer pending anchor | Flush only owned gesture during initial end, no new sampling | No-op; timer retired |
| 1024 pending-sample/buffer bound | No-op | Stop physical contact once, truncated end; token retained until anchor/drain | No successor allocation or buffer growth | Preserve reserved terminal/recovery, no new samples | No-op |
| 4096 whole-stroke point bound | No-op | End once; after anchor drains, FIFO keeps old terminal before next begin | Unreachable: pending-anchor1024 bound ends first | No new samples; existing terminal retained | No-op |
| Matching owned anchor resolves | Ignore stale identity/generation | Emit one begin and current buffered chunks, retain token/contact | Emit one begin, exact buffered chunks and one end; transfer to FIFO→idle | Emit only matching owned gesture effects; wait gesture+FIFO settlement before label | Ignore fenced callback |
| Matching owned anchor rejects / five-second anchor deadline | No owner to change | Explicit unaccepted-gesture failure; release capture/token, fence callback→idle if policy permits | Explicit failure, release token/fence callback→idle if policy permits | Explicit accepted-kept failure; retire only unaccepted gesture, keep immutable write receipts | No-op |
| Stale anchor success/rejection/deadline after another token starts | No-op | No-op to current owner | No-op to current owner | No-op to current owner | Ignore fenced callback |
| Five-second finish deadline (anchor may already be resolved) | No exit to change | No ordinary gesture change | No ordinary gesture change | Explicit accepted-kept failure; retire only unaccepted waiting anchor, retain immutable FIFO/unknown receipts for Check confirmation | No-op |
| Done / tool change / Escape outside label | No capture; if preparing defer first exit | Enter finishing before physical end; retain first tool intent | Enter finishing retaining token/anchor | Join same exit, no duplicate end or changed first intent | Existing metadata action only; never revive gesture |
| Save required label | No current gesture label transition | Refuse while drawing/unsaved gesture owns work | Refuse before owned work settles | Permit only after gesture ended and no open/FIFO/inflight/unknown/recovery work | Only separately eligible old metadata action; no geometry |
| Escape from settled label dialog | No-op | No-op | No-op | Clear exit intent and return held scene to idle/drawing admission | No geometry revival |
| Writer ACK / accepted status | Existing FIFO policy only | Drain FIFO; do not release live accumulator prematurely | Do not release pending anchor token | Drain immutable FIFO; label only when all owned work settled | Reconcile retained receipt facts only, no callback/scene revival |
| Unknown reply / Check confirmation | Policy blocks new capture; exact identity retained | Stop physical sampling, retain accepted-prefix terminal/recovery and exact operation | Policy uncertainty blocks successors; no new identity | Retain exit intent and bounded FIFO; exact status/retry, visible failure at finish deadline | Existing eligible receipt reconciliation only |
| Definite writer refusal / Continue recovery | Existing interrupted recovery gate | Retire unaccepted sampling; accepted prefix retained, authenticated seal recovery | Same, fence anchor | Retain exit intent; reconcile/seal accepted prefix then label | No old gesture revival |
| Handoff / navigation / expiry / pause | Existing policy authority update | Retire sampling/token/callback generation; accepted history retained | Retire token and fence late anchor; explicit interrupted outcome | Strong retirement overrides finish; accepted-kept notice and eligible old-label action | Idempotent; resume/new owner uses a fresh generation |
| Clear / end / dispose | Retire timer/source lifetime | Release physical capture, cancel token, fence callbacks/timers | Cancel token and fence callbacks/timers | Cancel token/timers; clear/end retire label eligibility as existing policy requires | Idempotent no-op |

Count repeated rejected downs at N/2N: zero added stroke IDs, captures, hit-tests, timers or samples; predecessor stays within1024 pending/4096 total bounds. Test anchor arrival between rejected down and its up, same/different pointer IDs, setup failure, fresh subsequent contact, and FIFO terminal-before-successor-begin. Run the full existing unknown/finishing/retirement matrix from this table. Measure real local and tunnel Chrome anchor latency and record sample count/p50/p95/max; frequent >150 ms requires TL review of a one-deep successor queue before acceptance.

## State and recovery contract

Actions: Done, Escape, select another tool, continue drawing, Check confirmation/retry exact operation, reconnect, resume cast, enter label, start new sketch, Undo, Redo, inspect recording.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Begin source is being admitted/built | Preparing held scene, input disabled | Retry exact operation | No accepted work |
| Command started, reply lost or sink uncertain | Confirmation pending, gesture stopped | Retry exact operation | Unsent local samples if tab closes; accepted history retained |
| Writer barrier proves refusal / queue at bound | Interrupted gesture with accepted prefix visible | Continue recovery to reconcile and seal accepted prefix | Rejected/unsent samples only |
| Voluntary exit with gesture/anchor/FIFO still pending | Saving…; new capture and label disabled | Await settlement within five seconds | None under normal settlement |
| Finishing deadline expired | Couldn’t save; accepted strokes kept | Check confirmation for retained immutable operation; acknowledge explicit unanchorable gesture failure | Only explicitly failed unaccepted gesture; accepted history retained |
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
| `createSketchPolicy`, FIFO sender | New identity after unknown / terminal input and exit races | Controlled user/network/ctl sequences on both adapters; delayed anchor and held/unknown reply across each exit control; deadline/recovery/retirement negative controls, one terminal and immutable retry identity |
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
- [x] Requalify logical gesture ownership and pass `node --test tests/webcast/sketch-*.test.mjs tests/webcast/viewer.test.mjs tests/webcast/operator*.test.mjs tests/webcast/annotation-placement.test.mjs`, Go assembly tests, N/2N guards and owned browser fixtures, then commit.
- [x] Resolve round1 BR-2/BR-3 with red/green regressions for authoritative console lifecycle metadata and queue-full terminal recovery, covering move and pointer-up.

### Task 6 — conformance, review and landing

- [x] Run `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'SketchLive|OperatorSketchLive' -count=1 -v`: compare partial pressure/Undo/Redo/label/replay and assert actual console/private-scene behavior, including named-frame negative controls.
- [x] Rerun host conformance on the corrected implementation and supply exact evidence for BR-5; retain interrupted-contact and moved-original/replacement regressions.
- [x] Update README, webcast/format atlas and index, including the invariant that the casting page never reaches the recorded scene, bindings or assets. Add only measured lessons.
- [x] Audit production artifact dispositions and source-reader inputs; pass `make test-affected-inventory`, artifactpath/repolint and targeted webcast/Node checks.
- [x] Open PR512 and pass the recorded local-check full fallback on the initial implementation.
- [x] Push corrected code, pass `make test-affected-inventory`, and finalize local close with evidence and `--no-project` because TL owns ops project. Resolve the binary-owned review findings; FIX-THEN-SHIP correction is reconciled.
- [x] Retain the recorded final-code check through the queue: replacement attempt735922b215e89a192d144d44af5f4a35 on61e43e35 passed the full-suite fallback in10m2s; sdlc-local/suite success confirmed. See Revisions for interrupted-run recovery.
- [ ] After TL staging go, stage/build and send verified revisions for the real console smoke. No landing until smoke succeeds; #754 retains wider iPad/echo acceptance.
- [ ] On SHIP and release of the smoke hold, run `sdlc merge --yes`, retain its runner, report the landed PR to the hub and continue #756→#753→#754.

## Architectural check

- **ARCH-DRY:** reuse NextSketch, existing driving seat, admission/body bounds, writer acknowledgment/barrier, CatchUp/Hub, anchor traversal and renderer; one shared tool for both surfaces. Server-derived render facts do not create another legal-transition owner.
- **ARCH-PURE:** classification follows effects and test dependencies, not file placement. Only the named value types, strict parsing/digests, deterministic reducer/display/gesture/buffer transforms, affine geometry and local byte accounting are PURE. Controller locks/revocation checks, Hub observation, journal random initialization/publication, policy ID-provider invocation, delivery/Model callbacks, paint scheduling/canvas writes, clocks, actor queues and HTTP/CDP/DOM are INTEGRATION, including when colocated with pure helpers. Unit data/corpus checks cover the former; stateful boundary doubles and owned live conformance cover the latter.
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

Host conformance on8be34c6fab5b1cc1c2431a7cadb1da915b2e793d executed all4 required live tests successfully (package9.541s); BR-5 receives this measured host evidence through its formal dispute. Stage/operator smoke and landing remain held.

Corrected-head suite encountered only the unchanged fixture FD flake, now pair#767 with same-head targeted passes. The retained recorded rerun is queued; mandatory close review proceeds independently with this limitation explicit. All automated acceptance is not complete until the recorded rerun is settled. Stage and landing remain held.


### 2026-10-21 03:55 PDT — BR-7 complete Core concepts effect/test classification sweep

Rule applied to every existing row: an entity is PURE only when its named symbols transform supplied/owned data deterministically and need no IO double; a file containing such a helper is not itself classified PURE. Callback, identity-generator, lock/channel, live-owner, canvas/DOM or external-execution dependencies make the wrapper INTEGRATION. This revises classification and verification ownership only; code, approved behavior and the previously verified file/status map are unchanged.

Audited all ten prior PURE rows. Kept strict commands/digests/replies, Unicode validation, receipt/reducer/codec/snapshot queries, anchor indexing and browser reducer queries as explicit deterministic symbols. Narrowed drawing authority to its value type (permit checks/admission remain with controller integration); source to owned reads/local accounting (Hub candidate observation moved to integration); policy to the deterministic gesture accumulator (the operation-ID callback driver moved to integration); rendering to display transforms/affine math (canvas painter, scheduler and live Model adapter moved to integration); and delivery to cursor/display DTOs, projection and byte-buffer offset logic (random journal ownership and install/apply callbacks moved to integration).

Audited all fifteen prior INTEGRATION rows and retained their verified new/modified/reused statuses: controller, Board service/hooks, reference/recording writers, cast Hub/assets, HTTP routes, shared tool/scene, viewer/geometry adapters, console runtime/assembly, debugger original binding, opaque scene, recording exclusion and existing doubles each cross the stated state/IO/user-input boundary. The new explicit wrapper rows separate effects already present in those mixed files; they introduce no new implementation paths.

Verification seams match the classification: command/Unicode/reducer/anchor/source-accounting/gesture/display/geometry/transfer checks compare owned bytes or values without IO mocks; policy-driver checks inject deterministic ID providers; journal/source-provider checks use the real in-memory Board/Hub and exercise random epoch/publication/ownership; delivery checks inject an actual display or stateful callback/refusal target; canvas/scheduler checks use the existing canvas/scheduler doubles; actor/controller/HTTP/CDP/DOM wrappers use the existing stateful Writer/browser/CDP harness and owned Chrome conformance. The function verification strategy table remains the detailed test map. ARCH-PURE no longer claims the canvas painter, random journal, Hub observation, live Model/ACK callbacks or production ID-provider driver are pure.

### 2026-10-21 — approved width restored

BR-6 restores the approved3 CSS px base on both producer and optimistic painter, rather than choosing the previous client4 value as the consistency oracle. Service regression fails at4 and passes at3, including persisted pressure0/.5/1 widths .75/1.875/3 and doubled viewer scale. The independent canvas test holds anchor preparation and verifies pressure0/.2/.8/1 using3*(.25+.75p), plus mouse/touch3. Actual producer doubles use3; arbitrary historical width4 fixtures remain supported. Final targeted Go/Node/assembly and reader inventory pass before commit.

All4 host Chrome tests executed successfully on bccbfa399b9a829f347ee21d22a1c45323fc5312 (package6.547s), including the approved-width producer/optimistic paths. The corrected-head recorded check is pending; the next local review may finish independently, with no staging or landing before that check and the TL-held smoke are settled.

### 2026-10-21 — BR-8 voluntary-exit re-plan checkpoint

The third boundary review reproduced loss of the pending gesture on all six Done/Escape/tool-change × open/released-contact combinations when anchor lookup resolves after leaving. ARCH-ORDER/ARCH-PURPOSE require a single exit owner: stop new capture, preserve the existing gesture and its terminal, settle its ordered work, then permit required-label completion. Implementation is paused for a concrete finishing-state design and TL approval; the audit includes pending anchor, queued and unknown work on both adapters, repeated exits and authoritative retirement. No new code has been written for this revision.

### 2026-10-21 — BR-8 concrete finishing design (awaiting TL approval)

ARCH-ORDER/ARCH-PURPOSE: one policy-owned finishing lifetime owns the voluntary exit. Done, Escape and tool change enter `finishing` before flushing contact. Retain the first requested destination, current stroke identity and a gesture-pending latch; repeated exits join this owner. New capture, Undo/Redo, entering another sketch and label submission are blocked. Only the already-owned stroke's late anchor/buffer/terminal can enter the existing bounded FIFO. The shared mount reports gesture completion after its effects are enqueued, or reports explicit unusable-anchor failure. Do not change generation or destroy the held scene for voluntary exit.

The policy permits required label exactly once after gesture completion AND no open stroke, queued command, inflight identity, uncertainty or reserved recovery terminal. Unknown results retain the exact operation and exit intent through Check confirmation. Definitive geometry refusal uses the existing interrupted/Continue recovery with the accepted prefix; the retained exit resumes finishing and seals that prefix before labeling. Authority loss, pause, navigation, expiry, clear/end and disposal retain their existing stronger retirement rules and invalidate late anchor callbacks; these cannot restore voluntary drawing. Escape from a settled required-label dialog still returns to the same scene. No second legal recording FSM or unbounded work is introduced.

The simpler alternative, await idle before dispatching leave, fails: the pending Promise set includes unrelated paint work and excludes a timed-out unknown command; it also leaves new capture admitted while waiting. A single policy latch plus stroke identity makes both capture exclusion and label readiness explicit.

Actions: Done, Escape, select another tool, Check confirmation, Continue recovery, save label.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Finishing with unresolved anchor or queued writes | Finishing stroke; capture and label disabled, scene retained | Wait for owned anchor/write settlement | None under normal settlement |
| Finishing with unknown write | Confirmation pending, scene and exit intent retained | Check confirmation using the same operation | No accepted history lost; unconfirmed outcome stays explicit |
| Definitive geometry refusal during finishing | Interrupted with accepted prefix retained | Continue recovery to reconcile/seal then label | Refused/unaccepted samples only, visibly reported |
| Unusable anchor | Explicit no-usable-anchor notice, then required label for accepted history | Save label or Escape back to sketch | Unanchorable buffered gesture, explicitly reported |
| All owned gesture/write work settled | Required label | Save label, or Escape to same sketch | None |
| Authoritative interruption or retirement during finishing | Existing interrupted/paused/ended UI | Existing late-label/recovery action as eligible | Unsent samples follow authoritative interruption contract |

Files: modify `cmd/internal/webcast/web/sketch-policy.js`, `web/sketch.js`, colocated Node harness/tests under `tests/webcast/`; update shared controls documentation and lessons. Both entities remain INTEGRATION as classified above. Reuse the stateful transport/model and console journal harnesses; no new production resource family.

Verification sequence: write controlled failing tests first, then implement the one finishing owner. Sweep Done/Escape/tool-change × open/released contact × delayed-anchor/queued/unknown work through viewer Model and console synthetic/journal adapters. Hold anchor and write replies independently; assert no premature label, exactly one terminal, preserved points, immutable retry identity, blocked new capture, first exit intent and no late resurrection on authority/clear/dispose. Include unusable anchor and definitive refusal followed by recovery. Run focused Node/Go, all browser tests, existing live Chrome privacy/drawing/handoff/two-viewer conformance, inventory preflight and a fresh recorded final-head check before landing. Staging remains held.

### 2026-10-21 — TL approval and bounded visible finishing

TL approved the exit re-plan (delivery ec1b8b37-c337-452a-a91e-45f37ba01996). Both surfaces must show visible “saving…” during finishing. Bound finishing by the existing operation deadline; on expiry show explicit “couldn't save; accepted strokes kept”, preserving exact unknown receipt identities for reconciliation and preventing late callbacks from presenting false success. Tests cover each Done, Escape, tool change, and navigation or handoff with pending anchor on both surfaces. The adapter audit also found preparation/reconciling gate bypasses: retain the first deferred exit through preparation and route all tool controls through the same exit owner. These refine preservation of existing work, without widening product scope.

### 2026-10-21 — finishing deadline ownership detail

Use the existing five-second console operation/CDP deadline as the shared finishing visibility bound. On expiry, report “Couldn't save; accepted strokes kept” and expose explicit recovery. Retain immutable unknown transport operations, accepted history and their admitted FIFO for confirmation; timeout is never a definitive refusal. An unresolved anchor with no durable stroke can be retired as unaccepted, with its callback generation fenced and the loss explicitly reported. Do not let a late anchor silently erase failure. Retained write acknowledgments may reconcile accepted facts, with the failure notice retained until user recovery. Bound leave's returned wait as well as its visible state; do not await the global mixed pending-Promise set indefinitely. Clear the finish timer at settlement/retirement/disposal.

The BR-8 revision also updates the current state enumeration, recovery rows and function-strategy row above so the live contract includes finishing and its bounded failure, rather than leaving the new rule only in historical revisions.

BR-8 implementation uses one shared `beforeAnnotationMode` gate on the tool, delegated to by both adapters, including preflight/deferred exit. Five-second `observeRequest` observers detach UI waiting and deduplication from a hung transport without canceling or reidentifying the writer's operation. Unknown write failure exposes Check confirmation; anchor-only failure instead retains label/escape acknowledgment, without advertising a nonexistent receipt. The label-dialog Escape clears the finished exit intent before ordinary drawing resumes.

### 2026-10-21 — BR-9 logical gesture ownership re-plan checkpoint

The review verified BR-8 and reproduced a successor-contact crash on both surfaces. Physical pointer release currently clears `contact` before the gesture has resolved its anchor and drained its terminal; a new contact mutates capture state before accumulator admission throws. ARCH-ORDER/ARCH-PURPOSE require one logical gesture owner held through anchor/buffer/terminal production, independent of physical contact. Stop implementation and enumerate pointer-up, pointercancel, lost capture, buffer/point-limit endings, explicit exit and every successor-input transition before choosing the admission rule. Reopened Task 5 pointer verification. The existing recorded runner remains retained pending a concrete reviewed correction.

### 2026-10-21 — BR-9 concrete logical-owner admission rule (awaiting TL approval)

Use the existing accumulator's `snapshot().active` token as the single resource for logical gesture ownership. Physical pointer-up, cancel, lost capture and automatic buffer/point-limit endings close contact but do not retire that token until anchor plus terminal production drains into the policy FIFO, or explicit failed-anchor/deadline/authority/disposal cancellation fences it. Once the token drains, later strokes may queue behind the prior terminal without waiting for its receipt; FIFO and uncertainty policy retain their existing authority.

Before allocating a successor stroke ID, contact, pointer capture, optimistic entry, hit-test or timer, check this token. While held, consume successor down/move/up on the sketch layer without recording or leaking browser/page input, and show “Finishing the previous stroke. Wait before drawing again.” Require a fresh down after ownership ends. Construct and validate the first sample/complete contact before publishing capture state, so failed admission never leaves partially initialized contact. Do not add a second ownership boolean or successor sample queue. A queue would require another bounded anchor/buffer/cancellation lifecycle merely to admit input before the owned predecessor is ready.

Use the same five-second UI observer around anchor lookup, including ordinary stroke completion, to bound this visible wait. Failure explicitly reports the unaccepted gesture, frees its owner and fences late results; already accepted history remains. Both adapters reuse the same controller and existing failure/retirement path. Adds no durable resource or higher memory bound.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Contact released but logical token awaits anchor | Finishing previous stroke; successor input consumed | Wait for anchor within existing five-second bound, then draw with a fresh down | No predecessor loss; attempted successor explicitly unavailable |
| Anchor failed/deadline | Explicit unsaved gesture, accepted history kept | Fresh down after failure or existing label/exit action | Unaccepted unanchorable gesture only |
| Logical terminal produced, predecessor receipt pending | Existing drawing/FIFO or confirmation state | Fresh down when policy permits; exact retry for uncertainty | None in accepted history |

Verification: controlled RED/GREEN on viewer and actual console for pointer-up/cancel/lost capture/1024 buffered-point automatic end with delayed anchor, rejected successor tails, anchor resolving between rejected down/up, anchor rejection/deadline/retirement, and fresh subsequent contact. The 4096 total-point ending is reachable only after anchor establishment because the pending-anchor buffer caps at1024; test its synchronous drain and FIFO terminal-before-successor-begin order separately. Assert no throws, no new allocation/capture/hit-test for rejection, exact original pressure points and one terminal. Requalify existing finishing, uncertain-write and retirement regressions, host conformance, inventory, recorded final-head suite and close.

### 2026-10-21 — TL-approved complete gesture table and latency qualification

TL approved BR-9 (delivery50744457-9fa4-450d-9421-52f62ea94a44), requiring the canonical five-state/every-event table above as the test source on both surfaces. Successor contacts during pending anchors are visibly dropped; measure actual local and tunnel Chrome anchor latency and bring frequent waits above roughly150 ms back to the TL for a possible one-deep queue. Browser owner implements/tests the shared admission resource; wire owner qualifies actual latency independently. Existing recorded runner remains retained until executable correction is committed.

### 2026-10-21 — complete-table independent audit

The read-only table audit added foreign-pointer no-ops, transactional setup rollback, Undo/Redo token admission and a separate finishing deadline event. It clarified precedence and changed premature every-cell coverage wording into an explicit verification requirement. Existing tests map normal sampling, voluntary exit, unknown/refusal, finish deadline and retirement families; BR-9 adds same/different rejected pointers, N/2N zero allocations, separate pending-sample/accumulator bounds, setup/capture failure and4096-point FIFO ordering. Root also identified stale anchor rejection after prior-terminal uncertainty: both success and failure callbacks must prove the original token identity, so a retired predecessor cannot cancel a new gesture. These enforce the approved single ownership rule.

### 2026-10-21 — BR-9 implementation and first latency qualification

The canonical table now drives both-adapter regressions for physical endings, successor tails/admission failure, bounded wait, controls, retirement and stale success/failure. Source uses the existing accumulator token, not a successor queue or duplicate ownership state. The released-contact/unknown-prior-terminal combination is covered explicitly; only unanchored unaccepted work is canceled and no nonexistent end command is manufactured. Root Node/operator, host live4 and inventory checks pass (issue Log contains exact evidence). The first actual local/tunnel anchor series has zero >150ms among1600 samples; console p95 is0.5ms both idle and with journal traffic. This supports retaining the approved simple admission rule for the measured desktop fixtures, with device/echo acceptance left to #754.

### 2026-10-21 — direct accumulator latency result

The final real Chrome fixture measures actual createSketchGesture end→anchor→terminal drain, separately from ACKs:1600 samples, zero >150ms. Local/tunnel viewer p95 both0.1ms; console idle/busy p95 0.5/0.6ms (max2.1/4.5ms). Tunnel ACK p95 34.1/31.7ms does not hold the token after terminal production. Run passed108.30s; issue Log pins raw samples and fixture command. No one-deep queue is justified by the measured small desktop source; #754 retains device/echo qualification.

### 2026-10-21 — BR-10 approved Enter-to-save parity

The next close verified BR-9 and disposed all prior criticals, returning FIX-THEN-SHIP for the missing approved keyboard action. #750 explicitly requires Enter to save a nonblank label and return to the requested tool. Delegate label-input Enter to the existing guarded saveLabel path, preserve IME composition, and test blank/repeated Enter and held acknowledgment on both adapters. Tool selection/source release still waits for accepted label persistence. This restores the approved control without introducing a state or mechanism. Follow the binary's post-verdict path: one bundled fix/review-artifact commit, then issue recovery reconcile, without a duplicate close review.

### 2026-10-21 — local acceptance finalized, final record retained

BR-10 fix and evidence were bundled once, then issue recovery reconcile finalized the FIX-THEN-SHIP close. Cardcodecomplete and completion binding are confirmed. Split Task6 local acceptance from the still-pending final-head recorded pass so the checklist reflects the actual remaining gate; no repeated close review is needed for that record. TL staging/real console smoke still precedes landing.


### 2026-10-21 — continuation recovery and final recorded pass

The continuation restart ended the old queued runner without a saved result. Its session, lease, checkout and attempt were absent on recovery. Replacement lease-3343331055 retained FIFO admission and passed the full suite on prose-only head61e43e35e7794260aa860c3b3ff269e687351e14: exit0 in10m2s after1372s queued, saved attempt735922b215e89a192d144d44af5f4a35, log hash prefixc0f339cf4f5a. GitHub success is confirmed. This closes the final-code recorded-check task; TL restart/staging release, real console smoke and landing remain outstanding. No design or source change.

### 2026-10-21 — operator-smoke availability guidance

TL delivery d446a25b-56c0-471e-b371-92b2ee676110 makes discoverable disabled controls part of acceptance: annotation-off console Chrome showed disabled Sketch/Undo/Redo and an enabled Done with no explanation. ARCH-DRY/STATE: derive presentation from the existing authority and active sketch, without adding admission state. Keep console availability guidance visible; name Chrome's 👆 location and distinguish another driving holder, pause and end. Bind Done disabled to the absence of an active sketch or pending preparation. Test initial denial, authority recovery, pause/end and handoff in the shared tool and console adapter, preserving existing finishing/label recovery. Run targeted Node and live Chrome checks, inventory, fresh close, final recorded verification and restage before accepted smoke.

### 2026-10-21 — active toolbar stacking from operator smoke

TL delivery185e0754-d052-4f48-8e87-485c63a8537b reports active controls disappearing over the held scene. Real Chrome RED proves all four button centers hit sketch-layer instead: the static toolbar sits below the positioned drawing canvas and private scene. ARCH-ORDER: explicitly position the shared toolbar above the canvas/scene while retaining the label layer's priority. Prove visible in-viewport hit targets for Sketch/Undo/Redo/Done during drawing, then use actual CDP pointer clicks for Undo/Redo/Done rather than direct controller calls. Verify no unintended extra ink, durable undo/redo and labelled exit. This corrects presentation/input routing without a new drawing state.

### 2026-10-21 — approved bounded console trust correction

TL22603f68-f45b-4ba9-99a6-428ed9bff6e1 defers concurrent console/viewer lanes and selective clearing. TLcbf33121-ef13-4ead-b9b2-911c2bd48636 approves only console sketching independent of annotation On/Off, while retaining the existing single-holder driver epoch/witness, generation fence, closed-board lifetime and global annotation-clear behavior. Earlier annotation-toggle guidance now applies to viewers; console unavailability explains an active viewer driver or readiness/pause/end.

| Concept | Owner | Revision |
|---|---|---|
| Operator-sketch admission | board/annotation.go | Opaque issued variant of existing AnnotationAdmission; expected generation and board lifetime still checked, capability phase ignored only by sketch Check. Ordinary annotation mutation methods retain ordinary permit checks. |
| Console availability | chrome_operator_sketch_wiring.go, operator-sketch.js | Remove annotation-On requirement; retain current holder, ready target, driver witness and CDP uncertainty fence. |
| Console ingress | cast.go | Capture the operator-sketch variant instead of ordinary viewer admission; final service and scene-open Check reuse the same opaque admission. |

Verification strategy: board unit tests issue operator admission with annotations off, reject stale generation and closed board, and prove it cannot authorize ordinary viewer/pointer mutations. A real cast/Chrome fixture draws, uses native Undo/Redo/Done and saves with annotation never enabled; existing viewer HTTP/drive checks must still refuse annotation-off. Preserve handoff/paused/ended and private-source tests. No new persistent state, recording grammar, source resource, scheduler or concurrency lane. Run targeted package/Node/live checks, audited inventory, fresh full close review and recorded suite on pushed final code; then restage and require real smoke before landing. The versioned two-lane contract is a separate unclaimed dependent issue, not part of this correction.
