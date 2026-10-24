# Cast feedback workspace implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent work, or superpowers-executing-plans for work requiring session context. Steps use checkbox syntax for tracking.

**Goal:** Give the operator one Notes panel and one unobstructed tool column (#856), then make feedback belong to the current browser page load (#857).

**Architecture:** Reuse #850's floating-panel owner and existing note/sketch commands. A unified keyed projection presents existing feedback without becoming another feedback store or submission authority. The recorder supplies a persisted page-load identity; Board derives feedback membership from creation boundaries and filters the atomic press-time selection introduced by #849.

**Tech Stack:** Go, existing CDP recorder and Board, native HTML/CSS/JavaScript, Node tests, existing headless Chrome fixtures, and `couch mock serve` for the operator's layout review.

## Scope and ordering

This is one coordinated design with two independently landed issues. #856 starts from main and waits for #850 and #849 to land before implementation that consumes their APIs. #857 starts on its own branch from landed main after #856; never stack unlanded issue commits. Its implementation section below is copied/referenced by the eventual #857 plan checkpoint and reviewed at its own change-code gate. Each issue has one close boundary, not artificial per-task milestones.

First deliver a clickable, synthetic layout mock for operator approval. No production UI implementation precedes that approval. The morning live build may be a staging-only local merge of approved branches, coordinated through the hub and the stage owner. Production landings remain separate.

Finished sketch labels remain read-only. Add note opens the existing unfinished-label flow. Show on page scrolls to and highlights a positively identified current-load anchor; when unavailable it says so, without guessing, navigation or reload. #855's submission instruction/skill rewrite is separate.

## Core concepts

Paths below are relative to `cmd/internal/webcast/` unless stated otherwise. New helper names are intended interfaces; reconcile this table with the actual implementation before each close.

### Pure entities

| Name | Lives in | Status / issue |
|------|----------|----------------|
| Feedback entries projection: keyed note/sketch rows from existing models | `web/feedback-entries.js` | new / #856 |
| Workspace layout: protected toolbar rectangle and panel bounds | `web/feedback-layout.js` | new / #856 |
| Floating panel transitions and session-local anchor | `web/floating-panel.js` from #850 | modified with optional geometry / #856 |
| Sketch painted-bounds grouping | `web/sketch-notes.js` from #850 | modified: pure helper retained, DOM selector removed / #856 |
| PageLoad: exact target, loader and backend root identity | `recording/page_load.go` | new / #857 |
| Segment load, current load, creation-load projection | `recording/record.go`, `recording/reduce.go`, `recording/sketch_state.go` | modified / #857 |
| Selection scope and press-time membership | `board/submission.go` after #849 | modified / #857 |

One recording has many page loads; a load has one target and potentially many activation segments. Each feedback item belongs to the load at its accepted creation boundary. Segment/document counters remain activation and draft fences; they are not repurposed as load identity. A→B→A can have a new activation segment and the same load.

The entries projection owns no saved note text, sketch history, delivery ledger or permission. It derives rows from the existing note/sketch owners and commands carry their exact IDs/revisions. The layout calculation owns only rectangles. Rows use small static type icons, not thumbnails. Reuse existing anchored-box/sketch overlay painting for hover glow; no screenshot or row-image cache is created. Natural future extensions are finished-label editing through a separately approved command and richer row presentation, not a second feedback store.

### Integration points

| Name | Lives in | Status / issue | Wraps |
|------|----------|----------------|-------|
| Unified Notes DOM adapter | `web/feedback-panel.js`, `web/feedback-panel.css`, `web/annotate.js` | new/modified / #856 | keyed rows, editor, scrolling panel |
| Tool-column adapter | `web/cast-tools.js`, `web/cast-tools.css`, `inject.js`, `operator.js`, `web/sketch.js`, `web/sketch.css` | new/modified / #856 | existing live/pause, Pointer, Pen, Note, palette and commands |
| Informational status adapter | `operator-sketch.js`, `web/feedback-panel.js` | modified / #856 | existing preview/status publication |
| Anchor action | `web/annotation-placement.js`, `web/render.js`, `web/viewer.js`, `operator.js` | modified / #856 | confirmed original anchor before scrolling and rechecking placement |
| Hover glow canvas owner | `web/feedback-glow.js`, `web/annotate.js`, `web/sketch.js` | new/modified / #856 | pixel-only painting with existing box/ink placement |
| Embedded asset and operator module graph | `server.go`, `operator_script.go`, `web/index.html` | modified / #856 | shared runtime JS/CSS and import/export seams |
| Recorder navigation/load owner | `recorder/recorder.go`, `recorder/mirror.go` | modified / #857 | CDP top-frame events and fenced snapshot |
| Operator projection | `board/operator_view.go`, `board/sketch_delivery.go`, `web/model.js`, `web/annotate.js`, `web/sketch-render.js`, `operator.js` | modified / #857 | current-load rows, pins, ink and controls |
| Recording codec and catch-up | `recording/contract.go`, `recording/catchup.go`, `recording/sketch_snapshot.go` | modified / #857 | file replay and restart |
| Historical query facts | `query/querier.go`, `query/sketch.go`, `board/feedback_context.go` | modified / #857 | all-history agent queries and source provenance |

CDP tests reuse the recorder's stateful fake Chrome (targets, frame/loader/root changes, delayed snapshot replies, attach/detach) through the production seam. Native Chrome conformance compares that model with same-URL navigation and tab switching. Owned recording storage is a disposable folder in tests. UI tests reuse the existing DOM/scene fixture and real headless Chrome; no live operator browser is used for automated tests.

## Operator mock and visible behavior

Mock assets live in `workshop/plans/000856-cast-feedback-workspace-plan.side/mocks/`: one `fixtures.json`, `mock-1/index.html`, `mock-1/app.js`, and `mock-1/tokens.css`. Runtime selection/lock/staging files are ignored using the existing mock template conventions. Use synthetic pages and feedback, not the operator's private recording. Each feedback round is a commit to mock-1; create mock-2 only for a different direction worth comparing.

Initial proposed layout: a protected right-hand tool column, with live/pause first and Sketch second. Sketch on reveals palette, available Undo/Redo, then Done in that same column. The yellow 300×320 Notes panel starts beside the column, with a fixed title/status header and a scrolling mixed list. Its minimized Notes (N) button and panel share #850's session-local anchor. All drag, resize and popup clamping excludes the toolbar rectangle; on narrow viewports the panel shrinks/scrolls rather than covering tools. At widths too small for both full surfaces, retain the toolbar and a minimized Notes control; expanding uses the remaining bounded area. Exact placement and minimum useful size are operator mock decisions, not assumptions to hard-code before review.

The panel is the sole reading/opening surface: ordinary notes retain their existing edit/save/cancel commands, sketch rows show a small sketch icon and label, unfinished sketches expose Add note, and completed labels expose View note read-only. Opening a row does not invent a new editor or bypass its unsaved state. Status messages are in the panel header; while minimized, the Notes control conveys status and reopening reveals the text. No separate Pending feedback box or Sketch notes selector remains. Anchored pins and ink remain on the page as feedback, not per-sketch read buttons. Hovering a row glows its sketch ink or annotation’s anchored box on the page; keyboard focus offers the same association. Hover never scrolls, mutates selection or advances stamps. Unknown/off-view anchors produce no guessed glow. Clear the transient highlight on leave, focus exit, row removal, minimize, load change and teardown; late placement replies are fenced by the same current-load/revision identity.

The mock supports mixed/empty/long lists, minimized/expanded panel, sketch tool reveal, paused/driven state, unsaved editor, read-only finished label, unavailable anchor and current-load change. Show the page-load rule in the flow even though persistence lands in #857. Serve with `couch mock serve workshop/plans/000856-cast-feedback-workspace-plan.side/mocks --port 0`; send the returned URL and a short review sequence to the hub. Hold production code for the operator's layout answer.

## State and recovery

Actions: open/minimize Notes, drag or keyboard-move, toggle Sketch, draw/undo/redo/Done, add/edit/save/cancel a note, view a finished label, show on page, pause/resume, switch tab, reload, retry capture, submit saved feedback.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|-------------------------------|---------------|-----------------|---------------|
| Confirmed current load, feedback available | Current-load Notes list and existing anchored feedback | Open/view/edit | None |
| Panel minimized or viewport resized | Notes button clamped away from tools | Open Notes | Session-local position may clamp; saved feedback intact |
| Sketch off / on, undo or redo unavailable | One column; only actionable history controls shown | Toggle Sketch or Done | Existing unfinished-label rules unchanged |
| Unsaved ordinary note or unfinished mandatory label | Editor in bounded panel, explicit unsaved status | Save or cancel | Cancel discards only that draft; submit preserves #849's unsaved-note refusal |
| Finished sketch label | Read-only text and sketch icon | Show on page or close | None |
| Anchor absent, ambiguous, detached or outside confirmed load | Plain “Cannot show on this page” status | Close row | No mutation, navigation or inferred location |
| Top navigation or target change awaiting an authoritative snapshot | “Loading page” status; no old-load feedback or creation controls | Wait, then retry capture if it fails | Existing saved history retained; draft follows existing stale-document cancellation policy |
| Snapshot fails, detaches, or returns after a newer generation | Unavailable current-page state; stale completion ignored | Retry capture | No saved history lost; no guessed identity |
| New load confirmed after reload | Empty list for that load | Add a note | Earlier feedback leaves UI/submit but remains in recording |
| Same load returns after pause or tab switch | Its saved rows and ink return | Open Notes | None |
| Submit admitted, then navigation occurs | Existing sending/delivery state; admitted exact selection remains pinned | Existing delivery recovery | No second selection or silent change to admitted feedback |
| Recorder/viewer restart on older recording without load metadata | History readable; unidentified feedback excluded from identified live load | Resume capture to establish current identity | Earlier legacy items remain queryable, not inferred into a new load |

The load fence is an owner-held generation, not a probe that attempts to prove no navigation occurred. The recorder publishes pending scope before snapshot IO and only the matching generation can confirm it. A top-frame navigation cancels/supersedes outstanding work; iframe events do not replace the top load. No polling or deadline changes. Transport generation, DocSeg draft admission, UI-suppression leases and Board locking remain their existing owners.

Unknown load does not introduce a new global submit refusal: select no page feedback until the current scope is confirmed, while preserving #849's ordinary text submission and unsaved-note guard. This conservative policy is explicit for review. It never silently includes old feedback to keep a count nonzero.

## #857 persisted identity and compatibility

Persist the authoritative optional structured `page_load` on the accepted segment boundary, with exact target ID, loader ID and backend root. Derive each note/sketch's creation load from its existing accepted document boundary and expose it in reducer/query projections. This records membership once without changing sketch request canonicalization or receipt digests. Do not append a digest-bearing field to SketchBegin: older readers preserve unknown sketch members but exclude them from their known canonical request, which would cause receipt mismatches.

Validate the new object at the recording boundary: owned names/types, required nonempty identity members, positive root and duplicate-key rejection; retain unknown members in raw extension storage, including nested members. Update the changed segment codec and every catch-up/snapshot rewrite to preserve those extensions. Existing plain encoding/json decoding is not evidence of preservation. An old schema-reader regression must decode and rewrite a newer segment through the additive extension seam without dropping unknown fields. Already shipped historical readers cannot be retroactively taught preservation; document that limit rather than claiming it. The recording remains append-only; existing historic lines are not migrated or rewritten.

Legacy segments without metadata remain readable/queryable. Missing or malformed identity never becomes a URL-derived, capture_id-derived or activation-counter-derived load. Malformed owned data refuses as the recording decoder already does; missing legacy metadata is explicitly unknown. Catch-up must carry each creation segment's identity before its dependent feedback, including inactive loads, so restart and full replay agree. Historical query is unfiltered; UI and press selection are filtered.

Top-frame Page.frameNavigated currently changes loader without forcing dirty/resnapshot. Make that event directly start the existing fenced resnapshot path. Confirm root through the snapshot and publish the complete identity atomically with that snapshot; do not publish loader-only identity. DOM.documentUpdated can still trigger the same path; duplicate notifications for the same completed identity do not clear feedback. Same-document app updates and fragment navigation preserve load when the authoritative triple is unchanged.

## Hot-path costs and operating envelope

This is an interactive local tool, co-tenanted with suites and agents. Do not add network services, background polling, screenshots per row or new capture workers. Existing writer/capture bounds remain authoritative; new identity strings use bounded CDP identifier lengths and reject oversized metadata at decode. Record exact existing limits during implementation before choosing constants; do not silently add a lower recording cap.

| Growth driver | Work and bound | Counted guard |
|---------------|----------------|---------------|
| F current feedback rows | One projection pass O(F); keyed DOM reconciliation O(F), no nested row search | `feedback-entries.test.js` N=32/64 counts visits and key lookups, <= constant×F |
| P visible sketch paths / V vertices | Reuse one existing painter pass and cached bounds; hover glow reuses that geometry without rebuilding paths | Hover fixture N=8/16 sketches counts geometry visits; highlighting one keyed item never rescans all ink |
| R resize/drag events | One layout computation and one measurement per update, independent of row count | Layout/DOM test counts measurements at 32/64 rows and rejects row-dependent reads |
| E recorder events / S snapshot nodes | Constant event classification; one bounded resnapshot per admitted generation, O(S) snapshot | Fake-CDP N=8/16 duplicate events counts snapshot calls, overlapping work bounded by current recorder owner |
| H retained feedback / L activation segments | O(H) projection with O(1) load lookup, not a scan of L for each item | Reducer/Board N=32/64 items and segments counts membership lookups |

UI-only panel/highlight state dies with mount teardown; observers, drag listeners, pointer capture and highlight handles are released on cancel, off, detach and dispose. Keyed row caches remove deleted/noncurrent keys rather than retaining every visited page. Page-load metadata grows once per existing segment and is removed with its recording under existing retention; there is no new durable index, cache or background process. Mock assets archive with the plan. Tests use disposable folders/profiles and join all owned work.

## Implementation tasks

### 1. Layout mock and approval (#856, design artifact)

- [x] Commit this plan and issue decisions before long-running checks or serving.
- [x] Build the synthetic clickable mock at the paths above, with one fixture source and explicit screen/version markers following existing mock examples.
- [x] Exercise the flow at 320, 768 and desktop widths; inspect toolbar/panel rectangles in a bounded headless browser. Keyboard arrow movement is exposed for operator review; native flow covered mixed rows, read-only labels, unsaved/save, unavailable anchor, tab return, app update, reload partition and tool reveal.
- [x] Serve the mock, send URL plus mixed-list/tool-reveal/narrow-window review to the hub, and record operator decisions in Revisions. Operator accepted the reworked layout.

### 2. Unified panel and column (#856, after #850/#849 land)

- [x] Add failing pure tests for `feedbackEntries` and `feedbackLayout` using the strategy table below; run Node tests and confirm the intended failures.
- [x] Implement projections/layout without IO; reuse the floating-panel reducer, painter and existing note/sketch IDs and commands.
- [x] Add the thin keyed panel adapter and connect it to annotate/sketch models. Remove #850's separate selector and its obsolete tests; retain its grouping/click/viewport regressions under the unified surface.
- [x] Move existing live/pause and sketch controls into one protected column; route status publication to the Notes header. Preserve control revisions, driver/off rules and UI-suppression gate.
- [x] Keep `noteDialog.unsaved` and `__couchcastNoteUnsaved` through editor mounting; keep sketch finished-label view read-only. Implement confirmed-anchor scroll/highlight with explicit unavailable result. Row hover and keyboard focus glow only the corresponding current-load sketch ink or annotation box, without scrolling or changing feedback selection.
- [x] Run focused Node and native operator tests, update `atlas/webcast-submission.md` and the relevant webcast atlas map, audit reader inventory, push, and run `sdlc local-check run --pre-close`.
- [ ] Close through SDLC's fresh review, fix findings on the tested source, and coordinate operator stage/live acceptance through the hub. Land #856 before starting #857's branch.


### 3. Page-load model and recorder fence (#857, independent branch)

- [ ] Start #857 from landed main and checkpoint its approved dependent plan; run its own change-code gate before implementation.
- [ ] Add failing pure codec/reducer tests for PageLoad validation, creation membership and additive extension preservation; run and inspect red results.
- [ ] Implement typed load metadata on segments and derived feedback provenance, including catch-up and snapshot retention; leave sketch receipt canonicalization unchanged.
- [ ] Add failing stateful CDP ordering tests at `Recorder.document`, navigation handling and resnapshot; implement pending/confirmed generation ownership with direct top-navigation trigger.
- [ ] Run focused recording/recorder tests, including late snapshot completion and replay/catch-up parity, then commit.

### 4. Current-load consumers and qualification (#857)

- [ ] Add failing Board/JS parity regressions at the selection and projection seams; replace historical-live-selection expectations while retaining unfiltered history-query assertions.
- [ ] Filter Notes, anchored pins, sketch ink, label actions and submission membership from the same derived load facts. Board filters atomically before `PressSelection`; later navigation cannot mutate an already admitted selection.
- [ ] Expose load provenance in agent query/source output; keep history query unfiltered and avoid changing existing privacy/delivery semantics.
- [ ] Run fake-CDP and non-skipped native Chrome qualification through existing disposable fixtures; retain exact source-bound evidence for the reload/tab/app-update flow.
- [ ] Update atlas with page-load versus activation-segment semantics and legacy behavior, audit reader inventory, push and run recorded pre-close. Close, obtain staged operator acceptance through the hub, then merge and report.

## Verification strategy

Tests are code, not a prose case inventory. The following maps production seams to adversarial input classes and independent invariants; executable matrices live in the tests.

| Production seam | Adversarial classes | Mechanical invariant |
|-----------------|---------------------|----------------------|
| `feedbackEntries`, keyed panel adapter | Mixed IDs/revisions, deletion, delayed model update, untrusted labels | One current row per authoritative key; text insertion, no markup execution; no replacement of active editor |
| `feedbackLayout`, floating panel transitions | Small bounds, keyboard/drag cancel, resize, teardown | Protected toolbar never overlaps panel/button/popup; every session resource ends |
| Tool column and status routing | Paused/driven/off, history unavailable, stale control revision | Existing action admission unchanged; one column and one status owner |
| `noteDialog.unsaved`, label/show commands | Changed text, held editor, finished label, ambiguous/stale anchor, hover/focus interrupted by reload or removal | Submit guard preserved; final label immutable; no guessed scrolling/navigation |
| PageLoad decode/rewrite and reducer | Missing/invalid/duplicate fields, unknown nested fields, legacy record, receipt-bearing sketches | Typed strict owned fields; unknowns preserved; digest and historical data unchanged |
| Recorder event/resnapshot transition | Duplicate/out-of-order navigation, detach, iframe, delayed capture | Pending before IO; only newest confirmed top-load snapshot becomes live |
| Board selection and Go/JS display projection | Different loads/targets, tab-return segments, delivered revisions, unknown scope | UI and press use same membership; exact admitted versions stay pinned |
| Native Chrome and history query | Real reload versus same-document change and tab lifecycle | Reload partitions UI/submit, same load restores, complete history remains queryable |

Commands: `node --test cmd/internal/webcast/web/*.test.js`; targeted `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder ./cmd/internal/webcast/query`; inspect the existing live test selectors and run them non-skipped with `WEBCAST_LIVE_CHROME=1` and the package's required native environment. Native watchdog must exceed scenario plus joined cleanup. Before every close run `make test-affected-inventory`, auditing each named reader rather than bulk refreshing certificates. Recorded verification uses `sdlc local-check run --pre-close` on the pushed source; train suite is the full landing check.

## Architecture decisions

- **ARCH-DRY:** One saved-feedback owner, one panel, one toolbar, one load source and existing press-selection owner; remove superseded selector rather than retain two controls.
- **ARCH-PURE:** Entry/layout/load/membership logic stays pure; DOM/CDP/file adapters execute effects.
- **ARCH-PURPOSE:** Both live display and submit partition by page load; history remains agent-queryable. The mock settles actual layout before production code.
- **ARCH-MOCK:** Reuse stateful fake Chrome and disposable recording folders, compare with native Chrome. No live fleet identity or browser state in automated tests.
- **ARCH-CONSTRAINTS:** Counted linear projections, bounded caches and one recorder generation; reuse existing storage/capture caps and no new screenshot worker.
- **ARCH-SECURE:** DOM labels are untrusted text; persisted identity validated at decode; target/root evidence comes from the trusted CDP recorder, not page-provided IDs. No credentials added.
- **ARCH-ORDER:** Pure panel transitions and recorder pending/confirmed generation own event ordering; atomic press snapshot remains valid through later navigation.
- **ARCH-FUNERAL:** Mount owns listeners/caches; recording retention owns additive metadata; tests join profiles/processes and plan mock archives with issue.
- **ARCH-EVOLVE:** Add optional segment metadata with strict owned fields and raw nested extension preservation; derive sketch membership without changing canonical digests; legacy unknown stays explicit.

## Revisions

### 2026-10-23 — TL decisions and #849 integration

Joint design requested for morning testability, with #857 still a separate main-based branch. Finished sketch labels stay read-only. Show on page uses only a positively confirmed current-load anchor. Incorporate #849's new press-time selection and unsaved-note-only guard; the preview is informational and must not regain submission authority. Earlier-load feedback has no UI drawer. Operator mock approval and plan approval remain required before production implementation.

### 2026-10-23 — Joint plan approved

Authenticated TL receipt bdff0bdf-86f8-4099-94c7-8de3cfac6ab4 approves this joint plan, including pending/unknown load contributing no page feedback and keeping the unsaved-note guard. The separate operator layout approval is still outstanding. Mock-1 is served locally for that review; production implementation remains held.

### 2026-10-23 — Operator mock feedback: icons and hover association

Receipt b5c92d0c-2c24-4098-bb02-e83190857da9 supersedes thumbnails: use a small static sketch icon distinct from the annotation icon. Hovering an entry makes only its corresponding sketch ink or anchored note box glow. Include equivalent keyboard focus behavior, clear highlight on lifecycle transitions, and retain current-load/confirmed-anchor guards. This removes thumbnail generation/cache work entirely. Mock-1 is revised in place; no production code changes.

### 2026-10-23 — Real tool names and isolated mock viewing

Operator feedback 96da76d2-2673-40a2-b1aa-38ef59fe0206: show the actual tools in one column: Live/Pause, Pointer, Sketch (Pen) and its colors, Annotation (Note), available Undo/Redo, Done. Remove the artificial Stroke action; drawing on the synthetic page exercises history instead. Pointer/annotation remain discoverable while sketch-specific tools reveal on Pen. This clarifies, rather than adds to, the production tool adapters.

The operator opened the mock inside the live cast Chrome and real injected tools covered it, including another tab's notes. View this mock in a non-cast Chrome window; do not change recorder origin-exclusion policy just for the mock. Add a New tab scenario with empty feedback even at the same URL. #857 membership remains exact target plus loader plus root, never load-only or URL-only: a new target starts empty, while return to a known same target/load restores its rows. Preserve the live observation as a native regression requirement using synthetic captures without copying private screenshots.

### 2026-10-23 — Layout accepted; production authorized

TL receipt e378240d-75dd-498d-96e8-91b542182269 records the operator’s “yes, like it” and authorizes #856 production followed by #857, aiming for the combined morning test stage with #849. No further layout approval is pending. Production still crosses change-code and consumes dependent issue APIs only after their main landing; pure independently scoped helpers can be prepared first without stacking commits.

### 2026-10-24 — Production integration and palette decision

#850/#849 are landed and consumed from main. Added thin `web/feedback-glow.js` canvas owner and `web/cast-tools.js` control-adoption owner alongside the planned panel: the former keeps hover pixel-only to avoid MutationObserver-driven visibility stamp churn; the latter reuses actual button listeners and permissions. `createPlacement.reveal` factors confirmed-original validation before scrolling and then rechecks placement. The separate sketch selector is removed, retaining only its pure painted-bounds grouping helper. Pending feedback now supplies header text and retains its publication/ack checks without DOM cards.

TL receipt 0ed24a81-0dcd-481a-a33b-9a6ee394b937 approves the one mock deviation: pen colours remain visible/selectable before starting, since the accepted begin fixes the colour and the existing adapter disables changes while active. No extra arming state is added. Undo/Redo/Done remain conditional. Morning stage steps must call this out.

Actual Node command is `node --test tests/webcast/*.test.mjs`; tests reside there, not web/*.test.js. New IO helpers are tested by `feedback-glow.test.mjs`, `cast-tools.test.mjs` and `feedback-panel.test.mjs`; trusted reveal extends `annotation-placement.test.mjs`. Runtime assets are embedded/routed in server.go and imported by the actual operator bundle. Existing source-bound native fixtures are migrated to the unified panel and column, retaining privacy and lifecycle assertions.

### 2026-10-24 — Landing and morning production-look sequencing

TL66eb5fd0-b8a4-40d7-a896-3ba247374694 approves #856 landing after SHIP and exact-source checks on the accepted mock plus native evidence; the combined morning stage covers the production look, with subsequent operator changes as follow-ups. #857 then starts on its own branch from landed main. The stage is main plus #857 if ready plus #842, only after hub sequencing. This supersedes the earlier pre-landing production-look ordering for #856.
