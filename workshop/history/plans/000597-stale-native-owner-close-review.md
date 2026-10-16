# Boundary Review — pair#597 (whole-issue close)

| field | value |
|-------|-------|
| issue | 597 — Reject stale legacy native owners |
| repo | pair |
| issue file | workshop/issues/000597-stale-native-owner.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d96738f8a9034af12459699e30c80eb02f146bc..50597c9ccf496628f74aa8b6c1de229852f0eb64 |
| command | sdlc close --issue 597 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-15T21:51:46-07:00 |
| verdict | REWORK |

## Review

Production diff and tests read; now checking whether the admission bypass covers non-recovery Couch-owned resumes.

Confirmed: ordinary Couch cold resume (no Recovery) now skips admission with no registry check. Running the targeted tests.

```verdict
verdict: REWORK
confidence: high
```

The recovery side matches the spec. Both recovery gates in `recovery_prepared.go` now call one registry-only check, `checkRecoveryConversationInUse`. The duplicate scan done at validation time is gone, and the Fresh reserved-address fix is correctly gated on a validated `RecoveryAuthority` plus a Couch-owned launch. The targeted tests pass in couchcore, launcher and wrapcmd. One problem blocks SHIP. In `createflow.go`, the native-admission bypass is gated only on `couchOwned`, not on `opts.Args.Recovery != nil`. Ordinary Couch cold resume (`couchcore/resume.go:599`, which uses `BuildCouchResumeLaunchProfile` with no Recovery) therefore also lost native admission. That path never gets the replacement registry check, so it now has no cross-slot duplicate-conversation protection at all. The plan says "Remove forensic/store admission from Couch recovery **only**". One existing test was changed to pin the broader behaviour.

**1. Strengths**
- `recovery_conversation_in_use.go:13-37`: the check is small and fails safe in the right direction. Dead, unknown, reused-PID or unreadable slots never refuse. An exact `NativeID` or `RequestedNativeID` match refuses and names the slot.
- `createflow.go:657-664`: Fresh calls `EnsureThreadAddress(...,true)` only after `ValidateRecoveryAuthority` succeeds and only when the launch is Couch-owned. Six real-file scenarios in `recovery_fresh_claim_test.go` show the marker is left unchanged on every refusal.
- `wrapcmd/native_admission.go:24`: the hosted bypass needs both `couchHosted` and an empty grant. A grant that is present still goes through `startNativePTY`, so a mismatch between launcher and wrapper fails closed.
- Fixture fix in `reboot_test.go:289`: each slot now gets its own conversation ID, which removes a hidden collision the new check correctly caught.
- The dead glue (`recovery_native_owner.go`, the Scoped adapter, the advice test) is removed cleanly, and the `artifactpath` manifest is reclassified to match.

**2. Critical**
- **`cmd/internal/launcher/createflow.go:824`**: `if explicitResume != "" && !couchOwned` skips `ReserveNativeResume` for every Couch-owned explicit resume, including ordinary `couch` cold resume (`couchcore/resume.go:599`, where Recovery is nil). The wrapper then receives an empty grant and `startAgentPTY` starts it without admission. That path never calls `checkRecoveryConversationInUse`.
  - Failure: slot B recovers conversation X while slot A is dead (the registry check passes). The user then cold-resumes A, whose established binding is still X. No check runs and X now runs in two slots.
  - `TestRequiredNativeResumeBindingLaunchesExactRootWithoutDefaults` (`createflow_test.go:541-557`, no Recovery) now asserts this broader behaviour.
  - Fix: use `explicitResume != "" && !(couchOwned && opts.Args.Recovery != nil)`, matching the Fresh branch at line 658. Revert the test's no-reserve assertion and add a recovery-carrying variant.
  - Family sweep in this window:
    - `createflow.go:824` and the test at `createflow_test.go:541` are the overbroad sites.
    - The comment at `createflow.go:821-823` says Couch "checked its registry", which is false for cold resume.
    - The doc comment on `wrapcmd/native_admission.go:20-23` makes the same claim.
    - With the launcher fix, the wrapper bypass becomes recovery-only automatically (an empty grant then means recovery), so the wrapper needs no code change. Its comment still needs fixing.

**3. Important**
- **Missing gate-level refusal coverage** (`recovery_resume_integration_test.go:357`). The plan says "validate registry collisions at both recovery gates". The old both-gates refusal tests were deleted. Now only the unit table covers refusal, and the integration test covers only the non-refusal case. Add a both-gates (preflight and after-prepare) case where another live slot records the same conversation. It should refuse with "in use by", produce zero Runner ops, and show no effects before the first gate.
- **Done-when clause "Fresh … remains retryable" is only half covered** (`recovery_fresh_claim_test.go`). Refusals leave the marker unchanged, but no test runs a refused or failed attempt and then a successful Fresh on the same reserved marker. Add a "changed attempt → corrected attempt succeeds" sequence.

**4. Minor**
- `CouchHostedEnv` is an OR of scope and tag, while launcher `couchOwned` requires both to match. The asymmetry is harmless today because the bypass requires an empty grant. Note it for #595.
- Inside the hosted bypass, `startAgentPTY` skips `ParseNativeResumeSelector`. That is acceptable only because the launcher always pins the selector first.

**5. Test coverage notes**
- The targeted runs pass: the couchcore registry, ignore-stale and RecoverPlan tests; launcher Fresh and RequiredNativeResume; wrapcmd hosted.
- The unit table covers same, requested, dead, unknown, reused PID, different, unknown binding, own slot, and stale-owner evidence: a good false-positive set.
- I did not verify a full recorded suite run or the build in this review.

**6. Architecture**
- ARCH-DRY: pass. `recordPositivelyLive` reuses `observeExactProcessOrUnknown`, and slot naming reuses `resumeEvidence` and `recoverySlotAddress`.
- ARCH-PURE: pass. The check is a method over injected `Threads` and `Proc`, tested with fakes.
- ARCH-PURPOSE: flagged under BR (the Critical above). The registry check, which is the source of cross-slot safety, is enforced on the recovery consumers only. The admission removal reaches the ordinary cold-resume consumer, which has no replacement check.

**7. Plan revisions**
- In the "registry-only recovery built" entry, change "createflow `ReserveNativeResume` skipped when the launch is Couch-owned" to "…Couch-owned **recovery** launch (Recovery authority present); ordinary Couch cold resume keeps admission until #595".

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: recovery-only-scope-gate
    title: |
      Native admission bypass gated on couchOwned, not Recovery, strips cross-slot protection from ordinary Couch cold resume
    detail: |
      createflow.go:824 skips ReserveNativeResume for any Couch-owned explicit resume. couchcore/resume.go:599 (BuildCouchResumeLaunchProfile, Recovery nil) now launches with an empty grant, wrapper startAgentPTY bypasses admission, and that path never calls checkRecoveryConversationInUse, so a conversation live in another slot can be duplicated. The plan says "Couch recovery only". Fix: gate on couchOwned && opts.Args.Recovery != nil, like the Fresh branch at line 658. Family sweep: createflow.go:824; createflow_test.go:541-557 (pins the overbroad behaviour without Recovery); stale comments at createflow.go:821-823 and wrapcmd/native_admission.go:20-23 claiming the registry was checked.
  - id: new
    severity: Important
    kind: defect
    family: done-when-both-states-coverage
    title: |
      No gate-level test that an exact registry collision refuses at both recovery gates
    detail: |
      The old both-gates refusal tests were deleted. recovery_resume_integration_test.go now covers only the non-refusing stale-evidence case, and refusal is pinned only by the unit table. Add preflight and after-prepare cases with the same conversation in another live slot, asserting the "in use by" error, zero Runner ops, and no effects before the first gate. Same family: the Done-when "remains retryable" clause has no test of a refused Fresh followed by a successful retry on the same reserved marker (recovery_fresh_claim_test.go).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7eb99c96-b29b-43f8-a649-949a8f9865d6","repository":"github.com/xianxu/pair","issue_id":"000597","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/fa569a328969e5bb","reviewed_head":"50597c9ccf496628f74aa8b6c1de229852f0eb64","slot":"pair:2","requested_at":"2026-10-16T04:49:29.662811Z"},"phase":"finished","pid":80249,"started_at":"2026-10-16T04:49:29.752688Z","finished_at":"2026-10-16T04:51:45.983678Z","session_id":"b2f87f73-e6be-489d-9843-b447db52ebc2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.257711708s
head: "50597c9ccf496628f74aa8b6c1de229852f0eb64"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T22:15:41-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 597 — Reject stale legacy native owners |
| repo | pair |
| issue file | workshop/issues/000597-stale-native-owner.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6f50978e0165d6f9ad4e6f17bf598ac0a780a194..ca8e2ee5d5cbdd713dd5c9878505418a19e0644d |
| command | sdlc close --issue 597 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-15T22:15:41-07:00 |
| verdict | REWORK |

## Review

Checking whether other Couch-owned launch paths (new slot, Fresh) can still reach the skipped admission with a resume selector.

Found a likely gap: ordinary Couch spawns with an explicit `--resume` argv. Writing a quick probe test to confirm.

Confirmed: a Couch spawn with explicit `--resume` argv now launches with no admission and no registry check. Running the new tests next.

```verdict
verdict: REWORK
confidence: high
```

Round 1 fixed both named instances, and I confirmed each fix by reverting it. The cold-resume registry check in `resume.go:601` now has a test that fails without it (`TestColdResumeRefusesConversationLiveInAnotherSlot` launched `couch-fake-1` when I disabled the check). The new both-gates acceptance test fails with only the after-prepare gate disabled (":3 resumed a conversation live in :2: <nil>"). BR-2 is addressed. BR-1 is not, because the underlying rule is still wrong. In `createflow.go:824` the launcher skips `ReserveNativeResume` for **any** Couch-owned launch that resolves a native resume selector. Only the two launches built by `BuildCouchResumeLaunchProfile` and `BuildCouchRecoveryLaunchProfile` run the replacement registry check first. An ordinary Couch spawn goes through `BuildCouchLaunchProfile`, and its explicit or path argv can carry `--resume <id>`, `--continue` or a picker. Nothing on the Couch side rejects resume selectors in that argv. I ran a probe test at head: a Couch-owned `BuildCouchLaunchProfile` launch with `claude --resume <uuid>` returned `code=0 reserved=0 launches=1 grantenv=""`. So it took no admission, passed no registry check, and the wrapper's `startAgentPTY` then started it without admission because `couchHosted && grant==""`. At base, the same launch reserved admission. The comments at `createflow.go:821-823` and `wrapcmd/native_admission.go:20-23` say "every Couch resume checks the registry", which is false for this path. The issue's own new lesson (`workshop/lessons.md`, #597 BR-1) describes exactly this gap.

1. **Strengths**
   - `couchcore/recovery_conversation_in_use.go`: one small check that reads only the registry. Only a positively live slot can refuse, and an unreadable binding never invents a collision. This matches the Spec's "unknown IDs do not refuse".
   - `recovery_conversation_in_use_test.go`: the 9-case table covers same, requested, dead, unknown, reused-PID, different, unknown-conversation, own-slot and stale-native-owner cases. These are the false-positive controls the Done-when clause asks for.
   - `recoverplan_acceptance_test.go:299`: both gates run on the real multi-slot recover world, including a check that preflight refuses before :3's workspace is prepared. I confirmed it fails without the gate.
   - `recovery_fresh_claim_test.go`: tests a refused Fresh followed by a retry on the same reserved marker, against real files, with a byte-identical marker check. This covers "remains retryable".
   - The fixture fix in `reboot_test.go` (one conversation ID per slot) is the correct response to the new check catching a fixture that wasn't realistic.

2. **Critical:** BR-1 is still open (disposed `not-addressed` below), with a fix sketch in its note.

3. **Important:** none new.

4. **Minor**
   - `atlas/couch.md:93` still describes native admission as the boundary against concurrent writers. Add one sentence saying that Couch-owned resume and recovery skip it in favour of the registry collision check (#597), and that ordinary Pair keeps it until #595.
   - `TestColdResumeRefusesConversationLiveInAnotherSlot` loops over `collide` without `t.Run`, so a failure doesn't name its case.

5. **Test coverage**
   - All targeted tests pass, and `go build ./...` is clean.
   - The missing test is a Couch-owned **non-resume-profile** launch whose argv carries a native selector, asserting that admission is reserved. `TestRequiredNativeResumeBindingLaunchesExactRootWithoutDefaults` only pins the skip side, so nothing pins the other side of the bypass.

6. **Architecture**
   - **ARCH-DRY: pass.** Both recovery gates and cold resume reuse the single `checkRecoveryConversationInUse`.
   - **ARCH-PURE: pass, with a note.** The check mixes snapshot IO and process observation, but it sits on the Couch shell with injected `Proc` and `Artifacts`. The tests use the existing fakes.
   - **ARCH-PURPOSE: flag (BR-1).** The bypass is keyed to "Couch-owned" rather than to "a launch preceded by the registry check". The source of truth (the check ran) is not enforced at the consumer (the launcher gate).

7. **Plan revisions:** the Log entry "Every Couch resume launch builder now passes the one registry check" is true only for the resume and recovery builders. Revise it once the gate is narrowed.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The named cold-resume instance is fixed and verified (removing the check at resume.go:601 makes TestColdResumeRefusesConversationLiveInAnotherSlot launch couch-fake-1). The class remains open. createflow.go:824 still skips ReserveNativeResume for every Couch-owned launch with a resolved selector. A Couch spawn built by BuildCouchLaunchProfile (spawnResolved / launchClaimedThread with explicit or path argv) carrying --resume ID, --continue or a picker gets no admission and no registry check; a probe at head gave code=0 reserved=0 launches=1 grantenv="" (base reserved). Rule: skip admission only for launches whose profile proves the registry check ran, i.e. couchOwned && (opts.Args.ResumeRequired || opts.Args.Recovery != nil). Every other Couch-owned selector launch keeps ReserveNativeResume, so the wrapper gets a grant and startAgentPTY goes through startNativePTY. Sites: createflow.go:824 gate; createflow.go:821-823 and wrapcmd/native_admission.go:20-23 comments ("every Couch resume"). Add a createflow test: BuildCouchLaunchProfile plus claude --resume <uuid> under Couch env must reserve admission.
  - id: BR-2
    disposition: addressed
    note: |
      TestRecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates passes, and its after-prepare case fails with the recovery_prepared.go:388 gate disabled (":3 resumed a conversation live in :2: <nil>"). The preflight case asserts zero prepares. The "retry after refusal" scenario in recovery_fresh_claim_test.go covers the retryable clause with a byte-identical marker check.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: docs-surface-sync
    title: |
      atlas/couch.md:93 still presents native admission as the concurrent-writer boundary for Couch
    detail: |
      Couch-owned resume and recovery now skip native admission and rely on checkRecoveryConversationInUse (in use by repo:N). Add one sentence naming the registry collision check (#597) and noting that ordinary Pair keeps admission until #595.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e3703c8a-ab89-425e-8c99-e297da7fc459","repository":"github.com/xianxu/pair","issue_id":"000597","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ef117cb7f0a30f23","reviewed_head":"ca8e2ee5d5cbdd713dd5c9878505418a19e0644d","slot":"pair:2","requested_at":"2026-10-16T05:10:57.205364Z"},"phase":"finished","pid":39160,"started_at":"2026-10-16T05:10:57.299607Z","finished_at":"2026-10-16T05:15:41.026525Z","session_id":"0da56c1e-8543-47c9-bc82-083bd679b991","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.471586417s
head: "ca8e2ee5d5cbdd713dd5c9878505418a19e0644d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T22:37:49-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 597 — Reject stale legacy native owners |
| repo | pair |
| issue file | workshop/issues/000597-stale-native-owner.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6f50978e0165d6f9ad4e6f17bf598ac0a780a194..7daa8f2bf37dfd1eaf52831b5b4718d9c4b588b2 |
| command | sdlc close --issue 597 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-15T22:37:49-07:00 |
| verdict | SHIP |

## Review

Focus on the last commit for BR-1 and atlas for BR-3.

Now revert the gate to confirm the regression test goes red.

```verdict
verdict: SHIP
confidence: high
```

Both open findings are fixed. For BR-1, the launcher at `cmd/internal/launcher/createflow.go:825` now skips native admission only when `couchOwned && (ResumeRequired || Recovery != nil)`. Those are the only two profile builders that set these flags: `BuildCouchResumeLaunchProfile` at `launch_args_policy.go:81` and `BuildCouchRecoveryLaunchProfile` at `recovery_authority.go:43`. Each has exactly one caller, and each caller runs `checkRecoveryConversationInUse` before it builds the profile (`couchcore/resume.go:601` before `:604`, and `recovery_prepared.go:388` before `:397`). The wrapper bypass is now inside `startNativePTY` (`wrapcmd/native_admission.go:40`) and covers only a Couch-hosted pinned (`NativeResumeExplicit`) resume that has no grant. The extra `startAgentPTY` wrapper was removed, so one start boundary remains. I reverted each fix in turn, and each new regression test failed. With the fixes in place, all four affected packages pass. For BR-3, the atlas now names the registry collision check and how far it reaches.

1. **Strengths**
   - The bypass condition is now the same set of profiles that run the registry check. Every caller of the two builders checks before it builds, so no path launches without one of the two protections.
   - `TestCouchSpawnWithResumeArgvStillReservesAdmission` (`native_createflow_test.go:368`) tests the exact gap BR-1 found: an ordinary Couch spawn whose argv contains `--resume`.
   - The wrapper test (`native_admission_test.go:426-432`) covers both cases: a Couch-hosted unpinned `--continue` with no grant is refused, and the hosted/non-hosted loop covers the pinned case.
   - `startAgentPTY` was folded into `startNativePTY`, so there is one native start boundary instead of two that do almost the same thing (ARCH-DRY).

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none worth recording. The test comments at `createflow_test.go:541-549` say "Couch-owned resume". That is accurate here because the fixture carries `ResumeRequired`.
5. **Test coverage**
   - Revert check 1: changing `registryChecked` back to `couchOwned` makes `TestCouchSpawnWithResumeArgvStillReservesAdmission` fail with "handed off without a grant".
   - Revert check 2: widening the wrapper bypass to `couchHosted` alone makes `TestCouchHostedResumeStartsWithoutNativeAdmission` fail at line 431.
   - `go test` passes for launcher, wrapcmd, couchcore and artifactpath.
   - Done-when clause 1 is covered in both states: collision refuses, and own, dead or unknown slots don't (BR-2, already disposed). Clause 2 is covered by `recovery_fresh_claim_test.go`.
6. **Architecture**
   - **ARCH-DRY: pass.** The duplicate start wrapper is gone, and the registry check is one shared helper used at all three sites.
   - **ARCH-PURE: pass.** The bypass is a pure predicate over profile flags and the selector kind, and the IO stays behind the injected `start` and store.
   - **ARCH-PURPOSE: pass.** I swept every consumer of the bypass condition (launcher gate, wrapper gate, both profile builders) and each one follows the registry-checked rule. Deleting the leftover launcher-side native-owner code is explicitly scoped to #595.
7. **Plan revisions:** none. The plan's "Couch recovery only" plus cold resume now matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      createflow.go:825 gates on couchOwned && (ResumeRequired || Recovery != nil); both builders' sole callers run checkRecoveryConversationInUse first (resume.go:601, recovery_prepared.go:388); wrapper bypass narrowed to pinned selector; both regression tests verified red on revert.
  - id: BR-3
    disposition: addressed
    note: |
      atlas/couch.md:93-97 now names the registry collision check, the in-use message, and that other launches plus ordinary Pair keep admission until #595.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9644e68c-4bbc-4899-b634-617411dfa540","repository":"github.com/xianxu/pair","issue_id":"000597","recipe":"small-diff-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/d0400c5bf07ca9ea","reviewed_head":"7daa8f2bf37dfd1eaf52831b5b4718d9c4b588b2","slot":"pair:2","requested_at":"2026-10-16T05:31:44.401955Z"},"phase":"finished","pid":71499,"started_at":"2026-10-16T05:31:44.502194Z","finished_at":"2026-10-16T05:37:48.822164Z","session_id":"4950f52c-517f-426c-9748-e0bc7dd51d5e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.287817s
head: "7daa8f2bf37dfd1eaf52831b5b4718d9c4b588b2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
