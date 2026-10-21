---
id: 000774
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: '5a4b5c5c7e852629ac1e11fef9b074d02a6008dd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T08:53:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6d28cfe7", done: "a9d7dd71"}
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

- [ ] Send the cause-specific plan to the hub before implementation.
- [ ] Reproduce the device change, implement the approved narrow recovery, and qualify negative controls and same-class consumers.
- [ ] Run recorded checks, close review and land; coordinate live fresh-fact verification with the hub.

## Log

2026-10-21: filed and claimed by pair:7 per ops:0 after #736 start-plan refused continuing its already-landed M1 branch. No dependency: M1 code is already on main, and #736 is to close around the verified logger-health instrumentation. This cause fix has priority over parked #773 (draft PR525, pushed checkpoint c4f5b1990; no own runner). Direction is approved, detailed plan still due before code.

## Revisions

2026-10-21 09:05 PDT — Hub approved the persisted-versus-live boundary and conservative interrupted-transaction limit; durable plan: workshop/plans/000774-diagnostic-device-rebind-plan.md. #736 closure SHIP/PR526 is parked per hub: queued suite cancelled by supported lease, waiting merge interrupted per contract, deferral pushed on its own branch. #774 starts from fresh main, no stacking or second checkout.

2026-10-21 — Implementation checkpoint: measured Open fixture RED with diagnostic reservation identity or size changed; retained preview RED with generation changed outside protocol (0.234s). Shared persisted classifier and locked current publication made both GREEN. Strict-live diagnostic RED (device refusal lacked its cause), then transaction-name RED for rotation/deletion/legacy append; named strict recovery now GREEN. Final focused cohort including current read-only preview, creation refusal, unknown-tail/intent/unsafe-path controls and failed checkpoint sync passed13.323s before final explicit Appending guard. Original reservation bytes and payload prefixes verified unchanged.

Ad hoc bounded peer audit caught current collection mistakenly sharing retained rebind policy and mismatch errors being discarded. Both corrected: current stays strict within this invocation; retained metadata alone is rebound before a new deletion intent, and refusal includes checkpoint reason. Package run, targeted race, recorded check and boundary review still pending; no live-write success claimed. #744 remains unlanded, #773 and #739 native checks stay held.
