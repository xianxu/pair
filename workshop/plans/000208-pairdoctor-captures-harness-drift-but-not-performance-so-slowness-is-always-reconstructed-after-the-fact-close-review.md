# Boundary Review — pair#208 (whole-issue close)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | whole-issue close |
| milestone | — |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..92a397f1591473589d553b0c82ef4559d02060b7 |
| command | sdlc close --issue 208 |
| reviewer | claude |
| timestamp | 2026-09-07T14:54:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The product is in good shape: I ran the real `:PairDoctor` path end to end and read the payload it produces, and every claim it makes is now backed by something measured — `redraw n/a (no UI attached…)` forcing `editor: unknown`, the pid join reaching the prompt, and the whole report degrading to `n/a (<reason>)` under total tool denial (verified in a sandbox where `ps`/`sysctl`/`top`/`iostat` all fail). What blocks a clean SHIP is not the product but the instrument: the closing commit `92a397f1` landed **three of its six behaviour changes with no test that fails without them** — I reverted each in a scratch copy and the full suite stayed green — in the very commit that closed BR-52, whose stated rule is that a new surface is covered only when every reachable branch is executed. Separately, `sh doctor/perf_test.sh` **fails** in this sandboxed agent shell (2 failures), and `test-perf-capture` is in `make test` (`Makefile.local:120`), so `make test` is red for the reader class `doctor/perf.sh:11` names as its target; that is BR-38 still open, and its ≥10-row remedy converted a vacuous pass into a hard failure without controlling the environment.

### 1. Strengths

- **The `n/a` rule finally holds under real denial.** `doctor/perf.sh:51-61` plus the explicit ladders render `host_cores=n/a (sysctl failed)`, `process_count=n/a (ps failed)`, `### procs → n/a (ps failed or is unavailable)` — no fabricated `0`, no bare `key=`. Reproduced live, not inferred. BR-5 is genuinely closed.
- **`tests/pair-doctor-test.sh:164-185` is a real pin, not a restatement.** Mutation-verified twice: neutering the clear at `nvim/init.lua:4255` takes "a successful send consumes the note" red; forcing the join branch off at `nvim/init.lua:4180` takes both rate assertions red.
- **`nvim/submission_test.lua:31-50` closes the mutation gap it was written for** — reverting `submission.lua:79` to `return true` exits 1, where before it left both suites green.
- **The asymmetric verdict holds end-to-end.** The real headless payload reads `editor: unknown (input 0.0ms, redraw n/a (no UI attached; redraw is a no-op), completion 0.4ms…)` — a missing leg yields `unknown`, which is exactly what `doctor/SKILL.md` requires before anyone excludes `#201`/`#203`.
- **Docs are ahead of the code, not behind it.** `atlas/index.md`, `doctor/README.md:70-88`, `README.md:609-616` and `doctor/SKILL.md`'s new procedure all state the two user-visible consequences (the draft buffer is the note; a successful send clears it), and `tests/perf-key-conformance-test.sh` closes the shell↔Lua key restatement against a real `perf.sh` run.

### 2. Critical findings

None.

### 3. Important findings

**I-1 — three of the closing commit's six behaviour changes are unpinned (family `untested-shell-surface`, 7th).** Per the family protocol: do not fix these three instances. The rule that covers all seven: *a behaviour change lands with a test that fails without it, and the check is mechanical — revert the hunk in a scratch copy and confirm a suite goes red.* The enumeration for this round is the closing commit's own behaviour hunks; I ran it. Pinned: `capture_record`'s schema + `empty_dict` (`nvim/doctor_test.lua:390-402`), the consume-on-success clear, the join, `submission.lua`'s real return. **Unpinned, mutation-verified:** (a) the `has_ui()` redraw gate at `nvim/init.lua:4062-4069` — and `has_ui` already carries a test seam, `vim.g.pair_test_has_ui` (`nvim/init.lua:689-691`), so the assertion costs one line; (b) the operator note prepended to the sidecar at `nvim/init.lua:4209-4211` — no test opens the sidecar at all; (c) the JSONL-append failure notify at `nvim/init.lua:4232-4239` — no test drives a failing `pair_write_data_file`. Prevalence 7/7 with BR-9, BR-25, BR-26, BR-36, BR-38, BR-52.

**I-2 — `make test` is red in a sandboxed agent shell.** This is BR-38 re-disposed `not-addressed`, not a new id. `doctor/perf_test.sh` reports *"perf.sh sample rows no longer match the grammar doctor.parse_samples reads"* and *"only 2 sample rows; the grammar pin validated almost nothing"*. Two causes, both in the same block: the `proc_rows -lt 10` check (`:110-113`) requires the ambient `ps` to work, and the grammar loop (`:91-104`) does not skip perf.sh's own `n/a (ps failed or is unavailable)` degradation line, so it flags a line perf.sh is *required* to emit as a shape violation. The fix the rule implies is the one BR-38 already named: a recorded-output `ps`/`top`/`vm_stat`/`iostat` fake on `PATH`, so the grammar is asserted against controlled input rather than the host.

**I-3 — the shed does not propagate to the divisor (family `failure-reported-as-measurement`, BR-39 residual).** The `WINDOW_ELAPSED` guard (`doctor/perf.sh:211-215`) closed the member it named; the last member of BR-39's own enumeration is live in two places. `at_s` is emitted in both samples and `parse_samples` has no branch that reads it (`nvim/doctor.lua:185-215`), so `nvim/init.lua:4180` divides by the **declared** `window or 2`. And `SWAP_A` is read at `perf.sh:171`, *before* `sample_a`, while `_b` is read at `:219` *after* `sample_b` — so the swap interval always exceeds `WINDOW` by the cost of two `ps` runs, which is precisely what grows on the degraded machine this tool targets.

### 4. Minor findings

- **`docs-gate` (5th, do not fix the instance):** the rule is *a comment or doc that enumerates something is a claim a test or a grep can check, and it is updated in the same change that invalidates it.* Live members: `nvim/doctor.lua:2` and `nvim/doctor_test.lua:2` both still say "no vim API here" while `capture_record` calls `vim.empty_dict` and the new test calls `vim.json`; `atlas/index.md`'s enumeration of the sidecar's contents ("compact half, joined per-process rates, and both raw `ps` samples") was not extended when the operator note was prepended to it in the same commit.
- `PAIR_PERF_WINDOW=0` makes the entire `## swap_rate` section **vanish** (awk aborts with "division by zero" on stderr), not yield `inf` — and `perf_test.sh:128` and `:156` drive perf.sh with exactly that value while asserting nothing about it (BR-28/BR-18).
- `perf.sh:216-217` still renders "vm_stat unavailable" when the *first* read fails, though `:221` correctly distinguishes the second (BR-29).
- `doctor.lua:91` still detects a reused pid only via `etime`; `cb < ca` at `:98` is unguarded (BR-31).
- `probe_line` (`perf.sh:262-273`) still discards hoprtt's sample count `$4` (BR-16).
- `hoprtt.go:148-177` still falls through to `pipeRTT(500)` on an unrecognised argument (BR-41); `child()` still reads `os.Stdin` while the package is registered `Streaming: true` with no stdin (BR-43).
- `perf_test.sh:59`'s `$repo/.perf-test-stub.$$` fallback is still ungitignored (BR-37); `:23`'s stray-line arm still passes any line containing a non-digit (BR-26).
- Four shapes of `collect()`'s ladder remain (`perf.sh:101-120`, `:229-240`, `:262-273`, `emit_sample`); `na_for` deduped the key lists only (BR-27).
- `#210` records BR-34/25/38/39 but still not the M2.6 deferrals (no window-length field, uncapped `perf-captures.jsonl`) that the plan says were deferred to it (BR-49).

### 5. Test coverage notes

The pure Lua suite is genuinely strong — `nvim/doctor_test.lua` drives `delta` from the recorded fixture, pins the budget-shed path, the `%w` probe-key defect, `safe_comm`, and the headline/truncation design. The Go side satisfies the Spec's known-quantity requirement (`hoprtt_test.go:64-75` bounds `/usr/bin/true` at 15 ms against the 18.7 ms harness bug). The gap is uniformly at the *glue-change* boundary: every time this issue has added a behaviour to `init.lua` without a matching mutation check, the behaviour has been correct and the next round has found it unprotected. `tests/pair-doctor-test.sh` now has the seams to close that (injectable runner, `_G.send_generated_prompt`, `vim.g.pair_test_has_ui`, `PAIR_DATA_DIR`) — the sidecar assertion and the UI-attached redraw assertion are each a few lines against seams that already exist.

### 6. Architectural notes

- **ARCH-DRY** — flag (minor): `na_for` and the single-sourced `PROBE_KEYS`/`BASELINES` with a live conformance test are the right shape, but `collect()`'s ladder is still re-implemented four times (BR-27).
- **ARCH-PURE** — pass with a nit: `doctor.lua` remains string-in/string-out and the join lives there rather than in shell, which is the design's best decision. `vim.empty_dict` is the first `vim` reference in the "pure" module; it is not IO, so purity holds — only the header's claim is now false.
- **ARCH-PURPOSE** — flag: I-1. A finding whose rule is "every reachable branch is executed" was closed by a commit that added three unexecuted branches. The class, not the three instances, is the deliverable.
- **ARCH-MOCK** — flag: I-2. `perf.sh` depends on five external binaries; the fake set is two stateless `exit 1` stubs plus two recorded-output `ps` fakes, and the grammar assertion runs against the host instead. Production flow and test flow do not share a boundary here.
- **ARCH-CONSTRAINTS** — pass with the known gap: the 6 s budget is declared, shed in reverse value order, and asserted (`perf_test.sh:34-36`); no stage is bounded, which is `#210` by operator decision. The nvim side is async with a 30 s kill, so the editor is never blocked.
- **ARCH-SECURE** — pass: `comm` is filtered at the single point of emission (`perf.sh:150-157`) rather than at one consumer, which also retires BR-32's mechanism — I confirmed zero `/Users/`/`/home/` paths can reach a re-capture, because only basenames are emitted.
- **ARCH-ORDER** — pass: `capture_running` is a single flag with one reset path, the throwing-spawn reset is tested, and the buffer-changed-mid-flight interleaving is driven by the injected runner. The remaining nondeterminism (two sessions sharing an untagged `pair_data_dir`) is named in the code comment rather than left implicit.

### 7. Plan revision recommendations

- **Core concepts table (`workshop/plans/000208-pairdoctor-perf-capture-plan.md:74-81`).** Revisions item 3 and the subcommand entry disclaim the table in prose, but the table itself still reads `| pair-hoprtt (pipe probe) | cmd/pair-hoprtt/main.go | new |` — a path that does not exist. Replace the row with `hoprttcmd.Run` at `cmd/internal/hoprttcmd/hoprtt.go`, and fold the entities listed in Revisions item 3 into the table so the greppable contract matches the code rather than being corrected several hundred lines below it.
- **New `## Revisions` entry — "the perf-capture suite does not run where the capture does".** Record that `doctor/perf_test.sh` asserts the sample grammar against the ambient host, that this makes `make test` fail wherever `ps` is denied, and that the recorded-fixture fake is the remedy — so the plan stops implying the shape is pinned everywhere it is claimed to be.
- **`#210`**: add the two M2.6 deferrals (no window-length field in the JSONL row; `perf-captures.jsonl` is uncapped) to its "deferred with it" table, so they survive the plan's archival.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      Verified live with ps/sysctl/top/iostat all denied — every key renders n/a with a reason; no fabricated process_count=0, no bare load=.
  - id: BR-16
    disposition: not-addressed
    note: |
      probe_line (perf.sh:262-273) still reads only $1/$2; hoprtt's sample count $4 is discarded.
  - id: BR-17
    disposition: addressed
    note: |
      sample() strips the first three fields and basenames on '/'; perf_test.sh:164 pins "Google Chrome" surviving and :160 pins no /Applications/ reaching the report.
  - id: BR-18
    disposition: not-addressed
    note: |
      No validation and no PAIR_PERF mention in README/atlas/SKILL; measured — WINDOW=0 makes the whole swap_rate section vanish with awk "division by zero".
  - id: BR-19
    disposition: not-addressed
    note: |
      grep for 4f9365b3 across workshop/ and atlas/ hits only prior gate-ledger rounds; neither the issue nor the plan records it.
  - id: BR-20
    disposition: addressed
    note: |
      doctor/README.md:70-88 now links perf.sh and documents the note/clear semantics.
  - id: BR-21
    disposition: addressed
    note: |
      Issue Plan M1 is ticked with the design-reversal note; the M1 close line carries evidence and the zellij number (14.385 ms) is in the M2 baseline table.
  - id: BR-26
    disposition: not-addressed
    note: |
      perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
  - id: BR-27
    disposition: not-addressed
    note: |
      na_for deduped only the key lists; the top block, disk block, probe_line and emit_sample each still re-implement the availability/failure/empty ladder.
  - id: BR-28
    disposition: not-addressed
    note: |
      Sharper than reported — at WINDOW=0 awk aborts and all three swap keys VANISH rather than yielding inf, and perf_test.sh:128/:156 drive that exact value.
  - id: BR-29
    disposition: not-addressed
    note: |
      perf.sh:216-217 still says "vm_stat unavailable" when the first read fails; only the second read (:221) distinguishes failure from absence.
  - id: BR-30
    disposition: addressed
    note: |
      doctor.lua:180 now names doctor/fixtures/ and :75-76 enumerates unmeasured.
  - id: BR-31
    disposition: not-addressed
    note: |
      doctor.lua:91 unchanged; cb < ca at :98 is still unguarded, so an unparseable etime yields a negative cpu_pct.
  - id: BR-32
    disposition: addressed
    note: |
      The finding's own alternative was taken — sample() basenames comm at the emitter, so no path can reach a re-capture; fixture verified free of /Users/ and /home/.
  - id: BR-37
    disposition: not-addressed
    note: |
      perf_test.sh:59 unchanged and .gitignore has no .perf-test-stub entry.
  - id: BR-38
    disposition: not-addressed
    note: |
      Reproduced — sh doctor/perf_test.sh exits 1 in this sandboxed shell (grammar violation on perf.sh's own n/a line, plus "only 2 sample rows"), and test-perf-capture is in make test.
  - id: BR-39
    disposition: not-addressed
    note: |
      The shed member is fixed (WINDOW_ELAPSED). The declared-vs-measured member is live — at_s is discarded by parse_samples, and SWAP_A is read before sample_a while _b is read after sample_b.
  - id: BR-41
    disposition: not-addressed
    note: |
      hoprtt.go:148-177 unchanged; an unrecognised argument still falls through to pipeRTT(500).
  - id: BR-42
    disposition: addressed
    note: |
      Plan Revisions item 3 enumerates the missing entities and a separate entry reverses every cmd/pair-hoprtt reference; the stale TABLE itself is carried as a plan-revision recommendation rather than re-raised.
  - id: BR-43
    disposition: not-addressed
    note: |
      hoprtt.go:35-45 still touches os.Stdin/os.Stdout while dispatcher.go:63 registers hoprtt Streaming with no stdin from main.go.
  - id: BR-49
    disposition: not-addressed
    note: |
      Issue 210 now records BR-34/25/38/39 but still not the missing window-length field or the uncapped perf-captures.jsonl.
  - id: BR-52
    disposition: addressed
    note: |
      All three prescribed members mutation-verified red — the clear, the join, and submission.lua's return. Residual unexecuted branches roll into the new coverage finding.
  - id: BR-53
    disposition: not-addressed
    note: |
      Behaviour is correct and reachable (payload renders redraw n/a, verdict unknown), but removing the has_ui() gate in a scratch copy leaves every suite green — and vim.g.pair_test_has_ui already exists as the seam.
  - id: BR-54
    disposition: not-addressed
    note: |
      The note does reach the sidecar, but no test opens the sidecar — reverting the prepend leaves pair-doctor-test.sh and doctor_test.lua both green.
  - id: BR-55
    disposition: not-addressed
    note: |
      The notify is present but no test drives a failing pair_write_data_file, so the branch is never executed.
  - id: BR-56
    disposition: addressed
    note: |
      Mutation-verified — deleting the vim.empty_dict line takes doctor_test.lua red on "an empty probes set encodes as an object, not []".
  - id: BR-57
    disposition: addressed
    note: |
      na_for at perf.sh:206 collapses the three loops; the n/a key assertions under tool denial still pass.
  - id: BR-58
    disposition: addressed
    note: |
      Answered as an explicit accepted trade-off with its reasoning recorded at nvim/init.lua:4048-4053, which is a legitimate resolution for this severity.
findings:
  - id: new
    severity: Important
    family: untested-shell-surface
    title: |
      Three of the closing commit's six behaviour changes are unpinned — the same rule the commit closed BR-52 on
    detail: |
      This is the 7th finding in family `untested-shell-surface`. Do NOT fix these three
      instances. The rule covering all seven: a behaviour change lands with a test that
      fails without it, and the check is mechanical — revert the hunk in a scratch copy
      and confirm a suite goes red. The enumeration is the closing commit's own
      behaviour hunks, and I ran it. Pinned (verified red on revert): capture_record's
      schema + empty_dict, the consume-on-success clear, the parse_samples->delta join,
      submission.lua's real return. Unpinned (verified green on revert): the has_ui()
      redraw gate at nvim/init.lua:4062-4069, the operator note prepended to the sidecar
      at nvim/init.lua:4209-4211, and the JSONL-append failure notify at
      nvim/init.lua:4232-4239. The seams to close all three already exist —
      vim.g.pair_test_has_ui (nvim/init.lua:689), PAIR_DATA_DIR in the test harness, and
      the injected capture runner. Prevalence 7/7 with BR-9, BR-25, BR-26, BR-36, BR-38,
      BR-52; the commit that closed BR-52 is itself the newest member, which is the
      signal that the rule was fixed at the instance level and never as a class.
  - id: new
    severity: Minor
    family: docs-gate
    title: |
      doctor.lua and doctor_test.lua still claim "no vim API", and atlas's sidecar enumeration omits the note added in the same commit
    detail: |
      This is the 5th finding in family `docs-gate`. Do NOT fix these instances — the
      rule: a comment or doc that ENUMERATES something is a claim a grep can check, and
      it is updated in the same change that invalidates it; the enumeration is
      mechanical (grep the changed symbol across *.md and the module headers that
      describe it). Live members: nvim/doctor.lua:2 and nvim/doctor_test.lua:2 both say
      "no vim API here" while capture_record now calls vim.empty_dict and the new test
      calls vim.json; atlas/index.md describes the sidecar as "compact half, joined
      per-process rates, and both raw ps samples" while the same commit prepended the
      operator note to it — under a paragraph whose own subject is the invariant that
      nothing of value exists only in the prompt. Prevalence 5/5 with BR-20, BR-30,
      BR-47, BR-51.
```

---

## Re-review — 2026-09-07T15:18:04-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | whole-issue close |
| milestone | — |
| window | df283abefd229b0b81b2f4bfa98a26310f75316a..307b4e74bc56d88dab6296951db72901ef81f619 |
| command | sdlc close --issue 208 |
| reviewer | claude |
| timestamp | 2026-09-07T15:18:04-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've verified the window: ran the full suite, and mutation-tested every claimed fix.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The closing commit's four claims all hold under mechanical check. I reverted each of the three behaviours BR-59 named — the `has_ui()` redraw gate, the operator note prepended to the sidecar, the JSONL-append notify — in a scratch copy of the tree, and each took `tests/pair-doctor-test.sh` red (2, 2 and 1 failures); and the BR-38 grammar pin now supplies its own `ps`, so it validates 12 rows in this sandboxed shell where `/bin/ps` is denied, and goes red when I change `sample()`'s tab separator to a pipe. Full `make test` is green except `cmd/pair-go`'s `TestPublicPairCommandFamiliesIgnoreCouchStore`, which fails on `fork/exec /bin/ps: operation not permitted` — pre-existing, out of window, environmental. What blocks SHIP if anything: `perf.sh` contains five field-extraction parsers over external-tool stdout and the closing commit built a recorded-output fake for exactly one of them, so three separate one-token mutations (cputime row separator, `top`'s `$(NF-1)`→`$2`, `iostat`'s column order) each leave the entire suite green while producing a wrong-but-plausible headline reading — the defect class this issue exists to eliminate, one enumeration row short.

**1. Strengths**

- `doctor/perf_test.sh:91-133` — the right fix for BR-38, not the cheap one. A recorded-output `ps` gives the same assertion the same 12 rows everywhere, which simultaneously un-breaks `make test` for the sandboxed agent `perf.sh` names as its reader and makes the pin non-vacuous. Verified both directions: green here where `ps` is denied, red on a separator mutation.
- `tests/pair-doctor-test.sh:191-207` — the `has_ui()` gate is driven through `vim.g.pair_test_has_ui`, the seam that already existed at `nvim/init.lua:689`, rather than a second UI check (ARCH-DRY). Both polarities are asserted: `n/a` + `verdict unknown` headless, and actually-timed with the flag set.
- `tests/pair-doctor-test.sh:212-226` — the sidecar test opens the file the payload names and reads the operator's own note back out of it. That pins the invariant (`nothing of value exists only in the prompt`) rather than the string that implements it.
- `tests/perf-key-conformance-test.sh` — the shell restatement of the single-sourced key set is enforced against a real `perf.sh` run, and the comment at `:29-31` records why `io.stdout` and not `print` (a vacuous pass). Renaming `window_seconds` in `perf.sh` takes `perf_test.sh:28` red; I checked.
- `cmd/internal/hoprttcmd/hoprtt_test.go:64-74` — the positive control the issue's Log demanded, with the band's width justified against the suite's own spawn load rather than picked to pass.

**2. Critical findings** — none.

**3. Important findings**

- `doctor/perf_test.sh:61` — **8th in family `untested-shell-surface`.** Per the family rule I am not asking for this instance to be fixed. The rule: *a shell parser over external-tool output is pinned by a recorded-output fake of that tool; an `exit 1` stub pins only the n/a ladder and asserts nothing about field extraction.* The enumeration is mechanical — every `awk` in `perf.sh` that indexes a field of a tool's stdout — and it has five members, of which the closing commit's fake covers one:

  | parser | pinned by a recorded fake? |
  |---|---|
  | `sample()` procs row (`perf.sh:150`) | yes — `perf_test.sh:98` and `:165` |
  | `cputimes()` row (`perf.sh:162`) | no |
  | `top` CPU-usage line (`perf.sh:115`) | no |
  | `top` WindowServer row (`perf.sh:117`) | no |
  | `iostat` columns (`perf.sh:238`) | no |

  Mutation-verified, each independently, full suite green after: changing `cputimes()`'s `\t` to a space takes the join from `rates=5 unmeasured=0` to `rates=0 unmeasured=5` — the per-process rates are SKILL.md's step 4 and the operator's own addition to the Spec, and they silently disappear; `$(NF-1)`→`$2` makes `cpu_idle_pct` report the *user* percentage under the idle key, a HEADLINE_KEY that reaches the prompt and that SKILL.md instructs the reader on; permuting `iostat`'s `$1,$2,$3` mislabels all three disk values. The mechanism is already built at `perf_test.sh:98-111` — the sweep is to add the same recorded-output fake for `top`, `iostat` and `vm_stat`, and to extend the existing `ps` fake's assertions to the `### cputime` rows (ARCH-MOCK, ARCH-PURPOSE).

**4. Minor findings**

- `doctor/perf_test.sh:113` — `grows=0` is dead; it is unconditionally reassigned at `:130`.
- 13 prior Minors remain open and are disposed `not-addressed` in the block below (BR-16, BR-18, BR-19, BR-26, BR-27, BR-28, BR-29, BR-31, BR-37, BR-41, BR-43, BR-49, BR-60), plus BR-39 at Important. None is a live wrong reading on a default-configured capture.

**5. Test coverage notes**

Coverage is now genuinely strong on the wiring layer — the injected `capture_runner` drives the buffer-changed-mid-flight interleaving, the throwing spawn, the failing send, the successful send, and now the failing JSONL append. The residual gap is entirely on the *producer* side: the tests control `ps` and nothing else, so every field index in `perf.sh` outside `sample()` is unverified. That is the finding above. Secondarily, `PAIR_PERF_WINDOW=0` is driven twice (`:110`, `:148`) and nothing asserts what happens to the swap block there — which is why BR-28's key-vanishing went unobserved by the suite despite the value being exercised.

**6. Architectural notes**

- **ARCH-DRY** — pass. `na_for` collapsed the three n/a loops; `has_ui()` was reused rather than duplicated; the key set is one declaration with a conformance test on the shell restatement. Residual: BR-27 (four re-implementations of `collect()`'s ladder), carried.
- **ARCH-PURE** — pass with residuals. The pid join lives in `doctor.lua` and runs under `nvim -l`. BR-43 (`child()` bypasses the injected writers) and BR-60 (the `no vim API here` headers are now false — `capture_record` calls `vim.empty_dict`) are both purity drift, carried.
- **ARCH-PURPOSE** — shadow-sweep on the single-sourced key set: `headline`, `capture_record.baselines` and `probes_from` all derive from `PROBE_KEYS`/`HEADLINE_KEYS`; `perf.sh` restates and is *enforced* by `tests/perf-key-conformance-test.sh`. All four consumers accounted for. Flagged on a different axis: the parse-contract enumeration above.
- **ARCH-MOCK** — flagged (the finding). `ps` has a recorded-output fake; `top`/`iostat`/`vm_stat` have stateless `exit 1` doubles only, so production flow and test flow share the boundary for one dependency out of four.
- **ARCH-CONSTRAINTS** — pass. The budget is asserted, not declared (`perf_test.sh:33-36`), the probes are reserved a slice, and the nvim side bounds the child at 30 s. `#210` carries the declared known gap.
- **ARCH-SECURE** — pass. `comm` is filtered at the single point of emission and pinned for a control byte, a full path and an embedded space (`perf_test.sh:158-186`); the committed fixture is basenamed so no host paths ship.
- **ARCH-ORDER** — pass. `capture_running` is a two-state flag whose reset has exactly one path, the spawn is `pcall`'d so a throw cannot strand it, and the tests own the ordering through the injected runner rather than observing whichever interleaving the machine produced.

**7. Plan revision recommendations**

One entry, carried forward from round 14's disposition of BR-42 rather than re-raised: the Core concepts table at `workshop/plans/000208-pairdoctor-perf-capture-plan.md:79` still lists `pair-hoprtt (pipe probe) | cmd/pair-hoprtt/main.go | new`, and the bullets at `:98-114` still describe the `GO_BINS` entry and `$PAIR_HOME/bin/pair-hoprtt` resolution. The Revisions entry at `:600` reverses all of it; the table and bullets should be edited in place to name `cmd/internal/hoprttcmd` / `pair hoprtt`, so the table stops claiming what the code does not deliver.

```findings
dispose:
  - id: BR-16
    disposition: not-addressed
    note: |
      probe_line (perf.sh:262-273) still reads only $1/$2; hoprtt's sample count $4 is discarded.
  - id: BR-18
    disposition: not-addressed
    note: |
      No validation; grep for PAIR_PERF across README.md, atlas/, doctor/README.md and doctor/SKILL.md returns nothing.
  - id: BR-19
    disposition: not-addressed
    note: |
      grep 4f9365b3 across workshop/ and atlas/ still hits only the gate ledger's own rounds.
  - id: BR-26
    disposition: not-addressed
    note: |
      perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
  - id: BR-27
    disposition: not-addressed
    note: |
      The top block, disk block, probe_line and emit_sample each still re-implement collect()'s availability/failure/empty ladder.
  - id: BR-28
    disposition: not-addressed
    note: |
      Reproduced at HEAD - PAIR_PERF_WINDOW=0 prints "awk: division by zero" and all three swap keys vanish; swapins_per_s is a HEADLINE_KEY, so a headline row is dropped, not just a value lost.
  - id: BR-29
    disposition: not-addressed
    note: |
      perf.sh:216-217 still says "vm_stat unavailable" when the first read fails.
  - id: BR-31
    disposition: not-addressed
    note: |
      doctor.lua:91 unchanged; cb < ca at :98 is still unguarded, so an unparseable etime yields a negative cpu_pct.
  - id: BR-37
    disposition: not-addressed
    note: |
      perf_test.sh:59 unchanged and .gitignore has no .perf-test-stub entry; note the three newer fakes (:98, :141, :165) use $TMPDIR with no worktree fallback, so the file is now inconsistent with itself.
  - id: BR-38
    disposition: addressed
    note: |
      Mutation-verified - the controlled ps at perf_test.sh:98-111 validates 12 rows in this shell where /bin/ps is denied, and a tab-to-pipe change in sample() takes it red.
  - id: BR-39
    disposition: not-addressed
    note: |
      The shed member stays fixed. The declared-vs-measured member is live - parse_samples reads window_seconds and discards at_s, so delta divides by the DECLARED window on exactly the machine where sleep 2 does not take 2s.
  - id: BR-41
    disposition: not-addressed
    note: |
      hoprtt.go:146-152 unchanged; an unrecognised argument still falls through to pipeRTT(500) and exits 0.
  - id: BR-43
    disposition: not-addressed
    note: |
      hoprtt.go:35-45 still touches os.Stdin/os.Stdout while dispatcher.go:63 registers hoprtt Streaming and main.go:90 passes no stdin.
  - id: BR-49
    disposition: not-addressed
    note: |
      Re-read issue 210 at HEAD - it records BR-34/25/38/39 and still neither the missing window-length field nor the uncapped perf-captures.jsonl.
  - id: BR-53
    disposition: addressed
    note: |
      Mutation-verified - replacing has_ui() with `true` at nvim/init.lua:4060 takes pair-doctor-test.sh red on both the n/a render and the unknown verdict.
  - id: BR-54
    disposition: addressed
    note: |
      Mutation-verified - dropping the note prefix at nvim/init.lua:4210 takes the two sidecar-content assertions red; the test opens the file the payload names.
  - id: BR-55
    disposition: addressed
    note: |
      Mutation-verified - replacing the `if not pair_write_data_file(...)` guard with a bare call takes the notify assertion red.
  - id: BR-59
    disposition: addressed
    note: |
      All three members reverted independently in a scratch tree; each took pair-doctor-test.sh red (2, 2 and 1 failures). The rule was applied as a class, not per-site.
  - id: BR-60
    disposition: not-addressed
    note: |
      nvim/doctor.lua:2 and nvim/doctor_test.lua:3 still claim no vim API while :293 calls vim.empty_dict; atlas/index.md:39 still enumerates the sidecar without the note, and doctor/SKILL.md's "compact report, joined per-process rates, and both raw ps samples" is a third member of the same enumeration.
findings:
  - id: new
    severity: Important
    family: untested-shell-surface
    title: |
      Four of perf.sh's five external-tool parsers are pinned only by exit-1 stubs, so a wrong field index ships as a plausible headline reading
    detail: |
      This is the 8th finding in family `untested-shell-surface`. Do NOT fix this
      instance. The rule covering it: a shell parser over external-tool output is
      pinned by a RECORDED-OUTPUT fake of that tool - an `exit 1` stub pins the n/a
      ladder and asserts nothing about field extraction. The enumeration is
      mechanical, every awk in perf.sh that indexes a tool's stdout, and it has five
      members: sample()'s procs row (perf.sh:150, pinned at perf_test.sh:98 and
      :165), cputimes()'s row (perf.sh:162, unpinned), top's CPU-usage line
      (perf.sh:115, unpinned), top's WindowServer row (perf.sh:117, unpinned), and
      iostat's three columns (perf.sh:238, unpinned). Mutation-verified
      independently, full suite green after each: changing cputimes()'s tab to a
      space moves the join from rates=5/unmeasured=0 to rates=0/unmeasured=5, so
      SKILL.md's step 4 and the operator's own Spec addition silently vanish;
      $(NF-1)->$2 makes cpu_idle_pct carry the USER percentage under the idle key, a
      HEADLINE_KEY the prompt carries and SKILL.md instructs the reader on; permuting
      iostat's $1,$2,$3 mislabels all three disk values. The mechanism already exists
      at perf_test.sh:98-111 - the sweep is a recorded-output fake for top, iostat
      and vm_stat plus cputime-row assertions on the existing ps fake (ARCH-MOCK,
      ARCH-PURPOSE). Prevalence 8/8 with BR-9, BR-25, BR-26, BR-36, BR-38, BR-52,
      BR-59.
```
