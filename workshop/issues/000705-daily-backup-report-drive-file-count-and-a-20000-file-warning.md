---
id: 000705
status: working
deps: [pair#700]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 1.83
collaboration-mode: collaboration
card_mirror: 'f7b9ae0e19f0d7a320482a2da665116c9403f2c5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T08:45:42-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Daily backup report: Drive file count and a 20000-file warning

## Problem

The daily backup freshness report does not show the Drive destination's file count or warn before a large flat directory becomes inconvenient. The operator requested this while pair#700 was already codecomplete (SHIP) and waiting in the landing line, so it is a separate small follow-up.

## Spec

Extend the existing daily continuous report with the Drive destination's observed file count, reusing the listing the pass already performs: no extra remote call. Use the existing final listing when present, or clearly identify the available observation; a failed/unavailable listing is unknown, never zero. Count actual returned file entries rather than unique name keys, including duplicate-name entries; qualify the directory/non-file distinction against the existing listing semantics.

When the observed destination file count is strictly greater than 20,000 by default, issue one daily warning suggesting year subfolders on Drive. Keep threshold configuration in the same inherited environment-variable style as pair#700 (proposed PAIR_ARCHIVE_DRIVE_MAX_FILES; choose and document a positive integer contract). Reuse the existing scheduler admission/primary-hub notification seam with a destination-specific key and documented per-day/lifetime suppression. Warn/report only: do not move existing Drive files or change layout automatically.

## Done when

- The daily human and typed freshness reports expose the observed Drive file count without another listing or other remote call.
- The default boundary is tested below, at and above 20,000; an env override and invalid threshold values have explicit behavior.
- Unknown/failed listings cannot produce a misleading zero or healthy observation; the existing partial backup report remains useful.
- Daily warning suppression and destination identity are tested through fake notification output; the warning suggests year subfolders.
- Existing stateful Drive fixtures count remote calls and cover duplicates/non-file entries and partial listing/pass outcomes.

## Plan

- [ ] Implement and qualify the count/report/admission extension in [the durable plan](../plans/000705-drive-file-count-warning-plan.md), using existing remote listings and daily output.
- [ ] Document, submit the recorded suite, close and land on SHIP; hand the project/live observation to the hub.

## Log

Operator request via hub receipt 277127be-5775-4a0e-93e4-8d24a00600a5. #700 was already past close with SHIP; the hub explicitly requested a small follow-up in that case. Existing seams: historyarchive/drive.go DriveReport and ObserveFlatInventory before/after results, archivecmd continuous human/JSON report, couchcmd archive scheduler and primary-hub warning seam. Filed unclaimed for later dispatch; #700's current landing is unchanged.

- 2026-10-19, exploration: claimed and start-plan from fresh main; no code changed. FakeRclone already models before/final/partial-copy behavior and exact call counters. It needs only directory representation for non-file counting. Pure/adapter source seams and named adversarial test strategies are in the durable plan; hub approval requested before change-code. No live Drive/desktop calls or pair:0 work.

- 2026-10-19, plan-quality round 1: PQ-1 requested test-strategy presentation refinement only. Replaced case inventories with named risky-function adversarial classes and mechanical/property guards; no implementation, estimate or policy change. Rerun change-code.

- 2026-10-19, plan-quality round 2 CLEAN: PQ-1 addressed, no new findings. Derived the estimate only after this acceptance. Calibration source reports stale; numbers remain provisional. No production or test code changed before admission.

## Revisions

- 2026-10-19, hub 3ca340ad: #700 is verified/ticked/rebuilt; dispatch #705 now. Keep #706 folder migration separate and waiting for #689.
- 2026-10-19, design: threshold env PAIR_ARCHIVE_DRIVE_MAX_FILES, default 20000, positive integer. Optional exact destination/folder/count/phase observation grows only the runtime report; existing before evidence survives a final listing failure, and absent/failed/empty-candidate listing remains unknown without extra calls. Shared #700 gate/output, one atomic full-flow close because explicit protocol validation and common delivery may exceed the 100-line production shell.

- 2026-10-19, hub 2ea831d5 approved the plan, environment default and fake-backed acceptance. Daily count/admission is keyed by configured destination root; root FolderID is evidence only, not a month-child key. #706 can sum month folders into this same observation later. No staging or real Drive/desktop calls; run change-code.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; source is tagged stale, so this is provisional. Read v2/v2.1 primitive ranges, v3.1 scaling and shared grammar after plan-quality acceptance. No target total was chosen first.

Decomposition: issue/spec; one small count/threshold/rendering model and one small runtime codec extension; the existing Drive/CLI/scheduler adapters and stateful fixture; docs; one atomic review/verification boundary. The stack is existing Go time/json/strconv and established Rclone/notification seams, so no greenfield-library or real-API discovery allowance. Base design picks are 1.0 / 0.2 / 0.2 / 1.5 / 0.1 / 0.1 hours, each discounted by 0.2 for the resolved spec. Base implementation picks are 0.2 / 0.5 / 0.5 / 1.0 / 0.1 / 0.5, each scaled by 0.40. Familiarity 1.0; thorough-plan design buffer 15%. Adapter and review primitives include fixture qualification and the recorded verification flow; queue idle is not extra implementation effort. Single repo, no new API or coordination multiplier.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: api-integration design=0.30 impl=0.40
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 1.83
```

Design 0.62 × 1.15 = 0.713; implementation 1.12 × 1.0 = 1.12; total 1.833, rounded 1.83 hours.
