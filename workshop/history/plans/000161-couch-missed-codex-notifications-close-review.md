# Boundary Review — 000161-couch-missed-codex-notifications#161 (whole-issue close)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9a8e88528ab254e0c1d4765bf0a003b692506599..fd95a9e5025508ed674078847bcb42612c796d4d |
| command | sdlc close --issue 161 |
| reviewer | codex |
| timestamp | 2026-09-01T16:25:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The lifecycle reducer, durable transcript path, captured Codex visual recovery, and joined Couch projection are well implemented and thoroughly tested. However, Claude-specific OSC progress authority is currently granted to every wrapped agent. Any agent emitting iTerm progress states `9;4;3` then `9;4;0` can generate a false completion notification. This violates the specified authority boundary and requires rework.

1. Strengths

- The pure reducer cleanly centralizes per-generation deduplication, keyed transcript matching, and tokenized timers in `notification_lifecycle.go`.
- Transcript records use strict decoding, closed semantic validation, stable identities, and exact reconciliation rather than transport identity alone.
- Journal filesystem work is isolated from the PTY loop, and the production `masterPump` regression proves output continues while journal IO blocks.
- The M4 acceptance test feeds wrapper-produced bytes into a running Couch console and verifies both the status chip and switcher message.
- Atlas documentation covers the new lifecycle journal, transcript authority, reducer, and rendered recovery surfaces. No README update is required because no new user-entered command, flag, or configuration key was introduced.

2. Critical findings

- [notification_rewriter.go:73](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/notification_rewriter.go:73), [wrap.go:2744](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/wrap.go:2744) — Claude-specific OSC progress authority is not scoped to Claude. `Feed` produces lifecycle observations regardless of `normalize` or agent, and `handleChunk` unconditionally reduces them. Thus Codex, Muse, Agy, or any other wrapped program can emit `OSC 9;4;3` followed by `OSC 9;4;0` and receive a generic completion after grace. The issue grants these transitions only to Claude. Gate lifecycle progress observations at an agent-aware boundary, and add production-path negative tests for every non-Claude supported agent plus the positive Claude case.

  This is the 5th finding in family `lifecycle-contract-coverage`. Earlier rounds fixed instances. Do not fix only one named agent: state and enforce the complete source-authority rule mapping every lifecycle signal to its permitted agents.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- `go test ./... -count=1` passed.
- `git diff --check` passed.
- The prior findings BR-1 through BR-11 have reachable regressions covering submission chords, native completion without injected progress, artifact inventory, reducer ordering/keying, blocked journal IO through `masterPump`, entity inventory, live lifecycle validation, pinned M3 changes, strict journal grammar, semantic reconciliation, and wrapper-to-Couch delivery.
- The missing negative test is agent scoping for OSC progress. Existing every-split tests exercise the parser without agent identity, so they cannot detect the production authority leak.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Completion sources converge on one reducer and one `emitOuter` sink.
- `ARCH-PURE`: Pass. Lifecycle policy and rendered-state recognition are pure; IO remains in watcher/tailer/proxy seams.
- `ARCH-PURPOSE`: Flagged. Claude-only progress authority was broadened to all agents, violating the issue’s declared source contract.
- `ARCH-MOCK`: Pass. Stateful filesystem/runtime and PTY/Couch fakes exercise the same production boundaries.
- `ARCH-CONSTRAINTS`: Pass. Journal reads are bounded, PTY forwarding is isolated from journal IO, channels are bounded, and polling intervals match the declared envelope.

7. Plan revision recommendations

Add a `## Revisions` entry stating the complete agent/source authority matrix, including:

- Claude: progress OSC and finalized Claude marker.
- Codex: native OSC, authorized transcript records, and rendered Working transition.
- Other agents: only their explicitly authorized existing sources.
- Every source must have positive tests for authorized agents and negative production-path tests across all unauthorized supported agents.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Legacy and KKP Alt+Enter production paths publish submission boundaries, with plain Enter remaining non-authoritative.
  - id: BR-2
    disposition: addressed
    note: |
      Native completion regressions now traverse real send boundaries without an unrelated progress opener.
  - id: BR-3
    disposition: addressed
    note: |
      The lifecycle sources are present in the exhaustive artifact manifest.
  - id: BR-4
    disposition: addressed
    note: |
      Reducer tests cover source ordering, keyed mismatch, new keyed turns, and abort outcomes.
  - id: BR-5
    disposition: addressed
    note: |
      Journal advancement runs in a worker, and a masterPump test proves PTY forwarding while injected journal IO blocks.
  - id: BR-6
    disposition: addressed
    note: |
      The Core concepts table names greppable entities with accurate delivered milestone statuses.
  - id: BR-7
    disposition: addressed
    note: |
      The opt-in live conformance invokes lifecycle validation for keyed same-root opener and terminal ordering.
  - id: BR-8
    disposition: addressed
    note: |
      The pinned range contains the M3 implementation and subsequent fixes.
  - id: BR-9
    disposition: addressed
    note: |
      Writer and tailer share strict decoding and closed validation of lifecycle record semantics.
  - id: BR-10
    disposition: addressed
    note: |
      Reconciliation requires complete semantic equality, with a producer collision matrix covering each field.
  - id: BR-11
    disposition: addressed
    note: |
      The joined acceptance test feeds wrapper-emitted bytes into the running Couch console and verifies both named surfaces.
findings:
  - id: new
    severity: Critical
    family: lifecycle-contract-coverage
    title: |
      Claude progress OSC grants completion authority to every wrapped agent
    detail: |
      NotificationRewriter emits Working and Stopped observations independently of agent identity, and handleChunk reduces them unconditionally. This is the 5th finding in family lifecycle-contract-coverage: define and enforce the complete per-agent source-authority matrix, with negative production-path coverage for every unauthorized supported agent.
```

---

## Re-review — 2026-09-01T16:31:28-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9a8e88528ab254e0c1d4765bf0a003b692506599..9cec68e3a78ef1146983f74e5596c96b3caa07c6 |
| command | sdlc close --issue 161 |
| reviewer | codex |
| timestamp | 2026-09-01T16:31:28-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range fulfills the issue’s lifecycle-notification contract, and BR-12 is addressed by an agent-aware production gate plus a mutation-sensitive authority-matrix regression. The implementation centralizes deduplication, preserves strict Codex transcript identity, isolates journal IO from the PTY loop, and verifies both Couch notification surfaces. No new blocking findings remain.

1. Strengths

- The pure reducer centralizes generation tracking, keyed completion, deduplication, and timer decisions in [notification_lifecycle.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/notification_lifecycle.go:62).
- BR-12’s source rule is enforced before reduction in [wrap.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/wrap.go:2745): only Claude’s progress OSC mutates lifecycle state.
- The production-path matrix covers Claude positively and Codex, Agy, and Muse negatively in [notification_rewriter_test.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/notification_rewriter_test.go:207), while confirming byte transparency.
- Transcript lifecycle records use closed Codex-only semantic validation in [lifecycle_event.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessionwatch/lifecycle_event.go:38).
- Atlas documentation covers the new lifecycle architecture and identity flow. No README update is required because no user-entered command, flag, or configuration surface was added.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- Focused BR-12 tests passed.
- Mutation verification changed `progressOSCAuthorized` to authorize every agent; the matrix test then failed for Codex, Agy, and Muse. The regression therefore fails without the fix.
- All ordinary packages in `go test ./... -count=1` passed except `cmd/pair-go`, whose tests could not execute `/bin/ps` under the review sandbox (`operation not permitted`). This is an environmental restriction, not a product failure.
- `git diff --check` passed.
- Core-concept entities exist at their documented paths, with pure entities directly unit-tested and integration seams exercised through stateful runtime, filesystem, PTY, and Couch fakes.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass — all completion sources converge on one reducer and `emitOuter` boundary.
- `ARCH-PURE`: Pass — lifecycle and rendered-screen policy are pure; IO remains in watcher, journal, and proxy adapters.
- `ARCH-PURPOSE`: Pass — Codex short/long transcript completion, native notification, rendered recovery, Claude progress/marker behavior, deduplication, and both Couch surfaces are delivered.
- `ARCH-MOCK`: Pass — stateful fakes exercise the production seams, with opt-in live Codex conformance covering external-shape drift.
- `ARCH-CONSTRAINTS`: Pass — polling, record size, channel capacity, incremental tailing, and PTY non-blocking behavior match the declared envelope.

7. Plan revision recommendations

None. The existing `## Revisions` entry accurately records the complete per-agent authority matrix and its production-path coverage.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Legacy and KKP Alt+Enter publish submission boundaries; plain Enter remains non-authoritative.
  - id: BR-2
    disposition: addressed
    note: |
      Native completion tests traverse real submission boundaries without an unrelated progress opener.
  - id: BR-3
    disposition: addressed
    note: |
      Lifecycle sources are included in the exhaustive artifact inventory.
  - id: BR-4
    disposition: addressed
    note: |
      Reducer coverage pins source ordering, keyed mismatch, new keyed turns, and abort outcomes.
  - id: BR-5
    disposition: addressed
    note: |
      Journal IO runs outside the PTY loop, with blocked-IO forwarding coverage.
  - id: BR-6
    disposition: addressed
    note: |
      The Core concepts table names existing entities at accurate paths and statuses.
  - id: BR-7
    disposition: addressed
    note: |
      Live conformance checks keyed Codex opener/terminal ordering and identity.
  - id: BR-8
    disposition: addressed
    note: |
      The pinned range contains the claimed M3 implementation and fixes.
  - id: BR-9
    disposition: addressed
    note: |
      Lifecycle records use strict framing and closed semantic validation.
  - id: BR-10
    disposition: addressed
    note: |
      Reconciliation requires exact semantic equality and is covered by the collision matrix.
  - id: BR-11
    disposition: addressed
    note: |
      Joined acceptance coverage feeds wrapper output into Couch and verifies status and switcher state.
  - id: BR-12
    disposition: addressed
    note: |
      Claude-only progress authority is gated before reduction; the production matrix covers every supported unauthorized agent and fails when the gate is removed.
```
