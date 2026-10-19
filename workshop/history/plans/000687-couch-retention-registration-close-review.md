# Boundary Review — pair#687 (whole-issue close)

| field | value |
|-------|-------|
| issue | 687 — Recognize Couch supervisor retention registration |
| repo | pair |
| issue file | workshop/issues/000687-couch-retention-registration.md |
| boundary | whole-issue close |
| milestone | — |
| window | 213f82ae5b9471b81d75bb9b8cd4e550b0274e1d..230a100ed9601fbb70209de6ad677f19ea8af004 |
| command | sdlc close --issue 687 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T21:51:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation preserves identity fences, open-holder protection, bounded registry writes, and sweep-local observations. Relevant tests passed, and no production correctness defect was found. One test violates the explicitly required PURE-entity gate and needs a small correction before re-review.

```findings
findings:
  - id: new
    severity: Critical
    family: pure-entity-test-isolation
    title: |
      ARCH-PURE: the PURE participation predicate is tested through filesystem IO and a fake probe
    detail: |
      cmd/internal/diagnosticlog/runtime_registry_test.go:48 calls supervisorFixture, which creates a temporary coordinator, injects leaseProcessWorld, and publishes runtimes.json; line 49 then reads that file. This violates the supplied PURE-entity test requirement for the predicate classified PURE in the plan at line 33. Construct ProcessRegistration and Registration literals directly for this table test, leaving registry IO and process doubles in the existing integration tests.
```

1. **Strengths**

   - Couch publishes its capability before admitting the coordinated store (`cmd/internal/couchcore/couch.go:168`).
   - Recognition requires supported capability, Couch role, exact PID/birth, and a fresh liveness check; native observations remain unchanged (`cmd/internal/diagnosticlog/runtime_registry.go:95`).
   - Registry rewrites preserve unknown members at all three required depths while rejecting ambiguous and malformed known data.
   - Production composition tests cover ordinary maintenance and explicit/scheduled collection, including lease changes and held payloads.

2. **Critical findings**

   - The PURE predicate test uses an IO-backed fixture, as detailed above. Severity follows the supplied Core concepts gate. The predicate itself is pure; retain that classification and replace its test setup with literals.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed full `storagegc`, `diagnosticlog`, and `gcruntime` packages; targeted Couch publication tests; mandatory native owned-child conformance without skipping; artifact classification and Godoc guards. `git diff --check` passed. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing registry, publication, and inspection seams are reused.
   - **ARCH-PURE — flag:** isolated predicate testing needs the correction above.
   - **ARCH-PURPOSE — pass:** rooted maintenance and explicit/scheduled GC consume registration evidence.
   - **ARCH-MOCK — pass:** stateful injected observations and owned native qualification exercise the boundary.
   - **ARCH-CONSTRAINTS — pass:** bounded metadata and N=8/16 constant-acquisition tests support the declared envelope.
   - **ARCH-SECURE — pass:** strict known-field validation and exact identity checks preserve conservative failure behavior.
   - **ARCH-ORDER — pass:** tests exercise identity changes, cancellation, publication failure/retry, and refreshed sweep evidence.
   - **ARCH-FUNERAL — pass:** existing bounded registry and dead-process pruning remain the lifecycle owner.

   Atlas documents the new architectural behavior. No new operator-facing command, flag, or configuration setting requires a README update. The operator restart verification remains explicitly deferred under the approved rollout arrangement.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording that predicate tests now use IO-free value fixtures, while registry and process-probe behavior remains covered by integration tests.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a996d582-a0d5-417d-8313-53923948b521","repository":"github.com/xianxu/pair","issue_id":"000687","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ac20ddb4b7aad3b3","reviewed_head":"230a100ed9601fbb70209de6ad677f19ea8af004","slot":"pair:3","requested_at":"2026-10-19T04:48:00.014053Z"},"phase":"finished","pid":90383,"started_at":"2026-10-19T04:48:00.793076Z","finished_at":"2026-10-19T04:50:59.348486Z","session_id":"01a1527c-ed66-78c1-b57d-444794477e6b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.252295334s
head: "230a100ed9601fbb70209de6ad677f19ea8af004"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T21:55:26-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 687 — Recognize Couch supervisor retention registration |
| repo | pair |
| issue file | workshop/issues/000687-couch-retention-registration.md |
| boundary | whole-issue close |
| milestone | — |
| window | 213f82ae5b9471b81d75bb9b8cd4e550b0274e1d..f479a6e80c2849d687539c6c4377d31b0fdc12c5 |
| command | sdlc close --issue 687 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T21:55:26-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the Spec and Plan. BR-1 is addressed, production lease recognition is covered by a regression that fails when its wiring is removed, and no new blocking findings remain. Verification of the running supervisor remains the documented next operator restart check.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      runtime_registry_test.go:47 now tests matchesRuntimeRegistration using value literals only. The pinned correction removes filesystem setup, registry reads and the fake probe; the test passes. This changes test isolation, not production behavior.
```

1. **Strengths**

   - Exact role, capability and PID/birth matching precede a fresh liveness check ([runtime_registry.go:95](/Users/xianxu/Library/Caches/sdlc/review-checkouts/add8c30cf075554b/cmd/internal/diagnosticlog/runtime_registry.go:95)).
   - Exact open-holder protection remains intact, including registered supervisors.
   - Codec tests cover unknown-member preservation, malformed known fields, duplicate keys and ambiguous JSON.
   - Ordinary maintenance, explicit GC and scheduled GC exercise production composition.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Full `storagegc`, `diagnosticlog` and `gcruntime` packages passed. The Couch constructor regression, artifact classification guard and Godoc guard passed. Codec fuzzing completed 118,110 executions without failure.

   Temporarily removing production lease wiring through a Go overlay caused `TestRuntimeRegistryOrdinaryWriterMaintenance` to fail at its registered-supervisor assertion. The pinned implementation passes. Tracked files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Registry ownership and lease consumption each have one implementation.
   - **ARCH-PURE — pass:** Predicate and codec tests use value/byte fixtures; IO stays in integration tests.
   - **ARCH-PURPOSE — pass:** Consumer sweep confirms ordinary maintenance and explicit/scheduled GC use the adapter.
   - **ARCH-MOCK — pass:** Stateful process/lease fixtures share production seams; owned-child conformance exercises native identity and FD inspection.
   - **ARCH-CONSTRAINTS — pass:** Existing metadata bounds remain; N=8/16 tests confirm constant sweep acquisitions.
   - **ARCH-SECURE — pass:** Strict parsing, identity fences and conservative uncertainty handling remain enforced.
   - **ARCH-ORDER — pass:** Tests exercise lease removal, identity replacement, cancellation and changed holders between preview and apply.
   - **ARCH-FUNERAL — pass:** Existing bounded registry and dead-process pruning are reused; no new durable artifact family appears.

   Atlas documents the new flow. No new user-facing command, flag or configuration key requires a README change.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"cebb990f-ab08-443c-ba8d-c19db34fc24b","repository":"github.com/xianxu/pair","issue_id":"000687","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/add8c30cf075554b","reviewed_head":"f479a6e80c2849d687539c6c4377d31b0fdc12c5","slot":"pair:3","requested_at":"2026-10-19T04:53:05.130222Z"},"phase":"finished","pid":85088,"started_at":"2026-10-19T04:53:05.336179Z","finished_at":"2026-10-19T04:55:26.035434Z","session_id":"01a15281-925d-7410-9a95-0c6c8c7496cb","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.660410375s
head: "f479a6e80c2849d687539c6c4377d31b0fdc12c5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
