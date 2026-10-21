---
collaboration-mode: discussion
---

# Cast sketch implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use @superpowers-subagent-driven-development for bounded independent tasks or @superpowers-executing-plans for work that depends on session context. Steps use checkboxes.

**Goal:** Let a viewer sketch a proposed page design while everyone watches the strokes appear, undo and redo their strokes, and give a coding agent the labelled result, DOM anchors, paths and a historical image.

**Architecture:** Add persistent sketch events to the existing version-1 DOM recording. The recorder remains the single ordered writer; Board owns admission and author authority; pure Go and JavaScript reducers own sketch state. Reuse the viewer renderer and anchor placement for live/replay display and the existing headless replay infrastructure for agent images.

**Tech stack:** Go, embedded JavaScript modules, Pointer Events, canvas overlays, JSONL, existing CDP and sandboxed DOM replay. No new library or service.

**Status:** Proposed design for operator review. No implementation authorized. The operator's latest direction supersedes the earlier local-draft proposal: sketch strokes stream from the start; Undo and Redo are the only drawing commands; colour follows the viewer's pointer. Questions are collected below rather than interrupting review. Implementation issues are proposed here and will be filed after approval.

## Current system and constraints

- `atlas/broadcast.md` maps terminal broadcasting; this feature belongs to DOM casting, mapped by `atlas/webcast.md`, `atlas/webcast-format.md` and `atlas/webcast-submission.md`.
- `recording.Stroke` and `web/annotate.js` implement pointer trails: chunks every 50 ms, at most 64 points per chunk, fading after 1500 ms. They are not persistent drawings. Preserve that meaning.
- `recording.NoteAt`, `notes.Structural`, `web/annotation-anchor.js` and `web/annotation-placement.js` already implement recorded identity, confidence matching, relative box projection and clipping. Reuse them.
- `board.AnnotationAdmission` and recorder-side final checks fence capability generations, pause and navigation. Plain `Emitter.Emit` acknowledges enqueueing, not persistence; sketch commands require the acknowledged writer seam used by prepared notes.
- `recording.Hub`, Go `CatchUp` and JS `catchUp` serve reconnects and seeks. They must retain sketch state explicitly; merely adding a new record type would lose it at segment compaction.
- `query.Surface` currently exposes DOM/events/notes, not images. `fidelity_check.go` and `web/replayer.js` already render historical DOM with recorded assets and frozen animations. Existing note capture PNGs are real-tab evidence and cannot stand in for a historical sketch image.
- Actual compatibility is narrower than the format prose suggests: unknown kinds survive verbatim in `events.jsonl`; unknown fields on known typed records are currently discarded on Encode. Unknown kinds in notes/pointer files are rejected. All new sketch kinds therefore belong in `events.jsonl`.

## Operator experience and choices

The viewer's existing pointer and note tools gain **✎ Sketch**. The drawing UI contains the current pointer swatch, Undo and Redo. Mouse, touch and stylus all produce a fixed-width polyline; pressure, shape recognition, text drawing, eraser, object selection and brush controls are out of scope. Undoing every stroke is how an author erases their sketch. There is no separate erase command.

Recommended interaction:

1. Choose ✎. Prepare a sketch at an ordered recording moment, then show a local held view of that page with a small “Sketching this moment” indicator. The cast and other viewers continue live. Drawing starts after preparation succeeds; no initial gesture is silently lost.
2. Draw. Each stroke appears locally immediately and streams in bounded chunks to everyone. It stays until undone or cleared; it never enters the pointer trail's fade store.
3. Undo hides the author's last completed visible stroke. Redo restores the last undone stroke. Drawing a new stroke after Undo clears the redo branch. Another viewer's activity never changes these stacks.
4. Selecting another existing tool ends this sketch and opens the required label field. Enter saves a nonblank label and returns to that tool; Escape returns to sketching. The label describes intent, for example “Move the price beside the plan name.” No recognition is attempted.
5. A saved sketch appears beside notes, showing label, author, colour and page/time. Before labelling, it appears as “Ana's sketch — label pending.” Selecting an old item opens its historical replay, rather than guessing that today's page is the original page.

The label step is metadata required by the project, not an additional drawing command. Until it succeeds, the recording preserves an explicitly unfinished sketch. Agent queries include unfinished sketches with `label_pending: true`; they never manufacture an intent label. A label does not retroactively change what replay knew before the label event.

### Decision needed: when to collect the required label

**Recommend:** on leaving ✎, as above. This permits immediate live sketching without a form first. Alternative: require the label on entering ✎, so every durable sketch is labelled from its first stroke. Missing labels after a crash are the tradeoff of the recommended choice. The agent must ask for intent before building from an unfinished sketch.

### Decision needed: hold the drawer's page while sketching

**Recommend:** hold only the drawer's rendered page at sketch start. This gives all paths one coherent DOM moment, even if the operator scrolls or resizes during a long sketch. Other viewers project paths onto their current page using the recorded anchors. The drawer still receives sketches, capability changes and pause/end notices. Alternatives: let the drawer's page continue moving and store a separate source moment for every stroke, producing multi-moment image evidence; or pause the whole cast, which interrupts everyone. A held drawer cannot also remote-drive; switching to driving first leaves sketch mode and requests the label.

### Decision needed: where drawing is available

**Recommend:** pointer-link and drive-link web viewers first, including the operator opening their own pointer link. View-only links see sketches but cannot draw/undo/redo. Show saved and in-progress sketches in the operator Chrome's existing annotation projection too. Adding drawing controls inside the real cast Chrome would require a held replay surface there as well; defer those controls unless the operator requires them now. Existing Chrome Pen/Note behavior remains owned by its current shared mount.

### Decision needed: undo scope after leaving a sketch

**Recommend:** Undo/Redo operate on the current unfinished sketch only; selecting ✎ after labelling creates a new sketch. Finished sketches remain historical design evidence. Alternative: reopen an authored saved sketch for Undo/Redo, which adds selection and label-revision semantics. Undo-all satisfies author erasure during sketching; if erasure after finishing is required, include reopening in the approved scope rather than adding an independent eraser.

## Core concepts

### Pure entities

Paths below are relative to `cmd/internal/webcast/` unless stated otherwise. “new” means proposed implementation, not something created by this design issue.

| Name | Lives in | Status |
|---|---|---|
| Sketch event union and limits | `recording/sketch.go` | new |
| SketchState / NextSketch | `recording/sketch_state.go` | new |
| Recording State sketch projection | `recording/reduce.go` | modified |
| Sketch geometry and bounds | `sketch/geometry.go` | new |
| Sketch request parsers | `board/sketch_wire.go` | new |
| Browser sketch reducer and geometry | `web/sketch-model.js` | new |
| Browser sketch interaction policy | `web/sketch-policy.js` | new |
| Sketch query values and text | `query/sketch.go`, `query/render_sketch.go` | new |

One recording has many sketches; one sketch has one author authority, one source moment, many immutable stroke identities, and one Undo/Redo history. A stroke has one `NoteAt` anchor and one or more ordered point chunks. SketchState owns references to strokes; no separate mutable canvas is authoritative. The first image/paths consumer is this feature; later in-progress collaboration can use the same events without redefining existing pointer strokes. `NextSketch` is the sole owner of legal sketch transitions (ARCH-PURE, ARCH-ORDER).

Geometry maps recorded page points through a recorded anchor box to a current anchor box. The Go implementation and JS implementation use one shared fixture corpus; both reuse the existing anchor confidence rules (ARCH-DRY). No new selector language or fuzzy matching system.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Board sketch admission / preparation | `board/sketch.go` | new | capability, author session, ordered recording |
| Acknowledged sketch writer | `recorder/sketch.go`, `board/writer.go` | new / modified | recorder queue and sink outcome |
| HTTP sketch routes | `server_sketch.go`, `server.go` | new / modified | locked pointer/drive links |
| Shared sketch mount | `web/sketch.js`, `web/sketch.css` | new | pointer events, canvas, transport |
| Held viewer scene | `web/sketch-scene.js`, `web/viewer.js` | new / modified | current Renderer and live Model |
| Operator display projection | `board/operator_view.go`, `operator.js`, `operator_script.go` | modified | excluded isolated-world UI |
| Historical image renderer | `sketch_image.go`, `fidelity_check.go`, `web/sketch-image.js` | new / modified | loopback replay and headless Chrome |
| ImageRenderer / FakeImageRenderer | `sketchimage/contract.go`, `sketchimage/fake.go` | new | replay image job and output lifecycle |
| Agent CLI | `query/cli.go`, `query/surface.go`, `query/fake.go`, `couchcmd` query wiring | modified | local recording reads and optional image output |

The writer tests use the existing stateful `board.Writer` and `FakeBrowser`/`FakeDriven`; add held-write and failed-sink controls through the same acknowledged interface. FakeImageRenderer models prepared/running/completed/failed/cancelled jobs, image dimensions, missing assets and output failures; it must exercise the same job API as real Chrome. Owned storage stays a portable recording directory plus an explicitly supplied output directory. The browser test fixture is an owned page, never the operator's profile (ARCH-MOCK).

## Recording protocol and ordering

Keep format `couch-webcast`, `v:1`. Add these event kinds, all in `events.jsonl`:

| Kind | Required payload beyond `t,k` | Meaning |
|---|---|---|
| `sketch-begin` | `id,author,owner,doc,tab,vp` | Begin an unfinished sketch; this record is its exact source moment. |
| `sketch-stroke-begin` | `sketch,id,at,width` | Pin a stroke to an element in that sketch's source DOM. |
| `sketch-points` | `sketch,id,chunk,pts` | Append page-coordinate `[x,y]` points; chunk numbers start at 0. |
| `sketch-stroke-end` | `sketch,id` | Seal the accepted points of the current stroke. |
| `sketch-undo` | `sketch,id` | Hide the last completed visible stroke named by the reducer. |
| `sketch-redo` | `sketch,id` | Restore the top redo stroke named by the reducer. |
| `sketch-label` | `sketch,label` | Finish the sketch with bounded, nonblank plain text. |

Example of the semantic payloads (the writer supplies envelope times):

```json
{"t":1401,"k":"sketch-begin","id":"s1","author":{"id":"v1","name":"Ana","color":"#ff0000"},"owner":"opaque-session-digest","doc":1,"tab":"tab-1","vp":{"w":1280,"h":800,"dpr":2}}
{"t":1402,"k":"sketch-stroke-begin","sketch":"s1","id":"p1","at":{"node":7,"kind":"element","path":["main","section#plans"],"box":[100,200,600,400]},"width":3}
{"t":1403,"k":"sketch-points","sketch":"s1","id":"p1","chunk":0,"pts":[[110,220],[180,240]]}
{"t":1404,"k":"sketch-stroke-end","sketch":"s1","id":"p1"}
{"t":1450,"k":"sketch-label","sketch":"s1","label":"Put the price beside the plan name"}
```

`owner` is a public digest using the existing note-owner mechanism, never the cookie or an editing credential. Author/name are attribution; HTTP requests cannot self-assign operator identity. Colour comes from the current server-validated pointer palette selection, frozen per sketch; changing the pointer swatch applies to the next sketch. **Decision needed:** recommend freezing colour for one sketch for readable history; alternatively propagate later swatch changes with a new explicit event.

At begin, the recorder checks live/document/capability and emits `sketch-begin` with a timestamp strictly greater than preceding records, then fences later records to `> begin.t`. Its acknowledged operation returns the sketch ID only after the sink accepts it. Thus reducing the recording through `begin.t` is exact even with split-file merging, without a format-wide sequence field. Test same-millisecond DOM/notes/pointer/begin interleavings in both the production writer and reference Writer. No subsequent client can choose the source time.

Board keeps a bounded immutable source State for each active sketch, captured from the ordered stream at `sketch-begin`. Stroke anchors are built with `notes.Structural.Anchor` against that State. The client supplies a frozen-renderer node ID and observed box; it does not supply trusted selector/text/path evidence. Require a finite positive anchor box and validate element identity against the frozen source. Choose the element under the first point, allowing the viewer to choose a containing page element through the existing hit-test behavior; all points in that stroke use that anchor. A line extending outside its element is allowed, since redesign arrows often do. There is no inferred shape or inferred second endpoint anchor.

### Decision needed: arrows spanning independently moving elements

**Recommend:** one anchor per stroke, selected at its start. A line connecting two independently reflowing elements follows its first element as one shape. Users can split a drawing into strokes on separate elements. Multi-anchor deformation would distort shapes and introduces another editing model. Agent output retains original geometry and every stroke's anchor so the intended relationship remains inspectable.

## Streaming transport and authority

Add `POST sketch-begin`, `sketch-stroke-begin`, `sketch-points`, `sketch-stroke-end`, `sketch-undo`, `sketch-redo`, `sketch-label`, and a read-only POST `sketch-status` for exact request reconciliation. Reuse castlock, `link.annotates()`, annotation generation headers, admission/body deadlines and generic refusal mapping. New routes use focused parsers, not a larger switch in the already substantial annotation mount.

Every mutation carries a random client operation ID plus sketch ID (begin allocates one), and a per-sketch monotonic operation number. Server identity is `(cast, authenticated owner, sketch, operation)`. The ordered writer stores its outcome in bounded in-memory receipts; the corresponding event carries the operation identity and body digest so retry outcomes can be rebuilt from the recording after reconnect. Exact repeats return the same result; another body under that identity is rejected. The same numbered command is the only retry allowed while outcome is unknown. Status returning “not observed yet” is not proof of absence: a writer barrier must confirm that the operation was refused before reusing its slot.

Only one in-flight command per author sketch; chunks are sent FIFO, at 50 ms cadence with a bounded client queue. Undo/Redo wait for a stroke-end acknowledgment. The first point is drawn optimistically; echoed points are keyed by stroke/chunk and replace the optimistic copy. A failed or unconfirmed chunk is visibly marked, never quietly treated as shared. Do not copy the fire-and-forget pointer POST behavior.

The resource proving exclusive authoring is one active-sketch lease per authenticated owner, held from accepted begin through label/cast end/revocation. A second tab with the same owner must join/resume that exact sketch or finish it; it cannot begin an independent competing sketch. Undo/Redo check the owner digest, current lease and exact expected stack top at the writer, not merely the claimed viewer ID. Existing viewer attribution can be spoofed; it is not erase authority.

On network disconnect the lease remains until reconnect or a 2-minute idle timeout. Reconnect refreshes current state and resumes the same operation numbers. On lease expiry, freeze the accepted partial result as unfinished, release the snapshot, reject late chunks, and retain its history. Replacing the launch or rotating the link cannot grant a new session ownership of someone else's sketch. **Decision needed:** recommend no ownership recovery across a rotated/expired authenticated session; retain the unfinished evidence and start a new labelled sketch. Alternative requires a separate explicit operator recovery grant.

## State, interruption and recovery

`NextSketch(state,event)` owns phases `drawing`, `label_pending`, `complete`, `interrupted`, `cleared`; `drawing` contains either no open stroke or one open stroke. Stroke history is an ordered visible stack plus a redo stack. Events are explicit typed variants, not independent booleans. Board/UI callers cannot directly mutate authoritative stacks. A new stroke clears redo at accepted stroke-begin, so a failed first chunk does not accidentally resurrect the old branch.

| State and event | Next state / effect |
|---|---|
| absent + accepted begin | drawing, reserve lease/source snapshot |
| drawing/idle + stroke-begin | drawing/open; clear redo, register immutable anchor |
| drawing/open + next points chunk | drawing/open; append points |
| drawing/open + duplicate chunk | unchanged if identical; reject conflicting body |
| drawing/open + stroke-end | drawing/idle; seal stroke, push nonempty stroke onto visible stack |
| drawing/idle + Undo(top) | drawing/idle; move top visible stroke to redo |
| drawing/idle + Redo(top) | drawing/idle; move top redo stroke to visible |
| drawing/idle + leave tool | label_pending; request label locally, no extra recording event required |
| drawing or label_pending + valid label | complete; release source snapshot and lease |
| any active + pause/disconnect | hold; finish no new stroke, retain accepted partial points and reconciliation |
| held + same-doc resume/reconnect | drawing with existing stacks, restart held scene from recorded begin moment |
| active + navigation/cast end/lease expiry | interrupted; seal accepted partial points, keep unlabeled evidence, release lease |
| active + annotation revocation/clear | cleared according to existing clear barrier; late operations rejected |
| interrupted/complete + late points/Undo/Redo | reject; never attach to a newer sketch |

Actions for the following table: reconnect, retry exact operation, resume cast, start new sketch, enter label, Undo, Redo, inspect recording.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Local samples queued, transport connected | drawing; local points distinguished until echoed | continue drawing | none within admitted bounds |
| Lost acknowledgment, write may have committed | reconnecting; current gesture stops | retry exact operation | unsent samples if tab closes |
| Writer definitely refused a chunk | stroke interrupted; accepted portion stays | Undo, then draw again | rejected/unsent part of the stroke |
| Same-document pause | paused; sketch retained | resume cast | no accepted work |
| Source document replaced | unfinished sketch in list, no current-page overlay | start new sketch | unsent samples; pending label may be absent |
| Tab closes before label | label pending/interrupted in recording | reconnect while lease lives; otherwise inspect recording | local unsent samples and unentered intent |
| Anchor absent/ambiguous/hidden | list item says placement unavailable | inspect recording at sketch moment | no stored paths or intent lost |
| Label blank or invalid | label required; sketch remains unfinished | enter label | none |
| Unknown image job outcome | image unavailable with reason | retry image export into a new output directory | no recording data |
| Cast-wide annotation clear | overlay and current list cleared | inspect recording before clear | earlier history stays; unsubmitted samples disappear |

Navigation/clear/end are the most dangerous ordering: a late chunk must never recreate a cleared sketch or appear on a replacement document. Sequence tests force these events between admission, dispatch, sink write and reply. Process death cannot undo written chunks. An incomplete final JSONL line is reported as incomplete evidence, not silently repaired or counted as a finished labelled sketch.

## Coordinates, placement and replay

All recorded points are top-page CSS pixels at the sketch's begin moment, independent of device pixel ratio and viewer scaling. The held drawer uses existing `toPage/fromPage`; browser-window resize changes only the fit transform. The original page viewport and nested scroller offsets remain held. No page scripts execute in the held replay.

For point `(px,py)`, recorded anchor box `(x,y,w,h)` and current box `(nx,ny,nw,nh)`:

```text
projected = (nx + (px-x)*nw/w, ny + (py-y)*nh/h)
screen = fromPage(projected, viewerScale, currentTopScroll)
```

Extract the common point/box affine transform beside existing `projectRegion`; do not duplicate note placement traversal. Extend placement to expose the resolved anchor box and clip so sketches reuse the same ancestor-scroll, shadow-root and frame handling. Reuse original node identity first and the existing confidence-scored replacement match otherwise. Refuse ambiguous or unsupported transformed-frame placement; retain the list item and historical paths. Width remains 3 CSS px in the page overlay, then scales with the viewer; it does not stretch with the anchor's aspect ratio. Dots use a round cap.

The held drawer keeps its current Renderer bound to a frozen Model at begin. `connect` advances a cloned live Model for incoming records but suspends page rendering; sketches and capability notifications still update. No unbounded incoming-record queue. On leave, destroy the held scene and reconnect for the existing fresh catch-up. Snapshot cloning must preserve Maps and graph references and be tested for independence. After reconnect, rebuild the held scene from the recorded begin prefix through a bounded snapshot response; this response is private to the owner and reuses existing sandboxed renderer modules and asset proxy.

Undo/Redo are timeline events. Seeking before Undo shows the stroke; after Undo hides it; after Redo restores it. `annotation-clear` additionally clears sketches, preserving prior replay positions. Navigation retains the cross-document list but only confidently places current-document sketches. Go and JS CatchUp retain sketch begin/chunks/history required by current items across resume and navigation; they may remove cleared items only after preserving the clear barrier. Agent historical queries always read the original recording.

## Agent view and image evidence

Extend `couch --broadcast-web-query DIR sketches [--sketch ID] [--json]`. Text lists each sketch's label or label-pending status, author, colour, source document/time, visible/undone stroke counts, anchors as `@N`, and the command for image export. JSON includes the immutable page-coordinate paths, per-stroke anchors/boxes, current Undo/Redo projection, source moment, label event time and status. `events` also names sketch lifecycle events. The recording is sufficient; no running Couch is required for text/JSON.

Add `sketch-image --sketch ID --out DIR [--at T]`: default uses the latest accepted stroke state and label, placed on the DOM at the sketch's source moment. `--at` chooses the sketch history cutoff, not a replacement DOM. Render with the existing Model/Renderer/Replayer, recorded assets only, page scripts disabled. Compose the chosen visible paths over the historical page; crop to their union plus 16 CSS px, clip to the recorded viewport, and emit `sketch.png` plus `sketch.json` with coordinate transform, source identity and limitations. Empty/fully-undone sketches produce an explicit empty result, not a fabricated image. A path outside the captured viewport stays in JSON and is reported as clipped.

Images are replay renderings, not original screenshots. Label them as such. Preserve existing uncaptured canvas/video/frame placeholders and missing-asset warnings. Never query today's live tab for substitute pixels. When historical rendering is unavailable, text/anchors/paths still succeed and the image result states why. Refactor the existing headless replay owner into a shared focused helper; fidelity and sketch export both use it. Image cancellation closes Chrome, the loopback server, CDP and temporary profiles before returning.

**Decision needed:** recommend explicit local CLI export for this version, rather than extending the existing Ctrl+Shift+Alt+Return note-submission bundle. The agent already reads the recording, satisfying the project's stated handoff. If the operator requires one-chord sketch submission alongside notes, that must be a separate approved implementation slice covering selection, privacy and acknowledgment semantics, not an implicit reuse of note-specific captures.

## Operating envelope and lifetime

Budgets below are initial design choices to confirm at review, not measured performance claims (ARCH-CONSTRAINTS):

| Driver | Proposed bound / behavior |
|---|---|
| Interactive sampling | existing 50 ms batches, max 64 points/batch; local draw on next animation frame |
| Points | max 4096 per stroke, 16384 per sketch; reaching the cap stops the stroke visibly, keeps admitted work |
| Strokes | max 64 per sketch, including undone strokes; Undo does not free historical capacity |
| Active sketches | max 4 cast-wide, one per authenticated owner; refuse further begin with a clear busy message |
| Frozen scene | max 8 MiB encoded source DOM/state per sketch, 32 MiB cast-wide; refuse preparation before reserving if too large |
| Command queue | 8 chunks/client, existing bounded server in-flight admission; stop sampling visibly on overflow, do not discard an unknown write |
| Mutation body | max 16 KiB chunk; max 1024 UTF-8 bytes for label; finite coordinate magnitude <= 1e7 CSS px |
| Shared drawing traffic | retain current cast-wide stroke-rate budget initially; sketch traffic shares it with pointer drawing |
| Durable sketch growth | max 32 MiB sketch events and 256 sketch begins per recording, enforced by writer before append; exhausted cast can view/query existing sketches but refuses new sketch commands |
| Receipt/source lifetime | receipts reconstructed from bounded sketch history; immutable source snapshots released at completion/interruption/clear/end; idle leases expire after 2 minutes |
| Image jobs | one Chrome per CLI invocation; <= 30 s, <= 4 MP image, <= 8 MiB PNG; recording input <= 128 MiB; fail visibly at bounds |

Growth drivers are DOM nodes D, retained sketches K, strokes S, points P, chunks C, viewers V and recording records R. Input sampling/chunk append is O(new points), with O(1) ID/chunk lookup. Undo/Redo is O(1) stack update. A dirty overlay repaint is O(visible P), coalesced once per animation frame, not one full repaint per point. Anchor candidates are scanned once per DOM/layout invalidation and reused per distinct anchor; avoid a full DOM scan for every point or every stroke. Snapshot capture/clone is O(D) once per active sketch, not per chunk. Stream fan-out is O(V) per chunk using existing queues. Offline image/query work is one recording pass O(R + selected P); avoid replay-from-start per stroke.

Counted tests at N and 2N must assert: N/2N chunk appends visit exactly newly appended points; repeated Undo/Redo does no point replay; 8/16 strokes sharing an anchor cause one candidate scan per invalidation; one paint is scheduled for 8/16 chunks in the same frame; query reduces R records once for 8/16 stroke answers. Rendering each growing stroke after every single sample would fail these counters even if timing happened to look fast.

The recorder creates event lines; replay and agents are their last consumers; deleting the recording directory removes them. Existing recordings have operator deletion rather than automatic retention; the added 32 MiB writer cap prevents unbounded sketch residue per recording (ARCH-FUNERAL). No persisted source-snapshot family is added. The local held scene and canvas die on mode exit/disconnect/end. Explicit image output belongs to its caller until deleted; temporary rendering directories are removed on every result. Export refuses existing output files and never writes into the source recording. Tests prefer TestScratch through TMPDIR and use private temporary directories.

## Format evolution and trust

Old version-1 viewers ignore the new event kinds and continue showing the DOM; they do not gain sketch rendering. Test with a frozen pre-feature decoder/model, including a whole recording passing `ReadDir/Validate`. No extra file and no meaning change to `stroke`, `note` or existing `annotation-clear` for old readers. New readers apply clear to sketches as an additional collection.

New sketch events preserve unknown members, including nested anchors/path members, via a raw-preserving wire representation attached to typed projections. Encode overlays only explicitly changed known fields on the preserved object. HTTP command objects remain closed/strict; persisted event objects allow unknown members but reject duplicate keys, wrong known types and missing required members. Since existing known-kind preservation is incomplete, do not claim a repo-wide repair: no existing record kind gains fields in this design. Test unknown-kind old-reader round-trip and new-reader nested unknown-member round-trip. Old catch-up can drop unknown events during compaction; its DOM replay remains compatible, but it must not be used to rewrite or recover sketch history. State that limit in the atlas (ARCH-EVOLVE).

Trust boundaries: locked web clients, isolated operator projection, JSONL loaded from disk, local export arguments and replay browser output. Validate finite numbers, IDs, counts, monotonic chunk indices, duplicate/conflicting operations and ownership before effects. Treat labels as text, never HTML. Paths are numeric polylines, never client-supplied SVG markup or script. Preserve existing origin privacy: recording and exports are private evidence unless explicitly granted otherwise. Do not automatically publish images into this public repo (ARCH-SECURE).

## Proposed implementation slices after approval

These are proposed issue boundaries, not Mx tags for design issue #750. File them after operator approval, with dependency on the prior landed slice. Each gets its own durable execution plan and required SDLC gates; no stacked development. The tasks below provide concrete file/test/acceptance direction for those plans.

### Slice A: recording model and anchored geometry

**Create:** `recording/sketch.go`, `recording/sketch_state.go`, `recording/sketch_test.go`, `recording/sketch_state_test.go`, `sketch/geometry.go`, `sketch/geometry_test.go`, `web/sketch-model.js`, `tests/webcast/sketch-model.test.mjs`, shared `tests/webcast/sketch-corpus.json`.
**Modify:** `recording/record.go`, `codec.go`, `reduce.go`, `catchup.go`, `web/model.js`, `web/replay.js`, existing codec/order/catchup tests and corpus.

- [ ] Write failing tests for streamed chunk joins, duplicate/conflicting/out-of-order chunks, empty/incomplete strokes, Undo/Redo/new-branch behavior, source begin timestamp fencing, label completion, clear and source-document retention. Include nested unknown fields and old-reader compatibility.
- [ ] Run `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast/sketch -run 'Sketch|CatchUp|Order' -count=1` and `node --test tests/webcast/sketch-model.test.mjs tests/webcast/replay.test.mjs`; confirm failures identify absent sketch semantics.
- [ ] Implement the event union, raw-preserving codec, pure transitions, geometry, reducer and catch-up. Independently stated invariant: only the active author's selected last stroke changes; an undo never deletes earlier timeline evidence.
- [ ] Rerun targeted tests plus existing codec/catch-up corpus tests; compare Go/JS results for every shared sequence prefix, including pause/resume/navigation.
- [ ] Document the additive format and compatibility limits in `atlas/webcast-format.md`; commit with issue, Co-Authored-By and Agent trailers. Close/review this slice through SDLC before consumers depend on it.

### Slice B: acknowledged transport and live drawing

**Create:** `board/sketch.go`, `board/sketch_wire.go`, `board/sketch_test.go`, `recorder/sketch.go`, `recorder/sketch_test.go`, `server_sketch.go`, `server_sketch_test.go`, `web/sketch.js`, `web/sketch-policy.js`, `web/sketch-scene.js`, `web/sketch.css`, corresponding `tests/webcast/sketch*.test.mjs`, `sketch_live_test.go`.
**Modify:** `board/writer.go`, `board/operator_view.go`, `cast.go`, `browser.go`, `server.go`, `web/viewer.js`, `web/annotate.js`, `web/annotation-placement.js`, `operator.js`, `operator_script.go`, existing fake browser/drive adapters.

- [ ] Write failing parser/admission/identity tests and stateful writer tests that hold dispatch across navigation, clear, lost replies and revocation. Cover second-tab lease collision, queue overflow and exact retry idempotency.
- [ ] Run `go test ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder ./cmd/internal/webcast -run 'Sketch|Annotation|Writer' -count=1`; confirm the new cases fail before implementation.
- [ ] Implement acknowledged begin/chunk/end/undo/redo/label/status calls, exact source-moment snapshots, budgets and leaf-lock discipline. Generalize the existing acknowledged writer plumbing where shared; do not clone the complete note capture service.
- [ ] Write failing JS sequence tests for optimistic echo replacement, missing ack, reconnect, label prompt, pointer colour, local hold and late page changes. Implement thin mount/policy/scene modules; use existing pointer chunking and geometry helpers where their semantics match.
- [ ] Run `node --test tests/webcast/sketch*.test.mjs` and targeted Go tests; the two-viewer browser test must observe the first incomplete stroke on viewer B before viewer A releases pointer-up, then Undo disappearance and Redo restoration.
- [ ] Count N/2N hot-path work as specified above. Test touchscreen scaling, nested scroll, frame/shadow anchors, local browser resize, source page reflow, and missing/ambiguous anchors. Verify operator projection sees persistent sketches without turning them into fading pointer trails.
- [ ] Update `atlas/webcast.md`, `atlas/webcast-format.md`, root README and `atlas/index.md` for the controls, holds, limits and failure behavior; commit. Run required recorded checks on pushed head, then SDLC close/review.

### Slice C: agent queries and historical image export

**Create:** `query/sketch.go`, `query/render_sketch.go`, their tests, `sketch_image.go`, `sketch_image_test.go`, `sketchimage/contract.go`, `sketchimage/fake.go`, `sketchimage/contract_test.go`, `web/sketch-image.js` and its JS tests.
**Modify:** `query/surface.go`, `query/fake.go`, `query/cli.go`, `query/events.go`, `query/contract.go`, query golden/agent tests, `fidelity_check.go`, query wiring in `cmd/internal/couchcmd/`, shipped Couch skill query guide.

- [ ] Write failing CLI/contract/golden tests for finished, unfinished, undone, redone, cleared, cross-document and missing-anchor sketches. Run `go test ./cmd/internal/webcast/query -run 'Sketch|Surface|Agent|Golden' -count=1` and confirm missing-command/model failures.
- [ ] Implement one-pass sketch query and render input selection. Assert source DOM uses begin time while paths/label use the selected later history cutoff. Include raw page paths and transform in JSON.
- [ ] Define ImageRenderer and its stateful fake first; write contract cases for success, missing asset, timeout, cancellation, bad PNG, bounded output and existing destination refusal. Extract/reuse the headless renderer owner and implement real historical export behind that seam.
- [ ] Run `go test ./cmd/internal/webcast/sketchimage ./cmd/internal/webcast/query ./cmd/internal/webcast -run 'Sketch|Fidelity' -count=1`; verify existing fidelity tests still pass after extraction. Run a bounded headless fixture through the real renderer and compare crop bounds and coloured path pixels with independent expected geometry.
- [ ] Update `atlas/webcast-format.md`, `atlas/webcast.md`, README and shipped `couch --skill` query instructions; explain replay-rendered pixels, unfinished labels and private exports. Commit, recorded check, SDLC close/review.

### Slice D: end-to-end operator acceptance

- [ ] Automate a fixture with two viewers, navigation, scroll, resize, frames, a held sink and a disconnect. Verify the recording validates, first partial stroke reaches the other viewer, Undo/Redo replay correctly at every event boundary, and query/image evidence identifies the intended elements.
- [ ] Run `go test ./cmd/internal/webcast/... -run 'Sketch|CatchUp|Conformance|Fidelity' -count=1` plus JS tests. Push the final head and run `sdlc local-check run --name affected` where declared by this repo. Each landing still follows its integration check; do not substitute a design-document check for feature verification.
- [ ] Ask the TL to arrange the real xianxu.dev cast with the operator. Draw and label a proposed change, observe streaming from a second viewer before pointer-up, Undo/Redo, scroll and resize, replay after navigation, and export the recording's image/paths. Give that evidence to a coding agent in the echo design process and verify it can state the requested change and anchor elements before building.
- [ ] Record actual browser/device, recording ID, commands/results and retained image locations in the implementation issue; summarize to the TL, who owns the project file. The project done-when requires this live acceptance, not just a headless fixture.

## Architectural check

- **ARCH-DRY:** reuse anchors, permission generation, rate admission, pointer colour/chunk cadence, ordered writer, renderer and headless lifetime; sketch-specific history has one reducer per language, held to shared fixtures.
- **ARCH-PURE:** geometry, validation and history transitions are pure; HTTP/CDP/canvas/files are thin adapters.
- **ARCH-PURPOSE:** actual live partial-stroke visibility, Undo/Redo, required intent label, recording/replay, all viewers, operator display and agent image/paths are covered; no substitution of an atomic final-image broadcast.
- **ARCH-MOCK:** reference Writer/fake Browser and stateful image job fake share production seams; real headless conformance plus operator acceptance test behavior the doubles model.
- **ARCH-CONSTRAINTS:** explicit point, history, scene, traffic, queue and image budgets; counted N/2N tests address growth rather than timing alone.
- **ARCH-SECURE:** authenticated session owns mutation; viewer ID is attribution; numeric paths and plain-text labels; strict persisted inputs and private exports.
- **ARCH-ORDER:** NextSketch owns stack changes; writer fences begin and late commands; unknown outcomes retain exact identity; interruption sequences are executable tests.
- **ARCH-FUNERAL:** event history dies with recording, bounded at writer; frozen scenes/leases/jobs have named release paths; exports have explicit caller ownership.
- **ARCH-EVOLVE:** additive event kinds in existing events file, old reader tests, nested raw-field preservation for new records, no unsupported universal preservation claim.

## Decision needed: approval and issue boundaries

Approve or revise the recommendations collected above, especially held drawing view, label timing, operator Chrome controls, post-finish Undo scope and handoff via CLI. Then file the implementation slices with the approved scope and dependencies. Do not run `sdlc change-code` or start implementation from this unapproved plan. No estimate has been invented before the settled plans clear their plan-quality gates.

## Revisions

- 2026-10-20: operator direction relayed by ops:0 supersedes TL's earlier atomic local-draft preference. Current proposal streams all stroke chunks, adds author Undo/Redo, and uses pointer colour. The atomic-save alternative would reduce protocol complexity but fails the operator's stated need to watch a sketch form, so it is rejected for this version.
