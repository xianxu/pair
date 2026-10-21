---
id: 000767
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'e98eb73d4229acc8d450cf7a93bfb98592cfd9e3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T05:33:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestProvisionFixtureIdentityUsesOneGitProcess]
flow: {kind: quick, provenance: inferred, spec: "d4a43449", done: "50383ae7"}
---

# flake: TestProvisionFixtureIdentityUsesOneGitProcess

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestProvisionFixtureIdentityUsesOneGitProcess: Same exact head ecef5f7af3515fead7c02d6464ae112b16498fe1: recorded full-suite attempt53424823b0e2ed3099f5dab5c5a415b2 failed only TestProvisionFixtureIdentityUsesOneGitProcess at provision_identity_fixture_test.go107 exit42 (raw log .git/sdlc/test-stats/failed/53424823b0e2ed3099f5dab5c5a415b2.log). After failure, go test ./cmd/internal/couchcore -run ^TestProvisionFixtureIdentityUsesOneGitProcess$ -count=1 -v passed; independent count10 passed10/10 in3.062s. Exit42 is exclusively the shim test -r /dev/fd/3 gate before realGit; Lease forwarding and test unchanged from origin/main. Underlying transient child-FD readability cause remains unproven; add child fstat/fcntl diagnostics before a fix.

## Spec

Use a child builtin read on FD3 plus an exact per-lease token to prove inheritance before executing real Git. Write/rewind the parent-owned lease and retain negative controls for absent/wrong-token descriptors. A mode000 open lease deterministically rejects the old pathname-permission oracle while the already-open descriptor remains usable. Extend Pair’s existing golint catalog with a bounded string-literal guard for shell permission tests on /dev/fd; document its lexical scope.

Investigate the inherited lease through OSProvisionIO and the identity fixture’s shell shim on macOS. Correct the evidenced root cause, preserving exactly one Git observation and meaningful inherited-lease proof; no retries, timeout increases or suppressed failures. Keep #752 parked and resume it first when the TL releases staging. ARCH-PURPOSE: a green repetition alone is not a root-cause fix.

## Done when

- An observed child failure distinguishes descriptor loss from a false pathname readability check, with retained diagnostics.
- A controlled regression fails before the correction; corrected native repetitions meet or exceed the reproducing sample count.
- Identity still performs exactly one Git process and the child proves the exact inherited lease; missing/wrong descriptors are rejected.
- Targeted tests, required inventory, recorded affected check and close review pass; closing retires the flake registration.

## Plan

- [x] Reproduce with child-side descriptor diagnostics and identify the failing boundary.
- [x] Add a controlled regression, correct the root cause and encode the mistake class in Pair’s composed golint catalog.
- [x] Verify native repetitions, the identity test family, lint and audited reader inventory.

Recorded-suite completion, close review and landing remain gates tracked in Log; the final recorded pass precedes landing.

## Log

### 2026-10-21
- 2026-10-21: closed — Native old gate false-negative1/14969 with immediate correct FD3 read; direct-token probe passes16000 starts twice, retained counter run attempts=16000 failures=0. Deterministic mode000/wrong-token RED then all4 controls GREEN. Identity-family count100 passes101.617s; full repolint8.623s; audited reader inventory/diff pass. Recorded full fallback lease-1363786489 on681c7986 queued and must pass before landing, review may overlap per tl-worker. Atlas local catalog updated; TL owns project. No timeout or production FD lifetime changes. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

- 2026-10-21: Authenticated TL dispatch4dfe9de6-e6f4-4d8a-bce5-31ab01faa1c4; preserved #752 branch and switched through main-slot9 to fresh main-based #767 branch via claim/start-plan. Unchanged identity test passed100 repetitions in28.616s. Parent lease is cleanup-owned and forwarded unchanged into ExtraFiles; diagnosis must distinguish shell pathname permission from actual inherited descriptor usability.

- 2026-10-21: Native scratch probe /Volumes/TestScratch/tmp/pair767-fd-9442s0hq reproduced1 false negative in14969 starts (worker0 sample968, seven other workers2000 each). After the old test -r failed, builtin read on FD3 returned lease-token with rc0; subsequent stat showed mode100600 uid501. This rules out early close/close-on-exec for the observed failure and identifies the pathname oracle as unsound. Apple Bash test.c calls sh_eaccess→sh_stataccess for /dev/fd, and XNU devfs_fdesc_support.c obtains attributes through a separate pathname/vnode path; direct read avoids that unnecessary permission observation. TL quick-plan receipt21ca07a9-38cf-4744-be1e-4a857d18ac9b.

- 2026-10-21: Controlled RED: mode000 inherited lease rejected with exit42, wrong-token readable lease incorrectly accepted. GREEN: builtin direct-FD token read passes readable/mode000, rejects absent/wrong-token before real Git; accepted identity calls each append exactly one Git invocation. Lint RED3/3 permission-probe fixtures then GREEN6/6 including direct read and unrelated paths. Full repolint passes8.623s after adding the new source to Git; identity-family100 repetitions pass101.617s. Prior unchanged identity+lease-family50 repetitions passed152.632s. Inventory passed after auditing the changed reader: only local temporary lease/call log and fixture sources, no new repository traversal; refreshed only its certificate.
- Native corrected probe completed16000/16000 starts with zero failures (8×2000), exceeding14969 observed old-gate starts. Retained source and diagnostic limitations in workshop/evidence/000767. Strengthened retained probe to count attempts/failures and return nonzero on any child failure; a second16000-start pass is running before the final checkpoint. No production OSProvisionIO lifetime or timeout changes.

## Revisions

### 2026-10-21 — root-cause refinement

Native evidence narrowed the fix from possible FD lifetime repair to the test’s pathname-permission oracle. Added direct inherited-token proof, deterministic mode000 and absent/wrong-token controls, a local catalog rule and a retained native probe. Scope remains the original flake; #752 staging has priority when released.

- 2026-10-21: Retained counter-enforcing native probe passes attempts=16000 failures=0, exit0 (second corrected16000-start run). PR520 opened. Recorded affected check started on committed681c7986e; retain its runner while it qualifies. No code changed after that source checkpoint.

### 2026-10-21 — verification sequencing

Split the original combined verify/close/land checklist row at the actual local-acceptance boundary: the implemented targeted checks are complete; the recorded suite may overlap close review under tl-worker’s contract, but its final pass still precedes landing. Its retained lease is lease-1363786489 on681c7986, attemptc69f744252fbee1128c07e1f8fbf551a, exec12716, detached checkout sdlc-local-3724971426. The existing graph/embed gap selects a full suite, queued fourth. No gate waived.
