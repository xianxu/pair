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
