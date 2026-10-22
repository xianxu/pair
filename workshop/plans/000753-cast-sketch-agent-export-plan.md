---
collaboration-mode: discussion
---

# #753 — Historical sketch evidence in operator submissions

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
| SketchQuery / SketchView | cmd/internal/webcast/query/sketch.go | new | One recording pass freezes source identity/context and selected later sketch history; no live DOM access. |
| DeclaredElementFacts | cmd/internal/webcast/query/element_facts.go | new | Bounded id/data-* facts for original anchors, ancestor chain and nearest intersected source elements; shared by sketches and ordinary annotations. |
| SketchRenderPlan / ImageTransform | cmd/internal/webcast/sketchimage/contract.go | new | Validated source viewport/crop/scale, visible pressure paths, PNG limits and qualified provenance; no IO. |
| Submission.Sketch | cmd/internal/submission/sketch.go | new | Typed immutable image/note/page/anchor/vector/history evidence; independent of existing CDP Note/Crop meanings. |
| PreviewSelection | cmd/internal/submission/sketch.go, couchtty/console_submission.go | new | One immutable candidate selection with current-console publication acknowledgment; freezes exactly the displayed batch. |
| Selection / SelectedSketch | cmd/internal/webcast/board/submission.go | modified | Frozen recording+sketch+accepted-prefix identity, never a moving latest-state lookup. |
| DeliveredSketchSet | cmd/internal/submission/sketch.go | new | Derive delivered selection keys from confirmed dispatched/resolved manifests; no second durable ACK source. |
| Submission validation / file inventory | cmd/internal/submission/model.go and codec.go (new) | modified | Strict known fields, preserved unknown fields, one enumeration of all evidence files and bounds. |

One SketchView owns many strokes and anchors, one source moment, one selected history cutoff and one label. Source facts are copied at begin, not reconstructed from a later page. No duplicate label state is added; selected version uses the existing accepted-prefix sequence/digest. #783 revisions will naturally change that prefix and become a new eligible version without treating an old ACK as newer delivery.

| INTEGRATION | Lives in | Status | Wraps / injected boundary |
|---|---|---|---|
| ImageRenderer | cmd/internal/webcast/sketchimage/contract.go, fake.go | new | Render(ctx, batch) returns bounded PNG/results; stateful fake tracks opens, requests, assets, cancellation and close. |
| Replay image session | cmd/internal/webcast/replay_image.go | new/extracted | Owned Chrome/CDP/profile/replay server lifetime extracted from fidelity_check.go; adapter injected behind ImageRenderer. |
| Sketch image adapter | cmd/internal/webcast/sketch_image.go, web/sketch-image.js | new | Recorded-only sandbox render plus shared paintSketchPaths; no current tab screenshot. |
| Query CLI | query/{surface.go,fake.go,contract.go,cli.go,render_sketch.go}, couchcmd query wiring | modified | Add sketches and sketch-image; preserve existing query entrypoints via a forwarding options seam. |
| Cast evidence builder | webcast/cast_submission.go, webcast/submission/{build.go,sketch.go} | modified/new | One bounded recording read, policy decisions, render batch, frozen immutable bundle. |
| Submission manager/store | couchtty/console_submission.go, submission/{store.go,publication.go} | modified | Sketch-only selection, preparation, PNG inventory, exact publication reconciliation and existing delivery transaction. |
| Private submission preview | webcast/operator-sketch.js, chrome_operator_sketch_wiring.go, couchtty/console_submission.go | modified | Read-only batch cards/count using existing pressure display and one coalesced owned publication worker. |
| Operator progress and guide | nvim/dom_submission_state.lua, couchcmd/skills/couch/SKILL.md, submission ReadingGuide | modified | Batch counts/provenance, manifest handoff and existing retry/target semantics. |

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

2026-10-21 19:04 PDT — TL745ea763 approves one finished multi-stroke sketch as one item, oldest eight unsent first with visible N more, and exact-version dispatched/resolved suppression across targets. Adds acceptance that the operator sees count and shapes before the chord submits. Added a read-only immutable private preview sharing the candidate selector and existing projection; no new browser delivery route. Proceed to full-flow change-code plan-quality review before implementation.


2026-10-21 19:08 PDT — PQ-1: specified the authoritative pure preview transition model, opaque frozen-preview constructor, manager→board atomic generation check and tracked/coalesced worker lifetime. PQ-2: replaced prose test inventories with named production functions and one adversarial/mechanical strategy per function; retained integration/native/quantitative checks. No product scope change.

2026-10-21 19:51 PDT — TL a7274b6d-b64e-40f0-b1db-969c0cc4f4ba approves a discovered prerequisite inside #753: repair CDP write admission/cancellation so bounded preview/render work can actually join. Replace non-contextual write-mutex admission with cancelable serialization; while the writer is owned, arm its pipe deadline and join cancellation before resetting/releasing it. Retire the connection on a partial frame. Generic owned writers without deadlines must close/unblock on active-write cancellation and report connection retirement. Add controlled active/queued cancellation, partial-write and stale-callback/deadline-reset regressions plus native renderer conformance. Existing timeout values and agent routing/authority stay unchanged. Pushes are held by TL until github.com DNS returns.
