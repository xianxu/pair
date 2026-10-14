# Boundary Review — pair#542 (whole-issue close)

| field | value |
|-------|-------|
| issue | 542 — PTY canceled-before-ack reports a live process group after the child is reaped |
| repo | pair |
| issue file | workshop/issues/000542-pty-canceled-before-ack-reports-a-live-process-group-after-the-child-is-reaped.md |
| boundary | whole-issue close |
| milestone | — |
| window | ccc0ee88ef133fdcf1d5902e9d505e014d046e48..b31ad711c02706281e933fab1d272b39544f848b |
| command | sdlc close --issue 542 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T19:35:14-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change fixes Darwin’s ambiguous EPERM result without weakening descendant ownership. Both runners share the corrected probe, and the survivor fixture establishes signal-handler readiness before parent exit. No blocking findings.

1. **Strengths**
   - `runner.go:209`: shared logic preserves ownership when membership queries fail.
   - `runner_group_darwin_test.go:16`: regression uses a real reaped child and kernel membership query. Restoring the old EPERM behavior made it fail.
   - `ptyrunner_test.go:158`: FIFO handshake removes the startup race; both runner variants verify surviving descendants.
   - `atlas/couch.md:2553`: accurately documents platform behavior. No new user-facing surface requires README changes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Independently passed 50 repetitions of cancellation, membership and survivor tests; focused race tests also passed.
   - Inspected retained reproduction evidence, low-load and parallel-load qualification, and the successful full-suite log. The full suite was not independently rerun.
   - Fresh artifact-package tests failed because this checkout lacks generated runtime assets. The generation prerequisite is documented in `Makefile.local`; the missing assets and affected baseline expectations are unchanged by this range.
   - Temporary mutation overlay was removed; tracked files remain unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** exec and PTY handles consume one shared ownership probe.
   - **ARCH-PURE — pass:** OS membership inspection remains a small, injected boundary; no business logic was moved into it.
   - **ARCH-PURPOSE — pass:** empty groups, surviving descendants and uncertain membership are covered. Retained evidence supports both requested load conditions.

7. **Plan revision recommendations:** None. The recorded revisions match the implementation; no Core concepts table is claimed.

```findings
{}
```
