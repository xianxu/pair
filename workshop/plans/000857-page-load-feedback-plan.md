# Page-load feedback implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent work or superpowers-executing-plans for work requiring session context. Steps use checkbox syntax for tracking.

**Goal:** A real reload or new tab starts with no earlier page feedback in the live UI or submit selection, while the recording retains all history and returning to the same tab/load restores its feedback.

**Architecture:** Persist an optional exact target/loader/backend-root identity on existing segment boundaries. Derive feedback membership from accepted creation segments without altering sketch canonical requests. The recorder fences navigation at ingress before synchronous snapshot IO; Board uses the same confirmed scope for atomic press selection and live projections.

**Tech Stack:** Go recording/reducer/Board, CDP recorder, existing browser JavaScript models, stateful fake Chrome, Node tests and disposable native Chrome fixtures.

## Approval and sequencing

This is the #857 portion of the joint #856/#857 design approved by TL receipt bdff0bdf-86f8-4099-94c7-8de3cfac6ab4. The operator's layout acceptance and subsequent #856 corrections belong to #856. This plan adds no new layout.

TL receipt 8b5caf7d-587c-44ab-9f60-a0832c521a5a authorizes data and recorder work now on this separate current-main branch, copying no #856 commits. UI integration waits for #856 to land. When #856's retained exact-source check passes, checkpoint this branch and return immediately to its merge. After landing, consume #856 through main, then finish this issue's consumers. One close boundary, no artificial task milestones.

The morning combined stage is hub-sequenced main plus #857 if ready and #842 if unlanded. Do not replace another slot's stage. Earlier-load feedback has no UI drawer; it remains available to agents through recording queries. #855 is separate.

## Core concepts

Paths are relative to `cmd/internal/webcast/`.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| PageLoad: exact browser load identity and equality | `recording/page_load.go` | new |
| Segment codec with recursive raw extensions | `recording/segment_codec.go`, `recording/codec.go`, `recording/record.go` | new / modified |
| Segment-to-load membership index | `recording/page_load.go`, `recording/reduce.go` | new / modified |
| Load transition state/events/effects (`nextLoadScope`, one-cell helper) | `recorder/load_scope.go` | new |
| Current-load press selection | `board/submission.go` | modified |
| Historical load provenance | `query/querier.go`, `query/sketch.go`, `board/feedback_context.go` | modified |
| Browser load membership and exact root validation | `web/page-load.js`, `web/model.js` | new / modified |

`PageLoad` owns opaque nonempty `Target` and `Loader` strings and positive int64 `Root`, the CDP backend root rather than a recording NodeID. Equality compares these three owned fields; missing load never matches another missing load. A recording has many loads; a load may have many activation segments. `DocSeg` stays the existing activation/draft fence, not a substitute identity.

`Seg.PageLoad` is optional for legacy data. Derive ordinary-note membership from `NoteView.Seg` and sketch membership from `SketchBegin.Doc`, looking up retained segment metadata in O(1). This avoids duplicating identity on every event or changing sketch digests/receipts. The lookup survives DOM reset. Each accepted segment contributes at most one index entry; recording retention owns its lifetime. Return immutable copies rather than mutable index pointers.

Historical query is unfiltered and exposes additive provenance. Live consumers use one shared membership predicate. Future load metadata extends the segment object, not canonical sketch admission. No new URL-based or capture-id-based identity exists.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Recorder ingress load owner | `recorder/page_load.go`, `recorder/recorder.go` | new / modified | queue ingress, selected session/top-frame facts, generation and snapshot cancellation |
| Snapshot acceptance | `recorder/recorder.go`, `recorder/mirror.go` | modified | frame-tree and document CDP IO, accepted segment/snapshot writes |
| Board live scope | `board/page_load.go`, `board/board.go`, `board/operator_view.go`, `board/sketch_delivery.go` | new / modified | pending/confirmed scope, draft admission and subscriber publication |
| Catch-up and rewrite | `recording/catchup.go` (existing consumer), `recording/contract.go` (modified), `recording/page_load_test.go` (new qualification) | reused / modified / new | persisted replay and compaction |
| Browser live projection | `operator.js`, `operator-sketch.js`, `web/annotate.js`, `web/sketch-render.js`, `web/sketch.js` | modified | current rows, pins, ink, labels, hover and confirmed anchor actions |
| Generic panel and entry adapters | #856's `web/feedback-entries.js` and panel/glow adapters | reused unchanged | already consume the filtered note/sketch projections and existing lifecycle callbacks |
| Durable submission provenance | `../submission/codec.go`, `../submission/sketch.go`, `../submission/sketch_page.go`, `../submission/page_load_test.go` | modified / new qualification | typed page-load facts at sketch source, finish and shared scene boundaries |
| Recorder fake Chrome | `recorder/recorder_test.go` | modified | targets, loaders, roots, event/reply ordering, held/failing snapshot IO |
| Native lifecycle fixture | `page_load_live_test.go` | new | existing disposable browser/profile helpers, real reload and tab lifecycle; `operator_live_test.go` reused unchanged |

The recorder owns a small mutex-protected ingress token containing generation, selected session/target and known top-frame identity. It does not read loop-owned maps unsafely. The recorder actor still serializes all snapshot execution; no new worker or polling loop is introduced. The Board remains a leaf lock, with no callback into recorder while held. Cancellation and close join existing owned work.

## Persisted contract and evolution

New segment example: `"page_load":{"target":"TARGET","loader":"LOADER","root":"123"}`.

Absence is legacy unknown; explicit null, missing owned members, wrong types, empty strings, malformed Unicode, duplicate keys and nonpositive/noninteger/overflow roots refuse. Require target to equal segment tab ID. Preserve legacy acceptance for unrelated existing fields rather than tightening the whole recording schema accidentally. Target and loader are opaque: no trimming, case folding or invented syntax.

Route segments through a dedicated codec using strictjson recursive split/encode and raw extension storage at segment, tab, viewport and load depths. Preserve arbitrary valid unknown JSON numeric tokens without float64 conversion. Exact owned discriminator names must not be selected through encoding/json's case-insensitive aliases. Encoding a typed edit preserves unknown raw values and never mutates shared records. Keep existing sketch decode/admission and canonical bytes unchanged.

There is no existing individual CDP identifier limit: CDP's entire message is capped at 16 MiB, while recording codec/writer have no generic line cap. Do not borrow SketchMaxIDBytes=128 or add a lower identity cap. Root is encoded as a canonical positive decimal string everywhere, with Go validating the value is in 1..MaxInt64 on read. Refuse leading zeros, signs, whitespace, fractions, exponent notation and numeric JSON values; do not round through JavaScript Number or maintain a secondary identity key. This corrects the joint plan's ungrounded reference to bounded individual CDP IDs without adding a new policy.

Catch-up retains complete segment records, including inactive creation segments, before dependent feedback and the sketch semantic snapshot. Do not carry identity through sketch clones: those intentionally clear their original wire extensions. Compare full replay, every-prefix catch-up and repeated compaction for both membership and historical metadata.

Pin additive evolution with a frozen preserving older segment schema: it decodes page_load as raw unknown, edits an owned legacy field, rewrites, then the current reader recovers exact identity and nested extensions. Already-shipped permissive older readers may drop unknown fields; document that limit, do not claim retroactive preservation. Existing historic recording lines are not migrated.

## Recorder ordering and Board ownership

`document()` currently performs synchronous CDP IO and mutates loader/snapshot sequence before acceptance (`recorder/recorder.go:802`). Return local `{root, loader, frameID, replySeq}` evidence instead. Frame-tree failure cannot reuse an old loader as authority. Reserve a matching generation before installing caches, mirror state or segment metadata; confirm Board scope only after successful accepted document writes and only if that generation still owns the scope.

Selected top-frame navigation, target activation/change and detach invalidate the token and publish Board pending scope at ingress, before snapshot IO or queue delay can expose stale page feedback. Classify from synchronized selected-session/top-frame facts and event parent identity, never actor maps. Iframe/OOP-frame events do not replace top scope. Top Page.frameNavigated directly schedules the existing resnapshot path; DOM.documentUpdated shares it. Same-document navigation preserves identity. Duplicate notifications for an already confirmed exact identity do not create a new load or clear its feedback.

Snapshot cancellation is owner-scoped. An unreserved superseded reply cannot mutate caches; a previously reserved commit may finish its historical writes but cannot confirm scope, clear a successor's pending state or mark the successor non-live. Keep execution serialized and cancellation-aware. Snapshot failure, missing root, detach or close leaves scope unavailable, recoverable through the existing explicit capture retry; never restore old scope optimistically.

### Enforced transition model and linearization (revised ownership)

This section supersedes the initial latest-generation-only reservation model. TL32581055 requires preserving queued activation history; TLf4d3e2cb approves one load owner with per-session epochs and ordered tickets. No visible state, user action, deadline or failure policy changes.

The conceptual distinction is **ordered recording intent versus current UI scope**. Live/Pause commands already own FIFO recording order; a later command must not retroactively cancel an earlier activation. Navigation can invalidate an in-flight observation of a session, while a newer selected tab merely prevents an older activation from becoming current UI authority.

One private `loadOwner` owns, under one mutex:

- Stable session cells: trusted session/target/top-frame facts, navigation epoch and attached state. The existing session lookup owns each cell; outstanding private tickets retain a detached cell only until settlement.
- Immutable ordered snapshot-intent tickets: serial, session-cell identity and cause (activation or refresh). Live creates an activation ticket at ingress; top navigation/document invalidation creates a refresh intent. Existing dirty/resync triggers obtain that same ticket kind before IO. Tickets are carried by already-owned recorder queue work, not a second queue or writer. A ticket can produce one accepted snapshot reservation; a superseded attempt leaves the same intent unfulfilled.
- Newest UI scope: monotonic generation, selected activation/session, pending/confirmed identity and intended-live flag. Pause changes intended-live/control state while retaining the same load, not clearing saved notes. Closing prevents new admission/UI confirmation while historical queue drain continues.
- At most one actor capture attempt or reserved commit. A capture samples session epoch and, only when it corresponds to the selected activation, the current UI generation. A reservation carries that attempt and immutable snapshot evidence.

`nextLoadScope` plus a pure one-cell transition helper receive owner values, the relevant session-cell value and a typed event; return next values and typed effects. The thin owner performs one session lookup, updates only that cell and applies effects under its mutex. No map-wide copy or mutation hidden inside a pure function. Private fields/ticket constructors and one `commitTopSnapshot` path prevent selected-top writes bypassing authority. The current committed pure prototype is replaced to implement this revised model before wiring production.

| State / event | Authority transition | Effect |
|---------------|----------------------|--------|
| Open / Live intent | Issue immutable activation ticket; select it and advance UI generation to pending | Enqueue FIFO intent, publish pending; later Live never revokes its historical ticket |
| Already intended-live on same target / duplicate Live | No new intent or generation | Preserve existing no-op command behavior |
| Open / Pause intent | Retain tickets, session epochs and load; intended-live becomes false | Existing FIFO Stop/control semantics; notes remain associated with the same load |
| Tracked session / real top navigation or document invalidation | Advance that session epoch; create/coalesce ordered refresh work; advance UI pending only if selected | Cancel only the matching session's unreserved capture; iframe/same-document events do not advance load epoch |
| Actor / SnapshotStart(ticket) | Sample current attached session epoch and trusted facts; UI generation only if this intent is selected | One serialized capture outside mutex; a navigation before start is observed at the current epoch rather than inventing old snapshot evidence |
| Capturing / SnapshotReady(ticket, epoch, complete evidence), unchanged epoch and matching session/target/frame | Reserve ticket once under mutex | Return opaque historical commit reservation independent of newest UI generation |
| Capturing / epoch superseded during IO | Discard unreserved evidence, preserve unfulfilled intent | Same actor observes the new epoch and retries that intent before advancing FIFO; at most one attempt per newly observed epoch, no timer or polling |
| Capturing / real failure or detached cell | Intent fails through existing unavailable/error path | Keep current scope unavailable if affected; no guessed prior loader; saved history retained |
| Reserved / later navigation or selected activation change | Reservation remains valid for ordered historical write; newest scope remains pending | Never cancel/retract accepted sink work; no ingress wait on sink IO |
| Reserved / successful CommitDone | Consume reservation and settle ticket once | Confirm Board only if selected activation, captured UI generation and session epoch still match; otherwise history only |
| Reserved / failed CommitDone | Consume reservation, retain sink failure | Current affected scope unavailable; no false rollback or confirmation |
| Any / stale, duplicate or foreign-token completion | No authority change | No cache/scope effects or successor cancellation |
| Open / Close | Stop issuing tickets and UI confirmation; keep accepted FIFO drain | Existing actor processes already-issued intents/reservations, then closes/cancels/joins owned work |
| Session / Detach | Mark cell detached, remove its lookup, invalidate affected current scope | Cancel unreserved session capture; retained ticket cannot bind to a new cell reusing the session string |

Snapshot acceptance linearizes at the successful ticket/epoch reservation transition under this same owner mutex. Historical mirror/cache installation and sink writes then execute on the sole actor outside it. The actor's serialization prevents an older reserved commit overwriting a newer actor installation. An unreserved old observation cannot commit. Post-write confirmation reacquires the owner and validates current selected activation, sampled UI generation and session epoch; publish to the leaf Board lock only while those still match. All transition publications release before queue ownership or CDP/sink IO. Board never calls back into the owner.

If initial Live capture is superseded by navigation during IO, its unfulfilled intent retains the initial activation reason while retrying the new observed epoch; it cannot yield to a following Pause and silently lose the initial segment. A refresh observing an already completed exact identity may resnapshot but creates no duplicate activation. The attempt-count guard counts actual CDP starts against activation/refresh intents and epoch advances, including cancelled attempts. A continuous navigation stream cannot manufacture attempts without new event evidence.

Any accepted write failure leaves current affected scope unavailable; already appended records remain history and existing sink-error handling stays authoritative. The visible state/recovery table remains unchanged. Native/fake tests retain the existing queued lifecycle test unchanged, plus cancellation mid-capture, navigation mid-write and close drain.

Board's scope transition accepts a monotonic generation and pending/confirmed value under its lock. Pending immediately invalidates live projections and page-feedback creation admission; confirmation succeeds only for the current generation and matching accepted state. Reducer metadata alone supplies replay facts; runtime pending overlays only the live owner. Preserve original DocSeg and annotation/control authority fences. Audit lock order across every caller before adding an acquirer.

`selectionLocked` filters by this scope under the same Board lock before #849's PressSelection removes delivered versions and applies the eight-sketch limit. Pending/unknown contributes no page feedback, without a new global submit refusal; ordinary text and the unsaved-note guard remain. A selection already admitted retains exact versions through later navigation. Historical queries stay unfiltered.

## State and recovery

Actions: wait, retry capture, pause/resume, switch tab/back, reload, add/edit/save/cancel feedback, submit, query recording history.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|-------------------------------|---------------|-----------------|---------------|
| Confirmed current load | Only that load's notes/sketches and controls | Add/edit/save feedback | None |
| Navigation or activation pending snapshot | Loading page, no earlier feedback or creation controls | Wait | Existing stale draft follows current cancellation policy; saved history retained |
| Failed snapshot, missing identity or detach | Current page unavailable; no page feedback | Retry capture | No saved history lost |
| Older completion after newer navigation | Newer loading/current state unaffected | Wait | Superseded snapshot discarded |
| Same-URL new loader/root or new target | New load starts empty | Add feedback | Earlier items leave UI only |
| Pause/resume or A→B→A with same triple | That load's feedback returns | Open/edit feedback | None |
| Same-document app or fragment update | Same load remains | Continue | None |
| Unknown legacy load | History queryable; unidentified items excluded from identified live scope | Resume capture | No history loss; no inferred membership |
| Submit admitted before navigation | Exact submitted versions remain pinned | Existing submit recovery | None beyond existing delivery policy |
| Unsaved ordinary note during submit | Existing unsaved-note refusal | Save or cancel | Cancel discards only draft |
| Restart/catch-up | Same confirmed history and membership as full replay | Resume capture | Pending runtime observation is re-established, never guessed |

Covered: real top navigation, snapshot races/failure, tab identity, same-load restoration, legacy unknown, atomic selection and history. Not covered: editing finished sketch labels, earlier-load UI browsing, browser-origin exclusion or submission instruction rewrite; those are separate scope decisions.

## Hot-path costs and resource bounds

| Growth driver | Work | Counted test |
|---------------|------|--------------|
| E CDP events / A queued snapshot intents | O(1) classification and one session-cell update; one serialized actor capture | 8/16 notifications plus queued activations count starts <= A + 2E, concurrent calls <=1, no duplicate activation for unchanged completed identity |
| S snapshot nodes | Existing one-pass O(S) document processing per admitted snapshot | Existing fake tree counter plus delayed/cancelled IO proves no extra scan or orphan work |
| Q queued intents / D attached sessions | One constant ticket per existing queue input, one cell per tracked session; retire with settlement/detach/close, no new durable index | Queued lifecycle/close test counts issued/settled tickets and retained detached cells |
| L activation segments | One O(1) index insertion each, O(L) retained metadata | 32/64 segments count index writes, full/catch-up parity |
| H feedback items / I identity bytes | One O(1) segment lookup per item; exact validation/comparison is O(I), so O(H*(1+I)) with no segment rescan | 32/64 mixed notes/sketches count lookup visits, reject nested segment scans |
| F displayed rows / P painted paths | Existing linear projection/painter; membership checked once per item | Node 32/64 fixture counts visits and clears noncurrent keyed state |

Reuse current queue bounds (100,000 DOM and non-DOM events), CDP call timeout (5 seconds), CDP frame limit and recording retention; do not change production deadlines. Identity metadata adds once per existing segment, no separate disk index or service. Mounted UI caches discard departed keys; hover and late replies fence by exact scope/revision. Tests use disposable roots/profiles, cancellation-aware fake gates, and joined owned cleanup.

## Implementation tasks

### 1. Recording identity and replay

Files: create `recording/page_load.go`, `recording/page_load_test.go`, `recording/segment_codec.go`, `recording/segment_codec_test.go`; modify `recording/record.go`, `recording/codec.go`, `recording/reduce.go`, `recording/contract.go` and their tests. Reuse `recording/catchup.go` unchanged: its existing complete-segment retention carries the new preserving metadata.

- [x] Write failing PageLoad and segment-codec tests using the function strategies below. Run `go test ./cmd/internal/webcast/recording -run 'Test(PageLoad|Segment)'` and inspect intended red failures.
- [x] Implement optional segment metadata and strict preserving segment codec. Add segment index and pure membership helpers without changing sketch request/digest fields.
- [x] Add preservation/replay tests through the production functions below. Run `go test ./cmd/internal/webcast/recording`; require PASS.
- [x] Commit the model/codec checkpoint with issue and author trailers.

### 2. Recorder ingress fence and Board scope

Files: create `recorder/page_load.go`, `recorder/page_load_test.go`, `board/page_load.go`, `board/page_load_test.go`; modify `recorder/recorder.go`, `recorder/mirror.go`, `recorder/recorder_test.go`, `board/board.go`, `board/submission.go`, `board/operator_view.go`, `board/sketch_delivery.go` and focused tests.

- [x] Add failing stateful fake-CDP and sink gates with explicit entry/release barriers and cancellation join, using the ordering strategies below; retain established event-order and stale-draft tests.
- [x] Implement the ingress-owned token and Board generation transition. Keep snapshot return evidence local until generation acceptance; propagate complete identity on each segment including resume/tab-return.
- [x] Add atomic selection and scope-publication tests through the named functions below. Add counted 8/16 and 32/64 guards from the cost table.
- [x] Run `go test ./cmd/internal/webcast/recorder ./cmd/internal/webcast/board` and focused `-race` ordering cases; require PASS and all fake work joined. Commit.

### 3. Live consumers, after #856 lands

Files: `web/model.js`, `web/annotate.js`, `web/sketch-render.js`, `web/sketch.js`, `operator.js`, `operator-sketch.js`, `board/feedback_context.go`, `query/querier.go`, `query/sketch.go`; tests in `tests/webcast/*.test.mjs` and colocated Go tests. Reuse #856's generic panel/entry/glow adapters unchanged through the filtered projections and lifecycle owners.

- [x] Merge landed main only after #856's landing; reconcile the actual entity table/file ownership before editing. Do not import its unlanded commits.
- [x] Add failing reducer/operator parity tests using the membership and lifecycle strategies below. Assert no old rows, pins, ink, labels or controls survive scope departure.
- [x] Filter all live feedback from the same derived current-load source, preserving selection/delivery/privacy stamps and #849's getter/unsaved guard. Clear transient hover/editor state through existing lifecycle owners; no new feedback store.
- [x] Add unfiltered query provenance assertions and exact root-string round trips. Run `node --test tests/webcast/*.test.mjs` and `go test ./cmd/internal/webcast/query ./cmd/internal/webcast/board`; require PASS. Commit.

### 4. Native qualification and landing

- [x] Extend existing disposable native fixtures with same-URL reload without a later user DOM edit, same-load resume/tab return, app-only update and new same-URL target. Assert current UI/press selection and retained query history independently; use actual navigation events and completion barriers, no sleep as ordering proof.
- [x] Run the existing named native tests with `WEBCAST_LIVE_CHROME=1`, a watchdog covering scenario plus joined cleanup, and retain complete non-skipped source-bound output. Preserve a synthetic regression for the operator's new-tab observation, never private screenshots.
- [x] Update `atlas/webcast-submission.md` and relevant recording/recorder map/index for load versus activation, pending/unknown behavior, query provenance and old-reader limits. Reconcile concepts table against actual ownership.
- [ ] Stage new tests, run `make test-affected-inventory`, audit every named reader individually, push and run `sdlc local-check run --pre-close`. Close through the mandatory fresh SDLC review; fix blockers on newly verified source.
- [ ] Ask hub to sequence the morning combined stage and operator steps. After acceptance and checks, enter `sdlc merge --yes` promptly; report actual landed PR/train, clear any owned stage through SDLC and finish cleanup.

## Function-level verification strategies

Names introduced here are intended interfaces; reconcile names with implementation without changing ownership or test invariants.

| Production function | Adversarial input/event class and mechanical guard |
|---------------------|--------------------------------------------------|
| `PageLoad.Validate`, `SamePageLoad` | Generated identity triples and malformed owned values: strict validation, exact equality, unknown never equals unknown, target/tab consistency |
| `decodeSegment`, `encodeSegment`, `Decode`, `Encode` | Fuzz JSON object structure and Unicode/numeric boundaries; strict owned authority, recursive duplicate refusal and raw unknown-byte preservation after typed edits, no shared mutation |
| `State.Apply`, `State.PageLoadAt`, `State.CurrentPageLoad` | Generated activation/feedback sequences: creation membership immutable, index survives DOM reset, one lookup per item, draft DocSeg unchanged |
| `CatchUp`, `CheckCatchUp` | Every generated valid prefix and repeated compaction: full replay and catch-up agree on load facts and complete history, sketch canonical bytes/digests/receipts unchanged |
| `nextLoadScope` and pure one-cell transition | Generated ordered-intent/session-event sequences: queued activations preserved, monotonic epochs, one reservation per intent, stale attempts have no effects, current confirmation requires matching selected activation/generation/epoch, close drains history and ends terminally |
| `Recorder.enqueue`, `document`, `commitTopSnapshot`, `snapshotTop`, `resnapshot` | Stateful fake interleavings with held/failing CDP and sink boundaries: pending precedes IO, reservation linearizes acceptance, superseding navigation remains responsive mid-write, no stale confirmation, joined cancellation and one actor writer |
| `Board.BeginPageLoad`, `Board.ConfirmPageLoad`, `selectionLocked`, `Selection.PressSelection` | Interleave scope transition and selection admission under barriers: one current scope, no unknown-page feedback, immutable admitted versions, existing delivered limits and unsaved guard preserved |
| `operatorViewLocked`, `OperatorSketchSince`, `sketchDisplayProjection`, browser model application / `feedbackEntries` | Cross-language replay of generated load transitions: current-only rows/pins/ink/label eligibility agree, complete unfiltered query history remains |
| `createFeedbackPanel`, annotation/sketch hover/reveal adapters as landed in #856 | Delayed placement and mount teardown sequences: departed-load replies cannot restore controls/highlights or target an earlier anchor; existing editor cleanup and stamp paths complete |
| `Querier.Notes`, sketch source-facts projection | Mixed-load history: all retained items remain queryable with exact provenance, including roots outside JS safe-integer range projected without rounding |

Frozen-schema rewrite and native reload/tab acceptance remain independent oracles rather than restating implementation output. Executable case inventories belong in tests, not in this plan.

## Architecture decisions

- **ARCH-PURE / ARCH-DRY:** One exact load entity and segment index; UI and submit derive membership, history stays complete. Existing Board/recorder and #856 panel remain owners.
- **ARCH-ORDER:** Ingress invalidates before synchronous IO; generation acceptance protects both state mutation and returned observations. Atomic press selection pins its own history.
- **ARCH-EVOLVE:** Optional segment metadata, recursive raw preservation and frozen preserving-reader rewrite; no change to canonical sketch digests.
- **ARCH-MOCK:** Stateful fake CDP with held and failing replies, compared with native disposable Chrome.
- **ARCH-CONSTRAINTS / ARCH-FUNERAL:** Counted linear work, existing budgets, one snapshot owner and joined cleanup; retention owns metadata lifetime.
- **ARCH-SECURE:** Trusted CDP identity, strict owned wire fields, opaque unknown preservation, no URL guesses and no credentials.

## Revisions

### 2026-10-24 — Independent data/recorder sequencing

Receipt 8b5caf7d-587c-44ab-9f60-a0832c521a5a supersedes only the joint plan's wait-before-branching order. Start from current main without #856 commits, work on data/recorder after this issue's change-code gate, wait for landed #856 before UI integration, and return to #856 merge as soon as its retained check passes. Read-only surveys locate the approved generation fence at ingress because the recorder loop blocks during snapshot IO. They also establish no individual CDP identifier cap; preserve the existing contract rather than invent one.

### 2026-10-24 — Plan-quality authority refinement

PQ-1 is addressed by the pure nextLoadScope transition table, private owner/reservation, shared selected-top commit function and explicit acceptance/confirmation linearization. TL96c958be approves reservation under the mutex followed by sole-actor IO outside it, and requires the held-write navigation regression. This preserves responsive ingress instead of holding its mutex across sink IO. PQ-2 is addressed by replacing procedural test-case lists with named production functions, adversarial classes and mechanical invariants; verification commands and counted budgets remain.

### 2026-10-24 — One exact wire identity

TL02693d4d-10fd-4b7f-a3fb-96d20a116b78 approves canonical positive decimal-string root in page_load, validated as int64 in Go, refusing leading zeros and signs. This supersedes the drafted numeric root plus special JavaScript projection. One representation spans disk, live transport, catch-up and browser membership, preventing lossy parsing without a second key. Round2 accepted PQ-1/PQ-2 with no new findings; implementation still waits for this final contract and estimate to clear change-code.

### 2026-10-24 — FIFO historical intent and current UI scope

Read-only recorder integration found that latest-generation-only reservations would discard earlier queued Live activations before snapshots start, violating unchanged TestRecorderSegments. TL32581055 requires queued history preserved. TLf4d3e2cb approves the revised transition section: per-session cells, immutable ingress tickets, epoch-fenced attempts with one event-driven retry per observed epoch, one actor outside the mutex for writes, newest-scope-only Board confirmation, and close draining already-issued history. This is the second mechanism change; a third requires re-planning recorder ownership as a whole. Recorder production edits remain paused until this revision clears plan-quality.

### 2026-10-24 — Detach fixture parity and landed UI integration

TL2eb592a8 approves ARCH-MOCK parity: fakeChrome refuses every method for a detached session. TestRecorderActivePageDetach first settles initial Live, retaining its exact A-then-B assertion; a separate pending-initial-capture detach test requires the existing snapshot-failure diagnostic, no fabricated A snapshot and a valid first-live B. This corrects fixture phase ordering under the already-approved detach rule, not the recorder ownership mechanism.

#856 landed as PR583 and is integrated through main. Actual ownership now includes the shared pure browser `web/page-load.js`, explicit current-scope fields on OperatorView and the existing sketch-delivery epoch reset. Projections filter before copying earlier pressure history; scope changes never append canonical clear records. Historical scalar query facts preserve comparable integrity witnesses. Trusted anchors reuse same-key mirror IDs and the existing anchor index. Board computes a sorted load projection once per anchor revision; recorder batches probe only requested nodes against that index, avoiding added whole-anchor copies/sorts. Existing original-binding delivery already scans feedback and DOM per batch: with B batches, H feedback and D anchor source-document buckets, new lookup is O(B*D*q) for fixed bounded q<=16, D<=H; it adds no cache or index lifetime. Allocation qualification at32/64 anchors retains four requested IDs, two matches and two allocations. Same-load inactive-node removals still pass through the existing graveyard policy.

### 2026-10-24 — Integration ownership and edge qualification

The scalar load provenance also crosses submission's preserving codec: optional page_load on source, finish and shared scene is owned there, typed-decoded canonically and validated against its source tab, with raw nested extensions retained. This completes the existing additive provenance design rather than changing selection. Initial managed pending scope returns an empty successful selection even before the first snapshot; unknown scope rejects fresh notes/pointers while history remains readable. Same-load return uses load membership for the existing per-color quota, and natural counter exhaustion refuses authority. Modern private fixtures now state identities; legacy wire histories remain unknown. All are qualifications of the approved model, not new visible states or recorder-ordering mechanisms.


### 2026-10-24 — Close round 1 ownership reconciliation and native recovery

BR-1 (ARCH-PURPOSE): reconciled the complete integration table and affected task file lists against the pinned Git diff. Catch-up already retains complete Seg records, so `recording/catchup.go` needs no edit; its unchanged path is qualified by the new load replay/compaction tests. #856's generic entry/panel/glow adapters already consume the filtered projections and lifecycle callbacks, so they remain unchanged. `operator_live_test.go` is reused, no `chrome_live_test.go` exists, and the new native scenario lives in `page_load_live_test.go`. Added the delivered durable submission codec/source/scene ownership omitted from the original table. This corrects traceability without inventing code changes to reused consumers.

BR-2 (ARCH-MOCK): the reviewer's disposable Chrome failed before page creation with GPU exit_code=6. On the exact reviewed head f5ffa47e6942019bc54a639630cfa1c8fe37a75b, reran `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestPageLoadLiveFeedbackLifecycle$' -count=1 -v -timeout=150s`: PASS5.460s, scenario5.28s, zero skips. Complete returned output is `000857-page-load-feedback-plan.side/evidence/native-close-r1-rerun.log`, SHA-256 cbaf43fc5b38aedb2cf2ccd2b49ba471966343020b4576c834c994b6ae9ba9a6. This is environmental startup recovery, not a production fix; next review should qualify it independently. BR-3 corrects atlas liveNotes load membership and unchanged adapter fallback.

Close measured actuals normally first, but refused because MeasurementBranchEvidence exceeds65536 bytes (ariadne#593); the hub's precise existing --no-actual authorization applies. The unchecked composite checklist includes pending stage and merge, so --no-plan-check records that sequencing without claiming acceptance. The retained recorded check remains lease-1008140854 on f5ffa47e; only documentation/evidence changes in this round. Operator staging waits for hub authorization and combines #857 with #866 after the existing morning stage frees pair:0.


### 2026-10-24 — Post-close Couch menu fixture qualification

The recorded full check exposed three Couch web-notes tests that construct a recording directly rather than running the recorder. The synthetic initial Seg lacked the load identity that production commitTopSnapshot supplies from CDP. Traced both real menu and HTTP admission first: the menu reads/resolves notes, the viewer request remains segment-based, and Board derives trusted identity from its accepted segment. Added explicit fixture load evidence only, with unchanged request and original behavioral assertions; improved diagnostic response text. Deterministic RED and whole-package/race GREEN are retained. This completes downstream qualification, not a new admission mechanism or scope change. Atlas and operator steps now name the approved legacy unknown-load refusal and reload recovery, per TL412ed457.
