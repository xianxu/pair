# Boundary Review — pair#606 (whole-issue close)

| field | value |
|-------|-------|
| issue | 606 — Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot |
| repo | pair |
| issue file | workshop/issues/000606-resource-monitor-a-slot-s-detached-test-suite-sdlc-local-check-is-reported-as-orphans-attribute-it-to-the-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | b666bef0e642caecc1a5caa21169bb823b0d6807..ee39876a52fc7fc8a51be8fdc89e2e5dd1dc8f84 |
| command | sdlc close --issue 606 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T12:38:17-07:00 |
| verdict | SHIP |

## Review

Diff and issue read; now checking the sampler pass bounds and running the tests.

Targeted tests pass; the background failure was me killing the slow full run. Now checking that the sampler test fails when the wiring is removed.

```verdict
verdict: SHIP
confidence: medium
```

**Summary.** This range adds what the plan promised, and the tests back it up. There is a pure `DetachedSlotRoots` attribution step (`cmd/internal/couchcore/resource_detached.go`). There is an identity-fenced `NativeResourceReader.Environment` reader, with a pure `ParseEnvironment` that keeps only the named keys. The sampler applies the new roots before selection, through a per-process-generation cache that is pruned against the table. Sampler cost now reports an `environment_calls` field, and `atlas/couch.md` is updated. All targeted tests pass in this checkout: `Detached|ResourceSampler|ParseEnvironment|ResourceEnvironment|ResourceBounded` across couchcore, couchcmd and procutil. I replaced the `sample.Slots = couchcore.DetachedSlotRoots(...)` line in `resources.go` and `TestResourceSamplerAttributesADetachedSuiteToItsSlot` went red ("processes map[10:pair:6], want the detached suite on pair:6"), which confirms the plan's "RED without the wiring" claim. Nothing blocks SHIP. What's left: one edge case where a claimed suite still raises an orphan alert, permanent caching of transient read errors, a small duplicated name list, and no written evidence for the live part of Done-when.

1. **Strengths**
   - Attribution is a pure function over `(table, slots, threadOf)`, with the environment read injected. The core test needs no IO, and it checks that the input slots are not mutated (`resource_detached_test.go:33`).
   - The fallback for platform binaries whose environment macOS withholds (`/bin/zsh`) asks the shell's direct children, which matches the measured cause described in the Plan.
   - Infrastructure is never claimed (zellij, agents, nvim), so the #583 rule that flags unattributed infrastructure is unchanged. The test pins both a zellij server and a codex agent that carry the slot's environment (`resource_detached_test.go:46-51`).
   - Per-generation cache with pruning (`resources.go:50-80`): one read per process lifetime, bounded by the live table (ARCH-FUNERAL holds). A read cancelled by the context is not cached.
   - Only `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG` are kept from the environment, and the test includes a `SECRET=` entry to prove it (ARCH-SECURE).

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `couchcore/resource.go:263-265`: a process seen before the detached roots existed keeps its remembered root. If that agent dies or is replaced while the suite runs, `rootDead` marks the suite as an orphan even though a live detached root now claims it. The spec says "a true orphan (no live slot or run claims it)", so this is an edge case for agent relaunch during a suite. It may be a deliberate trade-off against leftovers from a relaunched thread; if so, record it.
   - `couchcmd/resources.go:73-76`: any `Environment` error is cached as "no thread" for the process's whole lifetime. A transient failure (for example a read during exec) permanently stops that generation from being attributed. Consider caching only clean reads and errors that are known to be permanent (EPERM, process gone).
   - ARCH-DRY: the `infrastructure` set (`resource_detached.go:35-38`) restates `ClassifyUnattributed`'s `names` set (`resource.go:312-315`): zellij plus `AgentInventory`, with nvim added. Extract one shared helper so the two can't drift apart.
   - ARCH-CONSTRAINTS: `darwinProcArgs` allocates an `ARG_MAX` buffer per call. On the first pass that is about 1,359 large allocations (275 ms measured). The cost is acceptable and has been measured, but a reused buffer would remove the churn.
   - Done-when also asks for a live check that "a full suite on one slot produces no orphan alert". The Log records a live probe and the claims it found, but never says outright that no orphan alert fired during a full suite.

5. **Test coverage notes:** The pure tests cover the claim and fallback paths, infrastructure exclusion, an unrelated or dead thread, and the orphan flag before and after reparenting with and without the roots. The sampler test covers wiring and one read per generation, and it fails without the wiring (checked above). The live darwin test reads a real child's environment. Not covered: a remembered root dying while the detached root is live (the first Minor), and a transient environment error followed by a retry.

6. **Architectural notes**
   - ARCH-DRY: flag (Minor; duplicated infrastructure name list).
   - ARCH-PURE: pass. The pure core is injected with the reader; `ParseEnvironment` is pure.
   - ARCH-PURPOSE: pass. The environment approach covers every detached command of a slot, not just sdlc, which goes beyond the spec's in-flight-record idea, and the Plan explains why that record doesn't exist.
   - ARCH-MOCK: pass. `resourceCalls.environmentRead` has a fake in `resource_test.go`, the sampler fake implements `Environment`, and a live darwin conformance test exists.
   - ARCH-CONSTRAINTS: pass. The cost is measured (275 ms first pass, 0.26 ms cached), reported through `environment_calls`, and capped by `ResourceMaxTableRows`.
   - ARCH-SECURE: pass. Only two keys are retained. Any process can spoof the environment, but the result is attribution only and never permission for an effect.
   - ARCH-ORDER: pass. `pass` has a single serial caller (`resources.go:96`). Cache keys are process generations (PID plus identity), so a reused PID can't read a stale entry.
   - ARCH-FUNERAL: pass. The cache is pruned against each table.

7. **Plan revision recommendations:** none required. Optionally, add a Log line recording the live "no orphan alert during a full suite" observation that Done-when asks for.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: claimed-process-orphaned-by-stale-root
    title: |
      Remembered root death orphans a suite that a live detached root now claims
    detail: |
      SelectResourceProcesses keeps previous.ResourceRoot; if the original agent root dies or is replaced while the backgrounded suite runs, rootDead flags it orphan despite DetachedSlotRoots claiming it (resource.go:263-265). Either cover/document as deliberate relaunch tradeoff or prefer a live detached candidate.
  - id: new
    severity: Minor
    kind: defect
    family: transient-error-cached-as-permanent
    title: |
      Environment read errors cached as no-thread for the generation's lifetime
    detail: |
      resources.go:73-76 caches any err as ok=false forever; a transient failure (e.g. mid-exec read) permanently prevents attribution. Cache only clean reads or known-permanent errors.
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-process-kind-list
    title: |
      Infrastructure name set duplicates ClassifyUnattributed's names set (ARCH-DRY)
    detail: |
      resource_detached.go:35-38 and resource.go:312-315 both build zellij+AgentInventory; extract a shared helper so the two exclusion lists cannot drift.
  - id: new
    severity: Minor
    kind: defect
    family: done-when-evidence-missing
    title: |
      Live Done-when (full suite raises no orphan alert) not explicitly recorded
    detail: |
      The Log records a live probe and claims but not an observed full suite run with zero orphan alerts; add that observation.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b56e0004-b567-4068-a672-5f84f649ddc6","repository":"github.com/xianxu/pair","issue_id":"000606","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/5990f7ff0c46cd0e","reviewed_head":"ee39876a52fc7fc8a51be8fdc89e2e5dd1dc8f84","slot":"pair:2","requested_at":"2026-10-16T19:26:12.131288Z"},"phase":"finished","pid":12291,"started_at":"2026-10-16T19:26:12.473072Z","finished_at":"2026-10-16T19:38:13.995731Z","session_id":"041a3a98-8a40-40bc-a228-68388e8172f1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.365990625s
head: "ee39876a52fc7fc8a51be8fdc89e2e5dd1dc8f84"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
