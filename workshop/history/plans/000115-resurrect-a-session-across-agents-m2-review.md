# Boundary Review — 000115-resurrect-a-session-across-agents#115 (milestone M2)

| field | value |
|-------|-------|
| issue | 115 — Switch the agent driving existing work |
| repo | 000115-resurrect-a-session-across-agents |
| issue file | workshop/issues/000115-resurrect-a-session-across-agents.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2fd949943154ca90f8458b3651f2504bfe247722..HEAD |
| command | sdlc milestone-close --issue 115 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-07-16T16:52:07-07:00 |
| verdict | REWORK |

## Review

> **Raw reviewer transcript trimmed.** This sidecar held the verbatim
> reviewer CLI transcript — for M4, 99,158 lines / 5.3 MB, mostly the
> echoed prompt and diff. That bulk is reconstructible from git (the diff
> is the review window) and it broke later `sdlc close` runs: the review
> dispatcher passes its prompt as argv, and these sidecars fell inside the
> next review window, pushing it past ARG_MAX (`fork/exec: argument list
> too long`). The verdict and findings — the durable part — are kept below.
> Full transcript: `git show e36c1dc~1:workshop/plans/000115-resurrect-a-session-across-agents-m2-review.md`.

## Verdict and findings
```verdict
verdict: SHIP
confidence: high
```

M2 delivers the documented readiness and argument-default boundary. Shared nonce-bound readiness, automatic argument precedence, stale-session fallback, asynchronous Zellij observation, and post-readiness default persistence are implemented and directly tested. The focused suites pass; the only full-suite failure is an environment restriction preventing an unrelated `httptest` listener.

1. Strengths

- `cmd/internal/readiness/record.go:10` provides one shared, validated readiness wire contract consumed by launcher and pair-wrap.
- `cmd/internal/launcher/launch_args_policy.go:37` keeps precedence and resume normalization pure, including correct fresh-launch behavior for stale resume tokens.
- `cmd/internal/launcher/createflow.go:431` correctly orders start → readiness → default persistence → blocking wait; failure orders stop → reap without changing defaults.
- `cmd/internal/launcher/readiness_os.go:34` removes stale evidence before child start and requires exact identity, nonce, live PID, and bounded observation.
- README and `atlas/session-identity.md` both document the new user-facing and architectural surfaces.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

- ARCH-DRY: `agent-ready-<tag>.json` is assembled independently in `createflow.go:433` and `wrap.go:468` despite `ScopedPaths.Ready()` existing. A future shared lightweight path helper would reduce naming drift, though the current values agree and tests pin them.

5. Test coverage notes

- Focused readiness, wrapcmd, and launcher suites pass.
- Tests cover exact/wrong nonces, dead or invalid identities, early child exit, timeout teardown, observed session absence, all three agents’ stale explicit-resume syntax, malformed saved config, explicit-empty defaults, and post-ready persistence failure.
- PURE core entities are tested without filesystem, process, network, or mocks requiring IO.
- INTEGRATION behavior uses injected launch/default seams and process-level fake Zellij coverage.
- `go test ./...` reached all changed packages successfully but failed in unrelated `cmd/internal/model` because the sandbox forbids `httptest` from binding `[::1]:0`; this is not attributable to the diff.
- `git diff --check` passes.

6. Architectural notes for upcoming work

- ARCH-DRY: Pass. Readiness encoding/matching and launch-argument precedence each have one source of truth.
- ARCH-PURE: Pass. Policy and identity matching remain deterministic; process/filesystem work stays behind runtime or writer seams.
- ARCH-PURPOSE: Pass for M2. Both readiness observation and automatic repo-agent defaults are wired into production create flow, not merely introduced as unused infrastructure.

7. Plan revision recommendations

None. All M2 Core concepts exist at the reconciled paths and their PURE/INTEGRATION classifications match the implementation.
68,332
```verdict
verdict: SHIP
confidence: high
```

M2 delivers the documented readiness and argument-default boundary. Shared nonce-bound readiness, automatic argument precedence, stale-session fallback, asynchronous Zellij observation, and post-readiness default persistence are implemented and directly tested. The focused suites pass; the only full-suite failure is an environment restriction preventing an unrelated `httptest` listener.

1. Strengths

- `cmd/internal/readiness/record.go:10` provides one shared, validated readiness wire contract consumed by launcher and pair-wrap.
- `cmd/internal/launcher/launch_args_policy.go:37` keeps precedence and resume normalization pure, including correct fresh-launch behavior for stale resume tokens.
- `cmd/internal/launcher/createflow.go:431` correctly orders start → readiness → default persistence → blocking wait; failure orders stop → reap without changing defaults.
- `cmd/internal/launcher/readiness_os.go:34` removes stale evidence before child start and requires exact identity, nonce, live PID, and bounded observation.
- README and `atlas/session-identity.md` both document the new user-facing and architectural surfaces.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

- ARCH-DRY: `agent-ready-<tag>.json` is assembled independently in `createflow.go:433` and `wrap.go:468` despite `ScopedPaths.Ready()` existing. A future shared lightweight path helper would reduce naming drift, though the current values agree and tests pin them.

5. Test coverage notes

- Focused readiness, wrapcmd, and launcher suites pass.
- Tests cover exact/wrong nonces, dead or invalid identities, early child exit, timeout teardown, observed session absence, all three agents’ stale explicit-resume syntax, malformed saved config, explicit-empty defaults, and post-ready persistence failure.
- PURE core entities are tested without filesystem, process, network, or mocks requiring IO.
- INTEGRATION behavior uses injected launch/default seams and process-level fake Zellij coverage.
- `go test ./...` reached all changed packages successfully but failed in unrelated `cmd/internal/model` because the sandbox forbids `httptest` from binding `[::1]:0`; this is not attributable to the diff.
- `git diff --check` passes.

6. Architectural notes for upcoming work

- ARCH-DRY: Pass. Readiness encoding/matching and launch-argument precedence each have one source of truth.
- ARCH-PURE: Pass. Policy and identity matching remain deterministic; process/filesystem work stays behind runtime or writer seams.
- ARCH-PURPOSE: Pass for M2. Both readiness observation and automatic repo-agent defaults are wired into production create flow, not merely introduced as unused infrastructure.

7. Plan revision recommendations

None. All M2 Core concepts exist at the reconciled paths and their PURE/INTEGRATION classifications match the implementation.
