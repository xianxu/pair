---
id: 000706
status: codecomplete
deps: [pair#689]
github_issue:
created: 2026-10-19
updated: 2026-10-20
estimate_hours: 4.73
collaboration-mode: collaboration
card_mirror: '31bbaac1d7c303e67c4e690276869dcfb72e1e63' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T20:44:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 2.29
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

- [x] M1 — Shared layout/folder state model, compatible receipt codecs, stateful folder fake and bounded month-aware uploads with legacy checks.
- [x] M2 — Multi-folder label discovery and recorded-location restore, per-folder reporting, counted/native qualification and operator Drive check; close and land.

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
- 2026-10-20: closed — M1 and M2 SHIP; operator real Drive month upload, no-recopy retry and mixed root/month restore PASS (2 bundles, 12 files, staging absent, scratch removed). Stateful fake, installed streaming rclone odd/power-boundary and mismatch cases, real GPG mixed recovery, affected packages, race and repo guards PASS. Main integrated cleanly. Combined-head suite 4a37d05906fbb6bb09f57af823270ad0 queued; mandatory recorded pass still precedes landing under tl-worker review-overlap rule.; review verdict: SHIP
- 2026-10-20: closed M2 — Operator real Drive upload/retry and mixed root/month restore PASS at 14:18 on 8fc0f2af (2 bundles, 12 files, stage absent, scratch removed); installed streaming size/root/month/mixed regressions, affected packages, race and repo guards PASS. Noon main integrated cleanly; fresh combined-head recorded suite started and required pass precedes landing.; review verdict: SHIP
- 2026-10-20: closed M1 — BR-1 coordinator regressions RED before fix and GREEN after for successive confirmed/uncertain years and uncertain months at 99999 entries; upload/folder regression set PASS 16.533s; repolint/artifactpath PASS; prior complete archive packages PASS. BR-2 README documents M1 month layout, legacy refusal and shared per-pass budget. M2 recovery/reporting/live drill remains open.; review verdict: SHIP

- Re-read the Log, approved durable plan and TL-owned ops history-backup project; verified clean checkpoint branch and current ownership. Reported plan state to hub (submitted receipt 30bec209-1b71-4eac-b867-f389a347aafe). No new scope or design delta; #710 boundary and CI migration stash stay unchanged.
- Reproduced the two coordinator RED failures, then extracted the existing physical-folder verifier and implemented month routing. Root-namespace receipt keys now record canonical physical destinations; existing receipts avoid historical month scans. Legacy complete pairs remain at root; partial/ambiguous legacy pairs refuse before any mkdir/copy. Current-month inspection creates nothing. Pending folder batches share one byte budget and retain verified progress when another month fails.
- Added shared-budget, independent folder-failure, actual receipt namespace/location, unrelated-history and one/two-month command-count guards. With existing folders and downloaded checksum fallback, one/two active months take exactly 3+4*M rclone calls; existing 8/16 object tests defend constant per-month command cost. Targeted race checks pass.
- Found a parent-capacity hole while reviewing folder effects: EnsureChild could issue mkdir against an already-full 100000-entry parent. New regression failed with two IO calls/one mkdir; the pre-create capacity guard now refuses with zero IO. No retries, deadline changes or assertion weakening.
- Existing index/precedence fixtures assumed fresh uploads were legacy root objects, so month writes invalidated their stimulus. They now seed real legacy pairs before production adoption, preserving the whole original legacy recovery/refusal/precedence matrix; month discovery/restore tests remain M2. Full artifactpath and repolint packages pass. M1 targeted behavior passes. After correcting the legacy precedence fixture stimulus, both complete historyarchive and archivetest packages pass (69.935 s / 0.067 s). The prior package failures named exactly those four legacy precedence cases; their targeted rerun passed too. No flake classification or bypass. No real remote or operator-session mutation.

- M1 review round 1 REWORK: BR-1 found repeated child creation reused stale parent capacity; BR-2 required README layout/batching documentation at this boundary. Captured RED for confirmed and uncertain successive year creates and uncertain successive month creates. Applied one shared coordinator helper that retains capacity reservations after an uncertain effect and actual IDs after confirmation; no new durable state or IO added. Updated README inside its existing pending agent addition, preserving other review markers. Added the cache/capacity lesson and appended the plan refinement.

- M1 finalized SHIP on review round 2 with BR-1/BR-2 addressed and no new findings. SDLC committed/pushed its evidence. Continued the already-approved M2 scope; no operator/remote effects by the worker.
- M2 receipt projection RED/GREEN: LoadLabelView/SyncLocalLabelIndex now keep validated physical month locators from root-keyed receipts, preserve raw unknown receipt/object fields without rewriting them, and retain complete-observation precedence. Wrong-month/outside locators refuse. Existing #689 precedence tests remain green.
- M2 discovery uses strict two-level folder grammar, pinned before/after root/directory/pair evidence, isolated per-folder label stages and one atomic index commit. Aggregate bounds apply before download; the shared batch-refresh implementation clones/validates once and invalidates old locations in one scan, preserving the old single-refresh algorithm and unknown fields. Stateful tests cover disappearance, completed/mid-refresh rebind, second-folder partial downloads, corrupt/late labels, duplicate physical copies and metadata bounds. Legacy command count adds one fixed root re-resolution to confirm the configured binding (5 instead of 4 calls, still constant at 8/16 labels).
- M2 restore now retains deterministic recorded locations, validates layout before any remote IO, confirms pinned root/month identity, visits only selected folders and shares request bounds across groups. No recorded location requires sync-index. Corrupt preferred copies, stale identities and ambiguous inputs refuse without fallback; actual ciphertext hashes still precede every decoder. Focused restore and race checks pass.
- Per-folder reporting validates bounded additive observations, preserves Inventory-only compatibility, rejects conflicting bindings/counts and chooses final observations independently of list order. CLI and Couch warnings use actual physical destinations without inventing a tree total. Focused three-package tests, race checks and a short decoder fuzz run pass.
- Private installed-rclone child-folder conformance PASS: missing year/month creation, cached retry and immutable nested copy; no real remote or operator config used. Real GPG/zstd legacy full+literal-filtered round trip PASS; mixed legacy/month metadata-loss round trip PASS (2 bundles, 1642 ciphertext bytes, 21 raw bytes, original/member hashes agree, stage absent, retry performs no copy), all with a throwaway key. Required operator real-Drive acceptance is still pending and remains a landing gate.
- Combined historyarchive/archivetest/archivecmd packages PASS (43.508s/0.156s/0.450s), targeted Couch count tests PASS; full artifactpath/repolint PASS on the staged new sources. Full recorded repository suite, M2/close reviews and real Drive check remain outstanding.

- Parent review also found conformance-only empty root IDs were not normalized before recorded-location restore. A focused runner strips only resolution IDs; regression RED was a false binding-change refusal. Normalization now matches upload/index fallback identity, only in LocalConformance with no ID. Focused restore regressions pass (2.019s); production Drive identity requirements remain unchanged.

- Candidate PR #490 opened. Built clean in :6 at 301bd9c14d3845c1540e93de2f80febbf8c18677; bin/pair embedded vcs.revision matches and vcs.modified=false. No :0 or operator runtime changes. Final artifactpath/repolint pass; pure layout fuzz passed 10926 executions and folder-event fuzz passed 87231 executions (3s each, parallel 2).
- Started recorded suite attempt 9ec35740f7e8bf2624140cf43048b1ab on pushed candidate; it waits in normal FIFO (initial position 7), with its original runner retained. No competing suite or priority request. PR local-gates checks pass; no gate files changed.
- Operator readiness/checklist report submitted to hub (receipt 04aaef1a-01bd-4242-85d2-40f82bd83029). Hub confirms real-Drive upload/retry + root/month restore is queued in an operator side doc and will relay results. M2/whole close and landing wait for that required acceptance and the recorded suite.
- Hub cefa70f8-7f47-4e03-98dc-b04d217b0e3c confirmed #699 landed and authorized removing the obsolete workflow stash. Inspected its complete names/patch: only .github/workflows/local-gates-guard.yml and merge-check.yml. Verified timestamp-qualified reflog identity and dropped exactly cf709527564ea17b03504c0fb767c56a76d70504; no unrelated stash touched or workflow applied.

- Recorded suite 9ec35740f7e8bf2624140cf43048b1ab passed on 301bd9c14d3845c1540e93de2f80febbf8c18677 in 9m4s (log SHA-256 prefix 107b9ad08f46). This predates the operator-discovered restore regression and does not qualify the subsequent fix; a fresh recorded suite is required.
- Operator month upload succeeded, with new bundles at pair-backup:backups/pair-history/2026/10 and legacy pairs retained at root. Root-only and mixed Drive restores failed with bare ENOENT; no acceptance claimed. Private installed rclone reproduced all root/month/mixed failures with valid 3109-byte ciphertext. Debug output showed max-size 3109B rendered as 3.035KiB and excluded the file while exiting successfully; streaming reproduced max-transfer rounding too. No real remote was changed by the worker.
- TL approved removing max-size and using a coarse power-of-two transfer backstop, retaining exact listed-label size admission and post-copy size/hash verification. Added path/source/operation context to acquisition errors with wrapped causes. Installed production-path streaming root/month/mixed restores now pass; follow-up listed-size refusal and transport-boundary checks are being finalized. Old adapter tests expecting silent size filtering or an instantaneous exact transfer ceiling must reflect the explicitly revised contract. M2 close, operator retry and fresh suite remain outstanding.

- Restore fix verification: private installed production-runner streaming root/month/mixed and listed-size-mismatch-before-copy tests PASS (3.179s combined native set); actual 3109-byte and 2048-byte copies, a separate exact-1-MiB copy (0.248s), coarse-cap overflow and own postchecks exercised. Restored member hashes/sizes agree. Missing-download root/month diagnostics retain expected staged path, physical source and errors.Is(ENOENT).
- The first affected-package run identified one remaining old-contract growth assertion (it expected zero transfer bytes after a post-listing growth). Updated that same stimulus to the TL-approved contract: one extra private byte is refused by the exact postcheck, zero decoder/publication, stage absent; focused GREEN 0.319s. This is a contract revision, not a flake or test bypass. Final complete historyarchive/archivetest/archivecmd/repolint/artifactpath PASS (59.899s/0.095s/0.472s/10.335s/10.026s). Focused download/restore race PASS 23.240s. Fresh recorded suite and operator retry still required.

- Fix candidate 8fc0f2af00b396bb8fbc0649b7073af544217dab pushed to PR #490 and built clean in :6; bin/pair embedded vcs.revision matches and vcs.modified=false. Operator retry readiness report submitted to hub (fc591681-4836-4e04-8559-f3c1ff259eb9). New authoritative branch suite aeaca632bc4c3018d76919fc4926886e started on that pushed head and is waiting in normal FIFO; preserve its existing runner. Server local-gates checks pass. No :0 stage, real remote change or operator slot action.
- Read-only M2 close preflight stopped before review: measured actual unavailable, because historical suite attempt d9ec3ac06ea59f16a8bf0a86b3a49e02 has unknown slot provenance (queue evidence incomplete; actual not calibration evidence). Exact refusal sent to hub with request to restore evidence; no bypass. Then rerun sdlc actual --issue 706 as directed by the gate. Operator retry, fresh suite pass, M2/whole review and landing remain outstanding. Previous TTY recovery screen is unavailable; continuity used the durable issue/plan, Git and retained command evidence.

- Operator real-Drive acceptance PASS relayed by TL 2ea4ba73-4955-4756-93d6-3489c76ccf9b at 14:18, using the clean 8fc0f2af candidate: mixed root 83bc5df2… and month 519939f3… restore printed “Restored 2 bundle(s), 12 file(s)”. Both bundle directories contain MANIFEST.json and files; zero .pair-restore-staging remained, and scratch was removed. Earlier steps confirmed fresh uploads under pair-history/2026/10 (folder 1toBrOWx…), legacy bundles at root (1Q1lUgL…), and retry re-copied nothing. Required live acceptance is satisfied.
- Per TL, ran sdlc actual --issue 706. Exact refusal: “can't auto-compute (queue evidence incomplete; actual is not calibration evidence: suite attempt d9ec3ac06ea59f16a8bf0a86b3a49e02 has unknown slot provenance).” Only recovery offered is restoring the named evidence via TL, then rerunning actual. No supported exclusion offered: --no-actual is suggested only for all-legacy missing queue endpoints; correct-actual --exclude applies to completed calibration samples. TL found the attempt at main commit 2bb36c6f within landing-train/197 with no slot or matching landing/local-check outcome, and declined guessed provenance. Exact refusal reported for Ariadne escalation. No bypass or evidence mutation.

- TL restored d9ec3ac0… slot provenance from original train evidence: train 194, main 2bb36c6f (“Landing train 194: #622 (PR #474)”), matching pair:5 conductor agent, pre-ariadne#492 staged runner; original record retained. Reran actual successfully: 2.25h whole-issue preview, M2 gate measured its own 1.00h increment. No manual hours supplied.
- M2 review then stopped at the noon integrate gate before dispatch, naming main 1c5ee7a2f and the required git merge origin/main remedy. Canceled only this checkout's obsolete queued suite aeaca632bc4c3018d76919fc4926886e through local-check cancel, leaving other workers untouched. Main merged cleanly with no conflicts; a fresh combined-head recorded suite and review follow. Required operator acceptance remains valid for our restore behavior; combined-head automated checks qualify integration.

## Revisions

### 2026-10-19 — Make dual-layout execution and scope explicit

Preserve the original Spec and optional legacy move as a separate action. The durable plan proposes new month writes with legacy reads, root-scoped receipt storage and physical-folder locators, plus per-folder count reporting. Legacy partial/conflicting pairs are reported rather than silently duplicated or repaired. Explicit sync-index may enumerate the bounded two-level directory layout for disaster recovery; ordinary upload visits current/pending months only. #710 overlap is accounted for through generic all-catalog routing and backlog tests while its collection/scheduling and new-kind work remains separate. These refinements are proposed for the hub's approval, not implemented behavior.

- 2026-10-19: Hub approved the full plan (receipt f445934d-d16f-41e0-b552-46ff4ce15307), confirmed #706/#710 ownership boundary and operator-owned Drive check, and directed leaving the CI migration stash parked. change-code round 1 accepted architecture/scope but raised PQ-1 (function-level-test-strategy). Compressed test inventories into named risky functions with fuzz/generated sequence and independent invariant guards; no design delta. Re-running the plan gate before implementation.


### 2026-10-20 — Exact download admission and verification

TL db57f65a-d54f-4bbf-8016-f3d07bfc5ef3 approved the durable plan's restore-bound correction after the real-Drive failure: our label/listing and post-copy size/hash checks own exact acceptance; rclone gets only a coarse next-power-of-two backstop above each admitted batch, minimum 1 MiB. No request-wide instantaneous ceiling, no split requests, no source fallback or automatic repair. Root/month/mixed native streaming and odd/power-of-two size regressions defend the actual transport.
