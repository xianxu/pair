# Anchored notes and draw-time captures Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent tasks or superpowers-executing-plans for work needing session context. Steps use checkboxes for tracking.

**Goal:** Web-cast notes follow their elements, retain their original captured evidence when hidden, and submit every open note without taking new screenshots.

**Architecture:** Separate immutable note evidence from its current visual placement. One cast-owned capture-draft service serves both viewer and operator adapters, reusing the existing settled capture machinery outside the recorder event loop. The submission store becomes the durable owner of published captures; a note retains a stable reference for subsequent sends.

**Tech Stack:** Go, recorded DOM reduction, isolated-world CDP, shared browser JavaScript, local private files, Node tests and headless Chrome.

## Scope and approval

Web cast only. Terminal notes and terminal capture behavior stay outside this change. This is one atomic feature with one close review, not several artificial milestone boundaries. Await hub approval before `sdlc change-code --issue 538`; the plan-quality gate owns the formal design review. Estimate follows that gate.

#537 independently adds author-owned text editing. Preserve its latest-note pin lookup, trusted ownership checks, edit-note protocol and original capture when text changes. Merge its landed main changes before final verification; do not stack branches. #528 supplies the evidence-ranking approach, not a dependency on its unrelated session-resume implementation.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| AnchorEvidence and MatchDecision | `cmd/internal/webcast/notes/anchor.go`; `cmd/internal/webcast/web/annotation-anchor.js` | new |
| NoteAt and Note capture reference | `cmd/internal/webcast/recording/record.go` | modified |
| CaptureRecord and draft ownership transitions | `cmd/internal/webcast/notedraft/model.go` | new |
| SelectedNote and Selection | `cmd/internal/webcast/board/submission.go` | modified |
| Submission v3 capture provenance | `cmd/internal/submission/model.go` | modified |

- **AnchorEvidence / MatchDecision:** one immutable evidence set per note, many replacement candidates, zero or one chosen placement. Scoring operates on plain values; DOM discovery and visibility are adapters. The Go resolver and shared JS scorer use the same documented weights and conformance fixtures, rather than unrelated permissive fallbacks. Future evidence fields extend this one contract (ARCH-PURE, ARCH-DRY).
- **Note:** owns one optional opaque capture reference, not a filesystem path. Capture metadata records original document, route, tab, actual captured origin/URL, timestamp, box text, geometry and anchor evidence. Existing terminal and old web records remain readable. Authored text can change independently of immutable captured pixels/text.
- **CaptureRecord:** has one current storage location and ownership: unsaved draft, attached note, or published bundle. Internal save/transfer intents describe incomplete writes; they are not user-visible states. Cancellation is legal only for unsaved ownership. Multiple submission bundles may contain the same immutable evidence.
- **Selection:** all open notes within this cast recording, each with its own original document and current revision. Acknowledgment resolves exact selected revisions once, regardless of current page; a newer edit/reopen is protected.
- **Submission v3:** supports per-note/per-crop document and captured origin. Readers continue accepting v1/v2. Aggregate privacy remains private unless every included capture qualifies under current origin policy.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| NoteDrafts service and private Store | `cmd/internal/webcast/notedraft/service.go`, `store.go` | new | capture, clock, private disk and acknowledged annotation writer |
| Viewer prepare/save/cancel adapter | `cmd/internal/webcast/server_board.go`, `board/wire.go` | modified | authenticated annotation capability and HTTP |
| Operator prepare/save/cancel adapter | `cmd/internal/webcast/recorder/operator.go`, `operator.js` | modified | trusted isolated-world context and recorder actor |
| Placement adapters and shared annotation UI | `cmd/internal/webcast/web/render.js`, `web/annotate.js`, `operator.js` | modified | current DOM, visibility, gestures and thoughts list |
| Stored capture submission source | `cmd/internal/webcast/cast_submission.go`, `submission/build.go` | modified | private capture reference resolution and durable bundle publication |
| Capture engine | `cmd/internal/webcast/recorder/submission.go`, `capture_ready.go`, `ui_capture.go` | modified | existing CDP capture identity, settling and overlay suppression |

The cast constructs NoteDrafts once and injects it into Server and recorder options without an import cycle. Its capture dependency is implemented by the existing engine and a stateful fake that tracks document, pixels/text, pending completion and cancellation. Its writer fake stores acknowledged notes and can return pending/unknown outcomes. The filesystem tests use real temporary directories, injected clock and controlled crash points; no function-call-only mocks. These seams can later support other annotation surfaces without another lifecycle owner.

## Placement contract

1. Store bounded tag, normalized classes, stable ID/test attribute, existing structural path and normalized text (existing 200-character anchor bound), plus original anchor box and route identity. Route identity includes origin, path, query and fragment; it is local evidence, never automatically published as issue prose. A different current route hides the box until the original view returns.
2. Try the connected original node, but reject conflicting tag/stable identity or changed identifying text: virtualized node reuse must not steal a note. Otherwise rank replacement candidates within the compatible document/frame and route scope. Do not choose the first selector/text match.
3. Initial scoring contract for fixtures: tag is mandatory; matching stable identity +50, meaningful exact normalized text +40, structural ancestor suffix +20, class overlap up to +10. Conflicting stable identity or meaningful text disqualifies. Require at least 60, a 20-point lead, and either stable identity or meaningful text. Positional paths/classes/common text alone cannot authorize a replacement. Equal winners hide. Calibrate only against explicit positive/negative fixtures and record any changed weights.
4. Bound candidate enumeration to 2,000 nodes per scope; a truncated search cannot declare a unique winner. Cache against DOM/layout revision, invalidating on relevant text/attribute/subtree changes, route/document changes, scroll and resize. Keep disconnected documents out of caches.
5. Recheck connectivity, rendered size, hidden ancestors, opacity/visibility, clipping and viewport/frame intersection before painting. Nonvisible or ambiguous placement returns no rectangle. Both adapters include shadow/frame coordinate mapping; unsupported boundaries hide rather than guessing.
6. Project the original region's normalized offsets within the original anchor box onto the current anchor box, scaling with it. Missing/invalid anchor geometry hides the box. Remove historical absolute-region fallbacks from live placement. Historical replay remains explicitly historical.
7. Both surfaces show the same open-note list. Hidden/removed/ambiguous notes say “not on this view”; they remain editable/resolvable using the same current-note actions and stay in submission selection. Visibility never emits resolve or AnnotationClear. Existing explicit clear still means deletion and cleans its note-owned captures.

A heuristic cannot prove semantic identity in arbitrary pages. The enforceable safety contract is no placement without adequate nonconflicting evidence and a unique winner; uncertain cases stay in the list.

## Capture and visible behavior

Mouseup immediately starts capture and opens the draft with “capturing…”; authored text remains editable, Save waits for a ready capture, Cancel is always available. The tick appears only after confirmed capture, with the actual capture timestamp. Settling means the image is taken shortly after mouseup, not necessarily on that exact animation frame. Pixels and box text belong to the same validated capture window, before Save and independent of Submit.

Ready has one primary action, Save. A failure displays its reason and one primary action, Retry capture; it never freezes authored text. Retry explicitly takes a new shot. Successful Save attaches the prepared capture without resolving the anchor again. Cancel deletes the unsaved draft. No new discard key or confirmation dialog.

Use opaque random draft IDs scoped to cast and admitted capability; public Viewer.ID is attribution, not ownership proof. Prepare carries a request ID and pinned box/node/document/view identity. Validate finite geometry and admission at entry and completion. The server returns ID/status/time only, not PNG or extracted page text to a remote viewer. Save carries draft ID plus text/color and is idempotent by request ID. Cancel by draft or prepare request ID is idempotent, including when it precedes capture completion.

Keep prepare/status transport requests short: accepted prepare returns pending; clients reconcile status under an absolute 36-second operation budget, with the existing 30-second capture limit and bounded requests. Do not extend the operator's five-second RPC into an unbounded wait. One capture worker per cast, at most eight queued prepares; queue time consumes the same budget. Repeated request IDs join the existing operation. Cancelled/stale jobs cannot attach and their eventual output is deleted.

**Critical actor rule:** never synchronously call CaptureCrops from recorder.handleOperator: capture re-enters the recorder through barriers/r.ask. Resolve trusted path/context in the actor, execute capture outside it, then post completion and revalidate document, context and annotation generation. Do not hold board dispatch locks over capture IO.

## Storage, expiry and transfer

- Store under local Couch webcast capture storage, outside recording frames/assets and all public/replay routes. Directories 0700, files 0600; opaque IDs, validated bounded metadata and PNG, no caller-controlled paths or symlink traversal.
- Reuse 4 MP and 8 MiB per-crop bounds. Proposed private-store budget is 256 MiB and 128 capture records; reserve space before capture, count pending reservations, and refuse new captures with an actionable storage reason rather than evict attached evidence. Existing 64 MiB bundle and 1 GiB submission-store limits remain. Over-budget all-note submission fails as a whole, never silently drops notes.
- Unsaved TTL is ten minutes from capture completion, also bounded during preparation. Viewer disconnection/capability loss, operator context retirement and cast end cancel unsaved work; TTL is the crash/disconnect fallback. Cast startup sweeps orphan unsaved drafts after reconciling save intents. Saved open-note evidence survives cast end/restart and follows recording retention, not draft TTL. Published bundle evidence follows existing submission retention.
- Save reserves a note ID and persists attachment intent, then emits the note through the acknowledged writer, then finalizes ownership. A lost reply is reconciled against actual recorded note identity. Metadata alone never proves a saved note. Existing recording JSONL lacks per-record fsync: recovery checks what really survived and does not promise stronger durability.
- Submission obtains a stable claim over selected references, copies bytes into bundle staging, and uses existing Store.Prepare publication. Only after publication succeeds does it durably rebind the capture reference to the exact bundle/crop and unlink the former source. Crashes before rebind may leave duplicate bytes; they must never leave a missing only copy. Recovery reconciles bundle manifests and transfer intents before cleanup.
- Resolve-before-submit and explicit clear remove note-owned source captures. They serialize with attachment/transfer so cleanup cannot delete a crop being copied. Cancellation after a successful/lost-reply Save cannot delete attached evidence. Resolution after publication never deletes a bundle.
- Failed/uncertain dispatch preserves references and bundle evidence. A fresh submission may reuse original bytes from a prior published bundle. #524's new-submission/supersession behavior remains. Acknowledgment resolves selected notes idempotently, including earlier documents, without claiming unconfirmed delivery.

## Submission and compatibility

Select all open notes across recorded documents in this cast, not model.Live/current DocSeg. Build historical anchor context at each note's original document/time and use latest authored text/revision. No screenshot, page settling, current-page availability or current-route requirement remains on the submit path.

Each crop supplies its stored actual origin to the current #526 policy at submission time. A policy change after drawing takes effect; missing/invalid provenance stays private. Mixed-origin bundles combine restrictively. No policy edits or new sharing grants occur. Draw-time bytes are never served publicly merely because a note references them.

Old web notes lacking a capture are included with explicit “not captured when drawn” evidence; never silently substitute a new screenshot. Missing/corrupt stored captures yield an explicit unavailable reason and retain the authored note. Existing v1/v2 bundles remain readable and immutable. Update the reading guide to distinguish draw-time page text/pixels from authored text and historical anchor context. Terminal paths retain their existing formats and behavior.

## Implementation tasks

### 1. Anchor model and conservative placement

Files: new `notes/anchor.go`, `notes/anchor_test.go`, `web/annotation-anchor.js`, `tests/webcast/annotation-anchor.test.mjs`, shared fixture `notes/testdata/anchor-candidates.json`; modify `notes/structural.go`, `notes/structural_test.go`, `recording/record.go`, `web/render.js`, `operator.js`, `web/annotate.js`, `web/annotate.css`, `operator_script.go`, `server.go` and their existing tests.

- [ ] Write failing shared fixtures for stable rerender, reordered siblings, duplicated text, conflicting reused ID/node, changed route, weak evidence, ties and candidate exhaustion. Run Go notes tests and Node anchor tests; verify the old permissive resolver fails negative cases.
- [ ] Implement pure matching and optional bounded record fields, then adapters/relative projection. Add both-surface list coverage and live missing-anchor regression. Pass targeted tests, inspect diff and commit.

### 2. Private capture draft lifecycle

Files: new `cmd/internal/webcast/notedraft/{model,service,store}.go` and colocated `_test.go`; modify `cast_submission.go`, recorder options, `recorder/operator.go`, `recorder/submission.go`, `board/operator.go`, `board/wire.go`, `server_board.go`, and server/cast construction sites.

- [ ] Write lifecycle tests using stateful capture/writer fakes: pending/ready/cancel, TTL/disconnect/end, duplicate prepare/save, unknown writer result, late completion after generation revoke, limits and caller isolation. Verify failures before implementation.
- [ ] Implement one owner, private atomic storage, reservation/attachment intents and recovery. Add tests at every save write boundary; never re-enter recorder capture synchronously. Run targeted Go tests, including race checks for cancel/save and resolve/transfer; commit.

### 3. Viewer and operator two-phase UI

Files: `web/annotate.js`, `operator.js`, `cmd/internal/broadcast/web/collab.js`, `server_board.go`, `board/wire.go`; tests `tests/webcast/annotate.test.mjs`, `operator.test.mjs`, `operator-scope.test.mjs`, `recorder/operator_test.go`, `server_board_test.go`.

- [ ] Add failing tests for immediate pending status, confirmed tick, disabled Save until ready, editable text on any failure, retry, cancel-before-reply and dialog closure. Preserve terminal tests for shared dialog changes.
- [ ] Wire both adapters to the service. Increment gesture generation at each new draw; actively cancel abandoned request IDs instead of merely dropping replies. Add an optional discard callback carrying the former draft payload, distinct from success close and #537 edit mode. Run tests and commit.

### 4. All-note bundle transfer and acknowledgment

Files: `board/submission.go`, `board/operator_view.go`, `recording/reduce.go`, `webcast/submission/build.go`, `cast_submission.go`, `cmd/internal/submission/model.go`, `store.go`, `privacy.go`, and colocated tests.

- [ ] Write failing tests for hidden/earlier-document selection, original pixels after mutation/navigation, zero capture calls at submit, mixed origins/current policy, old formats, legacy missing capture, over-budget whole-selection failure, duplicate acknowledgments and edited/reopened revision protection.
- [ ] Implement v3 provenance, claim/copy/publish/rebind/unlink and document-independent conditional resolution using acknowledged board ordering. Fault every durable transfer step and test retry from a prior bundle. Pass targeted/race tests and commit.

### 5. Browser conformance, performance and documentation

Files: `live_submission_test.go`, `operator_live_test.go`, `viewer_annotate_live_test.go`; `atlas/webcast-submission.md`, `atlas/webcast-format.md`, `atlas/index.md`, `README.md`; current Couch submission reading guidance at its owning source. Update feature inventory if new source files require classification.

- [ ] Extend a local real-Chrome fixture with fade-in cards, app tabs, collapsed panel, SPA routes, replacement nodes and duplicate candidates. Exercise viewer and operator. Confirm overlays absent from PNGs and stored pixels/text unchanged after page mutation; cancel/resolve removes only appropriate local source files.
- [ ] Measure draw prepare/capture/settle/store and submit bundle/paste independently on static and animated fixtures. Log numbers; assert zero submit capture/settle calls instead of a brittle shared-machine wall-time threshold.
- [ ] Run targeted commands below, then full `make -k test` in background and inspect every target. Record inherited failures only with main evidence and owner; never let an early timing check hide Go tests. Run `make build` and documentation/format checks.
- [ ] Merge main and reconcile #537 edits without replacing capture evidence; rerun affected tests. Close through SDLC with real evidence and automatic fresh review, fix findings, publish PR, wait for CI, land on SHIP, report exact hashes/PR to hub. Stage operator smoke only if requested; do not occupy pair:0 during planning.

## Verification commands and expected results

- `go test ./cmd/internal/webcast/notes ./cmd/internal/webcast/recording ./cmd/internal/webcast/board ./cmd/internal/webcast/notedraft ./cmd/internal/webcast/submission ./cmd/internal/submission -count=1` — all pass after relevant tasks.
- `node --test tests/webcast/*.test.mjs` — both browser adapters and pure fixtures pass.
- `go test ./cmd/internal/webcast/recorder ./cmd/internal/webcast -count=1` — existing and new integration tests pass; live tests explicitly skipped only in this non-live run.
- `go test -race ./cmd/internal/webcast/notedraft ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder -count=1` — no data races.
- `WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run 'TestLive.*(Submission|Note)|Test.*Operator|Test.*ViewerAnnotate' -count=1 -timeout 180s -v` — record actual matched test names and Chrome assertions; include new conformance test under this pattern.
- `make -k test`, `make build`, `git diff --check` — inspect all failures; no unsupported passing claim.

## Revisions

- 2026-10-13: Initial durable plan after code exploration. The published skeleton spec now has concrete capture ownership, conservative matching, cross-document selection and test seams. No implementation or estimate yet. The implementation deliberately shares lifecycle ownership (ARCH-DRY) and keeps ranking/transitions pure (ARCH-PURE).

### 2026-10-13 — approved bound-hit projection

Hub approved fbbbcbbfe63fbabf4008a0fcbf333c0114c4c47a (receipt 09d22383-c733-4989-95df-bf202423d0c0) with this override to the initial pending/failure and storage paragraphs: **a full capture queue or store never blocks saving a note**. At mouseup, the tick position immediately shows “crop unavailable: capture busy” or “capture store full: submit or resolve notes to free space”. The primary action remains **Save note**; the latter message also names the actions that recover capacity. Create a bounded metadata-only preparation with explicit unavailable evidence, attach that evidence on Save, and include the reason in the bundle. It does not reserve crop bytes or evict attached captures. At the capture-record count limit, use a short-lived authenticated preparation token carrying bounded unavailable metadata rather than creating another persisted capture record; saving records that unavailable evidence directly in the note. Existing annotation admission/rate/open-note bounds still apply. Cancel discards the token; no crop exists to clean up. Tests must fill both queue and byte/count limits, save successfully on viewer and overlay, and assert visible reason plus bundle unavailable evidence. No automatic recapture at submission.

### 2026-10-13 — plan-quality dispositions PQ-1 through PQ-3

**PQ-1 (ARCH-ORDER): explicit authority.** `notedraft.Step(State, Event) (State, []Effect, error)` is the sole pure transition function. `State` is a tagged value: Preparing, Ready, Attaching, Attached, Transferring, Bundled, Retired; optional unavailable evidence is data in Ready/Attached, not another control state. `Event` is a closed tagged union: Begin, CaptureComplete, CaptureFailed, Save, WriteConfirmed, WriteAbsent, Publish, BundleConfirmed, BundleAbsent, Cancel, Expire, Resolve, Reconcile. `Effect` requests capture/cancel, durable intent writes, writer emission, bundle publication/lookup, reference replacement and deletion. Service fields and Store mutation methods are unexported; public adapter methods submit events, never set ownership. Persist the next intent before executing its external effect and feed its confirmed result back as an event. Snapshot/recovery constructors validate state invariants rather than accepting arbitrary adapter states.

| Current state / event | Outcome and evidence |
|---|---|
| Preparing / CaptureComplete | Ready only after admission/generation/identity validation and private persistence; stale completion retires and deletes output. |
| Preparing / queue or store bound | Ready with explicit unavailable reason; Save remains available. |
| Preparing or Ready / Cancel or Expire | Retired, cancel worker and delete only draft-owned files. |
| Ready / Save | Attaching with immutable request/note ID and intent persisted before writer emission. |
| Attaching / WriteConfirmed | Attached only when the recorded note matches reserved ID and capture reference. |
| Attaching / WriteAbsent | Return Ready if admission and TTL allow; otherwise retire. Absence requires writer completion/barrier or complete recovery replay, not a timeout. |
| Attaching / Cancel, Expire or lost reply | Preserve intent and bytes, record abandonment, reconcile writer result. Confirmed write owns evidence despite cancellation; proven absence permits deletion. Never retry as a different note while this outcome is unknown. |
| Attached / Publish | Transferring with submission ID, selected revision and source claim persisted before copying. |
| Transferring / BundleConfirmed | Validate matching durable manifest/crop, atomically rebind then unlink source; become Bundled. |
| Transferring / BundleAbsent | Release claim to Attached only after store reconciliation proves no published bundle. An error from Store.Prepare alone proves neither outcome. |
| Transferring / Resolve | Mark cleanup requested; finish bundle reconciliation first. Published bundle survives, unneeded source/reference can then retire. |
| Attached / Resolve | Retired and delete source after no outstanding publication claim. |
| Bundled / new Publish | Retain old immutable bundle as source through transfer; failed new publication leaves old evidence usable. |
| Bundled / Resolve | Retire note reference only; do not delete any published bundle. |
| Retired / late callback or repeated cancel | Idempotent no-op except delete an unowned late temporary output. |

At an operation's 36-second deadline the UI becomes editable and shows the reason; it never advertises an unconfirmed save. Unknown save displays “save not confirmed; retry Save note to check”, with the same request identity. Reconciliation does bounded lookups/barriers under the remaining operation budget; unresolved outcomes remain durable and are retried on explicit retry or startup, not in an unbounded worker. Cast-owned `Service.Close` cancels its worker context, prevents new work, waits within the capture cancellation bound, and leaves uncertain save/publication intents for recovery. Late callbacks enter Step with their generation and cannot revive retired work.

**PQ-2 (ARCH-PURE): function-level strategies.** These named production functions are the mechanical seams and test targets:

| Function | Adversarial strategy and independently checked invariant |
|---|---|
| `notes.ScoreAnchor` / JS `scoreAnchor` | Shared value fixtures varying text/identity/path/classes independently; conflicting evidence never yields eligibility and score does not depend on enumeration order. |
| `notes.ChooseAnchor` / JS `chooseAnchor` | Permute candidates, duplicate the best one, and truncate enumeration; only a unique above-threshold winner with the required margin may return an ID. |
| JS `projectRegion` | Degenerate/nonfinite original/current geometry and translated/scaled rectangles; result is finite with anchor-relative offsets preserved, or absent. |
| `notedraft.Step` | Enumerate legal/illegal event pairs and bounded adversarial event sequences, including completion/cancel/save/resolve permutations; at most one authoritative owner, no deletion of claimed/attached evidence, no confirmation without matching evidence. Tests call the production Step, not a copied model. |
| `notedraft.Reconcile` | Supply interrupted intent plus independently constructed recorded-note/bundle observations; unknown preserves evidence, proven absence can retire, matching durable evidence chooses its actual owner. |
| `notedraft.Capacity` | Exact byte/count boundaries, reservations, metadata-only notes and repeated transfer/resolve cycles; reservations cannot overcommit and released ownership returns capacity once. |
| `submission.Build` / v3 validation | Permute selected documents and origins, corrupt/miss captures and change policy; every selected ID appears exactly once, source provenance remains original and privacy cannot become less restrictive. |
| Board conditional resolution | Race acknowledgment with edits/reopens/navigation; only the exact selected revision resolves and repeated acknowledgment is idempotent. |

Stateful integration tests additionally cut execution after each durable intent/publication/reference write, reopen actual temporary stores and replay real records through Reconcile. The fake capture source records each snapshot's pixels/text/document so submit tests prove no new capture was requested.

**PQ-3 (ARCH-FUNERAL): capacity and cleanup.** The 128-record capture limit counts preparing reservations, ready crops, attaching intents and attached source crops. Each unresolved transfer counts once, not once per copy. After confirmed publication/rebind, the small note-to-bundle reference is indexed separately and no longer consumes a capture slot or source-byte quota; its metadata is bounded by recording note limits and existing recording byte limits. Completed intents are deleted in the same durable reference transaction (or removed by idempotent recovery if interrupted). Retired records, tombstones and cancellation request dedupe are collected after the ten-minute request lifetime; pending callbacks retain only bounded generation IDs.

A lightweight `Service.Sweep` runs on startup, before capacity admission and once a minute while active. It reconciles unresolved intents first, then removes expired unsaved records and references whose recording directory is proven absent, using the canonical recorded path and a successful filesystem read (permission/IO errors are unknown, not absence). This supplies cleanup for private captures outside a recording even when the recording is removed externally; no existing recording-delete callback is assumed. Cast end cancels unsaved work and stops the ticker; the next owner startup performs the same sweep. Attached captures for an existing recording survive cast end. Resolved/explicitly cleared notes retire references through the acknowledged board event callback; startup replay performs that cleanup if a process died first. Published bundle retention remains exclusively owned by the submission store. If retention removes the referenced bundle, future submission records explicit unavailable evidence and drops the dead reference; it never recreates pixels.

This revision answers the gate's ownership, testing and cleanup classes without changing approved user-visible behavior. #537 clarification: resolved-by metadata is recording/replay/query only; no resolved-list/history/reopen UI in #538. Open-note list remains the sole added list.

### 2026-10-13 — PQ-2 active test contract

The function-level strategy table above **supersedes the prose test-case inventories in Tasks 1–4**; those earlier inventories remain revision history, not a second active test contract. The row labeled “Board conditional resolution” names production `(*Board).ResolveSelection`. UI production seams additionally covered are `captureDraftController.project` (controlled completion/deadline/bound ordering; pending alone may disable Save), `captureDraftController.cancel` (cancel before/after token reply; no stale attachment), and `mountAnnotation` (stateful viewer/operator DOM adapter; hidden notes stay in one open-note list and rendered text never becomes HTML). Each task's first checkbox now means: add failing tests at those named seams using the strategy table, verify failure, implement, then rerun. PQ-1 and PQ-3 were addressed in round 2; this clarifies the remaining documentation issue without changing scope.

### 2026-10-13 — #537 integration and reconnect projection

Merged #537 ownership/editing after landing at 51ab81e9815ea65e819577cc92d504d2d524f1f5. Capture preparations carry trusted public note-owner identity separately from their private draft scope; the authenticated proof is rechecked after request-body reads and at write dispatch. Existing captures survive text edits. Since the all-open list includes earlier documents, its own-note editor admits historical open-note edits under the current document/generation fence, with the existing author check and revision ordering. Reconnect catch-up retains all note state and original segment identity while limiting DOM snapshots/strokes to the current view; otherwise late joiners would lose the promised list. Regression checks cover historical edits, stale admission refusal, capture preservation, and catch-up equivalence at every event prefix.
