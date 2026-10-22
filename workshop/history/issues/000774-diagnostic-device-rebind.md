---
id: 000774
status: done
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: 'ccbc92a8a4913e3a95d5f75805b7f0be701418ef' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T08:53:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.80
---

# diagnostic checkpoints survive device remounts

## Problem

Couch supervision logging failed at fresh startup after remount/reboot because persisted st_dev is transient. At the operator-coordinated 2026-10-21 08:44 restart, resources reported unavailable, offered=77 written=0 dropped=77, last write never; open error: diagnostic reservation identity or size changed. Instrumentation from #736 M1/PR502 exposed the actual case.

Read-only bracketed metadata observations were stable: supervision.jsonl actual device16777229, inode728006625, size26061670, mtime_ns1792450564739652454. state.current records device16777232 with identical inode, size and nanosecond mtime. A retained old reservation describes the previous committed append: before size26055958 plus length5712 equals current26061670. No payload contents were read or live files/processes changed.

recoverReservedAppend first rejects checkpoint matching because of device alone, then attempts recovery using the already-committed old reservation and refuses. Saving the current writer registration before recovery explains the fresh PID/state.json timestamp despite Open failure. This is not measured truncation, unknown tail or AMFI.

## Spec

Approved direction from ops:0: a persisted checkpoint identity must not depend on transient st_dev, or accept a device-only change when inode, size and nanosecond mtime exactly match. Plan the smallest cause-specific repair under the existing diagnostic lock. Preserve all bytes and existing generation/append/retention authority; do not blindly reset metadata or certify unknown tails. Refusals must name their actual mismatched component/recovery cause. No new retry loop, durable store or supervision admission dependency.

Audit the same persisted-identity class across current and retained generations; retain distinct refusal for inode, size or timestamp changes, unsafe paths and uncertain append outcomes. Keep persisted format unchanged if possible; any actual codec change must preserve unknown fields per ARCH-EVOLVE.

## Done when

- A deterministic fixture representing the measured already-committed append, stale reservation and device-only checkpoint mismatch fails before the fix, then opens and appends normally with original bytes preserved.
- Identical inode/size/nanosecond mtime is required for the device-only recovery; replacement, shrink/growth, timestamp mismatch and unsafe-path controls preserve fail-closed behavior.
- Recovery/refusal evidence explains the relevant identity mismatch; normal logging remains optional and bounded, with no hot-loop retry or replay of dropped facts.
- Required checks and review pass; the hub coordinates rebuilding/restarting Couch and verifies new supervision facts reach storage. No worker mutation of the live diagnostic store.

## Plan

- [x] Send the cause-specific plan to the hub before implementation.
- [x] Reproduce the device change, implement the approved narrow recovery, and qualify negative controls and same-class consumers.
- [x] Submit the pushed head to recorded verification and prepare close/landing; hub owns the remaining live check.

Remaining acceptance gates: recorded suite must PASS before landing; close review, normal sdlc merge and hub-coordinated restart/write confirmation. Queued is not passed.

## Log

- 2026-10-21 13:08 PDT — Infrastructure interruption, not test evidence: corrected branch suite b5c731deb7f01cf15c9f1100770c9a41 passed10m34s on87b7c4fb39cf449a64224f8ced70241b2cd1a3a5. Merge integrated main to cd0872c06c8504aef472cd66c4bdb3e68f3bd7c1. Integrated attempt6c8c05bd08a636459168afac3eb528f5 failed23s with Go cache ENOSPC under /Volumes/AgentScratch/go-build; automatic main baseline1cba845146dd1c537fdce15fbb15231b failed1s for the same infrastructure cause. Neither is evidence against the code or a registered flake. Failed logs are retained under the Git common directory sdlc/test-stats/failed with those attempt IDs. Hub restored the default SSD cache; verified go env GOCACHE = /Users/xianxu/Library/Caches/go-build. Retried sdlc merge --yes on unchanged integrated head, without overrides: it printed "landing train: not ready: CI gate: CI gate: PR #528 has failing checks on its head". gh pr checks confirms the only failing status is sdlc-local/suite: unattributable-failure, exit2 in23s, log hash9166bc50e983; hosted conformance, merge-check and local-gates checks pass. Stopped this waiting runner cleanly (Ctrl-C, exit130). No replacement suite ran, no manual status, flake registration or guard workaround. Await supported infrastructure-attempt recovery from the hub. This Log commit documents the failure; it does not authorize retrying on a new documentation head to evade the same-head guard. Hub-coordinated advancing-writes live acceptance remains owed.





- 2026-10-21: closed — Post-SHIP suite found branch-only exhaustive artifact source inventory omission for checkpoint_identity.go. Added its exact metadata-only NonArtifactSources classification alongside existing diagnostic helpers; no known-violation waiver or scanner change. Recorded RED followed by targeted TestProductionArtifactReferencesAreExactlyClassified PASS1.633s, diff clean. Original full suite is still completing all other packages; corrected head requires new recorded pass before merge. Waiting merge runner interrupted before this change. Hub restart/write acceptance remains owed; hub owns project. (fixed-then-ship); review verdict: SHIP
- 2026-10-21: closed — BR-1 real preview-to-deletion regression RED0.229s (replacement deleted) then strict snapshot carry GREEN; final focused race cohort PASS3.120s including all device/tail/pending/unsafe/sync controls; inventory PASS. Reuses existing Segment memory projection and reader-proof test seam, no format/stat hook or new retry. Corrected pushed code f05fa3182 now submitted for a replacement recorded affected/full suite; pending, landing still requires pass. Hub approved post-land coordinated restart to verify advancing writes and owns project. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-21: flow upgraded quick → full — 107 added lines in code files (limit 100)
2026-10-21: filed and claimed by pair:7 per ops:0 after #736 start-plan refused continuing its already-landed M1 branch. No dependency: M1 code is already on main, and #736 is to close around the verified logger-health instrumentation. This cause fix has priority over parked #773 (draft PR525, pushed checkpoint c4f5b1990; no own runner). Direction is approved, detailed plan still due before code.

## Revisions

2026-10-21 09:05 PDT — Hub approved the persisted-versus-live boundary and conservative interrupted-transaction limit; durable plan: workshop/plans/000774-diagnostic-device-rebind-plan.md. #736 closure SHIP/PR526 is parked per hub: queued suite cancelled by supported lease, waiting merge interrupted per contract, deferral pushed on its own branch. #774 starts from fresh main, no stacking or second checkout.

2026-10-21 — Implementation checkpoint: measured Open fixture RED with diagnostic reservation identity or size changed; retained preview RED with generation changed outside protocol (0.234s). Shared persisted classifier and locked current publication made both GREEN. Strict-live diagnostic RED (device refusal lacked its cause), then transaction-name RED for rotation/deletion/legacy append; named strict recovery now GREEN. Final focused cohort including current read-only preview, creation refusal, unknown-tail/intent/unsafe-path controls and failed checkpoint sync passed13.323s before final explicit Appending guard. Original reservation bytes and payload prefixes verified unchanged.

Ad hoc bounded peer audit caught current collection mistakenly sharing retained rebind policy and mismatch errors being discarded. Both corrected: current stays strict within this invocation; retained metadata alone is rebound before a new deletion intent, and refusal includes checkpoint reason. Package run, targeted race, recorded check and boundary review still pending; no live-write success claimed. #744 remains unlanded, #773 and #739 native checks stay held.

2026-10-21 — Targeted verification returned: diagnosticlog package PASS152.224s on the initial repair variant (before final transaction-name/creation controls), final focused race cohort PASS4.268s, final focused behavior cohort PASS13.323s, make test-affected-inventory PASS and diff check clean. PR528 created. Pushed code24378534fb56d364d708e389bbd963e6362958e1 is now in recorded affected's full-suite fallback, lease-2340917264 / attempt667b15abdf34e59ee63442d076ba30da, queued seventh when observed. Full-suite result is PENDING; no close/landing/live acceptance claimed. Boundary review may overlap this queue per tl-worker; landing still requires the pass.

2026-10-21 — Close round1 REWORK/BR-1 (live-observations-require-strict-identity): verified retained preview's first observation was discarded. New real filesystem transition regression RED0.229s: collector deleted a replacement after reader-proof changed the file and matching persisted metadata. Corrected by carrying preview's private observed generation and strictly comparing before retained metadata write/deletion intent. Final focused race cohort GREEN3.120s. Live device changes are separately pinned by synthetic stat control. No format/authority expansion. Old recorded lease-2340917264 cancelled while queued (exited, never acquired); final code must get a new recorded run.

2026-10-21 — Recorded final-code attempt3965e30dc21fb638d437a8f1c87e181f / lease-3617626155 acquired after4956s FIFO. Script/editor checks passed (upstream-provider fixture explicitly skipped unavailable source). Go suite found branch defect TestProductionArtifactReferencesAreExactlyClassified: checkpoint_identity.go missing from exhaustive production-source inventory. Kept the full run running to collect all remaining outcomes; no early stop or flake classification. Interrupted only this slot's waiting merge runner per documented Ctrl-C contract before the correction. The new helper compares supplied metadata, names no artifact path, and is now exactly listed in NonArtifactSources alongside existing diagnostic metadata helpers; no known-violation exemption or scanner relaxation. Targeted inventory test GREEN1.633s after recorded RED. Final corrected head needs fresh recorded verification; old suite still pending final result.
