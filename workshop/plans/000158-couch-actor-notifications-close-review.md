# Boundary Review — pair#158 (whole-issue close)

| field | value |
|-------|-------|
| issue | 158 — couch: actor notifications and attention routing |
| repo | pair |
| issue file | workshop/issues/000158-couch-actor-notifications.md |
| boundary | whole-issue close |
| milestone | — |
| window | f93bd568361d3c26ac46ea487c607ff695407689..0262119f9c15a80e052232fc42ab7edfefb15925 |
| command | sdlc close --issue 158 |
| reviewer | codex |
| timestamp | 2026-08-31T23:37:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation is broadly strong: focused and race suites pass, notification state is bounded and pure, and the PTY integration covers the real path. The boundary cannot cross because the mandatory Core concepts table contradicts the delivered replay design. Two Important issues also remain: the new public command is absent from README, and the nonblocking TTY writer silently accepts short writes.

1. Strengths

- The canonical codec centralizes sanitization, framing, and bounds in `cmd/internal/notifyosc/notification.go`.
- `AttentionLedger` cleanly owns bounded, ephemeral state and protects post-dispatch arrivals through identity-qualified acknowledgment.
- The real-PTY conformance test exercises production encoding, PTY observation, forwarding, attribution, and focused consumption.
- Replay excludes completed notification spans even when ring retention bisects an envelope.
- `atlas/architecture.md` accurately documents the new transport and Couch attention flow.

2. Critical findings

- `workshop/plans/000158-couch-actor-notifications-plan.md:24`: the Core concepts table claims a modified `ptychild.ReplayWindow` at `cmd/internal/ptychild/replay.go`. No such entity exists, and that file is unchanged in the review window. Replay-safe offsets and notification-span filtering were implemented across `Screen` and `Child` in `screen.go` and `child.go`. Revise the plan table and accompanying description to reflect the actual entities and locations.

3. Important findings

- `cmd/internal/notifycmd/run.go:31`: `OSRuntime.WriteNonblocking` ignores the byte count returned by `unix.Write`. A short write with `err == nil` is reported as success, leaving a truncated OSC on the terminal stream. Check `n == len(p)` and return `io.ErrShortWrite` otherwise. Extend the seam/test so a short write is representable and fails without the fix. This flags `ARCH-MOCK`: the current fake models only complete success or error, not the partial-write behavior of the real boundary.
- `cmd/internal/dispatcher/dispatcher.go:60`: the new user-facing `pair notify` command and legacy `--osc` compatibility behavior are not documented in `README.md`. Add invocation, behavior, and tolerant-failure semantics.

4. Minor findings

None.

5. Test coverage notes

Executed successfully:

- Full focused package suite covering codec, command, wrapper, PTY, Couch, dispatcher, runtime bundle, and artifact paths.
- Race suite for `notifyosc`, `wrapcmd`, `ptychild`, and `couchtty`.
- `bash -n bin/pair-notify`.
- Pinned-range `git diff --check`.

The partial-write case is not currently testable through `notifycmd.Runtime`, which is why the short-write defect survives the suite.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Canonical framing and sanitization have one codec; the attention ledger is the single unread authority.
- `ARCH-PURE`: Pass. Codec, ledger, menu projection, rendering, and replay decisions remain separated from Console and OS I/O.
- `ARCH-PURPOSE`: Flag. The delivered replay design is substantive, but the plan’s source-of-truth Core concepts table does not describe it truthfully.
- `ARCH-MOCK`: Flag. Stateful PTY integration is good, but the outer-TTY write fake cannot model partial writes.
- `ARCH-CONSTRAINTS`: Pass. Per-actor state, candidate buffering, pump backpressure, allocations, race behavior, and target workload are bounded and exercised.

7. Plan revision recommendations

Append a `## Revisions` entry explaining that the planned `ptychild.ReplayWindow` entity was not introduced. Replace that row with the actual modified entities:

- `ptychild.Screen` — replay-safe stream offset and notification observations in `screen.go`.
- `ptychild.Child` — retained notification spans and `ReplayThrough` filtering in `child.go`.

Also update the prose at lines 44–47 to use those names and paths.

```findings
findings:
  - id: new
    severity: Critical
    family: plan-code-traceability
    title: |
      Core concepts table names a nonexistent replay entity in an unchanged file
    detail: |
      The plan claims `ptychild.ReplayWindow` lives in modified `cmd/internal/ptychild/replay.go`, but no such entity exists and that file is unchanged. The delivered replay-safe cutoff and notification-span behavior lives across `Screen` and `Child`; append a plan revision and correct the table and description.
  - id: new
    severity: Important
    family: external-write-completeness
    title: |
      Nonblocking notification emission silently accepts short TTY writes
    detail: |
      `OSRuntime.WriteNonblocking` discards the count returned by `unix.Write`, so a partial canonical envelope with nil error is treated as successful. Check for `n != len(p)` and add a fake-runtime regression that fails without the fix (`ARCH-MOCK`).
  - id: new
    severity: Important
    family: user-facing-surface-documentation
    title: |
      README omits the new pair notify command
    detail: |
      The dispatcher adds the user-invokable `pair notify` surface and legacy option behavior, but README.md is unchanged. Document its invocation, canonical OSC behavior, and tolerant hook failure semantics.
```
