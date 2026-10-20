# Boundary Review — pair#714 (whole-issue close)

| field | value |
|-------|-------|
| issue | 714 — Cold-start recovery gaps: a failed resume wedges the slot in starting; post-reboot candidate scans incomplete; a codex slot lost its candidate |
| repo | pair |
| issue file | workshop/issues/000714-cold-start-recovery-gaps-a-failed-resume-wedges-the-slot-in-starting-post-reboot-candidate-scans-incomplete-a-codex-slot-lost-its-candidate.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d9ce7ca7988743e3ef201221a19c985c8cecec7..a8e72fd6bacd5ee8156f1459ed2f461103c946d9 |
| command | sdlc close --issue 714 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T21:27:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned changes satisfy #714’s approved failed-launch recovery and diagnostics scope. Confirmed failures become immediately retryable; uncertain attempts retain ownership. No blocking defects found. The recorded full-suite pass remains required before landing.

```findings
{}
```

1. **Strengths**
   - `recovery_launch_disposition.go:19` checks exact helper identity, completed exit, process-group absence and exact terminal evidence before release.
   - `starttransaction.go:94` retains recovery selection while clearing only the matching creating incarnation.
   - `recovery_launch_failure_test.go:16` exercises production dispatch, reopened diagnostics and immediate same-conversation retry.
   - Atomic diagnostic address composition has direct tests, fuzz invariants and a repository lint guard.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused recovery, lifecycle, operation-attempt and transition tests passed across couchcore, couchcmd and repolint.
   - Focused couchcore race tests passed.
   - Complete repolint, artifactpath and threadrecord package tests passed.
   - `git diff --check` passed; checkout unchanged.
   - Live conformance and the recorded full suite were not run during this review.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Reuses `ReconcileStart`, existing stores and recorder.
   - **ARCH-PURE — pass:** Transitions and address composition remain pure; observation and persistence stay in the integration layer.
   - **ARCH-PURPOSE — pass:** Implements the approved scope split.
   - **ARCH-MOCK — pass:** Uses existing stateful process/artifact seams and portable stores; scheduled conformance infrastructure exists.
   - **ARCH-CONSTRAINTS — pass:** Deadline, lock-contention and counted-work tests cover bounded finalization.
   - **ARCH-SECURE — pass:** Exact identities and strict validation remain enforced.
   - **ARCH-ORDER — pass:** Operation hold, revision/nonce fencing and stale-event rejection protect disposition.
   - **ARCH-FUNERAL — pass:** No new durable artifact family; existing diagnostic retention applies.
   - **ARCH-EVOLVE — pass:** No persisted format changes; existing compatibility behavior remains covered.

   Atlas documentation covers the new flow and lint rule. No new command, flag or configuration surface requires a README change.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"25bafec8-d210-4e98-9cfa-1aed675c94a5","repository":"github.com/xianxu/pair","issue_id":"000714","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c27ec91a044352e5","reviewed_head":"a8e72fd6bacd5ee8156f1459ed2f461103c946d9","slot":"pair:3","requested_at":"2026-10-20T04:26:05.661237Z"},"phase":"finished","pid":51916,"started_at":"2026-10-20T04:26:05.996022Z","finished_at":"2026-10-20T04:27:41.043534Z","session_id":"01a1578f-38f7-78c1-83fb-70627f6a4766","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.084462167s
head: "a8e72fd6bacd5ee8156f1459ed2f461103c946d9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
