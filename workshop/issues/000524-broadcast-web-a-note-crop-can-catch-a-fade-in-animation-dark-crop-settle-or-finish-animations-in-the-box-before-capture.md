---
id: 000524
status: codecomplete
deps: [pair#483]
github_issue:
created: 2026-10-12
updated: 2026-10-13
estimate_hours:
card_mirror: '64607a4bfbd0e74499185c51c206c05b8b2cf953' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T22:49:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:0
    worktree: /Users/xianxu/workspace/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 2.67
---

# broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture

## Problem

pair#483's smoke test (2026-10-12, the operator's blog): one of three crops (`crop-0.png`, the 'Couch: A Sneak Peek' post card) came out nearly black, text barely visible, while its box text was exact. The blog's cards use AstroWind's scroll-in effect (`intersect-once intersect-quarter motion-safe:md:opacity-0 motion-safe:md:intersect:animate-fade` in src/components/blog/ListItem.astro / GridItem.astro): they start at opacity 0 and fade in once in view. The capture caught the card at or near opacity 0. Bundle: ~/.local/share/pair/couch/webcast/submissions/9dc7b091…/crop-0.png.

## Spec

Find why the card was still faded at capture (measure first: does the CDP capture path re-render or restart the animation; is the cast Chrome throttling animations when not frontmost; is the intersection never reached). Then capture settled pixels: before each crop, wait (bounded) for animations/transitions intersecting the box to finish (`document.getAnimations()`), or force their finished state for the capture, and restore after. Never steal focus. Record in the crop's metadata when a capture couldn't settle.

## Done when

- The operator's blog card crop shows its text at full opacity; a test page with a fade-in on intersection reproduces the dark crop before the fix and passes after; a capture that can't settle is marked.

- Responsive fades already finished before capture stay finished with the same animation identity; capture itself emits no induced viewport change. A mutation restoring captureBeyondViewport=true fails the pixel/timeline regression.
- Approved viewport policy is enforced: partial crops carry their actual box and clipped_to_viewport, fully offscreen notes keep context with outside-viewport unavailable, and no page scrolling/style forcing occurs.
- Finite, delayed, paused/infinite and open-shadow animations produce validated settled/unsettled evidence under 1500 ms/note, 50 ms polls, 512 animations/20k traversal and 30 s total; old bundles remain readable as unknown.

## Plan

- [x] Measure the original bundle/recording and reproduce in a separate bounded headless Chrome.
- [x] Obtain TL approval of the measured design and viewport-only crop policy below.
- [x] Add deterministic animation/capture regressions; implement bounded settling and unchanged-viewport screenshot capture.
- [x] Verify metadata propagation, navigation/cancellation, real-browser pixels, full suite and docs.

Detailed proposal: [animation capture plan](../plans/000524-animation-capture-plan.md).

## Log

### 2026-10-13 — edited-draft completion guard

A controlled adapter regression reproduced a send completing after its buffer changed during the settle window: unconditional done consumed the edited draft. Completion now checks the original callback's current predicate before clearing, while acknowledgment still proceeds. RED then GREEN; Lua model/adapter and all production headless DOM chord cases pass. This is an additional verified Lua delta after the broad suite's Lua targets. The original real-Chrome readiness and crop conformance tests also pass on the fresh-send implementation (7.09 s and 0.95 s).

### 2026-10-13 — no-notes fresh-send sweep

The DOM chord's empty-note fallback still used the ordinary sender's retry lock. Extended the same fresh-send transaction to the fallback without changing ordinary Alt+Return behavior. Failure-phase regressions went RED then GREEN; the production headless no-notes chord now accepts edited current text after an uncertain write. Updated generic sender and all DOM headless cases pass. Full suite is still running; this final Lua delta was separately reverified after its test targets had run in that suite.

### 2026-10-13 — fresh-send implementation verified

Implemented the approved no-freeze policy: every DOM chord captures current text under a fresh ID; failed DOM transactions retain no retry-body lock. Durable supersession preserves actual delivery state/files, rejects stale begin and permits genuine late acknowledgment. Store writes use one shared atomic writer with injected persistence failures and same-ID successor safety. Acknowledgment receipts are independent of draft epochs and bounded; duplicate/stale callbacks cannot fork their retries. Ack/status can resume confirmed resolution after a busy worker, failure or restart.

RED/GREEN reproduced the original preparation freeze, uncertain-send replacement, and duplicate ack callback fork. Pure Lua model/adapter, generic sender headless tests, and the complete submission-transaction-nvim-test.sh suite pass (including pre-error/edit and uncertain-write/edit production chord cases). Fresh affected Go packages pass: submission, recorder, webcast/submission, board, webcast and couchtty. Subagent ran focused backend race tests successfully for submission, couchtty and board. Atlas and lessons updated. Full repository suite remains running at /tmp/pair524-fresh-submissions-suite.log; not yet claimed green. Next: boundary review, then clean committed rebuild and embedded VCS metadata verification before smoke.

### 2026-10-13 — Finder metadata blocker verified

The new regression reproduced the exact unexpected .DS_Store entry refusal, plus five other foreign-entry cases. Pending now filters non-directories and invalid submission IDs; genuine corrupt submission directories still refuse. Fresh submission, webcast/submission and webcast packages all pass; fresh focused review PASS; make build in pair:0 succeeds. No storage bounds changed.

### 2026-10-13 — recovered session and main integration

Merged origin/main at 844d29e698e481712f6a58f37f04e4d717a405f3 for operator staging. Resolved three conflicts by retaining both settlement/clipping tests and privacy tests, plus both capture settlement and origin metadata. ARCH-ORDER: retained #526 capture-origin identity comparison and origin propagation alongside #494 exact-context screenshot suppression. Fresh four-package run passed (submission, recorder, webcast/submission, webcast); opt-in headless TestLiveSubmissionCaptureReady passed in 8.39 s and TestLiveSubmissionCrop in 1.72 s. Operator smoke remains pending before landing.



- 2026-10-13: closed — BR-1 fixed by TL-approved document-wide animation observation: no target paint-bound exclusions; unchanged 1500ms/50ms/512/30s limits, explicit cap uncertainty. Node RED then GREEN includes leaf shadows and no geometry access. Live crop readiness passes twice (8.08s, 6.93s), child/pseudo/box-shadow/drop-shadow pixels and off-box infinite unsettled; initial strict timing mismatch disclosed with improved diagnostic. Existing crop conformance passes 3.11s with merged #494 helper and captureBeyondViewport=false. Full serial make test completed: changed packages/shell checks pass; known #519 failed, plus TestObserveDependencies/deleted workspace-host busy (isolated full test rerun PASS 4.583s; no couchcore issue diff). Log /tmp/pair524-probe/full-suite-document-wide.log. Docs and Revisions updated. Stage only on TL instruction.; review verdict: SHIP
- 2026-10-13: flow upgraded quick → full — 403 added lines in code files (limit 100); an earlier round of this close already ran the full review
### 2026-10-12 — capture-time restart measured

Original evidence: the first article already had data-animated=true and no no-intersect at recording start. Its fadeInUp animation is 1 s, opacity 0→1 and translateY(2rem)→0. Periodic recorded frames immediately before and after the dark crop show settled content. The original viewport was 864×997 at DPR 2; crop right edge 870.5 exceeded it by 6.5 CSS px. No live operator browser was touched.

Bounded isolated headless probes (temporary profile, 30 s test ceiling) used the production CaptureCrops path and localhost:4321 blog. A settled card had opacity=1, animation state=finished, currentTime=1000 before capture. It returned to opacity≈0.033 and a new animation startTime afterward. The same failure occurred with a crop entirely inside the viewport, so overflow alone is not the cause.

A direct Page.captureScreenshot A/B changed only captureBeyondViewport, with fromSurface=true and the exact original clip. true→false→true→false yielded bright-pixel counts 0→19891→0→19891. True emitted resize/media-query notifications and restarted the animation; false emitted neither and preserved the finished timeline. Thus pre-capture waiting alone cannot fix this bug: capture itself restarts the responsive animation. Background throttling and an unmet intersection trigger are not required to reproduce it. This identifies the causative CDP option, not every Chromium internal step.

Probe logs: /tmp/pair524-probe/blog.log and flags.log; measurement-only Go overlays live beside them. Synthetic finite-delay probe also showed that still-running fades legitimately produce black pixels before finishing, so bounded settling remains useful after removing the capture-induced restart. No product code changed.

### 2026-10-12 — first implementation verification

Implemented unchanged-viewport screenshots, synchronous bounded animation observations with Go-owned polling, clipped/offscreen evidence, and strict settlement schema/propagation. Targeted schema/recorder/builder/webcast packages pass; existing headless crop coordinate/conformance test passes (1.70 s). New capture option test failed on true before the fix. Fractional CSS box regression exposed spurious clipping from subtraction rounding; unchanged axes now retain their original dimensions. New responsive animation browser fixture and mutation checks are next. #494 notified that its pending branch wraps the screenshot with CaptureScreenshot; preserve that helper if it lands before #524.

### 2026-10-12 — real-browser regression and mutation

The new responsive fade fixture confirms the fix preserves the completed CSSAnimation identity/startTime and bright pixels. Mutation overlay restoring captureBeyondViewport=true failed because animation identity changed to a new running fade. Conformance also corrected two test/model assumptions: Chrome reports one 800 ms finite endTime as 799.9999999999999 (assert finished/progress instead of a rounded constant), and document.getAnimations excludes open-shadow animations. Implement explicit bounded open-shadow-root enumeration; the approved scope already includes these roots.

### 2026-10-12 — final bounded browser evidence

TestLiveSubmissionCaptureReady passes in 2.91 s: finished responsive fade identity/pixels, naturally completed finite and open-shadow animations, irrelevant infinite animation, in-box infinite metadata, viewport clip and fully offscreen unavailable. Existing TestLiveSubmissionCrop passed in 1.70 s. The isolated localhost blog production-path check passes in 3.52 s: actual box [0.4,282.2,863.6,138.7], clipped=true, settled, 19891 bright pixels, opacity stays 1 and animation startTime stays unchanged. Measurement logs remain /tmp/pair524-probe/blog-fixed.log and flags.log. Mutation reverting the screenshot option fails the live test on restarted animation identity; mutation dropping settlement propagation fails Cast tests. Latest main (#506) merged cleanly; serial full suite now running with GOMAXPROCS=2 and all nested Go package parallelism limited to 1.

### 2026-10-12 — full-suite close evidence

Serial full make test completed (GOMAXPROCS=2, nested Go package parallelism 1). All shell/editor checks and Go packages passed except TestNotificationBrokerBeforeExecAndCleanup, the pre-existing startup-hook failure tracked as pair#519 and previously reproduced on main during #483. Full output: /tmp/pair524-probe/full-suite.log. Affected packages all passed, including webcast (8.052 s), recorder (4.134 s), submission schema (1.198 s), and submission store (0.176 s). No new suite failure. Browser and mutation evidence above verifies the changed behavior; operator smoke remains a TL-scheduled shipment step.

## Revisions

- 2026-10-12 — Measured cause revises the initial hypothesis: the screenshot path restarts an already-finished responsive fade. Proposed design avoids viewport-changing capture, then waits boundedly for relevant finite animations. Proposed viewport intersection and explicit offscreen-unavailable policy require TL approval before implementation; see the durable plan. ARCH-PURPOSE: fix the capture-induced restart rather than adding a sleep before it.

- 2026-10-12 22:56 PDT — TL approved the full proposal (receipt 5992b8ba-8060-42e0-949a-d52fc973ae69): captureBeyondViewport=false, 1500 ms/note with 50 ms polling and 512-animation cap inside 30 s, explicit settled/unsettled, viewport-intersected crops/clipped_to_viewport, outside-viewport unavailable with recorded context retained. No forced styles or scrolling. change-code admitted implementation.


- 2026-10-12 — Done-when refined to spell out the approved viewport policy, measured responsive-restart regression and metadata/scope bounds. Original goal remains; no new feature scope.

## Shipment handoff

Close through SDLC after verified automated evidence; stage only on TL instruction, then land after acceptance and report to the hub. #494 screenshot-suppression integration remains the second lander's responsibility.


## Revisions (continued)

### 2026-10-12 — BR-1 animation influence correction

ARCH-PURPOSE: target geometry alone does not bound animated paint. Direct targets and enclosing ancestors remain covered; zero-size/off-box targets with light children, shadow content, or generated pseudo-elements can affect overflowing pixels. Conservatively treat those active animations as potentially relevant within the existing 512-animation and 1500 ms limits. Finite animations wait naturally; paused/infinite ones remain explicitly unsettled. No per-animation descendant walk or mutation is needed. Plain off-box leaf targets without generated content remain irrelevant. Regression fixtures cover finite/paused/infinite cases and browser pixel evidence. This corrects influence detection within the approved capture contract, with no new scope or limits.

BR-1 verification: production-probe Node regression first failed on finite overflowing-parent influence, then passes finite/paused/infinite child, shadow, generated pseudo, pseudo-object and pseudo-effect cases. Recorder package passes (4.146 s). Bounded live browser regression with the old probe overlay failed: settled at 49.96 ms of a 900 ms parent fade, still running, 0/28800 bright pixels. Fixed probe passes in 4.88 s, with naturally finished timelines and bright pixels for both overflowing child and ::before. Serial full suite completed again after this correction: all shell/editor checks and Go packages passed except the same pre-existing pair#519 startup-hook failure. Log: /tmp/pair524-probe/full-suite-br1.log. No new failures.


### 2026-10-13 — BR-1 second review and approval hold

Second close returned REWORK: overflowing children/pseudo correction is exercised, but an empty off-box leaf can still paint an opacity-animated box-shadow into the crop. Added production-probe regression and confirmed RED (leaf shadow running falsely settles); this test is intentionally failing pending the policy decision. Current production remains at the first correction. Requested hub approval (receipt 4ed19fc7-774d-4bb6-b118-402976e124d7) for simpler conservative observation of all active animations in the supported top-document/open-shadow scope, within unchanged budgets. Tradeoff: unrelated off-box animations may consume the wait or produce unsettled evidence. No partial CSS paint-footprint analyzer proposed. Do not implement this policy change until hub replies; no staging or merge. Prior full-suite evidence remains accurate for the committed first correction, not a claim that the new RED fixture passes.


## Revisions (2026-10-13 approval)

TL approved conservative document-wide observation (receipt 3e4e0d53-b86b-4ca6-a147-d487a5cef374). Replace all target-geometry exclusions with observation of every active animation in the top document and accessible open shadow roots. Painted influence includes children, pseudo-elements, shadows, filters, outlines and other paint expansion; no CSS paint-bound approximation is attempted. Unrelated off-box animations may consume the wait or mark evidence unsettled. Unchanged 1500 ms/note, 50 ms polls, 512 animations, 20k traversal, 30 s total; crossing limits explicitly marks unsettled. This supersedes earlier claims that plain off-box leaves are safely irrelevant. #494 is now merged; CaptureCrops uses its exact-context CaptureScreenshot helper with captureBeyondViewport=false. Stage only when TL says.


### 2026-10-13 — document-wide correction evidence

Removed all target geometry/style exclusions. Node fixtures fail on the prior probe and pass now for off-box active animations and leaf shadows; a throwing geometry accessor proves readiness no longer guesses painted bounds. 512 entries accepted; 513 explicitly inspection-limit unsettled. Four affected Go packages pass (schema 1.222 s, recorder 4.390 s, store 0.185 s, webcast 7.897 s). Updated real Chrome fixture verifies off-box infinite unsettled followed by cancellation, finite natural completion and bright pixels for child/pseudo/box-shadow/drop-shadow cases. Two consecutive runs pass (8.08 s, 6.93 s). Initial run failed the strict finished-fade timing comparison with same object/state/viewport; old pointer-only diagnostic did not preserve numeric values. Cause unestablished; improved numeric diagnostic retained and invariant unchanged, no passing claim for that run. Serial full suite running after main merge and policy correction.


### 2026-10-13 — final verification before third review

Existing TestLiveSubmissionCrop passes with merged #494 capture helper (3.11 s). Serial full suite completed after the document-wide correction and main merge: all shell/editor checks and changed Go packages passed. Full run was not green: TestObserveDependencies/deleted encountered workspace-host creation busy; isolated rerun of the complete TestObserveDependencies passed in 4.583 s, and the issue patch changes no couchcore files. Known TestNotificationBrokerBeforeExecAndCleanup pair#519 also failed as before. Evidence: /tmp/pair524-probe/full-suite-document-wide.log. No test invariant weakened. Close review must evaluate BR-1 against the approved document-wide policy, which supersedes geometry-based relevance in earlier plan text.


### 2026-10-13 — operator smoke blocker scope revision

TL requested folding the #483 store scan blocker into #524 (receipt 0a7fad73-59a4-4f59-b9d7-fb7727002cc3). Finder's .DS_Store is treated as an invalid submission by Pending, which also gates Prepare. Ignore non-directory entries and directories whose names cannot be submission IDs; preserve strict validation of actual submission directories and existing storage bounds. ARCH-PURPOSE: foreign filesystem metadata must not prevent collecting the crop needed for smoke. Plan: reproduce with .DS_Store/dotfiles/stray files, change the scan filter, retain corrupt-bundle refusal, run store and affected integration tests, review the small diff, rebuild in pair:0 and hand back the measured HEAD for smoke.


### 2026-10-13 — second smoke blocker, no-limbo recovery

TL requested pre-dispatch draft release and an actionable uncertain-dispatch retry/discard path, plus recovery from the old-code freeze. Investigation: both Lua controllers keep frozen text only in process memory. The .DS_Store admission failure precedes preparation and writes no bundle. Existing server operations have no discard; begin persists intent before actual pane write. Proposed scoped lifecycle/API addition and tests appended to the durable plan for TL confirmation. Do not advertise the unrelated erase-history key as a submission discard. No implementation of this larger addition yet.


### 2026-10-13 — operator revises recovery to fresh sends

TL d34bfd0a-9f86-45d0-b4c8-775874af760c explicitly accepts duplicate sends and instructs no freeze, current draft plus new ID per chord, superseded earlier pending/delivering evidence, and idempotent note resolution on real acknowledgment. This supersedes the preceding discard proposal. Durable plan appended with the approved policy and verification steps. Old local text freezes are process-memory only; durable uncertainty is released by explicit supersession, never inferred delivery. Clean committed build metadata is now a required smoke check after TL e5568928-9e12-4777-8183-0371917e66e5 found the previous pre-commit build embedded an older dirty revision.
