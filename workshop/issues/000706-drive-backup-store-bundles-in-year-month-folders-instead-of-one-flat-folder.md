---
id: 000706
status: working
deps: [pair#689]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '729e4838f2462d4d218b6307b142f6e468ecd8e7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T20:44:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Drive backup: store bundles in year/month folders instead of one flat folder

## Problem

The Drive backup destination `backups/pair-history/` is one flat folder (pair#678), growing by roughly 5,000-10,000 files a year and never shrinking (retention deletes local copies only). The operator (2026-10-19): rather than wait for the 20,000-file warning (#705), put files into year/month folders now.

## Spec

New uploads go to `backups/pair-history/<YYYY>/<MM>/` (from the bundle's time range, the same year/month the local catalog path uses), keeping each file's content-hash name. The pass lists only the folders it needs (the current month, plus any month with pending bundles) instead of the whole tree, which keeps listing cost bounded as Drive grows. Existing flat-folder files stay where they are and remain valid: the upload receipts and #689's label index record each bundle's location, so restore and verification read the recorded path, old or new. Duplicate detection (the "exactly one entry with that name" rule) applies within the bundle's month folder, plus the legacy flat folder for older bundles. Optional, separate step: move the existing flat files into month folders with server-side moves, verifying by hash afterwards. Records tolerate and preserve unknown fields (ARCH-EVOLVE).

## Done when

- A fresh upload lands in `<YYYY>/<MM>/`; a legacy flat-folder bundle still verifies and restores.
- A pass lists only the needed month folders (counted test); the duplicate and uniqueness checks hold in both layouts.

## Plan

Durable plan: [Drive month folders](../plans/000706-drive-month-folders-plan.md).

- [ ] M1 — Shared layout/folder state model, compatible receipt codecs, stateful folder fake and bounded month-aware uploads with legacy checks.
- [ ] M2 — Multi-folder label discovery and recorded-location restore, per-folder reporting, counted/native qualification and operator Drive check; close and land.

## Log

### 2026-10-19

- 2026-10-19: Verified #706 unowned, claimed and ran bare start-plan. Its sibling Ariadne catch-up rebuilt tooling but generated unrelated changes to .github/workflows/local-gates-guard.yml and merge-check.yml; preserved exactly those two files in named stash cf709527564ea17b03504c0fb767c56a76d70504, then start-plan succeeded. No workflow changes or labels enter #706.
- 2026-10-19: Read #710 gaps 1–5. Month upload must include older/all-kind catalog backlog; future catalog bundles can use the same label-derived path. The existing early return on local collection failures before DriveMirror remains #710 gap 5's scheduling/policy work, not a hidden expansion of #706. No source-kind discovery, encrypted catalog bootstrap, Couch snapshots or launchd added here.
- 2026-10-19: Drafted full-flow plan with shared UTC label.From layout, root-keyed receipts recording actual folder locations, pinned child IDs, serial per-folder batches under the existing global 4 GiB cap, atomic multi-folder sync-index, recorded-location restore, raw unknown-field codecs and counted tests. Await hub approval before change-code; estimate deferred until plan-quality passes.

## Revisions

### 2026-10-19 — Make dual-layout execution and scope explicit

Preserve the original Spec and optional legacy move as a separate action. The durable plan proposes new month writes with legacy reads, root-scoped receipt storage and physical-folder locators, plus per-folder count reporting. Legacy partial/conflicting pairs are reported rather than silently duplicated or repaired. Explicit sync-index may enumerate the bounded two-level directory layout for disaster recovery; ordinary upload visits current/pending months only. #710 overlap is accounted for through generic all-catalog routing and backlog tests while its collection/scheduling and new-kind work remains separate. These refinements are proposed for the hub's approval, not implemented behavior.
