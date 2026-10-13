---
id: 000483
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 3.82
card_mirror: '33b99eb2bbf0bf6fe4261fed9fc4d7356ddafd05' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T08:41:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# broadcast-web: DOM submission — send live notes with focused context and a crop to the agent in view

## Problem

Wave 3 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`, Direction). Notes are comments in the DOM's time and space, the counterpart of 🤖[] markers in a markdown file; the operator wants them submitted to the agent in view as a "DOM submission", the integrated form of typing a message and attaching a screenshot. Everything it needs has landed: notes in the viewer (#470), the notes model with focused context and structure-snapped crops (#461, #469), agent queries (#462), replay links (#471, #477 in progress).

## Spec

### Current contract — 2026-10-12, plan proposed in pair:6

The operator's Log and the revisions below supersede the initial proposal retained afterward.
Shift+Ctrl+Alt+Return from the draft nvim submits the draft and all open notes on the cast tab at admission;
an empty draft works when notes exist. One versioned shared Go submission type is persisted as
`<couch store>/webcast/submissions/<id>/submission.json` plus one `crop-N.png` per note, including overlapping
notes. The agent receives the draft plus a short absolute manifest pointer; no clipboard or rendered-copy file.
Terminal submission (#455), ticket creation and ticket assets are separate work.

Each note includes recorded author/text/time/anchor, focused context and compact element text, replay link and
recording path. PNGs come from the cast tab now and carry capture time and document identity. A document
already gone yields explicit `crop unavailable: page changed` with the recorded context retained; navigation
during capture refuses preparation. Selection is frozen before asynchronous work, so later notes are not consumed.

Prepare the bundle, reuse the authored-draft send transaction, then acknowledge delivery to resolve exactly those
notes. Retry freezes bundle, recipient and generated agent text. A failed or uncertain send preserves the draft
and notes; a delivered message whose acknowledgment fails retries acknowledgment only, never delivery.
The durable plan below proposes concrete resource limits and store retention for TL review before building.

### Initial proposal (superseded where the operator's Log or current contract differs)

Operator-shaped (the operator shapes it in the slot); the TL's starting proposal, from the project:
- **Ways in:** a button in the notes menu (`💭N`), a hotkey, and a combined submission from the draft nvim (Shift+Ctrl+Alt+Return: the draft's text plus the pending DOM notes).
- **What goes:** per note, the text, author and time; the element and its focused context (`notes.Model.Context`); a screenshot crop of the notes' cluster, snapped to structure (`NoteCrops`, captured from the operator's tab with `Page.captureScreenshot` and a clip: pixels to the agent only); a link to the moment in replay (`#t=`); and the recording's path for `couch --broadcast-web-query`. Notes far apart become one crop per cluster.
- **To whom:** the agent in view in Couch (the slot focused), delivered like a Couch message; submitted notes resolve on every screen.
- **The wire shape:** `notes.Context` gets a JSON shape here; keep it shared with pair#455 (terminal 💭N submission) so both submissions look alike to an agent.
- **Mock loop:** the agent revises, the page reloads, the cast carries on; live notes clear on reload.

## Done when

- 2026-10-12 acceptance-order revision (TL receipt 0146066c-f885-4a42-a2d6-eb731bd9e8fe): run the authorized low-parallel full suite and one bounded headless Chrome crop check, then close. Operator smoke on pair:0 follows #495/#506 when TL schedules it. This supersedes the older before-close ordering below, not the outstanding live acceptance itself.

- Current acceptance: two viewers leave notes on the blog; the single draft hotkey sends all open notes and
  optional draft text to the agent in view, with one PNG per note and readable recorded context; the agent fixes
  at least one correctly from the bundle alone. Empty draft, safe failure/retry, successful resolution on every
  screen, page-change crop behavior and cast continuation are verified. Operator smoke before close remains required.
- The older button/hotkey/combined wording below is superseded by the operator's single-hotkey decision.
- On the blog, two viewers leave notes; the operator submits them to an agent slot (blog:0) by button, by hotkey and combined with a draft; the agent fixes at least one correctly from the submission alone, without asking for the full page.
- The submission format is documented in the atlas, shared with #455's.
- The operator smoke-tested it before close.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.
Calibration is stale per estimate-source; treat the derived 3.82 focused ship-hours as provisional.
Schema/builder reuse existing notes/query types (smaller module, raw design .2 x .2, raw impl .5 x .4).
Store uses standard-library JSON/filesystem/PNG (greenfield design 1.5 halved for libraries x .2; impl .8 x .4).
Capture is a CDP integration (design 2 x .2; impl 1.5 x .4), plus real-API discovery (.6 x .4 implementation).
RPC mirrors existing Couch adapters (design .3 x .2; impl .5 x .4). Draft transaction is one Lua feature
(design 2 x .2; impl 1.5 x .4). Docs use .1 design/.2 raw implementation; one close review .05 design/.5
raw implementation. Familiarity is 1.0: current Go/Lua stack and existing seams. Thorough-plan buffer is 15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.20
item: greenfield-go-module design=0.15 impl=0.32
item: api-integration design=0.40 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: lua-neovim design=0.40 impl=0.60
item: atlas-docs design=0.10 impl=0.08
item: milestone-review design=0.05 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
design-buffer: 0.15
total: 3.82
```

## Plan

- [x] Shared submission format and pure builder/transition tests.
- [x] Bounded atomic store, note snapshot, current-tab crops and guarded resolution.
- [x] Couch prepare/status/ack RPC with fixed recipient and replay links.
- [x] Draft hotkey using the existing authored-send transaction with frozen retry identity.
- [x] Docs and bounded automated verification; record the separately scheduled smoke and shipment handoff below.

Detailed plan: [DOM submission implementation plan](../plans/000483-dom-submission-plan.md).

## Shipping acceptance pending

Operator smoke on pair:0 follows #495/#506, scheduled by the hub. Do not stage or merge before that smoke. After smoke/SHIP, merge current main (integrating #494 capture suppression if it has landed), run required gates and sdlc merge --yes, report to hub and stop. Local close is explicitly authorized before this acceptance by TL receipts 0146066c-f885-4a42-a2d6-eb731bd9e8fe and ad2abe0b-d61c-4c7d-bde8-e238d0f50971.

## Log

### 2026-10-12 — plan approval in pair:6

TL approved commit `ec88640dd` and the proposed operating limits/retention (Couch receipt
`0a3761ed-b2f7-4044-8a01-fb16fabd915f`), directing change-code. Resolve submitted notes after delivery as planned;
TL is confirming that policy with the operator. Architecture/API exploration found that draft retries compare
authored text only; the implementation must freeze the generated bundle pointer too (ARCH-ORDER).

### 2026-10-12

Operator shaping (in pane):
- **Images go as PNG files**, not the clipboard: with many slots running at once, a shared clipboard gets fought over.
- **All open notes are submitted**; the operator resolves the unwanted ones first.
- **One hotkey:** Shift+Ctrl+Alt+Return from the draft nvim sends the draft and the notes. An empty draft still submits when notes exist, because the notes are content too.
- **Target:** the agent in view by default. The draft's text says what to do with it, e.g. "this demonstrates XXX#NNN, update that ticket", and the agent files a new ticket or amends one. Tickets are the durable home, publishable and workable from any slot.
- **Ticket assets (operator idea):** an issue gets a per-ticket asset folder that shares its lifecycle, as parley.nvim's chats do (`assets/<chat-timestamp>/`, keyed by identity not slug; moves and deletes follow the chat).
- pair#488 decision (operator via TL): every note carries a bounding box (drag, or a default 1/10 × 1/10 box centred on a click); the crop is that box plus a small margin. No structure snapping, no clustering in #483's crop.
- **Format (operator):** JSON is the agent-facing shape (agents read it fine). One `submission.json` (shared Go type with #455) plus `crop-N.png`, no rendered text copy.
- **Store first, tickets second (operator):** a submission is written to `<couch store>/webcast/submissions/<id>/` (an absolute path that survives slot switches); the Couch message carries the draft text plus a short pointer to it. Filing or amending a ticket is the agent's step two, not #483's. Ticket assets filed as ariadne#337 (details local, unpublished); not a dependency.
- **Scope (operator):** a web submission takes only the cast tab's open notes; the terminal cast is #455's.
- **#488 seam (TL, from pair:3, #488 M1 on its branch):** `NoteCrops(st, notes)` with no hints; `Crops` and `notes.Hints` are gone. Each note crops its region + 16 px; overlapping crops merge; clipped to the page; every new note has a region. `NoteAt` gains `kind` (element) and `path` (container chain), the shape the terminal anchor will use. Build against it; merge main once #488 lands.
- **pair#493 (TL):** notes and strokes aren't written into recordings yet (queued for pair:6); live notes work. Until #493 lands, take the submission's notes from the live Board, and test against a fixture with notes or the live Board, not a fresh recording.

### 2026-10-12 — capture and persistence checkpoint

Implemented the portable shared submission schema/store, historical per-note builder, frozen Board selection with revision-guarded resolution, and current CDP crop capture. A CDP-to-recorder barrier processes navigation evidence before accepting screenshots (including same-URL reload identity). Targeted low-parallel tests passed for cdp, submission, board, recorder, historical builder and Cast, covering overlapping notes, page changes, bounded input and failed storage. Console/RPC and draft integration remain in progress; no live operator state or GUI was touched.

### 2026-10-12 — submission transport and draft checkpoint

The local CLI/RPC now verifies the calling incarnation for prepare/status/begin/ack. Console owns one bounded background job and durably records begin before delivery, dispatch before resolution, completion only after observed note-state writes. Draft retries freeze body, generated path and pane; direct pane actions do not change focus. Lost acknowledgments retry resolution only and preserve any next draft. Headless tests exercised both mapping modes, empty/comment-only notes, changed draft, failed preparation and lost acknowledgment; ordinary queue and bang-tag behavior also passed.

Review during implementation caught three order defects: a possibly-inline scheduler under the owner mutex, canceled jobs beginning side effects, and attachment equality ignoring generated text. Tests now defend each. Metadata-only store reads avoid decoding every retained PNG on status/ack; full evidence validation remains at publication/full Read. Capture now bounds cumulative PNG memory while collecting, and refuses a paused current document rather than labeling it page-changed.

TL authorized a serial full suite with Go parallelism 1 / GOMAXPROCS 2 and one bounded headless Chrome crop conformance run (receipt 0146066c-f885-4a42-a2d6-eb731bd9e8fe). Full suite is running; operator smoke on pair:0 is deferred until TL schedules it after #495/#506. No staging was performed.

- Coordination delta (#494; receipts d337fc0e-dbd8-4265-9a0d-fe57d61aff3d and a0310d7c-6814-4d28-b5dd-b6315cdd7fcf): whichever lands second integrates annotation suppression into centralized Recorder.CaptureCrops. The proposed seam is recorder.CaptureScreenshot(ctx, call, session, contextID, params, result); pass the exact admitted top-frame couchcast context, never consume bytes on error. Check main before landing.


### 2026-10-12 — conformance and full-suite findings

One authorized headless Chrome run passed: TestLiveSubmissionCrop (4.72 s) with a temporary profile, DPR 2, two overlapping clips below the initial viewport, scroll 1100 CSS px, and current red pixels after an initially blue recording. PNG sizes and page-coordinate centers matched. Headless DOM draft integration now also verifies queue-item consumption while preserving the parked star draft.

The full make test shell/Lua stages passed. Its Go stage found the newly introduced sources absent from the exhaustive artifact inventory; accurate Couch-namespace and generated-mirror entries were added, and TestProductionArtifactReferencesAreExactlyClassified then passed (1.968 s). Remaining Go packages are still running. Additional writer/recorder interleaving tests prove a resolve/reopen between admission and emission is preserved and the local revision guard never enters JSON recordings.

### 2026-10-12 — local verification complete, baseline failure disclosed

- Full serial suite: GOMAXPROCS=2 / Go parallelism 1 / make -j1 test, log /tmp/pair483-tests/full-suite.log. Shell/Lua stages passed. Go completed with three failing packages: artifactpath new-source inventory and keyhelp stale generated nvim bundle were repaired and their targeted tests passed; the remaining unchanged wrapcmd TestNotificationBrokerBeforeExecAndCleanup loses the startup notification.
- Baseline proof: the wrapper test also fails alone with scrubbed Pair/Couch/Zellij identity, and fails with all 25 changed production Go files replaced/deleted through an overlay restoring origin/main. No wrapper source differs from main. TL filed pair#519 and explicitly authorized close with this pre-existing failure disclosed (receipt ad2abe0b-d61c-4c7d-bde8-e238d0f50971). No test or review gate was bypassed.
- Current feature verification: targeted submission/CLI/Console/store suites pass; Board/recorder/codec revision-race tests pass; DOM adapter/transport/transaction and headless integration pass, including queue preservation and ack-only recovery. Ordinary queue and bang-tag tests pass. TestLiveSubmissionCrop passed on one headless temporary Chrome in 4.72 s.
- Guard mutation checks used Go overlays only: ignoring the manifest write error causes write/sync tests to fail; disabling capture identity comparison causes tab-switch and same-URL reload tests to fail. Original sources then pass the same tests. No working source mutation or operator focus change occurred.
- Current main merged cleanly (docs-only #496 delta); #494 has not landed yet. Its centralized capture helper remains a landing-order follow-up.

## Revisions

- 2026-10-12 — pair:6 resumed after the hub released the old shaping claim. The operator's later decisions now
  have an explicit current contract; initial proposal retained for provenance. #488 and #493 are landed, so
  recorded context can be read mid-cast. Plan proposed in `workshop/plans/000483-dom-submission-plan.md`.
- 2026-10-12 — Hub confirms one crop per note by calling `NoteCrops` per note; no historical replay pixels.
  Already-gone document: preserve recorded context and mark crop unavailable. Navigation during capture:
  refuse preparation. Delivery acknowledgment follows confirmed draft dispatch, with immutable retry identity.

- 2026-10-12 — crop source: the note's box (pair#488), not `NoteCrops` snapping/clustering. If #488 hasn't landed when #483 builds, crop from the note's `region` (always set once #488 lands) through the notes seam #488 fills.

- 2026-10-12 — TL: #488's plan settles the crop as each note's box + 16 px margin; structure-snapped crops and per-URL crop hints are retired. #483 builds its crop on the note's box through the notes seam #488 fills.

- 2026-10-12 — Operator confirms resolve-after-delivery (via TL receipt dde5ab61-cea4-4617-ad8f-6e4a67b32b86): submitting transfers ownership of the notes to the agent; whatever follows (a pensive, an issue, a fix) is the agent's. README and atlas state this ownership transfer explicitly; implementation already matches.

- 2026-10-12 — Local-close order finalized by TL: disclose pre-existing main failure pair#519, close, then wait for scheduled operator smoke before merge. The issue Plan now describes the completed local implementation/verification; outstanding shipment is tracked in Shipping acceptance pending rather than marked complete.

- 2026-10-12 21:19 PDT — Operator additions, confirmed by TL (receipts 8ea3022f-ac70-498d-a0e3-033af2f2414a and f9849280-01be-4cef-86d4-d5a6415ada73): preserve current capture-time box text alongside historical anchor text. Use the original unpadded note region, or anchor box when no region, and visible intersection in reading order; skip display:none, visibility:hidden, zero-size, off-screen and aria-hidden content. Label visible text and accessible names (aria-label, alt, title, role + name) separately. Bound each note to 8 KiB visible text, 8 KiB names, 256 entries and 20k visited nodes, explicitly marking truncation. PNG remains definitive appearance; text preserves characters; selectors locate elements. A changed-away document has unavailable current evidence. Include one canonical how_to_read string in submission.json and delivered agent text: manifest/crop meanings, absolute recording path, couch --broadcast-web-query DIR notes|dom|events|link; start with notes, dom --at <t> --sel @N, events --from <t-10000> --to <t>, and couch --skill for the guide. TL authorizes one additional bounded headless conformance run.
- 2026-10-12 21:19 PDT — Close round 1 returned REWORK. BR-1 requires repeated directory synchronization on post-rename durability retries (Prepare and all delivery transitions). BR-2 reclassifies DOMSubmission as INTEGRATION, with a pure state/event transition authority that declares effects and owns all retry state. BR-3 consolidates pane parsing and pair-wrap identity shared by submission transport and pair_poke. Fix and test these before rerunning close; no merge before scheduled operator smoke.

- 2026-10-12 21:27 PDT — TL accepts bounded box-text scope: top document plus open shadow roots. The explicit scope field names that coverage; intersecting skipped frames or object/embed content set truncated=true and name the skipped kind. Closed shadow internals cannot generally be detected by page JS; this scope excludes them. No separate-DOM CDP traversal in this item; PNG remains the complete visual evidence. Keep one bounded pass and one result shape (receipt 3bc1e4c2-36e8-4ee0-978c-6343943bd5ca; exact message verified, receipt delivery indeterminate although received).

- 2026-10-12 21:28 PDT — Re-entry change-code passed plan-quality round 4 (no open blockers) and estimate-quality INFO. The initial 3.82 h estimate remains the original scope baseline rather than being fitted to elapsed work; added box-text/guide and review rework are explicitly later scope. Hours mean idle-removed ship wall-clock including agent execution, not operator attention. Verification authorization now permits full suites, targeted tests and bounded temporary-profile headless Chrome without further permission (TL receipt 018bddf6-b3db-4a35-80f5-e79d91156ae4); visible windows, pair:0 staging, external machine changes and scope changes still go to hub.

- 2026-10-12 21:35 PDT — BR-1 fixed with repeated post-rename directory-sync failures through reopened Store instances and cross-operation retries; package tests pass. BR-2/BR-3 fixed through pure DOM state/effect model, stale completion epochs and shared pane parser, with unit tests plus all eight headless draft cases and review-controls passing. Canonical reading guide passes actual Build → Store reopen → prepared/status RPC → generated agent-text tests, including the absolute recording path and notes/dom/events/link/skill commands. Current box text passes node fixture and recorder tests for bounds, hidden/offscreen/range clipping, roles, open shadows and context replacement; bounded temporary-profile headless Chrome verifies current text and names beside current PNGs (latest run 1.12s, package 1.325s). Full-suite rerun and close review remain pending.

- 2026-10-12T21:46-07:00 — Revised-scope full suite completed: GOMAXPROCS=2 GOFLAGS=-p=1 make -j1 test, with the temporary go wrapper limiting hard-coded nested -p values to 1. Every shell/Lua stage and every Go package passed except unchanged TestNotificationBrokerBeforeExecAndCleanup in wrapcmd (startup hook missing), the same pre-existing main failure demonstrated by the earlier 25-file baseline overlay and tracked as pair#519. No other failures. Log: /tmp/pair483-tests/full-suite-revised.log. Runtime generated mirrors and artifact inventory checks passed. Latest main merged cleanly (only #519/#520 issue details). Ready to rerun close; operator smoke and merge remain pending on hub schedule.

- 2026-10-12T21:54-07:00 — Close round 2 disposed BR-1/BR-2/BR-3 as addressed, including mutation evidence. New BR-4: visible evidence must use a nonempty effective intersection of original note region, viewport and applicable ancestor clips, for both visible text and accessible-name entries. Fix the class by carrying clip bounds through the bounded traversal (including each overflow axis and composed shadow/slot ancestry), rejecting wholly offscreen/empty regions, and conservatively excluding unsupported clipping rather than asserting hidden content is visible. Add direct collector and headless-browser regressions for overflow clipping, nested scrolling, partially clipped text and offscreen wide elements; document any explicitly truncated unsupported clip. Targeted tests on final collector follow the completed full suite, whose only failure was pre-existing pair#519.

- 2026-10-12T21:56-07:00 — TL approved the final BR-4 scope (a4b90592-c792-41c3-ae93-34c7470c0828): handle rectangular overflow/scroll/paint containment and axis-aligned scaling; clip-path, masks, legacy clip or rotated clipping containers omit the subtree and explicitly mark unsupported clipping plus truncation. PNG remains complete. Do not grow scope after this round; remaining edge cases become a follow-up note.

- 2026-10-12T22:00-07:00 — BR-4 fixed in one composed-tree pass: empty intersections return no evidence; shared clipping covers both groups, independent overflow axes, nested scrollports, scaled client/border bounds, paint containment and slotted content. Conservative unsupported-clipping includes nonrectangular rounded overflow/paint clips and nonzero clip margins, plus the approved transform/clip/mask cases; these are explicit truncation, not claimed visible text. Remaining fidelity edge cases stay in the follow-up note. RED TestBoxTextCollector reproduced clipped text on old code. Final recorder/build/webcast package tests passed (2.411s/0.177s/7.802s); real headless Chrome conformance passed 1.21s, exercising hidden overflow names/text, wholly offscreen regions, nested scroll, partial characters, scaled borders and composed slots. Model validation tests passed, and removing unsupported-clipping admission via Go overlay failed its regression as expected. This targeted post-fix verification follows the completed full suite at 14d427d7 (only baseline pair#519 failed); no unrelated source changed.

## Follow-up note — broader visual text fidelity

After the approved clipping correction, consider native control value rendering,
complex occlusion/compositing and out-of-flow descendants escaping containing-block
clips as a separate fidelity study. Do not expand #483 into a full browser paint
model; the current PNG is the complete visual evidence and explicit truncation
records the supported collector boundary.
