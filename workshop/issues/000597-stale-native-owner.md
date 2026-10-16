---
id: 000597
status: codecomplete
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: 'd4f512e4f7e5d5ec2859212a6735e122308369c1' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T20:32:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "2aadeb9b", done: "105f9900"}
actual_hours: 1.75
---

# Reject stale legacy native owners

## Problem

Tonight's main Couch restart parked ariadne:4, then refused its resume because legacy readiness named reused PID 1111, now an unrelated system service born before the wrapper. ariadne:5 also refuses on lsof timeouts for unrelated legacy owners under load (TL receipt 368e95f0).

## Spec

Couch Resume from disk / Relaunch retire their own attributed processes, then launch the proven transcript without global native-owner/lsof/ledger forensics or reservation admission. Cross-slot checking uses only Couch's recorded conversation identities and positively live slot registry: an exact collision refuses with its slot; unknown IDs do not refuse. Fresh never runs ownership admission and advances only its validated exact reserved thread-address marker. Failed attempts remain startable through existing Fresh replacement/archiving, preserving evidence. Ordinary standalone Pair is unchanged until #595.

## Done when

- Headless tests show stale unrelated process/readiness evidence cannot block Couch recovery; same recorded conversation in another live slot refuses, own/dead/unknown identities do not.
- Failed native relaunch followed by Fresh consumes a validated reserved address and remains retryable; busy/kill-not-confirmed safety remains.
- Targeted/race checks, build, required recorded suite and mandatory SHIP review pass; report the supported patched-build recovery command and landing hash to hub, then return to #595.

## Plan

- Remove forensic/store admission from Couch recovery only, reuse existing own-process retirement and validate registry collisions at both recovery gates.
- Keep the proven Fresh reservation fix; test exact identity, retry, false-positive controls and no forbidden scan calls.
- Verify, close and land; TL owns restage and live recovery. Deletion of dead guard code remains #595.

## Log

### 2026-10-15
- 2026-10-15: closed — Recorded full suite green on 7daa8f2b (sdlc-local/suite 9m4s). couchcore/wrapcmd/launcher/artifactpath ok. Round-2 BR-1 class fixed: launcher skips admission only for couchOwned resume (ResumeRequired) or recovery (Recovery) profiles, the two registry-checked paths; ordinary Couch spawn with --resume ID reserves (TestCouchSpawnWithResumeArgvStillReservesAdmission, RED under old gate); wrapper bypass only for a Couch-hosted pinned resume (hosted --continue still refused). Atlas couch.md updated. Earlier: both recovery gates and cold resume refuse a conversation live elsewhere, each RED-checked; Fresh retry scenario. TL approved diff and BR-1 approach.; review verdict: SHIP

Urgent TL dispatch 368e95f0 verified submitted. #595 checkpointed at eb6528a5; this independent hotfix starts from main, with no stacked code. No live process or slot changes authorized or performed.

### 2026-10-15 — clean handoff to Claude (operator c6097dd3)

Implementation stopped on request; no further guard patching. Retained code is ONLY the Fresh claim fix in createflow.go plus recovery_fresh_claim_test.go; the redirected registry-only recovery path has NOT been built, staged or landed. Earlier #595 work is separate and checkpointed.

Measurements/findings:

- Fake RED TestLegacyRetainedPIDReuseIsNotNativeOwnership: legacy Claude readiness with no agent_identity, a retained child PID now naming /System/.../mdbulkimport and parent1, still refuses with LegacyNativeOwnerError. This reproduces the operator's PID1111 reuse class. Fake RED TestLegacyChildBornBeforeWrapperIsStale: wrapper birth200.1, child100.2 (same parent number) reaches lsof and refuses; the failing spy triggered the five-second bound. Those exploratory tests were removed on the redirect, not silently marked fixed; no forensic-filter production code was added.
- Timeout scope measured: pair:3 Codex readiness scope11873b98e33bf004/tag1-pair-7 ordinal71 PID73724; parley.nvim:2 Codex scope93a33a32667021a4/tag1-parley-nvim-26 ordinal12 PID74797. Both physical launch rows have no requested ID/binding, and native argv is fresh Codex. Older Claude records have different exact IDs and already skip lsof. Current ariadne:5 Codex target is 01a136db-6cd1-7a31-87f3-271411613545. The sole production writable-lsof caller resolves truly unbound possible owners; no different-ID short-circuit bug was found. TL27676298 accepted a temporary narrowed timeout guarantee, then f4341d43 superseded it with NO forensic guard at all for Couch recovery.
- Fresh root cause: createflow previously treated FreshRequired like resume and called RegisterExistingCouchThread, requiring an established marker even though Fresh recovery reserves a new address. Retained fix first validates RecoveryAuthority, then advances ONLY matching Couch-owned recovery Fresh via existing EnsureThreadAddress(...,true); ordinary resume stays read-only. Six-case real-file regression (inferred/tagged Fresh, stale attempt, wrong owner, missing marker, reserved resume) went RED on exact original error then GREEN. Full launcher package passed59.975s and targeted checks passed. No live slot was changed.
- ariadne:4 read-only state at diagnosis: /Users/xianxu/workspace/worktree/ariadne-slot4/.couch/thread.json rev3 creating1-ariadne-51 pid77805 identity1792121584.807883 nonce start-3104d1d50c806f8d; exact marker reserved. Old1-ariadne-31 archived in .couch/archive/5771eb3eb5d9621e/1-ariadne-31.json. launch_existing.go intentionally retains post-ack uncertain recovery claims, even if the helper exits; do not blindly roll that back or edit files. After patched rollout, supported Fresh retry is couch --reboot ariadne:4 --confirm (respond to an exact stop-set confirmation if offered). There is no proven recovery NOW on the broken binary, and ordinary --resume cannot select archived31 by repo:N.

Redirected implementation contract (operator f4341d43): retire the slot's OWN process group, start its proven transcript, use ONE Couch registry check for another LIVE slot recording the same agent/conversation (in use by pair:3), NO native-owner scan/lsof/ledger forensics. Only busy, own kill unconfirmed, or same conversation live elsewhere refuse. Fresh skips ownership and archives/replaces stale failed-attempt markers through the existing validated replacement path. Remove the guard calls here; deleting their code is #595. Note for successor: current messaging Binding/SlotLiveness has slot/session identity but NO native conversation field; do not quietly substitute global ledger scans for the requested registry lookup. No registry-only code exists yet.

No recorded full suite, build, close or SHIP review for #597 yet. Operator/TL own restage and live recovery. All work committed for takeover; no permission waiver or live mutation.

## Revisions

- 2026-10-15, TL 52593bb1: include stale established registration after park-ok-resume-failed and later Fresh refusal. Read-only diagnosis first; no live mutations. Keep existing claim authority and rollback, no new recovery mechanism (ARCH-DRY). Scope remains a narrow correction while #595 replaces the guard.

### 2026-10-15 — operator redirect (TL f4341d43)

Supersedes the legacy PID/lsof patch: resume/relaunch retire the slot's own attributed processes and restart the proven transcript. Remove global native-owner/store admission calls from Couch recovery launch paths; deletion stays #595. The only cross-slot refusal is another positively live Couch registry entry recording the exact same agent/conversation ID, naming its slot. Unknown recorded IDs do not manufacture collisions. Fresh skips native ownership completely and consumes its validated reserved address; stale failed-attempt evidence is preserved/archived through existing Fresh replacement. Refusals are busy, own-process retirement unconfirmed, or exact conversation live elsewhere. Build/test this narrow path correction, no live mutation; TL restages and recovers ariadne:4/5.

### 2026-10-15 — registry-only recovery built (pair:2, Claude; TL plan approval a426e5d9)

- Guard calls removed from Couch start paths: couchcore recovery_prepared.go preflight + post-prepare gates (native-owner scan → `checkRecoveryConversationInUse`), launcher recovery_authority_os.go duplicate validation-time scan (deleted outright), createflow `ReserveNativeResume` skipped when the launch is Couch-owned; wrapper `startAgentPTY` starts a Couch-hosted resume without admission. Standalone Pair keeps admission (#595 deletes the scan code).
- Registry check: Couch thread snapshot; each OTHER record with a positively live incarnation (exact PID+identity) and the same agent → its recorded conversation via the per-slot binding (`resumeEvidence`, same as its Resume row). Same ID (established or requested) refuses "conversation X is in use by repo:N"; dead/unknown/reused-PID/unreadable never refuses. Refusals on these paths: busy, kill not confirmed (existing retirement), in use by slot X.
- Deleted dead couch glue: `Couch.checkRecoveryNativeOwner` (+ relaunch-to-migrate advice and its test) and the Scoped adapter. Production delta vs handoff 4d3318ee: +64 −72 (net −8).
- Tests: registry unit table (9 cases); stale-native-evidence integration at both gates (RED against old gate: "live wrapper native ownership is unresolved"); Couch-owned createflow resume reserves/exposes no grant (RED with gate reverted); wrapper hosted-vs-ordinary. Fixture fix: slotRecordFixture gave every slot the same conversation ID, which the new check correctly flagged in TestRecoverPlanStepsConverge; now per-slot. wrapcmd TestMain unsets COUCH_THREAD_SCOPE/TAG so ordinary-wrapper tests stay hermetic when run from a Couch slot.

### 2026-10-15 — close review round 1 (REWORK) dispositions

- BR-1 (Critical), accepted, but fixed per the brief rather than as suggested: the brief puts relaunch/resume in scope, so instead of restoring admission for ordinary Couch cold resume, `resume.go`'s cold-resume path (BuildCouchResumeLaunchProfile, used by resume/relaunch) now runs the same `checkRecoveryConversationInUse` after its binding recheck, before child effects, rolling back the start claim on refusal. Every Couch resume launch builder now passes the one registry check. Test: TestColdResumeRefusesConversationLiveInAnotherSlot (RED with the call removed: collision launched couch-fake-1). Stale comments in createflow.go and wrapcmd/native_admission.go corrected.
- BR-2 (Important), accepted: TestRecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates on the real multi-slot recover world. Preflight case asserts refusal before :3's workspace is prepared; after-prepare case makes the collision appear during preparation. Each gate RED-checked by disabling it alone. Fresh retry: new "retry after refusal" scenario in recovery_fresh_claim_test (refused attempt leaves the reserved marker byte-identical, next validated attempt launches and registers).

### 2026-10-15 — close review round 2 (REWORK) dispositions

- BR-1 class (Critical), accepted: the launcher skip was keyed to "Couch-owned", not to "the registry check ran". An ordinary Couch spawn (BuildCouchLaunchProfile) whose argv names a conversation took no admission and no check. Launcher now skips only when `couchOwned && (ResumeRequired || Recovery != nil)`, the two profiles built by the registry-checked paths; every other launch reserves. Wrapper bypass narrowed to a Couch-hosted pinned (explicit) resume with no grant; unpinned resumes still need a grant (startAgentPTY folded into startNativePTY's couchHosted parameter). Tests: TestCouchSpawnWithResumeArgvStillReservesAdmission (RED under the old gate: "handed off without a grant"), hosted `--continue` refusal case.
- Minor atlas: atlas/couch.md names the registry collision check and the admission scope.
