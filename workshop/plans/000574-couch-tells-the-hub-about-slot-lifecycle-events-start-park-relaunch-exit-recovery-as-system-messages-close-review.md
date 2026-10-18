# Boundary Review — pair#574 (whole-issue close)

| field | value |
|-------|-------|
| issue | 574 — Couch tells the hub about slot lifecycle events (start, park, relaunch, exit, recovery) as system messages |
| repo | pair |
| issue file | workshop/issues/000574-couch-tells-the-hub-about-slot-lifecycle-events-start-park-relaunch-exit-recovery-as-system-messages.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ee00c35969770c9e6e6a01282393613617b0305..f7dcd06d22ebd05a56ce8b618ae0167df7bc5285 |
| command | sdlc close --issue 574 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T19:58:33-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation provides bounded collection, restart coalescing, canonical system provenance, and the required README/atlas updates. Two completion-projection defects block shipping: recovery modes can be misreported, and successful starts can report stale agent metadata. Review used the issue’s explicitly revised scope.

```findings
findings:
  - id: new
    severity: Critical
    family: preserve-confirmed-recovery-mode
    title: |
      RecoveryResumeResult is incorrectly treated as proof of Resume mode
    detail: |
      cmd/internal/couchtty/console_lifecycle.go:98 unconditionally labels every RecoveryResumeResult as recovered (Resume). That result also represents Continue (couchcore/recovery_prepared.go:91) and Fresh (recovery_prepared.go:435), including recovery-confirmation completions. The hub can therefore be told a conversation resumed when it actually started fresh. ARCH-PURPOSE: carry the actual typed recovery mode through the result and test successful Continue, Resume, Fresh, and confirmation paths through finishOperation.
  - id: new
    severity: Critical
    family: prefer-confirmed-completion-metadata
    title: |
      Started notices use the preceding menu agent instead of the adopted child's agent
    detail: |
      cmd/internal/couchtty/console_lifecycle.go:79 obtains Agent from the existing menu row and never overrides it from child.Record.Args.Stack. A successful agent switch can consequently announce the previous agent; a never-started row can omit the agent entirely. The confirmed launch already supplies the actual agent (couchcore/couch.go:488 and switchagent.go:415), while menu refresh occurs asynchronously after attachment. ARCH-PURPOSE: prefer the completed child's agent and add regressions with empty and conflicting pre-operation menu metadata.
```

1. **Strengths**

   - The collector is an IO-free reducer with bounded facts, immutable publication parts, and explicit receipt transitions.
   - Origin matching covers admission, queues, retained recovery, envelopes, and wrapper batch identity.
   - Socket-backed tests exercise busy hubs, recipient replacement, shutdown, and canonical receipts.
   - README, Couch skill, atlas, and artifact inventory updates cover the introduced surface.

2. **Critical findings**

   Both findings above require fixes and regression tests before crossing the boundary.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused lifecycle/origin tests passed across all five affected behavioral packages. Full package tests passed for `couchtty`, `couchcmd`, `couchmessage`, `wrapcmd`, and `artifactpath`.

   The full `couchcore` run failed with production-registration errors reporting `couch session repository scope changed`, then reached the 180-second timeout. That run does not establish a clean full-suite result; I have not attributed those failures to this diff.

   Existing lifecycle tests omit successful recovery-mode assertions and completed-child agent precedence, allowing both reported defects through. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses broker delivery and existing completion authority.
   - **ARCH-PURE — pass:** collector logic is directly testable without IO.
   - **ARCH-PURPOSE — flag:** confirmed lifecycle information loses recovery-mode and agent accuracy.
   - **ARCH-MOCK — pass:** stateful endpoints and portable socket fixtures share production boundaries.
   - **ARCH-CONSTRAINTS — pass:** bounded collection/publication; callbacks avoid discovery IO.
   - **ARCH-SECURE — pass:** external requests cannot assign canonical system origin.
   - **ARCH-ORDER — pass:** publication transitions preserve frozen revisions and uncertainty.
   - **ARCH-FUNERAL — pass:** new projection state is bounded and service-owned; workers join on shutdown.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying typed recovery-mode propagation and completed-child metadata precedence, with successful completion regressions covering both. The approved scope need not change.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c1a27c7f-2f04-4a32-a784-dfab7b64b45e","repository":"github.com/xianxu/pair","issue_id":"000574","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/b05fce0cf4d231fb","reviewed_head":"f7dcd06d22ebd05a56ce8b618ae0167df7bc5285","slot":"pair:3","requested_at":"2026-10-18T02:53:28.302702Z"},"phase":"finished","pid":30373,"started_at":"2026-10-18T02:53:28.559104Z","finished_at":"2026-10-18T02:58:32.920539Z","session_id":"01a14ced-b496-7550-9bec-ce977a6d9751","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.103446125s
head: "f7dcd06d22ebd05a56ce8b618ae0167df7bc5285"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
