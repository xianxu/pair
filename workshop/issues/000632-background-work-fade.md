---
id: 000632
status: codecomplete
deps: [pair#625]
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours: 1.74
card_mirror: 'd2450eba45df3891faf273094b4f7394369b353f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T22:52:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 2.63
---

# Keep background-command waits bright without blocking peer admission

## Problem

#625's admitted-wrapper work timestamp fixes open and completed peer-driven turns. A separate background-command case cannot use Settled: a free composer must continue receiving messages while a 10–30 minute suite runs in the background.

Read-only live evidence on 2026-10-16: ariadne:2, Claude 2.1.296, shows an empty composer, "1 shell still running" and a "1 shell" footer while waiting for its recorded suite. The authenticated resource snapshot at 19:33:16 PDT reports idle_known=true, idle=true, implying Settled=true. CPU/rate evidence was partial and is not the work proof.

The existing SessionFrame/SlotLiveness snapshot has no independent background-work field. Detached threads do not give Couch an agent-pane snapshot; peek reads it on demand via messageService.handleTail -> TailReader.Tail. Polling that endpoint would add an IO probe. Checked-in peer/claude/2.1.296 shell-footer.raw and idle-suggestion.raw contain the same positive shell counter.

## Spec

Provide display activity for running background commands independently of message/restart admission. Reuse an already-owned observation where available; if a new wrapper field is necessary, explicitly design its negotiated compatibility, identity/lifetime and bounded cache semantics. Do not set Settled=false solely because background commands exist. Preserve message receipt to a free composer while the command runs.

Approved scope (TL a807af4c): recognize Claude native shell footer only. Codex/Qoder remain unknown and retain their current behavior. The negotiated field and lifecycle are shared; additional agents require native captures plus positive/negative boundary tests. Live conformance is explicitly post-landing, after the next restart, owned/coordinated by the TL; no pair:0 staging.

## Done when

- A Claude background-command-only wait remains bright despite file activity older than a day, including a detached thread.
- The free composer continues receiving peer messages during that wait, with unchanged admission behavior.
- Completed/zero-work state eventually ages normally; quoted body/composer text and stale/disconnected/replaced wrapper evidence do not count as work.
- Recorded positive Claude footer and negative controls precede code; the approved post-landing live conformance checks both brightness and successful message receipt under a running background suite. Codex/Qoder remain unknown.
- The issue records which native evidence and boundary tests would qualify additional agents; post-landing live acceptance remains explicitly pending until performed.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. The calibration is marked stale by estimate-source, so these values are provisional. Existing terminal geometry, Unix session coalescing, fakeTimers and Console fixtures cover the libraries/stack; no new external library is needed. Footer classification uses a greenfield single-concern budget with design halved for the existing geometry, then ×0.2 for the approved plan. Other approved extension designs receive ×0.2. Familiar territory from #625; implementation values below apply the v3.1 40% scale exactly once.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.04
item: greenfield-go-module design=0.05 impl=0.24
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.02 impl=0.12
item: real-api-discovery design=0.00 impl=0.12
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 1.74
```

Rows respectively cover spec/approval, pure native footer classification, generation-fenced wrapper publication, negotiated transport compatibility, registry lifetime/completion, activity/admission integration, installed native conformance, docs, and one close review. Design0.35h ×1.15 plus implementation1.34h =1.7425h, rounded1.74h. Queue time is not execution work; machine suite and live rollout dependencies remain explicit rather than being hidden in a guessed total.

## Plan

- [x] Design the independent display signal and its lifecycle/compatibility before implementation.
- [x] Implement and verify generation-fenced Claude footer display evidence, preserving free-composer admission.
- [x] Implement and verify negotiated transport, admitted-session fade/aging and compatibility.
- [x] Update docs, complete the recorded suite and record the approved post-landing live steps. Ordinary review/PR/landing remain workflow gates after these pre-close checks.

Proposed design: [background-work display plan](../plans/000632-background-work-fade-plan.md). A negotiated display-only field is necessary; TL a807af4c approved the design, Claude-only native coverage, full flow and post-landing live rollout without staging.

## Log


- 2026-10-17: closed — Corrected candidate bead8f5d4368548f002be8133a907a441dda80e6 passed its first recorded full make test in 10m27s after2023s queue, log SHA256 5aa1917518d8c908480ab1f6d60829a98af98604bd3f9fa4ce4265ca983975d4. BR-1 ten behavioral RED cases now GREEN; captured active/no-shell controls, focused settle/admission race and seeded fuzz pass. BR-2 removed the unchanged Console test from changed-entity inventory and records reused validation. Lessons added for both classes. TL-approved Claude-only scope and post-landing installed live acceptance remain explicit; no pair:0 staging.; review verdict: SHIP
2026-10-16: filed from #625 under TL decision19e11f39-5e0e-46cf-8fe8-a1fbc76d223b; current #625 lands its existing work-state projection plus approved LastActivity reconnect seed. This independent surface follows that landing; no stacked code or live rollout performed here.

### 2026-10-16 — resumed dispatch and source trace

Verified TL dispatch6aca2623, claimed #632, published slot label and started planning on fresh main. First start-plan refreshed the Ariadne substrate and refused until rerun; second succeeded. Read-only trace confirms no cached background signal in the admitted snapshot, no existing job detector and no captured Codex/Qoder positive background footer. Claude's recorded native footer is positive while admission remains free; the completion prose above the composer is not authoritative. Existing settle callback is the bounded post-Feed source, but its unchanged-Settled early return must not suppress independent display updates. No production/test code edited. Durable plan proposes hello-v4 plus nullable BackgroundWork, generation-fenced sampling and existing #625 activity consumption; explicit TL scope/rollout/flow decisions pending.

## Revisions

2026-10-16: added a proposed durable plan for the new cross-process display field. Original Done-when remains the contract; initial Claude-only native evidence coverage is a requested scope decision, not an assumed approval.

### 2026-10-16 — approved design and coverage evidence

Verified submitted TL a807af4c-ef1d-41f8-b058-5c2dd66f21a8: hello-v4 nullable BackgroundWork, independent settle callback sampling and session invalidation approved; Settled/admission unchanged; Claude only, Codex/Qoder unknown; full flow; live check after landing/restart, no pair:0 staging. This explicitly narrows native recognition while retaining the shared protocol and display mechanism.

Read-only live peek after the plan draft found Codex0.162.1 pair:5 and ariadne:1 with a finished turn, free composer and native faint `1 background terminal running · /ps to view · /stop to close`; active-turn Codex0.162.1/0.163.0 slots show the same counter on the Working row. This is additional positive evidence, not a qualified parser or change to the approved scope. To add Codex, capture complete synchronized native terminal frames for ended-turn positive, completed/zero, plural, body/draft near-matches, and redraw/resize variants, establish geometry relative to its composer and test lifetime/compatibility. Qoder has no discovered positive capture: first observe/capture its native running-job UI or identify an owned native signal, then define equivalent false/unknown boundaries. No wording-only inference for either.

2026-10-16 revision: approved native scope and post-landing live schedule are now reflected in Spec/Done-when/Plan rather than leaving the original generic contract ambiguous.

### 2026-10-16 — plan-quality accepted

First plan-quality round raised PQ-1 (function-level-test-strategy), resolved by naming tested functions and their adversarial/mechanical strategies across all risky surfaces. Round2 CLEAN/high-confidence disposed PQ-1 addressed, no further findings. Normal change-code then refused only the missing estimate, as designed. Derived the itemized1.74h after that acceptance; no code has been changed.

### 2026-10-16 — implementation checkpoint, targeted verification

Change-code passed full flow with plan-quality CLEAN and estimate-quality INFO. Verification authoring and focused runs are included in each implementation row; suite execution/integration are covered by the validation/review allowance (the estimate is provisional and optimistic), while queue time is separately observed. The native-conformance row covers preparation and a bounded installed check once TL supplies a rollout; it does not imply a new API discovery or that deferred execution has happened. Any later worker execution is measured in its actual session, not converted to operator-attention time.

Behavioral RED after API-only scaffolds: both recorded native shell footers returned unknown instead of active; malformed grammar/no-shell controls also exercised. GREEN uses only the footer below the recognized Claude composer and bounded native management/counter fields. Seeded malformed-footer/geometry fuzz PASS1574executions in4.474s. Timer RED showed unchanged Settled suppressing active background evidence and complete native footer still unknown; GREEN uses proxy.settleState to take ONE terminal snapshot for both independent observations. Existing source generation and pending-output guards fence publication; state-copy-only SessionClient publication is ordered under the delivery source lock, so newer invalidation cannot overtake it. Source activity clears positive evidence before recheck. Controlled stale-probe and terminal frame/resize/close checks pass.

Actual free-composer regression replays idle-suggestion with a native running shell: settled=true plus active background; normal peer reserve/enqueue/paste/submit reaches Submitted and the shell footer remains active. Settled's rule and admission endpoints are unchanged. Wrapper focused race PASS1.755s; broad wrapper PASS73.799s, couchtty PASS66.911s (including the existing old-file/48h-silent-wait/completion/aging Console integration), artifactpath PASS12.318s. No new production files or artifact families.

Protocol behavioral RED preceded hello-v4 validation/emission/fallback implementation. Strict frozen v3/v2/legacy broker schemas, independent settled+background-active state, caller ownership, nil clearing, coalescing, reconnect latest-state and refusal/malformed/timeout guards pass; focused race PASS6.900s. Registry behavioral RED preceded owned per-session hint and monotonic work/completion transitions; focused registry/activity/real-client-transport tests and race pass. Full couchmessage package PASS15.605s; full couchcmd package still running at this checkpoint. Full repository recorded suite, ordinary close review and landing are not yet claimed.

Broad targeted verification completed: full couchcmd PASS127.927s with full couchmessage PASS15.605s on the final combined implementation. All earlier targeted/race/fuzz results remain applicable; only issue/atlas/README documentation changed afterward. Commit and push this candidate, then run the declared recorded full suite.

### 2026-10-17 — recorded full suite and relaunch checkpoint

Recorded suite on 134668267d85518ed21972b0753be273b2552f8e PASSED first attempt: exit0 in12m51s after4030s machine-queue wait. Retained log SHA256 de88c00992956d7ed94d1373b1cc67d2d3dbe1bf8cc6a33c46d71e238c84e17a; sdlc-local/suite recorded on that exact pushed code head. Full-log scan found no failure markers. README/atlas describe the independent Claude-only source and compatibility; the durable plan records the approved post-landing live checks. Pre-close checklist complete; close review, PR and landing still pending. Installed live acceptance remains deferred to TL's restart after landing; no pair:0 staging performed.

Verified submitted TL522a65cb-010e-4727-a3db-9d0d8481f523 requests an operator-approved rolling relaunch at the next point with no queued/running suite, merge or test, before the next long wait. This suite is now finished; checkpoint/push this evidence, report idle for relaunch, then resume ordinary close review after the hub completes or dispositions the relaunch. No extra test run is needed for this documentation-only checkpoint.

### 2026-10-17 — close round1 REWORK and verified corrections

TL6da34ade-c603-4f46-a52b-776b57ab720c verified submitted: rolling relaunch refused for the provisional Codex binding (#618 M2); continue close review, PR and landing. First normal close review independently passed all four affected package suites and focused race checks, but returned REWORK/high with BR-1 unknown-evidence-preservation and BR-2 core-concept-change-traceability. No close finalized.

BR-1 behavioral RED: ten cases returned known-none or active for arbitrary permission/footer text, truncated shell tokens, missing counter or malformed/unknown suffixes. The parser now validates the complete recognized permission and field grammar; a missing counter proves known-none only with the recorded no-shell cycle chrome. Four native no-shell captures replace the earlier synthetic misplaced-cycle negative control, while recorded running-shell positives and explicit zero grammar remain. BR-2 is prose-only: removed the unchanged Console test from the changed-entity table and recorded reused validation in plan Revisions. Added prevention rules for both finding classes to lessons. Targeted GREEN and subsequent candidate verification follow below; original full-suite record is retained but does not cover this code correction.

Correction verification: focused BackgroundWork/recorded-idle/settle/admission tests PASS0.405s; same focused race PASS1.946s; seeded malformed-footer/geometry fuzz PASS1120 executions in4.435s. git diff --check clean. Commit/push this correction and retain a new recorded full-suite result before the second ordinary close review.

Corrected candidate recorded full suite PASSED first attempt on bead8f5d4368548f002be8133a907a441dda80e6: exit0 in10m27s after2023s queue; log SHA256 5aa1917518d8c908480ab1f6d60829a98af98604bd3f9fa4ce4265ca983975d4. Full-log failure scan empty. Original successful suite and round1 REWORK remain recorded separately. Ready for normal second close review; installed live acceptance remains approved post-landing/restart.
