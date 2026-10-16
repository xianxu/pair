---
id: 000612
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: medium
card_mirror: '1ec7b07bd65bf0ac0cdf9f9d85969acb077af3e4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T05:47:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "79c256dc", done: "c955d608"}
---

# Warm reattach acceptance intermittently resolves native binding

## Problem

TestSwitcherWarmReattachWithoutNativeBindingReachesTerminal intermittently fails at cmd/internal/couchcmd/warm_reattach_test.go:142: warm acceptance resolved native binding 1 times (expected zero). The first observed run was #589's integrated-head full suite at 1a2e5f787b4021ff410f83896e2745f14c6b8f0c. It was the only failing test; all other packages passed, including couchcore in 548.732s. Full failure log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/a3db77884e3a91eb92e51b263f4d6a8a.log (sha256 prefix 118c2364bb97).

One automatic full-suite main baseline at d59420ed5eee5e3b94113bb8cac0c82cff5d4a59 passed. That single passing comparison was insufficient: the requested repeated whole-package matrix reproduces the same failure on both main and #589's head. The failure therefore does not require #589's changed dormant-readmission fixture.

## Spec

Establish a controlled ordering that explains the unexpected native-binding query. Preserve the warm-reattach contract: a surviving session must not need native-conversation resolution. Investigate whether the acceptance fixture's host-presence transitions disagree with the session state, or production inventory performs an inappropriate cold-binding probe while reattachment is in progress. Fix the identified owner without weakening the zero-query assertion, adding sleeps, or extending deadlines.


Confirmed ownership: the demonstrated failure is a fake/fixture defect, not evidence of a product regression. Clearing a detached session deletes sessionPresence, then SetPairSession(true) restores only the pair binding, so the contradiction persists until explicitly corrected. Production presence is independent of attached-client count. The real inventory correctly requests cold proof when the fake says SessionAbsent and has no live TTY observations yet.

Proposed quick-flow design: add one atomic fake attachment transition that removes detached-client state while retaining the existing session presence, pair binding, and pane identity. Use it in the three warm-acknowledgment fixtures (warm_reattach, continuation_acceptance and slot_operations_acceptance) and the attached-client setup in warm_attach_readonly. Cold/fresh replacement and disappearance fixtures retain their distinct semantics. Pin the transition's invariants directly and deterministically query real actionable inventory inside the warm acknowledgement before adoption (ARCH-MOCK, ARCH-ORDER, ARCH-DRY). No production session-resolution policy changes, sleeps, or deadline extensions.

## Done when

- A controlled reproduction identifies which operation issues the unexpected ResolveEstablished call and what evidence it observed.
- The fixture and production path agree on session presence across warm attachment; zero native-resolution behavior is protected by a regression that fails without the correction.
- The complete couchcmd package passes repeated runs and race verification, and a recorded full suite passes.

## Plan

- [x] Reproduce the acknowledgment transition with explicit host presence and no live TTY observations; identify the native-resolution caller and compare the production presence contract (ARCH-MOCK, ARCH-ORDER).
- [x] Confirm the responsible fake/fixture or product owner, then implement a minimal invariant-preserving correction with a regression that fails without it.
- [ ] Verify the relevant fake contracts and repeated complete couchcmd runs, race detection, and the recorded full suite; close review and report for landing.

## Log

2026-10-16: Filed unassigned by pair:1 under ops:0's explicit classification rule after ruling out a #589-only failure through this whole-package matrix. The main side used a clean detached checkout at the exact main commit, with runtimebundle generation before testing; the checkout was removed after all runs finished. No implementation or production-slot changes were made.

Commands on both commits: go test -trimpath ./cmd/internal/couchcmd -count=5; go test -trimpath -race ./cmd/internal/couchcmd -count=1.

- Head 1a2e5f787b4021ff410f83896e2745f14c6b8f0c: count=5 FAILED only the warm-reattach assertion, 441.776s; race count=1 PASSED, 163.217s. Logs /tmp/pair-589-head-package-count5.log and /tmp/pair-589-head-package-race.log.
- Main d59420ed5eee5e3b94113bb8cac0c82cff5d4a59: count=5 FAILED only the identical assertion, 414.092s; race count=1 PASSED, 164.459s. Logs /tmp/pair-589-main-package-count5.log and /tmp/pair-589-main-package-race.log.

Additional bounded controls (shuffle seed 1792152713078555000): dormant+warm pair reproduced 3/100 warm failures, 15.511s (/tmp/pair-589-fixture-pair.log); warm alone reproduced 1/100, 13.608s (/tmp/pair-589-warm-alone.log). Restoring the ORIGINAL dormant-test body from main through /tmp/pair-589-original-control-qf81jja_/overlay.json, then running ONLY warm reattach, reproduced 9/200, 33.301s (/tmp/pair-589-original-control.log). The original 30 warm-only repetitions had passed (6.005s), showing why that sample was insufficient.

Unconfirmed root-cause hypothesis from inspection: warm_reattach_test.go's AfterAcknowledge calls SetDetachedSession(address, empty), which deletes pairSessions and sessionPresence, followed by SetPairSession(address, name, true), which restores pairSessions but leaves sessionPresence absent. Background ActionableThreadInventoryContext can consume that absence without live observations yet and legitimately enter its native-binding query branch. The counter is a field on each FakeThreadArtifactCollisionChecker (newRT allocates a new instance), not process-wide. No controlled barrier or call-stack measurement has yet proved this hypothesis; the next owner should establish that ordering before choosing a fix.

2026-10-16: pair:1 claimed #612 after #589 landed (PR #363). Entered planning. A read-only parallel audit compares the production session-presence checker and fake state transitions while the main agent builds a controlled measurement. No implementation changes yet.

2026-10-16: Controlled measurement /tmp/pair-612-measure-r9x71iig/overlay.json failed deterministically with live PairSession Present=true, SessionPresence State=absent, followed by the real actionable inventory resolving exactly one native binding. Command: go test -trimpath -overlay=/tmp/pair-612-measure-r9x71iig/overlay.json ./cmd/internal/couchcmd -run '^TestWarmAcknowledgmentPresenceMeasurement$' -count=1 -v (FAIL 0.453s, intended assertion). This proves the inconsistent fake state and its consuming branch without a scheduler delay. Source audit confirms the production presence checker lists attached and detached live sessions as present; the zero-native-query contract remains correct. A bounded independent read-only audit found the same warm clear→set pattern in continuation acceptance, slot operations acceptance, and the readonly attached-client fixture; cold/fresh uses and disappearing-session tests are distinct. Proposed atomic fake transition and deterministic acknowledgement coverage sent to the TL for design approval before implementation.

2026-10-16: Verified TL dispatch receipt 17dc19d1-c52e-4c0e-b10b-3b6c08c849ae explicitly authorizes the small plan to proceed after posting. Entered quick flow via change-code. Added a real inventory call inside warm acknowledgment before terminal adoption: original setters now deterministically fail the acceptance test (1.484s; retry produced 2 native-binding resolutions; /tmp/pair-612-deterministic-red.log). New all-index-claims regression also fails with both present and absent second owners under the old detached-only counting (/tmp/pair-612-claims-red.log). Implemented AttachSession as a mutex-protected removal of client-free state only, retained ownership from the shared pair-session index projection, migrated exactly the four planned warm-fixture sites, and added preservation/idempotence/no-creation/duplicate-claim contracts. Production presence and cold-policy code unchanged.

2026-10-16: Focused fake/public warm attach/continuation/socket acceptance checks PASS (couchcore 26.254s, couchcmd 23.769s); deterministic warm acceptance PASS 100 repetitions (8.158s). Strengthened the attachment contract to observe the attached session without a duplicate claimant first; an overlay that turns AttachSession into a no-op FAILS at that exact assertion (0.245s; /tmp/pair-612-noop-mutant.log), healthy contracts PASS (0.249s). Full couchcmd count=5 and race once were started for post-fix verification.

2026-10-16: Complete couchcmd race check PASS (118.578s; /tmp/pair-612-package-race.log). Complete couchcmd count=5 PASS (307.952s; /tmp/pair-612-package-count5.log). Recorded suite running on 82dbf4cc98600f824ba08f556f64920438a7ba25 (PR #370). Later local commits are documentation only.

## Revisions

2026-10-16 — Root-cause classification: controlled host-evidence measurement and production-contract audit establish fake/fixture ownership. Refine the correction from an open investigation to an atomic attached-client transition preserving session presence/binding/panes, with the enumerable warm-fixture family swept in the same patch. Production admission and recovery policy stay unchanged.

2026-10-16 — Retained ownership across attachment: the independent audit found DetachedSessions counts ownership from detachedSessions, unlike ListedPairSessionContext and production, which count index associations. The proposed attachment transition must not erase a surviving ownership claim. Derive detached-query claims from all pairSessions names (including non-present bindings), sharing the existing listed-query derivation where practical; cover a duplicate claimant that attaches, a claimant created only through SetPairSession (present and absent), and an unrelated unique detached session. No additional attached-state map. Sent this refinement to the TL while the design response remains pending.
