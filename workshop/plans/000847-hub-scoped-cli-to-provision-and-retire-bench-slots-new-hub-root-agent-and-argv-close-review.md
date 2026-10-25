# Boundary Review — pair#847 (whole-issue close)

| field | value |
|-------|-------|
| issue | 847 — Hub-scoped CLI to provision and retire bench slots (new hub root, agent and argv, fresh) without a Couch restart or console steps |
| repo | pair |
| issue file | workshop/issues/000847-hub-scoped-cli-to-provision-and-retire-bench-slots-new-hub-root-agent-and-argv.md |
| boundary | whole-issue close |
| milestone | — |
| window | b85cebd610807b9f779a83941241c6d720592dae..053cfbcf41d12b8a484b19152e355b75b1bccb11 |
| command | sdlc close --issue 847 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T04:45:18-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing: I've been through the core and message diffs, and now I'm reading the CLI, guard and test-harness changes.

Checking the liveness helper and the acceptance test, then I'll run the unit tests.

Checking two console-side type switches on `RebootResult` that bench-start's new result type may miss.

Now running the affected packages' unit tests.

The failures come from the review sandbox refusing `git init`. Rerunning just the new bench tests in verbose mode to confirm.

Trying again with a temp dir inside the checkout, which may fall within the sandbox's allowed paths.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The work largely matches the Spec and Plan. `HubOwnership.Provisions` is a small, well-tested rule. The guard runs both at admission and again when the queued operation takes effect, and a test checks that revoking a claim in between blocks the effect. bench-start goes through Reboot's existing Fresh path, so there is still only one way to launch. Receipt provenance is optional and omitted when empty, so older receipts read unchanged. README, atlas, the couch skill and help text all cover the new verbs. Two issues should be fixed before close. First, bench-start can report `succeeded` when no agent was started. Second, the strict-retire "no survivors" proof can be lost if writing the orphan registry fails, and a retry would then retire the slot while a process is still running. Both are cheap to fix. Two unit tests could not run here: the review sandbox refuses `git init` in temp directories. Every other new couchcore, couchcmd and couchmessage test passed.

1. **Strengths**
   - `cmd/internal/couchcore/hub_ownership.go:597-655`: Provisions reads which rule `resolve` applied (hub slot, override, claim or default), so ownership by default or by a runtime move can never count as a claim. `Resolve` keeps its signature. The test has one row per source and caller role (`hub_ownership_provisions_test.go`).
   - `cmd/internal/couchcmd/bench_guard_test.go:225-238`: a claim revoked between admission and effect is refused at the effect.
   - The explicit profile is threaded through the existing preflight `LaunchProfileResolution` parameter (`recovery_fresh_missing.go:893-902`) rather than a second launch path. Tests check that `repo-preferences.json` stays byte-identical for both live and vacant slots.
   - The bench ops are deliberately declared `PresentationTUI` rather than `PresentationInternal`. `couch --internal` has no hub check, and `cli_test.go` pins its refusal of these ops.
   - Survivors are filtered by repo scope, and `ResolveRepoScope` is keyed per slot worktree, so a production slot's orphans cannot block a bench retire.

2. **Critical findings:** none.

3. **Important findings**
   - **bench-start reports success when nothing launched** (`cmd/internal/couchcore/bench_operation.go:81-83`). If Reboot takes its ArchiveOnly branch (`reboot.go:123-147`), it returns no start and a nil error. That happens when the record is unreadable or the checkout is gone, which is plausible here because the bench runner owns checkouts. BenchStart then returns `out, nil`, and `classifySlotOperationOutcome` marks the receipt `succeeded` with no provenance. The Spec requires receipts the observer can verify.
     - Fix: when `!ok && err == nil`, return a typed `SlotOperationError` (for example "bench-start archived the record but started nothing: <Reason>").
     - Test: a unit test where the checkout is missing.
   - **The strict-retire proof can be silently lost** (`bench_retire.go:57-61` together with `teardown.go:236-241`). The plan says the orphan registry is the only record once the agent has stopped. But `Teardown` turns a failed registry `Add` into a note, and RetireBenchSlot ignores `proof.Notes`. The first call still refuses, because `LeftRunning` is in hand. On retry the agent is gone, so `leftRunning` returns nil. The registry has no entry, so the retry archives and reports the slot retired while a survivor runs.
     - The same thing happens when the registry's global 256-entry cap (`orphan_registry.go:21`) evicts this slot's entry.
     - Fix: after Teardown, check that every `LeftRunning` entry is in the registry, or have Teardown return the Add error to the strict caller. Refuse unless the record is safely stored.

4. **Minor findings**
   - The plan and atlas say "an orphan whose recorded identity is unknown counts as a survivor while its PID is alive". The code does not do this:
     - `leftRunning` skips children with an empty identity (`teardown.go:522`).
     - `observeExactProcess` treats a mismatched or empty recorded identity as Dead (`couch.go:1288`).
     - What is tested is "PID existence unknown" (`SetUnknown`), which is a different case.
   - `couchtty/console_lifecycle_failure.go:84` switches on `couchcore.RebootResult`, which a `BenchStartResult` never matches. A bench-start whose embedded reboot set `SessionNotStopped` loses its actionable warning source. This is a missed entry in the existing type-switch registry, the same class as this issue's own lesson.
   - `prepareBenchOperation` repeats reboot's vacant-slot (`SlotOpNoThread`) special case (`slot_operation.go:971-973`). The duplication is small, but one helper would remove it.
   - Two tests could not run here (see section 5).

5. **Test coverage notes**
   - These passed here: `TestParseBenchProfile`, `TestRebootExplicit*`, `TestBenchStartCarriesProvenance`, `TestBenchSurvivors`, `TestRetireBenchSlotRefusesWhileAnOrphanLivesThenArchives`, `TestHubOwnershipProvisions`, `TestOperationDeclarations*`, couchcmd `Bench*`, the ParseCLI, routing and arity tests, couchmessage, and artifactpath.
   - `TestVacantSlotFreshStartTakesExplicitProfile` and `TestRetireBenchSlotVacant` failed only because of the sandbox (`git init refused … outside the reviewer's directories`), as did the testcouch provisioning tests.
   - I did not run the native `TestDisposableCouchBenchSlots`. The implementer's log reports it passing for both claude and codex.
   - Gaps:
     - No test for bench-start on the ArchiveOnly path.
     - No test where the registry write fails, or the cap evicts an entry, followed by a retry.
     - Nothing covers Teardown `LeftRunning` on a live agent through RetireBenchSlot. The refusal test only seeds the registry.

6. **Architecture notes**
   - **ARCH-DRY:** pass, with the minor duplication above. `parkBenchSlot`, `prepareRetirement` and `Teardown` are shared.
   - **ARCH-PURE:** pass. Provisions and ParseBenchProfile are pure. The guard is a thin layer over the ownership snapshot.
   - **ARCH-PURPOSE:** flagged for the false success. The receipt is the proof the issue asks for, so a `succeeded` with nothing started under-delivers. The type-switch miss is a sibling of the class named in the lesson.
   - **ARCH-MOCK:** pass. The native harness runs fixture agents. In-package fakes model process liveness across calls.
   - **ARCH-CONSTRAINTS:** pass. These are operator-paced heavy verbs on the existing bounded queue.
   - **ARCH-SECURE:** pass. Argv is validated by the broker (ParseBenchProfile) and again on the launch path (`ValidateFreshAgentArgs` in `slotrecovery.go:442` and `recovery_preflight.go:27`). Authority comes from the binding-verified caller workspace.
   - **ARCH-ORDER:** pass with a note. bench-start parks and then reboots in two steps. If the reboot fails, the slot stays parked and a retry works. The evidence gap in the second Important finding is an observation that does not survive a retry.
   - **ARCH-FUNERAL:** flagged, tied to the second Important finding. The registry cap is a bound on the writer, but strict retire depends on it, so eviction silently drops evidence. Otherwise retire archives the record and keeps the worktree as designed.
   - **ARCH-EVOLVE:** pass. `Provenance` is optional and omitted when empty, older receipts decode unchanged (tested), and the `.hub` format is untouched.

7. **Plan revision recommendations**
   - Add a "## Revisions" entry. Either drop the claim that an unknown-identity orphan counts as a survivor and replace it with what the code does (a PID whose liveness is unknown counts; an unrecorded identity is never registered), or implement the claim.
   - Add a "## Revisions" entry: a bench-start that archived without starting anything is refused, not reported succeeded (once the fix lands).

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: receipt-success-without-effect
    title: |
      bench-start reports succeeded with no provenance when Reboot archives only and starts nothing
    detail: |
      bench_operation.go:81-83 returns (out, nil) when result.Started() is false. Reboot's ArchiveOnly branch (reboot.go:123-147: unreadable record or missing checkout) then becomes a succeeded receipt with no launch. Return a typed SlotOperationError carrying the Reboot Reason, and add a unit test where the checkout is missing.
  - id: new
    severity: Important
    kind: defect
    family: strict-proof-evidence-loss
    title: |
      bench-retire retry can archive while a survivor lives if the orphan registry lost its entry
    detail: |
      Teardown turns a failed LeftRunningRegistry.Add into a note (teardown.go:236-241), and RetireBenchSlot ignores proof.Notes (bench_retire.go:57-61). On retry the agent has stopped, so leftRunning returns nil and the registry has no entry, and the slot is archived. Eviction by the 256-entry global cap has the same effect. Check that LeftRunning entries are in the registry, or return the Add error to strict callers, and refuse if not. Add a test for a failed registry write followed by a retry.
  - id: new
    severity: Minor
    kind: defect
    family: plan-claims-unimplemented-rule
    title: |
      Plan and atlas say unknown-identity orphans count as survivors; code never records or counts them
    detail: |
      leftRunning skips children with an empty identity (teardown.go:522), and observeExactProcess treats a mismatched or empty recorded identity as Dead. The test only covers unknown PID existence (SetUnknown). Revise the plan and atlas, or implement the rule.
  - id: new
    severity: Minor
    kind: defect
    family: exhaustive-result-type-switch
    title: |
      console lifecycle warning switch handles RebootResult but not BenchStartResult
    detail: |
      couchtty/console_lifecycle_failure.go:84 does not match the BenchStartResult wrapper, so a SessionNotStopped warning on a bench-start is not marked actionable.
  - id: new
    severity: Minor
    kind: unverifiable
    family: sandbox-git-fixture
    title: |
      Vacant-slot bench tests could not run in the review sandbox
    detail: |
      Ran: go test ./cmd/internal/couchcore -run 'TestVacantSlotFreshStartTakesExplicitProfile|TestRetireBenchSlotVacant', also with TMPDIR set inside the checkout. Both failed with "git init refused ... outside the reviewer's directories (sdlc #339)". Answered by a host run of these two tests passing.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"75fc0a92-60db-49e3-bfa2-1d9661c2528a","repository":"github.com/xianxu/pair","issue_id":"000847","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/0124dd9c656d8da1","reviewed_head":"053cfbcf41d12b8a484b19152e355b75b1bccb11","slot":"pair:6","requested_at":"2026-10-25T11:40:43.342324Z"},"phase":"finished","pid":68262,"started_at":"2026-10-25T11:40:43.570957Z","finished_at":"2026-10-25T11:45:14.133816Z","session_id":"f4104832-c0c3-4b14-ba70-f69d1ffcfa17","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.099096417s
head: "053cfbcf41d12b8a484b19152e355b75b1bccb11"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
