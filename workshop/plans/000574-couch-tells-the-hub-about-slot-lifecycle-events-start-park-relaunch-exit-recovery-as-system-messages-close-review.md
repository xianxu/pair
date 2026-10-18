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

---

## Re-review — 2026-10-17T20:26:18-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 574 — Couch tells the hub about slot lifecycle events (start, park, relaunch, exit, recovery) as system messages |
| repo | pair |
| issue file | workshop/issues/000574-couch-tells-the-hub-about-slot-lifecycle-events-start-park-relaunch-exit-recovery-as-system-messages.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ee00c35969770c9e6e6a01282393613617b0305..50f8d6831f6cf62c1ea481de5da9190604022082 |
| command | sdlc close --issue 574 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T20:26:18-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed, with regressions that fail when the old behavior is restored. The affected tests and focused race checks pass. One confirmed routing bug blocks shipping: pending lifecycle notices do not follow a slot’s ownership move.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Recovery constructors preserve Continue, ResumeDisk and Fresh; finishOperation projects the typed mode, including confirmation results. Recovery regressions pass; restoring unconditional Resume makes the Console regression fail.
  - id: BR-2
    disposition: addressed
    note: |
      Completion projection prefers the adopted child's Args.Stack. Regressions cover empty/conflicting menu agents across operation types; removing that preference makes the regression fail.
findings:
  - id: new
    severity: Critical
    family: revalidate-current-recipient-ownership
    title: |
      Pending lifecycle notices remain assigned to the previous owning hub
    detail: |
      cmd/internal/couchcmd/lifecycle_notices.go:167 resolves ownership only when observing a fact, and :210-224 refreshes connections without rerouting existing facts. A disposable regression collected a fact under hub A, moved its slot to hub B, made B ready, and drained the cohort: no notice reached B because the fact still belonged to A. If A remains available, it can receive the stale projection instead. ARCH-PURPOSE, ARCH-ORDER: resolve current ownership before freezing/admitting unsent projections, preserve accepted-input uncertainty, and test ownership changes before drain and admission.
```

1. **Strengths**

   - Typed recovery modes and adopted-agent metadata now survive final completion projection.
   - The collector isolates deterministic transitions from publication IO and bounds facts, payloads and attempts.
   - Canonical origin checks cover receipt recovery, immutable identity and wrapper compatibility.
   - README, atlas and Couch skill document the new surface and trimmed scope.

2. **Critical findings**

   - Ownership routing described above, at `cmd/internal/couchcmd/lifecycle_notices.go:167` and `:210`. Reconcile pending projections with current ownership before publication.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed focused tests across couchcore, couchtty, couchcmd, couchmessage, wrapcmd and artifactpath.
   - Passed focused race checks across the five runtime packages and the complete couchmessage suite.
   - Both prior-fix behavioral negative controls failed as expected.
   - The new ownership-move reproduction failed against pinned head.
   - Disposable copies were removed; the reviewed checkout remains unchanged.

6. **Architectural notes for upcoming work**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: reuses broker, batching and completion authorities. |
   | ARCH-PURE | Pass: collector transitions run without IO. |
   | ARCH-PURPOSE | Flag: pending notices miss the current owning hub. |
   | ARCH-MOCK | Pass: existing stateful endpoint and socket seams are reused. |
   | ARCH-CONSTRAINTS | Pass: bounded state, multipart bodies and admission capacity are exercised. |
   | ARCH-SECURE | Pass: external requests cannot select canonical system origin. |
   | ARCH-ORDER | Flag: ownership changes are absent from pending-projection reconciliation. |
   | ARCH-FUNERAL | Pass: ephemeral collector state and workers end with the service. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining ownership-change transitions for pending and frozen projections, including the admission revalidation point and tests proving that uncertain accepted input is never replayed.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1e15afc6-c4c8-45e2-a7fe-f6425a7a4e96","repository":"github.com/xianxu/pair","issue_id":"000574","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/d5acc848102d83ab","reviewed_head":"50f8d6831f6cf62c1ea481de5da9190604022082","slot":"pair:3","requested_at":"2026-10-18T03:23:37.819756Z"},"phase":"finished","pid":21713,"started_at":"2026-10-18T03:23:38.019848Z","finished_at":"2026-10-18T03:26:17.245684Z","session_id":"01a14d09-5056-75b2-ade7-6f1df363b7ab","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.014729875s
head: "50f8d6831f6cf62c1ea481de5da9190604022082"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T20:57:38-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 574 — Couch tells the hub about slot lifecycle events (start, park, relaunch, exit, recovery) as system messages |
| repo | pair |
| issue file | workshop/issues/000574-couch-tells-the-hub-about-slot-lifecycle-events-start-park-relaunch-exit-recovery-as-system-messages.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ee00c35969770c9e6e6a01282393613617b0305..f962a74b9031836f2783f9d472cc395ff3fd5b57 |
| command | sdlc close --issue 574 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T20:57:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The implementation delivers the revised lifecycle-notice scope. BR-3 is addressed under the documented ownership-change contract, with passing regression tests and a negative control that fails without the fix. No new blocking defects found. Full couchcore verification remains incomplete because the broader run timed out.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      Ownership changes discard unadmitted projections, notify the new owner, and preserve accepted-input uncertainty. TestLifecyclePublisherDropsMovedOwnership passes all four phases; removing the ownershipChanged call makes the regression fail. The disposable checkout was removed.
```

1. **Strengths**

   - Ownership changes fence extracted jobs and preserve original receipts for admitted input (`lifecycle_notices.go:124`, `hub_move.go:125`).
   - The pure collector bounds state, freezes immutable payloads, and handles uncertainty without replay.
   - Origin validation covers admission, receipt recovery, batching, and wrapper delivery.
   - README, atlas, and Couch skill document configuration, provenance, compatibility, and revised scope.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused tests passed across couchcore, couchtty, couchcmd, couchmessage, and wrapcmd, both normally and under the race detector. The BR-3 negative control failed as expected.

   Broader tests passed for couchtty, couchcmd, couchmessage, wrapcmd, and artifactpath. Couchcore reported a repository-scope launch failure and timed out after three minutes. The failing launch test passed with temporary directories outside the checkout; the complete couchcore suite was not subsequently verified.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses routing, broker admission, and batching.
   - **ARCH-PURE — pass:** Collector transitions are deterministic and directly tested without IO.
   - **ARCH-PURPOSE — pass:** Delivers the revised scope and explicit ownership-change policy.
   - **ARCH-MOCK — pass:** Stateful endpoints and owned socket/child seams exercise integration.
   - **ARCH-CONSTRAINTS — pass:** Facts, attempts, payloads, and publication concurrency are bounded.
   - **ARCH-SECURE — pass:** Closed external requests cannot select system origin; receipt imports validate authority.
   - **ARCH-ORDER — pass:** Tests control drain, admission, replacement, and uncertain outcomes.
   - **ARCH-FUNERAL — pass:** New collector state is ephemeral; workers cancel and join with the service.

7. **Plan revision recommendations:** None required.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3d84c556-44af-44da-a8fc-600c358873f0","repository":"github.com/xianxu/pair","issue_id":"000574","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/1e71b7e610ec96de","reviewed_head":"f962a74b9031836f2783f9d472cc395ff3fd5b57","slot":"pair:3","requested_at":"2026-10-18T03:52:18.95488Z"},"phase":"finished","pid":62494,"started_at":"2026-10-18T03:52:19.197935Z","finished_at":"2026-10-18T03:57:37.790147Z","session_id":"01a14d23-941f-7b51-b04f-626d0d5bca5c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.336096s
head: "f962a74b9031836f2783f9d472cc395ff3fd5b57"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
