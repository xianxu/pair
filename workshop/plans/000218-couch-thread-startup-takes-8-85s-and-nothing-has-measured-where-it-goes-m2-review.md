# Boundary Review — pair#218 (milestone M2)

| field | value |
|-------|-------|
| issue | 218 — couch thread startup takes 8.85s and nothing has measured where it goes |
| repo | pair |
| issue file | workshop/issues/000218-couch-thread-startup-takes-8-85s-and-nothing-has-measured-where-it-goes.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | ece8248c900cf3559abfcb87d62651df66717887..b650fe71f5c093054a40e2ebe0fb87c798ae40aa |
| command | sdlc milestone-close --issue 218 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T22:38:02-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised M2 acceptance: shared initial observations preserve fresh ownership fences, registration discovers new generations, and cancellation remains scoped correctly. The tracker explicitly records the incomplete performance experiment and unchanged concurrency default. No blocking findings.

```findings
{}
```

1. **Strengths**

   - `launcher/session_owner_pass.go:33` captures once, copies the map, and memoizes failures without per-slot fallback.
   - `couchcore/session_owner_pass.go:49` binds shared evidence to the checker and data root; registration explicitly bypasses it.
   - `couchtty/console.go:1039` releases pass state during teardown and invokes cleanup outside the mutex.
   - Identity-read errors preserve uncertainty through downstream classifiers; tests distinguish identity `EINVAL` from command-read disappearance.
   - README and atlas describe the diagnostic output and observation lifecycle.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Full tests for procutil, launcher, couchcore, couchtty, couchcmd, artifactpath, and diagnosticlog.
   - Focused race tests covering capture sharing, cancellation, competing claimants, registration bypass, Leave wiring, console cleanup, identity errors, and query observers.
   - `git diff --check`.

   Tests use controllable barriers and stateful worlds to exercise ordering and authority invariants. The opt-in live fleet experiment and native Linux execution were not rerun; no completed benchmark or concurrency recommendation is inferred.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass — shared probe logic and existing discovery seams remain authoritative. |
   | ARCH-PURE | Pass — classification stays separate from observation IO. |
   | ARCH-PURPOSE | Pass — revised acceptance is delivered; deferred measurements remain explicitly incomplete in the tracker. |
   | ARCH-MOCK | Pass — injected stateful worlds cover observation and lifecycle interactions. |
   | ARCH-CONSTRAINTS | Pass — capture has a timeout, fanout remains bounded, and production parallelism is unchanged. |
   | ARCH-SECURE | Pass — ambiguous evidence refuses authority; cached observations cannot replace fresh effect fences. |
   | ARCH-ORDER | Pass — existing reattach states govern completion; deterministic tests cover cancellation and changed generations. |
   | ARCH-FUNERAL | Pass — scopes are released on completion/teardown; controlled fixtures have cleanup paths. |

   Core-concept locations and classifications match the implementation when read with the plan’s explicit superseding revisions.

7. **Plan revision recommendations:** None. Existing revisions clearly distinguish delivered work from the deferred cold attribution and fleet comparison.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.633331791s
head: "b650fe71f5c093054a40e2ebe0fb87c798ae40aa"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
