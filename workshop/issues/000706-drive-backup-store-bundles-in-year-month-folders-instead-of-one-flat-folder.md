---
id: 000706
status: working
deps: [pair#689]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 4.73
collaboration-mode: collaboration
card_mirror: '2daebdaefa6749b932ca308dc827431d4cddc18e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T20:44:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Calibration is stale, so the result is provisional. The approved thorough plan earns the 0.2 design discount and 15% design buffer; familiarity is 1.0 after #678/#689. Implementation values below already use v3.1's 40% scaling; no second scale.

Library check: standard time/path/JSON plus existing extensible-object helpers cover the layout mechanics, so the greenfield layout's 1.25 h design midpoint is halved, then multiplied by 0.2 = 0.125. No library supplies the production folder event/effect model: keep its full 1.25 midpoint times 0.2 = 0.25. Rclone API integrations each use the 2 h design midpoint times 0.2 = 0.4. Smaller extensions use 0.15 times 0.2 = 0.03; index multi-file wiring uses cross-cutting midpoint 0.6 times 0.2 = 0.12. Docs use 0.125 times 0.2 = 0.025. Three review boundaries use 0.1 times 0.2 = 0.02 design and 0.35 times 0.4 = 0.14 implementation each. One external rclone discovery/qualification budget is 0.45 times 0.4 = 0.18 implementation. No novel stack, cross-repo propagation or scope-pivot allowance. Operator scheduling waits are excluded.

Rows map respectively to layout, folder transitions, receipt codecs, folder adapter, hierarchy fake, upload coordinator, index integration, restore acquisition, folder reports, documentation, M1/M2/close reviews and rclone conformance. Design subtotal 1.5 h, implementation subtotal 3 h; total 1.5*1.15+3 = 4.725 h, rounded 4.73 h.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.125 impl=0.22
item: greenfield-go-module design=0.25 impl=0.22
item: smaller-go-module design=0.03 impl=0.14
item: api-integration design=0.4 impl=0.4
item: smaller-go-module design=0.03 impl=0.2
item: api-integration design=0.4 impl=0.6
item: cross-cutting-refactor design=0.12 impl=0.2
item: smaller-go-module design=0.03 impl=0.2
item: smaller-go-module design=0.03 impl=0.14
item: atlas-docs design=0.025 impl=0.08
item: milestone-review design=0.02 impl=0.14
item: milestone-review design=0.02 impl=0.14
item: milestone-review design=0.02 impl=0.14
item: real-api-discovery design=0 impl=0.18
design-buffer: 0.15
total: 4.73
```

## Log

### 2026-10-19

- 2026-10-19: Verified #706 unowned, claimed and ran bare start-plan. Its sibling Ariadne catch-up rebuilt tooling but generated unrelated changes to .github/workflows/local-gates-guard.yml and merge-check.yml; preserved exactly those two files in named stash cf709527564ea17b03504c0fb767c56a76d70504, then start-plan succeeded. No workflow changes or labels enter #706.
- 2026-10-19: Read #710 gaps 1–5. Month upload must include older/all-kind catalog backlog; future catalog bundles can use the same label-derived path. The existing early return on local collection failures before DriveMirror remains #710 gap 5's scheduling/policy work, not a hidden expansion of #706. No source-kind discovery, encrypted catalog bootstrap, Couch snapshots or launchd added here.
- 2026-10-19: Drafted full-flow plan with shared UTC label.From layout, root-keyed receipts recording actual folder locations, pinned child IDs, serial per-folder batches under the existing global 4 GiB cap, atomic multi-folder sync-index, recorded-location restore, raw unknown-field codecs and counted tests. Await hub approval before change-code; estimate deferred until plan-quality passes.

- Implementation admitted after CLEAN plan-quality review; estimate adopted at 4.73 h. Component tests are embedded in their implementation rows; no actual hours guessed. M1 remains open.
- Context checkpoint: completed shared UTC layout and folder event/effect model; generated sequence/fuzz invariant tests pass. Receipt/object codecs preserve opaque fields with bounded strict known-field validation and legacy-byte compatibility. Hierarchical fake and pinned child-folder adapter completed; focused fake/race, receipt and adapter tests pass. New ordinary source paths registered in artifactpath.
- Upload behavior RED captured unchanged production coordinator: TestDriveMirrorWritesMonthWithoutListingUnrelatedHistory fails because 2025/12 is absent (still writes flat); TestDriveMirrorDoesNotDuplicatePartialLegacyPair fails because current writer completes the legacy pair. TestDriveMirrorAdoptsLegacyPairWithoutMigration already passes. No coordinator edits yet; these failures are expected TDD evidence, not a passing M1.
- NEXT: extract existing per-folder upload verification from DriveMirror.Run and add the outer coordinator: project all catalog candidates once, root-namespace receipts recording actual physical destinations, legacy pair adoption/conflict refusal, only current/pending months visited, pinned cached parent inventories, one shared 4 GiB budget across serial folder batches. Preserve exact final-list verification, unknown fields and local-backup independence. Then targeted package/artifactpath checks and M1 review; M2 sync-index/restore/reporting and operator Drive drill remain outstanding. No PR/full suite or real remote mutation yet. CI migration stash cf709527564ea17b03504c0fb767c56a76d70504 remains parked per hub.

### 2026-10-20 — resumed after #726

- Re-read the Log, approved durable plan and TL-owned ops history-backup project; verified clean checkpoint branch and current ownership. Reported plan state to hub (submitted receipt 30bec209-1b71-4eac-b867-f389a347aafe). No new scope or design delta; #710 boundary and CI migration stash stay unchanged.
- Reproduced the two coordinator RED failures, then extracted the existing physical-folder verifier and implemented month routing. Root-namespace receipt keys now record canonical physical destinations; existing receipts avoid historical month scans. Legacy complete pairs remain at root; partial/ambiguous legacy pairs refuse before any mkdir/copy. Current-month inspection creates nothing. Pending folder batches share one byte budget and retain verified progress when another month fails.
- Added shared-budget, independent folder-failure, actual receipt namespace/location, unrelated-history and one/two-month command-count guards. With existing folders and downloaded checksum fallback, one/two active months take exactly 3+4*M rclone calls; existing 8/16 object tests defend constant per-month command cost. Targeted race checks pass.
- Found a parent-capacity hole while reviewing folder effects: EnsureChild could issue mkdir against an already-full 100000-entry parent. New regression failed with two IO calls/one mkdir; the pre-create capacity guard now refuses with zero IO. No retries, deadline changes or assertion weakening.
- Existing index/precedence fixtures assumed fresh uploads were legacy root objects, so month writes invalidated their stimulus. They now seed real legacy pairs before production adoption, preserving the whole original legacy recovery/refusal/precedence matrix; month discovery/restore tests remain M2. Full artifactpath and repolint packages pass. M1 targeted behavior passes. After correcting the legacy precedence fixture stimulus, both complete historyarchive and archivetest packages pass (69.935 s / 0.067 s). The prior package failures named exactly those four legacy precedence cases; their targeted rerun passed too. No flake classification or bypass. No real remote or operator-session mutation.

## Revisions

### 2026-10-19 — Make dual-layout execution and scope explicit

Preserve the original Spec and optional legacy move as a separate action. The durable plan proposes new month writes with legacy reads, root-scoped receipt storage and physical-folder locators, plus per-folder count reporting. Legacy partial/conflicting pairs are reported rather than silently duplicated or repaired. Explicit sync-index may enumerate the bounded two-level directory layout for disaster recovery; ordinary upload visits current/pending months only. #710 overlap is accounted for through generic all-catalog routing and backlog tests while its collection/scheduling and new-kind work remains separate. These refinements are proposed for the hub's approval, not implemented behavior.

- 2026-10-19: Hub approved the full plan (receipt f445934d-d16f-41e0-b552-46ff4ce15307), confirmed #706/#710 ownership boundary and operator-owned Drive check, and directed leaving the CI migration stash parked. change-code round 1 accepted architecture/scope but raised PQ-1 (function-level-test-strategy). Compressed test inventories into named risky functions with fuzz/generated sequence and independent invariant guards; no design delta. Re-running the plan gate before implementation.
