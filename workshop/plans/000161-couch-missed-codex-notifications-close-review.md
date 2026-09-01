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
