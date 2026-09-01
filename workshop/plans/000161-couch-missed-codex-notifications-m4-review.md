# Boundary Review — 000161-couch-missed-codex-notifications#161 (milestone M4)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 2fb98f8bae84baf80736ef2f7524cf18e7a65bc2..8fed78d330a8e72e816fc0d204ece81a7ad41c7c |
| command | sdlc milestone-close --issue 161 --milestone M4 |
| reviewer | codex |
| timestamp | 2026-09-01T16:15:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The rendered Codex recovery is cleanly separated, uses a captured PTY transition, reaches the existing lifecycle reducer, and emits the expected canonical OSC. The full Go suite and diff hygiene pass. One Important coverage gap prevents SHIP: the claimed production acceptance test does not connect wrapper-produced bytes to Couch; it separately synthesizes the same OSC at the Couch boundary.

1. Strengths

- [codex_working.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/codex_working.go:13) keeps recognition pure and the proxy adapter thin.
- The recognizer rejects quoted, prose, completed, unrelated, and misplaced lookalikes.
- [codex_working_test.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/codex_working_test.go:70) replays captured terminal output through `handleChunk`, verifies activity-to-stop lifecycle state, and confirms canonical OSC output.
- The production source manifest and atlas both cover the new architectural surface.
- No duplicated completion policy was introduced; visual recovery feeds the existing reducer and sink.

2. Critical findings

None.

3. Important findings

- [notification_test.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/couchtty/notification_test.go:124) — `ARCH-PURPOSE`: the test named `TestCanonicalWrapperNotificationProjectsToStatusAndSwitcher` creates its input directly with `notifyosc.Encode`. It therefore cannot detect a disconnect or incompatibility between wrapper emission and Couch ingestion. The wrapper test independently compares output with the same encoder, so both tests can remain green without any production boundary joining them.

  This is the 3rd finding in family `submission-boundary-reachability`. Earlier rounds fixed instances. Do not fix only this test site: state and enforce the rule that acceptance tests for cross-package delivery must carry bytes produced by the upstream production adapter into the downstream production consumer. Add a harness that replays the Codex fixture through `handleChunk`, captures the resulting wrapper bytes, feeds those exact bytes into Couch, and asserts both status and switcher projections.

4. Minor findings

None.

5. Test coverage notes

- `go test ./... -count=1` passed.
- Focused `wrapcmd`, `couchtty`, `artifactpath`, and `sessioninventory` tests passed.
- `git diff --check` passed.
- The visual recovery regression would fail if the recognizer/adapter were removed.
- The Couch projection test only validates pre-existing projection behavior and does not fail when the M4 wrapper-to-Couch connection is absent.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass — all completion sources converge on the existing lifecycle reducer and `emitOuter`.
- `ARCH-PURE`: Pass — rendered-screen recognition is deterministic; terminal IO remains in the proxy adapter.
- `ARCH-PURPOSE`: Flagged — the named end-to-end acceptance boundary is not exercised.
- `ARCH-MOCK`: Pass — captured stateful PTY replay uses the production terminal model; live lifecycle conformance remains present.
- `ARCH-CONSTRAINTS`: Pass — recognition is synchronous but bounded by the already-capped terminal dimensions, with no new IO, fan-out, or blocking work.

7. Plan revision recommendations

Add a `## Revisions` entry stating that M4 review found the wrapper-to-Couch acceptance boundary split across two tests. Uncheck Task 7’s production-acceptance item until one test transports wrapper-produced bytes into Couch and observes both surfaces.

```findings
findings:
  - id: new
    severity: Important
    family: submission-boundary-reachability
    title: |
      The claimed wrapper-to-Couch production acceptance path is split across disconnected tests
    detail: |
      This is the 3rd finding in family `submission-boundary-reachability`. `codex_working_test.go` verifies wrapper output against `notifyosc.Encode`, while `notification_test.go` independently injects bytes from that same encoder rather than bytes emitted by the wrapper. State and enforce the class rule that cross-package acceptance tests transport upstream production output into the downstream production consumer; replay the Codex fixture through `handleChunk`, feed those exact emitted bytes into Couch, and assert both status and switcher projections.
```
