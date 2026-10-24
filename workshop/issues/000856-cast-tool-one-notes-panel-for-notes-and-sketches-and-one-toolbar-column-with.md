---
id: 000856
status: codecomplete
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours: 2.43
card_mirror: '77aba7721ea71f71f965b2c3bf036685b93a31df' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T21:35:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: N/A
---

# Cast tool: one Notes panel for notes and sketches, and one toolbar column with tools revealed on toggle

## Problem

## Spec

**Cast tool: one Notes panel for notes and sketches, and one toolbar column (operator, 10-23, from their echod
annotations).** Agreed design, from ops `session.side/2026-10-23-24-cast-tool-learnings.md` (with screenshots
described there):
- **One Notes panel for everything:** each note and each sketch is an entry (a sketch entry shows its thumbnail, its
  note, add/edit note, and "show on page"); the panel is the only place notes and sketch notes are read or opened.
  Replaces the per-sketch buttons and pair#850's separate "Sketch notes (N)" button.
- **One toolbar, one column:** live/pause first; Sketch as a toggle that, when on, reveals the palette, Undo, Redo and
  Done below it; Undo and Redo hidden when they can't act. Nothing (notes, popups, status) is drawn over the toolbar.
- **Status messages** (such as "Pending feedback is out of date") show in the Notes panel's header, not as a floating
  box.
The exact positions get settled against a mock flow (the operator's note: "once we have the mock flow set up, we can
be a lot more precise how things are laid out"), so the plan starts with a mock of the layout for the operator to
react to before code.

## Done when

- Notes and sketches are entries in one panel; there are no per-sketch on-page buttons (tests).
- The toolbar is one column with the reveal-on-toggle behaviour, and no other cast UI overlaps it at common viewport
  sizes (tests), plus the operator's live look.

- Sketch rows use small type icons, with no thumbnails; hovering or focusing a row glows only its confirmed page geometry and clears on leave/lifecycle change.
- The column uses Live/Pause, Pointer, Sketch (Pen) with colors, Annotation (Note), available Undo/Redo and Done. Finished labels remain read-only; status stays in the Notes header; unsaved-note submission protection remains intact.

## Plan

Durable joint design: [Cast feedback workspace](../plans/000856-cast-feedback-workspace-plan.md). #857 remains a dependent issue on its own eventual main-based branch.

- [x] Deliver the clickable layout mock and record operator approval.
- [x] Implement the unified panel and protected one-column toolbar after #850/#849 land.
- [x] Qualify the unified panel and column through Node and native browser tests; audit the reader inventory.

Close and landing use their SDLC gates. The recorded exact-source pre-close check must pass before landing; see the approved production-look revision below.

## Log

### 2026-10-24 — Actual attribution refusal; authorized N/A recovery
- 2026-10-24: closed — TLfab98e9b authorizes N/A: earlier rounds measured2.72h then3.33h; ariadne#580 now refuses branch attribution for062e2184-2117-4c26-b929-ceec3b5a8030: concurrent claims have unknown, resting or ambiguous active branch during2026-10-23T21:35:40-07:00 to2026-10-24T04:36:48Z (#850/#856 transition). Refusal/reflog retained, no fabricated hours. All604 Node tests and inventory pass. Final-source8 native/module tests PASS32.750s on087631062, retained native-final-source.log:16 actual panel/toolbar/status/editor measurements at1280/768/320/320x480, overlapmutation rejected, real40px captureddrag, pressure/privacy/tab/SPA/handoff and unsaved guards. BR1-3 regression fixes remain. Recorded lease-306801268 queued; pass required before landing. Acceptedmock/native evidence authorized; morning stage covers production look.; review verdict: SHIP

The third close stopped before dispatching review: branch attribution for062e2184-2117-4c26-b929-ceec3b5a8030 in pair-slot4 had concurrent claims with unknown/resting/ambiguous active branch during2026-10-23T21:35:40-07:00 to2026-10-24T04:36:48Z. The earlier rounds measured2.72h and3.33h under the previous engine. TLfab98e9b-8864-49ce-b36c-1dc135ebac80 identifies ariadne#580 (landed01:26) and concurrent #850/#856 claims as the cause; ariadne:3 owns the workflow fix. TL explicitly authorizes close with --no-actual, recording both earlier measurements and the exact refusal in verified. Do not transplant old numeric hours or fabricate new attribution. This close will use N/A/exclude calibration while preserving code verification. Retained suite lease-306801268 stays queued with its grant.


### 2026-10-24 — Final-source native set and recorded-check lease

One current-source run on087631062be649f7578cab38fe75ddc6c3044ed1 passed all8 named native/module tests in32.750s, no skips, including the final16-state composed geometry oracle and captureddrag. Output is retained in [native-final-source.log](../plans/000856-cast-feedback-workspace-plan.side/evidence/native-final-source.log). This supersedes mixing earlier interaction output with later geometry output: one source now supplies the complete qualification. All604 Node tests and reader inventory pass.

Recorded pre-close full fallback is retained as terminal5418, lease-306801268, run2d195b5c900c7a2bf81b53c80416abe4, detached snapshot /Volumes/TestScratch/local-checks/sdlc-local-1248088162; queued FIFO11 when reported. Hub was sent the exact pushed head/lease for priority re-grant. Preserve this runner; do not restart while queued/running. Next: full close round3 over the qualified source/evidence, then await the recorded pass and promptly enter merge. No stage action until hub sequencing; #857 starts only from landed main.


### 2026-10-24 — BR-4 composition evidence retained

[workspace-geometry.log](../plans/000856-cast-feedback-workspace-plan.side/evidence/workspace-geometry.log) retains the exact final-source non-skipped result on ac362f793a33997bcc0503208ed07171a8dd2711: all16 actual toolbar/panel/status/editor measurements, deliberate overlap rejection, captured40px drag, existing save/stale/prepare/unsaved guards, PASS6.63s. The red minimized-overlap log is retained alongside it; node-final.log records all604 Node passes. Reader inventory passed. Source trees match the committed test blob named in the log. This completes the specific composed-workspace qualification that round2 left open; earlier native-review-fixes.log continues to cover drawing pressure, privacy, tab/SPA and handoff on unchanged production paths.


### 2026-10-24 — Final native composition source green

Final TestOperatorFeedbackLivePrepareAndStalePanel passed6.63s (package6.861s), non-skipped, with exactly16 independent production rectangle measurements, overlap-negative mutation rejection, real ordinary/sketch editors and actual40px captured panel drag. Test source blob28b086fa339b18c4a0ad8a59d427a74018cce54a, printed by Git. Temporary prototype probes were removed. The minimized-width source fix is covered by this exact final matrix. All604 Node tests pass1.702s (zero skipped). Stage new test-source, finish reader inventory, commit/push, queue the final head and send its lease to TL for re-grant; then full close round3 with retained composition evidence. Do not create another check on the old cancelled heads.


### 2026-10-24 — Composed native matrix and drag diagnosis

With intrinsic max-content sizing, all16 production measurements passed (expanded Notes/status, minimized Notes/status, real sketch-label editor and ordinary-note editor at1280x800,768x800,320x800,320x480). A runtime panel-position mutation was rejected by the independent overlap oracle. Geometry checks wait for the existing private capture to finish and the UI host to be unsuppressed, rather than treating intentional screenshot suppression as a layout defect. Narrowing legitimately preserves the clamped anchor; the fixture restores page gesture access using the supported keyboard movement before continuing original drawing targets.

A further real-drag assertion exposed synthetic-input sequencing, not a production owner cancellation: CDP mouseMoved button:none with heldbuttons1 admitted then lost pending capture without an explicit release, resize, status or host mutation; the poisoned gesture prevented a subsequent comparison. A fresh run using mouseMoved button:left (the established standalone native drag fixture form) passed6.49s, package6.731s, with actual40px panel movement, all16 geometry states and the negative oracle. No drag production code was changed. Temporary prototype diagnostics are being removed; retain the final-source native result before queueing.


### 2026-10-24 — Native composition oracle exposes real minimized-width bug

The actual production matrix passed expanded states, then caught minimized320px overlap: Notes x183.17,width124.72,right307.89 crossed the toolbar beginning252. The inline minimized width:auto overrode CSS max-content. Measurement while still at the old offscreen anchor used shrink-to-fit min-content width; after clamping, the element expanded and escaped the reserved content. Preserve intrinsic max-content width before measurement (still capped by existing maxWidth), so one measured width remains valid after positioning. This is a correction to the approved complete-hit-area clamping contract, with the native resize matrix as the regression; no timeout increase. Focused27 panel/floating tests pass, native rerun pending.


### 2026-10-24 — Close round2: behavior fixed, composition evidence outstanding

BR-1/2/3 were addressed, including disposable mutation checks by the reviewer. BR-4 retained a precise gap: standalone panel bounds and reachable Pen buttons do not prove the actual protected column is separate from panel/status/editors at common widths. The working-host interaction evidence was accepted. Extending TestOperatorFeedbackLivePrepareAndStalePanel with independent native rectangle measurements at1280/768/320 and320x480 through the actual injected production workspace. Cancelled queued lease-2914357650 through SDLC successfully, no result recorded, so the final test head can be queued once without a guaranteed stale fixture. TL e567a7da revoked the grant while no check is queued and will re-grant on the final queued head. Existing PR583 remains open, server CI green; no merge runner or pair:0 action.


### 2026-10-24 — Retained native evidence for BR-4

The corrected production source is Git commit791e8742c1459ba95e1f84ce4ac83cec7265bf03. Non-skipped host output is retained at [native-review-fixes.log](../plans/000856-cast-feedback-workspace-plan.side/evidence/native-review-fixes.log); all8 named tests passed27.857s, including actual browser gestures, private capture, preview/unsaved refusal, minimized-editor reopening and viewport separation at1280/320. [node-review-fixes.log](../plans/000856-cast-feedback-workspace-plan.side/evidence/node-review-fixes.log) retains all603 Node passes. The source was committed unchanged immediately after these runs. BR-4's review-host GPU startup refusal is not product evidence; this working-host output supplies the requested qualification. Inventory re-audit passed after staging the new test. Requeued recorded pre-close is retained in terminal34212 on this source; no second run is active.

### 2026-10-24 — Close round1 REWORK and correction

Close review raised BR-1 (sketch membership depended on viewport placement), BR-2 (annotation hover scaled only origins), BR-3 (minimized status invisible), and BR-4 (review sandbox Chrome GPU startup failed). Reproduced/fixed the three behavior defects with red/green tests through pure and actual DOM adapters. Accepted identities retain keyed rows across offscreen/null/restored geometry; both corners of box and clip project through the viewer; minimized Notes visibly says Status with the full message in tooltip/accessibility text, remeasuring its bounded control. Also fixed the related adopted-editor reachability: opening note or label restores minimized Notes before focus. No saved-feedback/command authority change.

TL03d62472 directed cancelling the old queued check and fixing the obsolete native preview fixture before requeue. Exact lease-2850435122 cancellation completed successfully, no test result recorded. Migrated its Note/Pen/history and status selectors, preserving real CDP gestures and all Board/prepare/freeze/unsaved guards: native TestOperatorFeedbackLivePrepareAndStalePanel passed4.92s. Other native TestOperatorLiveAnnotation passed4.97s and OpenContactHandoffLabel passed1.83s. Focused post-fix panel/floating tests pass27/27; sketch/helper tests72/72; geometry/UI tests15/15. Complete Node/native reruns are in progress before the fixed head is pushed and requeued. No old check, merge runner or stage owned here remains.


### 2026-10-23

### 2026-10-23 — Joint design checkpoint

Claimed #856 and #857; started #856 from main while #850's final-source recorded check runs from its own detached snapshot. No #850 runner holds this checkout. #857 remains dependent and will get its own branch from landed main. Read the recorder/Board/UI seams and a bounded page-load survey; the durable plan records the state table, counted growth guards and codec/digest compatibility risk. No production changes yet. Next: commit the plan, send it to the hub tonight, then build the synthetic clickable layout mock for operator feedback. Preserve #850's retained check lease-3551124609 and return for merge when it passes.

Final correction qualification: all603 Node tests passed3.219s. One non-skipped native run passed all8 named tests (CastNotesPanelLive, OperatorLiveAnnotation, OperatorFeedbackLivePrepareAndStalePanel, OperatorSketchLiveDrawing, OperatorSketchLiveOpenContactHandoffLabel, and three bundle assembly/execution checks), package27.857s. The preview native fixture now minimizes before a new annotation and before Done, requiring the real editor rectangle to be visible; both restored correctly. No native test was skipped. Fix source is ready to checkpoint and requeue, with review-host GPU refusal answered by retained native evidence from the working host.

## Revisions

### 2026-10-23 — TL's morning scope and #849 seam

Finished sketch labels are read-only for this delivery; unfinished labels keep Add note. Show on page scrolls/highlights only a confirmed current-load anchor and reports unavailable without guessing, navigation or reload. #857 is designed here but lands separately. Keep #849's noteDialog.unsaved getter and press-time saved selection; the preview stays informational. These decisions supersede the original sketch add/edit wording for finished labels. Exact panel placement is still subject to operator review of the mock before implementation.

### 2026-10-23 — Plan approved; mock ready

TL approved the joint plan (receipt bdff0bdf-86f8-4099-94c7-8de3cfac6ab4). Created synthetic mock-1 under the canonical plan.side/mocks directory with one shared fixtures.json. Served at http://127.0.0.1:51106/ by retained terminal 71609. Node syntax and diff-whitespace checks passed. A temporary native qualification reused the repository's headlessPage CDP fixture and passed non-skipped in 1.49s (package 1.677s): mixed entries, finished read-only label, unsaved/save, unavailable anchor, tab return, same-load app update, reload partition, tool reveal, and independent rectangle assertions for expanded/minimized Notes against the toolbar at 1280/768/320×800. Inspected screenshots; no panel/toolbar overlap. The temporary qualification source was removed after use; this is mock evidence, not production feature qualification. Next: send the mock URL to the hub for operator layout feedback, then implement only after acceptance and landed #850/#849. #850's existing lease remains queued without duplication.

### 2026-10-23 — Operator mock refinement

Applied authenticated feedback b5c92d0c-2c24-4098-bb02-e83190857da9 to mock and joint plan: replace thumbnails with compact sketch/annotation type icons; hover or focus a row glows its corresponding page geometry without scrolling. Missing anchors do not guess. Highlight clears when leaving/reconciling/minimizing/changing page. Coordinate #842/#843 with pair:7: #856 owns protected column/layout, while send-button and tooltip behavior remain their issues.

### 2026-10-23 — Icons and hover supersede initial thumbnail scope

The operator's latest design removes sketch thumbnails from the original Spec. Entries use type icons and page-side hover glow instead. The plan records the corresponding lifecycle, current-load identity and no-selection-mutation requirements. Production code still waits for the layout look.

### 2026-10-23 — Revised mock qualification

Non-skipped native Chrome check passed 0.63s (package 0.789s): four compact type icons and no thumbnail element; sketch hover glows only its anchor; annotation hover glows its box; hover does not scroll; missing anchor clears prior highlight; actual keyboard focus highlights; minimize and reload clear transient glow. Initial keyboard-focus probe exposed that property-assigned focusin/focusout handlers did not run; switched to explicit event listeners and reran with actual focus (no synthetic focus event). Temporary fixture removed after qualification. Mock server remains at http://127.0.0.1:51106/ with live reload. Node syntax and diff checks rerun. #850 retained check acquired its slot after 4776s and is now running; no duplicate or cancellation.

### 2026-10-23 — Actual tools and mock viewing boundary

Applied operator feedback 96da76d2-2673-40a2-b1aa-38ef59fe0206: mock now includes Pointer and Annotation (Note), names Sketch as Pen, retains Live/Pause, colors, conditional Undo/Redo and Done, and removes Stroke. Marking a synthetic card in Pen mode supplies a history action. Instructions say open in a non-cast Chrome window: the real cast injection otherwise overlays the mock; no capture/origin exclusion changes were made. Added a new-tab scenario starting empty. #857's target+loader+root scope and native same-URL new-target regression cover the live cross-tab leak.

### 2026-10-23 — Tool mock verification

Non-skipped native Chrome qualification passed 0.80s (package 1.036s): actual tool IDs and no Stroke action; Pen page action exposes Undo, Undo exposes Redo; Note page action opens an unsaved editor and Cancel discards its draft; same-URL New tab starts empty and returning restores four original rows; panel/toolbar rectangles remain separate at 1280/768/320×800. Temporary fixture removed after use. Node syntax and whitespace checks pass. Next: operator layout review in a non-cast Chrome window at the existing mock URL. Production code remains held.

### 2026-10-23 — Operator layout acceptance

Authenticated TL e378240d-75dd-498d-96e8-91b542182269 records operator “yes, like it” on the reworked mock and directs production #856 then #857, aiming for the morning stage together with #849. Done-when now explicitly includes approved icon/hover and actual-tool refinements. Mock work is complete. #850 remains in its existing exact-source suite; no stacked issue commits. Next: change-code plan-quality/estimate gates and independently testable production work; integrate parent APIs only from landed main.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration source is marked stale/provisional by sdlc estimate-source.*

#856 only; #857 gets its own estimate after its gate. Treat the browser panel, tool column and hover/anchor interaction as three screen/state-machine primitives (same UI/state/test concern as tui-screen, implemented in existing JS). Panel and hover each choose v2 design=1h ×0.2 thorough-spec discount and impl=1h ×0.40 v3.1; column chooses design=.5h ×.2 and impl=.6h ×.4. Shared wiring uses cross-cutting-refactor design=.5h ×.2 and impl=.4h ×.4. Native CDP qualification uses real-api-discovery impl=.6h ×.4. Docs use .1h design ×.2 and .2h impl ×.4; one close review uses .5h impl ×.4. Familiarity=1: this is the same cast UI worked in #850. Library availability: existing floating panel, painter, command owners and native CDP fixture supply the reusable mechanisms; no novel framework or greenfield service. The approved clickable mock resolves layout decisions. Sum design=.62h, scaled impl=1.72h, +15% design buffer=.093h, total=2.433h, rounded2.43h. This estimates focused work, not FIFO waiting.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: tui-screen design=0.20 impl=0.40
item: tui-screen design=0.10 impl=0.24
item: tui-screen design=0.20 impl=0.40
item: cross-cutting-refactor design=0.10 impl=0.16
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0 impl=0.20
design-buffer: 0.15
total: 2.43
```

### 2026-10-24 — Prerequisites landed; production started

#850 landed in train342 through PR579 (car PR576), exact-source suite11m46s and integrated suite16m9s, final CI green, retained runner exited0 after archive/cleanup. #849/PR578 is also on main. Switched back to #856 and merged landed origin/main once before integration, preserving the independent issue branch and approved gate checkpoint. Applied independent feedbackEntries and feedbackLayout helpers after observing missing-module red in repo, then all15 Node tests passed (including typed keyed rows,32/64 linear visits, generated geometry and protected-toolbar containment). Code is now on #856, not a stacked parent branch. Next: shared panel/column adapters and trusted hover/reveal integration; preserve noteDialog.unsaved and press-time submission.

Morning stage instruction551aad52-aa8b-4fc9-84d6-6c5b9a9dfe10: combine #856, #857 if ready and pair:7's #842. Ask hub before staging; pair:0 remains leased by #851. Pair:7 will supply verified head/operator steps after recorded check and SHIP.

Integration survey: annotations publish existing note placement; sketches publish existing cached painted bounds. Hover uses a pre-mounted pixel canvas so pointer/focus does not mutate DOM and churn visibility stamps. Show on page must use confirmed original identity, never fuzzy replacement or historical rectangles; offscreen reveal needs a separate trusted-original action before normal placement refresh. Forward placement options through the viewer adapter. Tests live under tests/webcast/*.test.mjs (correcting the earlier plan's illustrative web/*.test.js command).

### 2026-10-24 — Unified production workspace checkpoint

Implemented one keyed Notes panel with annotation/sketch icons, existing editors and read-only finished labels; pixel-only hover glow; confirmed-original Show on page; one protected column adopting the actual Live/Pause, Pointer, Pen, palette, Note and useful history controls. Embedded/routed the new assets and migrated the real operator bundle and replay adapters. Removed the independent sketch selector and preview cards; publication/ack validation remains informational, and the shared header displays status plus pending feedback counts. Existing unsaved-note and press-time submission authority are unchanged. ARCH-DRY: controls retain their existing command/state owners and the panel owns presentation only.

TL decision 0ed24a81-0dcd-481a-a33b-9a6ee394b937: keep the pen palette visible beside Pen before starting; no separate arming step. Undo/Redo/Done appear only when useful, with existing recording and color rules. This deviation from the mock must be named in the operator stage steps.

Verification so far: all591 Node tests passed before the final header fix; the focused13 panel tests now pass. Non-skipped native TestCastNotesPanelLive passed2.40s at desktop/narrow widths. Migrated TestOperatorSketchLiveDrawing reached a real status conflict twice (Annotation is off hid Pending feedback); added a red regression and composed header count with the highest-priority message, avoiding identical DOM writes. The corrected native test passed6.31s (package6.546s), including real pressure gestures, note save, capture privacy/focus and row identity, A-to-B-to-A, mask restore and same-context SPA replacement. Operator bundle assembly tests passed0.551s. No native runner remains active; no recorded #856 check has started.

Next: rerun complete Node and focused Go/native checks on this source, stage new files before auditing make test-affected-inventory, commit/push, then recorded pre-close and full close review. Production operator look remains outstanding; coordinate the combined #856/#857-if-ready/#842 stage with the hub before touching pair:0. #857 remains a separately claimed main-based dependent issue, not implemented here. The old mock server is stopped.

Final checkpoint rerun: all592 Node tests passed (0 failures;1.928s), including the header regression. The concepts table now names actual module ownership and the retained pure painted-bounds helper.

### 2026-10-24 — Production-look sequencing approved

Authenticated TL66eb5fd0-b8a4-40d7-a896-3ba247374694 approves landing #856 after SHIP and checks: accepted mock plus native evidence suffices, with the morning combined stage covering the production look and later feedback filed separately. This revises Done-when's live-look timing, not its expected behavior. Then start #857 from landed main; morning stage is main plus #857 if ready plus #842. No stage action is authorized until the hub sequences pair:0.

Post-checkpoint verification: make test-affected-inventory passed; native TestCastNotesPanelLive passed1.88s and all three operator module assembly/execution checks passed (combined package2.361s). Recorded pre-close on pushed2d79cedc resolved to affected, then correctly selected full fallback for the Node test inputs. Retained terminal28316 owns lease-2850435122, run018c9e9769db284d99ed1540a4998d34, detached snapshot /Volumes/TestScratch/local-checks/sdlc-local-2771889491; queued FIFO11 when reported. Do not start a second runner or restart while queued. Full close review may overlap under the fleet rule; landing waits for the exact-source result.
