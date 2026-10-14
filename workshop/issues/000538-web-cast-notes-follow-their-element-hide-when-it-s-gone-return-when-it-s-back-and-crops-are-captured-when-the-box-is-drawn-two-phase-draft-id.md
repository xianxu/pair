---
id: 000538
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours: 5.48
card_mirror: 'a2f8f4ee751a0ca4edd1f3e48f155344bc6285b7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T16:08:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# web cast: notes follow their element (hide when it's gone, return when it's back) and crops are captured when the box is drawn (two-phase draft id)

## Problem

Web-cast note boxes can remain on unrelated content after their anchor disappears, and submit-time screenshots no longer show what the operator annotated. Submit also spends most animated-page latency waiting for crops. Notes need durable draw-time evidence and conservative element-following visibility.

## Spec

Web cast only; terminal cast has no DOM and is out of scope. On mouse up, a viewer or the operator overlay sends the drawn box to the cast owner. The server captures the crop and box text from the cast tab together, using #524 bounded settling and #494 UI suppression, stores them locally, and returns a draft ID. Saving sends authored text plus that ID and attaches its stored capture. A small capture tick indicates when the shot completed. Submission bundles stored captures instead of taking new screenshots.

A note follows its anchor. If the element is removed, hidden, on another app tab/collapsed panel/route, hide the box and keep the note open in the thoughts list marked “not on this view”. Show it again when a confident visible match returns, including replacement nodes after SPA rerenders. Score structural path, text, tag/classes and identity evidence; ambiguous or weak matches remain hidden. Never use a stale rectangle or arbitrary first match to draw on another element. Prefer evidence-ranked candidates over refusing solely because node identity changed.

Submit includes ALL open notes in the cast, including hidden notes. Current visibility must not discard their stored evidence. Cancellation deletes a capture draft and crop; an unsaved draft expires on viewer loss, cast end or a bounded timeout and is deleted. Resolving a saved note without submitting deletes its crop. Submission transfers the capture into the durable #483 bundle; later disappearance of the element does not delete it. Define crash-safe ownership so a failed or duplicate send cannot lose evidence or falsely claim delivery.

Capture artifacts remain in the local Couch store before submit. Do not return crop bytes to the remote viewer as part of the draft protocol. At submission, #526 private-by-default rules remain authoritative, evaluated using the stored capture's origin. Keep a few visible states with clear actions; internal capture/storage states must not leak into the user flow.

## Done when

- [x] Viewer and operator-overlay boxes disappear on hidden/removed anchors and return only on a confident visible match, with ambiguous/reordered/reused-node cases covered.
- [x] Draw-time capture and text are attached through a two-phase draft ID; capture feedback is visible and stale/cancelled results cannot attach to another note.
- [x] All open notes, including hidden and earlier-view notes, submit their original captures without screenshot/settling work at submit.
- [x] Cancel, unsaved expiry, viewer/cast loss, resolve-before-submit and bundle transfer have tested cleanup and crash/retry behavior; disappearing elements retain evidence.
- [x] Capture files stay local/private before submission, private-by-default submission rules remain intact, and legacy notes have explicit honest fallback behavior.
- [x] Pure scoring/lifecycle tests, controlled server/recorder tests, viewer/overlay browser conformance, affected checks and full-suite evidence are recorded; docs describe the resulting behavior.

## Plan

Durable design: [anchored notes and draw-time captures](../plans/000538-anchored-notes-draw-capture-plan.md). Approved by hub; change-code passed at a478a7b9d.

- [x] Map existing anchors, note recording, capture ownership, privacy and submission seams; write durable plan and obtain approval.
- [x] Implement evidence-scored visibility and draw-time capture lifecycle through viewer and operator paths.
- [x] Bundle stored captures for every open note and verify cleanup, failures, privacy and latency.
- [ ] Review, build, publish and land on SHIP.

## Log

### 2026-10-13
- 2026-10-13: closed — All affected Go packages passed; 342 Node tests, inventory, clean build and five real-Chrome cases passed. BR-1 original-node bridge additionally passes race tests and real Chrome with common Save anchors on a 2100-node page for operator and viewer-authored notes; BR-2 final inventory reconciled. Original crops survive edits and submit performs zero captures. Full suite starved at the gate, ariadne#357; hub authorized this evidence plus required PR CI including macOS conformance. --no-plan-check: only this review and subsequent publish/land row remains unchecked; no claim future gates are complete.; review verdict: SHIP

- Received operator design through ops:0; verified submitted Couch receipt 3db8432f-0b02-44e1-92dd-138eff615d47. #538 already claimed by pair:6. No implementation yet. Initial start-plan refreshed this slot and its ariadne tools, then requested rerun with the refreshed binary.
- Initial inspection: Board.Selection uses model.Live over current DocSeg; submission.Build enforces one selected document and asks the cast owner for new crops. Structural.Resolve accepts a unique selector even without matching text. These seams must change so note evidence survives view changes while projection remains conservative.

- Durable plan completed: shared cast-owned draft service, capture outside the recorder actor, conservative shared anchor fixtures, all-open selection across documents, private storage and crash-safe transfer into submission bundles. Proposed bounds: ten-minute unsaved TTL, one capture worker/eight queued jobs per cast, 256 MiB/128 local capture records; attached evidence is never evicted to make room. ARCH-PURE isolates ranking/transitions; ARCH-DRY gives viewer/operator one lifecycle owner.
- Code exploration found ghost boxes from unconditional note.region/historical at.box fallback and operator boxOf returning null. Current operator prepare creates a token without capturing; calling CaptureCrops inside handleOperator would deadlock its event barriers. Board selection/ack and submission v2 currently assume one document; all require explicit changes. No implementation or test run for #538 yet.
- Coordinated with pair:7 (#537), verified receipt 35db0042-3926-4028-8377-29f59bcc0b67: preserve current-note pin callbacks, author ownership checks, edit-note protocol, and immutable original capture on text edit. They preserve prepare/save/cancel. Reconcile from main before final checks.

- Hub approved plan fbbbcbbf with explicit projection for queue/store bounds: show capture-busy or storage-full reason at mouseup, primary action Save note remains available, and persist honest unavailable evidence without crop bytes. Added authoritative Revisions entry and tests; proceeding to change-code.

- Plan-quality round 1 raised PQ-1 ownership transitions, PQ-2 named function test strategies and PQ-3 reference/intent cleanup. Added a single Step authority, concrete uncertainty transitions, sequence/recovery test seams, quota accounting and sweep hooks in plan Revisions. Baseline Go notes/board/recording packages and 45 targeted Node tests pass. #537 confirmed no resolved-list/history/reopen UI.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only, after plan-quality CLEAN round 3. Calibration is marked stale by estimate-source, so these are provisional ship-wall-clock hours.

Primitive mapping: issue/spec; pure anchor scoring; private capture store/recovery; lifecycle model/service; viewer/operator capture API integration; bundle publication/provenance integration; shared browser dialog/list (TUI-screen interaction analogue); docs; one boundary review. Raw design picks 1.0/1.0/1.5/1.5/2.0/2.0/1.5/0.3/0.1 with thorough-spec ×0.2 except docs 0.06 and review 0.1 direct. Impl raw picks 0.3/0.6/0.8/0.8/1.5/1.5/1.0/0.2/0.5, each scaled ×0.4 once. Familiarity 1.0: current Go/CDP/JS stack and #524/#533 seams are known. Library check: existing capture readiness, private submission publication, board authority and DOM renderer are reused; no library substitutes for cast-specific ownership or scored evidence, so no speculative library discount. Design 2.26 ×1.15 + impl 2.88 = 5.479, rounded 5.48.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.2 impl=0.12
item: greenfield-go-module design=0.2 impl=0.24
item: greenfield-go-module design=0.3 impl=0.32
item: greenfield-go-module design=0.3 impl=0.32
item: api-integration design=0.4 impl=0.6
item: api-integration design=0.4 impl=0.6
item: tui-screen design=0.3 impl=0.4
item: atlas-docs design=0.06 impl=0.08
item: milestone-review design=0.1 impl=0.2
design-buffer: 0.15
total: 5.48
```

- change-code passed at a478a7b9d (plan-quality CLEAN; estimate-quality INFO, 5.48 h). Estimate integration rows cover browser conformance, recovery integration and suite/merge checks; core store/model rows cover isolated primitives. Remaining design allowance covers adapter/authority seams rather than revisiting approved behavior.
- First implementation checkpoint: pure Go/JS scoring and projection, conservative structural resolver (including iframe replacement refusal without frame-route evidence), frozen board preparation/reservation, private atomic capture store and Step ownership model. New service prepares with one worker/eight queued jobs and signed metadata-only unavailable preparations at capacity. Targeted notes/recording/board/notedraft tests pass; JS scorer 19/19; board/store race tests pass. No UI or submission path is complete yet.

- Second implementation checkpoint: recorder acknowledges frozen note reservations outside capture work; HTTP/operator draft adapters enforce generation and owner scopes. Submission v3 carries per-note page/privacy and source document, selects every open note, and resolves by exact revision across documents. Private source-to-bundle transfer/references pass race tests; publication confirmation distinguishes absent/unknown/confirmed. Targeted recorder/board/core tests and all submission-manager tests pass. Browser placement suite passed 191 tests before dialog integration; capture UI, recovery integration and stored-capture submission remain in progress. #537 ownership API receipt 94c7ebc7-def5-4aa6-8843-96a47a8c5c14 verified; production viewer draft routes remain fail-closed until its authenticated context helper lands and is wired.

- Integration checkpoint: browser+terminal JavaScript 284/284; core submission/notes/board/recording/build packages all pass; webcast and couchtty full packages pass. Recorder integration exposed one stale fake expecting ordinary capture failure to be Save-ready; projection now correctly reserves that behavior for approved capacity failures, and the fixture is being updated. Real-Chrome fade oracle passed through draw-time adapter (7.85 s); broader browser conformance/phase measurements remain. Inventory coverage passes. Review found recording IDs can be reused across documents: renderer now grants connected-ID authority only to current-document notes; earlier notes require scored matching. Unknown-save regression now preserves the actual writer error and clears it only after confirmed observation. ARCH-ORDER recovery checks distinguish complete durable bundle, absent directory, and partial/unknown publication.

- Real Chrome `TestLiveDrawCaptureProtocolsAndStoredSubmission` passed (2.66 s, `/tmp/pair538-draw-protocol-live.log`): real viewer HTTP and isolated operator prepare/status/save; operator removes original element before Save; hide/return and duplicate-candidate ambiguity; red page mutation still submits original white PNGs; exactly two draw captures and zero submit captures. Static prepare acknowledgment 19.6 ms, capture context 3.0 ms / settle 3.6 ms / screenshot+text 57.0 ms = 63.6 ms. Animated acknowledgment 26.5 ms, context 0.7 ms / settle 617.4 ms / screenshot+text 33.8 ms = 651.8 ms for a 600 ms CSS animation. Ready including store+poll: 271.8/698.5 ms; viewer Save 513.4 ms includes 500 ms polling, operator Save after removal 54.1 ms. Submit read/claim/build 24.9 ms; publication 32.9 ms; rebind/source cleanup 49.1 ms. Settling dominates animated capture as intended; removing that wait would violate the opaque-crop requirement. Static capture is chiefly screenshot/text overhead. These are local fixture measurements, not a general latency guarantee.

- Stored-capture backend complete in targeted verification: full notedraft race suite passed (40.451 s), including new-cast/different-recording recovery, live-peer exclusion, uncertain publication, corrupt pixels/metadata, and unrelated-record corruption during Save. Malformed source bytes remain preserved and counted; they no longer block unrelated notes, and capacity fallback tokens reveal only the approved action message. Query context now uses one shared context/narration helper at draw time, keeping T/Context/Element/Before coherent even when the element disappears before Save. Focused context races and real Chrome context assertions passed. The older selector conformance test now carries full frozen evidence while invalidating Node; all fixtures still require nonzero successful matches and reject wrong-element matches (21/39, 401/828, 155/825). Selector-only duplicates are explicitly refused. Expanded legacy viewer/operator Chrome fixtures still need service injection; their initial failures are recorded in `/tmp/pair538-browser-conformance.log`, not counted as passing conformance.

- Cross-document projection verification: Board.Live and OperatorView retain all open notes; manual resolve finds historical notes under current-page admission and conditional revision. Per-page color quotas remain unchanged. Targeted Board/Recorder race tests passed (1.318/1.541 s, `/tmp/pair538-history-race.log`), including pause/navigation/revision refusal. Existing Chrome operator/viewer fixtures now use the draft service and passed together (11.72/9.09 s, `/tmp/pair538-live-fixtures-final.log`). The viewer resume fixture had omitted viewport scroll records; restoring its actual scroll makes the retained anchor visible without weakening placement safety. Full `make -k test` started through the machine gate at 17:12 local (`/tmp/pair538-full-suite.log`); no custom load/lock waiting. #537 is in boundary review; final authenticated owner wiring and merge verification remain.

- Final pre-merge Chrome conformance passed all four named tests in 39.094 s (`/tmp/pair538-chrome-final.log`): stored draw protocol, settled submission crop oracle, isolated operator annotation, and two viewers. Board/Recorder full race packages passed (1.268/13.339 s, `/tmp/pair538-board-recorder-full-race.log`). Lifecycle audit found missing viewer-disconnect wiring despite TTL coverage; authenticated owner scopes now count streams and expire unsaved drafts only on the final disconnect, retaining saved/uncertain evidence and invalidating old capacity tokens. Full webcast/notedraft race packages passed (13.312/11.672 s; tool session 94059 exit 0); regression coverage includes multiple tabs, another owner, unknown writes and reconnect. Full suite remains queued; final #537 owner identity integration is still outstanding.

- Merged main at 6788a2dffb4d3383d32fae0b4dbb77f0b5bfe7bb, including #537 landing 51ab81e9815ea65e819577cc92d504d2d524f1f5 (PR #314). Prepared notes now stamp authenticated public ownership independently of private draft scope, with post-body and pre-write revocation checks. Shared edit/capture transitions retain held-editor protection, latest-note lookup and original capture evidence. Historical open-note edits use current-page admission and unchanged owner checks. Reconnect catch-up now preserves all retained notes and original segments; shared Go/JS prefix equivalence covers revisions and resolver metadata. Node browser+terminal suites 334/334; Board/recording race packages pass (1.417/20.043 s); real Chrome viewer ownership/edit, operator edit/capture and two-viewer tests pass (34.702 s, `/tmp/pair538-merge-edit-live.log`). Post-merge stored/opaque crop tests also pass (12.502 s, `/tmp/pair538-merged-crops.log`). Broad affected Go packages remain running; the machine-gated full suite has not started yet, so its eventual snapshot includes this integration.

- Integration committed at 5e129ac4b; clean-tree `make build` passed. All webcast subpackages and core submission passed the broad merged run; broadcast passed in 125.152 s, with couchtty still running at this checkpoint. Artifact inventory passes. Full suite remains queued under the machine gate.

- Broad merged verification completed successfully: all webcast packages, core submission, broadcast (125.152 s), couchtty (67.280 s), inventory, Node 334/334, clean build and five named real-Chrome cases. Hub receipt db65eb48-e62f-4da1-aae3-d02e396daaa5 explicitly authorizes close on this evidence plus PR CI: **full suite starved at the gate, ariadne#357**. The queued run was interrupted without running its targets; no full-suite passing claim. Remaining unchecked Plan row is this close review and subsequent publication/landing; these follow the acceptance gate.

- Close round 1 returned REWORK: BR-1 found missing connected-original identity in the operator placement adapter (common text / large-page originals); BR-2 found stale design names and classifications in the plan inventory. Both findings accepted. Added failing adapter regressions; trusted recorder/CDP identity bridge is being implemented without weakening replacement scoring. Appended an authoritative implementation inventory to Revisions, preserving the original design history. Review accepted the full-suite starvation disposition; no new suite request.

- BR-1 repair verified: recorder grants only current-document, same-context original backend identities; bounded CDP delivery supplies actual DOM objects to weak-reference operator placement. Replacements still require the unchanged confidence rule. Adapter regressions cover common text, >2,100 nodes, stale/foreign documents, no rebinding, pause and pruning. Node suites 342/342 pass; all webcast Go packages pass; focused bridge/recorder race tests pass (1.344/1.168 s); artifact inventory passes. Real Chrome TestOperatorLiveAnnotation passes in 10.00 s with operator and viewer-authored common Save anchors both visible on the oversized page. Durable plan inventory and atlas now match these seams.
