---
id: 000615
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'f844c2caa31b93c60504795c579b27d32df5e572' # card fields mirrored from issue-cards; edit via sdlc
---

# Review lanes for couchcore remaining test budget

## Problem

After #611 R2 landed (PR371, merge24fe0959f4eaa6577c0d28e207953c899c2944e3), the requested main control still takes263.916s, above the hub's approximately200s threshold for bringing the lane proposal back for review. The final landing suite passed with couchcore357.969s. #610's temporary20m allowance remains unchanged. This is a separate capacity decision after the cost reductions, not approval to add a runner.

## Spec

Review R4 against the measured residual profile and the cost of maintaining new machinery. The existing package now has substantial headroom to Go's600s default; crossing200s triggers this review, not automatic adoption. Preserve the landed cheaper fixtures, every assertion/domain/case disposition, normal self-exec/TestMain routes and opt-in live conformance. Full flow and hub approval are required before implementation.

Candidate R4 retains the intact package and partitions Go-discovered Test/Fuzz/Example names into three deterministic sorted round-robin lanes, with exact terminal-event coverage accounting. Other packages run once before the lanes. At the measured caller limits, three core commands each use GOMAXPROCS=1/-parallel=3/-p=1, capping aggregate core CPU3, test workers9 and build jobs3; smaller caller limits reduce active lanes. Ordinary Go owns execution/JSON/timeouts. No handwritten inventory, timing cache, new skips, retries or increased deadline. Existing explicit live routes remain serial; a live opt-in on the runner falls back to ordinary unpartitioned Go. Owned process groups must cancel/join before return, with scoped temporary logs. No .sdlc/CI configuration change is proposed.

Proposed future qualification, subject to design review: two same-head isolated passes with each lane<=300s; one recorded full suite with each lane<=420s and command wall<=900.89s; full inventory/disposition parity. Remove #610 only after the agreed steady-state budget qualifies. No implementation is authorized by this issue filing.

## Done when

- [ ] The hub reviews the measured residual and explicitly approves a full-flow design or declines the lane machinery with a recorded reason.
- [ ] If approved, the implementation preserves every discovered case/disposition and ownership/cancellation contract, with semantic negative controls and bounded caller capacity.
- [ ] If approved, real isolated/full-suite measurements satisfy the agreed budgets, the temporary allowance is addressed explicitly, and mandatory review/landing gates pass.

## Plan

Pending hub decision. R4 is captured here for review; the claimed worker must write the approved durable plan and clear plan-quality before any code. Earlier proposal and root-cause history: [#611 plan](../history/plans/000611-couchcore-test-execution-budget-plan.md), [profile](../history/plans/000611-couchcore-test-execution-budget-profile.md). The hub owns ops/test-economy project records.

## Log

2026-10-16: Filed unassigned under verified TL decision898fac62-3c98-445e-9bf3-0bd7b9f78de2: land R2 first, run one control with no other full suite, and only if still above approximately200s bring R4 back as a separate item. R2 is landed/closed, SHIP with zero findings; this item has no approved implementation plan.

Main control headc3b9b451bfefa2ef41fc6740faa8f5ccb3150ebe (landed main archive commit): command `/usr/bin/time -l go test -trimpath -json ./cmd/internal/couchcore -count=1`, GOMAXPROCS=3, effective GOFLAGS=-p=3. Checked both machine-wide suite locks; queued258s, then reserved both for the entire measurement. No other Go/Make driver at start or end. PASS package263.916s, command266.46s;1,379 top-level cases =1,362pass/17existing skips. Serial phase121.386830s, remaining142.529s. No changed disposition among retained cases; exactly the two obsolete top-level tests and three obsolete subcase names removed by upstream #595 differ from the pre-integration1,381-case profile. R2 removed no cases. The ordered2,406,528-point domain guard passes.

This is suite-isolated, not an idle-host guarantee: load averages26.56/27.28/24.19 at start and19.05/21.64/22.36 at end. Queue time is excluded from package/command timing. JSON SHA256e25a787fbcc3bbcb23f276d26ff5602f06bcce395f4d3cba40c5c29d3ba51c7d. Local temporary raw evidence/host snapshots: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-611-profile.mIMPri/quiet-main.*; this issue preserves the result without depending on those files.

Residual timings: recovery convergence22.55s, bounded continuation21.70s, lifecycle14.74s; host/totality/metamorphic properties1.71/1.33/0.31s, fleet4.98s, SetAside0.82s. Real reconcile/observe/names child work sums35.41/24.85/13.51s; their inclusive scheduling windows108.68/105.92/109.52s include waiting behind unrelated children. Go's0.00 parent elapsed excludes parallel children; never use it as evidence of zero cost. Child durations overlap and cannot be added to package wall.

Updated three-lane model:460/460/459names; measured serial sums34.30/47.36/39.46s and parallel work sums484.78/385.36/399.45s. Sequential plus parallel/3 predicts195.89/175.81/172.61s. This omits changed contention, compilation/discovery and tail scheduling; it is a review input, not a verified speedup. Assess maintenance/throughput tradeoffs before approving machinery.
