# Boundary Review — 000115-resurrect-a-session-across-agents#115 (milestone M4)

| field | value |
|-------|-------|
| issue | 115 — Switch the agent driving existing work |
| repo | 000115-resurrect-a-session-across-agents |
| issue file | workshop/issues/000115-resurrect-a-session-across-agents.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | d9664fee52a238b2ef7a64dd1cd70e16f561751a..HEAD |
| command | sdlc milestone-close --issue 115 --milestone M4 |
| reviewer | codex |
| timestamp | 2026-07-18T23:36:36-07:00 |
| verdict | REWORK |

## Review

> **Raw reviewer transcript trimmed.** This sidecar held the verbatim
> reviewer CLI transcript — for M4, 99,158 lines / 5.3 MB, mostly the
> echoed prompt and diff. That bulk is reconstructible from git (the diff
> is the review window) and it broke later `sdlc close` runs: the review
> dispatcher passes its prompt as argv, and these sidecars fell inside the
> next review window, pushing it past ARG_MAX (`fork/exec: argument list
> too long`). The verdict and findings — the durable part — are kept below.
> Full transcript: `git show e36c1dc~1:workshop/plans/000115-resurrect-a-session-across-agents-m4-review.md`.

## Verdict and findings
```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M4 delivers the main same-tag handoff path with durable journal states, target ownership persistence, README/atlas updates, and useful unit coverage around rollback/recovery. I did not find a correctness blocker in the reviewed code, but the boundary should not be treated as clean SHIP yet because the delivered process-level/concurrency coverage is materially narrower than the M4 plan and Done-when claims.

1. Strengths:
- `HandoffJournal` / `PlanRecovery` keep recovery decisions pure and directly tested, with target-ready replay now covering ownership/default persistence in [handoff_state.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_state.go:400).
- The coordinator writes ordered journal transitions before/after effects and captures transcript cutoff after source quiescence in [handoff.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff.go:58).
- Target ownership is durably published through `agent-<tag>` plus ledger append before forward finalization in [handoff.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff.go:171).
- README and atlas cover the new user-facing same-tag switch flow and acceptance seam in [README.md](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/README.md:235) and [atlas/session-identity.md](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/atlas/session-identity.md:137).

2. Critical findings:
- None.

3. Important findings:
- Missing planned process/concurrency acceptance breadth, ARCH-PURPOSE: Task 17 requires the process fake to cover Codex-to-Claude return, decline, conflicts, concurrent rejection, stale-recovery racers, readiness/timeout rollback, render-failure choices, stale markers, and crashes at each effect-before-journal window ([plan](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/workshop/plans/000115-resurrect-a-session-across-agents-plan.md:1497)). The delivered script exercises only one Claude-to-Codex happy path plus state assertions ([tests/pair-agent-handoff-test.sh](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/tests/pair-agent-handoff-test.sh:121)). Fix sketch: either add the missing process scenarios, or revise the plan/Done-when to state that the remaining failures are unit-level only.
- Missing planned lock/concurrency tests, ARCH-PURPOSE: Task 16 names `tag_operation_test.go`, rename locking tests, and concurrent tag-operation race coverage ([plan](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/workshop/plans/000115-resurrect-a-session-across-agents-plan.md:1434), [plan](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/workshop/plans/000115-resurrect-a-session-across-agents-plan.md:1461)), but the diff adds `tag_operation.go` without those test files. Fix sketch: add focused race/unit tests for attach vs handoff, create vs handoff, two creates, and crossed renames, or add a plan revision narrowing the claim.
- State-change handling does not match the plan’s “restart the decision once” contract, ARCH-PURPOSE: attach revalidation fails closed with an error at [createflow.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/createflow.go:213), while the plan says to restart the decision once and show the new result ([plan](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/workshop/plans/000115-resurrect-a-session-across-agents-plan.md:1457)). This is safe, but it is not the documented UX/recovery behavior.

4. Minor findings:
- The old `TestRunLaunchPickInferredAgentMustNotInheritCliArgs` comment still says explicit agent+args must not show the picker, which now conflicts with M4’s intended explicit-agent handoff behavior.

5. Test coverage notes:
- I ran `go test ./cmd/internal/launcher -run TestRunLaunchPickInferredAgentMustNotInheritCliArgs -count=1`; it passed.
- I did not run the full verification matrix.
- Unit coverage is broad for pure/preflight/journal/coordinator cases, but process-level acceptance is still mostly a happy-path smoke.

6. Architectural notes for upcoming work:
- ARCH-DRY: Pass. The diff reuses launch-arg/default logic and queue planning rather than creating agent-specific handoff variants.
- ARCH-PURE: Pass with caveat. Core decision/recovery planning is pure; handoff orchestration is a large integration shell but effects are injected through `Runtime`.
- ARCH-PURPOSE: Flagged above. The implementation covers the main purpose, but the acceptance proof does not yet derive every claimed consumer/failure path from the source of truth.

7. Plan revision recommendations:
- Add a `## Revisions` entry if the team intentionally accepts unit-level coverage for failure/concurrency cases instead of the broader Task 17 process-level matrix.
- Add a `## Revisions` entry if “state changed between picker and lock” is now intended to fail closed rather than restart the picker once.
126,883
```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M4 delivers the main same-tag handoff path with durable journal states, target ownership persistence, README/atlas updates, and useful unit coverage around rollback/recovery. I did not find a correctness blocker in the reviewed code, but the boundary should not be treated as clean SHIP yet because the delivered process-level/concurrency coverage is materially narrower than the M4 plan and Done-when claims.

1. Strengths:
- `HandoffJournal` / `PlanRecovery` keep recovery decisions pure and directly tested, with target-ready replay now covering ownership/default persistence in [handoff_state.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_state.go:400).
- The coordinator writes ordered journal transitions before/after effects and captures transcript cutoff after source quiescence in [handoff.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff.go:58).
- Target ownership is durably published through `agent-<tag>` plus ledger append before forward finalization in [handoff.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff.go:171).
- README and atlas cover the new user-facing same-tag switch flow and acceptance seam in [README.md](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/README.md:235) and [atlas/session-identity.md](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/atlas/session-identity.md:137).

2. Critical findings:
- None.

3. Important findings:
- Missing planned process/concurrency acceptance breadth, ARCH-PURPOSE: Task 17 requires the process fake to cover Codex-to-Claude return, decline, conflicts, concurrent rejection, stale-recovery racers, readiness/timeout rollback, render-failure choices, stale markers, and crashes at each effect-before-journal window ([plan](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/workshop/plans/000115-resurrect-a-session-across-agents-plan.md:1497)). The delivered script exercises only one Claude-to-Codex happy path plus state assertions ([tests/pair-agent-handoff-test.sh](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/tests/pair-agent-handoff-test.sh:121)). Fix sketch: either add the missing process scenarios, or revise the plan/Done-when to state that the remaining failures are unit-level only.
- Missing planned lock/concurrency tests, ARCH-PURPOSE: Task 16 names `tag_operation_test.go`, rename locking tests, and concurrent tag-operation race coverage ([plan](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/workshop/plans/000115-resurrect-a-session-across-agents-plan.md:1434), [plan](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/workshop/plans/000115-resurrect-a-session-across-agents-plan.md:1461)), but the diff adds `tag_operation.go` without those test files. Fix sketch: add focused race/unit tests for attach vs handoff, create vs handoff, two creates, and crossed renames, or add a plan revision narrowing the claim.
- State-change handling does not match the plan’s “restart the decision once” contract, ARCH-PURPOSE: attach revalidation fails closed with an error at [createflow.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/createflow.go:213), while the plan says to restart the decision once and show the new result ([plan](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/workshop/plans/000115-resurrect-a-session-across-agents-plan.md:1457)). This is safe, but it is not the documented UX/recovery behavior.

4. Minor findings:
- The old `TestRunLaunchPickInferredAgentMustNotInheritCliArgs` comment still says explicit agent+args must not show the picker, which now conflicts with M4’s intended explicit-agent handoff behavior.

5. Test coverage notes:
- I ran `go test ./cmd/internal/launcher -run TestRunLaunchPickInferredAgentMustNotInheritCliArgs -count=1`; it passed.
- I did not run the full verification matrix.
- Unit coverage is broad for pure/preflight/journal/coordinator cases, but process-level acceptance is still mostly a happy-path smoke.

6. Architectural notes for upcoming work:
- ARCH-DRY: Pass. The diff reuses launch-arg/default logic and queue planning rather than creating agent-specific handoff variants.
- ARCH-PURE: Pass with caveat. Core decision/recovery planning is pure; handoff orchestration is a large integration shell but effects are injected through `Runtime`.
- ARCH-PURPOSE: Flagged above. The implementation covers the main purpose, but the acceptance proof does not yet derive every claimed consumer/failure path from the source of truth.

7. Plan revision recommendations:
- Add a `## Revisions` entry if the team intentionally accepts unit-level coverage for failure/concurrency cases instead of the broader Task 17 process-level matrix.
- Add a `## Revisions` entry if “state changed between picker and lock” is now intended to fail closed rather than restart the picker once.
