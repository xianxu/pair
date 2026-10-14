# Boundary Review — pair#552 (whole-issue close)

| field | value |
|-------|-------|
| issue | 552 — Deliveries to the hub expire while the hub's own turn is open (90s deadline): operator submits and worker decisions are lost or late |
| repo | pair |
| issue file | workshop/issues/000552-deliveries-to-the-hub-expire-while-the-hub-s-own-turn-is-open-90s-deadline-operator-submits-and-worker-decisions-are-lost-or-late.md |
| boundary | whole-issue close |
| milestone | — |
| window | f5f2e8050e53cf49461115d6fd35d1daf2318e04..ca8f17e75e93c1f257b7af28ab8ad4f3b4d64d7f |
| command | sdlc close --issue 552 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T14:16:41-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range implements both documented root-cause fixes and the bounded tripwire. Review and tests found no blocking defects. README and atlas updates cover the new behavior. Live rollout remains explicitly assigned to the TL after landing; this review did not independently repeat the preserved production replay.

```findings
{}
```

1. **Strengths**

   - Confirmed-empty evidence reaches lease completion directly, avoiding another render requirement (`peer_delivery.go:561`).
   - The pure transition preserves uncertain-write protection and refuses unsafe timeout release (`automatic_input.go:52`).
   - Supervisor admission reserves full-incarnation sequence evidence before notification effects, preventing duplicate and reordered alerts (`automatic_input_stall.go:38`).
   - Reporter work has bounded capacity, a two-second RPC deadline, cancellation coverage, and joined teardown.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Full tests for `wrapcmd`, `couchmessage`, `couchcmd`, and `artifactpath`.
   - Targeted race tests for automatic-input transitions, reporter/supervisor behavior, and both root regressions.
   - `git diff --check`.

   Root regressions explicitly disable recovery. Tests cover unsafe admission guards, uncertain outcomes, occurrence ordering, replacement incarnations, overflow, RPC timeout, and cancellation. The claimed repository-wide suite was not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses shared composer parsing, diagnostics, transport, hub delivery, and desktop presenter.
   - **ARCH-PURE — pass:** Transition and validation logic are IO-free; integration adapters own effects. Core-concept classifications match the implementation.
   - **ARCH-PURPOSE — pass:** Addresses both reported causes; timeout recovery cannot conceal failures in the root regressions.
   - **ARCH-MOCK — pass:** Integration tests use temporary storage, private sockets, stateful service fixtures, and fake terminal output.
   - **ARCH-CONSTRAINTS — pass:** Enforces the declared threshold, queue capacity, evidence cap, and RPC deadline.
   - **ARCH-SECURE — pass:** Validates typed reports and current caller identity; notification text is supervisor-composed.
   - **ARCH-ORDER — pass:** Production lease changes use the transition function; controlled tests exercise duplicate, reordered, blocked, and cancelled effects.
   - **ARCH-FUNERAL — pass:** Reporter lifetime is joined, suppression records are bounded, and diagnostics use existing managed retention.

7. **Plan revision recommendations:** None. Existing revisions explain reconnect retention, full-binding IDs, and post-landing live acceptance.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.501087625s
head: "ca8f17e75e93c1f257b7af28ab8ad4f3b4d64d7f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
