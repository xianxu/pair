# Boundary Review — pair#865 (whole-issue close)

| field | value |
|-------|-------|
| issue | 865 — Live Couch sustains ~1.3 cores: attribute and bound it |
| repo | pair |
| issue file | workshop/issues/000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4e3764c65acfff86a9073f1546e38251798005b7..32688203878a60d27f756ad6f075c5b17f48a04e |
| command | sdlc close --issue 865 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T23:22:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range fulfills the documented attribution and capture-allocation correction. Independent checks reproduce the allocation improvement and confirm that the regression tests fail without the fix. Consumer ownership and synchronization semantics remain intact. No blocking findings; the documents correctly leave aged whole-process CPU and fseventsd causality unresolved.

```findings
{}
```

1. **Strengths**

   - `cmd/internal/terminal/endpoint.go:252` reuses storage only when geometry and lengths match, overwrites every row/cell, and clears absent cells.
   - Snapshot and Publication retain independent clones. Tests verify retained publications survive recapture and resize, and consumer edits cannot mutate private storage.
   - `cmd/internal/terminal/endpoint_capture_test.go:15` independently doubles capture count and screen size, enforcing zero allocations after warmup.
   - Live profile hashes and executable metadata match the evidence. Re-running pprof confirms 44.80 CPU seconds over 30.22 seconds and capture’s 46.11% sampled allocation share.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently verified:

   - Focused capture, synchronization, resize, publication, EOF and history tests pass, including under `-race`.
   - Complete `ptychild` tests and pure Frame, Geometry and ProfileLifecycle controls pass.
   - Restoring baseline capture through a temporary overlay makes all four allocation cases fail with 128/256 allocations; synchronized backing reuse also fails.
   - Removing nil-cell clearing makes the absent-cell regression fail with stale `B` content.
   - Identical 100-iteration capture benchmarks measure baseline approximately 222/435 KB and two allocations per operation; candidate measures zero bytes and allocations at both geometries.
   - Reader inventory and pinned-range whitespace checks pass. Temporary overlays were removed; the checkout remains clean.

6. **Architectural notes**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass — retains existing capture, repair, clone and publication paths. |
   | ARCH-PURE | Pass — backend reads stay in Endpoint; listed pure entities have IO-free tests. |
   | ARCH-PURPOSE | Pass — delivers the approved measured correction without claiming resolution of unmeasured aged CPU. |
   | ARCH-MOCK | Pass — uses the actual emulator and existing stateful transport fake; no new external dependency. |
   | ARCH-CONSTRAINTS | Pass — N/2N tests enforce the allocation budget; existing geometry limits remain. |
   | ARCH-SECURE | Pass — validation and repair remain intact; reused storage is fully overwritten. |
   | ARCH-ORDER | Pass — no new states or asynchronous work; existing mutex ownership and controlled sequence tests cover affected transitions. |
   | ARCH-FUNERAL | Pass — no pool or spare cache; replaced buffers become collectible with existing ownership. |
   | ARCH-EVOLVE | Pass — no persisted format or rewrite changes. |

   Core-concept locations and modification claims match the code. Atlas documents the changed ownership behavior. No new command, flag or configuration requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"db7154b5-185d-430a-8dc2-19b87da023ff","repository":"github.com/xianxu/pair","issue_id":"000865","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4ef56f190283d310","reviewed_head":"32688203878a60d27f756ad6f075c5b17f48a04e","slot":"pair:9","requested_at":"2026-10-25T06:16:19.188005Z"},"phase":"finished","pid":20796,"started_at":"2026-10-25T06:16:19.283078Z","finished_at":"2026-10-25T06:17:50.245691Z","session_id":"01a171b3-ed72-7d00-80bd-9b640646d524","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.096488875s
head: "32688203878a60d27f756ad6f075c5b17f48a04e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
