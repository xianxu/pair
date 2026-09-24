---
id: 000315
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T20:26:43-07:00
flow: {kind: full, provenance: inferred}
actual_hours: 0.48
---

# Allow fresh slot launches to establish their reserved conversation

## Problem

pair:2 fresh-slot times out because Pair rejects the newly reserved conversation ID as not already established.

## Spec

StartFreshSlot allocates a new conversation ID. After resolving saved preferences,
validate fresh arguments before any claim or metadata replacement. Build the
ordinary Couch launch profile and call launchTrackedThread without Fresh or
Resume. Pair uses its existing EnsureThreadAddress reserved-to-established
transition and Couch's awaitThreadRegistration, as ordinary creation does.

Remove the superseded #315 RegisterFreshCouchThread runtime method, LaunchNonce
fields/validation/transport, and their dedicated tests. Restore pre-#315 same-ID
replacement behavior; switch-agent and checkpoint flows retain their existing
orientation/readiness protocol. Keep slot live-owner refusal, atomic replacement,
archival, preferences and directories. This introduces no new lifecycle states,
operations or durable artifacts (ARCH-DRY/PURPOSE/FUNERAL).

## Done when

- A fresh slot allocates a new conversation ID and uses ordinary Pair creation/registration, without an established-conversation prerequisite or special readiness nonce.
- Remove the #315-only RegisterFreshCouchThread operation, launch_nonce profile field and nonce forwarding; same-ID resume/switch/continuation behavior stays unchanged.
- Reject restoration arguments before replacing current metadata; retain live-owner refusal, conversation history, preferences and workspace files.
- A composed fresh-slot regression runs the real launcher and real claim storage through the same harness as ordinary creation.

## Plan

- [ ] Parameterize TestSpawnComposesProductionPairRegistrationBoundary in couch_test.go for ordinary and fresh-slot starts. Run real LaunchNative and claim storage; leave FreshRegistration nil and assert neither FreshRequired nor ResumeRequired in the emitted profile. Verify reserved-to-established promotion and final saved launch profile.
- [ ] In slotrecovery.go StartFreshSlot call ValidateFreshAgentArgs immediately after slotLaunchProfile, before claims/current replacement; use BuildCouchLaunchProfile and omit Fresh:true. Keep TestSlotFreshInvalidPreferencePreservesCurrent and live/unresolved-owner/concurrent-mutation tests as guards.
- [ ] Restore #315-only edits in launcher/runtime.go, osruntime.go, thread_claim.go, args.go, launch_args_policy.go, createflow.go and couchcore/launch_existing.go to their pre-fix contracts. Remove new runtime test methods, fresh_slot_claim_test.go, slot_fresh_nonce_test.go and added nonce tests; the composed test now guards the actual slot path.
- [ ] Run the composed test, slot recovery/preference tests, launcher suite and affected race tests; build pair and couch and inspect diff. Retain existing same-ID fresh/resume/continuation tests without weakening their assertions.
- [ ] Rewrite atlas/couch.md's #315 section to document ordinary new-ID launch; scan production/docs for retired RegisterFreshCouchThread/LaunchNonce/launch_nonce symbols. Append project scope revision and final test evidence, close, publish; operator restarts Couch for a live fresh-slot trial.

## Log

### 2026-09-23
- 2026-09-23: closed — New-claim and missing-nonce regressions both failed before fixes and passed after. Launcher suite 19.127s; targeted launcher/core race suites 1.686s/15.935s; make pair bin/couch and diff check passed. Real claim files and readiness reader exercised. First full core suite passed 237.467s; final core/CLI rerun underway. Upstream bootstrap gateway aff72f82 unchanged by this task.; review verdict: SHIP
- 2026-09-23: flow upgraded quick → full — 207 added lines in code files (limit 100)

Read-only diagnosis plus isolated real-binary probe reproduced: existing Couch thread registration does not match requested established address. Live tag couch-d41b8dc708afd578 has a reserved claim and exited helper. Original setup error is separate.

## Revisions

### 2026-09-23 — preserve checkpoint replacement policy

The fresh-registration operation is only for exact Couch-owned fresh launches without a checkpoint. Existing checkpoint continuation retains its established-only registration contract. This avoids broadening continuation while fixing new slot launch.

### 2026-09-23 — implementation evidence

New real-filesystem launcher regression failed on the reserved case before implementation and now passes. Established fresh, missing/malformed/wrong-identity claims, unowned reservations and resume refusal are covered. Existing checkpoint reserved rejection remains passing. Launcher suite passed (13.448s); affected race tests passed (1.650s). make pair bin/couch passed. Isolated real-binary probe at pair:2 progressed from claim rejection to the fake Zellij handoff, and the exact reservation became established; no real agent was started. Live metadata is at pair-slot2/.couch, not pair-slot2/pair/.couch. Operator invited to retry fresh-slot using rebuilt binary.

### 2026-09-23 — review window provenance

The branch carries prior Add slot work (#313). Bootstrap/CI gateway changes in the broad boundary window came from `aff72f82` (build: adopt ariadne#239 seeded gateway files), already an ancestor of origin/main; this issue does not change those files against origin/main. Review #315 on its launcher registration change and real-claim regression. Preserve unrelated upstream work.

### 2026-09-23 — live retry exposed second handshake mismatch

Claim registration succeeded and Claude started, but Couch timed out because its transaction nonce was not sent in a plain fresh profile. Pair only reused Orientation.Attempt; without orientation it minted an unrelated nonce. Stop publication and extend this issue: carry an explicit LaunchNonce in trusted profile/LaunchArgs, inject for every Couch fresh launch, and require agreement with orientation when both exist. Ordinary/resume profiles reject an explicit fresh nonce. Preserve fallback for existing orientation-only profiles. Test sender/profile/launcher/real-ready reader, then rebuild both binaries. Existing live pair:2 should be recovered, not replaced just to test.

### 2026-09-23 — full handshake fix

Initial full Couch core suite passed (237.467s), but live retry exposed the missing nonce transport. New sender and launcher tests both failed before the nonce change, then passed through profile decoding and the real readiness reader. Added nonce/Orientation disagreement and non-fresh rejection checks. Both binaries rebuilt. Running Couch must restart for the sender change. Current live Claude session remains untouched pending operator recovery choice.

### 2026-09-23 — final verification and recovery

Final suites passed: launcher 19.127s, couchcore 353.928s, couchcmd 46.399s; targeted race launcher 1.686s and core 15.935s. Boundary review SHIP, no findings. Operator authorized stopping the unused failed-start session: exact index mapped tag couch-9860e667eaa561bb to 📁pair-couch-2; zellij kill-session succeeded and server/wrapper/editor/Claude PIDs were all absent afterward. Couch restart and fresh-slot live retry remain operator smoke steps.

### 2026-09-23 — operator-approved simplification supersedes handshake repairs

The user requested fewer states and combinations. The preceding fresh-registration
and explicit nonce extensions are superseded: StartFreshSlot mints a brand-new
address, so use BuildCouchLaunchProfile and ordinary StartSpawn registration.
ValidateFreshAgentArgs moves before ownership/claim/metadata work. Remove the
new registration runtime method and launch_nonce transport; restore the existing
same-ID replacement protocol. Retain slot directory ownership, atomic current
replacement/history and saved preferences. ARCH-DRY/PURPOSE: reuse the existing
new-conversation lifecycle instead of expanding the same-ID replacement state
space. Reuse the production launcher integration harness for both entry points.

### 2026-09-23 — plan review reconciliation (PQ-1/PQ-2/PQ-3)

Updated active Spec and Plan to the approved simplification; earlier handshake
spec remains preserved by the preceding revision entries and Git history.
Named production functions, test guards, deletion targets and atlas/project
updates explicitly. The normal-versus-slot composed test fails on the old
FreshRequired profile, confirming it detects the wrong path before implementation.
