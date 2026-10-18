---
id: 000644
status: codecomplete
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
severity: medium
card_mirror: '9c79c4f190502d4cb0498271561cac296b158214' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T21:09:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "22a29c94", done: "589faf29"}
actual_hours: 1.41
---

# The resource sampler cost gate flaps with fleet load: first pass is not steady state and nothing names the load

## Problem

`TestResourceSamplerNativeCost` (`cmd/internal/couchcmd/resources_test.go`, env-gated on
`PAIR_RESOURCE_COST=1`) is a coin flip on fleet load. Its gate is #587's approved qualification: mean
sampler CPU under 120 ms and max under 300 ms across three passes, i.e. "below 0.2% of one core at the
default minute cadence". Two runs of identical code on the same host, hours apart, while closing pair#583:

    load ~14, 1270 rows: pass1 142.409ms (1585 calls)  pass2 63.952ms (64)  pass3 65.244ms (64)
                         mean 90.535ms  max 142.409ms  0.150892%  -> PASS
    load ~56, 1323 rows: pass1 245.274ms (1581 calls)  pass2 80.329ms (66)  pass3 84.535ms (64)
                         mean 136.712ms max 245.274ms 0.227854%  -> FAIL "resource CPU budget exceeded"

The load ~56 run had three other slots' suites going, including ariadne's sharded full suite. Nothing in
the failure output says so: it reads exactly like a sampler regression.

Two causes, neither of them the thing the budget is meant to catch:

1. The first pass is not steady state. It makes ~1521 more native calls than passes 2 and 3 — one
   environment read per process generation, which is #606's detached-slot attribution, and which
   `atlas/couch.md` already documents as "about 0.3 s for 1,200 processes on Couch's first pass, then only
   new processes". #587 landed 2026-10-15 and qualified at 71.331 ms mean / 74.264 ms max over ~1007 rows
   and 58 native attempts; #606 landed the day after and roughly doubled the first pass. The constant was
   never re-derived. #587's own plan records the TL ruling that "the binding CPU budget applies to steady
   sampling, not a rare maximum-cardinality admission burst" — and a three-pass mean now folds in a cold
   pass that is neither.
2. The whole measurement is host-shaped. Rows, syscall latency and the process count all move with what
   else the fleet is doing, and the assertion has a fixed constant on the other side of it.

Steady state was inside the budget in both runs (64-85 ms). Only pass 1 moved, from 142 ms at rest to
245-265 ms under load.

Because the test is env-gated it runs in neither CI nor `make test`, so nobody sees either failure mode:
the repo's only conformance for the sampler's cost budget is unreliable and invisible at the same time.

For whoever takes this, two measurements from the same session that bound the new work rather than the old:

- pair#583's orphan nvim probe is not a contributor. At its widest gates on that host it cost 2 argv reads
  and 2 cwd probes, about 69 us per pass (per-call native cost: cwd 8.609 us, argv 26.107 us). Removing
  #583's widening from the test under the same load ~56 gave mean 138.438 ms — worse, so it is noise.
- Relaxing `UnattributedShape`'s `p.PPID == 1` to `>= 1` took the same pass from 2 to 122 argv+cwd probe
  pairs on a host running ~30 pair embeds. The PID-1 gate, not the age threshold, is what bounds the probe
  count; keep that visible wherever the budget is re-derived.

## Spec

- The conformance measures what the budget governs, and says what it measured. Either the statistic
  becomes steady-state with the one-time cold pass bounded separately, or the constant is re-derived
  against a first pass that includes #606's environment scan. The choice and the measurement behind it get
  recorded; the number is not picked to make the test pass.
- The cold first pass gets its own stated bound, so #606's environment scan sits under a number and not
  only under atlas prose.
- The run reports the host state it was taken in (table rows at minimum, load average if cheap), and the
  failure message names it, so a red run distinguishes "the host was busy" from "the sampler regressed"
  without a second investigation.

## Done when

- `PAIR_RESOURCE_COST=1 go test ./cmd/internal/couchcmd/ -run TestResourceSamplerNativeCost` gives the same
  verdict on an idle host and on one running other slots' suites, and the numbers behind every constant are
  recorded in the issue.
- The cold first pass has its own bound, and the atlas cost paragraph, the test's gate and the recorded
  measurement agree.

## Plan

Shape: replace the single absolute-CPU verdict with two load-invariant verdicts — a **work** verdict on
every pass, and a **same-run-ratio** CPU verdict on the steady passes — and report the host state in every
line and every failure message. Test-only; no production code changes.

Choice taken (spec bullet 1): the statistic becomes **steady-state**, with the one-time cold pass bounded
separately. Measurement behind it, 5 native runs spanning loadavg 30.70 → 34.77 (one under 12 busy loops;
table and derivations in `## Log`): native call counts are load-stable (cold 1607-1610, steady 60-64, a
≤0.2% spread) while CPU is not (cold 161.8→251.8 ms, +55%; steady 65.4→86.3 ms, +32%). The steady pass's
CPU *measured against a same-run history-write baseline* is load-stable (ratio 1.11-1.38); the cold pass's
is not (3.04→3.76), because the cold pass is env-syscall-dominated and syscalls inflate more under load
than the userspace marshal both are divided by. #587's TL ruling already put the admission burst outside
the binding budget, so the cold pass gets a work bound and a report, not a CPU assertion.

- [x] **Same-run history-write baseline (new fixture step).** Before the passes: `runtime.LockOSThread`,
      then the mean thread CPU of 3 reps of `json.Marshal(seed)` + `Save` into a *second* temp
      `couchcore.ResourceHistoryStore` (so the baseline never touches the fixture's history file),
      bracketed by `reader.SelfCPU`. This is the pass's own dominant userspace cost measured on this host
      under this load: one steady pass writes the near-cap history exactly once (`StorageWrites=1`,
      `StorageBytes≈4.19 MB`). Fatal if the baseline measures 0, which would make the ratio gate vacuous.
      Landed as `historyWriteCPU` next to the test.
- [x] **Host state per pass (spec bullet 3).** Read the table once immediately before each pass; its row
      count is the work bounds' basis, and `rows=`/`loadavg=` go into every report line and every failure
      message. Load average comes from a new darwin-tagged test helper
      (`resources_cost_host_darwin_test.go`: `unix.SysctlRaw("vm.loadavg")` — the value is a binary struct,
      three little-endian `fixpt_t` at offsets 0/4/8 over an `int64` fscale at offset 16) with a `!darwin`
      stub (`resources_cost_host_other_test.go`) returning `loadavg=unknown`. No exec: the repo's other two
      loadavg readers (`doctor/perf.sh:_loadavg`, `cmd/probes/reattachcost`) shell out to `sysctl`, and a
      fork is the wrong shape inside a cost conformance. Report-only — no gate reads it.
- [x] **Work verdict, every pass.** Each bound is derived from the code's own structure, not from a host
      measurement, so none of them moves with load:
      - `NativeCalls == TableCalls + UsageCalls + CommandCalls + CwdCalls + EnvironmentCalls + 2` — `pass()`
        makes exactly two `SelfCPU` reads and every other native call lands in a category counter, so
        unaccounted native work (a new probe, a hidden retry) fails here. Held in 5/5 runs
        (1610 == 2+56+0+0+1550+2).
      - `TableCalls <= 6` — 3 attempts x (size + read). Measured 2 in every run.
      - `UsageCalls <= 2 x rows`, `EnvironmentCalls <= 2 x rows`, `CommandCalls <= 2 x rows`,
        `CwdCalls <= CommandCalls` — a native read costs at most 2 calls (`procutil` charges the read plus
        its generation fence; a *failed* read costs 1, so these are bounds, not equalities), and no category
        reads a row more than once per pass: usage observes each selected row and each candidate once
        (`UnattributedShape` excludes selected rows), the #606 memo allows one env read per generation,
        argv is read once per nvim candidate, and cwd only after a successful argv read. Measured slack on
        this host: usage 56 against a 2502 bound, env 1550 against 2502 — the env bound is 1.6x the
        observed cold scan, so a second read per generation fails it. `rows` is the test's own read a few
        milliseconds before the pass; the 476 rows of slack is what absorbs churn in that window.
      - `StorageWrites == 1` and `ResourceHistoryLimit-65536 <= StorageBytes <= ResourceHistoryLimit` —
        pins the near-cap write the ratio's denominator assumes. Measured 4,190,342-4,193,832.
      - Pass 1 only: `EnvironmentCalls >= 2` — the #606 scan ran; without it the cold work bound would pass
        vacuously if the scan were ever removed.
- [x] **Cold verdict (pass 1): work bound plus a report, no CPU assertion (spec bullet 2).** Report `cpu`
      and its ratio to the same-run history write alongside `rows` and `loadavg`. The cold pass's stated
      bound is the work bound above — **one env read per process generation at 2 native calls each, over at
      most the table's rows** — which is the number #606's scan now sits under, replacing atlas prose. Its
      CPU is the load-sensitive quantity by construction (measured +55% across the load step at unchanged
      call counts), so asserting on it would rebuild the flake; the recorded observations go to this issue
      and the ratio goes to the report line (hub ruling).
- [x] **Steady verdict (passes 2-3): `CPUTimeNS <= 2.0 x serializeBaseline`, per pass.** K=2.0 derivation:
      observed steady ratios 1.23/1.25 (loadavg 30.70), 1.15/1.38 (30.92, 12 busy loops), 1.11/1.16
      (34.77) — worst 1.38, so K is 1.45x the worst observation across the measured load spectrum, rounded
      to a statable budget: *a steady pass may not cost more than twice the history write it performs*.
      Recorded here, not tuned per run. Teeth: it fails on a doubled history write, on a userspace
      (reducer/projection/clone) regression of similar size, and on a #606 memo regression — a cold-shaped
      pass every time measures 3.04-3.76, so the ratio is the memo's native backstop (its exact
      per-generation owner stays the fake-reader `TestResourceSamplerAttributesADetachedSuiteToItsSlot`).
      The steady mean and `sixty_second_cpu_pct` are still reported but no longer asserted: the absolute
      percentage is an observation about this host (measured 0.11-0.14% steady; the old three-pass mean
      that folded in the cold pass measured 0.163% idle and 0.215-0.224% loaded, which is the flap).
      Hub approved K=2.0 over the tighter 1.75.
- [x] **Failure messages name the host state** — `rows`, `loadavg`, and the counters behind the failed
      bound — so a red run reads as either "the host was busy" or "the sampler regressed" without a second
      investigation.
- [x] **`atlas/couch.md`.** Rewrite the cost paragraph (today: "averaged 72.438 ms ... below 0.2%, with a
      300 ms maximum pass CPU budget") to state the new budget shape — per-pass work bounds plus steady CPU
      at most 2x the same-run history-write baseline — and the reason the absolute budget was dropped, and
      amend the #606 paragraph's "about 0.3 s for 1,200 processes" to the work bound. Per the hub's
      ruling the atlas carries the bound and the gate shape only, not measured milliseconds: numbers in
      prose go stale, so the measurements stay in this issue and the atlas cites it. Done: the stale
      averages (1.643 ms, 72.438/79.311 ms, 0.120730%, 58 native attempts) and the 0.2%/300 ms budget are
      gone from `atlas/couch.md`, replaced by the per-category call costs, the work bounds, the same-run
      ratio, the host-state reporting, and the unchanged "well under 1% of one core" product requirement.
- [x] Delete the scratch diagnostic `cmd/internal/couchcmd/zz_cost_diag_test.go` (uncommitted).
- [x] Verify (done-when): `PAIR_RESOURCE_COST=1 go test ./cmd/internal/couchcmd/ -run
      TestResourceSamplerNativeCost` (a) as-is, (b) under 12 busy loops, (c) while another slot's suite
      runs — same verdict in all three, logs retained; targeted `go test ./cmd/internal/couchcmd/
      ./cmd/internal/couchcore/`; then the full suite as `sdlc local-check run --name suite` on the pushed
      head. Five conformance runs, not three — the fifth is on the landing head. The suite took two
      attempts: the first red on seven load-shaped tests in packages this patch cannot reach, filed as
      #659; the second green on the same code. Both are recorded below.

**Not in scope.** No production-code change: the conformance asserts only what a native run can show (real
reader call multiplication at host scale, and CPU proportionality), while exact per-category accounting
stays owned by `TestResourceSamplerSelectionCallsAndImmutableView` and the env memo by
`TestResourceSamplerAttributesADetachedSuiteToItsSlot` (ARCH-DRY, one owner per invariant). The conformance
also stays env-gated — the gates are now load-invariant, so running it in CI or `make test` has become
possible, but what CI should cost is a separate decision from this fix.

## Log

### 2026-10-17
- 2026-10-17: closed — Both advisory findings from the close review are fixed and re-verified. Finding 1, prose that restated measurements: the atlas no longer calls the replaced gate wall time (it measured absolute sampler-thread CPU) and now says the conformance gates work plus steady CPU as a ratio to a same-run baseline; the test doc comment no longer attributes the under-0.2-percent call-count figure to loadavg 15.10-112.12, which came from the planning runs at 30.70-34.77; costSteadyRatioK no longer quotes a cold-ratio range of 3.04-3.76 that run 4 at 2.62 and the reviewer run at 3.78 had already left behind on both sides; and the per-pass basis comment no longer presents one hosts 38-percent slack as a guarantee. Each now states the relationship and cites this issue Log for the numbers, which is the rule the hub set for the atlas. Finding 2, spec bullet 3: the persistence-error failure inside the pass loop now names the host state like the incomplete-measurement, work-bound and steady-ratio failures do, so the atlas claim that every failure names it is true rather than nearly true. Verification on the edited test: PAIR_RESOURCE_COST=1 go test ./cmd/internal/couchcmd/ -run TestResourceSamplerNativeCost PASS at loadavg 40.69, rows 1290-1298, history-write baseline 66.195 ms, cold pass 234.833 ms at ratio 3.55, steady passes 84.074 ms at 1.27 and 82.267 ms at 1.24, every work bound held, one near-cap write of 4,190,350-4,193,832 bytes. That is the sixth passing run of the same gate across loadavg 11.16 to 112.12, counting the reviewers independent run at 60.31 during the close. gofmt clean on the three touched Go files; go vet clean on darwin and on GOOS=linux, which covers the not-darwin load-average stub. No production code changed and no gate moved: the delta is comments, one atlas sentence, and one t.Fatal becoming a t.Fatalf carrying the host string. (fixed-then-ship); review verdict: SHIP
- 2026-10-17: closed — Five runs of unchanged code at loadavg 11.16, 15.10, 18.78, 23.23 and 112.12 give one verdict, PASS: fifteen passes, twelve of them steady, and every work bound held in all of them (native accounting exact against table+usage+command+cwd+environment+2, table 2 calls, usage 56, argv/cwd 0, env 1545-1555 cold and 0-4 steady, one near-cap write of 4,190,350-4,193,832 bytes). Worst steady ratio anywhere is 1.38 against K = 2.0. The old fixed budget of 120 ms mean and 300 ms max splits the same five runs: it would have failed the three loaded ones (means 125.687, 149.202, 160.993 ms) and passed the two quiet ones (105.854 and 97.071 ms), which is the reported flap, reproduced and then removed. The cold pass is bounded by work with its CPU reported only, because its CPU is not reproducible even at a matched load: +59 percent CPU for +1.3 percent native calls at loadavg about 15, and a cold ratio of 2.62 at loadavg 11.16 against 3.73 at 112.12 for the same 1550 environment calls. Every pass line and every failure message names the host state it measured (rows plus loadavg from vm.loadavg on darwin, with a not-darwin stub; GOOS=linux go vet clean). The atlas cost paragraph now carries the work bound and the steady-ratio rule with no milliseconds, per the hub ruling. Full suite: attempt 1 on 6a7940eec red on seven load-shaped tests in five packages this five-file test-only patch cannot reach, filed as #659 with the evidence table; attempt 2 on 615ab5d90 green, exit 0 in 11m54s, zero FAIL lines, those five packages 2.1x to 7.3x faster on unchanged code. Both logs retained at /tmp/pair644-suite-attempt1.log and /tmp/pair644-suite-attempt2.log. Targeted go test on couchcmd and couchcore green; gofmt and go vet clean on the three touched Go files.; review verdict: SHIP

Claimed, `sdlc start-plan` (first run refused: it caught up the woven ariadne dependency and rebuilt sdlc,
"nothing was branched or started. Re-run"; the second run branched). Measured with a scratch diagnostic —
the real fixture plus `EnvironmentCalls`, loadavg, and two same-run baselines (a native table read, and 3
reps of marshal+save of the seeded near-cap history). Logs retained: `/tmp/pair644-idle.log` (run A),
`/tmp/pair644-loaded.log` (runs B and C, 12 busy loops).

| run | loadavg (1m) | rows | pass | native | table | usage | cmd | cwd | env | cpu ms | wall ms | cpu/serialize | storage bytes |
|-----|--------------|------|------|--------|-------|-------|-----|-----|-----|--------|---------|---------------|---------------|
| A | 30.70 | 1251 | 1 | 1610 | 2 | 56 | 0 | 0 | 1550 | 161.805 | 245.776 | 3.04 | 4193832 |
| A | 30.70 | 1251 | 2 | 60 | 2 | 56 | 0 | 0 | 0 | 65.385 | 77.792 | 1.23 | 4190350 |
| A | 30.70 | 1251 | 3 | 60 | 2 | 56 | 0 | 0 | 0 | 66.415 | 79.114 | 1.25 | 4191312 |
| B | 30.92 | 1276 | 1 | 1609 | 2 | 56 | 0 | 0 | 1549 | 251.791 | 522.001 | 3.76 | 4193832 |
| B | 30.92 | 1276 | 2 | 60 | 2 | 56 | 0 | 0 | 0 | 77.250 | 125.786 | 1.15 | 4190351 |
| B | 30.92 | 1276 | 3 | 60 | 2 | 56 | 0 | 0 | 0 | 74.109 | 123.590 | 1.11 | 4191321 |
| C | 34.77 | 1272 | 1 | 1607 | 2 | 56 | 0 | 0 | 1547 | 228.830 | 430.406 | 3.67 | 4193832 |
| C | 34.77 | 1272 | 2 | 64 | 2 | 56 | 0 | 0 | 4 | 86.322 | 244.557 | 1.38 | 4190342 |
| C | 34.77 | 1272 | 3 | 60 | 2 | 56 | 0 | 0 | 0 | 72.157 | 107.042 | 1.16 | 4191312 |

Baselines and the old gate's verdict on the same runs:

| run | table-read baseline cpu | serialize baseline cpu | old mean (3 passes) | old max | old verdict (mean<120 ms, max<300 ms) |
|-----|------------------------|------------------------|---------------------|---------|---------------------------------------|
| A | 1.102 ms | 53.170 ms | 97.868 ms (0.163114% of a core) | 161.805 ms | PASS |
| B | 1.192 ms | 67.032 ms | 134.383 ms (0.223972%) | 251.791 ms | FAIL "resource CPU budget exceeded" |
| C | 0.812 ms | 62.372 ms | 129.103 ms (0.215172%) | 228.830 ms | FAIL "resource CPU budget exceeded" |

Runs B and C reproduce the reported flap on unmodified code, and neither failure message named the load.
Steady state was inside the old budget in all three runs (65.4-86.3 ms); only pass 1 moved (161.8 ms at
loadavg 30.70 → 251.8 ms under 12 busy loops).

What the measurement settled, and two derivations that were wrong first:

- **Env calls are 2 per read, not 1.** `NativeResourceReader.Environment` (`cmd/internal/procutil/resource.go:157-177`)
  charges the `environmentRead` plus `resourceFence`'s `identityRead` (`resource.go:89-110`), so 1550 calls
  is **775 reads over 1251 rows**. The "env ≤ rows" formula this issue's first derivation reached for is
  wrong by 2x; the sound bound is `2 x rows`. Same shape for `Usage` (2 on success, 1 on failure), which is
  why the work verdict is a bound and not an equality: `ClassifyUnattributed`
  (`couchcore/resource.go:396-423`) also requires `RSSKnown` and `RSSBytes > floor`, so a candidate whose
  usage read failed spends a call and never reaches the report the test can count.
- **775 of 1251 rows are visited, not all of them.** `DetachedSlotRoots` (`couchcore/resource_detached.go:59-68`)
  reads the environment only for PPID-1 rows that pass its `candidate` filter and, for one that fails to
  claim, its candidate children; `zellij`, `nvim` and every agent name are never visited. That is why the
  cold scan is ~62% of rows here rather than 100%, and why `2 x rows` is a bound with 1.6x slack rather
  than a tight equality.
- **The table read is the wrong denominator.** Its CPU is 0.812-1.192 ms, so `mean/table` ran 88.8-159.1
  across runs — the ratio just restates the pass's whole cost and moves with load. The history-write
  baseline is the right one because it is the steady pass's own dominant work: the ratio reads as "how much
  more than the write does a pass cost".
- **CPU-per-row is worse than the ratio.** The steady pass's marginal cost over its baseline,
  `(pass cpu - baseline)/rows`, spread 7.7-18.8 us (2.4x) across the same runs where the serialize ratio
  spread 1.11-1.38 (1.24x): subtracting two noisy numbers amplifies, dividing two co-moving ones cancels.
- **Probe fan-out stays visible but unasserted.** `cmd=0 cwd=0` in all 9 passes — this host had no PPID-1
  nvim rows at those moments — so the filer's measurements remain the record for the probe path (2 argv +
  2 cwd at the widest gates, ~69 us per pass; relaxing `UnattributedShape`'s `p.PPID == 1` to `>= 1` took it
  from 2 to 122 probe pairs). The work bounds catch per-row *multiplication*, not fan-out growth: fan-out
  is bounded in code by the PID-1 gate and `ResourceMaxProcesses`, and it stays visible in the report's
  `cmd`/`cwd`/`unattributed`/`orphan_nvim` columns, which is what this issue asked for.
- **darwin load average:** `syscall.Sysctl("vm.loadavg")` returns the raw binary struct, which prints as
  garbage; `syscall.SysctlRaw` does not exist on darwin. `unix.SysctlRaw` with the fixpt/fscale layout
  above works, and `golang.org/x/sys` is already a direct dependency (`go.mod:12`).

Implemented after the hub approved the plan (delivery 7118f0d8) with three rulings: K = 2.0 stays as
derived; the cold-pass ratio goes in the test's report line but the atlas keeps the work bound only,
because numbers in prose go stale; no CI follow-up. Landed as `historyWriteCPU` (the same-run baseline),
`hostLoadAverage` in `resources_cost_host_darwin_test.go` with a `!darwin` stub, the per-pass work bounds
and the steady ratio gate inside `TestResourceSamplerNativeCost`, and the two `atlas/couch.md` paragraphs.
Scratch diagnostic deleted.

Verdict on the new gate — four runs of the same code (run 4 after a readability pass over the test that
changed no bound), logs at `/tmp/pair644-new-asloaded.log`, `/tmp/pair644-new-burners.log` and
`/tmp/pair644-new-fleetload.log`. "Old mean" is what the fixed 120 ms/300 ms budget would have said about
the same three passes:

| run | condition | loadavg (1m) | rows | baseline ms | cold cpu ms (ratio) | steady cpu ms (ratios) | old mean | new verdict |
|-----|-----------|--------------|------|-------------|---------------------|------------------------|----------|-------------|
| 1 | as-is | 15.10 | 1266-1267 | 69.817 | 226.846 (3.25) | 77.656 (1.11) / 72.560 (1.04) | 125.687 ms → would fail | PASS |
| 2 | 12 busy loops | 23.23 | 1263-1266 | 77.197 | 261.870 (3.39) | 90.288 (1.17) / 95.449 (1.24) | 149.202 ms → would fail | PASS |
| 3 | another slot's sdlc.test shard | 112.12 | 1304-1309 | 78.258 | 291.819 (3.73) | 99.066 (1.27) / 92.093 (1.18) | 160.993 ms → would fail | PASS |
| 4 | as-is, quietest of the four | 11.16 | 1246 | 62.383 | 163.678 (2.62) | 75.337 (1.21) / 78.547 (1.26) | 105.854 ms → would pass | PASS |

Run 3 is roughly twice the load the filer saw (load ~56), and the verdict did not move: every work bound
held in all twelve passes (native accounting exact, table 2, usage 56, argv/cwd 0, env 1545-1555 cold and
0-4 steady, one near-cap write of 4,190,350-4,193,832 bytes), and the worst steady ratio across all ten
measured steady passes to date is 1.38 against K = 2.0. Runs 1-3 against run 4 are also the flap in one
place: the old budget failed the loaded three and passed the quiet one, on code that did not change.

The cold ratio's own spread is the reason it is reported rather than asserted — 2.62 at loadavg 11.16
against 3.73 at 112.12, a 42% swing for the same work (1555 and 1550 environment calls).

One more argument for bounding the cold pass by work rather than CPU, from run 1: at loadavg 15.10 with
1,266 rows its first pass cost 226.846 ms, while the filer's run at load ~14 with 1,270 rows cost
142.409 ms. That is +59% CPU for +1.3% native calls (1605 vs 1585) at effectively the same load, so the
cold pass's absolute CPU is not reproducible even at a matched loadavg — memory pressure and what the
other slots are doing move it too. Its call count is reproducible, which is what the bound now uses.

**Full-suite record, attempt 1 on head `6a7940eec`: red, and not from this change.** `sdlc local-check run
--name suite` waited 494 s for a full-suite slot, then ran 19m51s and exited 2 (make pid 34945,
22:13:31-22:33). Seven tests failed across five packages, and every one is a timeout or a missing-tool
verdict rather than a wrong answer:

| package | test | verdict |
|---------|------|---------|
| broadcast | `TestPointerRateLimit` | "only 11 of 50 rapid posts were limited" |
| couchcmd | `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees/warm-reattached` | "signal: killed"; the fixture's push reports "'origin' does not appear to be a git repository" |
| couchcmd | `TestClipboardCopyConsoleKeysReachContentFreeFactLog` (both console-key subtests) | "missing real execution evidence", ElapsedMS 1108 and 2177 |
| couchcmd | `TestParkHelperAttachmentRequiresClientGrowth` (both subtests) | "zellij --session pair-readiness action list-clients: context deadline exceeded" |
| couchcore | `TestRecoveryFreshOrdinarySlotKeepsOrdinaryLaunch`, `TestRecoveryFreshMissingRecordLeavesForeignOwnersUntouched/arrives-during-prepare` | "git worktree list --porcelain -z: timed out after 5s: signal: terminated" |
| couchtty | `TestWorkbenchPerformanceIdleUITick` | eight activity probes "context deadline exceeded" at 2.3-6.1 s each, then "activity worker did not finish" |
| diagnosticlog | `TestOpenFileInspectionConformance`, `TestUnregisteredChildOpenFileBlocksLegacyExpiry` | "lsof unavailable or incomplete" |

This change cannot reach any of them. The branch patch is five files — `resources_test.go` (whose
`TestResourceSamplerNativeCost` is env-gated and skipped in a suite run), the two new `hostLoadAverage`
files, `atlas/couch.md` and this issue — and `go test ./cmd/internal/procutil` passed in the same run
(1.306 s). What the run did have was load: `sysctl vm.loadavg` read `{ 5.40 78.11 97.03 }` at 22:34, right
after the run, so the 5- and 15-minute averages across the window were 78 and 97 while ariadne's sharded
suite and other slots were also running. A 5 s `git worktree list` budget, a zellij IPC deadline, an `lsof`
walk and a rate limiter's real-time window are exactly what a loadavg near 100 consumes.

Retained: `/tmp/pair644-suite-attempt1.log` and the record's own copy at
`/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/dbdf8d64d556e87195180367761a97a2.log`. Today's
other failed suite logs on this machine (19:46, 17:55, 16:13, 15:28, 12:48) fail *different* tests —
their owners' in-flight conformance work — so this is not a repeat of one known red.

Attempt 2 is the same code, unchanged, and waits for pair:3 to release the two full-suite tokens for its
quiet pair#658 run: a rerun inside another slot's quiet window would load both runs and prove nothing.

The seven are filed as #659 (unclaimed, at the hub's direction — one issue for the class rather than an
addition to #485, which is narrower: the wrapcmd harness timers from #479's `-race` soak). The class is a
test that puts a wall-clock budget on an external tool or on real time and then asserts the budget. This
issue's own rerun is not blocked on #659: the patch here touches none of those packages.

**Full-suite record, attempt 2 on head `615ab5d90`: green.** Exit 0 in 11m54s, log sha256 `05bf3a74a00c`
on Mac.localdomain, recorded as `sdlc-local/suite` on `615ab5d9`. Zero `--- FAIL` lines in the whole run,
and every package that was red in attempt 1 passes: broadcast 29.752 s, couchcmd 48.515 s, couchcore
291.124 s, couchtty 51.255 s, diagnosticlog 28.689 s.

The two attempts together turn the load attribution from an argument into a measurement, because nothing
changed between them but the host:

| package | attempt 1 (loadavg 78-97) | attempt 2 (loadavg 10-27) | speedup |
|---------|---------------------------|---------------------------|---------|
| couchcmd | 298.719 s, 3 tests red | 48.515 s, green | 6.2× |
| couchcore | 804.234 s, 2 tests red | 291.124 s, green | 2.8× |
| couchtty | 375.715 s, 1 test red | 51.255 s, green | 7.3× |
| diagnosticlog | 206.398 s, 2 tests red | 28.689 s, green | 7.2× |
| broadcast | 63.815 s, 1 test red | 29.752 s, green | 2.1× |
| whole run | 19m51s, exit 2 | 11m54s, exit 0 | 1.7× |

Attempt 2 also waited 1172 s for a full-suite slot (ariadne's sharded run held both tokens), so it started
as the machine was quieting; `sysctl vm.loadavg` read `{ 9.90 11.99 26.83 }` at its end against
`{ 5.40 78.11 97.03 }` at attempt 1's. The two runs are effectively the loaded and quiet ends of one fleet
day on identical code — which is the same flap shape this issue fixes in the sampler gate, one level up: a
verdict that moves with the host instead of with the code.

Retained: `/tmp/pair644-suite-attempt2.log`, and the record's own copy at
`/Users/xianxu/workspace/pair/.git/sdlc/local-checks/615ab5d908379a1defe1efd3efa08ad64f60eace/suite.log`.

**Run 5, on the landing head.** Appended rather than folded into the table above, whose counts this
supersedes. The conformance re-run once on the exact head being closed, `30b84d0c1`, at loadavg 18.78:
PASS in 0.69 s. Rows 1255-1258, history-write baseline 51.847 ms, cold pass 165.072 ms (ratio 3.18),
steady passes 63.135 ms (1.22) and 63.007 ms (1.22); env 1545 cold then 4 and 0, usage 56, table 2,
argv/cwd 0, one near-cap write of 4,190,350-4,193,832 bytes, `steady_sixty_second_CPU_pct=0.105119`.

That makes five runs of unchanged code at loadavg 11.16, 15.10, 18.78, 23.23 and 112.12 — fifteen passes,
twelve of them steady — one verdict everywhere, and the worst steady ratio anywhere is still 1.38 against
K = 2.0. The old fixed budget splits the same five: it would have failed the three loaded runs (means
125.687, 149.202 and 160.993 ms against 120 ms) and passed the two quiet ones (105.854 ms, and run 5's
97.071 ms). A gate whose answer depends on which of those the host happens to be having is the bug this
issue was filed for.

### 2026-10-17 — post-close: the review's two advisory findings

Close shipped the window (`Review-Verdict: SHIP`, a8f352e1..988914e0) with two Minor findings, and both
are worth landing rather than noting. The first is the same family the hub already ruled on for the atlas —
a measured number written into prose is stale the day after it is measured:

- `atlas/couch.md` said the conformance "gates work rather than wall time". The gate it replaced measured
  absolute sampler-thread CPU, not wall time, and the new one still gates steady CPU — as a ratio to a
  same-run baseline. It now says that.
- The test's doc comment credited "native call counts moved under 0.2%" to loadavg 15.10-112.12, but that
  figure came from the planning runs at 30.70-34.77; the verification runs show cold environment calls
  1545-1555 (about 0.6%) and steady native calls 60-66.
- `costSteadyRatioK`'s comment quoted a cold-shaped ratio of "3.04-3.76" that run 4 (2.62) and the
  reviewer's own run (3.78) had already left behind on both sides.
- The per-pass basis comment claimed the row-scaled bounds "leave at least 38% of rows of slack" as though
  the code guaranteed it. That was one host's observation; the comment now says the slack is measured here
  rather than promised.

The second is a gap against this issue's own spec bullet 3: the persistence-error `t.Fatal` inside the pass
loop was the only failure there that did not name the host state. It now carries the same rows-and-loadavg
string as the incomplete-measurement, work-bound and steady-ratio failures, so the atlas's "every failure
names it" is true rather than nearly true.

Run 6 on the edited test: PASS at loadavg 40.69, rows 1290-1298, baseline 66.195 ms, cold 234.833 ms
(ratio 3.55), steady 84.074 ms (1.27) and 82.267 ms (1.24), every work bound held. With the reviewer's
independent run at loadavg 60.31 during the close (cold 3.78, steady 1.16 and 1.15), the gate now has six
passes across loadavg 11.16 to 112.12 and one verdict. `gofmt` clean; `go vet` clean on darwin and on
GOOS=linux, which covers the not-darwin load-average stub. Landed by `sdlc close --fixed-then-ship`, which
reviews only this delta.

## Revisions

- 2026-10-17, after the hub's plan approval: the atlas bullet dropped its measured milliseconds. The plan
  as filed would have put the cold pass's measured CPU range (161.8-251.8 ms) and env-call counts into
  `atlas/couch.md`; the hub ruled that the atlas keeps the work bound and the gate shape only, since numbers
  in prose go stale, and that the cold ratio belongs in the test's report line. The measurements stay in
  this issue's `## Log`, which is where the derivations live. No gate changed.
