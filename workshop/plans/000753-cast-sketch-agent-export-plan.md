---
collaboration-mode: discussion
---

# #753 — Historical sketch evidence in operator submissions

**Current execution contract:** the approved 2026-10-22 shared-screen revision under Revisions below supersedes the historical per-sketch submission/image and preview clauses. The earlier implementation and its CLI/export contracts remain the baseline, not instructions to send individual sketch PNGs in new bundles.

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. This is a proposed full-flow plan; TL approval and change-code precede implementation.

**Goal:** Let the operator submit finished sketches to the pinned coding agent, each with its historical rendered image, note, original page URL and exact recorded element identities; expose the same facts through local queries/export.

**Architecture:** Extend the existing operator draft submission transaction with typed sketch evidence. A pure recording query freezes source facts at sketch begin and accepted vector/label history at selection; one injected replay image renderer serves CLI exports and submission preparation. Existing durable bundle delivery is the sole acknowledgment owner.

**Tech stack:** Go recording/query/submission core, shared JavaScript pressure painter and sandboxed replay, CDP/headless Chrome, existing Neovim operator submission transaction.

**Status:** TL approved the core design and both product choices (delivery 745ea763-8e84-4ac2-8897-f40e82a884c0), with the preview requirement below. No implementation before change-code clears. Canonical source of revised scope: issue #753 Revisions, TL deliveries cbb63f0f and 0f71ee20 (2026-10-21).

## Operator contract and alternatives

1. **Recommended:** press the existing Shift+Ctrl+Alt+Return in the operator draft. Completed unsent sketches join the existing open-note submission. The same authored message, pinned agent target, prepare/progress/send/ack transaction and recovery controls apply. No extra browser button or delivery route.
2. CLI-only export preserves the original narrow issue but leaves the operator manually assembling an image/text package. It fails the newly requested natural submission flow (ARCH-PURPOSE).
3. Browser-to-agent submission would add competing target/delivery authority. Reject in favor of the existing operator transaction (ARCH-DRY).

A submitted shape is one finished, labeled sketch, including its accepted visible strokes. Two separately finished sketches produce two images and two labels. Unfinished, cleared and entirely undone sketches remain queryable with explicit status but are not auto-submitted. Labeled interrupted sketches with visible strokes remain eligible; interruption/truncation is explicit. Submission never clears the ink or changes its editing state.

Proposal for bounded batches: at most eight sketches per preparation, ordered by creation; existing note inclusion remains unchanged. Status says how many sketches are in this batch and how many remain, and successful dispatch makes the next batch eligible on the next operator submission. An over-byte-budget or failed render refuses preparation with the draft/evidence retained; no shape disappears silently. The eight-sketch envelope and oldest-unsent-first ordering are TL-approved; meeting the existing deadline still requires measurement.

## Before-send preview (TL acceptance addition)

The console Chrome private UI displays Next submission: N sketches, M more, with a compact pressure-ink thumbnail and attached note for each selected shape. Thumbnails show the shapes, not newly captured live-page pixels; final evidence uses the historical renderer described below. Preview cards are keyed by exact sketch version, so metadata refreshes cannot replace an unrelated label editor or steal focus.

One immutable PreviewSelection owned by the existing submission manager carries ordered sketch/version keys, count/remaining and a monotonically revised delivery identity. It is a view of the same pure candidate selector used by preparation, not a second queue or delivery authority. The existing private console projection publishes it alongside existing sketch display data and acknowledges that revision; one coalesced owned worker refreshes on sketch metadata or store delivery changes, never a polling headless renderer. Superseded/retired host acknowledgments cannot enable a new preview.

For sketch-containing submissions, preparation freezes that displayed/acknowledged preview. If a selected version was cleared/changed or the preview is not current, preparation refuses visibly (Selection changed; review the updated preview and submit again), refreshes the preview, and leaves draft/ink untouched. Later changes after preparation do not mutate the frozen bundle. Ordinary note-only submissions retain their current flow and CLI query/export need no preview. Browser preview is read-only: it cannot choose an agent, begin delivery or ACK a send; only the existing operator chord does that.

Add pure preview state/selection tests and native acceptance that the operator can see all selected shapes/count before pressing the chord, including eight oldest + N more, after navigation, and stale-preview refusal. This is the approved acceptance extension; no second send button or browser-to-agent route.

### Preview transition ownership (PQ-1)

`submission.NextPreview(state, event) -> (state, effects)` is the pure authority. `PreviewState` has private tagged payloads: Dormant (no host/candidate), Dirty (latest host and candidate), Publishing (one immutable host/candidate/revision/operation), Shown (that exact acknowledged projection), and Stopped. Callers cannot write its fields. Only the manager-owned wrapper applies NextPreview; CDP and board callbacks enqueue typed events, never assign readiness.

| State + event | Next state | Effect / invalidation |
|---|---|---|
| Any except Stopped + HostChanged | Dirty or Dormant | Retire old publication identity; cancel its IO and publish on the new host only. |
| Dirty/Publishing/Shown + CandidatesChanged | Dirty | Replace latest candidate stamp, invalidate any shown/old operation; coalesce refresh. |
| Dirty + Refresh | Publishing | Issue one read-only projection with fresh operation identity. |
| Publishing + exact PublishAccepted | Shown | Accept only matching host generation, candidate stamp, revision and operation. |
| Publishing + exact PublishFailed | Dirty | Show preview unavailable; retry on explicit prepare/refresh or a later source event, no retry loop. |
| Any + stale publication result | Unchanged | Ignore; cannot restore a retired preview. |
| Shown + Prepare | Shown or Dirty | Try the atomic freeze below; success creates existing preparation job, mismatch invalidates and refreshes. |
| Dormant/Dirty/Publishing + Prepare | Unchanged | Refuse sketch send with review-preview guidance; request refresh when possible. |
| Any + Dispose | Stopped | Cancel/join the sole refresh worker and discard view state. |
| Stopped + any other event | Stopped | No effects or revival. |

The wrapper owns the current model under submissionManager.mu. Existing request flow already takes this mutex before reading board selection. Add a cheap Board candidate-generation stamp that changes only when eligible sketch metadata/clear changes; never on pointer/miscellaneous page activity. The board provides `FreezeSubmissionSelection(expectedGeneration)` under board.mu, returning an immutable selection only if the generation matches. The manager holds its mutex across policy Prepare and this short board call, so host/delivery events cannot interleave; order is manager.mu then board.mu, and board notifications never call the manager under board.mu. No filesystem, CDP, vector copy or rendering occurs under board.mu.

Delivered-version facts are derived/validated outside board.mu by the existing preparation/preview owner. Manager.mu serializes its confirmed store-transition generation with the preview. The candidate stamp combines board generation and delivered-set generation. Once a durable transition changes suppression, the manager invalidates the preview in that same serialized step. There is no external writer API for this Couch-owned store; malformed or externally modified manifests encountered while deriving it refuse publication, not permission to silently resend.

`submissionPreview.freeze` is the only constructor of an opaque `FrozenPreview`; sketch preparation requires it, while note-only preparation keeps its existing path. Thus a caller cannot bypass Shown or obtain a sketch selection from the ordinary unguarded Board.Selection call. After freeze, later mutations cannot change that job's immutable selected prefix.

The refresh worker is one Console.GoTracked child with a lifetime-derived context and a capacity-one wakeup channel. It performs projection IO outside locks, returns results through typed events, and reads the newest candidate after a coalesced invalidation. Host replacement cancels the owned publication context; console disposal cancels and joins the worker through existing tracked lifetime. A canceled/late operation has no write authority even if its CDP reply arrives. Tests drive NextPreview with controllable order and drive the production wrapper with a blocked fake publisher/board freeze guard; no test relies on a scheduler race.

## Core concepts

| PURE entity | Lives in | Status | Contract |
|---|---|---|---|
| Querier.SketchesAt / SketchView / SketchSelection | cmd/internal/webcast/query/sketch.go | new | One ordered recording scan freezes original source facts and the exact accepted history record boundary; no live DOM or file access. |
| DeclaredElementFacts / DeclaredElementFactsAt | cmd/internal/webcast/query/element_facts.go | new | Owned bounded literal declarations; select anchors before ancestors and nearest context without letting unselected nodes spend the output budget. |
| RenderPlan / Path / Plan | cmd/internal/webcast/sketchimage/contract.go | new | Original pressure geometry plus a separately validated viewport/crop/scale transform; no IO. |
| Batch / Image / ValidateBatch / ValidateImages | cmd/internal/webcast/sketchimage/render.go | new | Values and deterministic validation for exact source/history identity, complete plans and bounded PNG content. The ImageRenderer effect is listed separately below. |
| Submission / Sketch / SketchSource / SketchImage | cmd/internal/submission/model.go, sketch.go | modified/new | Typed historical note/image/page/anchor/vector evidence; no writer admission owner or receipts. |
| PreviewState / NextPreview / PreviewSelection / PreviewPublication | cmd/internal/submission/preview.go | new | Pure transition authority and immutable publication identities; effects request IO/freeze but do not perform them. |
| PreviewItem / SelectSketchPreview | cmd/internal/submission/preview.go, preview_selection.go | new | Oldest-eight selection from provided candidate and delivered-version values; exact key matching and remaining count, no filesystem derivation. |
| Selection / SelectedSketch / SketchCandidate | cmd/internal/webcast/board/submission.go, recording/sketch_state.go | modified/new | Owned selection values and cheap ordered accepted-prefix metadata, separate from the Board's locked freeze operation. |
| Validate / ValidateSketch / EvidenceFiles / preserving codec | cmd/internal/submission/model.go, sketch.go, codec.go | modified/new | Semantic evidence validation, one referenced-file inventory, and strict typed JSON with nested extension preservation. |
| Build / ContextAt | cmd/internal/webcast/submission/build.go | modified | Project ordinary-note context and declared identities from supplied recording values at the original source moment. |

One SketchView owns many strokes and anchors, one source moment, one selected history cutoff and one label. Source facts are copied at begin, not reconstructed from a later page. No duplicate label state is added; selected version uses the existing accepted-prefix sequence/digest. #783 revisions change that prefix and become eligible without treating an old ACK as newer delivery.

| INTEGRATION | Lives in | Status | Wraps / injected boundary |
|---|---|---|---|
| ImageRenderer / FakeRenderer / RunRendererContract | cmd/internal/webcast/sketchimage/render.go, fake.go | new | Context-bound batch effect with a stateful session/failure/cancellation fake and shared conformance contract. |
| headless replay session | cmd/internal/webcast/replay_image.go | extracted/modified | Owned Chrome/CDP/profile lifetime, used by fidelity and historical image rendering; joined bounded cleanup. |
| sketchImageRenderer / private image server | cmd/internal/webcast/sketch_image.go, web/sketch-image.js | new | Recorded-only sandbox DOM/assets and shared pressure painting through the injected ImageRenderer seam. |
| local image export / exclusive publication | cmd/internal/webcast/sketch_image_export.go, sketch_image_publish_*.go | new | Confined private output, complete image/metadata package and platform no-replace publication. |
| query CLI / SketchImageExporter | cmd/internal/webcast/query/cli.go, surface.go, sketch_export.go; couchcmd/run.go | modified/new | Reads a recording once and injects local export/link effects; existing Run forwards to RunWithOptions. |
| Cast.PrepareSubmission | cmd/internal/webcast/cast_submission.go | modified | Bounded recording read, pure evidence projection, original-origin policy and one injected render batch. |
| Store.DeliveredSketchVersions | cmd/internal/submission/store.go | new | Filesystem-backed scan of durability-confirmed dispatched/resolved manifests across targets; supplies values to pure SelectSketchPreview, never a second ACK source. |
| submissionManager / Board freeze / Store publication | cmd/internal/couchtty/console_submission.go, console_submission_preview.go; webcast/board/submission.go; submission/store.go, publication.go | modified/new | Existing preparation/delivery transaction, coalesced owned preview worker and atomic manager→Chrome host→Board lock order. |
| private preview projection | cmd/internal/webcast/chrome_submission_preview.go, chrome_operator_sketch_binding.go, operator-sketch.js, operator.js | modified/new | Exact host/version/cursor acknowledgment, read-only keyed cards and existing UI-open capture deferral; no browser dispatch. |
| CDP write ownership | cmd/internal/cdp/conn.go | modified | Context-aware serialized admission, joined owned deadline cancellation and partial-frame retirement. |
| operator progress / handoff guide | nvim/dom_submission_state.lua; couchcmd/skills/couch/SKILL.md; submission/model.go | modified | Batch counts/provenance and the existing pinned-target/retry transaction. |

## Source and history clocks

The immutable selection contains recording ID, sketch ID, expected NextSequence and last accepted receipt digest. During the single ordered scan, retain source page URL/title, tab/document, viewport/scroll, anchor facts and original DOM/layout at SketchBegin time. Select the exact accepted prefix through the frozen sequence, including Undo/Redo visibility and label. Later navigation, source updates or labels must not leak into this version. Refuse a missing/changed prefix instead of substituting newer history.

The image renders the original source DOM at begin with only that sketch's selected visible ink. Original x/y/pressure and original anchor boxes remain unchanged in JSON. A separate transform records source scroll/page origin, source crop, output dimensions/scale and clipping; pressure width uses the existing shared .25+.75*p rule. Include a modest 16 CSS-pixel context pad, clipped to the recorded source viewport and four-megapixel output bound. A shape outside that view is explicitly clipped/unavailable; do not scroll an unrelated current tab or claim uncaptured content.

CLI proposal:

    couch --broadcast-web-query DIR sketches [--sketch ID] [--at T] [--json]
    couch --broadcast-web-query DIR sketch-image --sketch ID --out DIR [--at T]

Here --at bounds history, not source page time. Text output identifies both clocks; JSON includes author/color, state/reason, all original anchors and paths, source URL/title and declared IDs. Export refuses existing output, source-recording descendants (including symlink aliases), and nonprivate output. Atomically publish PNG and transform/provenance JSON together; no output directory implies no export.

## Declared element IDs

Reuse one pure extractor for submission, query and export. At the original source moment, include:

- Every stroke's original anchor recording node ID and its literal element id/data-* attributes.
- Anchor ancestors through the first element with a declared id (or root), with a 32-ancestor inspection cap and explicit truncation if exceeded.
- Recorded elements whose layout boxes intersect the original visible-ink bounding box, sorted nearest to its center then recording node ID, retaining at most 64 distinct elements per sketch.

Proposed caps: 20,000 inspected DOM nodes/source moment, 16 data-* attributes/element, 256 bytes/name, 1024 bytes/value and 64 KiB total declared-fact bytes/sketch. Preserve literal bounded values as JSON strings; never infer application meanings, generate a CSS selector, or execute recorded markup. Report truncation/counts and missing-layout facts. Anchor IDs have priority over context candidates; if all mandatory anchors cannot fit, retain every anchor handle and report unavailable declared attributes rather than dropping the association.

The original issue's generic identity requirement also covers ordinary annotations: derive their declared facts at the recorded capture/context moment, not current page state. Browser CDP PageIdentity and draw-time Crop remain unchanged.

## Manifest, privacy and compatibility

Add an optional typed sketches array to the existing version-3 manifest. Each item has sketch ID, immutable selected sequence/digest, author, note, original source {recording document/tab/time, URL/title, provenance: recorded-dom}, anchors/declared facts, selected visible vectors, state/limitations and image {file, replay-rendered provenance, transform}. It is not a Note with a forged capture timestamp or cdp provenance. Keep existing note validation intact.

Keep version 3 because the addition does not reinterpret existing fields. Older stores currently reject unknown manifest fields and thus fail closed; they must not partially ACK a sketch-containing manifest. A compatibility fixture built against the old decoder verifies rejection and original bytes unchanged. The new codec preserves unfamiliar fields at every nested changed object during Prepare/Transition/Confirm/rewrite; strict required fields, duplicate keys and known types remain enforced. Unknown evidence references are never interpreted or deleted. Do not relax validation by replacing DisallowUnknownFields with lossy generic decoding.

Centralize referenced-file enumeration for notes AND sketch images. Sweep readMetadata, readSubmission, Prepare, ConfirmPrepared, immutable comparisons and any pruning/retention consumer, with symlink/file identity checks unchanged. No independent list of images in each caller.

Origin policy is evaluated per original source page. Missing/invalid origin information is private; one private item makes the aggregate private. Recorded facts and URLs are untrusted evidence, not instructions. Existing agent guide restrictions apply to all sketch images and text: only explicitly shareable evidence plus repository asset policy permits attachment to public issues; no policy editing by agents. Submission remains local until the operator's existing action authorizes sending to the pinned agent.

## Delivery and visible-state contract

Actions: submit draft, inspect progress, retry/status the same submission, choose/reconcile target through existing UI, submit next batch after accepted delivery.

| Internal state/event | Visible state | Recovery/effect | Possible loss |
|---|---|---|---|
| Preview unavailable or selected version changed | Review updated selection | Refuse sketch prepare, refresh cards; no send or draft mutation | None |
| Finished unsent sketches + submit | Preparing N sketches, M remaining | Freeze selection and target, render once in existing worker | None; draft untouched |
| Render/read/asset/budget refusal | Preparation failed with reason | Retain draft/selection; retry existing transaction | No delivery/ACK |
| Prepared immutable manifest | Ready to send | Existing begin/send transaction | None |
| Delivering + lost ACK | Delivery unconfirmed | Existing same-ID status/recovery; no automatic replacement send | May already have reached target |
| Dispatched/Resolved | Sent | Filter only exact delivered versions from future selections | No ink/label deletion |
| Old ACK after new label version | Older version sent | New version remains eligible | No false acknowledgment |
| Navigation during preparation | Preparing original source | Render recorded begin page and selected history | None |
| Target changes before send | Existing target mismatch | Existing refusal/reconciliation; never redirect silently | No new-target send |
| Restart/disposal | Existing pending transaction | Reconcile publication/delivery from durable manifest | Local unsaved draft governed by existing editor durability |
| More than batch cap | Preparing first N, M remain | Next explicit submission sends remaining versions | No silent omission |

Delivered-version filtering derives from existing dispatched/resolved manifests for that recording, across target incarnations: the operator's successful submission acknowledges those shapes just as ordinary note resolution does. Prepared/Delivering/unknown bundles never suppress eligible versions. No sketch clear, new recording event, acknowledgment file or hidden parallel registry. If an active unresolved submission already exists, reuse existing transaction recovery before creating a new one. Retain shape visual/edit state on ACK.

## Operating envelope, cost and cleanup

- Existing recording input 128 MiB, sketch history 32 MiB/256 sketches/64 strokes/16,384 points each. One read/ordered pass per selection/query; per requested source DOM scan is bounded and explicitly counted. No per-stroke full-recording Reduce call.
- Proposed eight-sketch render batch, one owned headless Chrome, one serial render at a time. Four MP and 8 MiB per PNG; 64 MiB manifest+images; existing one-GiB store and 10,000-entry admission caps. Preserve the existing 30-second preparation deadline/36-second client observer; cap/refuse work rather than bumping timeouts. Measure eight-shape native preparation before claiming this envelope is viable.
- Cost is O(R + D + P + E log E), where R is records visited once, D is total DOM nodes inspected at requested source moments, P selected points, E bounded intersected elements. Different source moments may revisit DOM; do not claim O(R+one DOM). Count visits at N/2N for records/points/same-source requests; group same-source sketches to avoid duplicate snapshot/DOM work. No raster decode on ordinary metadata-only query.
- Chrome/profile/CDP/event drain/replay server belong to one session scope. Context cancellation and every error join all owned processes/readers before return. Stateful fake asserts lifetime; native cleanup verifies actual children/profile/server closure. Fidelity must use the extracted lifetime with unchanged scoring semantics and tests.
- Renderer temp artifacts are removed on return; explicit CLI exports belong to the operator's selected directory. Submission PNGs/manifests retain existing bounded store lifecycle and private permissions; no second cache/store. A bounded in-memory delivered-key map may be derived per request, never written as a second authority. Store deletion removes delivery history as well as evidence and is documented; no false permanence claim.

## Architecture lenses

| Principle | Decision |
|---|---|
| ARCH-DRY | Existing submission target/transaction/store and shared pressure painter; shared file inventory and renderer lifetime. |
| ARCH-PURE | Query facts, ID extraction, render plan, version filtering and manifest validation have IO-free tests. |
| ARCH-PURPOSE | Actual sketch-only operator handoff, not just an export command; all required page/element facts travel with images. |
| ARCH-MOCK | Stateful renderer/CDP fixture behind ImageRenderer; native conformance; existing portable folder-backed submission store. |
| ARCH-CONSTRAINTS | Batch/deadline/pixels/bytes/node budgets with counted costs and explicit refusal. |
| ARCH-SECURE | Recording/store inputs are untrusted; strict preserving decode, source confinement, private outputs, recorded assets only, pinned target. |
| ARCH-ORDER | Frozen source/history version plus existing durable delivery states; unknown ACK never means absent. |
| ARCH-FUNERAL | Joined render session; bounded existing store; temporary profiles removed, exports explicit. |
| ARCH-EVOLVE | Additive typed evidence, old decoder fail-closed, no changed note/crop meanings, unknown nested fields survive rewrites. |

## Function-level verification strategy (PQ-2)

Names below are the intended production entrypoints; colocated tests invoke the real pure functions without IO mocks.

| Function | Adversarial input / mechanical guard |
|---|---|
| `query.SketchesAt(records, selection)` | Fuzz valid/invalid ordered histories and cutoff identities; independent prefix-reducer oracle, counted record/point visits and immutable source witnesses. |
| `query.DeclaredElementFacts(state, anchors, region)` | Generate deep/broad DOM and arbitrary attributes/layouts; exact priority/deterministic ordering, byte/node caps and counted N/2N visits. |
| `sketchimage.Plan(view, limits)` | Fuzz nonfinite/extreme geometry and pressure; finite bounded transform, unchanged originals and independently computed pixel containment. |
| `submission.DecodePreserving / EncodePreserving` | Fuzz duplicate/invalid known members and nested unknown JSON; strict rejection or byte-semantic unknown round-trip across delivery rewrites. |
| `submission.EvidenceFiles(bundle)` | Generate mixed typed references and hostile paths/aliases; one duplicate-free confined inventory used by every file consumer. |
| `submission.UnsentSketches(candidates, delivered, limit)` | Generate exact/stale/uncertain delivery versions and target changes; stable oldest-first bound, no suppression without confirmed exact version. |
| `submission.NextPreview(state, event)` | Model-based event sequences with reordered publication/host/candidate outcomes; no stale Shown or freeze and terminal Stopped invariant. |
| `submissionPreview.freeze / Board.FreezeSubmissionSelection` | Controlled invalidation at the actual lock seam; immutable selected prefix or typed refusal, no observation-to-freeze gap. |
| `submission.ValidateSketch` | Fuzz provenance, source/history, vector/image identity and budgets; reject forged captured evidence and inconsistent transforms. |

Integration contracts exercise the same seams with the stateful ImageRenderer/publisher and portable real Store. Native conformance establishes recorded-only assets, pressure pixels, existing-deadline batch feasibility and actual joined Chrome/CDP/server/profile cleanup. Existing fidelity, note-crop, submission uncertainty and target-pinning contracts remain required.

## Execution plan (after approval)

### Task 1 — Historical query and declared element facts

Files: query/{sketch.go,element_facts.go,surface.go,fake.go,contract.go,cli.go,events.go} and colocated tests. Implement SketchesAt and DeclaredElementFacts through their RED/green strategies above; share the extractor with ordinary annotation context.

- [x] Run the focused new query strategies RED, implement the pure functions and shared Surface contract, then run `go test ./cmd/internal/webcast/query -count=1` GREEN.
- [x] Check counted scan budgets and CLI text/JSON goldens; commit this coherent query slice.

### Task 2 — Shared bounded historical image renderer and CLI export

Files: sketchimage/{contract.go,fake.go,contract_test.go}, replay_image.go, sketch_image.go, web/sketch-image.js, query/render_sketch.go and couchcmd wiring; extraction from fidelity_check.go.

- [x] Drive Plan and the stateful renderer/export contract RED; extract the joined session and implement recorded-only rendering with shared paintSketchPaths and atomic confined output.
- [x] Run `go test ./cmd/internal/webcast/sketchimage ./cmd/internal/webcast/query ./cmd/internal/webcast -count=1`, applicable Node painter/image tests and existing fidelity conformance GREEN; commit.

### Task 3 — Typed evidence and preserving durable publication

Files: submission/{sketch.go,codec.go,model.go,store.go,publication.go}, webcast/submission/{build.go,sketch.go}, cast_submission.go and colocated tests.

- [x] Drive codec/validation/file-inventory strategies RED; implement typed sketches and the single evidence inventory across all publication/rewrite consumers, preserving initial note/crop meanings.
- [x] Build immutable selected evidence from one recording read with the injected render batch; run `go test ./cmd/internal/submission ./cmd/internal/webcast/submission ./cmd/internal/webcast -count=1` and old-decoder refusal/unknown-field contracts GREEN; commit.

### Task 4 — Visible preview, operator transaction and acceptance

Files: board/submission.go, couchtty/console_submission.go, private operator projection/preview files, nvim/dom_submission_state.lua, ReadingGuide, couchcmd/skills/couch/SKILL.md, README and atlas/webcast-submission.md.

- [x] Drive UnsentSketches, NextPreview and the atomic freeze strategies RED; integrate the opaque frozen-preview boundary and read-only pressure thumbnail cards/count. Existing durable delivery transitions remain the only ACK authority.
- [ ] Exercise the production manager/store through the normal-suite disposable Couch fixture and native Chrome acceptance. The operator sees oldest-eight shape previews/count, submits two historical shapes after navigation and the pinned agent receives correct images/notes/URLs/IDs. Verify Claude and Codex; Qoder when credits. Use fixture sinks until authorized real smoke.
- [x] Measure maximum approved batch under the existing deadline; revise the envelope with TL if it fails rather than bump timeouts. Verify joined renderer/publisher disposal and stable input focus under updates.
- [ ] Update guides/atlas, audit reader inputs and run `make test-affected-inventory`; push, record required affected/suite, close through SDLC, stage by hub authorization, run the real operator preview-and-submit smoke, then land on acceptance. No Mx tags unless independently closed milestones are agreed.


## Revisions

### 2026-10-22 — shared visible-viewport feedback (approved; execution delta)

Authority: TL a5b94b71-7cf8-4a19-927e-d9cbdfdd373e after the operator's end-to-end live check. One root-owned page image replaces individual submitted sketch images. It uses the latest completed chosen sketch's exact recorded finish context; selected unsent ink only, index badges matching cards, full visible viewport, oldest compatible page/view up to8 and all other unsent shapes counted as N more. Pointer/private UI are absent. Notes and sketches remain two typed lists in the existing single submission/ACK transaction; one Pending feedback title and both counts. Note crops remain saved draw-time evidence. Annotation crop cards/private PNG transport are separately filed unclaimed #805. Local single-sketch export remains source-begin/cropped; legacy staged bundles remain immutable/readable. No new writer, ACK owner, finish-frame archive or browser dispatch route (ARCH-DRY/ARCH-PURPOSE/ARCH-EVOLVE).

This is the current executable delta; earlier per-sketch submission clauses are historical baseline and remain applicable only to legacy bundles and the single-sketch CLI export. The spec is approved; code admission still requires change-code. Re-derive the formal total estimate after revised plan-quality clears; the previous8.4h estimate describes the baseline. Full issue close must review this entire current contract, not a fix-only interdiff.

#### Core concepts for the revision (implemented symbols)

Paths below are relative to `cmd/internal/` unless prefixed with `tests/`. The table below names the implemented symbols; the final native/recorded/operator acceptance task remains pending.

| PURE entity/function | Lives in | Status and relationship |
|---|---|---|
| `PreviewItem` | submission/preview.go | unchanged comparable exact-delivery identity; never put view metadata in this map key |
| `PreviewSketchCandidate`, `PreviewNote`, `PreviewSelection` | submission/preview.go | new candidate wrapper/note witness, modified selection; owns copied source/finish facts, exact note revisions and optional selected page context |
| `SelectFeedbackPreview`, `sketchimage.SamePageView` | submission/preview_selection.go, webcast/sketchimage/page.go | new pure policy replacing the old sketch-only selector; one authority for preview and freeze batching |
| `PageScene`, `PagePlan`, `PageImage`, `PlanPage`, `ValidatePageBatch`, `ValidatePageImage` | webcast/sketchimage/page.go | new shared viewport render contract; one scene and image for1..8 exact sketch versions, ordered indices and original paths |
| `SketchPage`, `Sketch`, `ValidateSketchPage`, `ValidateSketch`, `EvidenceFiles` | submission/sketch_page.go, submission/sketch.go | new optional root image owner; modified sketch variant and central inventory; legacy Image or shared Index+Paths+Finish, never both |
| `DecodePreserving`, `EncodePreserving` schemas | submission/codec.go | modified strict conditional representation with unknown nested extensions retained |
| `NextPreview` | submission/preview.go | modified payload equality/cloning, same tagged states/events/effects and one transition owner |

Source and finish facts reuse `query.SketchSourceFacts` (scalar T/index/doc/tab/URL/title/VP/scroll). A shared page scene is a finish fact plus recording identity. Original source facts, anchors, points and pressure never change when projected into the finish scene. These are owned values: no DOM, live reducer, map or browser object is retained. Original declared IDs continue to be extracted at BEGIN. One root image has N sketch items; each item has one index and its original paths. Future label versions reuse the same identity axis and become sendable without a new ACK owner.

| INTEGRATION seam | Lives in | Status / existing resource reused |
|---|---|---|
| `Board.Put`, `Selection`, `FreezeSubmissionSelection` | webcast/board/board.go, submission.go, feedback_context.go | modified ordered reducer witness cache and atomic selection freeze; no IO under Board.mu |
| `PageRenderer.RenderPage`, `FakePageRenderer`, `RunPageRendererContract` | webcast/sketchimage/page.go, page_fake.go | new seam with stateful fake; owns active/canceled/completed calls and injected blocked/failing outcomes |
| `NewSketchPageImageRenderer`, `RenderPage` | webcast/sketch_page_image.go, web/sketch-page-image.js | new thin adapter using existing replay session, asset server/cache, pressure painter and owned Chrome/CDP cleanup |
| `PrepareSubmission`, `appendSubmissionSketches` | webcast/cast_submission.go | modified single recording read, independent witness verification, one shared image, existing crop and publication ownership |
| manager preview/freeze | couchtty/console_submission_preview.go | modified typed candidate conversion and acknowledged exact feedback freeze |
| private Chrome DTO/publication | webcast/chrome_submission_preview.go, webcast/operator-sketch.js | modified title/counts/indexed thumbnail cues, same private host lifetime and ACK fencing |
| Store read/prepare/confirm/transition | submission/store.go, publication.go | same folder-backed store and durability protocol; central file inventory includes one shared image |

The fake and real renderer implement the same `PageRenderer` seam and run `RunPageRendererContract`: a call starts once, active work is bounded to one, cancellation joins before replacement, failure returns no usable image, and wrong sketch/version/scene coverage refuses. Portable Store fixtures own all files. Native Chrome conformance exercises the identical page renderer contract plus pixel content, private asset confinement and child/profile/server cleanup (ARCH-MOCK).

#### Candidate capture, compatibility and atomic freeze

`Board.Put` is already the ordered recording Sink. Add an ordinal initialized to−1, incremented for every delivered record including head/segments/retries; capture immutable BEGIN and accepted completion scene witnesses while applying those records. Keep them in a map bounded by the reducer's256 retained sketches, deleting entries on clear/retirement and overwriting only the exact accepted version. Never read the journal in preview; never keep a screenshot per finish. Tests compare these ordinals with query's actual file indices, including equal-time records and duplicate events.

`SelectedSketch` becomes a struct containing its existing ID/NextSequence/Digest plus source/finish facts (rather than adding metadata to `PreviewItem`). Populate candidate descriptors in creation order from reducer candidates and the small witness map. Missing witnesses produce a visible unavailable/refusal result; do not guess the current page. `Selection` also includes all existing open notes as ID/revision witnesses. Advance the feedback generation on actual note creation/edit/state changes, labeled sketch eligibility/version changes and clear. Pointer/pressure events and unrelated DOM updates do not trigger new preview evidence work.

The pure selector filters only exact delivered sketch versions using unchanged `map[PreviewItem]struct{}`. The oldest remaining sketch chooses its completion-context compatibility group. `sketchimage.SamePageView` compares recording identity, doc, tab, URL, viewport W/H/DPR and recorded top scroll; title/time/index do not define view compatibility. Preserve creation order within that group, choose at most8, and count every other unsent sketch in Remaining. Select the greatest completion record index among the chosen group as the exact page scene. Original source doc/tab must be projectable into this scene; preparation refuses incompatible/missing source geometry rather than silently dropping that sketch or selecting another group.

Keep existing note inclusion semantics: all open notes across documents with their saved crops; eight is a sketch cap, not a new combined cap. Notes are not filtered with sketch delivery keys. Notes resolve through the existing exact-revision ACK path. This revision brings their count and exact preview witness into the existing transaction; #805 supplies crop cards and private crop bytes later. No additional note/card image route here.

`PreviewSelection` owns copied Notes, sketch candidate witnesses and optional PageScene in addition to unchanged Items/Stamp/Remaining. Equality checks all values; publication access clones slices. Notes-only selections have no PageScene and still require acknowledged preview. Empty feedback can use the existing ordinary-text path. `FreezeSubmissionPreview` carries the full proposed selection through manager→Chrome-host→Board lock order; Board checks generation, exact note ID/revision set, selected sketch versions and witnesses under one lock, then copies exactly those lists/context. It never replaces shown notes with newly open notes. Host retirement, edit/resolve/reopen, clear, delivery generation change and changed page witnesses refuse the stale freeze and trigger the existing refresh path. IO, Store scans and journal reads remain outside those locks.

#### Shared render and manifest shape

New submissions retain version3 and existing notes[] unchanged. They carry:

```text
sketch_page: {
  file: "sketch-page.png", provenance: "replay-rendered",
  source: {recording_id, t, record_index, doc, tab, url, title, vp, scroll},
  region: [scrollX, scrollY, viewportWidth, viewportHeight],
  scale, width, height, clipped, limitations,
  privacy, privacy_origin, privacy_rule
}
sketches[i]: {all existing identity/source/history/elements/privacy fields,
              index: i+1, paths: original visible paths, finish: exact completion source facts, image: ABSENT}
```

Legacy sketches instead require `image` with the unchanged cropped plan and have no index/paths/finish/shared page. Implement Image as optional pointer with explicit variant validation and migrate legacy constructors/accessors without changing legacy encoded bytes. A root page requires1..8 shared items with indices exactly1..N; shared items require a page; no legacy/shared mixtures, missing paths, empty ink, duplicated IDs/indices, unknown image filenames or multiply referenced evidence. New shared items retain history time/index and begin source, and require a Finish source fact whose time/index equals that exact selected completion. Finish facts establish each item's compatible view; the root scene must equal the latest Finish by ordinal, including title and URL. Source, Finish and root witnesses are verified against the recording during preparation; stored validation enforces these scalar relationships without claiming to reconstruct omitted DOM. Validate all common identity/author/label/privacy/elements/path bounds independently of raster format; recompute legacy crop transforms only for the legacy variant. Shared page dimensions/scale/region must equal `PlanPage`'s finite bounded viewport transform, and every source/version must match the frozen render result. Root privacy includes the actual base URL plus all chosen sketch decisions; submission aggregation additionally includes notes under the existing conservative rule.

The codec's explicit known-field schema becomes conditional, not permissive: `image` required only for legacy; `index`, `paths` and `finish` required for shared; all page fields listed above required except genuinely optional limitations/privacy_origin, with nested source/viewport/tuple presence and types strict. Explicit null is rejected for required page/scalar/tuple evidence; preserve established nullable containers and privacy_rule union. Unknown root/item/path/page/source members survive all read-modify-write paths. Deletion and null-injection tests enumerate every newly required field, including valid0/false/empty values; a legacy item forbids index/paths/finish even when zero/null. v1/v2 forbid shared evidence, note-only submissions omit the page, and optional omission tests defend intended compatibility. Old-decoder fixture from the pre-revision source refuses the new image-less items, so it cannot partially deliver a new bundle; old per-sketch fixtures remain readable and immutable in the new reader (ARCH-EVOLVE/ARCH-SECURE).

`EvidenceFiles` enumerates notes' existing crops plus exactly one sketch-page.png for shared batches, or each legacy image for legacy batches. All Store readMetadata/readSubmission/Prepare/ConfirmPrepared/Transition/immutable checks continue through that one inventory; no second file list. Validate PNG header/content, actual dimensions, file byte budget and safe path identity at the same boundaries as legacy images. Existing delivered-version scans ignore raster representation and derive the same unchanged exact keys. An already sent version remains suppressed after upgrading; a later exact version remains sendable.

`ValidatePageBatch` verifies source/history witnesses against the one read-complete recording in one ordered pass, including the latest chosen exact finish record. Reuse/refactor the existing ValidateBatch ordered witness verification rather than running one full pass per sketch. Replay precisely `records[:scene.RecordIndex+1]`, never seek by timestamp; same-time later mutations/assets cannot leak into the frame. Scene capture is finish-context DOM plus selected paths only. Keep CLI single-sketch ImageRenderer/export unchanged.

`RenderPage` owns one existing private replay session, one Chrome and one output capture. Reuse `Replayer`, renderer placement lookup, `projectSketchPoint` and `paintSketchPaths`; preflight every chosen path before painting because the old painter may skip absent targets. Reject missing/ambiguous/replaced/hidden anchors, nonfinite projections, source-document mismatch and centerline segments outside the fixed viewport/element clip. Do not clamp an entirely offscreen shape into view or omit a path. Ordinary pen-radius fringe at an edge is the declared natural viewport clip (set clipped/limitation explicitly); do not reject normal edge drawing solely for antialiasing fringe. Original paths remain complete in JSON. The image never includes live pointer, tool UI or previously sent/unselected ink: the private replay page creates only source DOM, chosen paths and indices.

Paint compact fixed-size number badges1..N near each shape in card order, clamped within the viewport. Use a bounded deterministic choice of nearby positions and explicit readable contrast; no layout/search subsystem. If a chosen path or its index cannot be represented unambiguously, refuse with the affected sketch/index and reason. Return exact ordered rendered version/path/index coverage and verify it before publication. Never convert partial rendering into a successful subset. The four-megapixel transform may downscale the entire viewport, preserving aspect ratio.

#### Visible state and temporal ownership

Actions: review Pending feedback, submit, retry preparation, status/reconcile the same submission after uncertain delivery, submit next batch. Existing pure `NextPreview` and submission transaction own transitions; this revision adds data to their witnesses, not parallel status flags.

| Internal state/event | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Dirty/Publishing after note edit or sketch finish | Pending feedback updating | Review refreshed counts/cards before submit | None |
| Shown exact selection, including notes-only | N sketches, M annotations; indexed sketch thumbnails; K more sketches | Submit existing chord | None |
| Incompatible unsent views/overflow | Current compatible group plus visible K more sketches | Submit current batch, then review next batch | None; other ink remains pending |
| Stale host/version/note revision before freeze | Feedback changed; review again | Review refreshed panel then submit | None; draft retained |
| Geometry/asset/budget refusal | Preparation failed with named reason/index | Retry existing preparation after correcting feedback | No send; no draft or ink deletion |
| Canceled/expired publication, even late successful callback | Existing Dirty/unavailable | Review/retry | No stale Shown; owned worker joined |
| Prepared then delivery ACK lost | Delivery unconfirmed | Existing same-ID status/reconcile | May already be delivered; no automatic resend |
| Dispatched/Resolved exact versions | Sent; ink stays; next pending group appears | Explicit next submit | No ink/label deletion |
| Old manifest loaded after upgrade | Existing legacy submission state | Existing status/reconcile | No image conversion or evidence rewrite |

Late/out-of-order/duplicate publish ACKs remain gated by host+stamp+operation; canceled work settles via NextPreview before identical-input wakes. Navigation during prepare cannot replace the frozen scene. Process death after immutable publication follows existing Store recovery. Unconfirmed delivery never suppresses new selection. No new writer, browser dispatch, note resolution authority or durable cache (ARCH-ORDER/ARCH-FUNERAL).

#### Operating envelope and measured proof

Interactive preview performs no recording IO: O(1) per ordered record to retain scalar context and O(C+M) per changed feedback set (C<=256 sketch candidates, M existing bounded open-note set). Host-only/delivery wakes reuse descriptors; no full historical reduction until prepare. Count R and2R metadata inspections, C and2C selector visits and zero journal reads across repeated host-only/unchanged wakes. Reuse the current delivered-generation cache; invalidation follows confirmed ACK, including idempotent durability confirmation.

Preparation reads at most128MiB once, validates/reduces the relevant prefix O(R), projects at most8*64=512 paths and8*16384=131072 samples. DOM/source-fact work retains existing20k-node/64KiB-fact bounds and existing original-context extraction grouping; avoid per-event candidate/DOM snapshot copies. Legacy crop and shared viewport plans use one `fitPagePixels` transform authority. One shared output <=4MP (about16MiB RGBA), <=8MiB PNG, whole bundle <=64MiB; Store remains<=1GiB with existing pruning/admission. Assets remain confined to the existing64MiB private cache. Selection witnesses are at most256 scalar source/finish pairs and die with Board/clear; no per-finish frame archive. Render temp/profile/server resources die with joined scope. Submission images die under existing Store cleanup. Browser network credentials/machine settings are unchanged.

Existing30s preparation,36s client observer and5s+1s joined cleanup budgets stay unchanged. Overload refuses visibly, no timeout bump. Compare one/two/eight-shape native preparation and measured maximum frame bytes; verify max8 under the same deadline on macOS. These are batch operations off the editor hot path; no foreground browser, production fleet harness or unbounded parallel renderer. Full suite remains machine-queued through SDLC.

#### Function strategies and executable tasks

Each task is a coherent TDD slice with a narrow commit; plain checkboxes, one final full close. Task dependency order is selection metadata → page renderer/schema contracts → orchestration/UI → integrated/native/operator acceptance. The schema and renderer contract slices may be delegated independently after the same gate passes, with disjoint file ownership.

| Production function | Executable adversarial/mechanical strategy |
|---|---|
| `Board.Put` / `Selection` | Colocated ordered-stream test compares cached source/finish with independent query at exact indices; duplicate/same-time/clear fixtures and R/2R inspection counters |
| `sketchimage.SamePageView` / `SelectFeedbackPreview` | Pure generated groups/versions with independent oldest-compatible oracle; exact suppression,8 cap,all deferred count,unchanged key identity |
| `NextPreview` / clone/equality | Existing model sequences extended with note revisions/context; mutate caller slices, delayed ACK, host replacement and cancellation must not revive stale Shown |
| `FreezeSubmissionSelection` | Actual manager/host/Board lock-seam controls: note create/edit/resolve/reopen and sketch mutation between publication and freeze refuse atomically, no journal IO |
| `PlanPage` / `ValidatePageBatch` | Fuzz finite/extreme source/viewport/paths, forged histories and exact same-time prefixes; one/two/N counted validation passes, originals unchanged |
| `ValidateSketch` / page validation / codec | Required-field deletion/null/type inventory, exclusive-variant matrix, unknown-extension roundtrip and legacy fixtures; no scalar-zero loophole |
| `EvidenceFiles` / Store consumers | Real portable folders with tampered/missing/extra/aliased files, wrong PNG dimensions, durable transition faults; one shared image inventory across each consumer |
| `RenderPage` | Same stateful fake/real contract plus Node geometry preflight tests: missing/replaced/hidden/offscreen paths refuse whole batch; all chosen ink+indices and no unselected/pointer/toolUI |

- [x] **Task5 — exact feedback witnesses and batch policy.** Add RED tests in board/submission_test.go, new board/feedback_context_test.go, submission/preview_selection_test.go and preview_test.go. Run focused Go tests and retain failure output. Implement cache/witness types, pure selector and exact note freeze; migrate fake casts/adapters together. Run `go test ./cmd/internal/webcast/board ./cmd/internal/submission ./cmd/internal/couchtty -run 'Submission|Preview|Feedback' -count=1` GREEN; commit with measured counters.
- [x] **Task6 — shared page contract and historical renderer.** Add RED `sketchimage/page_test.go`, contract tests and `tests/webcast/sketch-page-image.test.mjs`; implement pure PlanPage/witness verification and fake first, then the real replay adapter/painter preflight. Run `go test ./cmd/internal/webcast/sketchimage ./cmd/internal/webcast -run 'Sketch.*Page|Page.*Sketch|SketchImage' -count=1` and the actual Node test entrypoint GREEN. Add native exact-prefix/reflow/edge/badge/private-frame tests alongside sketch_image_test.go; keep existing fidelity/export tests.
- [x] **Task7 — preserving manifest and Store.** Add RED variant/field-deletion/legacy/unknown/PNG tests in submission/sketch_page_test.go, codec_strict_test.go and sketch_store_test.go. Implement optional legacy image, shared root/index/paths, common validation and central EvidenceFiles. Run `go test ./cmd/internal/submission -count=1` plus fuzz smoke GREEN. Prove the old reader refuses new evidence and the new reader preserves original legacy bytes/unknowns; commit.
- [x] **Task8 — wire one render into the existing submission and mixed-count panel.** Add RED PrepareSubmission/page fake and couchtty manager integration tests, note-only stale ACK controls and `tests/webcast/operator-submission-preview.test.mjs` count/index/keyed-focus assertions. Implement one recording read/shared render, fixed scene validation/privacy, both feedback counts, indexed per-sketch thumbnails and remaining-sketch cue. Preserve existing note crop assembly, target pinning, uncertainty/ACK lifecycle and Lua submission transaction. Run focused Go packages, Node tests and `bash tests/submission-transaction-nvim-test.sh` GREEN; commit.
- [ ] **Task9 — final contract and native/recorded/operator evidence.** Update ReadingGuide, README, embedded couch query/export help and atlas/webcast-submission.md around shared viewport/context/index semantics, original sources and Pending feedback; preserve single-sketch CLI/legacy docs. Reconcile this plan's actual symbols and #805's deferred scope. Stage new source files before reader inventory; audit affected reader inputs, classify new production files in artifactpath inventory, run lint/artifactpath/`make test-affected-inventory`, targeted race and diff checks. Record native one/two/eight batch pressure/index/source/privacy/cancel+join results on committed source with raw log/hash and unchanged deadline. Push, run final recorded suite (resume only registered-flake-only failures via supported command), full SDLC close, and refresh PR description. Ask hub for coordinated restage only after automated readiness. Operator sees two indexed thumbnails+both counts, submits one shared viewport PNG containing exactly those shapes and notes/URL/elements, second submit does not resend, ink stays, incompatible leftovers remain N more. Verify both Claude/Codex parsing via disposable fixture consumers; real operator smoke is final acceptance before merge.


2026-10-22 — executable delta reconciliation: `sketchimage.SamePageView` is the one shared compatibility function; no second submission wrapper was added. Cast's injected boundary is `CastConfig.SketchPages` / `NewSketchPageImageRenderer`. `PlanPage(scene)` returns viewport-only `PagePlan`, `PageBatch` owns original histories plus exact Finish facts, and `PageImage` adds measured Clipped/ordered coverage alongside PNG/limitations. Core `SketchPage` owns the flat stored page fields; each shared sketch owns Paths+Finish. Existing `Plan` validates/extracts original paths in both formats, but its individual crop is compared/stored only for legacy; no duplicate vector validator. Cast/Chrome freeze now accepts the full `PreviewSelection`, rederives the page, then converts exact candidates/notes to Board's three-argument freeze. Synchronous explicit test-fixture ACKs preserve transaction scheduler/capture tests without weakening the production note-only gate. Initial source/refusal/native/Node tests passed; final committed-source conformance, recorded suite, full close and renewed operator stage remain Task9.

2026-10-21 — BR-7 inventory completion: sketch privacy_rule remains its existing nullable/string/object union; when represented as a rule object, both origin and privacy are required. This preserves explicit null and diagnostic-string forms without treating a partial rule object as evidence. The complete fixture must exercise all declared-fact groups for sketches and present note.elements, so a skipped absent optional object cannot mask a required-field family.

2026-10-21 — BR-7 required-member correction (ARCH-SECURE/ARCH-EVOLVE/ARCH-PURPOSE): round2 deletion probes established that scalar type checks do not establish presence. Extend the existing non-reflective codec schema with explicit per-object required-member metadata, checked before filtering/typed decoding. Sketch requires id, recording_id, next_sequence, digest, author, note, history_t, history_record_index, phase, source, elements, image, privacy and privacy_rule; reason and privacy_origin remain optional. Source requires t, record_index, doc, tab, url, title, vp, scroll and provenance, including explicit valid zero/empty values; viewport requires w/h/dpr. DeclaredElementFacts requires anchors/ancestors/intersections/truncated/missing_layout/inspected_nodes/bytes (arrays remain nullable); each ElementFact requires node, with parent/tag/id/data/unavailable/truncated optional. Image requires file/provenance/plan, limitations optional. RenderPlan requires sketch/region/scale/width/height/clipped/paths; each Path requires id/at/width/pts/truncated; sketch-path anchors require node/box while descriptive fields remain optional. Sketch authors require id/color, name optional. Ordinary note.elements remains optional/null, but a present fact object has the same required contract; legacy root/note/anchor defaults and unknown extensions remain unchanged. Add deletion regressions across the complete inventory, explicit valid zero/false/empty evidence, positive optional-omission compatibility, and real Store read-path tampering. This closes a wire-validation gap without a new delivery route, state or product scope.

2026-10-21 19:04 PDT — TL745ea763 approves one finished multi-stroke sketch as one item, oldest eight unsent first with visible N more, and exact-version dispatched/resolved suppression across targets. Adds acceptance that the operator sees count and shapes before the chord submits. Added a read-only immutable private preview sharing the candidate selector and existing projection; no new browser delivery route. Proceed to full-flow change-code plan-quality review before implementation.


2026-10-21 19:08 PDT — PQ-1: specified the authoritative pure preview transition model, opaque frozen-preview constructor, manager→board atomic generation check and tracked/coalesced worker lifetime. PQ-2: replaced prose test inventories with named production functions and one adversarial/mechanical strategy per function; retained integration/native/quantitative checks. No product scope change.

2026-10-21 19:51 PDT — TL a7274b6d-b64e-40f0-b1db-969c0cc4f4ba approves a discovered prerequisite inside #753: repair CDP write admission/cancellation so bounded preview/render work can actually join. Replace non-contextual write-mutex admission with cancelable serialization; while the writer is owned, arm its pipe deadline and join cancellation before resetting/releasing it. Retire the connection on a partial frame. Generic owned writers without deadlines must close/unblock on active-write cancellation and report connection retirement. Add controlled active/queued cancellation, partial-write and stale-callback/deadline-reset regressions plus native renderer conformance. Existing timeout values and agent routing/authority stay unchanged. Pushes are held by TL until github.com DNS returns.

2026-10-21 — BR-5 core-concept traceability correction (ARCH-PURE): reconciled the proposed inventory with implemented symbols/paths. PreviewSelection moved in the inventory from nonexistent submission/sketch.go placement to preview.go; replaced the proposed PURE DeliveredSketchSet with pure SelectSketchPreview plus INTEGRATION Store.DeliveredSketchVersions. Replaced proposed query/render_sketch.go and webcast/submission/sketch.go with actual sketch_export.go, sketch_image_export.go and existing submission/build.go. Named RenderPlan/Path and the effect/validation split in render.go, the actual preview host bridge, and the admitted CDP integration. This records an implementation-map correction; product policy and the approved delivery mechanism are unchanged.

2026-10-21 — round1 REWORK correction work: BR-1 requires every canceled publication to settle through NextPreview before identical-input refresh; BR-2 budgets selected declarations in mandatory-anchor/ancestor/nearest-context order; BR-3 preserves valid region-only notes without synthetic node-zero facts; BR-4 strictly checks known scalar/tuple member types without coercing null to zero. Each family receives a controlled RED→GREEN regression. BR-6 requires final-source native conformance evidence from a supported Chrome environment; the review checkout reported GPU/EPERM failures, while the main worker environment has passing native results. Operator acceptance remains a separate landing gate.

2026-10-21 — BR-2 source/output budget clarification: source snapshots retain owned maps of immutable strings, bounded by 20,000 node headers and 16 bounded data declarations plus one ID per node. The 64KiB facts budget is spent only on selected output, in mandatory-anchor→ancestor→nearest-context order, reserving space for remaining mandatory handles. Unselected attributes cannot starve the chosen anchor. BR-1 class sweep also invalidates the derived delivered-version cache on every confirmed ACK, including confirmation of a disposition already visible after a previously uncertain fsync; the durable Store remains the sole acknowledgment authority.

### 2026-10-22 — BR-8/9 close correction: original identity and consumer guidance

Shared historical projection requires the connected original recorded node. The existing createPlacement adapter accepts originalOnly:true for this caller; prepareHistoricalPageInk wires it into every chosen path before painting. Default live annotation matching retains its existing replacement behavior. Missing/disconnected, hidden, conflicting and wrong-document/tab originals refuse the whole image, with connected-original controls and an actual Renderer adapter regression plus native replacement-record replay. This fulfills the already-approved unprojectable-ink refusal, without changing batching, permissions, replay versions or timeout budgets (ARCH-PURPOSE/ARCH-SECURE). Agent handoff guidance names the shared sketch_page/sketch-page.png/index/paths form and the immutable legacy image.plan.paths form, full-page privacy and oldest compatible batching; the Lua emitted prompt is verified through the real Neovim transaction tests.

### 2026-10-22 — approved painting-only submitted-feedback mask

Authority: TLc78d3900 approves this revision after the operator accepted the indexed full-page image. Supersedes every earlier live ink-stays clause; source/recording/replay semantics remain unchanged. Existing conditional annotation resolution stays, while the live mask also hides confirmed exact notes when resolution waits. #808 toolbar work is excluded. No new recording event or persistent acknowledgment owner (ARCH-DRY/ARCH-PURPOSE/ARCH-EVOLVE).

#### Core concepts and seams for this revision

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| DeliveredFeedback (exact sketch keys and note revisions) | cmd/internal/submission/store.go or dedicated delivered_feedback.go | new | PURE value, derived only from existing manifests |
| LiveFeedbackMask (scope, revision, exact current keys) | cmd/internal/webcast/board/feedback_display.go | new | PURE wire value; no vectors |
| createFeedbackMask / exact sketchVisible and noteVisible | cmd/internal/webcast/web/feedback-mask.js | new | PURE bounded parse/index/match |
| SketchMetadata digest / display version metadata | recording/sketch_state.go, web/sketch-model.js, board/sketch_delivery.go, web/sketch-render.js | modified | PURE metadata, no event/snapshot schema change |
| Store.DeliveredFeedbackVersions | cmd/internal/submission/delivered_feedback.go | new | INTEGRATION Store scan/confirmation; existing fake fsync seam |
| Board.SetDeliveredFeedback, FeedbackMask, SubscribeFeedbackMask | cmd/internal/webcast/board/feedback_display.go | new | INTEGRATION existing Board lock/subscriptions; derived bounded cache |
| Cast.SetDeliveredFeedback / manager refresh | cast_submission_candidates.go and couchtty/console_submission_preview.go | modified | INTEGRATION existing preview worker/Store/Board; stateful fake cast |
| OperatorView + viewer SSE mask delivery | board/operator_view.go and server.go | modified | INTEGRATION existing delivery workers/stream; no new goroutine |
| Live painters and note projections | operator.js, operator-sketch.js, web/viewer.js, web/sketch.js, web/annotate.js | modified | INTEGRATION shared pure mask predicate; existing browser fixture/native harness |

Store.DeliveredSketchVersions remains compatible, deriving through one scan implementation. DeliveredFeedbackVersions confirms only dispatched/resolved bundles; includes annotation-only bundles and exact note revisions. No duplicate directory scan per refresh. Board receives converted exact keys through Cast, matches current candidates/notes under its existing lock, and keeps only current identities (at most existing sketch/note authority bounds). Later sequence/revision changes cannot equal an older key. No recorded mutation or event is emitted by setting a mask. Board selection excludes exact masked notes, so an annotation-off resolution wait cannot reoffer already-delivered feedback; sketch suppression remains the established selector. Empty/no-op updates do not advance revisions or notify subscribers. Board teardown closes subscriptions; mask dies with the cast and is rederived on attach/reconnect from the Store, without a sidecar.

The existing preview worker loads confirmed deliveries independently of private-host availability, then applies under manager→Board lock order only while that run/delivery generation remains current. A lookup/confirmation error retains the prior confirmed mask, invalidates preparation and shows existing unavailable feedback; no uncertain delivery is newly hidden. A successful ACK invalidates the cache through existing submissionDeliveredLocked; duplicate ACK is safe. Cast replacement joins the old worker before installing a new one. Re-read selection after applying the mask before building the acknowledged preview. No detached task or timer changes (ARCH-ORDER/ARCH-FUNERAL).

Console receives a full mask in its existing acknowledged OperatorView; viewer gets a separate live-stream feedback event with an initial/current full mask and coalesced updates. This is transport-only display metadata, never a recording record. Stream identity plus mask scope/revision fence stale callbacks; reconnect/catchup retains the current mask for that stream, and a replacement stream replaces its scope. Historical replay has no feedback publisher/hook. The pure JS mask matches exact sequence+digest and revision, not only IDs; metadata adds constant-size last accepted digest/next sequence (computed from the last ordered receipt in Go, maintained in the JS reducer on commit/restore). Console private sketch bootstrap and incremental headers carry that same version. Missing version metadata never hides a path. Shared paint filtering leaves display paths/point offsets and all reducer data intact; it triggers full canvas invalidation on mask change, and note pins/list/dialog redraw. Preview thumbnails retain their own unfiltered source and exact selector.

Actions: submit, confirm existing receipt, draw/edit new feedback, reconnect.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| prepared/delivering/unknown ACK | feedback and ink remain | confirm existing receipt | none; never substitute a new send |
| confirmed dispatched/resolved exact version | exact submitted ink/pins absent live; panel advances | draw new feedback | only live display visibility; history retained |
| confirmed old version, newer sequence/revision | newer feedback visible and sendable | submit new version | none |
| partial/incompatible batch | only confirmed selected batch absent | submit remaining feedback | none |
| display delivery lost/reconnect | current full mask reinstalled; vectors retained | reconnect | none; no recorded clear |
| Store confirmation/load fails | keep prior confirmed display, preparation unavailable | confirm existing receipt after recovery | no new hiding |
| replay/history/export | historical ink remains | normal replay/query | none |

Hot path: one existing bounded Store scan per delivered generation, not per event/paint. Mask installation indexes D delivered keys and visits C current sketch/note headers once O(D+C), no points/receipts replay; counted N/2N tests enforce visit counts. Wire size is bounded by current sketches and existing notes capacity. JS mask parse/index O(M) per update, O(1) version lookup per sketch path/note during existing painting; sketch digest metadata O(1) per event/restore receipt. No extra per-point work or background worker (ARCH-CONSTRAINTS/ARCH-PURE).

#### Task10 — execute the approved mask revision (single full-close boundary)

- [x] TDD Store derivation: only confirmed delivery, notes-only/mixed, uncertain fsync/duplicate ACK, cross-target exact versions; preserve existing DeliveredSketchVersions API.
- [x] TDD Board derived mask: current-version matching, deep ownership, no-op notifications, stale versions/newer changes, selective batches, selection excludes masked notes, no record writes; scope/revision/subscriber teardown; counted N/2N header visits.
- [x] TDD pure JS exact-mask parser and shared metadata: wrong digest/sequence/revision never hides, malformed/stale mask atomic refusal, optional legacy display metadata safe; bootstrap and incremental equivalence without point rescans.
- [x] Wire worker/manager/Cast and both live transports; test ACK/unknown/lost-host/reconnect/cast replacement through stateful fakes, initial full snapshot, stale stream rejection and recorded-history preservation.
- [x] Wire live painters and annotation projections: mask change removes old canvas pixels, selected only, unsent/newer visible, replay unaffected; real native console/viewer/reconnect conformance.
- [ ] Update actual atlas/README/agent handoff guidance to live-hide/history-retain behavior, preserve old revisions as history; inventory, affected targeted/race/browser/Neovim checks, final recorded suite and full close on revised contract.
- [ ] Restage only on hub go, quick operator re-look after confirmed send (selected gone, remaining visible, historical ink retained), then supported merge after recorded pass and acceptance.

### 2026-10-22 — mask transport constraints and reconnect ordering

Implementation audit clarified that historical open notes are not globally capped by the48-per-page admission rule. The mask has no artificial note-count cap: reuse the64MiB full wire/bundle budget, validate a proposed full Board mask before installation with worst-case uint64 revision space, and atomically refuse oversize without notifications or truncation. JS enforces the same UTF8 byte budget before index allocation, retaining the last valid mask. A49-historical-note regression and exact-byte boundary controls cover this correction. An over-budget projection follows the existing preparation-unavailable recovery, keeping prior confirmed display state; no new recorded format or persistent authority. The server publishes its subscription's initial full live mask before any catch-up record, so a reconnect cannot briefly repaint submitted ink. Console note projections are filtered under Board's actual revision authority; its legacy Note objects carry no revision, so the browser never guesses revision0 to hide a newer console note. UI metadata digest comes from the owned prepared receipt, not caller-mutated input (ARCH-SECURE/ARCH-ORDER/ARCH-CONSTRAINTS).

### 2026-10-22 — BR-10 confirmed-display-continuity class correction

A transport outage never disproves an already-confirmed delivery. The live display lifetime owns the last confirmed mask; rotating/retrying a transport preserves it, and only a validated full replacement or actual display disposal may replace/forget it. Same-scope revision fencing survives reconnect; the new stream token admits its first validated full scope atomically and then rejects another scope on that stream. Old producer callbacks stay fenced. Manual resync also preserves the mask while replacing model/renderer. Repeated transient and terminal errors must not redraw submitted ink/pins (ARCH-ORDER/ARCH-PURPOSE).

Class enumeration: automatic EventSource error; manual record resync/new stream; stale or malformed replacement; omitted/stale console mask; inactive console target producer; historical catch-up prefixes; Store/byte-budget failures; actual display disposal. The latter two error paths already retain Board knowledge. Console now validates a full nonstale mask before advancing outer view revision/applying server-filtered notes; missing/zero/stale masks refuse the entire snapshot. The inactive-target producer carries the SAME authoritative view's valid full mask with empty inactive page/notes/strokes. This preserves coherence without inventing browser note revisions.

A caught-up live model still suppresses only the confirmed exact sketch sequence/digest or note revision, with newer versions visible. During reconnect, a historical prefix BELOW a known confirmed version cannot yet be current live state: hold its painting until it reaches that proof. Equal sketch sequence with a different digest remains a different version, visible. Retain one latest Store-confirmed watermark not above the authoritative current version per retained identity, including resolved historical notes; editing/resolving/reopening does not erase that confirmed knowledge. Selection and operator note projections continue to compare the actual current revision exactly, so newer feedback stays sendable/visible. Actual annotation-clear removes the identity and releases its mask proof. This supersedes Task10's earlier exact-current-only cache/prune detail, avoiding an older submitted note flashing before its later edit/state catch-up. No new delivery claim, recording event or registry; every watermark still derives from existing confirmed manifests. Cache remains bounded by retained reducer identities and64MiB wire envelope, and dies with the cast (ARCH-DRY/ARCH-FUNERAL/ARCH-CONSTRAINTS).

Regressions: production connect keeps note/sketch visibility and redraw count unchanged throughout repeated readyState0/2 errors; stale same-scope replacement, invalid/full new scope, retired callbacks; actual annotation pins/list/dialog stay absent across note0→edit1→confirmed2 prefixes but show newer3; whole console snapshot rejects missing/invalid/stale mask without consuming its outer revision; inactive producer preserves mask; Board edits/resolves/reopens retain latest confirmed floor and newer current notes remain visible. Native real EventSource error dispatch exercises the actual painter without a new timeout or harness. All are in the normal bounded suites except the existing opt-in native run.
