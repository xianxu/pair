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

---

## Re-review — 2026-08-31T23:44:46-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 158 — couch: actor notifications and attention routing |
| repo | pair |
| issue file | workshop/issues/000158-couch-actor-notifications.md |
| boundary | whole-issue close |
| milestone | — |
| window | f93bd568361d3c26ac46ea487c607ff695407689..b05206fa71a890f0b61d9784b00933c274816934 |
| command | sdlc close --issue 158 |
| reviewer | codex |
| timestamp | 2026-08-31T23:44:46-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation and focused suites are strong, and all three prior findings are addressed. However, the revised Core concepts table still claims generated runtime-bundle files and `atlas/index.md` were modified, although none are present in the pinned diff. The review contract treats a Core concepts contradiction as Critical, so the plan must be corrected before close.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The plan now names the delivered Screen replay-safe offset and Child replay-span entities, with an appended revision explaining the ownership change.
  - id: BR-2
    disposition: addressed
    note: |
      OSRuntime rejects n != len(p), and TestOSRuntimeReportsShortNonblockingWrite directly fails against the prior implementation.
  - id: BR-3
    disposition: addressed
    note: |
      README now documents pair notify, canonical OSC 777 output, legacy options, and tolerant hook failure behavior.
findings:
  - id: new
    severity: Critical
    family: plan-code-traceability
    title: |
      Core concepts table still marks unchanged generated and index files as modified
    detail: |
      workshop/plans/000158-couch-actor-notifications-plan.md:74 claims the runtime pair-notify mirror, runtime manifest, and atlas/index.md are modified, but none appears in the pinned range; the runtime assets are ignored generated files rather than tracked changes. This is the 2nd finding in family plan-code-traceability. Do not patch only this row: state and apply the rule that every Core concepts path/status must resolve against the committed boundary diff, separating tracked modified surfaces from unchanged indexes and derived ignored outputs.
```

1. Strengths

- The canonical codec centralizes sanitization and framing in `notifyosc`, avoiding parallel Pair/Couch protocol definitions.
- `OSRuntime.WriteNonblocking` now validates the complete atomic write, with a regression that directly exercises the production seam.
- Attention state is centralized in the pure `AttentionLedger`, including bounded retention, deduplication, capture-qualified acknowledgement, and overflow rebasing.
- README and `atlas/architecture.md` cover the new command and notification flow.
- Focused verification passed across `notifyosc`, `notifycmd`, `dispatcher`, `wrapcmd`, `ptychild`, `couchcore`, `termcmd`, `couchcmd`, and `couchtty`; `git diff --check` was clean.

2. Critical findings

- [000158 plan:74](/Users/xianxu/workspace/pair/workshop/plans/000158-couch-actor-notifications-plan.md:74): Correct the Core concepts integration row using a full committed-path/status sweep. Split `atlas/architecture.md` as modified; describe runtime assets as derived/ignored if appropriate; do not claim `atlas/index.md` changed.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

The short-write regression would fail without BR-2’s count check. Pure codec/ledger tests, stateful integration coverage, real-PTY conformance, malformed-stream benchmarks, and focused package suites cover the principal failure classes.

6. Architectural notes

- `ARCH-DRY`: Pass—canonical encoding and attention ownership each have one source.
- `ARCH-PURE`: Pass—codec, framing decisions, and ledger transitions are separated from terminal IO.
- `ARCH-PURPOSE`: Pass for runtime behavior; flag for the inaccurate boundary traceability record.
- `ARCH-MOCK`: Pass—production seams are injected and exercised with fakes plus real-PTY conformance.
- `ARCH-CONSTRAINTS`: Pass—retention, scanner memory, batch backpressure, throughput, and UI latency envelopes are bounded and tested.

7. Plan revision recommendations

Append a `## Revisions` entry stating that a committed-range sweep found the bundled integration row conflated tracked modifications, unchanged indexes, and ignored derived outputs. Record the corrected classifications and apply that classification rule to every Core concepts row.
