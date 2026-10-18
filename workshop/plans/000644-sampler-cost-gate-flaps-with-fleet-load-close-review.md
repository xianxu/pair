# Boundary Review — pair#644 (whole-issue close)

| field | value |
|-------|-------|
| issue | 644 — The resource sampler cost gate flaps with fleet load: first pass is not steady state and nothing names the load |
| repo | pair |
| issue file | workshop/issues/000644-sampler-cost-gate-flaps-with-fleet-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8f352e17f3a1449df8d87c55151e76f3f4c6b31..988914e084893cdf03b8d1467b18d463d1879e42 |
| command | sdlc close --issue 644 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T23:28:34-07:00 |
| verdict | SHIP |

## Review

I've checked the diff, compiled it on both build targets and run the cost test on this host. The change is test-only and I found nothing blocking, only some inaccurate comment and atlas wording.

```verdict
verdict: SHIP
confidence: high
```

The change replaces the old fixed CPU budget for the sampler (mean under 120 ms, max under 300 ms) with checks that shouldn't move with host load. Every pass is checked on how much native work it did. The two steady passes are also checked on CPU, as a ratio to a history write measured in the same run. Every report line and every failure message now names the table row count and load average. No production code changed.

I confirmed the main assumptions in the code:
- The baseline does the same work as the pass's own write: `json.Marshal` plus `Save` (`resources_persist.go:55-57`).
- `SelfCPU` measures one OS thread (`procutil/resource.go:206`). Both the baseline helper and `pass()` (`resources.go:167`) pin themselves to a thread, so the two CPU numbers are comparable.

`go vet` passes for darwin and for `GOOS=linux`, which checks the non-darwin load-average stub. `PAIR_RESOURCE_COST=1 go test ./cmd/internal/couchcmd/ -run TestResourceSamplerNativeCost -count=1` passed on this checkout at a load average of 60.31. That is an independent sixth run, under different load from the five recorded in the issue: cold pass ratio 3.78, steady ratios 1.16 and 1.15, and every work bound held. Nothing blocks shipping. The findings below are all Minor and are about wording, not behaviour.

**Strengths**
1. **The native-call accounting check is exact.** `resources_test.go` requires native calls to equal the sum of the category counters plus the two `SelfCPU` reads, so any unaccounted native work fails. That is a check that doesn't depend on load and has real teeth.
2. **The "cold environment scan" bound on pass 1** (`EnvironmentCalls >= 2`) stops the cold work bound passing vacuously if the scan is ever removed. It was thought through.
3. **The CPU-ratio gate is a real backstop for the #606 environment memo.** If that memo broke, a steady pass would look like the cold one, at a ratio of 3+. That fails `costSteadyRatioK = 2.0` as well as the environment work bound.
4. **`historyWriteCPU` is careful:** it writes to its own store, pins the thread, fails if the counter moves backwards, and fails if the baseline measures zero (which would make the ratio meaningless).
5. **The load-average reader** uses `unix.SysctlRaw` with a length check and a guard against a zero scale factor. It doesn't shell out, and it degrades to `loadavg=unknown` when the value can't be read, as a report-only value should.

**Critical:** none.

**Important:** none.

**Minor** (all one family: measured or remembered claims written into comments and the atlas)
- **`atlas/couch.md:4471`:** says the conformance "gates work rather than wall time". The old gate was absolute sampler CPU, not wall time, and the new gate still checks steady-pass CPU (as a ratio). Suggested wording: "gates work, and steady CPU relative to a same-run baseline, rather than absolute CPU".
- **`resources_test.go`, the test's doc comment:** "across loadavg 15.10 to 112.12 the pass's native call counts moved under 0.2%". The ≤0.2% figure came from the planning runs A–C, at load 30.70–34.77, not from the 15.10–112.12 runs. In the verification runs cold environment calls ranged 1545–1555 (about 0.6%), and steady native calls ranged 60–66 (up to about 10%).
- **`costSteadyRatioK` doc comment:** "a pass shaped like the cold one costs 3.04-3.76 of a write". Run 4 recorded 2.62, and my run measured 3.78, so this range is already out of date.
- **Comment on the per-pass table read:** "leave at least 38% of rows of slack" is what one host happened to show, not something the code guarantees.
- **Persistence-error `t.Fatal(e)` in the pass loop** still doesn't name the host state, while every other failure in the loop now does.

These are exactly the "numbers in prose go stale" problem the hub ruled on for the atlas, and the test comments have the same issue. The cheap fix is to cite #644 instead of restating the numbers.

**Test coverage notes**
- The work bounds and the ratio gate were exercised in both states the "Done when" names: the five recorded runs span loads 11–112, and my run at load 60.31 adds one more.
- I didn't revert-test any bound. There was no prior fix to verify, and the exact per-category accounting is owned by the fake-reader tests the plan names (`TestResourceSamplerSelectionCallsAndImmutableView`, `TestResourceSamplerAttributesADetachedSuiteToItsSlot`).

**Architecture**
- **ARCH-DRY: pass.** The baseline repeats the persist path's marshal and save on purpose, as the thing being measured against. The load-average reader is new and avoids shelling out, which the plan justifies.
- **ARCH-PURE: pass.** Test-only change, and the IO lives in small helpers.
- **ARCH-PURPOSE: pass.** All three spec bullets are delivered: the steady-state statistic, a separately stated work bound for the cold pass, and host state in every report line and failure message. The atlas, the test's gate and the issue's recorded measurements agree on the shape of the budget.

**Plan revisions:** none needed.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: prose-claims-restate-measurements
    title: |
      Atlas and test comments restate measured numbers that are wrong or already stale
    detail: |
      Every instance in this window: (1) atlas/couch.md:4471 says the conformance "gates work rather than wall time", but the old gate was absolute CPU, not wall time, and the new gate still checks steady CPU as a ratio. (2) The TestResourceSamplerNativeCost doc comment says native calls "moved under 0.2%" across loadavg 15.10-112.12; that figure came from runs A-C at 30.70-34.77, while the verification runs show cold environment calls 1545-1555 (about 0.6%) and steady native calls 60-66. (3) The costSteadyRatioK comment says a cold-shaped pass costs "3.04-3.76" of a write, but run 4 recorded 2.62 and this review measured 3.78. (4) The per-pass table-read comment says "at least 38% of rows of slack", which is what one host happened to show, not a guarantee. Cite #644 instead of restating the numbers, as the hub ruled for the atlas.
  - id: new
    severity: Minor
    kind: defect
    family: failure-messages-name-host-state
    title: |
      The persistence-error t.Fatal in the pass loop does not name the host state
    detail: |
      In TestResourceSamplerNativeCost, `t.Fatal(e)` for a persistence error is the only failure inside the pass loop that does not append the `(rows=... loadavg=...)` host string; the incomplete-measurement Fatalf, the work-bound Errorf and the steady-ratio Errorf all include it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"114ae4ca-ee58-420a-8c07-f702c615c480","repository":"github.com/xianxu/pair","issue_id":"000644","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/45504772956dc72f","reviewed_head":"988914e084893cdf03b8d1467b18d463d1879e42","slot":"pair:4","requested_at":"2026-10-18T06:27:20.249017Z"},"phase":"finished","pid":3683,"started_at":"2026-10-18T06:27:20.466081Z","finished_at":"2026-10-18T06:28:33.595349Z","session_id":"b2f6b2ee-c668-4cbc-895d-fe84a101a66c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.904447583s
head: "988914e084893cdf03b8d1467b18d463d1879e42"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T23:35:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 644 — The resource sampler cost gate flaps with fleet load: first pass is not steady state and nothing names the load |
| repo | pair |
| issue file | workshop/issues/000644-sampler-cost-gate-flaps-with-fleet-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | af0e0473d97390df336a3359df91dac5633dd0a1..9f0c6eec78e7b9417a08817df5e273d0982f3784 |
| command | sdlc close --issue 644 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T23:35:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are fixed in this window, and the window adds no new defects. On BR-1, all four wrong or stale claims are rewritten so they cite #644 instead of restating numbers. The atlas now says the gate checks work, plus steady CPU as a ratio to a same-run baseline, instead of absolute CPU. I swept the test's comments and the atlas paragraph for other restated measurements and found none. On BR-2, the persistence-error failure now uses `t.Fatalf("%s (%s)", e, host)`, the same host string as the other failures in the pass loop. Nothing else in the window is code: two comment blocks and one format string in a test file. Both platform targets still vet. The issue file only gained Log entries, the `codecomplete` status and `actual_hours`. `## Spec`, `## Revisions` and `## Done when` are unchanged, so the scope is the same.

**Strengths**
1. `resources_test.go:333-338` keeps what the reader needs (K is headroom over the worst steady ratio across the load range; a cold-shaped pass fails the gate) and moves the measurements to the issue's Log. That is the right fix for this family, not another number that goes stale.
2. `resources_test.go:452-455` now says plainly that the slack is measured, not guaranteed. That matches what the code actually enforces: the bounds are multiples of the row count.
3. The atlas sentence (`atlas/couch.md:4471-4473`) now describes the gate's real shape and agrees with the test.

**Critical:** none.

**Important:** none.

**Minor:** none new. One leftover from the BR-2 sweep, below.

**Test coverage notes**
- Neither change alters what the test decides. BR-1 is comments only, and BR-2 only changes a failure message. Per the claimed-fixes rule, no regression test is needed for either. `go vet` passes on darwin and on `GOOS=linux`.

**Architecture (interdiff only)**
- **ARCH-DRY: pass.** Nothing duplicated is added.
- **ARCH-PURE: pass.** Test comments and a message string only.
- **ARCH-PURPOSE: pass.** For BR-1, every instance in the class was fixed, not just the one cited. For BR-2, two `t.Fatal(err)` calls in the pass loop still don't append the host string:
  - The `reader.Table` error comes before `host` exists, and there's no row count to report yet.
  - The `os.Stat` error is a setup file-system failure, not a measurement verdict. "Busy host or regression?" doesn't apply to it.

  Neither is a gate verdict, so the atlas claim that every failure names the host state holds for every measurement failure.

**Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      All four instances fixed (atlas/couch.md:4471-4473, resources_test.go:333-338, 346-354, 452-455); they now cite the issue 644 Log instead of restating numbers, and a sweep of the test comments and atlas paragraph found no other restated measurement.
  - id: BR-2
    disposition: addressed
    note: |
      The persistence error now calls t.Fatalf with the host string. The two t.Fatal(err) calls left in the loop are a table-read error raised before host exists and an os.Stat setup error; neither is a measurement verdict.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"44dfc06b-546a-4b44-89ac-422bc5438b61","repository":"github.com/xianxu/pair","issue_id":"000644","recipe":"fix-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/85ee5b0da3e511b5","reviewed_head":"9f0c6eec78e7b9417a08817df5e273d0982f3784","slot":"pair:4","requested_at":"2026-10-18T06:34:43.016476Z"},"phase":"finished","pid":87153,"started_at":"2026-10-18T06:34:43.254956Z","finished_at":"2026-10-18T06:35:29.421945Z","session_id":"097011ba-878f-44d0-b975-8a2f032f7506","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.0909175s
head: "9f0c6eec78e7b9417a08817df5e273d0982f3784"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
