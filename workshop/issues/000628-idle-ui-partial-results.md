---
id: 000628
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: 'b2490cd924fb53cfd68b42b23aee7760e0433a43' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T11:26:22-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "5277190d", done: "531c08b9"}
---

# Idle UI invariant intermittently returns only 16 of 20 activity results

## Problem

During pair#619 close review on 2026-10-16 around 18:39 PDT, head 3f5f7fed4f03aa2829be24ae339cb473260bf000, the affected-package run failed TestWorkbenchPerformanceIdleUITick (27.72s): workbench_activity_invariant_test.go:77 reported pass results observed=16 and four failed rows (work-01, work-03, work-04, work-20). Couchtty package failed in 65.202s. This is a different assertion from #626's outer 30-second activity-worker wait; shared causality is not established.

Retained failing tool record: /tmp/pair619-review-activity-failure.json; source review session 01a14781-9960-7871-8ade-430a4a8c28fa. The version-controlled #619 close-review sidecar records the failing package run and unchanged targeted passing repeat. First recorded full suite on the same code (f500a1ccf3b165f070d40a2b5478605b6a02b48b) passed in 12m46s, log SHA256 prefix 6746e0ae12fd. Later changes are documentation only. No code or timeout fix was made in #619.

## Spec

Determine why the real activity fixture returns failed rows under a package run while the unchanged isolated repeat passes. Preserve the zero-read/decode invariant and distinguish transport/query deadlines from missing evidence. Load or per-row deadlines are hypotheses only, not measured causes.

## Done when

- Reproduce the partial-results assertion, identify its cause and add a deterministic regression.
- Prove the correction without weakening the idle-tick accounting or merely increasing a timeout.

## Plan

- [x] Attribute the four failed rows: confirm probe errors cannot occur (Latest swallows per-signal errors) and the rows hit the per-probe 2s deadline; tie the dominant cost to #626's measured quadratic framing (Log).
- [x] Reproduce the partial-results class at current main by restoring the pre-#626 framer through a Go overlay under the retained multi-package command.
- [x] Deterministic regression for the deadline path itself: an injected per-pass deadline shows a slow probe surfacing as failed (never silently dropped), keeping previous UI evidence and recovering on the next pass; no production deadline or accounting change.
- [x] Verify: the retained multi-package command green at current main; recorded full suite on the pushed head.
- [x] Close and land.

## Log

The reviewer ran an unchanged isolated repeat alongside the relevant key pass-through/routing tests; all passed. Issue #619 changes only help metadata and documentation. Its SHIP verdict stands; this independent fixture symptom is retained separately for investigation.

### 2026-10-17 — causality: the four rows are probe-deadline overruns, driven by the cost #626 removed

The probe (`ThreadActivityProbe`) can only fail a row in two ways (`console_activity.go:96-125`): the console is stopping (`c.lifetime.Err()`), or the per-row `context.WithTimeout(..., 2s)` expired. `threadactivity.Latest` swallows per-signal errors by design (activity.go:62-70), so `err` is nil unless `InScope` refuses the address — deterministic, not intermittent. The observed failure (observed=16, failed=4: work-01/03/04/20) is therefore four probes exceeding the 2-second deadline. The `couch: terminal: presenter unavailable` lines in the retained record are console `Stop()` teardown noise from consoles that never ran `Run`, not part of the failure.

Shared causality with the closed #626 is now established by measurement, not hypothesis: at the failing head `3f5f7fed4` (which predates `31bc02234`), the shared framer rescanned its whole pending prefix per 64KiB chunk — `scan_helpers_work_test.go` measures that as 6,888,539,021 delimiter-searched bytes for one 30,000,013-byte input, i.e. the fixture's cold pass searched ~137 GB. #626's overlay before/after (retained in its issue Log) shows cold generation 5.558s → 2.969s and delimiter-search CPU 450ms → 40ms after the bounded search landed. At ~280ms/probe average before the fix versus ~148ms after, any sustained scheduler or co-tenant load on this 22-slot host converts the largest probes into 2s deadline overruns — the recorded run's host was the loaded review session itself. Post-#626 headroom is ~13x. The failure class (rows exceeding their probe deadline land in `failed`, and with no wrapper evidence and no previous observation a cold-pass row leaves the UI without activity) is real production behavior that no test currently pins; #626's searched-byte oracle guards the cost seam (sessioninventory), not the couchtty deadline outcome. This issue adds the missing deterministic guard there; it does not duplicate #626's regressions (ARCH-DRY) and changes no deadline (the acceptance contract).

### 2026-10-17 — reproduction: the deadline mechanism, deterministic under an amplified pre-#626 framer

The pre-#626 `scan_helpers.go` (`31bc02234^`) was restored through a `go test -overlay` — pair:6's own before-measurement method, never mutating the checkout — with a 4ms per-call delimiter-search tax so one cold probe's cost crosses the production 2s bound deterministically instead of waiting for load. `go test -count=1 -v -overlay=… ./cmd/internal/couchtty -run '^TestWorkbenchPerformanceIdleUITick$'`: every cold probe then exceeded 2.2s with `available=false context=context deadline exceeded` — exactly the per-row evidence the historical run's four failed rows were made of — and with all twenty tripping, the outer 30s guard fired first (`activity worker did not finish`, #626's sibling failure). The historical run sat between the two: four overruns fired the inner partial-results assertion while the pass stayed inside 30s. Retained repro log `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair628.V2W75h/repro.log`, sha256 `58ef46564ccc68bd18ef7b4554d11d3e6d7b698174e517a5233e950363c415ad`. For calibration the unamplified old framer under the retained four-package command at host load ~18 still passed (couchtty 73.0s), matching #626's retained per-probe traces (~267-335ms cold in isolation, 6.7x headroom): the historical trip needed bursty contention on the loaded review host, which the overlay removes from the equation.

### 2026-10-17 — regression: the deadline outcome is now pinned at the Console

`TestActivityProbeDeadlineSurfacesFailureAndRecovers` (couchtty) drives one Console whose per-probe bound is shrunk through the new `probeTimeout` seam — a field defaulted in `New` to the unchanged production const (`activityProbeTimeout`); the single construction site keeps the zero value unreachable. With one stalled row against a 100ms bound (a first query costs ~40ms of native-root bootstrap, so the injected bound sits above the real floor): the cold overrun surfaces the row in `failed` while the healthy row observes — the partial-results shape asserted directly — and the UI carries no activity for it; the next healthy pass recovers it; a later overrun of an established row keeps its previous evidence rather than dropping the row (pair#247's fade must not flicker to unknown). Green single and 3x under `-race`; full couchtty package green (51.7s). The retained four-package command is green at head `40b2d08a8` under host load ~18 (couchtty 70.9s). One wrinkle the test forced into the open on the way: at a 20ms injected bound both rows failed because the ~40ms bootstrap floor exceeds it — sizing the injected bound above the real floor is part of the regression's contract, recorded here so the next reader does not shrink it below.

### 2026-10-17 — the recorded suite caught my own regression: the rename dropped the probe bound's initializer, and the entry above's greens were lucky

The recorded suite on `90fc5d943` went red in couchtty and the entry above's "zero value unreachable" claim was false at that head: `40b2d08a8` (the gofmt-alignment fix for the very seam this issue adds) realigned `New`'s composite literal and silently dropped `probeTimeout: activityProbeTimeout`. Every console then created its probe contexts with a zero deadline — an immediately-expiring `context.WithTimeout` — so whether a probe was recorded as failed depended on a scheduling race between the probe and its own zero timer. Unloaded runs (the local single-test greens, `-race` x3, the 51.7s package, the retained four-package command) won that race and read green; the loaded suite lost it: the first activity test's two probes both came back `failed`, the wait for observed activity ran the full package budget (1188.93s of `testwait` polling), and every later failure in the log was the exhausted `-timeout 20m` deadline, not additional causes. Diagnosis was measurement, not hypothesis: couchcore run concurrently with couchtty reproduced the wedge in isolation (229.42s, identical cascade); a SIGQUIT mid-hang showed the Run loop idle in select with no workers and no tokens; a temporary diagnostic test under load dumped the state that named it — `probeTimeout: 0s`, `probe ctx errors: 2`, two finished generations, rows landed, activity map empty. Fix `917383ee8`: restore the one initializer line, plus `TestNewConsoleDefaultsTheProbeBound`, which pins the default to the package const and went red (`probeTimeout = 0s, want the package const 2s`) on exactly this shape before the fix — the deadline test itself could not catch it because it sets the field explicitly. Loaded verification after the fix: the same recipe that wedged twice (couchtty full package alongside couchcore plus busy loops) is green at 35.9s. Method notes for the next flake: `testwait` bounds waits by the test deadline (margin 10s), so under `go test -timeout` a lost event consumes the package budget and cascades as instant failures with shrinking budgets (`giveUpAt`'s remaining/2) — read the FIRST failure and its duration, the rest is aftermath; and a composite-literal realignment is exactly where initializers get dropped, which is why the constructor default now has its own pinning test.

### 2026-10-17 — close review round: mechanism corrected, regression bound widened

The close boundary review returned FIX-THEN-SHIP with two Important findings; both were verified before acting. BR-1 (mechanism): a non-positive `context.WithTimeout` bound returns an already-cancelled context — Go cancels synchronously when the deadline has passed (context.go cancels when `dur <= 0`), so the dropped initializer failed every probe deterministically on any machine. First-hand revert check at `946c2b15c` (initializer dropped via overlay, unloaded): `TestNewConsoleDefaultsTheProbeBound` red (`0s, want the package const 2s`) and `TestWorkbenchPerformanceIdleUITick` red in 0.19s with all 20 rows failed — no race; the retained repro durations are all `testwait` give-ups (65.00s = 75s timeout minus the 10s margin; 229.42s = 4m minus margin; the suite's 1188.93s = 20m minus margin), not won-or-lost timers. BR-2 (headroom): the regression's 100ms injected bound sat only ~2.5x above the ~40ms first-query floor — the same loaded-host condition that produced #628 could push the healthy row over it and flake the test. The injected bound is now 1s (~25x) and the stall is ctx-bound (10s backstop), so an overrunning row returns exactly at its deadline; the pinned outcomes (cold overrun fails visibly with no activity, recovery, established overrun keeps evidence) are unchanged. Green single (2.05s) and 2x under `-race` (5.5s).

## Revisions

- 2026-10-17 — the close boundary review (BR-1) corrected the mechanism claim in the two entries above, verified against the Go source and a first-hand revert check: a zero `WithTimeout` bound does not race the probe; Go returns the context already cancelled (synchronous cancel when `dur <= 0`), so every probe fails deterministically and "the greens were lucky / won the race" is false. The greens attributed to `40b2d08a8` cannot have run against the dropped initializer — a green couchtty run is impossible in that state (the invariant test fails with `observed=0` in ~0.2s; the activity tests that await an observed pass instead burn their full `testwait` budget). The prior entries stand as history; this correction supersedes their race narrative, and the matching lesson in `workshop/lessons.md` is rewritten (doubt the record before inventing a reconciling mechanism).
