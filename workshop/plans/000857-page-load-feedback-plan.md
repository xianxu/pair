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
| Current-load press selection | `board/submission.go` | modified |
| Historical load provenance | `query/querier.go`, `query/sketch.go`, `board/feedback_context.go` | modified |
| Browser reducer load membership | `web/model.js` | modified |

`PageLoad` owns opaque nonempty `Target` and `Loader` strings and positive int64 `Root`, the CDP backend root rather than a recording NodeID. Equality compares these three owned fields; missing load never matches another missing load. A recording has many loads; a load may have many activation segments. `DocSeg` stays the existing activation/draft fence, not a substitute identity.

`Seg.PageLoad` is optional for legacy data. Derive ordinary-note membership from `NoteView.Seg` and sketch membership from `SketchBegin.Doc`, looking up retained segment metadata in O(1). This avoids duplicating identity on every event or changing sketch digests/receipts. The lookup survives DOM reset. Each accepted segment contributes at most one index entry; recording retention owns its lifetime. Return immutable copies rather than mutable index pointers.

Historical query is unfiltered and exposes additive provenance. Live consumers use one shared membership predicate. Future load metadata extends the segment object, not canonical sketch admission. No new URL-based or capture-id-based identity exists.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Recorder ingress load owner | `recorder/page_load.go`, `recorder/recorder.go` | new / modified | queue ingress, selected session/top-frame facts, generation and snapshot cancellation |
| Snapshot acceptance | `recorder/recorder.go`, `recorder/mirror.go` | modified | frame-tree and document CDP IO, accepted segment/snapshot writes |
| Board live scope | `board/page_load.go`, `board/board.go`, `board/operator_view.go`, `board/sketch_delivery.go` | new / modified | pending/confirmed scope, draft admission and subscriber publication |
| Catch-up and rewrite | `recording/catchup.go`, `recording/contract.go` | modified | persisted replay and compaction |
| Browser live projection | `operator.js`, `web/annotate.js`, `web/sketch-render.js`, #856's `web/feedback-entries.js` and panel adapters | modified after #856 lands | rows, pins, ink, labels, hover and confirmed anchor actions |
| Recorder fake Chrome | `recorder/recorder_test.go` | modified | targets, loaders, roots, event/reply ordering, held/failing snapshot IO |
| Native lifecycle fixture | existing `chrome_live_test.go` / `operator_live_test.go` and recorder live fixtures | modified | disposable browser/profile/recording, real reload and tab lifecycle |

The recorder owns a small mutex-protected ingress token containing generation, selected session/target and known top-frame identity. It does not read loop-owned maps unsafely. The recorder actor still serializes all snapshot execution; no new worker or polling loop is introduced. The Board remains a leaf lock, with no callback into recorder while held. Cancellation and close join existing owned work.

## Persisted contract and evolution

New segment example: `"page_load":{"target":"TARGET","loader":"LOADER","root":123}`.

Absence is legacy unknown; explicit null, missing owned members, wrong types, empty strings, malformed Unicode, duplicate keys and nonpositive/noninteger/overflow roots refuse. Require target to equal segment tab ID. Preserve legacy acceptance for unrelated existing fields rather than tightening the whole recording schema accidentally. Target and loader are opaque: no trimming, case folding or invented syntax.

Route segments through a dedicated codec using strictjson recursive split/encode and raw extension storage at segment, tab, viewport and load depths. Preserve arbitrary valid unknown JSON numeric tokens without float64 conversion. Exact owned discriminator names must not be selected through encoding/json's case-insensitive aliases. Encoding a typed edit preserves unknown raw values and never mutates shared records. Keep existing sketch decode/admission and canonical bytes unchanged.

There is no existing individual CDP identifier limit: CDP's entire message is capped at 16 MiB, while recording codec/writer have no generic line cap. Do not borrow SketchMaxIDBytes=128 or add a lower identity cap. Root uses positive int64 on disk; project it as an exact opaque string to JavaScript rather than rounding through Number. This corrects the joint plan's ungrounded reference to bounded individual CDP IDs without adding a new policy.

Catch-up retains complete segment records, including inactive creation segments, before dependent feedback and the sketch semantic snapshot. Do not carry identity through sketch clones: those intentionally clear their original wire extensions. Compare full replay, every-prefix catch-up and repeated compaction for both membership and historical metadata.

Pin additive evolution with a frozen preserving older segment schema: it decodes page_load as raw unknown, edits an owned legacy field, rewrites, then the current reader recovers exact identity and nested extensions. Already-shipped permissive older readers may drop unknown fields; document that limit, do not claim retroactive preservation. Existing historic recording lines are not migrated.

## Recorder ordering and Board ownership

`document()` currently performs synchronous CDP IO and mutates loader/snapshot sequence before acceptance. Return local `{root, loader, frameID, replySeq}` evidence instead. Frame-tree failure cannot reuse an old loader as authority. Only a matching ingress generation may install loader/frame/snapshot caches, mirror state, segment metadata and confirmed scope, after accepted document writes.

Selected top-frame navigation, target activation/change and detach invalidate the token and publish Board pending scope at ingress, before snapshot IO or queue delay can expose stale page feedback. Classify from synchronized selected-session/top-frame facts and event parent identity, never actor maps. Iframe/OOP-frame events do not replace top scope. Top Page.frameNavigated directly schedules the existing resnapshot path; DOM.documentUpdated shares it. Same-document navigation preserves identity. Duplicate notifications for an already confirmed exact identity do not create a new load or clear its feedback.

Snapshot cancellation is owner-scoped. A superseded completion cannot mutate caches, confirm scope, clear a successor's pending state or mark the successor non-live. Keep execution serialized and cancellation-aware. Snapshot failure, missing root, detach or close leaves scope unavailable, recoverable through the existing explicit capture retry; never restore old scope optimistically.

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
| E CDP events | O(1) classification/token update; one serialized snapshot owner | 8/16 duplicate notifications count snapshot starts, concurrent calls <=1, no duplicate activation for unchanged completed identity |
| S snapshot nodes | Existing one-pass O(S) document processing per admitted snapshot | Existing fake tree counter plus delayed/cancelled IO proves no extra scan or orphan work |
| L activation segments | One O(1) index insertion each, O(L) retained metadata | 32/64 segments count index writes, full/catch-up parity |
| H feedback items | O(H) projection with one O(1) membership lookup each | 32/64 mixed notes/sketches count lookup visits, reject nested segment scans |
| F displayed rows / P painted paths | Existing linear projection/painter; membership checked once per item | Node 32/64 fixture counts visits and clears noncurrent keyed state |

Reuse current queue bounds (100,000 DOM and non-DOM events), CDP call timeout (5 seconds), CDP frame limit and recording retention; do not change production deadlines. Identity metadata adds once per existing segment, no separate disk index or service. Mounted UI caches discard departed keys; hover and late replies fence by exact scope/revision. Tests use disposable roots/profiles, cancellation-aware fake gates, and joined owned cleanup.

## Implementation tasks

### 1. Recording identity and replay

Files: create `recording/page_load.go`, `recording/page_load_test.go`, `recording/segment_codec.go`, `recording/segment_codec_test.go`; modify `recording/record.go`, `recording/codec.go`, `recording/reduce.go`, `recording/catchup.go`, `recording/contract.go` and their tests.

- [ ] Write failing PageLoad equality/validation and segment codec tests. Exercise absent/null/malformed/duplicate Unicode/number fields, exact target match, opaque unknown nested numbers, typed edits, shared encoding and legacy acceptance. Run `go test ./cmd/internal/webcast/recording -run 'Test(PageLoad|Segment)'` and inspect intended red failures.
- [ ] Implement optional segment metadata and strict preserving segment codec. Add segment index and pure membership helpers without changing sketch request/digest fields.
- [ ] Add frozen older-schema rewrite, every-prefix catch-up and repeated-compaction tests with inactive loads and immutable sketch digest/receipt controls. Run `go test ./cmd/internal/webcast/recording`; require PASS.
- [ ] Commit the model/codec checkpoint with issue and author trailers.

### 2. Recorder ingress fence and Board scope

Files: create `recorder/page_load.go`, `recorder/page_load_test.go`, `board/page_load.go`, `board/page_load_test.go`; modify `recorder/recorder.go`, `recorder/mirror.go`, `recorder/recorder_test.go`, `board/board.go`, `board/submission.go`, `board/operator_view.go`, `board/sketch_delivery.go` and focused tests.

- [ ] Add failing stateful fake-CDP gates at getFrameTree/getDocument with explicit entry/release barriers and cancellation join. Pin pending before IO, navigation during held IO, stale completion, failure, detach, close, iframe and same-document cases; retain established event-order and stale-draft tests.
- [ ] Implement the ingress-owned token and Board generation transition. Keep snapshot return evidence local until generation acceptance; propagate complete identity on each segment including resume/tab-return.
- [ ] Add atomic selection tests for current-only membership, pending empty page selection, unchanged unsaved guard, exact admitted versions after navigation, and same-load return. Add counted 8/16 and 32/64 guards from the cost table.
- [ ] Run `go test ./cmd/internal/webcast/recorder ./cmd/internal/webcast/board` and focused `-race` ordering cases; require PASS and all fake work joined. Commit.

### 3. Live consumers, after #856 lands

Files: `web/model.js`, `web/annotate.js`, `web/sketch-render.js`, `operator.js`, #856's panel/entry/glow adapters as actually landed, `board/feedback_context.go`, `query/querier.go`, `query/sketch.go`; tests in `tests/webcast/*.test.mjs` and colocated Go tests.

- [ ] Merge landed main only after #856's landing; reconcile the actual entity table/file ownership before editing. Do not import its unlanded commits.
- [ ] Add failing reducer/operator parity tests: same URL distinct tabs, reload, same-document app update, A→B→A, pause/resume, unknown scope, held editor and delayed hover/anchor completion. Assert no old rows, pins, ink, labels or controls survive scope departure.
- [ ] Filter all live feedback from the same derived current-load source, preserving selection/delivery/privacy stamps and #849's getter/unsaved guard. Clear transient hover/editor state through existing lifecycle owners; no new feedback store.
- [ ] Add unfiltered query provenance assertions and exact root string projection. Run `node --test tests/webcast/*.test.mjs` and `go test ./cmd/internal/webcast/query ./cmd/internal/webcast/board`; require PASS. Commit.

### 4. Native qualification and landing

- [ ] Extend existing disposable native fixtures with same-URL reload without a later user DOM edit, same-load resume/tab return, app-only update and new same-URL target. Assert current UI/press selection and retained query history independently; use actual navigation events and completion barriers, no sleep as ordering proof.
- [ ] Run the existing named native tests with `WEBCAST_LIVE_CHROME=1`, a watchdog covering scenario plus joined cleanup, and retain complete non-skipped source-bound output. Preserve a synthetic regression for the operator's new-tab observation, never private screenshots.
- [ ] Update `atlas/webcast-submission.md` and relevant recording/recorder map/index for load versus activation, pending/unknown behavior, query provenance and old-reader limits. Reconcile concepts table against actual ownership.
- [ ] Stage new tests, run `make test-affected-inventory`, audit every named reader individually, push and run `sdlc local-check run --pre-close`. Close through the mandatory fresh SDLC review; fix blockers on newly verified source.
- [ ] Ask hub to sequence the morning combined stage and operator steps. After acceptance and checks, enter `sdlc merge --yes` promptly; report actual landed PR/train, clear any owned stage through SDLC and finish cleanup.

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
