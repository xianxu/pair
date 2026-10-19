---
id: 000705
status: working
deps: [pair#700]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b3c483100900227312f0ee5ed8a0281abde0f476' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T08:45:42-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
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

## Revisions

- 2026-10-19, hub 3ca340ad: #700 is verified/ticked/rebuilt; dispatch #705 now. Keep #706 folder migration separate and waiting for #689.
- 2026-10-19, design: threshold env PAIR_ARCHIVE_DRIVE_MAX_FILES, default 20000, positive integer. Optional exact destination/folder/count/phase observation grows only the runtime report; existing before evidence survives a final listing failure, and absent/failed/empty-candidate listing remains unknown without extra calls. Shared #700 gate/output, one atomic full-flow close because explicit protocol validation and common delivery may exceed the 100-line production shell.
