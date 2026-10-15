# Boundary Review — pair#218 (milestone M1)

| field | value |
|-------|-------|
| issue | 218 — couch thread startup takes 8.85s and nothing has measured where it goes |
| repo | pair |
| issue file | workshop/issues/000218-couch-thread-startup-takes-8-85s-and-nothing-has-measured-where-it-goes.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c0ec24396de2bd1e145c4a9e951859a8985dcaee..7e5a5533d95342af9a3f926f5ad5cee7bf0fa00e |
| command | sdlc milestone-close --issue 218 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T17:02:51-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

M1 delivers narrowed native discovery and timing instrumentation while preserving fresh ownership checks. No blocking defect was confirmed. README and atlas cover the new surface. One live handoff failed before an identical rerun passed, which limits confidence in measurement repeatability.

```findings
{}
```

1. **Strengths**
   - `cmd/internal/procutil/command.go:56` reuses snapshot enrichment and preserves cancellation, unavailable evidence, and generation fencing.
   - `cmd/internal/launcher/session_owner_test.go:193` tests competing claimants both during pane inspection and before effects.
   - `probes/zellijcalls/main.go:62` replaces the server shim process; its regression test checks PID, argv, environment, streams, and exit status.
   - Timing tests verify default silence, consecutive deltas, context scope, and stopping before blocking handoff.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   All four reviewed packages passed: procutil, launcher, Couchcore, and zellijcalls. Focused race tests also passed.

   The live fixture initially failed with “No session … found” at handoff. An identical rerun passed: attach readiness took 2.29 seconds; detach took 330 ms. Server survival, incarnation retirement, and unrelated-sentinel survival were verified. These samples establish no precise speedup. The preserved baseline procutil source hash matches the pinned base.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared enrichment implementation.
   - **ARCH-PURE — pass:** pure selection remains separate from native IO.
   - **ARCH-PURPOSE — pass for M1:** discovery consumers migrated; shared passes and comparative experiments remain M2.
   - **ARCH-MOCK — pass:** stateful process/owner worlds exercise existing production seams.
   - **ARCH-CONSTRAINTS — pass:** narrowed argv work; deadlines and concurrency defaults retained.
   - **ARCH-SECURE — pass:** unavailable evidence refuses authority; timing output excludes commands and paths.
   - **ARCH-ORDER — pass:** fresh enumeration preserves exclusivity across pane IO and effect checks.
   - **ARCH-FUNERAL — pass:** no persistent cache introduced; fixtures provide cleanup.

   Core-concept claims were checked against the superseding revisions. M2 should resolve or classify the intermittent handoff failure before treating successful timing samples as representative.

7. **Plan revision recommendations:** None required for M1. Shared-pass wiring, 6/20/40 measurements, and the timeout-comment update remain explicitly assigned to M2.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.099891s
head: "7e5a5533d95342af9a3f926f5ad5cee7bf0fa00e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
