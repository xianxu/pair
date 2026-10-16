# #611 couchcore timing profile

## Isolated package baseline

Command: `/usr/bin/time -l go test -trimpath -json ./cmd/internal/couchcore -count=1`.

Code pin: `4a1245d43c052a63379252c43f0ada579be78a1b` (planning commits only above main). Go 1.27.1, darwin/arm64, 12 logical CPUs. Caller environment: `GOMAXPROCS=3`, `GOFLAGS=-p=3`; couchcore TestMain therefore defaults to nine parallel tests. Package-isolated: no other Go suite appeared in the initial process snapshot. OS load was 16.41/21.90/23.80, so this does not claim a fully idle machine.

- PASS, package execution **326.006 s**, command wall **328.25 s**.
- **1,376** discovered/executed top-level cases: **1,359 pass, 17 skip** (existing opt-in/platform dispositions).
- **379 sequential** cases; **997** cases call `t.Parallel`.
- First parallel continuation: **143.743848 s** after package start. Sequential reported elapsed sum: **143.34 s**.
- Remaining package interval: **182.262 s**. Parallel reported elapsed sum: **1,596.18 s**, overlapping across about nine active tests. Do not add overlapping per-test times to package wall time.
- Command user/system CPU: 199.81/239.01 s; maximum resident size reported by time: 594,591,744 bytes. This is command instrumentation, not a summed concurrent process memory profile.
- Raw JSON SHA-256: `e45f670a3506a5687ad440a12e4d278c1921a9f2fbe11bbf1b12d3bb1704827a`.
- Local raw evidence: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-611-profile.mIMPri/isolated.jsonl`, `isolated.time`, `isolated-host.json`, `isolated-summary.json`. These are diagnostic temporary files; this summary is the durable evidence.

| Slowest completed top-level test | Execution seconds | Scheduling |
|---|---:|---|
| TestReconcileAgreesWithRealGit | 46.26 | parallel |
| TestRecoverPlanStepsConverge | 42.55 | parallel |
| TestReAdoptionExitsAreTotalAndCoded | 28.75 | parallel |
| TestContinuationRequestsBoundedOwnedSet | 28.45 | parallel |
| TestHostJudgmentsReadOnlyHostFacts | 25.91 | parallel |
| TestDeriveRecoverPlanIsTotalOverTheEvidenceDomain | 24.92 | parallel |
| TestRecoverReconcileReadingIsMetamorphic | 24.67 | parallel |
| TestFleetInventoryLiveConformance | 24.61 | parallel |
| TestObserveEachBrokenResource | 24.29 | parallel |
| TestRepositoryNamesAtConversationAllocationBoundaries | 23.96 | parallel |
| TestSetAsideHoldsMatchThePlan | 22.97 | parallel |
| TestRecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates | 11.01 | sequential |
| TestAcceptanceReportBudgetTenSlots | 9.74 | sequential |
| TestInterruptedSwitchResumePreservesTargetAuthority | 9.06 | sequential |

Source inspection: real Git repositories/worktrees and index discovery dominate the identified setup paths; per-operation attribution has not been measured. SetAsideHoldsMatchThePlan constructs nine independent provisioned fixtures. RecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates constructs two separate six-slot worlds. Parent-killed hour-long helper sleeps are intentional lifetimes, not evidence of an hour-long test.

## Partition model (prediction, not verification)

Sort Go-discovered top-level names and distribute round-robin; keep total parallel budget nine. For three shards (459/459/458 names), measured sequential sums are 38.24/53.80/51.30 s and parallel elapsed sums 480.02/481.71/634.45 s. Dividing each parallel sum by three yields approximate busy times **198.25/214.37/262.78 s**. This omits changed contention, orchestration and long-tail scheduling; measured runs must qualify the design. Two shards predict a busiest lane of 300.75 s; four shards with two workers each predict 288.15 s. Three is the proposed smallest useful bounded configuration for the current envelope.

## Suite-equivalent baseline

In progress. A temporary Makefile includes the real root Makefile and overrides only the final Go recipe with `go test -json ./... -count=1 -timeout 20m`; all existing prerequisites, environment filtering and the machine-wide suite lock remain. The 20m is #610's current baseline, not an increased allowance. Command stdout and time metadata are in `suite-before.log` and `suite-before.time` beside the isolated evidence. Record queue wait, full wall and couchcore events separately before any split is implemented.
