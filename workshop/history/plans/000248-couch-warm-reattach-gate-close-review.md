# Boundary Review — pair#248 (whole-issue close)

| field | value |
|-------|-------|
| issue | 248 — Couch switcher blocks warm reattach without a native binding |
| repo | pair |
| issue file | workshop/issues/000248-couch-warm-reattach-gate.md |
| boundary | whole-issue close |
| milestone | — |
| window | ec5c4f477c7e5b48f176e43ff93d4a73a7cfcac9..2a7ede468d62fcc3f4fd925ed549a26875072a0a |
| command | sdlc close --issue 248 |
| reviewer | codex |
| timestamp | 2026-09-14T10:52:02-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change satisfies #248’s warm-reattachment contract. Inventory and execution share session-based eligibility, foreground and startup selections preserve warm-only intent, and cold resume retains native-binding requirements. No blocking findings.

1. **Strengths**
   - Shared proof matcher checks exact address, agent, session name and observation cardinality.
   - Post-claim recheck rejects session replacement and rolls back ownership before launching a helper.
   - Acceptance test crosses real inventory, keyboard selection, operation dispatch and Console input/output.
   - Zellij query failures discard partial observations; cancellation and explicit empty inventory have regression coverage.
   - README and atlas explain warm attachment’s limits accurately.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:
   `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/launcher -count=1`

   Pinned-range `git diff --check` also passed. Full `make test`, race testing and live operator smoke were not independently repeated.

6. **Architectural notes**

   - **ARCH-DRY — pass:** one warm matcher serves classification, execution and recheck.
   - **ARCH-PURE — pass:** matching and menu intent remain pure; IO stays in existing orchestration seams. Core-concept declarations match implementation.
   - **ARCH-PURPOSE — pass:** foreground, startup and background entrypoints preserve attachment-only intent.
   - **ARCH-MOCK — pass:** portable stateful fixtures exercise production seams; existing scheduled Zellij conformance covers attachment states.
   - **ARCH-CONSTRAINTS — pass:** candidate-bounded queries retain five-second deadlines; warm paths perform zero native resolutions.
   - **ARCH-SECURE — pass:** failed observations grant no attachment authority; success fabricates no native binding.
   - **ARCH-ORDER — pass:** deterministic hooks exercise stale selection and session replacement; existing claims and cleanup own interrupted starts.
   - **ARCH-FUNERAL — pass:** no new durable artifact family; existing tracked-start cleanup remains responsible for helper ownership.

7. **Plan revision recommendations:** None. The pending close and operator-smoke handoff are represented accurately.

```findings
{}
```
