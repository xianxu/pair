# Boundary Review — pair#218 (whole-issue close)

| field | value |
|-------|-------|
| issue | 218 — couch thread startup takes 8.85s and nothing has measured where it goes |
| repo | pair |
| issue file | workshop/issues/000218-couch-thread-startup-takes-8-85s-and-nothing-has-measured-where-it-goes.md |
| boundary | whole-issue close |
| milestone | — |
| window | a22b501b10674eaf44452a4aecb902761b7396f5..36e8111e58ba6d7a4c9669bc083157cfcc8582f0 |
| command | sdlc close --issue 218 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T22:47:22-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The implementation matches the final revised scope: candidate-only process enrichment and shared initial ownership observations, with fresh authority checks preserved. No actionable defects found. Cold attribution and the incomplete concurrency experiment remain explicitly tracked in #560/#562/#563. Confidence is medium because the full package run encountered one timing-sensitive test failure.

```findings
{}
```

1. **Strengths**

   - `cmd/internal/launcher/session_owner_pass.go:33` captures once, clones evidence, memoizes errors, and isolates sibling cancellation.
   - Fresh after-pane and effect checks remain intact; tests cover competing claimants, changed generations, sockets, and disappearance.
   - `cmd/internal/couchcore/launch_existing.go:403` bypasses cached observations during registration, allowing newly born servers to become visible.
   - README and atlas document the timing flag, measurement limits, and pass lifecycle.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed process-reader, launcher, Console, probe, targeted fleet-harness, artifact, and diagnostic regression checks, plus targeted race tests across all four affected core packages.

   The full Couch package run failed `TestRetentionExplicitResumeTouchesButBackgroundReattachDoesNot/false`: its 200 ms registration deadline expired before any ownership poll. The unchanged test passed three isolated repetitions. This is consistent with deadline sensitivity under suite load; the full run was not green.

   Live fleet measurements and native Linux conformance were not rerun.

6. **Architectural notes**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass: common snapshot and ownership logic reused. |
   | ARCH-PURE | Pass: candidate/classification logic remains pure; IO stays behind injected seams. |
   | ARCH-PURPOSE | Pass against the explicit final scope revision; deferred measurements have tracker issues. |
   | ARCH-MOCK | Pass: stateful process/owner worlds exercise production boundaries; live conformance remains available. |
   | ARCH-CONSTRAINTS | Pass: capture/query bounds retained; concurrency default unchanged without sufficient evidence. |
   | ARCH-SECURE | Pass: unavailable identity evidence preserves uncertainty; diagnostics avoid argv disclosure. |
   | ARCH-ORDER | Pass: existing reducer governs reattach progress; controlled tests cover cancellation and changing evidence. |
   | ARCH-FUNERAL | Pass: pass contexts are released on completion/teardown; fixture cleanup is scoped. |

7. **Plan revision recommendations:** None. Existing revisions explicitly supersede the deferred acceptance items without marking them complete.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.824158167s
head: "36e8111e58ba6d7a4c9669bc083157cfcc8582f0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
