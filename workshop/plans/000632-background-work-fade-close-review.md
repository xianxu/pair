# Boundary Review — pair#632 (whole-issue close)

| field | value |
|-------|-------|
| issue | 632 — Keep background-command waits bright without blocking peer admission |
| repo | pair |
| issue file | workshop/issues/000632-background-work-fade.md |
| boundary | whole-issue close |
| milestone | — |
| window | 316474acd6fdd7f61fc8c06dd51b96e2dc3ac0aa..36a0c2599c93803fb74838b344fb0a0c72651488 |
| command | sdlc close --issue 632 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T00:34:46-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The display signal reaches the production activity projection independently of admission, and all four relevant package suites passed. Two corrections remain: the detector fabricates known-none evidence from unqualified footer text, and the Core concepts table claims a modification absent from the pinned range.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: unknown-evidence-preservation
    title: |
      Require a qualified complete footer before reporting known-none
    detail: |
      cmd/internal/wrapcmd/peer_composer.go:29-55 accepts any line beginning with “⏵⏵ ” below the composer. If no field contains the complete substring “shell”, it returns false: both “⏵⏵ garbage” and a truncated “… · 1 she” become known-none instead of unknown. This violates the plan’s incomplete/unqualified-footer contract; an active-to-false report can also fabricate completion through registry.go:228-231. Validate recognized no-shell grammar and return nil for malformed or truncated fields. Add negative tests for these cases. ARCH-SECURE, ARCH-ORDER.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-change-traceability
    title: |
      Correct the Core concepts table’s Console test modification claim
    detail: |
      workshop/plans/000632-background-work-fade-plan.md:49 marks cmd/internal/couchtty/console_activity_test.go as modified, but the pinned name-status and patch contain no change to that file. The revision at line 111 explains intentional reuse, yet leaves the table contradictory. Remove this row from the changed-entity inventory and document it as reused validation, with a Revisions entry reconciling the table. This is Critical under the requested Core concepts cross-check. ARCH-PURPOSE.
```

1. **Strengths**

   - Background evidence is independent of `Settled`; the delivery regression confirms a free composer still reaches `Submitted`.
   - Generation checks reject stale probes, including source changes during classification.
   - Compatibility tests freeze older schemas and cover downgrade, refusal, malformed acknowledgment, timeout, and reconnect.
   - Registry tests cover pointer ownership, replacement, disconnect, monotonic timestamps, and unknown evidence.

2. **Critical findings**

   Both findings above include locations and fix sketches. The footer correction needs executable regression coverage; the table correction requires accurate documentation only.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed independently:

   - Full `couchmessage`, `couchcmd`, `couchtty`, and `wrapcmd` package tests.
   - Focused background tests with `-race`.
   - `git diff --check`.

   Existing grammar tests omit partial shell tokens and arbitrary prefixed footer text. Live acceptance remains explicitly pending under the approved post-landing rollout.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses composer geometry, transport, registry, and activity projection.
   - **ARCH-PURE — pass:** classification and registry transitions remain deterministic.
   - **ARCH-PURPOSE — flag:** reconcile the changed-entity inventory.
   - **ARCH-MOCK — pass:** stateful terminal fixtures and real socket tests exercise shared production boundaries.
   - **ARCH-CONSTRAINTS — pass:** bounded viewport scanning and existing debounce/throttle cadence.
   - **ARCH-SECURE — flag:** unqualified terminal text becomes affirmative no-work evidence.
   - **ARCH-ORDER — flag:** malformed observations collapse uncertainty into known-none; generation fencing otherwise looks sound.
   - **ARCH-FUNERAL — pass:** evidence belongs to the existing session lifetime; no durable artifact family is added.

7. **Plan revision recommendations**

   Add a `## Revisions` entry correcting the Console test inventory and identifying its unchanged tests as reused validation. Preserve the existing requirement that incomplete or unqualified footers return unknown.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"792dfcf0-b995-4283-b4f3-9e46390fb2a2","repository":"github.com/xianxu/pair","issue_id":"000632","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/a9cef0961c4907f5","reviewed_head":"36a0c2599c93803fb74838b344fb0a0c72651488","slot":"pair:6","requested_at":"2026-10-17T07:31:09.284679Z"},"phase":"finished","pid":23489,"started_at":"2026-10-17T07:31:09.52236Z","finished_at":"2026-10-17T07:34:45.761704Z","session_id":"01a148c5-91c8-7a60-a38f-bf8ef1076c42","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.60160125s
head: "36a0c2599c93803fb74838b344fb0a0c72651488"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
