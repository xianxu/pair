# Boundary Review — pair#208 (milestone M1)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..b08fabc053687e1b415dc6aaca74a0e5f3c9892b |
| command | sdlc milestone-close --issue 208 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-06T23:35:17-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The Go probe is the strong half of this boundary: one in-process timer serves both modes, the positive control is real and effective (I confirmed `/usr/bin/true` reads 1.543 ms and a pipe hop 0.007 ms — matching `#201`'s 1.9 ms and the ~7 µs baseline), and the `GO_BINS` mechanism the plan-quality gate flagged was verified rather than assumed (`bin/pair-hoprtt` exists, `make -n` fires the recipe). Two things block SHIP. First, `spawnRTT` discards `exec.Cmd.Run()`'s error, so a command that never ran is reported as a fast measurement: `pair-hoprtt -spawn 5 -- zellij action query-tab-names` with no active session prints **9.531 ms** and exits 0, sitting next to perf.sh's own printed "zellij ~13ms" baseline — a reader concludes the round-trip is healthy when zellij printed "There is no active session!" and exited 1. That is exactly the failure this issue was filed to eliminate ("a measurement without a positive control cannot distinguish 'no effect' from 'my instrument is broken'"), and the shipped control's floor was loosened from the plan's `0.5` to `0.05`, which is below the 0.637 ms a failed exec actually reads. Second, a 2.9 MB Mach-O `pair-hoprtt` is committed at repo root, referenced by nothing, and the follow-up commit added it to `.gitignore` without `git rm --cached` — irreversible in a base-layer repo's history once merged. A caveat on scope: `ps`, `top`, `sysctl` and `iostat` are denied to this review shell, so I exercised perf.sh's collection paths only in their failure mode; findings about the collection *code* (the stray `0` line, the `n/a` guard asymmetry) are reproduced independently of that, but I could not measure the 6 s budget.

## 1. Strengths

- **The positive control is genuine, not decorative.** `main_test.go:50` would go red on the 18.7 ms harness bug it names, and the relational test `TestPipeHopIsFarCheaperThanSpawn` (`main_test.go:75`) catches the "both modes accidentally measure the same thing" failure that absolute bands miss. All five tests pass in 2.0 s.
- **ARCH-DRY on the Go side is honored.** `msSince` (`main.go:100`) is the single timer for both modes; there is no second implementation to drift, and no pre-existing percentile helper in the tree was duplicated (`grep percentile` → 0 hits outside this file).
- **perf.sh follows its own rule 1 literally.** There is genuinely no arithmetic in the shell beyond a 3-term swap-rate awk; the pid join is left for tested Lua, which is the right call and the plan's most defensible decision.
- **The `ps %cpu` lesson is encoded in code, not just prose** — `perf.sh:40-42` explains why one `top -l 2` feeds two rows and why its first sample must be discarded, and `SWAP_A` (`:67-69`) rides the main window rather than paying its own sleep. That overlap technique is the fix finding #6 needs.
- **`TestUsageErrorsRatherThanSilentlyMeasuringNothing`** (`main_test.go:92`) pins all three argv-validation paths; verified non-zero exits.

## 2. Critical findings

**C1 — `spawnRTT` swallows the run error, so a failed command reads as a measurement (`cmd/pair-hoprtt/main.go:99`).** `_ = c.Run()` discards the error the source raised; the elapsed time of the failure is appended as a sample. Reproduced:

```
$ hoprtt -spawn 5 -- /no/such/binary   →  0.637 0.679 0.679 5   (exit 0)
$ hoprtt -spawn 5 -- zellij action query-tab-names
      →  9.531 9.757 9.757 5   (exit 0)   # zellij: "There is no active session!", exit 1
```

`doctor/perf.sh:118` guards only `command -v zellij`, i.e. *absent*; the plan's ARCH-ORDER row says "`zellij` is absent **or the session is gone** → record `n/a` with the reason". The session-gone half ships as a fabricated 9.5 ms against a printed 13 ms baseline. Fix the **class** in `spawnRTT`, not at the zellij call site: count failures, and either return `(samples, failures)` so `report` can print `n/a (cmd failed N/N: <err>)`, or exit non-zero when every run failed. `/usr/bin/true` has the same hole in perf.sh (the *test* Stat-guards it; perf.sh does not). Then add the test that goes red without it — `-spawn 3 -- /no/such/binary` must not print a median.

**C2 — a 2.9 MB compiled binary is committed at repo root (`pair-hoprtt`, added in `c290191c`).** `git ls-tree` at HEAD shows `100755 blob … pair-hoprtt`, `file` says Mach-O arm64, and nothing in the tree references it (`GO_BINS` builds to `bin/`, perf.sh reads `$PAIR_HOME/bin/`). The next commit added `/pair-hoprtt` to `.gitignore`, which does not untrack an already-tracked path. Fix: `git rm --cached pair-hoprtt` **before** this boundary is crossed — after merge into a base-layer repo that propagates to dependents, removing it costs a history rewrite.

## 3. Important findings

**I1 — `grep -c` + `|| echo 0` emits a stray bare line, breaking the `key=value` contract (`doctor/perf.sh:50,51`).** `grep -c` prints `0` *and* exits 1 on no match, so the `||` fires and appends a second `0`. Reproduced in a real capture:
```
pair_family_procs=0
0
build_procs=0
0
```
`build_procs` matching nothing is the *common* case, so nearly every capture ships a malformed line into the M2 parser. Fix: drop `|| echo 0` (grep already prints 0), or `{ ...; } || true` around the substitution.

**I2 — `at_ns` carries whole seconds (`doctor/perf.sh:73,83`).** `date +%s` is epoch seconds; the key claims nanoseconds. Two consequences: a consumer that trusts the name is off by 10⁹, and the *actual* window (which on a struggling machine exceeds `sleep 2` — the whole reason per-sample stamps exist rather than just `window_seconds`) is measured at 1 s resolution, giving ±50 % on every rate the join computes. `date +%s.%N` works on this host (verified). Rename to `at_epoch_s` and take sub-second precision.

**I3 — the degrade-to-`n/a` rule is applied to the probe block only (`doctor/perf.sh:36,38,46` vs `:52`).** When the underlying tool fails, `load=` and `cpu_idle_pct=` emit empty values and `process_count=0` emits a fabricated zero — a physically impossible reading that a downstream consumer will read as data. `windowserver_cpu_pct` one line away *does* carry the `(v==""?"n/a":v)` guard. Same for `## disk`: `command -v iostat` tests presence, not success, so a failing iostat produces an empty section with no `n/a` and no reason (reproduced). Sweep the guard across every `kv` whose value comes from an external tool. (ARCH-SECURE at-review: the failure path must degrade visibly rather than substitute a value downstream reads as evidence.)

**I4 — the 6 s budget is documented as "Enforced, not hoped" and nothing enforces it (`doctor/perf.sh`).** `grep -n 'deadline\|timeout\|skipped'` → zero hits. `ZJ_SAMPLES=5` is a fixed count, not a deadline; there is no elapsed check and no "probes skipped" reporting, both of which the plan's ARCH-CONSTRAINTS and its ARCH-ORDER budget-exceeded row require. Structurally the three waits are serial and un-overlapped — `top -l 2` (~1–2 s) → `sleep $WINDOW` (2 s) → `iostat -w 1 -c 2` (~1–2 s) — leaving no headroom on a *healthy* machine before probes, and unbounded room on the degraded one this tool exists for. `SWAP_A` at `:67` already demonstrates the fix: start `top` and `iostat` in the background before the window and collect after. Also give `spawnRTT` a per-command timeout (`exec.CommandContext`); a wedged `zellij action` currently hangs `c.Run()` forever, and M2's async capture would never fire its callback (ARCH-ORDER: no cancellation path, extent not lexically bounded).

**I5 — `perf.sh` ships with no regression test and no entry in `make test`'s hand-maintained target list (`Makefile.local:113`).** `doctor/doctor_test.sh` + `test-doctor` (`Makefile.local:266`) is the established local pattern for exactly this kind of shell diagnostic, and it was not followed. I1, I2 and I3 are all what a ten-line "every non-blank line outside a `###` section is `key=value`" assertion catches. This is the third instance of the family the plan-gate ledger already opened as `unverified-repo-mechanism` (GO_BINS — fixed; `make test-lua` — PQ-6, still open; `make test` — this). Sweep the enumeration in one pass. It also matters for ARCH-MOCK: perf.sh's output *is* the contract M2's `delta` consumes, and pinning it means checking in a real recorded capture as the fixture M2 tests against — otherwise the fixtures will be written from the plan rather than from what the script emits.

**I6 — the positive control's floor was loosened 10× below the value that catches the bug next door (`cmd/pair-hoprtt/main_test.go:56`).** The plan's own M1.1 snippet specifies `med < 0.5`; shipped is `med < 0.05`. The test comment justifies only the 15 ms *ceiling*. At 0.05 the control no longer rejects the 0.637 ms an instantly-failing exec produces (C1) — the guard was widened past the lie it exists to catch. Restore `0.5`, or record the reason in a plan revision.

**I7 — `pair-hoprtt` lands on the operator's PATH, contradicting a documented invariant (`Makefile.local:32` → `:84`).** `install` iterates `GO_BINS`, and the comment three lines above the changed line states "Pair remains the single user-facing toolbox binary (#104)", with couch's pre-exec helper as the one recorded exception. `perf.sh` resolves the probe at `$PAIR_HOME/bin/` and never consults PATH, so nothing needs it installed. Either split a `BUILD_ONLY_BINS` out of `GO_BINS`, or amend that comment with the reason a general-purpose *arbitrary-command runner* belongs in `$HOME/.local/bin`.

**I8 — docs gate: the new runnable surface is undocumented in both READMEs.** `atlas/index.md` was updated (good), but the doc it points at, `doctor/README.md`, has a "## Run it" section and gained nothing for `perf.sh` — which the plan explicitly designs to be run standalone by an agent. And root `README.md:260` asserts "`make install` also installs a **second** binary, **`couch`**", which I7 makes false. Both are the same class: user-facing surface added without its doc.

## 4. Minor findings

- ARCH-DRY: the `vm_stat … awk /Swapins/…` pipeline is duplicated verbatim at `doctor/perf.sh:69` and `:95`, in a file that defines `sample()`/`cputimes()` helpers 25 lines earlier — should be `swapsample()`. The three probe-output awks (`:115,116,118`) are likewise one `probe_kv <prefix> <cmd…>` helper.
- `PROBE="${PAIR_HOME:-}/bin/pair-hoprtt"` (`:23`): with `PAIR_HOME` unset the degraded message says "not built — run make build", which misdiagnoses an env problem as a build problem. `doctor.sh:27` self-locates via `$(cd "$(dirname "$0")" && pwd)`; perf.sh should do the same, or say "PAIR_HOME unset".
- `summary(nil)` panics on index `-1` (`main.go:41-45`). Unreachable from today's `main`, but it is one new call site away and C1's fix adds one.
- `build_procs` (`:51`) greps `compile$|link$|^go$`; the plan's table and the issue Spec both say "go/compile/link/**test**".
- `sample()`'s awk (`:61`) prints `$4` only, truncating `comm` at the first space — macOS `ps -o comm=` emits full executable paths, which routinely contain spaces.
- `top -l 2 -n 60` (`:43`) has no `-o cpu`; macOS `top` orders by pid, so WindowServer — the row the plan calls "the untested candidate" and the reason for the render-path work — may fall outside the 60-row window and silently read `n/a`.

## 5. Test coverage notes

The Go tests are good where they exist and I verified they pin real behavior, not the implementation. The gap is one-sided: every test covers a mode that *works*, none covers a mode that *fails*. The three failure paths that ship a wrong answer rather than an error — exec failure (C1), collection-tool failure (I3), malformed output (I1) — have zero coverage, and one of them (I1) fires on essentially every real invocation. Concretely, add: (a) `-spawn N -- /no/such/binary` must not print a median; (b) `doctor/perf_test.sh` asserting the `key=value` line shape and that a stubbed-failing `top`/`ps` on `PATH` yields `n/a`, not empty-or-zero — that stub is also the ARCH-MOCK seam perf.sh currently lacks; (c) wire it into `make test`. Note also that `build(t)` runs `go build` once per test (4×); a `sync.Once` or `TestMain` would cut ~1.5 s.

## 6. Architectural notes for upcoming work

ARCH-PURE and ARCH-DRY(Go) **pass** — `summary` is genuinely pure and unit-tested without IO, the timer is shared, and perf.sh's no-arithmetic rule keeps the error-prone join in testable Lua. ARCH-PURPOSE, ARCH-MOCK, ARCH-CONSTRAINTS, ARCH-SECURE and ARCH-ORDER each **flag**, cited above. For M2: (a) `delta`'s fixtures must be *recorded from a real perf.sh run*, not hand-written, or they will encode the plan rather than the format — and they should include a capture taken while `zellij`/`iostat` fail, so the vanished/started/reused-pid cases meet the degraded rows too; (b) PQ-6 is still open and shares I5's family — `artifactpath` is a Go internal package Lua cannot call, and `make test-lua` (`Makefile.local:193`) is a hand-maintained recipe that no M2 task adds `nvim/doctor_test.lua` to; sweep the hand-maintained-list class once rather than per-site; (c) I4's missing deadline is what makes M2.4's async `on_exit` unbounded — fix it in M1 or M2 inherits a capture that can hang forever with the operator's note held hostage in the callback.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000208-pairdoctor-perf-capture-plan.md` covering the five places the plan now claims what the code does not deliver:

1. **ARCH-CONSTRAINTS / ARCH-ORDER budget rows** — "perf.sh takes a deadline… Exceeded → the report says which probes were skipped" is not implemented (I4). Implement, or revise to state the budget is advisory and say what replaces the enforcement.
2. **ARCH-ORDER zellij row** — "`zellij` is absent **or the session is gone** → record `n/a` with the reason": only the *absent* half ships (C1).
3. **ARCH-SECURE argv allowlist** — the plan promises argv for `go/compile/link/zellij/pair*/nvim`; perf.sh emits `comm` for everything. The code is *more* conservative, so revise the plan (or defer the allowlist to M2 explicitly) rather than loosening the code.
4. **M1.1 test band** — the plan's snippet says `med < 0.5`, shipped is `0.05` (I6). Record the value and its reason in one place.
5. **Snapshot table** — "the fleet: count **and aggregate RSS**" and "build storm: go/compile/link/**test**" are not delivered as stated; either add them or mark them derived by `delta` in M2.

---

## Re-review — 2026-09-06T23:39:21-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..4f9365b34c7d2309affa5a2a1a67f549cf7323a3 |
| command | sdlc milestone-close --issue 208 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-06T23:39:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1's architectural bet is sound and well executed: the pid join lives as pure Lua in `nvim/doctor.lua`, tested headlessly under `nvim -l`, and `doctor/perf.sh` genuinely does no arithmetic — the plan's central decision is honored, not just written down. `pair-hoprtt` is real, the `/usr/bin/true` positive control is in the suite and passes, and `GO_BINS` + the recipe stanza actually produce `bin/pair-hoprtt` (verified). What blocks SHIP is that the deliverable reproduces the very failure mode the issue exists to eliminate: **a probe that reports a plausible number when the measurement did not happen.** I reproduced three instances — `zellij action query-tab-names` exits 1 here ("There is no active session!") and `perf.sh` printed `zellij_action_ms=10.183`; `parse_duration('12:garbage')` returns `12`; and a pid alive in both samples but missing a cputime row lands in *no* bucket of `delta`, contradicting that function's own documented contract. Add a 2.9 MB Mach-O binary committed at the repo root (against the `/pair-hoprtt` ignore rule this same window added), and this needs a fix pass before the boundary. All fixes are localized; the design does not need rework.

**Verification limits, stated honestly:** `ps`, `top` and `sysctl` are denied in my shell, so I could not validate the collection rows. The `time sh doctor/perf.sh` = 2.19 s I measured is therefore **not** evidence for the ≤6 s budget — most collectors did not run. It did, however, expose the degradation behavior when those tools fail (finding I-1).

---

## 1. Strengths

- **`nvim/doctor.lua:56-99` — the join is where the plan said it would be, and it is genuinely pure.** `delta` takes two sample tables and a window; the vanished / started / reused-pid interleavings are reproducible in a test with no editor, no IO, no fixture harness. This is the ARCH-ORDER "seam to inject ordering" the entry asks for, and most diffs don't have one.
- **`cmd/pair-hoprtt/main_test.go:44-56` — the positive control exists, runs, and its band is justified in the comment.** Reverting it is not needed to see it bite: an 18.7 ms harness fails a 15 ms ceiling. `TestPipeHopIsFarCheaperThanSpawn` is a good second axis — it catches the two modes collapsing into one measurement.
- **`Makefile.local:32,349-353` — the `GO_BINS` trap the plan gate flagged was actually swept**, with the reason recorded in the recipe comment rather than in a commit message. `bin/pair-hoprtt` exists on disk; `make build` works.
- **`doctor/perf.sh:14-17` — the `ps %cpu` prohibition is enforced structurally**, not just documented: the script emits `time=` and `etime=` and lets Lua compute rates. The `etime`-went-down reused-pid detector (`:57-59` → `doctor.lua:83`) is a genuinely non-obvious case handled correctly.
- **ARCH-SECURE went the conservative direction**: no `env` dump, no argv at all (safer than the plan's own allowlist design).

## 2. Critical findings

**C-1 — `cmd/pair-hoprtt/main.go:96` — `spawnRTT` discards the exit status, so a failing command is reported as a healthy latency.**
`_ = c.Run()` times the fork+exec of a command that may never have executed. Reproduced on this machine:
```
$ hoprtt -spawn 10 -- /usr/bin/definitely-not-here
0.599 0.726 0.726 10        # exit 0
$ zellij action query-tab-names
There is no active session!   # exit 1
$ sh doctor/perf.sh | grep zellij
zellij_action_ms=10.183       # reads as a healthy ~13ms baseline
```
`perf.sh:117` guards only `command -v zellij`, which is precisely *not* the failure the plan's own ARCH-ORDER row names ("zellij is absent **or the session is gone** → record `n/a` with the reason"). The consequence is worse than a missing row: it fabricates evidence for the one hypothesis (`#201`/`#203`'s round-trip cost) this issue was filed to settle, and M1.4's "verify zellij ~13 ms" was signed off against exactly this fabricated number.
*Fix sketch:* count failures in `spawnRTT`; if any sample's `Run()` errored, print `failed=N` as a fourth/fifth field (or exit non-zero with the first error on stderr) and have `perf.sh` render `zellij_action_ms=n/a (command failed: <err>)`. Then re-verify M1.4 from inside a live zellij session.

**C-2 — `nvim/doctor.lua:79-92` — a pid alive in both samples but missing from one cputime sample is silently dropped, which is the exact failure `delta` was written to prevent.**
The docstring promises three cases "each counted rather than silently dropped". There is a fourth: `if ca and cb then` has no `else`. Reproduced:
```
procs A={1,2} B={1,2}; cpu A={1,2} B={1}
→ rates=1 vanished=0 started=0 reused=0 rows_a=2 rows_b=2
   (pid 2 is alive in both samples and appears in no bucket)
```
This is reachable in production, not hypothetical: `perf.sh:60-65` collects `procs` and `cputime` in **two separate `ps -A` invocations** per sample, so their pid sets genuinely differ — and they diverge *most* under a spawn storm, which is the scenario the vanished/started counters exist for.
*Fix sketch:* two changes, both needed. (a) In `perf.sh`, make it one pass — `ps -Ao pid=,etime=,time=,rss=,comm=` — so the two views cannot skew. (b) In `delta`, add `out.no_cputime = out.no_cputime + 1` in the `else`, and assert in a test that `#rates + vanished + started + reused + no_cputime` equals the union of pids. That invariant is what makes the counters trustworthy.

**C-3 — a 2.9 MB Mach-O arm64 build artifact is committed at the repo root.**
`c290191c` added `pair-hoprtt` (2 896 786 bytes, `Mach-O 64-bit executable arm64`) at the tree root; `95f73548` then added `/pair-hoprtt` to `.gitignore:53` — which does nothing, because the file is already tracked. It is still tracked at HEAD and the working tree is clean. This is a base-layer repo whose files propagate to dependents.
*Fix sketch:* `git rm --cached pair-hoprtt && rm pair-hoprtt`, then commit. Since the branch is unmerged, prefer rebasing `c290191c` to drop the blob entirely rather than leaving 2.9 MB permanently in main's history.

## 3. Important findings

**I-1 — `doctor/perf.sh:32-52,60-65,103-107` — collector failures render as values, not as `n/a`.** Same rule as C-1, different half of the file. With `ps`/`top`/`sysctl` unavailable (reproduced in this shell), the capture emitted:
```
host_cores=?          load=            cpu_idle_pct=
process_count=0       windowserver_cpu_pct=n/a
### cputime  (empty)  ### procs  (empty)   ## disk  (empty)
```
`process_count=0` is a fabricated claim, and empty `load=` / `cpu_idle_pct=` / empty sample blocks are indistinguishable from a real reading of nothing. This contradicts the file's own header rule ("Every probe degrades to `n/a` with a reason"). It is realistic, not academic: the header advertises standalone re-runs by an agent, and agents in this repo run under a sandbox that denies exactly these tools. *Fix:* one `probe()` helper that substitutes `n/a (<tool> unavailable)` on empty/failed output, applied to every collector including the two sample blocks.

**I-2 — `doctor/perf.sh:50-51` — `grep -c … || echo 0` emits a stray bare line, corrupting the report on any quiet machine.** `grep -c` prints `0` *and* exits 1, so the `||` fires and appends a second `0`. Reproduced: `build_procs=0` followed by a bare `0` line — and `build_procs` is zero whenever no build is running, i.e. on the healthy-baseline capture the Done-when requires. *Fix:* drop the `|| echo 0` (grep already prints `0`), or use `| grep -cE … || true`.

**I-3 — `doctor/perf.sh:50-51` — `comm` on macOS is a path, so `^go$` can never match and the `pair` pattern over-matches.** The repo's own `doctor/emitter-health.sh:25-31` documents `comm` as a full path. `build_procs` anchors `^go$` against `/opt/homebrew/…/bin/go` → never matches, silently zeroing `#203`'s variable (`compile$`/`link$` still match). `pair_family_procs` matches unanchored `pair` against full paths, so every process launched from `/Users/…/workspace/pair/` counts. *Fix:* match on the basename (`awk -F/ '{print $NF}'` before grep) and anchor. Caveat: `ps` is denied in my shell, so this is reasoned from the path-format evidence above rather than executed.

**I-4 — `nvim/doctor.lua:71-73` — `parse_duration`'s validity guard is unreachable dead code, so malformed input yields a fabricated number.** `parts[#parts+1] = tonumber(piece)` never stores a `nil`, so `if not v then return nil end` can never fire. Verified: `parse_duration('12:garbage') == 12`, `parse_duration('garbage:12') == 12`, `parse_duration('1:2:3:4:5') == 13403045`. The docstring's promise ("A value we cannot read returns nil rather than 0 — 0 is a real number that would silently mean 'used no CPU'") is not delivered for any input containing at least one numeric field. This is the ARCH-SECURE at-review case: text from a subprocess, treated as well-formed, substituting a value downstream reads as evidence. *Fix:* `local v = tonumber(piece); if not v then return nil end; parts[#parts+1] = v`, plus a field-count bound (≤3 after day-stripping). Add `parse_duration('12:garbage') == nil` to the test — it goes red today.

**I-5 — `doctor/perf.sh` has no test and is in no make recipe, and `delta`'s tests use hand-written literals rather than the recorded fixtures the plan promised.** `doctor/doctor_test.sh` + `make test-doctor` (`Makefile.local:266-267`) is the established local pattern for exactly this file shape — and it exists because a prior M1 review asked for it (`#45`). The plan's ARCH-MOCK says "`doctor.delta` is tested against recorded fixture pairs **checked into the repo**"; what shipped is `sample({['1']={etime=100,…}}, {['1']=5})` — the author's model of the format, asserted against code written from the same model. Nothing pins that `perf.sh` actually emits what `delta` expects: no parser exists, so the seam is unexercised end to end. I-2 is precisely the bug a `perf.sh` smoke test would have caught (assert every line outside the sample blocks matches `^(#|$|[a-z_]+=)`). *Fix:* add `doctor/perf_test.sh` + a `test-perf` recipe wired into `make test`; capture one real `perf.sh` run into a checked-in fixture and drive `delta` from it.

**I-6 — `doctor/perf.sh` is missing from `runtimebundlegen.explicitAssetPaths` — the third instance of the hand-maintained-list family the plan gate already raised twice.** `cmd/internal/runtimebundlegen/generate.go:27-32` is a hand-maintained list (`doctor/README.md`, `doctor/SKILL.md`, `doctor/doctor.sh`, `doctor/emitter-health.sh`); `assetDirs` walks `nvim/` wholesale but **not** `doctor/`. Verified against the checked-in manifest: `doctor/doctor.sh` → present, `doctor/perf.sh` → absent. So an installed/Homebrew pair extracts a runtime with the drift half and not the perf half, and M2's `$PAIR_HOME/doctor/perf.sh` will point at nothing. `pair-hoprtt` is a second decision to make here — the bundle carries no helper binaries since `#104 M3`, so on an installed pair the probe must degrade rather than be found. This is the enumeration the `unverified-repo-mechanism` family implies (GO_BINS ✓, `artifactpath.NonArtifactSources` ✓, `explicitAssetPaths` ✗, a make test recipe ✗ — see I-5); ARCH-PURPOSE asks for the whole list swept in one pass. *Fix:* add `"doctor/perf.sh"` to `explicitAssetPaths` and to `embed_test.go:25-26`'s want-list, then `make runtimebundle-generate`.

**I-7 — the ≤6 s budget is declared "enforced, not hoped" and is not enforced.** The plan's ARCH-CONSTRAINTS says "`perf.sh` takes a deadline and each probe's sample count is sized to fit. Exceeded → the report says which probes were skipped rather than running long." `perf.sh` has no deadline, no elapsed tracking, and no skip path — only fixed counts. The composition is `top -l 2` + `sleep $WINDOW` + `iostat -w 1 -c 2` + 500 pipe hops + 30 fork+execs + 5 zellij round-trips, sequential and unbounded, on the only kind of machine it ever runs on. My 2.19 s measurement is not evidence (see the verification note). *Fix:* record `t0`, check remaining budget before each probe block, and emit `skipped=<probe> (budget)` — or revise the plan to say the budget is sized, not enforced (see §7).

## 4. Minor findings

- `doctor/perf.sh:73,83` — the key is `at_ns` but the value is `date +%s` (seconds). A live trap for M2's unwritten parser; rename to `at_s`.
- `doctor/perf.sh:69` and `:95` — the `vm_stat | awk '/Swapins/…'` pipeline is duplicated verbatim (ARCH-DRY); extract `swapstat()`.
- `doctor/perf.sh:23` — `${PAIR_HOME:-}` unset yields `/bin/pair-hoprtt`, and the degradation text then misdiagnoses ("not built — run make build"). Say `n/a (PAIR_HOME unset)` for that case.
- `cmd/pair-hoprtt/main.go:44-48` — `summary` indexes `samples[-1]`-style on an empty slice; unreachable today (both callers guarantee ≥1), but one line of guard makes it safe for reuse.
- `cmd/pair-hoprtt/main.go:76-84` / `perf.sh:115` — `pipeRTT` `break`s on a mid-run error and returns a short sample set; `perf.sh`'s `awk` prints only `$1 $2` and discards the count, so a 3-sample run reads identically to a 500-sample one.
- `doctor/perf.sh:61` — `awk '{print …$4}'` takes only the first whitespace token of `comm`, truncating paths with spaces; and `comm` emits full executable paths (workspace/project names) where the plan specified "process name and pid".
- `doctor/perf.sh:22` — `PAIR_PERF_WINDOW` flows unvalidated into `sleep` and `awk -v w=`; non-numeric gives an awk division error. Also undocumented in atlas/README.
- `4f9365b3` ("M2: the pure join") landed inside the M1 boundary. Benign for coverage — it is reviewed here — but M1's close will tick M1 while M2.2b is already done.
- `doctor/README.md` describes the `doctor/` contents and was not updated for `perf.sh` (atlas/index.md was — that gate passes).
- Issue `## Plan` M1 is unticked and `## Log` records no M1.4/M1.5 evidence. Given C-1, the M1.4 zellij number should be re-taken and logged, not carried forward.

## 5. Test coverage notes

- **Go:** 5/5 pass (`go test ./cmd/pair-hoprtt -count=1`), `go vet` clean, `go build ./...` clean, `artifactpath` green. The positive control is real and would fail on the 18.7 ms bug.
- **Lua:** `nvim -l nvim/doctor_test.lua` passes; it is already wired at `Makefile.local:206` (so plan-gate PQ-6's `test-lua` half was already satisfied — the `explicitAssetPaths` half in I-6 was not).
- **The gap that matters:** every test in this diff exercises code the author wrote against a format the author imagined. Nothing runs `perf.sh` and feeds its bytes to `delta`. C-1, C-2, I-1, I-2 and I-4 all live in that unexercised seam, and none of the five would turn a single existing test red. Concretely missing: a `perf.sh` output-shape test (kills I-2), a `parse_duration('12:garbage') == nil` case (kills I-4), a bucket-completeness invariant on `delta` (kills C-2), and a `-spawn` test against a failing command (kills C-1).
- No test asserts the drift payload is unaffected by the new code. `M.payload` is untouched so it cannot have drifted, but M2.5 promises this pin — pulling it forward is cheap insurance.

## 6. Architectural notes

- **ARCH-DRY — flag (Minor).** One shared `msSince` timer across both hoprtt modes is exactly right. Two duplications in `perf.sh`: the `vm_stat` awk twice, and two `ps -A` passes where one carries every field — the latter is not cosmetic, it *causes* C-2.
- **ARCH-PURE — pass, and it is the diff's best quality.** The error-prone join is a pure function with no vim API, no IO, running under `nvim -l`. `perf.sh` does zero arithmetic. The plan's central bet was kept.
- **ARCH-PURPOSE — flag.** The issue's own general rule is *"a measurement without a positive control cannot distinguish 'no effect' from 'my instrument is broken'."* The diff installs that control for the harness-vs-command axis (C-1's test) and omits it for the probes' own failure paths — C-1, I-1, I-4 are three instances of one rule appearing inside the deliverable that exists to eliminate it. Fix them as a class (every collector and probe reports absence as `n/a (<reason>)`, never as a value), not one site at a time.
- **ARCH-MOCK — flag.** Plan promised recorded fixture pairs; literals shipped. See I-5.
- **ARCH-CONSTRAINTS — flag.** Declared envelope is not implemented. See I-7.
- **ARCH-SECURE — flag (Minor + I-4).** No env dump and no argv is good and better than planned. `parse_duration` is the boundary parse that fabricates (I-4); full executable paths are a modest leak for a report designed to leave the machine.
- **ARCH-ORDER — pass with one flag.** `delta`'s two-table signature is a real ordering seam and the reused-pid case is handled; that is the highest-leverage thing the entry asks for. The flag is C-2's fourth interleaving, which arises precisely because the two `ps` passes are two events rather than one.
- **For M2:** decide the installed-pair story now (I-6). `:PairDoctor` will need a defined behavior when `perf.sh` or `bin/pair-hoprtt` is absent from an extracted runtime, and — given C-1 — "the probes returned numbers" must not be conflatable with "the probes ran".

## 7. Plan revision recommendations

Append a `## Revisions` section to `workshop/plans/000208-pairdoctor-perf-capture-plan.md`:

1. **M1.3 / degradation scope.** M1.3 claims per-probe `n/a` degradation. Delivered for a missing `pair-hoprtt` and a missing `zellij` *binary* only; a failed command (C-1) and every failed collector (I-1) still emit values. Either record the narrowed scope or state the fix as an M1 task.
2. **ARCH-CONSTRAINTS / budget.** "Enforced, not hoped: `perf.sh` takes a deadline… Exceeded → the report says which probes were skipped" describes code that does not exist. Revise to "sample counts sized to fit; no runtime deadline" or add the deadline as an M1 task. Note that M1.5's verification cannot be performed in a sandboxed shell where `ps`/`top` are denied.
3. **ARCH-MOCK / fixtures.** "tested against recorded fixture pairs checked into the repo" — delivered as hand-written literals with no parser and no fixture. Revise, or carry the fixture into M2.2b as an explicit deliverable.
4. **ARCH-SECURE / argv allowlist.** The plan specifies "process name and pid, not full argv, except an allowlisted set (`go`, `compile`, `link`, `zellij`, `pair*`, `nvim`) where the argv is the diagnostic value." Implementation emits no argv and full executable *paths*. Record the change (it is the safer direction) and note that `build_procs` is now the only signal for `#203`'s variable — which makes I-3 load-bearing.
5. **M1 file list / the hand-maintained-list enumeration.** M1.2b names `GO_BINS` alone. Write the full enumeration a new `doctor/*.sh` or `cmd/*` must be added to: `GO_BINS` + recipe, `artifactpath.NonArtifactSources`, `runtimebundlegen.explicitAssetPaths` + `embed_test.go`, and a `make test-*` recipe. This is the `unverified-repo-mechanism` family's third round; the enumeration is what stops a fourth.
6. **M2.2b landed early.** `doctor.delta` + its tests shipped in the M1 window (`4f9365b3`). Note it so M2's boundary base is not mistaken for the branch point.

```findings
findings:
  - id: new
    severity: Critical
    family: failure-reported-as-measurement
    title: |
      spawnRTT discards c.Run()'s error, so a failing command is reported as a healthy latency
    detail: |
      cmd/pair-hoprtt/main.go:96 uses `_ = c.Run()`. Reproduced: `hoprtt -spawn 10 -- /usr/bin/definitely-not-here` prints `0.599 ... 10` and exits 0, and because `zellij action query-tab-names` exits 1 here ("There is no active session!"), doctor/perf.sh printed `zellij_action_ms=10.183` — a fabricated healthy reading for the exact hypothesis (issue 201/203 round-trip cost) this issue exists to settle. perf.sh:117 guards only `command -v zellij`, not the session-gone case the plan's own ARCH-ORDER table names. Count failed runs and render `n/a (command failed: <err>)`; re-take M1.4's zellij number from inside a live session.
  - id: new
    severity: Critical
    family: silent-drop-in-join
    title: |
      delta silently drops a pid alive in both samples but missing one cputime row, contradicting its own contract
    detail: |
      nvim/doctor.lua:86-92 — `if ca and cb then` has no else, so a pid present in both procs samples but absent from one cputime sample lands in no bucket. Reproduced: procs A={1,2} B={1,2}, cpu A={1,2} B={1} yields rates=1 vanished=0 started=0 reused=0. Reachable in production because perf.sh:60-65 collects procs and cputime in two separate `ps -A` invocations, which diverge most under a spawn storm — the scenario the counters exist for. Fix both ends: one `ps -Ao pid=,etime=,time=,rss=,comm=` pass in perf.sh, and a counted `no_cputime` bucket in delta with a test asserting every input pid lands in exactly one bucket.
  - id: new
    severity: Critical
    family: build-artifact-committed
    title: |
      A 2.9MB Mach-O arm64 binary is committed at the repo root, against this window's own gitignore rule
    detail: |
      c290191c added `pair-hoprtt` (2,896,786 bytes, Mach-O 64-bit arm64) at the tree root; 95f73548 then added `/pair-hoprtt` to .gitignore:53, which has no effect on an already-tracked file. Still tracked at HEAD in a base-layer repo whose files propagate. Prefer rebasing c290191c to drop the blob entirely rather than `git rm --cached` in a follow-up, since the branch is unmerged and the history is still cheap to fix.
  - id: new
    severity: Important
    family: failure-reported-as-measurement
    title: |
      perf.sh collector failures render as values, not n/a, contradicting the file's own stated rule
    detail: |
      With ps/top/sysctl unavailable (reproduced in this shell), doctor/perf.sh emitted `host_cores=?`, `load=`, `cpu_idle_pct=`, `process_count=0`, and empty `### cputime` / `### procs` / `## disk` blocks — `process_count=0` being a fabricated claim and the empty values indistinguishable from a real reading. The header (:19-20) promises every probe degrades to `n/a` with a reason. Realistic, not academic: the header advertises standalone re-runs by an agent, and agents here run sandboxed with exactly these tools denied. One `probe()` helper substituting `n/a (<tool> unavailable)`, applied to every collector including both sample blocks.
  - id: new
    severity: Important
    family: report-line-contract
    title: |
      grep -c with `|| echo 0` emits a stray bare 0 line, corrupting the report on any quiet machine
    detail: |
      doctor/perf.sh:50-51 — `grep -c` prints 0 AND exits 1, so the `||` fires and appends a second 0. Reproduced: `build_procs=0` followed by a bare `0` line, which breaks the k=v line contract M2's parser will read. build_procs is zero whenever no build is running, i.e. on the healthy-baseline capture the Done-when requires. Drop the `|| echo 0`, or use `|| true`.
  - id: new
    severity: Important
    family: report-line-contract
    title: |
      macOS `ps -o comm=` is a path, so `^go$` never matches and the pair pattern over-matches
    detail: |
      doctor/perf.sh:50-51 anchors `^go$` against a full path like /opt/homebrew/.../bin/go, silently zeroing issue 203's build-storm variable (compile$/link$ still match). `pair_family_procs` matches unanchored `pair` against full paths, so every process launched from /Users/.../workspace/pair/ counts. The repo's own doctor/emitter-health.sh:25-31 documents comm= as a full path. Match on the basename before grepping and anchor. Caveat: `ps` is denied in my shell, so this is reasoned from that path-format evidence rather than executed.
  - id: new
    severity: Important
    family: failure-reported-as-measurement
    title: |
      parse_duration's validity guard is unreachable dead code, so malformed input yields a fabricated number
    detail: |
      nvim/doctor.lua:67-73 — `parts[#parts+1] = tonumber(piece)` never stores a nil, so `if not v then return nil end` can never fire. Verified: parse_duration('12:garbage')==12, ('garbage:12')==12, ('1:2:3:4:5')==13403045. The docstring's promise that an unreadable value returns nil rather than a number is not delivered for any input with at least one numeric field. Assign tonumber to a local and check it before appending, add a field-count bound, and add `parse_duration('12:garbage') == nil` to doctor_test.lua — it goes red today.
  - id: new
    severity: Important
    family: untested-shell-surface
    title: |
      doctor/perf.sh has no test and no make recipe, and delta's tests use literals rather than the promised recorded fixtures
    detail: |
      doctor/doctor_test.sh + `make test-doctor` (Makefile.local:266-267) is the local precedent, created in response to a prior M1 review. The 128-line M1 deliverable has neither. Separately the plan's ARCH-MOCK promises delta is "tested against recorded fixture pairs checked into the repo"; what shipped is hand-written literals asserted against code written from the same mental model, with no parser and nothing pinning that perf.sh emits what delta expects. The stray-0 bug above is exactly what a line-shape smoke test would have caught. Add doctor/perf_test.sh + a test-perf recipe wired into `make test`, and drive delta from a captured real sample.
  - id: new
    severity: Important
    family: unverified-repo-mechanism
    title: |
      doctor/perf.sh is missing from runtimebundlegen.explicitAssetPaths — third instance of the hand-maintained-list family
    detail: |
      cmd/internal/runtimebundlegen/generate.go:27-32 is a hand-maintained list; assetDirs walks nvim/ wholesale but not doctor/. Verified against the checked-in manifest: doctor/doctor.sh present, doctor/perf.sh absent. An installed or Homebrew pair therefore extracts the drift half and not the perf half, and M2's $PAIR_HOME/doctor/perf.sh will point at nothing. pair-hoprtt needs a decision too — the bundle carries no helper binaries since 104 M3. Add "doctor/perf.sh" to explicitAssetPaths and to embed_test.go:25-26, then `make runtimebundle-generate`. Sweep the whole enumeration (GO_BINS, artifactpath, explicitAssetPaths, make recipe) rather than this one site.
  - id: new
    severity: Important
    family: unenforced-operating-envelope
    title: |
      The 6s budget is declared "enforced, not hoped" but perf.sh has no deadline or skip path
    detail: |
      The plan's ARCH-CONSTRAINTS says perf.sh takes a deadline and reports which probes were skipped when exceeded. perf.sh has no elapsed tracking and no skip logic — only fixed sample counts across `top -l 2` + sleep + `iostat -w 1 -c 2` + 500 pipe hops + 30 fork+execs + 5 zellij round-trips, sequential and unbounded, on the only kind of machine it ever runs on. My 2.19s measurement is not evidence: ps/top/sysctl were denied in that run so most collectors did nothing. Either implement the deadline or revise the plan to say the budget is sized, not enforced.
  - id: new
    severity: Minor
    family: report-line-contract
    title: |
      at_ns holds `date +%s` seconds, a live trap for M2's unwritten parser
  - id: new
    severity: Minor
    family: duplicated-logic
    title: |
      The vm_stat awk pipeline is duplicated verbatim at perf.sh:69 and :95; extract swapstat()
  - id: new
    severity: Minor
    family: failure-reported-as-measurement
    title: |
      PAIR_HOME unset yields PROBE=/bin/pair-hoprtt and the degradation text misdiagnoses it as "not built"
  - id: new
    severity: Minor
    family: unguarded-edge-case
    title: |
      summary() indexes an empty slice; unreachable today but one guard line makes it safe for reuse
  - id: new
    severity: Minor
    family: failure-reported-as-measurement
    title: |
      perf.sh discards hoprtt's sample count, so a truncated pipe run reads identically to a full one
  - id: new
    severity: Minor
    family: report-line-contract
    title: |
      perf.sh:61 awk $4 truncates command paths containing spaces, and comm= emits full paths where the plan said process name
  - id: new
    severity: Minor
    family: unguarded-edge-case
    title: |
      PAIR_PERF_WINDOW flows unvalidated into sleep and awk -v, and is undocumented in atlas/README
  - id: new
    severity: Minor
    family: boundary-hygiene
    title: |
      4f9365b3 (M2.2b, the pure join) landed inside the M1 boundary; note it so M2's base is not mistaken for the branch point
  - id: new
    severity: Minor
    family: docs-gate
    title: |
      doctor/README.md describes the doctor/ contents and was not updated for perf.sh (atlas/index.md was)
  - id: new
    severity: Minor
    family: traceability
    title: |
      Issue Plan M1 is unticked and the Log records no M1.4/M1.5 evidence; given the zellij finding, M1.4's number must be re-taken before it is logged
```

---

## Re-review — 2026-09-06T23:55:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..2284b41247340aec386afa83eb998502bc14aa40 |
| command | sdlc milestone-close --issue 208 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-06T23:55:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The three Criticals from the last round are genuinely fixed and I revert-verified two of them (`delta`'s missing bucket and `parse_duration`'s dead guard both go red when undone), the stray-`0` line is now pinned by a real shape test, and `pair-hoprtt` now refuses to report a failing command as fast latency — I confirmed live that `zellij_action_ms` went from a fabricated `10.183` to `n/a (probe failed)`. What blocks SHIP is two things. First, the shipped `perf.sh` looks for its probe at `$PAIR_HOME/bin/pair-hoprtt`, and the runtime bundle's `bin/` contains exactly `pair-help` and `pair-notify` — so every installed or Homebrew pair loses all three latency probes and is told to "run make build", which it cannot. `doctor/perf.sh` was added to the bundle in this very window, so the file ships and its payload does not. Second, BR-5's claimed fix is dead code: `collect()` is defined at `doctor/perf.sh:32` with **zero call sites**, and re-running the capture in this shell still produces `load=`, `cpu_idle_pct=`, `process_count=0` and an empty `## disk` block — the exact fabricated-zero output the finding named, now with a helper standing next to it that reads as protection.

## 1. Strengths

- `nvim/doctor.lua:86-104` — `delta` now enumerates a **total** partition (vanished / reused / rates / unmeasured / started) and the comment says why each bucket exists. I reverted the `unmeasured` branch in a scratch copy and `doctor_test.lua` went red (`FAIL unmeasured counted, not dropped: got 0 want 1`). This is the strongest thing in the diff and it is the piece the plan correctly identified as most error-prone.
- `nvim/doctor.lua:40-49` — `parse_duration` validates *before* accumulating, with a comment naming the precise failure mode of the old shape (nil arithmetic errors before the guard runs). Revert-verified red (`got 60 want nil`).
- `doctor/perf_test.sh:19-26` — the STRAY-line check is a real oracle, not a restatement. I re-introduced `|| echo 0` into `count_comm` in a scratch copy and the test failed with `report contains a bare value line with no key`.
- `cmd/pair-hoprtt/main.go:116-126` — `report`'s shape rule (a 5th field appears only on failure, so a healthy line keeps its four-field shape) is a good design: the caller cannot mistake a broken probe for a fast one *and* an old parser is not broken by the addition.
- `workshop/lessons.md:3638-3658` — the hand-maintained-list lesson names all three lists and, crucially, notes that their guards fire at *different times*, which is why the class took several rounds. That is the enumeration ARCH-PURPOSE asks for.

## 2. Critical findings

**`doctor/perf.sh:23,161` — the shipped capture can never locate its probe.** `PROBE="${PAIR_HOME:-}/bin/pair-hoprtt"`, but the runtime bundle's `bin/` is `["bin/lib/adapt-log.sh","bin/lib/dev-rebuild.sh","bin/pair-help","bin/pair-notify"]` (read from the generated `manifest.json`) — no Go binaries since #104 M3, and `pair-hoprtt` is not among them. `make install` puts it in `~/.local/bin`, which is on `PATH` but is not `$PAIR_HOME/bin`. So an installed pair prints `probes=n/a (pair-hoprtt not built — run make build)` forever, losing pipe-hop / fork-exec / zellij — the three rows the issue's own evidence table exists to settle — and the degradation reason misdirects the reader to a command that does not apply to their layout. Fix: resolve the probe by trying `$PAIR_HOME/bin/pair-hoprtt`, then the repo's `bin/`, then `command -v pair-hoprtt`, and make the n/a reason say which lookups were tried.

**This is the 3rd finding in family `unverified-repo-mechanism`.** Do not fix only this site. The rule that covers all three: *a plan or a diff may name a repo path or mechanism only alongside a check that it resolves in **every layout the artifact ships to** — dev checkout, `make install`, and extracted runtime bundle — and from the **language and layer** the calling code sits in.* Prevalence 3/3 rounds that named a mechanism: PQ-1 failed the build path (`GO_BINS` overrides the `cmd/*` scan), BR-1 failed the language path (`artifactpath` is Go, the caller is Lua), this fails the install-layout path. The enumeration the rule implies, to be swept in one pass rather than one per round: for each new runtime file or binary, confirm (a) it builds, (b) it is in the bundle manifest, (c) its *callers'* path expressions resolve under all three layouts, (d) a test pins each.

## 3. Important findings

**`nvim/doctor.lua:128-131` + `nvim/doctor_test.lua:52` — `verdict` returns a claim when it has no measurement, and the test asserts that direction.** `math.max(tonumber(insert_ms) or 0, ...)` turns absent timings into `0`, so `verdict(nil, nil) == 'fast'`. Per the Spec (issue lines 77-81), "nvim is fast" is what *excludes the whole scheduling family* (`#201`/`#203`) for that symptom — so a report where nvim was never successfully timed will exclude two issues on the strength of a measurement that did not happen. `doctor.lua:118-124` states the correct rule three functions above (`note_from_lines` returns nil for blank because "an empty note would read as 'the operator said nothing was wrong', which is a claim, where absence is not"), and `verdict` breaks it. The test at `doctor_test.lua:52` is written from the same mental model as the code and happily asserts the wrong direction. Fix: return `nil` (or `'unknown'`) when neither timing is a number, and flip the assertion.

**This is the 6th finding in family `failure-reported-as-measurement`.** Do not fix only this instance. The rule: *every value-producing function in the capture — shell and Lua alike — must render an absent or failed measurement as **absence** (`nil` / `n/a (<reason>)`), never as an in-domain value.* Measured prevalence 6/6: BR-2 (`c.Run()` error discarded), BR-5 (empty collectors printing `key=`), BR-14 (a misdiagnosed degradation reason), BR-16 (a truncated sample run reading as a full one), the dead `collect()` below, and this. The enumeration to sweep in one pass: every `kv` call site in `doctor/perf.sh`, every exported function in `doctor.lua`'s `#208` block, and `report`/`summary` in `cmd/pair-hoprtt/main.go` — each gets one absent-input case in its test file.

**BR-5 not addressed, and the fix is unreachable.** `collect()` (`doctor/perf.sh:32-40`) has zero call sites — I grepped, and I re-ran the capture: `load=`, `cpu_idle_pct=`, `process_count=0`, `host_cores=?`, and `## disk` renders as an empty block with no key at all. `process_count=0` is a fabricated claim; `windowserver_cpu_pct` has an `n/a` default in its awk (`:76`) while `cpu_idle_pct` two lines earlier does not — that inconsistency is inside the same six lines. `perf_test.sh` does not catch this because `grep -q "^load="` matches `load=` exactly as well as `load=2.1/1.9/1.8`.

**BR-11 not addressed — the deadline is on the wrong end of the run.** `over_budget()` guards only `probe_line` (`:142`), i.e. the ~0.1 s tail. `top -l 2` (`:67`), `sleep "$WINDOW"` (`:104`) and `iostat -d -w 1 -c 2` (`:128`) are unconditional and are what actually consume the budget, so the total is still unbounded — and when it *is* exceeded, the thing skipped is the probes, the measurement the issue exists to collect, after the expensive collectors already ran. `perf_test.sh:36`'s `elapsed ≤ budget` assertion passed here only because `ps`/`top`/`sysctl` are denied in this shell and elapsed was 2 s; on the struggling machine this tool is for, that assertion goes red instead of the capture degrading.

**BR-9 not addressed in substance.** `doctor/perf_test.sh` + `make test-perf-capture` exist and are wired into `make test` — good — but they pin only the line shape and the missing-binary path. Nothing pins the two failure-rendering contracts the Criticals were about: a probe whose command fails, and a collector that produces nothing. And `delta` is still driven by hand-written literals; the plan's ARCH-MOCK says "tested against **recorded fixture pairs checked into the repo**", and there is still no parser and nothing asserting that `perf.sh`'s `### procs` / `### cputime` blocks are what `delta` consumes.

**BR-10 not addressed (the pin half).** `explicitAssetPaths`, both `artifactpath` lists, `GO_BINS` and the recipe are all done and I verified `doctor/perf.sh` is in the generated manifest. But `cmd/internal/runtimebundle/embed_test.go:25-26` still lists only `doctor/SKILL.md` and `doctor/doctor.sh` — the one place that would catch a future drop was named in the finding and skipped, so the class sweep is one enumerable site short (ARCH-PURPOSE at-review).

## 4. Minor findings

- BR-12 not addressed: `at_ns` still holds `date +%s` seconds (`perf.sh:97,107`) — a live trap for M2's parser, and `elapsed_seconds` shares that 1 s granularity against a 6 s budget.
- BR-13 not addressed: the `vm_stat` awk pipeline is still duplicated verbatim at `perf.sh:93` and `:119` (ARCH-DRY).
- BR-14 not addressed: `PAIR_HOME` unset still yields `PROBE=/bin/pair-hoprtt` (`:23`) with a "not built" reason.
- BR-15 not addressed: `summary` (`main.go:41-45`) still indexes an empty slice; unreachable today, one guard line.
- BR-16 not addressed: `probe_line`'s awk (`:145-150`) discards `$4` on the success path, so a truncated pipe run reads identically to a full one.
- BR-17 not addressed: `sample()`'s `awk $4` (`:85`) still truncates paths containing spaces, and `comm=` emits full paths where the plan's ARCH-SECURE said process name — paths carry usernames and workspace names into a report designed to leave the machine.
- BR-18 not addressed and now wider: `PAIR_PERF_WINDOW` **and** the new `PAIR_PERF_BUDGET` flow unvalidated into `sleep`, `awk -v` and `$(( ))`, and neither is documented in atlas or README.
- BR-19 not addressed: `4f9365b3` (M2.2b) is still inside the M1 window, along with `verdict`/`note_from_lines` — M2's base is not the branch point.
- BR-20 not addressed: `doctor/README.md` describes `doctor/`'s contents and still does not mention `perf.sh` (atlas/index.md was updated).
- BR-21 not addressed: issue Plan M1 (line 133) is unticked and `## Log` records no M1.4 (known-value verification) or M1.5 (budget) evidence.
- `doctor.lua:75-76` — `delta`'s docstring still advertises the pre-fix return shape; `unmeasured` is missing from it.
- `spawnRTT` (`main.go:100-112`) records failed runs' durations into `samples`, so `len(samples)` on the 5-field line counts attempts, not measurements. Harmless today because the caller degrades, worth a word in the comment.

## 5. Test coverage notes

`nvim/doctor_test.lua` and `cmd/pair-hoprtt/main_test.go` both pass (`all doctor.lua tests passed`; `ok github.com/xianxu/pair/cmd/pair-hoprtt 2.141s`), as does `make test-perf-capture`. Two of the three Critical fixes are revert-verified red. The gaps: (a) no test anywhere exercises a *failing* probe command — I had to verify BR-2 by hand (`pair-hoprtt -spawn 8 -- sh -c 'exit $(($$ % 2))'` → `4.072 4.617 4.617 8 4`, and `perf.sh` rendering `n/a (probe failed)`); (b) `perf_test.sh` cannot distinguish `key=` from `key=<value>`, which is why BR-5's regression survives a green suite; (c) `delta`'s fixtures are literals, not recorded samples, so nothing connects `perf.sh`'s output format to the consumer that will parse it; (d) `embed_test.go` does not pin `doctor/perf.sh`. Also note `parse_duration` accepts `1:2:3:4:5` → `13403045`, `0x10` → `16`, and `-1:00` → `-60` — no field-count or sign bound, which was the third part of BR-8's recommendation.

## 6. Architectural notes for upcoming work

- **ARCH-PURE — pass.** `delta`, `parse_duration`, `verdict`, `note_from_lines` are string/table-in, value-out and run under `nvim -l` with no IO; `summary` is a pure function unit-tested directly. The pure/shell split is exactly where the plan said it would be, and it is the reason the Critical join bug was fixable in one line with a test.
- **ARCH-ORDER — pass at the pure seam, gap at the collector.** `delta`'s `(state, event) → bucket` partition is now total and each interleaving is injectable in the test. But `perf.sh` runs `sample()` and `cputimes()` as **two separate `ps -A` passes** (`:84`, `:87`); under a spawn storm — the scenario the counters exist for — they diverge and those pids land in `unmeasured` rather than being measured. The silent drop is gone, but BR-3's other half (one `ps -Ao pid=,etime=,time=,rss=,comm=` pass) was not done, and it is a one-line change that removes the bucket entirely.
- **ARCH-DRY — flag.** `collect()` is dead code sitting beside seven inline, mutually inconsistent fallbacks; the `vm_stat` pipeline is duplicated. Both are the same shape: the shared helper exists (or should) and the call sites don't use it.
- **ARCH-MOCK — flag.** `perf_test.sh` runs the real script against the real machine, so its result depends on which system tools this shell can reach. In my environment `ps`/`top`/`sysctl` are denied and the suite still reported "passed" against an almost-empty report. There is no fixture path from `perf.sh` output to `delta`; M2 will need one anyway to write the parser.
- **ARCH-SECURE — flag (Minor).** `comm=` full paths and unvalidated `PAIR_PERF_*` are the two live items; both are already recorded as BR-17/BR-18.
- **ARCH-CONSTRAINTS — flag.** See BR-11: the envelope is declared hard and enforced at the cheapest 5% of the run.
- **ARCH-PURPOSE — flag.** The hand-maintained-list class was swept well and written down in `lessons.md`; the enumeration is one site short (`embed_test.go`), and the `unverified-repo-mechanism` class's install-layout member was never enumerated at all.

## 7. Plan revision recommendations

1. **M2.6 (plan line 387-391)** — still says the rolling-file write goes "via `artifactpath`", a Go internal package `nvim/doctor.lua` cannot call. Revise to `pair_data_dir()` (`nvim/init.lua:484`) + `pair_tag()`. This is BR-1's open half and the plan-gate's PQ-6, now unaddressed across two gates.
2. **Core concepts table (plan lines 74-81)** — add `verdict` / `FRAME_MS` (new pure entities that landed in this window and are in no row), and update `delta`'s documented return to include `unmeasured`.
3. **M1.1 (plan line 298) and issue Plan M1 (issue line 135)** — the plan sketch says `med < 0.5`, the issue says "1–4 ms, never 18", and the shipped test at `main_test.go:56` uses `0.05`. A 10x-lower floor weakens the positive control the milestone is named for. Either restore `0.5` or record a `## Revisions` entry saying why the floor moved and updating the issue line to match.
4. **ARCH-CONSTRAINTS (plan lines 173-178)** — either extend the deadline to the collectors, or revise to say the budget is *sized* for the collectors and *enforced* only on the probe tail, and say explicitly which probes are the ones worth dropping (only `zellij_action` costs anything).
5. **ARCH-MOCK (plan lines 264-268)** — either deliver the "recorded fixture pairs checked into the repo" or revise the claim to the literals that shipped.
6. **`$PAIR_HOME/bin/pair-hoprtt` (plan lines 105-110)** — the plan asserts this location and its degradation text; it is wrong for the shipped layout. Revise with the resolution order, and add the install-layout check to the mechanism rule.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Rename to pair-hoprtt landed and GO_BINS is correct; plan M2.6 line 387-391 still routes the Lua rolling-file write through artifactpath.
  - id: BR-2
    disposition: addressed
    note: |
      Verified live, not from the diff: zellij probe now renders n/a (probe failed) instead of a fabricated 10.183; partial failure emits the 5-field line. No regression test pins it — folded into BR-9.
  - id: BR-3
    disposition: addressed
    note: |
      Revert-verified red (unmeasured counted, not dropped). Residual: perf.sh still runs two separate ps passes (:84,:87), so storm pids land in unmeasured rather than being measured; one ps -Ao pid=,etime=,time=,rss=,comm= pass removes the bucket.
  - id: BR-4
    disposition: addressed
    note: |
      pair-hoprtt is untracked at HEAD (removed in 0a7e6e27). The 2.9MB blob remains in c290191c; the branch is still unpushed, so dropping it by rebase is cheap now and permanent after merge.
  - id: BR-5
    disposition: not-addressed
    note: |
      collect() is defined at perf.sh:32 with ZERO call sites. Re-ran the capture: load=, cpu_idle_pct=, process_count=0, host_cores=?, empty ## disk block. The fix reads as protection while doing nothing.
  - id: BR-6
    disposition: addressed
    note: |
      Revert-verified: re-adding `|| echo 0` to count_comm makes perf_test.sh fail with "report contains a bare value line with no key".
  - id: BR-7
    disposition: addressed
    note: |
      count_comm now takes the basename via awk -F/ and anchors the pattern. Not executable here (ps denied), reasoned from the comm= path format the fix itself documents.
  - id: BR-8
    disposition: addressed
    note: |
      Revert-verified red. Residual (Minor, not re-raised): no field-count or sign bound, so 1:2:3:4:5 -> 13403045, 0x10 -> 16, -1:00 -> -60.
  - id: BR-9
    disposition: not-addressed
    note: |
      perf_test.sh + make test-perf-capture exist and are wired into make test, but pin only line shape and the missing-binary path; nothing pins a failing probe or a failing collector, and delta is still literals rather than recorded fixture pairs.
  - id: BR-10
    disposition: not-addressed
    note: |
      explicitAssetPaths, both artifactpath lists, GO_BINS and the recipe are done and doctor/perf.sh is in the generated manifest. embed_test.go:25-26 was named and skipped, and the pair-hoprtt bundling decision was never made - see the new Critical.
  - id: BR-11
    disposition: not-addressed
    note: |
      over_budget() guards only probe_line (:142). top -l 2, sleep WINDOW and iostat -c 2 are unconditional and are the budget; the total is still unbounded and the skip drops the probes rather than the expensive collectors.
  - id: BR-12
    disposition: not-addressed
    note: |
      perf.sh:97,107 still emit at_ns from date +%s.
  - id: BR-13
    disposition: not-addressed
    note: |
      vm_stat awk pipeline still duplicated verbatim at perf.sh:93 and :119.
  - id: BR-14
    disposition: not-addressed
    note: |
      perf.sh:23 still yields /bin/pair-hoprtt when PAIR_HOME is unset; :161's reason still says "not built".
  - id: BR-15
    disposition: not-addressed
    note: |
      summary (main.go:41-45) still indexes an empty slice with no guard.
  - id: BR-16
    disposition: not-addressed
    note: |
      probe_line's awk still discards $4 on the success path.
  - id: BR-17
    disposition: not-addressed
    note: |
      sample() at perf.sh:85 still uses awk $4; comm= still emits full paths.
  - id: BR-18
    disposition: not-addressed
    note: |
      Now wider - PAIR_PERF_BUDGET joins PAIR_PERF_WINDOW, both unvalidated into sleep/awk -v/$(( )) and undocumented in atlas and README.
  - id: BR-19
    disposition: not-addressed
    note: |
      4f9365b3 is still in the window, and verdict/note_from_lines landed with it; no ## Log entry records the boundary.
  - id: BR-20
    disposition: not-addressed
    note: |
      doctor/README.md still does not mention perf.sh.
  - id: BR-21
    disposition: not-addressed
    note: |
      Issue Plan M1 (line 133) unticked; ## Log has no M1.4 or M1.5 evidence.
findings:
  - id: new
    severity: Critical
    family: unverified-repo-mechanism
    title: |
      perf.sh looks for its probe at $PAIR_HOME/bin/pair-hoprtt, a path that never exists in a shipped pair
    detail: |
      3rd finding in this family, so the deliverable is the RULE, not the site.
      Rule: a plan or diff may name a repo path or mechanism only alongside a
      check that it resolves in every layout the artifact ships to (dev
      checkout, make install, extracted runtime bundle) AND from the language
      and layer the calling code sits in. Prevalence 3/3 rounds that named a
      mechanism: PQ-1 failed the build path, BR-1 failed the language path,
      this fails the install-layout path. Evidence: the generated
      manifest.json's bin/ entries are exactly bin/lib/adapt-log.sh,
      bin/lib/dev-rebuild.sh, bin/pair-help, bin/pair-notify - no Go binaries
      since 104 M3. This window added doctor/perf.sh to the bundle, so the
      shell ships and its payload does not; every installed pair prints
      probes=n/a and is told to run make build, which does not apply to its
      layout. Sweep the enumeration the rule implies for every new runtime
      file: builds, in the bundle manifest, callers' path expressions resolve
      under all three layouts, a test pins each.
  - id: new
    severity: Important
    family: failure-reported-as-measurement
    title: |
      verdict() turns absent timings into 'fast', and its test asserts that direction
    detail: |
      6th finding in this family, so state the rule rather than patching the
      site. Rule: every value-producing function in the capture, shell and Lua
      alike, must render an absent or failed measurement as absence (nil, or
      n/a with a reason), never as an in-domain value. Prevalence 6/6: BR-2,
      BR-5, BR-14, BR-16, the dead collect() helper, and this. At
      doctor.lua:129 `tonumber(x) or 0` makes verdict(nil,nil)=='fast', and per
      the Spec (issue lines 77-81) 'fast' is what excludes the whole scheduling
      family (201/203) - so a report where nvim was never timed excludes two
      issues on a measurement that did not happen. doctor.lua:118-124 states
      the correct rule three functions above. doctor_test.lua:52 asserts the
      wrong direction, which is the same-mental-model test the gate warns
      about. Enumeration to sweep in one pass: every kv call site in perf.sh,
      every exported function in doctor.lua's 208 block, and report/summary in
      cmd/pair-hoprtt/main.go - each gets one absent-input case.
```

---

## Re-review — 2026-09-07T00:17:26-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..cd1315acd4ebec2845b09eca6dd789196c4fe23e |
| command | sdlc milestone-close --issue 208 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T00:17:26-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The architecture of this milestone is right and round 2 fixed most of what round 1 named — `collect()` is now actually called (I reverted `_proccount`'s guard and `doctor/perf_test.sh` went red), the bundle plumbing is complete and pinned by `embed_test.go`, and `delta` is driven from a real recorded capture through a real parser rather than literals. Three things block the boundary. **`make test` is red right now in this shell**: `doctor/perf_test.sh:51`'s bare `mktemp -d` ignores `$TMPDIR` on macOS, so `make test-perf-capture` exits 1 before running a single assertion — reproduced, and it is the gate the close depends on. **BR-22 is not actually fixed**: PATH-first resolution repairs the `make install` layout but not the shipped one (the Homebrew formula builds only `./cmd/pair-go` and `PAIR_HOME` is the extracted bundle, which carries no helper binaries), and I confirmed by reverting the fix in a scratch copy that the whole suite stays green — nothing pins it. **The budget sheds the wrong things**: with `PAIR_PERF_BUDGET=2` all three probes render `n/a (budget exceeded, probe skipped)` while `top -l 2` — the single most expensive collector — always runs, because it is checked before it starts and comes first. On a degraded machine, which is the only machine this tool is for, the capture drops precisely the three rows that have known baselines.

## 1. Strengths

- **The pure/IO split is honored, not just written down.** `doctor/perf.sh` really does emit raw samples and `nvim/doctor.lua:parse_samples` + `delta` do the join headlessly. `nvim/doctor_test.lua:145`'s `accounted == rows_a` is a genuine contract invariant — every pid lands in exactly one bucket — not a mock reasserting the implementation. ARCH-PURE: pass.
- **The positive control is real and the failure path is honest.** The fixture records `fork_exec_ms=1.588`, `pipe_hop_ms=0.006`, `zellij_action_ms=13.323` against the plan's ~1.5 ms / ~7 µs / ~13 ms bands — M1.4 is genuinely done. I confirmed `./bin/pair-hoprtt -spawn 3 -- /usr/bin/false` exits 1 rather than reporting excellent latency.
- **The n/a rule is pinned by a test that fails without the fix.** Reverting `_proccount`'s `awk 'END{if (NR==0) exit 1; ...}'` produced `FAIL process_count=0 is a fabricated value`. That is the standard the gate asks for, met.
- **Bundle plumbing swept all four hand-maintained lists** (`GO_BINS` + recipe, `artifactpath` inventory + generated-mirror classification, `explicitAssetPaths`, `embed_test.go`), and `go test ./cmd/internal/artifactpath ./cmd/internal/runtimebundle/...` passes against the regenerated manifest.
- **The fixture is under `doctor/`, not `nvim/`, with the reason recorded** (`nvim/doctor_test.lua`: the bundle walks `nvim/` wholesale). That is layout reasoning that prevents a future leak rather than discovering one.

## 2. Critical findings

**C1 — `doctor/perf_test.sh:51`: bare `mktemp -d` makes `make test` fail in an agent shell.** Reproduced: `make test-perf-capture` → `mktemp: mkdtemp failed on /var/folders/…: Operation not permitted`, `make: *** [test-perf-capture] Error 1`. macOS `mktemp` with no template uses `confstr(_CS_DARWIN_USER_TEMP_DIR)`, so `TMPDIR=… ` cannot redirect it; I verified `mktemp -d "${TMPDIR:-/tmp}/perfstub.XXXXXX"` works and that the suite is otherwise green with that one-line change. `set -e` makes the script die before any assertion runs, so the failure surfaces as a bare mktemp error with no `N failure(s)` line.

**C2 — BR-22 is not addressed: the probe still does not exist in the shipped layout, and no test pins the fix.** See the disposition below.

## 3. Important findings

**I1 — `doctor/perf.sh`: the deadline sheds the probes first, so the target condition produces the least useful report.** `PAIR_PERF_BUDGET=2 sh doctor/perf.sh` yields `disk=n/a`, `pipe_hop_ms=n/a`, `fork_exec_ms=n/a`, `zellij_action_ms=n/a`. The healthy fixture already runs `elapsed_seconds=5` of a 6 s budget, so on a machine slow enough to be worth capturing, all three probes are skipped by construction. Meanwhile `top -l 2 -n 60` (~2 s, unbounded on a loaded host) is checked *before* it starts and therefore always runs. **This is the 2nd finding in family `unenforced-operating-envelope`** — do not just move one check. The rule: *a capture's budget must reserve capacity for the measurements the issue exists to take, and shed contextual collectors before diagnostic ones; sequential position is not a priority order.* Concretely, run the three probes before `top`/`iostat`, or give them a reserved slice of the budget. Secondary consequence: `doctor/perf_test.sh:36`'s `[ elapsed -le budget ]` will go red under exactly the load the tool targets, so `make test` becomes machine-load dependent.

**I2 — BR-5 residual: `sample()`/`cputimes()` bypass `collect()` and their fallback is dead code.** See the disposition below.

**I3 — BR-23 residual: the enumeration was named but only a third of it was swept.** See the disposition below.

## 4. Minor findings

- `doctor/perf_test.sh:23` — the stray-line check's `*[!0-9]*) continue` arm skips every line containing a non-digit, so only all-digit strays are caught. I injected `say "an unattributable stray line"` into a scratch `perf.sh` and the suite passed; a bare `0` is correctly caught.
- `doctor/perf.sh:163` — the `swap_rate` awk is arithmetic in shell, which the file's own rule 2 (line 13: "does no arithmetic") says does not happen; it also divides by `$WINDOW` with no zero guard and produces negative rates if a counter goes backwards.
- `doctor/perf.sh:157` (implicit) — a `vm_stat` that exists but exits non-zero renders `swap=n/a (vm_stat unavailable)`; the tool was available, it failed. Reproduced with a stub.
- `doctor/perf.sh` — three hand-rolled copies of `collect()`'s availability/failure/empty ladder (the `top` block ~:88, `disk` :171, `probe_line` :202). ARCH-DRY: `collect()` exists precisely to be that ladder; it just cannot emit multiple keys. A `stage()` variant would collapse all three — and is the reason the sample blocks escaped the rule at all (I2).
- `nvim/doctor.lua:157` — the comment says "The fixture in `nvim/fixtures/`"; it is in `doctor/fixtures/`, and `doctor_test.lua` explains why. `nvim/doctor.lua:75`'s "Returns { … }" list omits `unmeasured`, which the code sets and the test asserts.
- `nvim/doctor.lua:95` — reuse is detected only via `etime`. If `pa.etime` is nil (unparseable), a reused pid falls through to the rate branch and `cb - ca` yields a negative `cpu_pct`. A `cb < ca` check is the cheaper and stronger signal.
- `doctor/fixtures/perf_capture.txt` is a real capture from the workbench committed to a public repo. It happens to carry only system daemons because it was truncated to the first 60 pids; nothing records that as the rule, so the next re-capture ships `/Users/<name>/workspace/<repo>/…` paths.

## 5. Test coverage notes

- What is well pinned: the collector `n/a` rule (revert-verified red), the `grep -c` stray-`0` bug, the probe-all-failed exit, the `/usr/bin/true` positive control, `delta`'s bucket-completeness invariant against a real capture, and `doctor/perf.sh`'s presence in the bundle.
- What is not pinned: **the C2 fix** (revert-verified green), the probe being *found* — `perf_test.sh` asserts nine keys, none of them `pipe_hop_ms`, so only the degraded path is covered; `parse_samples` has no absent/malformed-input case; `summary([])` and `report(nil, 0)` panic and nothing exercises them; the empty `### cputime`/`### procs` blocks are not asserted by the tool-denial loop, which enumerates only five `kv` keys.
- The denial fixture (`stub/{ps,top,sysctl,vm_stat,iostat}` → `exit 1`) is a good stateful-enough double for ARCH-MOCK purposes and production and test flow do share the boundary. ARCH-MOCK: pass, with the caveat that the *found-probe* half of the seam has no assertion.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag (Minor, above): four shapes of the same failure ladder in one 228-line file.
- **ARCH-PURE** — pass. The join is where the plan promised, and it is tested without IO.
- **ARCH-PURPOSE** — flag (I1, and the C2/I3 dispositions). Three of this round's findings fix the site a prior finding named while enumerable siblings of the same class stay in the tree. The `unverified-repo-mechanism` family is now 4/4 rounds; the enumeration BR-22 demanded should be written down once — *dev checkout, `make install` PATH, extracted runtime bundle, Homebrew formula, sandboxed agent shell* — and every new runtime file checked against all five, with a test per layout.
- **ARCH-MOCK** — pass (see above).
- **ARCH-CONSTRAINTS** — flag (I1). The envelope is declared "hard" and is a soft per-stage deadline: a stage that starts under budget runs to completion unbounded, so a slow `top` alone can carry the capture past 6 s.
- **ARCH-SECURE** — pass with a note. No `env` dump, no argv, and the fixture carries no home paths. `sample()` emits full executable paths rather than the planned process name; combined with the committed fixture that is the leak surface to write a rule for.
- **ARCH-ORDER** — pass for `delta`: the three-plus-one case enumeration is real, counted, and tested. For M2, the interesting ordering is the async in-flight span, and the plan's table already names the right event (consume only if the buffer still matches). Note that `delta`'s `window` argument has no relationship to the emitted `at_s` values — M2 should pass `at_s(b) - at_s(a)`, not the requested `window_seconds`, or the rates are wrong exactly when the machine is slow enough to stretch the sleep.

## 7. Plan revision recommendations

The plan file was not touched in this window and now contradicts the code in three places. Add one `## Revisions` section (the plan has none, and AGENTS.md requires appending rather than overwriting):

1. **Probe location.** Lines 107–108 still say "`perf.sh` locates it as `$PAIR_HOME/bin/pair-hoprtt`" and prints "`n/a (hoprtt not built — run make build)`". The code resolves PATH-first with a checkout fallback and prints different text. Record the layout enumeration and which layouts are supported.
2. **Budget.** The ARCH-CONSTRAINTS block says "≤ 6 s wall clock, hard … Enforced, not hoped". What shipped is a per-stage soft deadline with no per-command timeout. Either say "sized, with per-stage shedding" or specify the reservation I1 asks for.
3. **M2.6's mechanism.** Line 389 still routes the Lua rolling-file write "via `artifactpath`" — a Go internal package with no CLI surface (verified: every caller is under `cmd/`). Name `pair_data_dir()` (`nvim/init.lua:484`) + `pair_tag()` (`:480`) instead, or specify the CLI subcommand that would expose it.
4. Optionally note that `perf.sh` does compute the swap rate, so the "does no arithmetic" invariant is "does no *per-process* arithmetic".

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The naming half landed pre-window (df283abe); the plan file was not touched in this window and M2.6:389 still names artifactpath, which has no CLI surface — every caller is Go under cmd/.
  - id: BR-5
    disposition: not-addressed
    note: |
      The kv collectors are fixed and revert-verified, but sample()/cputimes() (perf.sh:121-122) bypass collect() and their `|| say "n/a (ps unavailable)"` at :137/:139 is dead — the pipeline exits with awk's status, so a failed ps yields silently empty blocks.
  - id: BR-9
    disposition: addressed
    note: |
      perf_test.sh + test-perf-capture wired into `make test`, and parse_samples now pins the perf.sh -> delta contract against a real recorded capture.
  - id: BR-10
    disposition: addressed
    note: |
      generate.go, embed_test.go and the artifactpath manifest all carry doctor/perf.sh; the regenerated manifest.json contains it and the bundle tests pass.
  - id: BR-11
    disposition: addressed
    note: |
      A stage-level deadline with skip reasons and an emitted elapsed_seconds now exists; the shed ORDER it produces is raised separately.
  - id: BR-12
    disposition: addressed
  - id: BR-13
    disposition: addressed
  - id: BR-14
    disposition: addressed
  - id: BR-15
    disposition: not-addressed
    note: |
      cmd/pair-hoprtt/main.go:41 summary() still indexes samples[0] with no length guard; still unreachable from main, still one line from being safe.
  - id: BR-16
    disposition: not-addressed
    note: |
      probe_line's awk (perf.sh:206) still prints only $1 and $2; the sample count $4 is discarded, so a pipeRTT that broke out early reads identically to a full 500-sample run.
  - id: BR-17
    disposition: not-addressed
    note: |
      perf.sh:121 still takes awk $4 of `comm=`, truncating any executable path containing a space, and still emits full paths where the plan said process name.
  - id: BR-18
    disposition: not-addressed
    note: |
      PAIR_PERF_WINDOW and PAIR_PERF_BUDGET still flow unvalidated into sleep, `[ -ge ]` and `awk -v`, and neither appears in atlas/index.md or doctor/README.md.
  - id: BR-19
    disposition: not-addressed
    note: |
      4f9365b3 is still inside the M1 window along with verdict/note_from_lines; no Log entry or plan note records that M2's base is not the branch point.
  - id: BR-20
    disposition: not-addressed
    note: |
      doctor/README.md is not in this window's diff and still describes only the drift half; the atlas entry landed but the README a reader of doctor/ opens did not.
  - id: BR-21
    disposition: not-addressed
    note: |
      Plan M1 is still unticked and the Log still ends at 2026-09-06; the M1.4/M1.5 numbers now exist in the fixture (pipe 0.006ms, fork 1.588ms, zellij 13.323ms, elapsed 5) but are not recorded in the issue.
  - id: BR-22
    disposition: not-addressed
    note: |
      PATH-first fixes `make install` only. ../homebrew-pair/Formula/pair.rb builds solely ./cmd/pair-go and installs bin/nvim/zellij, and PAIR_HOME at runtime is the extracted bundle root, which carries no helper binaries — so a shipped pair still prints probes=n/a with advice that does not apply to its layout. I reverted the PATH-first block in a scratch copy and the whole suite stayed green, so no test pins the fix.
  - id: BR-23
    disposition: not-addressed
    note: |
      verdict(nil,nil) is fixed, but verdict(nil, 2) still returns 'fast' and doctor_test.lua:56 asserts that direction — a half-absent measurement rendered as a full in-domain verdict is the same defect at a different arity. Of the enumeration BR-23 named, only doctor.lua's exports were swept: perf.sh's sample blocks still fabricate absence (see BR-5), parse_samples has no absent-input case, and summary/report in cmd/pair-hoprtt still panic on empty input.
findings:
  - id: new
    severity: Critical
    family: unverified-repo-mechanism
    title: |
      doctor/perf_test.sh:51's bare `mktemp -d` makes `make test-perf-capture`, and therefore `make test`, fail in a sandboxed agent shell
    detail: |
      This is the 4th finding in family `unverified-repo-mechanism`, so the rule
      is the deliverable, not the site. Rule: a mechanism this repo names must be
      checked in EVERY environment and layout it will run in, enumerated in one
      place — dev checkout, `make install` PATH, extracted runtime bundle,
      Homebrew formula install, and the sandboxed agent shell that runs `make
      test` before every close. Prevalence 4/4 rounds that named a mechanism:
      PQ-1 failed the build path, BR-1 the language path, BR-22 the install-layout
      path, this the execution-environment path.
      Reproduced: `make test-perf-capture` exits 1 with `mktemp: mkdtemp failed
      on /var/folders/...: Operation not permitted`, before any assertion runs,
      because macOS mktemp with no template reads confstr(_CS_DARWIN_USER_TEMP_DIR)
      and cannot be redirected with TMPDIR. Verified fix and green suite with
      `mktemp -d "${TMPDIR:-/tmp}/perfstub.XXXXXX"`. Note this is the same file
      whose header advertises that it is "routinely run by sandboxed agents".
  - id: new
    severity: Important
    family: unenforced-operating-envelope
    title: |
      The budget sheds the three probes first, so a degraded machine — the only condition this tool targets — yields a capture with no probe rows
    detail: |
      This is the 2nd finding in family `unenforced-operating-envelope`. Rule,
      not site: a capture's budget must reserve capacity for the measurements the
      issue exists to take and shed contextual collectors before diagnostic ones;
      sequential position in the script is not a priority order. Reproduced with
      `PAIR_PERF_BUDGET=2 sh doctor/perf.sh`: disk, pipe_hop, fork_exec and
      zellij_action all render `n/a (budget exceeded...)` while `top -l 2 -n 60`
      — the single most expensive collector, and unbounded on a loaded host — is
      checked before it starts and therefore always runs. The healthy fixture
      already reports elapsed_seconds=5 of 6. The probes are the only rows with
      published baselines and are Done-when item 1. Fix: order the probes ahead
      of top/iostat, or reserve a probe slice of the budget. Secondary: this makes
      perf_test.sh:36's `[ elapsed -le budget ]` load-dependent, so `make test`
      goes red under exactly the conditions the capture is for.
  - id: new
    severity: Minor
    family: untested-shell-surface
    title: |
      perf_test.sh:23's stray-line check only fires for all-digit lines, so any other unattributable line passes
    detail: |
      The `*[!0-9]*) continue` arm (there to skip tab-separated ps rows) swallows
      every stray line containing a non-digit. Verified: injecting `say "an
      unattributable stray line"` into a scratch perf.sh left the suite green; a
      bare `0` is correctly caught. Match the ps rows positively (a line with a
      tab, inside a `### ` block) instead of negatively by character class.
  - id: new
    severity: Minor
    family: duplicated-logic
    title: |
      Four shapes of collect()'s availability/failure/empty ladder in one file, which is how the sample blocks escaped the rule
    detail: |
      collect() (perf.sh:43) exists to be that ladder, but it can only emit one
      key, so the top block (~:88), the disk block (:171) and probe_line (:202)
      each re-implement it and sample()/cputimes() (:121-122) skip it entirely.
      A `stage()` variant that takes a multi-key emitter would collapse all four
      and is the structural fix behind the BR-5 residual.
  - id: new
    severity: Minor
    family: report-line-contract
    title: |
      perf.sh:163's swap_rate is shell arithmetic, contradicting the file's own rule 2, and divides by WINDOW unguarded
    detail: |
      This is the 5th finding in family `report-line-contract`. Rule: every value
      the report emits either comes from a tool verbatim or is computed in tested
      Lua; perf.sh's header rule 2 (line 13) states this and the swap block
      violates it. Same block yields inf at WINDOW=0 and negative rates if a
      counter goes backwards. Either move it into doctor.delta with the pid join
      or amend rule 2 to say "no per-process arithmetic".
  - id: new
    severity: Minor
    family: failure-reported-as-measurement
    title: |
      A vm_stat that exists but exits non-zero renders `swap=n/a (vm_stat unavailable)`, misnaming the failure
    detail: |
      This is the 7th finding in family `failure-reported-as-measurement`. The
      rule the family needs, stated once: every emitter distinguishes THREE
      outcomes — tool absent, tool failed, tool returned nothing — and names which
      one it hit, because "unavailable" sends a reader to install something that
      is already installed. collect() already does this correctly; the hand-rolled
      swap block at perf.sh:155-157 does not. Reproduced with a `vm_stat` stub
      that exits 1.
  - id: new
    severity: Minor
    family: docs-gate
    title: |
      Comment drift in nvim/doctor.lua — the fixture path and delta's documented return shape are both wrong
    detail: |
      This is the 2nd finding in family `docs-gate`. Rule: a comment that names a
      path or a return shape is a claim a test can check; when it names a
      filesystem path, assert it, and when it enumerates fields, enumerate all of
      them. doctor.lua:157 says "The fixture in nvim/fixtures/" (it is in
      doctor/fixtures/, and doctor_test.lua explains why it must not be under
      nvim/); doctor.lua:75's "Returns { rates, vanished, started, reused, rows_a,
      rows_b }" omits `unmeasured`, which the code sets and the test asserts.
  - id: new
    severity: Minor
    family: unguarded-edge-case
    title: |
      delta detects a reused pid only through etime, so an unparseable etime lets a reused pid produce a negative cpu_pct
    detail: |
      This is the 3rd finding in family `unguarded-edge-case`. Rule: a derived
      value with a known-impossible range must be rejected at the point of
      derivation, not only by the proxy signal that usually implies it. At
      doctor.lua:95, if pa.etime or pb.etime is nil the reuse branch is skipped
      and the pid falls into the rate branch, where cb < ca yields a negative
      rate. `cb < ca` is itself sufficient evidence of reuse and is cheaper than
      the etime comparison; the fixture test's `r.cpu_pct >= 0` assertion would
      then be enforced by the code rather than by the fixture's luck.
  - id: new
    severity: Minor
    family: recorded-fixture-redaction
    title: |
      doctor/fixtures/perf_capture.txt is a real host capture in a public repo, safe only by an unrecorded truncation accident
    detail: |
      The fixture carries zero `/Users/` paths only because it was cut to the
      first 60 pids, which on this host are all system daemons. Nothing records
      that as the selection rule, and perf.sh emits full executable paths, so the
      next re-capture (a wider window, a busier machine) will commit
      home-directory and project paths. Write the redaction/selection rule next
      to the fixture, or filter comm to its basename in sample().
```

---

## Re-review — 2026-09-07T00:37:16-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..dc5a3d039a41ab339ca64be59fad299de303decf |
| command | sdlc milestone-close --issue 208 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T00:37:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The probe half of M1 is genuinely good and I verified it end to end: `pair hoprtt` reads 0.006 ms for a pipe hop and 1.5–1.7 ms for `/usr/bin/true` (M1.4's known quantities, both in band), the positive control is in the suite, and turning the probe into a `pair` subcommand is the right answer to BR-22 — pinned by a test that builds the real binary and would go red on a revert. The budget now sheds contextual collectors before probes, reproduced with `PAIR_PERF_BUDGET=2`. What blocks a clean SHIP is a cluster of gaps I reproduced rather than inferred: the two `### cputime`/`### procs` sample blocks still render a failed `ps` as *empty* rather than `n/a` (BR-5's residual — the pipeline's exit status is awk's, so the `|| say "n/a"` never fires), nothing in any test exercises those rows' grammar (I changed the separator from tab to pipe in a scratch copy and every test stayed green), the BR-25 shed-order fix has no test at all so a revert is invisible, and the durable plan still describes `cmd/pair-hoprtt/main.go` + a `GO_BINS` entry at eight places — including M1.2b and the `$PAIR_HOME/bin/pair-hoprtt` path that BR-22 called Critical — with no `## Revisions` entry, which is a live trap for whoever executes M2 from it.

## 1. Strengths

- **`cmd/internal/hoprttcmd/hoprtt.go:16-21` — the subcommand move is the right fix, not the cheap one.** BR-22 could have been patched with a better path expression; instead the probe became part of the one binary that always ships. `hoprtt_test.go:47` pins it by building `./cmd/pair-go` and invoking through it, which is what makes the fix non-revertible. This is ARCH-PURPOSE done properly — the class, not the site.
- **`doctor/perf.sh:23-31` + `:43-46` — the shed order is now a stated design with a reserve, and it works.** `PAIR_PERF_BUDGET=2 sh doctor/perf.sh` yields `cpu_idle_pct=n/a (budget reserved for probes; top skipped)`, `disk=n/a (…iostat skipped)`, `sample_b skipped=…` while `pipe_hop_ms=0.006` and `fork_exec_ms=1.489` both survive. Exactly the inversion BR-25 asked for.
- **`doctor/perf.sh:73-82` — `_countmatching` records *why* it is awk and not `grep -c`.** Two review-found bugs (stray bare `0`; a failed `ps` fabricating `0`) are explained in place, and `ps`'s exit status is checked separately so it can propagate. I confirmed `pair_family_procs=n/a (ps failed)` under real denial.
- **`nvim/doctor.lua:118-124` and `M.verdict` — the asymmetry is correctly reasoned and correctly tested.** `verdict(nil, 99) == 'slow'` but `verdict(nil, 2) == 'unknown'`; partial evidence proves slow and never clears the editor. `doctor_test.lua` asserts both arities, and the old `tonumber(x) or 0` would fail them.
- **`cmd/internal/hoprttcmd/hoprtt.go:110-129` — `spawnRTT` counting failures and `Run` exiting 1 when all invocations fail.** Verified: `pair hoprtt -spawn 3 -- /usr/bin/false` exits non-zero, and `perf.sh` renders `zellij_action_ms=n/a (probe failed)` here rather than a fast-looking number.

## 2. Critical findings

**`workshop/plans/000208-pairdoctor-perf-capture-plan.md:79,98-108,283,309-313,402` — the plan still specifies the design round 3 deliberately reversed, with no `## Revisions`.**

The Core concepts table (`:79`) says `pair-hoprtt` lives in `cmd/pair-hoprtt/main.go`, status new; the code is `cmd/internal/hoprttcmd/hoprtt.go`, a subcommand, and `cmd/pair-hoprtt/` was deleted in `c8f2d044`. `:107` still says "`perf.sh` locates it as `$PAIR_HOME/bin/pair-hoprtt`" — the exact expression BR-22 called Critical. `:311` (M1.2b) still instructs adding it to `GO_BINS` and verifying `bin/pair-hoprtt` exists. `:389` (M2.6) still routes the Lua rolling-file write through `artifactpath`, which the plan gate disposed `not-addressed` twice. The plan's last commit is `df283abe` — the review base — so nothing in this window touched it.

This is a document fix, but it is not cosmetic: M2 is executed *from* this plan, and as written it re-introduces the finding this milestone spent a round removing.

Fix sketch: append a `## Revisions` entry (timestamp + reason + delta) recording the subcommand move and its rationale, then update `:79` to `cmd/internal/hoprttcmd/hoprtt.go`, rewrite `:98-108` and `:283`, replace M1.2b with the dispatcher/`runStreamingSubcommand` wiring that actually landed, and settle M2.6's path source (`pair_data_dir()` at `nvim/init.lua:484` + `pair_tag()`) in the same pass.

## 3. Important findings

**a. `doctor/perf.sh:130-131,141-149` — a failed `ps` renders the sample blocks as *empty*, not `n/a`.** Reproduced twice in this shell (where `ps` is denied) and again under `perf_test.sh`'s own stub PATH: `### cputime` and `### procs` are both empty under `## sample_a` and `## sample_b`. The cause is that `cputimes()`/`sample()` are pipelines, so `ps`'s failure is masked by awk's exit 0 and `|| say "n/a (ps unavailable)"` never runs. Downstream this is worse than blank: `parse_samples` returns two present-but-empty samples and `delta` reports rows_a=0/rows_b=0/vanished=0 — a confident "nothing is running". `perf_test.sh`'s denial loop only checks five scalar keys, so it is green.

**b. `doctor/perf.sh` — the 6 s budget is gated between stages but no stage is bounded, and the healthy-path headroom is ~1 s.** This is the **3rd finding in family `unenforced-operating-envelope`** (BR-11: no deadline at all; BR-25: wrong shed order). Fixing another instance will not hold; the rule is: *a wall-clock budget is enforced only when every operation that can exceed it is itself bounded — a check before a stage starts does not bound the stage.* Evidence: `top -l 2 -n 60` (`:108`), `iostat -d -w 1 -c 2` (`:183`), `sleep "$WINDOW"` and every `probe_line` invocation run to completion once entered. The committed fixture reads `elapsed_seconds=5` against `budget_seconds=6`, so there is one second of slack — and the operator's own symptom quoted in the issue is *"top took 10 seconds"*. `pipeRTT(500)` (`hoprtt.go:178`) is likewise sized for a healthy host: at the ~10 ms/hop the issue says visible lag would require, that single probe is 5 s. Because `perf_test.sh:36` asserts `elapsed <= budget`, `make test` goes red under exactly the conditions the capture exists for. Enumeration the rule implies: wrap each external stage in a bounded runner (a `timeout`-equivalent or a per-stage sample budget derived from remaining time), and size `pipeRTT`'s count from the observed first-sample latency rather than a constant.

**c. `nvim/doctor.lua:157-186` (`parse_samples`) — a budget-shed `## sample_b` becomes an empty-but-present sample, so `delta` reports every process as vanished.** This is the **8th finding in family `failure-reported-as-measurement`**. The rule was already stated at BR-23 *and* its sweep enumeration was written ("every exported function in doctor.lua's 208 block"); this is a member of that enumeration the hand-sweep missed, which is the family reporting that a hand-sweep is not the right instrument. Reproduced with a synthetic capture containing `## sample_b` / `skipped=budget reserved for probes`: `rates=0 vanished=2 started=0 rows_a=2 rows_b=0`. On a real host that is `vanished=856` — a fabricated spawn-storm claim from a measurement that never happened, in the one condition (a squeezed capture on a degraded machine) the tool exists for. Deliverable at this prevalence is mechanical, not another sweep: one table-driven test in `doctor_test.lua` that iterates the exported functions of the `#208` block and asserts each maps absent/failed input to absence, so the next function added is covered by construction. The site fix is `parse_samples` returning `nil` for a block carrying `skipped=` or no `###` sections.

**d. `doctor/perf_test.sh` — nothing tests the sample-row grammar, so the "contract" `doctor_test.lua` claims to pin does not exist.** This is the **3rd finding in family `untested-shell-surface`**. Rule: *when two components are joined by a recorded fixture, the recording pins only the consumer; the producer needs its own live-run assertion, or drift on the producer side is invisible.* `doctor_test.lua:…` states "the perf.sh → delta contract is pinned. A change to either side that breaks the other now fails here" — that claim is false. Demonstrated: in a scratch copy I changed `sample()`'s emitter from tab-separated to pipe-separated; `sh perf_test.sh` printed "perf.sh shape tests passed" and exited 0, and `doctor_test.lua` reads a static file so it never sees `perf.sh` at all. `perf_test.sh`'s required-key list also omits every probe key, so the probes could vanish entirely and stay green — which is why finding (b) below and BR-25 have no failing test. Fix: assert a live run's `### procs` rows against `^[0-9]+\t\S+\t[0-9]+\t.+$` and `### cputime` against `^[0-9]+\t\S+$`, and add `pipe_hop_ms` / `fork_exec_ms` to the key list.

## 4. Minor findings

- `doctor/perf_test.sh:54` — the `mktemp` fallback writes `$repo/.perf-test-stub.$$` into the worktree, and that path is not in `.gitignore`. In this agent shell bare `mktemp -d` fails ("Operation not permitted"), so the fallback is taken on *every* run; an interrupted suite leaves a directory of fake `ps`/`top` executables at the repo root. **2nd in family `build-artifact-committed`** — rule: *any path a build or test writes into the worktree is gitignored in the same change that introduces the write.*

## 5. Test coverage notes

- The positive control (`hoprtt_test.go:64`) and the mode-divergence test (`:77`) are the two that matter, and both are real: they exercise the shipped binary, not the package function.
- `nvim/doctor_test.lua` is strong on `delta`'s four join cases and on the fixture's "every pid lands in exactly one bucket" invariant.
- Gaps, in order of what they would have caught: no test for a budget squeeze (BR-25's fix is unpinned); no test for the sample-row grammar (finding d); no test for `parse_samples` against a real `perf.sh` run rather than a recording; `TestSummaryOnEmptyInputDoesNotPanic` asserts `med == 0`, which is the in-domain-value shape BR-23's rule rejects — harmless today because `summary(nil)` is unreachable from `Run`, but the assertion teaches the wrong contract.
- Suite state: `nvim -l nvim/doctor_test.lua` passes; `go test ./cmd/internal/hoprttcmd` passes; `make test-perf-capture` passes. `go test ./...` fails in 10 packages, all `operation not permitted` from `ps`/pty in this shell — pre-existing environment denial, unrelated to this diff.

## 6. Architectural notes

- **ARCH-DRY — flag.** `collect()`'s availability/failure/empty ladder is re-implemented at `:101-120` (top), `:178-189` (disk) and `:211-222` (probe_line), and skipped entirely by `sample()`/`cputimes()`. BR-27 named this; it is also the structural cause of finding (a).
- **ARCH-PURE — pass.** The join lives in pure Lua and runs under `nvim -l`; `perf.sh` emits raw samples and does no per-process arithmetic. The one exception is the swap block (BR-28).
- **ARCH-PURPOSE — flag.** M1's purpose is delivered in code, but the plan that carries the purpose into M2 still describes the reversed design (Critical above), and M1.4/M1.5's verification exists only in this review's transcript, not in the issue Log.
- **ARCH-MOCK — pass with a note.** The failing-tool stub PATH in `perf_test.sh` is a real seam and exercises the degradation path; the recorded fixture is the right conformance artifact. What is missing is drift detection *on the producer* (finding d) — the fake models failure but nothing models success.
- **ARCH-CONSTRAINTS — flag.** See Important (b): the declared envelope is not enforced, and the implementation's constants are sized for the machine this tool is not for.
- **ARCH-SECURE — flag.** The report is designed to leave the machine, and `sample()` emits `comm` as a full executable path (`/Users/…/workspace/…`), where the plan's own ARCH-SECURE section promised "the process name and pid". BR-17 records this at Minor; under this lens it deserves Important, and BR-32 (the fixture is clean only because a 60-pid truncation happened to catch only system daemons) is the same rule at the fixture. `PAIR_PERF_WINDOW`/`BUDGET`/`PROBE_RESERVE` reach `sleep`, `[`, `$(( ))` and `awk -v` unvalidated: `PAIR_PERF_BUDGET=abc` prints `[: abc: integer expression expected` three times and still emits a report; `PAIR_PERF_WINDOW='1; echo pwned'` is not injectable (correctly quoted) but yields `window_seconds=1; echo pwned` and swap rates divided by a window that never elapsed.
- **ARCH-ORDER — pass on the probe, flag on the capture.** `pipeRTT`'s child is lexically bounded by its `defer`. The capture's own ordering is not: no stage is cancellable once entered (b), and the newly-reachable "sample_b was never taken" state is not represented — it is encoded as an empty sample that downstream code reads as a measurement (c).

## 7. Plan revision recommendations

The plan needs one `## Revisions` entry covering all of it:

> **2026-09-07 — the probe is a `pair` subcommand, not a binary.** Reason: BR-22 — a standalone `pair-hoprtt` exists only after `make install`; the Homebrew formula builds only `./cmd/pair-go` and `PAIR_HOME` at runtime is the extracted bundle root, which has carried no helper binaries since #104 M3, so every installed pair would have printed `probes=n/a`. Delta: Core concepts `:79` → `cmd/internal/hoprttcmd/hoprtt.go`; `:98-108` drop the `GO_BINS`/`$PAIR_HOME/bin/pair-hoprtt` rationale and replace with dispatcher registration (`dispatcher.Families`) + `runStreamingSubcommand`; `:283` files list → `cmd/internal/hoprttcmd/{hoprtt.go,hoprtt_test.go}`; M1.2b `:311-313` → "register the family and route it in `cmd/pair-go/main.go`; verify `bin/pair hoprtt` runs", the resolution order in `perf.sh` being PATH-first then `$PAIR_HOME/bin/pair`.

And a second entry, or an amendment to M2.6 `:389`: `artifactpath` is `cmd/internal/*` and unreachable from Lua — name `pair_data_dir()` (`nvim/init.lua:484`) + `pair_tag()` as the path source, closing BR-1 / PQ-6 at its third disposition.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan untouched since the base commit; :389 still routes the Lua write through artifactpath and :79/:98-108/:311 still describe the reversed cmd/pair-hoprtt + GO_BINS design.
  - id: BR-5
    disposition: not-addressed
    note: |
      Residual is the two sample blocks; reproduced live and under perf_test.sh's own stub PATH — `### cputime`/`### procs` render empty because the pipeline's exit status is awk's, not ps's.
  - id: BR-15
    disposition: addressed
    note: |
      Guard at hoprtt.go:53 plus TestSummaryOnEmptyInputDoesNotPanic; note the test asserts med==0, an in-domain value, which teaches the wrong contract even though the path is unreachable from Run.
  - id: BR-16
    disposition: not-addressed
    note: |
      probe_line still prints only $1/$2; the sample count $4 is used solely in the failure branch, so a pipeRTT that broke early still reads as a full run.
  - id: BR-17
    disposition: not-addressed
    note: |
      sample() at perf.sh:130 still takes awk $4 and comm is still a full path; under ARCH-SECURE this is worth more than Minor because the report is designed to leave the machine.
  - id: BR-18
    disposition: not-addressed
    note: |
      Widened, not fixed — PAIR_PERF_BUDGET=abc emits three `[: integer expression expected` errors and still reports; PAIR_PERF_WINDOW='1; echo pwned' yields window_seconds=1; echo pwned. Still undocumented in atlas/README.
  - id: BR-19
    disposition: not-addressed
    note: |
      Nothing in the window records that 4f9365b3 (M2.2b, the pure join) landed inside the M1 boundary.
  - id: BR-20
    disposition: not-addressed
    note: |
      doctor/README.md is untouched in this window and README.md:607 points readers there for the doctor surface.
  - id: BR-21
    disposition: not-addressed
    note: |
      All M1 checkboxes remain `- [ ]` and the Log has no 2026-09-07 entry. Numbers now available to record: pipe 0.006 ms, fork+exec 1.5-1.66 ms, perf.sh 2.26 s of a 6 s budget; zellij unmeasurable in this shell.
  - id: BR-22
    disposition: addressed
    note: |
      Subcommand + dispatcher registration, pinned by TestProbeIsReachableThroughTheShippedPairBinary which builds ./cmd/pair-go; verified `./bin/pair hoprtt` returns 0.006 ms.
  - id: BR-23
    disposition: addressed
    note: |
      verdict is now asymmetric and doctor_test.lua fails without it; the sweep the finding demanded is incomplete — see the new parse_samples finding.
  - id: BR-24
    disposition: addressed
    note: |
      mktemp fallback at perf_test.sh:54; bare `mktemp -d` still fails in this shell, so the fallback is the live path and `make test-perf-capture` passes.
  - id: BR-25
    disposition: not-addressed
    note: |
      Behaviour verified fixed (PAIR_PERF_BUDGET=2 keeps pipe_hop and fork_exec while shedding top/iostat/sample_b), but NO test exercises a squeeze and the probe keys are absent from perf_test.sh's required-key list, so reverting the shed order stays green. Also the secondary is untouched: elapsed can still exceed budget because no stage is bounded.
  - id: BR-26
    disposition: not-addressed
    note: |
      perf_test.sh:23 still `*[!0-9]*) continue`.
  - id: BR-27
    disposition: not-addressed
    note: |
      Still four shapes — collect() :51, the top block :101, disk :178, probe_line :211 — with sample()/cputimes() skipping the ladder entirely, which is the structural cause of BR-5's residual.
  - id: BR-28
    disposition: not-addressed
    note: |
      Still awk arithmetic in perf.sh and still divides by WINDOW unguarded; worse now, because when the budget sheds sample_b the `sleep "$WINDOW"` never runs yet the rate is still divided by WINDOW, so the divisor names a window that did not occur.
  - id: BR-29
    disposition: not-addressed
    note: |
      perf.sh:165-166 still says "n/a (vm_stat unavailable)" for a vm_stat that exists and failed.
  - id: BR-30
    disposition: not-addressed
    note: |
      doctor.lua still says "The fixture in nvim/fixtures/" (it is doctor/fixtures/) and delta's Returns list still omits `unmeasured`, which the code sets and the test asserts.
  - id: BR-31
    disposition: not-addressed
    note: |
      delta still detects reuse only via etime; an unparseable etime drops the pid into the rate branch where cb < ca yields a negative cpu_pct.
  - id: BR-32
    disposition: not-addressed
    note: |
      No redaction or selection rule recorded next to doctor/fixtures/perf_capture.txt; it is clean of /Users/ paths only by the same truncation accident.
findings:
  - id: new
    severity: Critical
    family: traceability
    title: |
      The plan still specifies the cmd/pair-hoprtt + GO_BINS design that round 3 reversed, with no "## Revisions" entry
    detail: |
      This is the 2nd finding in family `traceability`. Rule, not site: a durable
      artifact that describes the work is updated in the SAME round the work
      changes it, via an appended "## Revisions" entry — because the plan is what
      the next milestone is executed from, and a stale one re-introduces the
      finding the change removed. Prevalence 2/2 (BR-21: the issue Plan and Log
      do not record what M1 did; this: the plan does not record what M1 became).
      Evidence: the plan's last commit is df283abe, the review base. Core
      concepts line 79 says `pair-hoprtt` lives in `cmd/pair-hoprtt/main.go`;
      the code is `cmd/internal/hoprttcmd/hoprtt.go` and `cmd/pair-hoprtt/` was
      deleted in c8f2d044. Line 107 still says perf.sh "locates it as
      $PAIR_HOME/bin/pair-hoprtt" — verbatim the expression BR-22 called
      Critical. M1.2b (:311-313) still instructs a GO_BINS entry and a
      `bin/pair-hoprtt` check. M2.6 (:389) still routes the Lua rolling-file
      write through artifactpath, disposed not-addressed twice by the plan gate.
      Fix: one Revisions entry plus edits at :79, :98-108, :283, :311-313, :389,
      :402.
  - id: new
    severity: Important
    family: unenforced-operating-envelope
    title: |
      The budget is checked between stages but no stage is bounded, so a single slow collector blows it without limit
    detail: |
      This is the 3rd finding in family `unenforced-operating-envelope`, so the
      deliverable is the rule: a wall-clock budget is enforced only when every
      operation that can exceed it is itself bounded; a check before a stage
      starts does not bound the stage. Prevalence 3/3 — BR-11 (no deadline at
      all), BR-25 (wrong shed order), this (no bound once a stage is entered).
      Evidence: `top -l 2 -n 60` (perf.sh:108), `iostat -d -w 1 -c 2` (:183),
      `sleep "$WINDOW"` (:158) and each probe_line invocation run to completion
      once started. The committed fixture reads elapsed_seconds=5 against
      budget_seconds=6 — one second of slack — and the operator symptom quoted
      in the issue is "top took 10 seconds". pipeRTT's hardcoded 500 samples
      (hoprtt.go:178) is 5 s at the ~10 ms/hop the issue says visible lag would
      require. Because perf_test.sh:36 asserts elapsed <= budget, `make test`
      goes red under exactly the conditions the capture exists for. Enumeration
      to sweep in one pass: wrap each external stage in a bounded runner, and
      derive probe sample counts from remaining budget and the observed first
      sample rather than from constants.
  - id: new
    severity: Important
    family: failure-reported-as-measurement
    title: |
      parse_samples turns a budget-shed sample_b into an empty-but-present sample, so delta reports every process as vanished
    detail: |
      This is the 8th finding in family `failure-reported-as-measurement`. The
      rule was already stated at BR-23 and its sweep enumeration was written
      ("every exported function in doctor.lua's 208 block"); this is a member of
      that enumeration the hand-sweep missed, which is the family reporting that
      a hand-sweep is the wrong instrument. Reproduced under `nvim -l` with a
      capture containing `## sample_b` / `skipped=budget reserved for probes`:
      delta returns rates=0 vanished=2 started=0 rows_a=2 rows_b=0 — on a real
      host that is vanished=856, a fabricated spawn-storm claim from a
      measurement that never happened, in exactly the squeezed-capture condition
      the tool exists for. The same shape arises from BR-5's residual, where a
      failed ps yields two empty samples that read as "nothing is running".
      Deliverable at this prevalence is mechanical: one table-driven test in
      doctor_test.lua iterating the exported functions of the #208 block and
      asserting each maps absent/failed input to absence, so the next function
      added is covered by construction. Site fix: parse_samples returns nil for
      a block carrying `skipped=` or no `###` sections.
  - id: new
    severity: Important
    family: untested-shell-surface
    title: |
      Nothing tests perf.sh's sample-row grammar, so the perf.sh-to-delta contract the Lua test claims to pin does not exist
    detail: |
      This is the 3rd finding in family `untested-shell-surface`. Rule: when two
      components are joined by a recorded fixture, the recording pins only the
      CONSUMER; the producer needs its own live-run assertion, or drift on the
      producer side is invisible. Prevalence 3/3 (BR-9 no test for perf.sh at
      all; BR-25's fix unpinned; this). doctor_test.lua asserts "the perf.sh ->
      delta contract is pinned. A change to either side that breaks the other
      now fails here" — that claim is false: it reads the static file
      doctor/fixtures/perf_capture.txt and never invokes perf.sh. Demonstrated
      in a scratch copy: changing sample() from tab-separated to pipe-separated
      output left `sh doctor/perf_test.sh` printing "perf.sh shape tests passed"
      and exiting 0 (perf_test.sh:23's `*[!0-9]*) continue` swallows the rows,
      per BR-26). perf_test.sh's required-key list also omits every probe key,
      which is why BR-25's shed-order fix has no failing test. Fix: assert a live
      run's `### procs` rows against ^[0-9]+\t\S+\t[0-9]+\t.+$ and `### cputime`
      against ^[0-9]+\t\S+$, and add pipe_hop_ms/fork_exec_ms to the key list.
  - id: new
    severity: Minor
    family: build-artifact-committed
    title: |
      perf_test.sh's mktemp fallback writes .perf-test-stub.$$ into the worktree and nothing gitignores it
    detail: |
      This is the 2nd finding in family `build-artifact-committed`. Rule: any
      path a build or test writes into the worktree is gitignored in the same
      change that introduces the write. Prevalence 2/2 (BR-4: the staged
      pair-hoprtt binary; this). perf_test.sh:54 falls back to
      "$repo/.perf-test-stub.$$" when mktemp fails — and bare `mktemp -d` fails
      in the sandboxed agent shell this file's header names as its target
      reader, so the fallback is the live path there, not the exception. An
      interrupted run leaves a directory of fake ps/top/sysctl executables
      untracked at the repo root.
```

---

## Re-review — 2026-09-07T00:50:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..a0688c0d9b0e6d224bd3dbb58eb17dde51dbcbd8 |
| command | sdlc milestone-close --issue 208 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T00:50:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1's core artifact works and I verified it end to end: `pair hoprtt` reports a 7 µs pipe hop and a 1.5 ms fork+exec through the real shipped binary, `doctor/perf.sh` renders every named collector as `n/a (<why>)` under tool denial, and the squeezed-budget shed order now protects the probes exactly as round 3 claimed. What blocks the boundary is that four Important findings carried into round 7 are still open — two of which I reproduced unchanged — and the round's own headline fix is unreachable in the environment its file names as its reader. BR-35 reproduces verbatim (`## sample_b` / `skipped=` still parses to an empty-but-present sample; `delta` returns `vanished=2`, which on a real host is `vanished=856` — a fabricated spawn-storm claim from a measurement that never happened). BR-25's shed-order fix is behaviorally correct but pinned by nothing. BR-34 is untouched. And the BR-36 grammar assertion added in `a0688c0d` passes vacuously here: `ps` is denied in this shell, so `### procs` is empty, the loop iterates zero rows, and I confirmed the pipe-separator mutation the commit message calls "mutation-verified" leaves the suite green — it only goes red once I inject a `ps` stub. AGENTS.md §5 requires Critical/Important cleared before the boundary; seven remain.

## 1. Strengths

- **`hoprtt_test.go:47` builds the real `pair` binary and invokes the subcommand through it.** This is the right instrument for the BR-22 class — reverting the dispatcher wiring in `cmd/pair-go/main.go:90` cannot pass. Verified green (`go test ./cmd/internal/hoprttcmd/... 3.347s`).
- **The positive control is honest.** `TestSpawnTimerMeasuresTheCommandNotTheHarness` (hoprtt_test.go:64) with a deliberately wide 0.05–15 ms band still catches the 18.7 ms harness bug it exists for, and `TestPipeHopIsFarCheaperThanSpawn` (`:77`) is a good cross-check that the two modes have not converged.
- **`collect()` (perf.sh:51) is a genuinely good three-outcome ladder** — absent / failed / returned-nothing, each named. I ran perf.sh in this ps/top/sysctl-denied shell and got `load=n/a (sysctl returned nothing)`, `process_count=n/a (ps failed)`, `cpu_idle_pct=n/a (top failed)`. No fabricated zeros. BR-5's main body is real work.
- **The shed order is correct and I reproduced it.** `PAIR_PERF_BUDGET=2 sh doctor/perf.sh` sheds `top`, `iostat` and `sample_b` while `pipe_hop_ms=0.007` and `fork_exec_ms=1.511` survive. Exactly the inversion round 3 promised.
- **`delta`'s bucket contract is enumerated and fixture-driven** (doctor_test.lua:170: `#rates + vanished + reused + unmeasured == rows_a`). That is the ARCH-ORDER lens applied properly to the join.
- **The hand-maintained-list sweep is complete this time** — `GO_BINS`, both `artifactpath` lists, `runtimebundlegen.explicitAssetPaths`, and `embed_test.go` all updated in one round, with the lesson written down (`workshop/lessons.md`). `go test ./cmd/internal/artifactpath/... ./cmd/internal/dispatcher/...` is green.

## 2. Critical findings

None new. BR-33 (the prior Critical) is addressed.

## 3. Important findings

**N-1 — `doctor/perf_test.sh` asserts a live run against the ambient system, so in the shell it names as its reader it validates nothing.** `perf_test.sh:14` runs `perf.sh` against whatever the machine provides. In this agent shell `ps` is denied (`/bin/ps: Operation not permitted`), so `### cputime` and `### procs` are empty sections; the BR-36 grammar loop (`:84-99`) iterates zero rows, `rows` (`:98`) is computed and never read, and `:101-102` assert only that the *headers* exist. I mutated `sample()` to pipe separators in a scratch copy and `sh doctor/perf_test.sh` printed `perf.sh shape tests passed`, exit 0; injecting a `ps` stub that emits recorded rows made the same mutation fail correctly. Fix (also closes BR-25's pinning gap in one pass): promote the existing stub dir at `:54-58` from stateless `exit 1` doubles to a recorded-output `ps`/`top`/`vm_stat`/`iostat` fake, assert `rows > 0`, and add a `PAIR_PERF_BUDGET=2` run asserting `pipe_hop_ms`/`fork_exec_ms` are present and `cpu_idle_pct` is shed.

**N-2 — `swap_rate` divides by `WINDOW` even when the window never elapsed, so a shed capture reports a rate over time that did not pass.** `perf.sh:153-160` skips `sleep "$WINDOW"` entirely when the budget is squeezed, but `:162-174` unconditionally divides the vm_stat counter difference by `WINDOW`. Reproduced: the full run reports `pageins_per_s=40.0`; `PAIR_PERF_BUDGET=2` reports `pageins_per_s=3.5` from two reads microseconds apart, presented identically. Same axis: `delta`'s `cpu_pct` divides by the *declared* window while `at_s` (the measured one) is emitted and discarded.

**N-3 — `parse_samples` is documented as the perf.sh→delta contract but returns 2 of the 3 things `delta` needs, and the missing one crashes it.** `doctor.lua:164` returns `sample_a, sample_b` only; `delta(a, b, window)` also needs `window`, which lives in the capture as the text `window_seconds=2`. M2's caller must therefore re-parse the capture with its own pattern — a second, untested copy of the contract `parse_samples` exists to own — and if it forwards the string, `doctor.lua:80` raises `attempt to compare string with number` (verified under `nvim -l`). Fix: return a third `meta` value carrying `window_seconds` and both `at_s` values as numbers, and have `delta` coerce with `tonumber(window)` before comparing.

## 4. Minor findings

- The root `README.md` is unchanged for `pair hoprtt` and `make test-perf-capture`. Not raised as a separate finding: README's subcommand list (`:254`) is explicitly non-exhaustive ("`pair clip …`, …") and it enumerates no make targets. `doctor/README.md` is the real gap and is already BR-20.
- `make test-perf-capture: $(BIN_DIR)/pair` declares a dependency the script may not exercise — `perf.sh:199` resolves `pair` on PATH first, so an older installed `pair` shadows the just-built one. Coincidentally harmless here (PATH's `pair` is a symlink into `bin/`), but the target's guarantee is weaker than it reads.
- The plan's ARCH-SECURE argv allowlist (`go`, `compile`, `link`, `zellij`, `pair*`, `nvim` get full argv; everything else gets name+pid) is not implemented at all — `sample()` prints `comm` for every process and argv for none. Folded into BR-17's disposition.
- `Run` (hoprtt.go:146) silently ignores an unrecognised first argument and runs the pipe probe; `pair hoprtt -spwan 5` measures the wrong thing and exits 0.

## 5. Test coverage notes

`nvim -l nvim/doctor_test.lua` and `go test ./cmd/internal/hoprttcmd/...` are both green and both pin real logic — the Lua suite drives `delta` from the recorded capture and asserts the bucket-accounting invariant rather than restating the implementation. The gap is entirely on the shell side: `doctor/perf_test.sh` is the only test of `perf.sh`, it has no control over its subject's inputs, and three separate behaviours it is supposed to protect (the sample-row grammar, the shed order, the `n/a` rule for the sample blocks) are unpinned or vacuous. Nothing exercises `perf.sh`'s degraded paths as *data* — the stub dir is the seam and it is one commit away from being useful.

## 6. Architectural notes

- **ARCH-DRY — flag.** BR-27 stands: four re-implementations of `collect()`'s ladder (`:101-120` top, `:178-189` disk, `:211` probe_line, and `sample()`/`cputimes()` at `:130-131` which skip it entirely). BR-13's `_swapnow` extraction shows the shape the rest should take.
- **ARCH-PURE — pass.** The join is pure Lua tested headless with no mocks; `summary` is pure and unit-tested; `perf.sh` is thin orchestration. The split holds.
- **ARCH-PURPOSE — flag.** The purpose is an honest reading *at the bad moment*. The degraded path is where the tool spends its life and where it still fabricates (N-2, BR-35, BR-5's residual). This is the easy-subset pattern: the healthy path is finished, the degraded path is the point.
- **ARCH-MOCK — flag.** `perf.sh` shells out to six external tools with no stateful double. The plan reasons the exemption ("system tools cannot be faked meaningfully"), but N-1 shows the opposite: a ~10-line recorded-output `ps` makes the producer assertions run. The seam already exists.
- **ARCH-CONSTRAINTS — flag.** BR-34 untouched — a budget checked *between* stages bounds nothing once a stage is entered, and `perf_test.sh:36`'s `elapsed <= budget` therefore goes red under exactly the load the capture exists for.
- **ARCH-SECURE — flag.** BR-32 (recorded host capture safe only by an unrecorded truncation accident), BR-17 (full executable paths), BR-18 (`PAIR_PERF_*` unvalidated into `sleep` and `$(( ))`; `PROBE_RESERVE` is new this window and joins them). No credential surface; the report leaving the machine is by design and stated.
- **ARCH-ORDER — mixed.** `delta`'s `(state, event)` enumeration is explicit and injected-fixture tested — good. But the *shed* transitions are not enumerated: `sample_b: shed` has downstream consumers (`swap_rate`, `parse_samples`, `delta`) and none of them observe it. And `perf_test.sh` can observe exactly one interleaving with no seam to inject another (N-1).

## 7. Plan revision recommendations

The `## Revisions` entry added in `a0688c0d` is good and covers the reversal, the shed order, `verdict`'s asymmetry and the fixture location. Two additions:

- **M2.6 / `artifactpath`.** The Revisions entry's blanket override names `cmd/pair-hoprtt` / `bin/pair-hoprtt` / `GO_BINS` / recipe stanzas — it does not reach `:389`, which still routes the Lua rolling-file write through `cmd/internal/artifactpath`, a Go internal package with no CLI surface that Lua cannot call. Append: *"M2.6's rolling-file path comes from `pair_data_dir()` (nvim/init.lua:484) + `pair_tag()` (:4012), the existing Lua idiom; `artifactpath` is a Go internal package with no CLI surface and is not reachable from `nvim/doctor.lua`."*
- **Core concepts table row for `delta`.** The table promises `delta (two-sample join) | nvim/doctor.lua | new`; as built the surface is `parse_samples` + `delta` + `verdict` + `note_from_lines` + `parse_duration`, and `parse_samples` (the contract function N-3 is about) has no row. Add it, and record that `delta` requires a `window` the parse step does not currently return.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan :389 still routes M2.6's rolling-file write through artifactpath; verified no CLI surface exists and pair_data_dir() (nvim/init.lua:484) remains the idiom. The GO_BINS/pair- prefix half is moot now the probe is a subcommand.
  - id: BR-5
    disposition: not-addressed
    note: |
      collect() is right for the named collectors, but emit_sample's `cputimes 2>/dev/null || say "n/a (ps unavailable)"` is a pipeline whose status is awk's, so the fallback is unreachable; with ps denied here both sample blocks render empty and delta reads rows_a=0/rows_b=0 as "nothing is running".
  - id: BR-16
    disposition: not-addressed
    note: |
      probe_line's awk (perf.sh:216-221) still emits only $1/$2; the sample count $4 survives only inside the failure message.
  - id: BR-17
    disposition: not-addressed
    note: |
      perf.sh:130 unchanged. Also: the plan's ARCH-SECURE allowlist (name+pid for all, argv for go/compile/link/zellij/pair*/nvim) is not implemented at all.
  - id: BR-18
    disposition: not-addressed
    note: |
      WINDOW, BUDGET and the newly added PROBE_RESERVE are all unvalidated and undocumented; PROBE_RESERVE > BUDGET makes collectors_done true immediately.
  - id: BR-19
    disposition: not-addressed
    note: |
      No mention of 4f9365b3 or M2.2b landing inside M1's window in the issue or the plan.
  - id: BR-20
    disposition: not-addressed
    note: |
      grep for "perf" in doctor/README.md returns nothing. Root README also unchanged for `pair hoprtt` / `make test-perf-capture`, though its subcommand list is explicitly non-exhaustive.
  - id: BR-21
    disposition: not-addressed
    note: |
      Issue Plan M1 still unticked; Log has only the 2026-09-06 entry. M1.4's zellij ~13ms baseline is still unmeasured — the probe renders `n/a (probe failed)` in this shell.
  - id: BR-25
    disposition: not-addressed
    note: |
      Behavior IS fixed and I reproduced it (PAIR_PERF_BUDGET=2 sheds top/iostat/sample_b, probes survive), but nothing pins it: perf_test.sh's key list carries no probe key and there is no squeezed-budget run, so reverting the shed order leaves make test green.
  - id: BR-26
    disposition: not-addressed
    note: |
      perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged; the new grammar loop is a separate pass and does not replace it.
  - id: BR-27
    disposition: not-addressed
    note: |
      Four ladders still present: top (:101-120), disk (:178-189), probe_line (:211), and sample()/cputimes() (:130-131) bypassing collect() entirely.
  - id: BR-28
    disposition: not-addressed
    note: |
      perf.sh:172's swap arithmetic is unchanged and still divides by WINDOW unguarded.
  - id: BR-29
    disposition: not-addressed
    note: |
      perf.sh:166 still reports a vm_stat that exists but exits non-zero as "(vm_stat unavailable)".
  - id: BR-30
    disposition: not-addressed
    note: |
      doctor.lua:163 still says "The fixture in nvim/fixtures/"; :75 still omits `unmeasured` from the documented return shape.
  - id: BR-31
    disposition: not-addressed
    note: |
      doctor.lua:90 unchanged; no `cb < ca` guard in the rate branch, so an unparseable etime still lets a reused pid produce a negative rate.
  - id: BR-32
    disposition: not-addressed
    note: |
      No redaction or selection rule recorded; doctor/fixtures/ contains only perf_capture.txt, and sample() still emits full executable paths.
  - id: BR-33
    disposition: addressed
    note: |
      Revisions entry appended recording the reversal, the shed order, verdict asymmetry and the fixture location; per AGENTS.md the append-don't-overwrite convention makes the body edits optional. The M2.6/artifactpath item it also named stays open as BR-1.
  - id: BR-34
    disposition: not-addressed
    note: |
      top -l 2 -n 60 (:108), iostat -d -w 1 -c 2 (:183), sleep "$WINDOW" (:158), probe_line, and pipeRTT's hardcoded 500 samples (hoprtt.go:178) all still run to completion once entered.
  - id: BR-35
    disposition: not-addressed
    note: |
      Reproduced verbatim under nvim -l: a capture with `## sample_b` / `skipped=budget reserved for probes` still yields rates=0 vanished=2 started=0 rows_a=2 rows_b=0. dc5a3d03 covers only '' and "no sample block at all"; the table-driven test the finding asked for was not written.
  - id: BR-36
    disposition: addressed
    note: |
      Verified by mutation — pipe-separated sample() output fails the new grammar loop once ps produces rows. Its vacuity in a ps-denied shell is raised separately.
  - id: BR-37
    disposition: not-addressed
    note: |
      .gitignore carries no .perf-test-stub pattern; perf_test.sh:54's fallback is unchanged.
findings:
  - id: new
    severity: Important
    family: untested-shell-surface
    title: |
      perf_test.sh asserts a live run against the ambient system, so the BR-36 grammar pin validates zero rows wherever ps is denied
    detail: |
      This is the 4th finding in family `untested-shell-surface`. Do NOT fix this
      instance — the rule is: a shell test must CONTROL the environment its
      subject reads; a live run against the ambient system asserts nothing when
      the system denies the tool, and a green run is then a sample of size zero
      reporting as coverage. Prevalence 4/4 (BR-9 no test at all; BR-25's fix
      unpinned; BR-36 the recorded fixture pinning only the consumer; this).
      Evidence: `ps` is denied in this agent shell (`/bin/ps: Operation not
      permitted`), so `### procs` and `### cputime` render as empty sections;
      the loop at perf_test.sh:84-99 iterates zero times, `rows` (:98) is
      computed and never read, and :101-102 assert only that the headers exist.
      I mutated sample() to pipe separators in a scratch copy and
      `sh doctor/perf_test.sh` printed "perf.sh shape tests passed" exit 0;
      the same mutation fails correctly once I put a recorded-output ps stub on
      PATH. Sweep in one pass: promote the existing stub dir (:54-58) from
      stateless `exit 1` doubles to a recorded-output ps/top/vm_stat/iostat
      fake, assert rows > 0, and add a PAIR_PERF_BUDGET=2 run asserting
      pipe_hop_ms/fork_exec_ms survive and cpu_idle_pct sheds — which also
      closes BR-25's missing pin (ARCH-MOCK, ARCH-ORDER).
  - id: new
    severity: Important
    family: failure-reported-as-measurement
    title: |
      swap_rate divides by WINDOW even when sample_b was shed and the sleep never ran, reporting a rate over time that did not pass
    detail: |
      This is the 9th finding in family `failure-reported-as-measurement`. Do
      NOT fix this instance. The rule this round's instances need, stated once:
      when a stage is SHED or FAILS, every value derived from that stage must
      shed with it — the shed has to propagate through the dependency graph, not
      stop at the stage that was skipped. Evidence: perf.sh:153-160 skips
      `sleep "$WINDOW"` under a squeezed budget, but :162-174 unconditionally
      divides the vm_stat counter difference by WINDOW. Reproduced: the full run
      reports pageins_per_s=40.0; `PAIR_PERF_BUDGET=2` reports
      pageins_per_s=3.5 from two reads microseconds apart, rendered
      identically. The enumeration this rule implies, all live: swap_rate
      (here); parse_samples/delta reading a shed sample_b as a real one (BR-35);
      the sample blocks degrading to silence rather than n/a (BR-5's residual);
      and, on the same axis, delta's cpu_pct dividing by the DECLARED window
      while the measured one (`at_s`, emitted per sample) is discarded.
  - id: new
    severity: Important
    family: incomplete-parse-contract
    title: |
      parse_samples is billed as the perf.sh-to-delta contract but omits window_seconds, which delta needs and which crashes it as a string
    detail: |
      doctor.lua:164 returns only sample_a and sample_b, while
      `delta(a, b, window)` also needs the window — which lives in the capture
      as the text `window_seconds=2`. M2's caller must therefore re-parse the
      capture with its own pattern, a second untested copy of the contract this
      function exists to own (ARCH-PURPOSE's hand-maintained restatement), and
      if it forwards the string unconverted doctor.lua:80 raises "attempt to
      compare string with number" — verified under nvim -l. That is a crash on
      the primary path, at the moment the operator invoked the tool because the
      machine was already struggling. Fix: return a third `meta` value carrying
      window_seconds and both at_s values as numbers (which also supplies the
      measured window the preceding finding needs), and coerce with
      tonumber(window) at delta's guard so a bad value degrades to the existing
      empty result rather than throwing.
```

---

## Re-review — 2026-09-07T01:02:36-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..d10714051a203c7d3021ea15136a0532e8e17fd3 |
| command | sdlc milestone-close --issue 208 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T01:02:36-07:00 |
| verdict | REWORK |

## Review

I've inspected the full window, run every suite, and reproduced the degraded paths directly.

```verdict
verdict: REWORK
confidence: high
```

The round-4 commit's three Lua-side fixes are real and mutation-verified — I reverted `usable()` and `tonumber(window)` in a scratch copy and the suite went red both times, so BR-35 and BR-40 are genuinely pinned. But two things block the boundary. **(1) `make test` is red right now**: BR-38's fix asserted `proc_rows >= 10` against the *ambient* system rather than controlling it, and `ps` is denied in the sandboxed agent shell the file's own header names as its target reader — `sh doctor/perf_test.sh` fails with "only 0 sample rows". That is the exact class BR-24 already fixed once, reintroduced by the fix for the finding that forbade this route. **(2) BR-25 is not actually closed**: the shed *order* is fixed (verified — `PAIR_PERF_BUDGET=2` keeps the probes), but `PROBE_RESERVE` is not a reservation. `probe_line` guards on `over_budget` (elapsed ≥ BUDGET), so any unbounded stage that overruns still sheds every probe. I reproduced it with a `top` stub that takes 8 s — the operator's literal reported symptom is "top took 10 seconds" — and got `pipe_hop_ms=n/a (budget exceeded)`, `fork_exec_ms=n/a`, `zellij_action_ms=n/a`, `elapsed_seconds=8` against `budget_seconds=6`. Zero probe rows in the one condition this tool exists for, plus a red `perf_test.sh:36`. Both are cheap to fix but neither can be crossed.

## 1. Strengths

- **The subcommand reversal is right and is now pinned by a test that builds the real binary.** `cmd/pair-go/main.go:90` + `dispatcher.go:63` route `pair hoprtt` through the one binary every distribution ships, and `hoprtt_test.go:47` runs `go build ./cmd/pair-go` and invokes through the product, not the package — the wiring cannot silently regress.
- **BR-35/BR-40 are complete by the standard that matters.** Reverting `usable()` (`nvim/doctor.lua:200-205`) produced `FAIL a SHED sample_b is nil` + `FAIL a shed sample must not report every process as vanished`; removing `window = tonumber(window)` (`:80`) produced `attempt to compare string with number`. The tests fail without the fixes.
- **`collect()`'s three-outcome ladder works.** Under a full tool-denial stub run, every scalar collector rendered `n/a (<tool> failed|returned nothing|unavailable)` — not one fabricated value, including `process_count`, which was the original BR-5 fabrication.
- **`delta`'s bucket contract is asserted against a real capture**, not literals: `doctor_test.lua:170` checks `#rates + vanished + reused + unmeasured == rows_a`, so a pid reaching no bucket fails the suite.
- **`doctor/perf.sh:73-77`'s comment is exemplary** — it records *why* `grep -c` was replaced by awk (prints 0 *and* exits 1; `|| true` turned a failed `ps` into a fabricated 0). That is the kind of comment that stops a regression.

## 2. Critical findings

None. The blocking items are dispositions of prior Important findings (BR-25, BR-34, BR-38 below), not new defects.

## 3. Important findings

All three are re-raised prior findings; see the dispose block. Summarised:

- **BR-38 not-addressed** — `doctor/perf_test.sh:105-108` asserts against the ambient system. Reproduced: `sh doctor/perf_test.sh` → `FAIL only 0 sample rows`, exit 1, so `make test-perf-capture` and `make test` are red in the sandboxed agent shell. The rule BR-38 stated (control the environment; promote the `exit 1` stubs at `:54-58` to recorded-output ps/top/vm_stat/iostat fakes) was not applied. Dead leftovers from the patch: `in_sample`/`rows` at `:85,:91,:98` are computed in a subshell and never read.
- **BR-25 not-addressed** — reproduced above; `PROBE_RESERVE` gates collector *start*, not probe *survival*.
- **BR-34 not-addressed** — same experiment; no stage is bounded (`top -l 2 -n 60` `:108`, `iostat -d -w 1 -c 2` `:190`, `sleep "$WINDOW"` `:159`, each `probe_line`, and `pipeRTT(500)` at `hoprtt.go:178`).
- **BR-5 residual not-addressed** — `emit_sample`'s `cputimes 2>/dev/null || say "n/a (ps unavailable)"` (`:146,:148`) is dead: the `||` sees awk's exit status, not ps's, so under denial both `### cputime` and `### procs` render as *silence*, not `n/a`. Confirmed in the stub run.

## 4. Minor findings

- `pair hoprtt --spawn 5 -- x` (or any unrecognised flag) silently falls through to the 500-sample pipe probe and exits 0 — new finding below.
- The plan's Core-concepts table never gained rows for `parse_samples`, `parse_duration`, `verdict`/`FRAME_MS` — new finding below.
- The generated bundle mirror `cmd/internal/runtimebundle/assets/runtime/files/doctor/perf.sh` is stale vs `doctor/perf.sh`; harmless (regenerated by `make test`'s `runtimebundle-generate` prerequisite), noted only so it isn't mistaken for drift.
- 16 further prior Minors remain open unchanged (BR-1, BR-16–BR-21, BR-26–BR-32, BR-37, BR-39).

## 5. Test coverage notes

`nvim -l nvim/doctor_test.lua` passes (mutation-verified twice). `go test ./cmd/internal/hoprttcmd/` passes in 3.2 s. `sh doctor/perf_test.sh` **fails**. The shell surface remains the weak side: no test enters the `PAIR_PERF_BUDGET` seam even though it exists, so the shed ordering — the behaviour reversed twice now — is observed by exactly zero assertions, and the swap-shed fix from this round (`perf.sh:167-171`) has no test either. One `PAIR_PERF_BUDGET=2` run asserting `pipe_hop_ms` survives, `cpu_idle_pct` sheds, and `swap=n/a (sample window was shed…)` would pin three open findings at once.

## 6. Architectural notes

- **ARCH-DRY — flag.** BR-27 stands: four re-implementations of the `collect()` ladder. The sample-row grammar is additionally restated in three languages (`perf.sh:130`, `perf_test.sh:94-97`, `doctor.lua:190`) with no single source.
- **ARCH-PURE — pass.** `delta`/`parse_samples`/`verdict`/`note_from_lines`/`parse_duration` are string-in/table-out and run under `nvim -l` with no IO. Sole flag: `perf.sh:179`'s swap arithmetic (BR-28) contradicts the file's own rule 2.
- **ARCH-PURPOSE — flag.** BR-38 named a class and wrote the enumeration; the round fixed the site and skipped the enumeration. BR-39's four-member enumeration is 2/4 swept — `at_s` is still emitted per sample and discarded, so `delta` divides by the *declared* window while the *measured* one is on disk.
- **ARCH-MOCK — flag.** No stateful fake for `ps`/`top`/`vm_stat`/`iostat`. The doubles are stateless `exit 1` scripts, which can only test the denial path, never the grammar.
- **ARCH-CONSTRAINTS — flag.** "≤ 6 s wall clock, hard. Enforced, not hoped" is contradicted by a reproduced 8.3 s run with no probe rows.
- **ARCH-SECURE — flag.** `comm=` emits full executable paths into a report designed to leave the machine (BR-17), and the fixture's clean state is an unrecorded accident (BR-32). Separately, the plan's promised argv allowlist is not implemented — the built behaviour is *safer*, but the plan now over-claims.
- **ARCH-ORDER — flag.** The interleaving seam exists (`PAIR_PERF_BUDGET`, `PAIR_PERF_PROBE_RESERVE`) and no test enters it, so every shell assertion is a sample of size one from whichever ordering the ambient machine produced.

## 7. Plan revision recommendations

1. **`## Revisions` — "the probe reserve is a start-gate, not a reservation."** Record that ARCH-CONSTRAINTS' "Enforced, not hoped" is currently false, with the measured counter-example (`top` stub 8 s → 8.3 s elapsed, zero probe rows), and state the replacement rule: a wall-clock budget is enforced only when every stage that can exceed it is itself bounded.
2. **`## Revisions` — Core concepts table completion.** Add rows for `parse_samples` (PURE, `nvim/doctor.lua`, new), `parse_duration` (PURE, new), `verdict`/`FRAME_MS` (PURE, new, asymmetric contract), and mark `doctor/fixtures/perf_capture.txt` as the recorded fixture. Three pure entities shipped in M1 that the table the next milestone greps does not list.
3. **`## Revisions` — M2.6's mechanism.** `artifactpath` is `cmd/internal/artifactpath` with no CLI surface; the rolling-file write is Lua. Replace with `pair_data_dir()` (`nvim/init.lua:484`) + `pair_tag()` (`:4012`).
4. **Issue `## Plan`** — the M1 row still reads "`cmd/hoprtt` (pipe-hop probe…)", a path that does not exist; the plan's Revisions covers the plan file but not the issue file.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The pair- prefix half is overtaken by the subcommand reversal, but plan M2.6 still routes the Lua write through artifactpath, which has no CLI surface (verified: no reference in cmd/pair-go/main.go or dispatcher.go).
  - id: BR-5
    disposition: not-addressed
    note: |
      Scalar collectors are fixed and verified, but emit_sample's `cputimes || say n/a` is dead (the || sees awk's status, not ps's), so both sample sections still render as silence under denial.
  - id: BR-16
    disposition: not-addressed
    note: |
      probe_line's awk still prints only $1/$2; the sample count in $4 is discarded.
  - id: BR-17
    disposition: not-addressed
    note: |
      perf.sh:130 still truncates at awk $4 and emits comm= as a full executable path.
  - id: BR-18
    disposition: not-addressed
    note: |
      PAIR_PERF_WINDOW/BUDGET/PROBE_RESERVE all flow unvalidated; none documented outside workshop/.
  - id: BR-19
    disposition: not-addressed
    note: |
      No Log entry or plan note records that 4f9365b3 (the M2.2b join) landed inside the M1 window.
  - id: BR-20
    disposition: not-addressed
    note: |
      grep perf doctor/README.md returns nothing.
  - id: BR-21
    disposition: not-addressed
    note: |
      Issue Plan M1 and every plan M1.x checkbox are unticked; the Log has only the 2026-09-06 filing entry, so M1.4 and M1.5 have no recorded evidence.
  - id: BR-25
    disposition: not-addressed
    note: |
      Shed ORDER is fixed and verified at PAIR_PERF_BUDGET=2, but PROBE_RESERVE only gates collector start; probe_line guards on over_budget, so an 8s top stub yields zero probe rows and elapsed_seconds=8 over a 6s budget.
  - id: BR-26
    disposition: not-addressed
    note: |
      perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
  - id: BR-27
    disposition: not-addressed
    note: |
      Four ladder shapes remain: the top block, the disk block, probe_line, and the sample blocks that skip collect() entirely.
  - id: BR-28
    disposition: not-addressed
    note: |
      perf.sh:179 still does the rate arithmetic in awk and divides by WINDOW without a zero guard.
  - id: BR-29
    disposition: not-addressed
    note: |
      Reproduced with a vm_stat stub exiting 1: renders `swap=n/a (vm_stat unavailable)`. Same misnaming at _loadavg, which reports "returned nothing" when sysctl failed, because the pipeline's status is awk's.
  - id: BR-30
    disposition: not-addressed
    note: |
      doctor.lua still says "The fixture in nvim/fixtures/" and still omits `unmeasured` from delta's documented return shape.
  - id: BR-31
    disposition: not-addressed
    note: |
      The reuse branch is still etime-only; no `cb < ca` guard at the point of derivation.
  - id: BR-32
    disposition: not-addressed
    note: |
      The fixture still carries no redaction/selection rule; sample() still emits full paths, so the next re-capture commits home-directory paths.
  - id: BR-34
    disposition: not-addressed
    note: |
      Reproduced: a `top` stub taking 8s runs to completion and the capture reports elapsed_seconds=8 against budget_seconds=6, which also makes perf_test.sh:36 go red under exactly the conditions the capture exists for.
  - id: BR-35
    disposition: addressed
    note: |
      Mutation-verified: reverting usable() in a scratch copy produced two failures including "a shed sample must not report every process as vanished".
  - id: BR-37
    disposition: not-addressed
    note: |
      Confirmed live: `mktemp -d` fails in this agent shell, so `$repo/.perf-test-stub.$$` is the taken path, and .gitignore has no entry for it.
  - id: BR-38
    disposition: not-addressed
    note: |
      The fix asserts rows against the ambient system instead of controlling it, so `sh doctor/perf_test.sh` now FAILS ("only 0 sample rows") wherever ps is denied, turning make test red for the reader the file header names. The stateless stubs were not promoted to a recorded-output fake, and in_sample/rows at :85-98 are now dead subshell locals.
  - id: BR-39
    disposition: not-addressed
    note: |
      The swap_rate instance is fixed but pinned by no test; 2 of the 4 enumerated members remain live - the sample blocks still degrade to silence (BR-5 residual), and delta still divides by the DECLARED window while the measured at_s values are parsed away.
  - id: BR-40
    disposition: addressed
    note: |
      Mutation-verified twice: removing tonumber(window) crashes the suite, removing the window_seconds parse fails the assertion. The at_s half of the fix sketch is carried by BR-39.
findings:
  - id: new
    severity: Minor
    family: unguarded-edge-case
    title: |
      pair hoprtt silently ignores unrecognised arguments and runs the 500-sample pipe probe instead
    detail: |
      This is the 4th finding in family `unguarded-edge-case`. Do NOT fix this
      instance. The rule covering all four: a value read from outside the
      function - an argv token, an env var, a counter difference - is rejected
      at the point it is read when it is not one of the forms the code
      understands; falling through to a default produces a successful-looking
      measurement of something the caller did not ask for. Evidence:
      hoprttcmd.go:148-152 tests args[0] against exactly "-child" and "-spawn",
      so `pair hoprtt --spawn 5 -- x` runs pipeRTT(500) and exits 0 with a
      pipe-hop number labelled as whatever the caller thought it asked for -
      the same defect TestUsageErrorsRatherThanSilentlyMeasuringNothing exists
      to prevent, one token away from the cases it covers. Prevalence 4/4, all
      live: this, BR-31 (a reused pid detected only via etime, so an
      unparseable etime yields a negative rate), BR-28 (WINDOW=0 divides to
      inf), BR-18 (PAIR_PERF_* unvalidated into sleep and awk -v). The
      enumeration to sweep in one pass is those four sites.
  - id: new
    severity: Minor
    family: traceability
    title: |
      Three pure entities shipped in M1 have no row in the plan's Core concepts table
    detail: |
      This is the 3rd finding in family `traceability`. Do NOT fix this
      instance. The rule: the Core concepts table is the greppable contract the
      next milestone reads, so every entity that ships gets its row in the SAME
      round it lands - a table that lags is a plan claiming a smaller surface
      than the code delivers, which is the same defect as one claiming a
      surface the code lacks (BR-33). Live members: `parse_samples`
      (nvim/doctor.lua, billed in its own docstring as THE perf.sh-to-delta
      contract), `parse_duration`, and `verdict`/`FRAME_MS` - the last of which
      M2.3 describes in prose with a symmetric contract the code has since
      replaced with an asymmetric one. Prevalence 3/3 with BR-21 (the issue's
      M1 row still names cmd/hoprtt, a path that does not exist) and BR-19 (no
      record that 4f9365b3 landed inside the M1 window). The sweep is one
      `## Revisions` entry adding the rows and correcting the two stale
      references.
```

---

## Re-review — 2026-09-07T11:05:44-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..b57cd08a0acfd3003aab61a147c39eb5fe666c40 |
| command | sdlc milestone-close --issue 208 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T11:05:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1 surface is real and the pure core is genuinely good: `doctor.lua`'s five pure functions run headless under `nvim -l`, `delta` enumerates its four buckets with a "every pid lands in exactly one bucket" assertion driven from a recorded capture, `verdict` is correctly asymmetric, and `pair hoprtt` is reachable through the *shipped* binary with a positive control that would have caught the 18.7 ms harness bug. Tests pass: `go test ./cmd/internal/hoprttcmd` ok, `make test-lua` ok (`all doctor.lua tests passed`), `artifactpath`/`runtimebundle` ok. What blocks a clean SHIP is one thing that is cheap and one thing that is not: **`make test` is red right now in this shell** (`sh doctor/perf_test.sh` → `FAIL only 0 sample rows`) because BR-38's fix asserts against the ambient system, and `ps`/`top`/`sysctl`/`iostat` are denied in the agent shell that `perf.sh:10-12` names as its own target reader — so the repo's close-gate evidence path is unrunnable for one class of reader. Alongside it, BR-39's rule ("a shed stage must propagate through the dependency graph") was applied to the site it named and not to the class: two of the four sites its own enumeration listed are still live, including `delta` dividing by the *declared* `window_seconds` while the *measured* `at_s` delta is emitted and discarded — which overstates every `cpu_pct` on exactly the slow machine this tool exists for. The BR-34 hardening split into `#210` is an explicit operator decision and I treat it as settled, not as an open gate item.

## 1. Strengths

- **`nvim/doctor.lua:75-125` — `delta` is the right code in the right place.** The plan argued the pid join belongs in tested Lua rather than shell and the code delivers exactly that: vanished / started / reused / unmeasured are four named buckets, and `doctor_test.lua`'s fixture block asserts `#rates + vanished + reused + unmeasured == rows_a`. That closure assertion is stronger than the per-case tests around it and is what makes BR-3's class unrepeatable.
- **`cmd/internal/hoprttcmd/hoprtt_test.go:47-54` — the reachability test builds the real `pair` binary.** This is the fix for the design reversal actually being pinned, not asserted. Reverting the `cmd/pair-go/main.go:90-91` dispatch case makes this go red, which is what the previous PATH-first attempt lacked entirely.
- **`hoprtt.go:117-129` + `:171-174` — the failure count.** Timing a command that fails instantly reports a dead dependency as the fastest number in the report; counting failures and refusing to print a median when all of them failed is the correct shape, and `probe_line` (`perf.sh:223-228`) consumes the 5th field rather than ignoring it. Verified live: `zellij_action_ms=n/a (probe failed)`.
- **Shed order verified behaviorally.** `PAIR_PERF_BUDGET=2 sh doctor/perf.sh` sheds `top`, `iostat` and `sample_b` while `pipe_hop_ms` and `fork_exec_ms` both survive. The reverse-value ordering the round-3 reversal introduced does hold under a squeeze.
- **`workshop/lessons.md:3638-3675`** — five rules, each traceable to a specific finding, including the one worth the most ("ask which distributions ship this before asking how the build finds it"). AGENTS.md §4 satisfied properly rather than ceremonially.

## 2. Critical findings

None.

## 3. Important findings

**`doctor/perf_test.sh:103-108` — BR-38 not-addressed: the fix swapped a vacuous pass for a hard failure, without controlling the environment.**
The `proc_rows -lt 10` guard removes the false green, but `test-perf-capture` is in the `test` chain (`Makefile.local:120`), so `make test` now fails outright wherever `ps` is denied. Reproduced: `sh doctor/perf_test.sh` → `FAIL only 0 sample rows; the grammar pin validated almost nothing`, exit 1; `ps -Ao pid=` → `operation not permitted`. The rule is unchanged and still unapplied — *a shell test must control the environment its subject reads*; both failure modes are the same defect wearing different clothes. The sweep is still the one BR-38 named: promote the stub dir (`perf_test.sh:54-58`) from stateless `exit 1` doubles to recorded-output `ps`/`top`/`vm_stat`/`iostat` fakes, and assert against those. (ARCH-MOCK, ARCH-ORDER — this test can observe exactly one interleaving, the ambient one.)

**`doctor/perf.sh:141-149` — BR-5 not-addressed: the two sample blocks still degrade to silence.**
Reproduced with `ps` denied: `### cputime` and `### procs` render as **empty sections**, while every neighbouring collector correctly reads `n/a (ps failed)`. `cputimes()` and `sample()` are `ps | awk` pipelines, so the pipeline exit status is awk's (0) and the `|| say "n/a (ps unavailable)"` arm at `:146`/`:148` is unreachable. The Lua side degrades safely (`usable()` returns nil on an empty `procs`), so this is a reporting-honesty gap rather than a fabrication — but it is the one collector class still outside the rule the file's own header states at `:9-12`.

**`nvim/doctor.lua:97` + `doctor/perf.sh:144` — BR-39 not-addressed: the class was not swept, and the remaining member produces a wrong number.**
The `swap_rate` instance is fixed and verified (`PAIR_PERF_BUDGET=1` → `swap=n/a (sample window was shed; no interval to rate over)`). But BR-39's own enumeration named four sites and two are live. The material one: `perf.sh` emits `at_s=<epoch>` per sample — the *measured* interval — and `parse_samples` discards it (no branch matches `at_s=` inside a sample block), so `delta` divides by the *declared* `window_seconds`. On a machine where `sleep 2` takes 5 s — the only condition this capture is for — every `cpu_pct` reads ~2.5× truth, rendered identically to a correct one. Fix: parse `at_s` per sample and have `delta` prefer `b.at_s - a.at_s` when both are present, falling back to the declared window.

## 4. Minor findings

- **BR-1** not-addressed (residual): plan M2.6 (`:387-392`) still routes the Lua rolling-file write "via `artifactpath`" — `cmd/internal/artifactpath` has no CLI surface (`grep artifactpath cmd/pair-go/main.go` → nothing), so Lua cannot reach it. Use `pair_data_dir()` / `pair_tag()`. The `GO_BINS` / `pair-` prefix half is overtaken by the subcommand reversal.
- **BR-16** not-addressed: `probe_line`'s awk (`perf.sh:227`) prints `$1`/`$2` only; the sample count `$4` is discarded, so a truncated pipe run reads identically to a full one.
- **BR-17** not-addressed: `sample()` (`:130`) still takes `$4`, and `comm=` on macOS is a full executable path — so the report carries `/Users/…/workspace/…/bin/pair` where the plan's own ARCH-SECURE section promised "process name and pid, not full argv", for a report that by design leaves the machine.
- **BR-18** not-addressed, with sharper evidence: `PAIR_PERF_WINDOW=0` **and** `PAIR_PERF_WINDOW=abc` both render the entire `## swap_rate` section as *nothing at all* — no key, no `n/a`, no reason (awk divide-by-zero / non-numeric `-v w`). With `abc`, `sleep` errors but `WINDOW_ELAPSED=1` is still set. No shell injection (quoting is correct). Still undocumented in atlas/README.
- **BR-19** not-addressed: `4f9365b3` ("#208 M2: the pure join") is still inside the M1 window with no record; M2's base will be mis-read as the branch point.
- **BR-20** not-addressed: `doctor/README.md` has zero occurrences of "perf" and still describes `doctor/` as the drift tool only. `atlas/index.md` was updated; its sibling was not. (`doctor/SKILL.md` is correctly M2.7's job.)
- **BR-21** not-addressed: the issue's `## Plan` M1 row is unticked and still names `cmd/hoprtt`; `## Log` has no 2026-09-07 entry and no M1.4/M1.5 evidence.
- **BR-26** not-addressed: `perf_test.sh:23`'s `*[!0-9]*) continue` still swallows every stray line containing a non-digit.
- **BR-27** not-addressed: `collect()` (`:51`) is still re-implemented at `:101-120`, `:185-196` and `:218-229`, and skipped entirely by `sample()`/`cputimes()` — which is structurally why BR-5's residual exists (ARCH-DRY).
- **BR-28** not-addressed: `:179` is still shell arithmetic against the file's own rule 2 (`:14-17`), and `WINDOW=0` now silently erases the section rather than printing `inf`.
- **BR-29** not-addressed: `:172-173` still renders a `vm_stat` that exists-but-exits-non-zero as `n/a (vm_stat unavailable)`.
- **BR-30** not-addressed: `doctor.lua:157` still says "The fixture in nvim/fixtures/" (it is `doctor/fixtures/`), and `:75`'s return-shape comment still omits `unmeasured`.
- **BR-31** not-addressed: `doctor.lua:95` still detects reuse only via `etime`; `cb < ca` is not itself rejected.
- **BR-32** not-addressed: `doctor/fixtures/perf_capture.txt` still carries 0 `/Users/` paths only by the truncation accident; no selection/redaction rule is recorded next to it.
- **BR-37** not-addressed: `perf_test.sh:54`'s fallback is the **live** path here — `mktemp -d` fails in this shell (`mkdtemp failed … Operation not permitted`), so `stub="$repo/.perf-test-stub.$$"`, and `.perf-test-stub.*` is absent from `.gitignore`.
- **BR-41** not-addressed: `hoprttcmd.go:148-152` still tests `args[0]` against exactly two tokens; `pair hoprtt --spawn 5 -- x` runs `pipeRTT(500)` and exits 0.
- **BR-42** not-addressed: the plan's Core concepts table is unchanged — `pair-hoprtt` still points at `cmd/pair-hoprtt/main.go`, and `parse_samples` / `parse_duration` / `verdict`+`FRAME_MS` have no rows.
- **NEW** `hoprtt.go:35-45` vs `:146`: `Run` receives injected `stdout`/`stderr` but `child()` reads `os.Stdin` and writes `os.Stdout` directly, and the dispatcher registers `hoprtt` as `Streaming: true` while `main.go:91` passes no stdin. See findings block.

## 5. Test coverage notes

The Lua side is well covered and driven from a recorded fixture rather than literals — that is the right shape and it holds. The Go side has a real positive control and a real reachability test. The gap is entirely in shell: `perf_test.sh` has **no** `PAIR_PERF_BUDGET` run, so the shed order I verified by hand is pinned by nothing (`#210` records this as BR-25's residual), and no `PAIR_PERF_WINDOW` case, so the empty-`## swap_rate` section above is invisible to the suite. Both fall out for free once the stub dir becomes a recorded-output fake per BR-38's rule — one fake unlocks the budget assertion, the grammar assertion, the degraded-collector assertion, and mutation-detection, in one pass.

## 6. Architectural notes

- **ARCH-DRY** — flag, BR-27: four re-implementations of the `collect()` ladder plus the `n/a (budget reserved for probes; … skipped)` string at four sites.
- **ARCH-PURE** — pass, and this is the milestone's strongest axis. Five pure functions in `doctor.lua`, tested with no IO but a fixture read; the shell shell stays orchestration. One nit is the new finding below.
- **ARCH-PURPOSE** — flag, BR-39: the enumeration was written and then half-swept. A finding that names its own class and gets its named instance fixed is the exact pattern this principle exists to catch.
- **ARCH-MOCK** — flag, BR-38: `perf.sh` shells out to five system tools and the only doubles are stateless `exit 1` stubs. The recorded fixture is a stateful fake for the *join* but nothing fakes the *collection*, which is why the suite cannot run at all when the tools are denied.
- **ARCH-CONSTRAINTS** — flag, BR-34, disposed to `#210` by operator decision. Noted, not re-litigated.
- **ARCH-SECURE** — flag, BR-17 + BR-32: the report leaves the machine carrying full executable paths, against the plan's own stated contract, and the committed fixture is a real host capture with no recorded redaction rule.
- **ARCH-ORDER** — flag, BR-38 again at its highest-leverage lens: `perf_test.sh` can observe exactly one interleaving (whatever the ambient host is doing) with no seam to inject another. `delta`'s bucket enumeration and the budget's shed-state machine both read cleanly off the code and pass.

## 7. Plan revision recommendations

One `## Revisions` entry, dated 2026-09-07, covering all of:
1. **Core concepts table rows** for `parse_samples`, `parse_duration`, `verdict`/`FRAME_MS` (all PURE, `nvim/doctor.lua`, new), and correct the `pair-hoprtt` row's path from `cmd/pair-hoprtt/main.go` to `cmd/internal/hoprttcmd/hoprtt.go` — the prose reversal landed but the table still claims the reversed design.
2. **M2.6's mechanism**: strike "via `artifactpath`" and name `pair_data_dir()` + `pair_tag()`, with the file:line and the reachability check the rule now demands.
3. **M2.3's `verdict` prose**: it still describes the symmetric 16 ms threshold the code replaced with the asymmetric one.
4. **A boundary note** that `4f9365b3` (M2.2b, the pure join) landed inside the M1 window, so M2's base is the M1 close and not the branch point.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan M2.6:389 still says "via artifactpath"; that package has no CLI surface Lua can reach. The GO_BINS/pair- prefix half is overtaken by the subcommand reversal.
  - id: BR-5
    disposition: not-addressed
    note: |
      Reproduced with ps denied: '### cputime' and '### procs' render as empty sections; the '|| say n/a' arms at perf.sh:146,148 are unreachable because the pipeline exits with awk's status.
  - id: BR-16
    disposition: not-addressed
    note: |
      probe_line's awk at perf.sh:227 still prints only $1/$2; the sample count is discarded.
  - id: BR-17
    disposition: not-addressed
    note: |
      sample() at perf.sh:130 still takes $4 of a full comm path; the report carries home-directory paths against the plan's own ARCH-SECURE contract.
  - id: BR-18
    disposition: not-addressed
    note: |
      Sharper evidence: PAIR_PERF_WINDOW=0 and =abc both render the entire '## swap_rate' section as nothing at all -- no key, no n/a, no reason. No shell injection; quoting is correct.
  - id: BR-19
    disposition: not-addressed
    note: |
      Still no record that 4f9365b3 landed inside the M1 window.
  - id: BR-20
    disposition: not-addressed
    note: |
      grep -c perf doctor/README.md = 0; it still describes doctor/ as the drift tool only.
  - id: BR-21
    disposition: not-addressed
    note: |
      Issue Plan M1 row unticked and still naming cmd/hoprtt; no 2026-09-07 Log entry, no M1.4/M1.5 evidence.
  - id: BR-25
    disposition: addressed
    note: |
      Verified live: PAIR_PERF_BUDGET=2 sheds top, iostat and sample_b while pipe_hop_ms and fork_exec_ms survive. Behaviour correct but pinned by no test; that residual is tracked in issue 210.
  - id: BR-26
    disposition: not-addressed
    note: |
      perf_test.sh:23's negative character-class arm is unchanged.
  - id: BR-27
    disposition: not-addressed
    note: |
      collect() is still re-implemented at perf.sh:101-120, :185-196, :218-229 and skipped by sample()/cputimes() -- structurally why BR-5's residual survives.
  - id: BR-28
    disposition: not-addressed
    note: |
      perf.sh:179 is still shell arithmetic against the file's own rule 2, and WINDOW=0 now silently erases the whole section instead of printing inf.
  - id: BR-29
    disposition: not-addressed
    note: |
      perf.sh:172-173 still reports a vm_stat that exists but fails as "vm_stat unavailable".
  - id: BR-30
    disposition: not-addressed
    note: |
      doctor.lua:157 still says nvim/fixtures/; doctor.lua:75's return-shape list still omits unmeasured.
  - id: BR-31
    disposition: not-addressed
    note: |
      doctor.lua:95 still gates reuse on etime alone; cb < ca is not rejected at the point of derivation.
  - id: BR-32
    disposition: not-addressed
    note: |
      Fixture still carries 0 /Users/ paths only by the truncation accident; no selection or redaction rule recorded beside it.
  - id: BR-34
    disposition: addressed
    note: |
      Split to issue 210 by explicit operator decision and filed with a Spec, a mechanism and a Done-when. Treated as settled, not re-litigated at this gate.
  - id: BR-37
    disposition: not-addressed
    note: |
      Confirmed the fallback is the LIVE path here -- mktemp -d fails with "Operation not permitted" in this shell -- and .perf-test-stub.* is still absent from .gitignore.
  - id: BR-38
    disposition: not-addressed
    note: |
      The >=10-row guard swapped a vacuous pass for a hard failure: sh doctor/perf_test.sh exits 1 here, and test-perf-capture is in the make test chain. The rule (control the environment) is still unapplied.
  - id: BR-39
    disposition: not-addressed
    note: |
      swap_rate is fixed and verified, but 2 of the 4 sites the finding's own enumeration named are live: the sample blocks' silence, and delta dividing by the declared window while the measured at_s delta is emitted and discarded.
  - id: BR-41
    disposition: not-addressed
    note: |
      hoprttcmd.go:148-152 unchanged; an unrecognised argv token still falls through to pipeRTT(500) and exits 0.
  - id: BR-42
    disposition: not-addressed
    note: |
      Core concepts table unchanged: pair-hoprtt still points at cmd/pair-hoprtt/main.go, and parse_samples / parse_duration / verdict still have no rows.
findings:
  - id: new
    severity: Minor
    family: injected-io-seam-bypassed
    title: |
      hoprttcmd.Run takes injected writers but child() reads os.Stdin and writes os.Stdout, and the package is registered as a streaming subcommand with no stdin
    detail: |
      Run (hoprtt.go:146) accepts stdout/stderr and report() honours them, but
      child() (hoprtt.go:35-45) touches os.Stdin/os.Stdout directly. The
      dispatcher registers hoprtt with Streaming: true (dispatcher.go:63) while
      cmd/pair-go/main.go:91 passes no stdin, so the signature says "reads no
      stdin" and the code contradicts it. This is a newly-introduced internal
      package that downstream M2 work will call, so the surface is worth
      settling now: a future test adding {"-child"} to
      TestUsageErrorsRatherThanSilentlyMeasuringNothing's table would block on
      the real terminal rather than the injected buffer. It works today only
      because the child is always a re-exec'd process. Take a stdin io.Reader
      like every sibling Run, or document that -child is process-level only and
      keep it out of the injected-writer path (ARCH-PURE).
```
