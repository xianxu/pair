# Boundary Review — pair#865 (milestone M2)

| field | value |
|-------|-------|
| issue | 865 — Live Couch sustains ~1.3 cores: attribute and bound it |
| repo | pair |
| issue file | workshop/issues/000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 57add3c1ffac9c1d8147d7728860fc6a5ccd6abf..9676dff56fc63b95885f3ec084127d766ca1976e |
| command | sdlc milestone-close --issue 865 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T23:06:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned M2 change meets the documented capture-allocation contract while preserving consumer ownership and existing publication behavior. Independent baseline/candidate checks confirm the allocation reduction. No blocking findings; aged whole-process CPU remains outside this correction’s qualified scope.

```findings
{}
```

1. **Strengths**

   - Storage reuse checks geometry and lengths, overwrites every row/cell, and explicitly clears absent cells (`cmd/internal/terminal/endpoint.go:252–268`).
   - Snapshot and Publication still clone private storage; regression tests verify retained publications and consumer edits remain isolated.
   - Allocation tests independently double capture count and screen size (`cmd/internal/terminal/endpoint_capture_test.go:15`).
   - Atlas and allocation evidence document ownership, bounds, and measurement limits. No new user invocation requires a README update.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Focused capture, synchronization, resize, publication, EOF, and history tests passed, including under `-race`.
   - Complete `ptychild` tests and `make test-affected-inventory` passed.
   - Restoring baseline capture through a temporary Go overlay caused all four allocation cases and synchronized backing reuse to fail.
   - Independent 100-iteration benchmarks measured baseline capture at approximately 222/435 KB and two allocations per operation; candidate measured zero bytes and zero allocations at both geometries.
   - Working tree remained clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses existing capture, repair, and clone paths.
   - **ARCH-PURE — pass:** Mutable backend reads remain in Endpoint; listed pure entities retain IO-free tests.
   - **ARCH-PURPOSE — pass:** Delivers the approved allocation correction and semantic controls.
   - **ARCH-MOCK — pass:** Uses the actual emulator and existing stateful transport fake; adds no external dependency.
   - **ARCH-CONSTRAINTS — pass:** N/2N controls enforce zero steady-geometry allocations; geometry limits remain intact.
   - **ARCH-SECURE — pass:** Existing validation/repair remains; stale cells are explicitly cleared.
   - **ARCH-ORDER — pass:** Adds no lifecycle state or asynchronous work; existing mutex and controlled event-sequence tests cover affected transitions.
   - **ARCH-FUNERAL — pass:** No pool or spare geometry cache; replaced backing becomes collectible.
   - **ARCH-EVOLVE — pass:** No persisted format changes.

   Core-concept paths and claimed modification status match the code.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9bae9569-6c28-44c5-9ab5-9fdf28f3aa05","repository":"github.com/xianxu/pair","issue_id":"000865","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/56203746be704796","reviewed_head":"9676dff56fc63b95885f3ec084127d766ca1976e","slot":"pair:9","requested_at":"2026-10-25T06:05:21.369366Z"},"phase":"finished","pid":22282,"started_at":"2026-10-25T06:05:21.468378Z","finished_at":"2026-10-25T06:06:49.557682Z","session_id":"01a171a9-e3e0-7851-8260-a3d1e3d0bf93","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.483102583s
head: "9676dff56fc63b95885f3ec084127d766ca1976e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
