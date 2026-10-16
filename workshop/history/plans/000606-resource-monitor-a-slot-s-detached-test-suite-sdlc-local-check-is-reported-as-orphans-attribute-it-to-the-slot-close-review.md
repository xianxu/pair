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

---

## Re-review — 2026-10-16T12:45:56-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 606 — Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot |
| repo | pair |
| issue file | workshop/issues/000606-resource-monitor-a-slot-s-detached-test-suite-sdlc-local-check-is-reported-as-orphans-attribute-it-to-the-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0f0abd3e95009c619bcf8d42001b20b21e1f9cb5..63babdc3e4dec229d2035224cc77000da456b7ea |
| command | sdlc close --issue 606 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T12:45:56-07:00 |
| verdict | SHIP |

## Review

Checking the fixes against the four prior findings; next I run the tests and revert each fix to confirm its regression test fails without it.

```verdict
verdict: SHIP
confidence: high
```

This fix window covers one code commit, `63babdc3e`. It changes four files and adds no scope: `## Spec`, `## Revisions` and `## Done when` are unchanged, and the issue file only gains the close log line and status. The commit fixes BR-1 and BR-2, and each fix has a regression test that fails without it. I checked this in the disposable checkout. I reverted the commit's changes to `resource.go` and `resources.go` and ran the tests again. `TestDetachedSuiteOutlivesItsAgentsRelaunch` failed: the binary was still rooted at the dead agent (`Root:{PID:5}`), with `Orphan:true`. `TestResourceSamplerAttributesADetachedSuiteToItsSlot` also failed ("processes map[10:pair:6], want the detached suite on pair:6"). I then restored the checkout, and both test packages pass on head. BR-2 is only partly fixed: the error cache still uses a deny-list rather than the allow-list the finding asked for. BR-3 and BR-4 were not touched. All the prior findings were Minor and none blocked the gate, and the window introduces nothing Critical or Important. So it ships, with those Minors still open.

1. **Strengths**
   - `resource.go:250-261`: when the remembered root has died, ownership passes to a live root, but only one with the same slot and the same thread. A suite can't drift to a different slot, and a process with no live claimant still goes orphan.
   - `resource_detached_test.go:83-107`: the new test replays the real sequence: select, the agent dies, the suite reparents, `DetachedSlotRoots`, select again. It goes red without the fix.
   - `resources_test.go:107`: the expected read order `[20,21,30,21]` checks two things at once: a failed read is asked again on the next pass, and a settled generation is still read only once.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `resource.go:253`: the handover check restates the `rootDead` check a few lines below (`resource.go:265`), but the two disagree. The handover treats a live root with an empty identity as dead and skips the `duplicate[]` guard; `rootDead` does neither. Compute `rootDead` once for the remembered root and use it for both (ARCH-DRY). The effect is harmless: only the displayed root changes.
   - BR-2 still open: other errors from `environmentRead` are still cached as permanent (`resources.go:76-83`).
   - BR-3 and BR-4 still open (see dispositions).

5. **Test coverage:** both new behaviours have regression tests, and I confirmed both fail with their fix reverted. No test covers a non-identity transient error from `environmentRead`.

6. **Architecture**
   - **ARCH-DRY: flag (Minor).** The new handover check duplicates the root-death check (first Minor above). BR-3's duplicated name list is still there.
   - **ARCH-PURE: pass.** The handover lives in the pure `SelectResourceProcesses`. The retry decision sits in the thin sampler shell, and its tests run on a fake reader.
   - **ARCH-PURPOSE: pass for BR-1.** The fix covers the whole class: any dead or reused remembered root with a live claimant from the same thread. It isn't limited to the case the finding named. BR-2 fixes one instance (identity errors), not the class (see its disposition).

7. **Plan revisions:** none required. Optionally, add a Log line once the live "a full suite produces no orphan alert" check has been seen (BR-4).

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      resource.go:250-261 hands a dead or reused remembered root over to a live same-slot, same-thread candidate. TestDetachedSuiteOutlivesItsAgentsRelaunch fails with the fix reverted (Root PID 5, Orphan true).
  - id: BR-2
    disposition: not-addressed
    note: |
      Only IdentityUnavailableError and ctx errors skip the cache now. Other environmentRead errors (resource.go:170-172, e.g. EINVAL from a mid-exec PROCARGS2 read) are still cached as permanent. The finding asked to cache only clean reads or known-permanent errors (EPERM/ESRCH). The partial fix has a test that goes red without it.
  - id: BR-3
    disposition: not-addressed
    note: |
      The interdiff does not touch resource_detached.go:35-38 or resource.go:324-327. The two zellij+AgentInventory sets are still duplicated.
  - id: BR-4
    disposition: not-addressed
    note: |
      The Log still says live no-orphan-alert acceptance "needs the running Couch on this build (TL after landing)". No full-suite observation is recorded.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-root-death-predicate
    title: |
      Handover root-death check restates and diverges from rootDead (ARCH-DRY)
    detail: |
      resource.go:253 treats a live root with an empty identity as dead and ignores duplicate[], while rootDead at resource.go:265 requires a non-empty identity and checks duplicate[]. So a live root whose identity read failed is still handed over. Compute the remembered root's death once and use it for both checks.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6f4c429b-e337-4ddf-89c4-c691ae4114ca","repository":"github.com/xianxu/pair","issue_id":"000606","recipe":"fix-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ac17a4a6fd8dac02","reviewed_head":"63babdc3e4dec229d2035224cc77000da456b7ea","slot":"pair:2","requested_at":"2026-10-16T19:43:52.361711Z"},"phase":"finished","pid":20696,"started_at":"2026-10-16T19:43:52.553601Z","finished_at":"2026-10-16T19:45:54.34329Z","session_id":"36dba369-6014-447a-8c60-0038cee1f06b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.404322917s
head: "63babdc3e4dec229d2035224cc77000da456b7ea"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T13:22:04-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 606 — Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot |
| repo | pair |
| issue file | workshop/issues/000606-resource-monitor-a-slot-s-detached-test-suite-sdlc-local-check-is-reported-as-orphans-attribute-it-to-the-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 83441b3f7f3e64f04424708149268fd804c67f5d..3532f6d70fdb4d4af46f6fecdbbb7fe591e66664 |
| command | sdlc close --issue 606 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T13:22:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This window changes one code file. `cmd/internal/artifactpath/manifest.go` adds two entries to `NonArtifactSources`: `cmd/internal/couchcore/resource_detached.go` and `cmd/internal/procutil/environment.go`. Both files exist at head. With those entries in, `go test ./cmd/internal/artifactpath` passes on head (`ok … 10.849s`); without them, the recorded suite would fail `TestProductionArtifactReferencesAreExactlyClassified`. Everything else in the window is tracker material: the round-2 gate ledger and review record, one entry in `workshop/lessons.md`, and in the issue file a close-log line, `card_mirror` and `actual_hours`. `## Spec`, `## Revisions` and `## Done when` are unchanged, so scope did not change. There are no conflict resolutions to judge. None of BR-2, BR-3, BR-4 or BR-5 is touched, so all four stay `not-addressed`. They are all Minor and none blocks, and the window adds no Critical or Important defect, so it ships with them still open.

1. **Strengths**
   - `manifest.go:685` and `:1124`: each new production source is listed next to its siblings from the same package. That fixes the merge-time failure at the cause, rather than weakening the test.
   - The new `workshop/lessons.md` entry records the rule behind this fix (a new `.go` file needs a manifest entry; run `go test ./cmd/internal/artifactpath`), not just this one case.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - Nit: `resource_detached.go` is listed after `resource_roots.go`, which breaks the alphabetical order of the surrounding `resource_*` entries. It has no effect on behaviour.

5. **Test coverage:** the existing manifest classification test covers this change, and it passes on head. No new behaviour needs a new test.

6. **Architecture (interdiff only)**
   - **ARCH-DRY: pass.** The change only adds entries to the existing single inventory; nothing is duplicated.
   - **ARCH-PURE: pass.** The change is declarative data with no IO.
   - **ARCH-PURPOSE: pass.** Both new production files from this issue (`resource_detached.go`, `environment.go`) are classified, which is the full set the window needs. The open Minors are untouched and stay open; they are not deferred as follow-ups.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-2
    disposition: not-addressed
    note: |
      The interdiff does not touch resources.go or resource.go. environmentRead errors other than identity-unavailable and context errors are still cached as permanent.
  - id: BR-3
    disposition: not-addressed
    note: |
      The interdiff does not touch resource_detached.go:35-38 or resource.go. The zellij+AgentInventory name sets are still duplicated.
  - id: BR-4
    disposition: not-addressed
    note: |
      The new issue Log line records only unit tests. There is still no observation of a live full suite run with zero orphan alerts.
  - id: BR-5
    disposition: not-addressed
    note: |
      resource.go is unchanged in this window. The handover root-death check still restates rootDead and diverges from it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8dd0fc99-3a71-42d0-b646-55d296dd890f","repository":"github.com/xianxu/pair","issue_id":"000606","recipe":"fix-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/461064ffcb106f8e","reviewed_head":"3532f6d70fdb4d4af46f6fecdbbb7fe591e66664","slot":"pair:2","requested_at":"2026-10-16T20:21:21.771899Z"},"phase":"finished","pid":71028,"started_at":"2026-10-16T20:21:21.921543Z","finished_at":"2026-10-16T20:22:03.4849Z","session_id":"604b4848-8eca-429f-9a55-94b78d0fe224","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.759470959s
head: "3532f6d70fdb4d4af46f6fecdbbb7fe591e66664"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
