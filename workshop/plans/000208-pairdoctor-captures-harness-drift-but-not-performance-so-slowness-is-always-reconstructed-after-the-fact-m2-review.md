# Boundary Review — pair#208 (milestone M2)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2fb5a79e7dbf3ef4ab63d72e01989e86a6e056f9..4b4b7563fde05d2087abe573a8f9bb1c9f3d4624 |
| command | sdlc milestone-close --issue 208 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-07T13:26:16-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 delivers the shape the issue asked for — `:PairDoctor` now captures at invocation, runs asynchronously so the editor stays live, writes the full capture to a sidecar and sends a short head-first pointer, and the SKILL.md procedure is genuinely good. The async wiring, the buffer-preservation compare, and the `#211` response are all sound. What blocks SHIP is that the **editor-vs-environment discriminator does not measure the chain it names**: `time_editor()` fires a synthetic `TextChangedI` from a `:` command, so `run_completers()` hits its insert-mode guard (`nvim/init.lua:3801`) and returns before doing any work — I verified this under headless nvim (mode is `n`, and the handlers see `current_buf` = the draft, not the scratch). The Spec calls this "the single most valuable bit", and `doctor/SKILL.md` instructs the reader to exclude `#201`/`#203` on the strength of `editor: fast`. An exclusion drawn from a measurement that never ran the named chain is precisely the defect class this issue was opened to eliminate. Two further verified defects sit in the untested `init.lua` glue: the rolling-log probe keys come out as `hop_ms`/`exec_ms`/`action_ms` (they don't match perf.sh or the row's own `baselines`), and the sidecar has a fixed filename so the pointer in an earlier prompt silently resolves to a later capture.

## 1. Strengths

- **`nvim/init.lua:4028` — the async shell-out is done right.** First `vim.system` in `nvim/`, `vim.schedule`-wrapped callback, in-flight guard, and the note read at t0. The plan's named data-loss path is genuinely closed at `:4093-4101`: the buffer is consumed only when its content still equals what was read, with a notify otherwise.
- **`doctor/perf.sh:136-145` — a real fix with a real pin.** Capturing `ps` into a variable before the pipe fixes BR-5's residual (a pipeline exits with awk's status), and `doctor/perf_test.sh:111-131` puts a failing `ps` on PATH and asserts `n/a`. Reverting to `ps | awk` makes that test go red. I could not execute it (ps is denied in this shell — BR-38, demoted), but the mechanism is correct by inspection.
- **`nvim/doctor.lua:305 safe_comm` + the `\014` test** — a mutation-checked fix for an observed defect (WhatsApp's argv carrying SO), not a speculative one.
- **`nvim/doctor.lua:310-338 format_delta`** honors the n/a-is-not-zero rule where it is easiest to skip: churn is printed unconditionally, and an idle window says so explicitly instead of rendering nothing.
- **`nvim/doctor_test.lua:270-300`** pins the `#211` response as a *property* (path precedes the note, path within the first 400 bytes, payload under 1800 bytes) rather than as a golden string. That is the right instrument for a truncation-survival design.

## 2. Critical findings

**C1 — `nvim/init.lua:3970-3987`: the discriminator's `input` timing cannot observe the completion chain, and the report sells it as grounds for exclusion.**

This is the **10th finding in family `failure-reported-as-measurement`.** Earlier rounds fixed instances (BR-2, BR-5, BR-8, BR-14, BR-16, BR-23, BR-29, BR-35, BR-39). Do not fix this instance alone. **The rule, stated once:** *every emitted reading names the precondition its measurement required and asserts it at the point of measurement; when the precondition does not hold the reading is `n/a (<unmet precondition>)` and every verdict derived from it degrades to `unknown`.* Failure-of-the-tool was only one shape of the family; failure-of-the-*preconditions* is the same rule at a new arity, and it is what round 10 found.

Enumeration to sweep in one pass, both live in this window:

1. **`time_editor` (init.lua:3970).** `:PairDoctor` is a `:` command with no keymap (grepped), so mode is `n` when the timing runs. `run_completers` (init.lua:3798-3802) returns at `if mode:sub(1,1) ~= 'i' then return end`. Verified under `nvim --clean -l`: the synthetic event fires with `mode="n"` and `current_buf` = the draft, not the scratch. So the number measures `nvim_buf_set_lines` plus autocmd dispatch, not `#202`'s completion chain — which is the entity the Spec, the plan's Core concepts row, and `doctor/SKILL.md`'s exclusion table all name. Secondary: because handlers read the *current* buffer rather than `args.buf`, the plan's "runs on a SCRATCH buffer, never the draft" holds only for the text insertion — the slug-mirror autocmd (init.lua:4201) still schedules against the draft.
2. **`headline`'s allowlist (doctor.lua:346-352).** perf.sh renders a failed collector under a *different key* than its success form (`swap=n/a` vs `swapins_per_s=`, `probes=n/a` vs `pipe_hop_ms=`, `disk=n/a` vs `tps=`), so the hand-maintained `want` set drops them and the prompt omits the row with no explanation. Reproduced under `nvim -l`: a capture with `probes=n/a (pair not on PATH)` and `swap=n/a (vm_stat unavailable)` produced a headline with **no probe line and no swap line at all** — indistinguishable from a tool that has no such section. The shed path is the common trigger: a squeezed budget always renders `swap=n/a (sample window was shed…)`.

Fix sketch: (a) drive the timing with the scratch buffer current and in insert mode so the chain actually runs, or relabel the number and the `verdict` contract to what it covers and drop the `#202` claim from the plan, SKILL.md and the payload text — either is acceptable, silently keeping the claim is not; (b) make a collector's failure rendering reuse its success key (`swapins_per_s=n/a (…)`, `pipe_hop_ms=n/a (…)`), which removes the need for `headline` to know about failure keys at all, and add one test asserting that for each headline key a degraded capture still produces a line.

## 3. Important findings

**I1 — `nvim/init.lua:4079`: the rolling row's probe keys are wrong, which is exactly what BR-40's rule predicted.**

This is the **2nd finding in family `incomplete-parse-contract`.** BR-40 stated the rule: *`parse_samples` is the perf.sh→`doctor.lua` contract, so a caller that re-parses the capture with its own pattern is a second untested copy of that contract.* The prediction landed. `for k, v in (raw or ''):gmatch('(%w+_ms)=([%d%.]+)')` — Lua's `%w` excludes `_`, so the capture backtracks past the prefix. Verified under `nvim -l`: `pipe_hop_ms=0.007` → `hop_ms`, `fork_exec_ms=1.578` → `exec_ms`, `zellij_action_ms=14.385` → `action_ms`. Every row in `perf-captures.jsonl` therefore carries probe keys that match neither `perf.sh`'s emitted names nor the `baselines` table written into the same row (`doctor.lua:270`, which uses the full names). The row's stated purpose — "the baselines travel WITH the row … legible on its own" — is defeated: nothing joins `hop_ms` to `pipe_hop_ms` except a human guess. `doctor_test.lua:236` asserts `r.probes.pipe_hop_ms`, i.e. the intended key, so the test passes while the only production caller produces different ones.

Fix per the rule, not the site: no pattern in `init.lua` may read `perf.sh` output. Add `doctor.probes_from(text)` (pure, driven by `doctor/fixtures/perf_capture.txt` so the producer pins the consumer) and have `capture_record` take its result. Grep-enumeration for the sweep: every `:match`/`:gmatch` over `raw`/`compact` in `nvim/init.lua` — today exactly this one.

**I2 — no test drives the `:PairDoctor` wiring at all, and M2.4 promised one.**

This is the **5th finding in family `untested-shell-surface`.** BR-38 stated the rule for shell (*a test must CONTROL the environment its subject reads*); the same rule is unapplied one language over. Plan M2.4 says the failed-capture note-preservation "is the one behaviour a test pins" — nothing pins it. `make test-lua` covers `doctor.lua`'s pure functions only; `pair_doctor()`, `time_editor()`, the sidecar write, the JSONL append, the in-flight guard and the buffer-consume compare are all unexecuted by any suite. I1 is the proof: the one defect that shipped lives in exactly the untested wrapper, and the pure test asserting the *intended* key passed throughout. The repo already has the pattern — `tests/draft-complete-mode-test.sh` drives a headless nvim through `_G.PairDraftCompleteTest`. Enumeration to sweep: expose a `_G.PairDoctorTest` seam with an injectable capture runner, then pin (a) note preserved when the buffer changed mid-flight, (b) note preserved when the capture failed, (c) probe keys round-tripping from a recorded perf.sh capture into the JSONL row, (d) the guard resetting.

**I3 — `nvim/init.lua:4056`: the sidecar has one fixed filename, so the prompt's pointer goes stale silently.**

`perf-capture-latest.txt` is overwritten by every capture, and `pair_data_dir()` is not tag-scoped, so concurrent sessions share it. The atlas entry added in this window states the invariant as "nothing of value exists only in the prompt" — with a fixed name, after a second capture the first capture's rates and samples exist nowhere, and prompt #1 now points at capture #2's contents with nothing marking the substitution. The design's own documented recovery path makes this likely rather than exotic: SKILL.md says "If the prompt you received looks cut off, that is the known bug: read the file" — and the operator's natural response to a truncated send is to re-run `:PairDoctor`, which destroys the file the earlier prompt points at. Fix: name the file by capture (`perf-capture-<epoch>.txt`, or tag+epoch), point at that, and keep a `-latest` symlink if a stable name is wanted for humans. While there, the two `pair_data_dir()` + `mkdir` + `io.open` blocks (`:4053-4062`, `:4076-4087`) should be one small writer helper (ARCH-DRY).

**I4 — `ps`-derived text reaches the sidecar and the agent unfiltered.**

This is the **2nd finding in family `recorded-fixture-redaction`.** The slug names the site rather than the rule; **the rule that covers both:** *`ps`-derived text is filtered — basename plus control-byte strip — at the single point it is emitted, before it reaches any consumer: fixture, prompt, file, or terminal.* `safe_comm` (doctor.lua:305) applies that filter to exactly one egress (`format_delta`). The sidecar written at `init.lua:4059` concatenates `compact` and the full `raw` samples verbatim, and `doctor/SKILL.md` instructs the agent to **open that file** — so the `^N` byte the round-7 fix was written for reaches the terminal again by the new path, one file over. A newline in a `comm` path is worse: it splits a sample row and can forge a `key=value` line inside the reader's evidence. `perf.sh`'s own standalone stdout (advertised in SKILL.md) has the same gap. Fix: filter in `sample()` at the point of emission (which also makes the committed fixture safe by construction rather than by BR-32's truncation accident) — I am not re-raising BR-17 or BR-32; this is the control-byte/egress axis, and one filter closes all three.

## 4. Minor findings

- **`nvim/init.lua:3991/4025` — `capture_running` has no reset on any path but the callback.** If `vim.system` throws at spawn, or if `perf.sh` hangs (the `#210` gap, which covers only the shell side), the flag stays `true` and `:PairDoctor` is dead for the session — on exactly the struggling machine it exists for. One `pcall` around the spawn plus a `timeout` in the `vim.system` opts. Family: `inflight-guard-without-reset` (new).
- **`nvim/init.lua:4013` — the destructive consume targets `nvim_get_current_buf()`, not the draft.** 5th in family `unguarded-edge-case`; BR-41's rule already covers it (*a value read from outside is rejected at the point it is read when it is not one of the forms the code understands*). The repo has the resolver: `pair_slug_draft_buf()` / `draft_path_for_tag()` at `init.lua:4112`. Low likelihood (the review and scrollback panes are separate nvim processes), but the operation is an unconditional whole-buffer wipe.
- **Plan checkboxes: every box, M1.1 through M2.9, is unticked at HEAD.** 4th in family `traceability`; BR-42's rule stands. It matters concretely here rather than as bookkeeping: two M2.6 sub-requirements were genuinely *not* delivered — the row carries no window length (`doctor.lua:263`), and `perf-captures.jsonl` has no cap despite "cap the file so it cannot grow without bound" — and with nothing ticked they are indistinguishable from the items that were.
- `nvim/doctor.lua:164` still says "The fixture in `nvim/fixtures/`" (it is `doctor/fixtures/`) — BR-30, demoted, noted only because the same window's Revisions entry records the correct location.
- `init.lua:4069` `if not body then return end` is unreachable (PAIR_HOME is checked at `:3997`) but skips the JSONL append and the consume if it ever fires.

## 5. Test coverage notes

The pure layer is well covered and the new assertions test properties rather than strings. The gap is structural: **the boundary between `doctor.lua` and `init.lua` has no test**, and both verified defects (C1's mode gate, I1's probe keys) live there. Note the pattern the review brief warns about — `doctor_test.lua:236` asserts `probes.pipe_hop_ms`, which is what the author *meant* the caller to produce; the caller produces `hop_ms` and the suite stays green. A fixture-driven `probes_from` plus a headless `:PairDoctor` harness closes both. Separately, `verdict`'s asymmetry is well tested at the function level but nothing tests that a `verdict` derived from a *precondition-violating* measurement degrades — which is C1's rule expressed as a test.

## 6. Architectural notes

- **ARCH-DRY — flag.** The probe key set is restated in four hand-maintained places: `perf.sh`'s `kv` calls, `headline`'s `want` (doctor.lua:347), `capture_record`'s `baselines` (doctor.lua:270), and `init.lua:4079`'s regex. Three of the four are already inconsistent with each other. One declared key table in `doctor.lua`, consumed by all three Lua sites and asserted against a live `perf.sh` run, collapses it.
- **ARCH-PURE — flag.** The new `doctor.lua` functions are correctly pure and run under `nvim -l` with no IO. The violation is I1: capture parsing — business logic — sits in the IO shell.
- **ARCH-PURPOSE — flag.** The shadow-sweep over the single-sourced key set finds the four restatements above. More seriously, the issue's stated purpose ("the single most valuable bit" is the discriminator) is under-delivered by C1 while the docs claim it in full.
- **ARCH-MOCK — flag.** `perf.sh`'s system tools have no stateful fake; the seam is the join against a recorded fixture, which is a reasonable choice and was argued in the plan. But M2 added a second external dependency (`vim.system` spawning `perf.sh`) with no seam and no fake, so no test can run the capture end to end. `perf_test.sh`'s stateless `exit 1` stubs are the right idea at the wrong fidelity (BR-38's rule, still unapplied).
- **ARCH-CONSTRAINTS — flag.** The binding constraint (don't block the UI) is honored correctly and is the best decision in the diff. The unbounded side remains: `vim.system` gets no `timeout`, so the nvim-side half of `#210` is unguarded (see Minor).
- **ARCH-SECURE — flag.** I4. Also worth naming for future work: `perf-captures.jsonl` persists the operator's free-text draft content, uncapped, in a shared non-tag-scoped directory. The draft is where prompts are typed, so "whatever was in the buffer" is the threat surface, not just symptom prose.
- **ARCH-ORDER — flag.** The plan's states×events table is the strongest artifact in this issue and the implementation follows it for the events it lists. Two gaps: the table has no row for *the capture never completes*, which is the one event the caller cannot block (Minor above); and the at-review lens on test oracles applies at full force — with zero tests over the wiring, every interleaving the table enumerates is unobserved, so a green suite reports no coverage of the ordering at all.

## 7. Plan revision recommendations

Append to `workshop/plans/000208-pairdoctor-perf-capture-plan.md` `## Revisions`:

1. **M2.6 landed via `pair_data_dir()`, not `artifactpath`** — closes BR-1's plan half, which is still open at HEAD. Record in the same entry that the row carries **no window length** and `perf-captures.jsonl` is **uncapped**, both contrary to M2.6 as written, or deliver them.
2. **M2.3's discriminator as built does not exercise the completion chain** — state the reduced scope (or the fix), because `doctor/SKILL.md`'s exclusion table and the payload text at `doctor.lua:230-233` both rest on the plan's stronger claim.
3. **Core concepts table**: add rows for the entities that shipped without one — `strip_samples`, `headline`, `format_delta`, `parse_samples`, `parse_duration`, `verdict`/`FRAME_MS` — and rename the `CaptureRecord` row to `capture_record` to match the code. This is BR-42's rule applied at the M2 boundary.
4. **Tick M1.1–M2.9** or mark the ones that were not done, so the archived plan does not read as "nothing was executed".

---

## Re-review — 2026-09-07T13:50:40-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 208 — PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact |
| repo | pair |
| issue file | workshop/issues/000208-pairdoctor-captures-harness-drift-but-not-performance-so-slowness-is-always-reconstructed-after-the-fact.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2fb5a79e7dbf3ef4ab63d72e01989e86a6e056f9..a8b5fd388252343a16d064ae6d33ac5038626cf4 |
| command | sdlc milestone-close --issue 208 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-07T13:50:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The rework commit lands four of the five things it claims — `probes_from` is revert-verified (I patched the old `(%w+_ms)` pattern back into a scratch copy of `nvim/` and the row came out `{"exec_ms":1.5,"hop_ms":0.007}`, which `tests/pair-doctor-test.sh` catches), the degraded-capture headline now renders every key (I ran the real `perf.sh` under a sandbox with `ps`/`top`/`sysctl` denied and got all ten `HEADLINE_KEYS` as `n/a (<why>)`), the sidecar is per-capture, and the wiring finally has a headless test that pins the note-preservation rule and the in-flight guard. What blocks SHIP is that the **Critical from round 10 is not fixed in substance**: splitting `complete_now` off the insert-mode gate moved the early return one guard down rather than removing it. `nvim_buf_call` runs the completers in the autocmd window with the cursor at (1,0), so `path_complete`/`word_complete`/`spell_complete` all return at their `if col == 0 then return end` guard. Measured under the real `init.lua`: the completion leg takes **0.0040 ms** and the JSONL row records `"editor":"fast"` — the exact reading `doctor/SKILL.md` tells the agent to exclude `#201`/`#203` on, still drawn from a chain that did no work.

## 1. Strengths

- **`nvim/doctor.lua:415-434` `probes_from` + `tests/pair-doctor-test.sh:71-84`** — the fix and the instrument that catches it, and the instrument genuinely goes red without the fix (revert-verified above). The test drives the *real* `init.lua` headlessly through an injected runner, which is the structural gap round 10 named.
- **`doctor/perf.sh:198-200`** — a failed collector now renders under its success key. Verified end to end: a fully denied environment still produces `load=n/a (…)`, `swapins_per_s=…`, `pipe_hop_ms=…` and a complete headline, so no consumer needs a second list of failure key names.
- **`nvim/init.lua:4041-4046`** — `verdict` variadic over legs, with `n/a` on a leg that could not run, is the right shape. The asymmetry (one slow leg proves slow; `fast` requires every leg measured) is correct and well tested at `nvim/doctor_test.lua:381-385`.
- **`nvim/init.lua:4183-4190`** — `pcall` around the spawn plus `timeout = 30000` closes the "guard never resets" path, and `tests/pair-doctor-test.sh:97-100` pins it with a throwing runner.
- **`nvim/init.lua:4006-4020` / `pair_write_data_file`** — one writer for both artifacts, never throwing, returning `nil` so the payload can say "could not be saved" instead of pointing at a file that isn't there.

## 2. Critical findings

**C1 — `nvim/init.lua:4024-4038`: the completion leg still does not run the chain it names, and the report still sells `fast` as grounds for exclusion.**

This is the **11th finding in family `failure-reported-as-measurement`.** Round 10 already stated the rule — *every emitted reading names the precondition its measurement required and asserts it at the point of measurement; when the precondition does not hold the reading is `n/a (<unmet precondition>)` and every verdict derived from it degrades to `unknown`* — and the rework fixed the one precondition the finding happened to name (mode) without writing the enumeration the rule implies. That is the instance, not the class.

Measured, not inferred. Booting the real `nvim/init.lua` headless and wrapping `_G.PairDoctorCompleteNow`:

```
called buf=2 col=1 line="pairdoctor timing probe" mode=n
complete_now took 0.0040 ms
ROW: {"editor":"fast", …}
```

`vim.api.nvim_buf_call(scratch, …)` cannot reuse a window (the scratch buffer is never displayed), so it uses the autocmd window with the cursor at line 1, col 0. `path_complete` (`:1644`), `word_complete` (`:1817`) and `spell_complete` (`:1949`) each return at `if col == 0 then return end`. The expensive work `#202` is about — `picks_load`, reading `agent_output_path()`, scoring and sorting the span pool, scanning the buffer (`:1825-1877`) — never executes. `doctor/SKILL.md:79` still tells the reader that `editor: fast` means "**every** leg — buffer insert, redraw, and the `#202` completion chain — completed inside one frame", and to exclude `#201`/`#203` on it.

The enumeration the rule demands, written once: the precondition of *every* leg must be asserted where the leg is timed.
- **completion** — precondition is "the cursor sits after a completable token in the timed buffer". Today unasserted and false. Fix: set the cursor to end-of-line inside the `nvim_buf_call` (`WORD_TRIGGER_MIN` is 1, so the probe line qualifies), and — because `vim.fn.complete()` raises `E785` outside Insert mode — split the candidate build off the `complete()` call the same way `complete_now` was split off the gate, so the expensive half is timeable. Then assert the precondition (`vim.fn.col('.') > 1`, and a signal that the chain got past its token gate); if it does not hold, render `completion n/a (<why>)`, which already forces `unknown`.
- **input** — `nvim_buf_set_lines` + `nvim_exec_autocmds('TextChangedI')`; the autocmd debounces onto a timer, so this leg measures a buffer write plus a timer schedule. That is defensible if the payload says so; "buffer insert" in SKILL.md is close enough, but do not let it read as keystroke handling.
- **redraw** — `vim.cmd('redraw')` is a no-op headless; name the precondition (a real UI attached) rather than reporting `0.0ms`.

And the test half: nothing in `tests/pair-doctor-test.sh` asserts anything about the editor legs, which is why round 10's fix could ship without measuring. One assertion that the timed chain reached its candidate build closes the class.

## 3. Important findings

**I1 — `doctor/perf.sh:198-200` + `doctor/perf_test.sh:28`: the "single-sourced key set" is not enforced across the shell boundary, so the C1(b) defect can return silently.**

This is the **3rd finding in family `duplicated-logic`.** The rule: *one declaration per fact, and where a second language cannot import the declaration, a test asserts the two agree — a hand-maintained restatement is a deferred consumer, not a finished one* (ARCH-DRY, ARCH-PURPOSE's shadow-sweep). The rework collapsed the three *Lua* restatements into `doctor.PROBE_KEYS`/`HEADLINE_KEYS`/`BASELINES` (`nvim/doctor.lua:398-413`) and the plan's Revision 4 plus the commit message both claim the set is now one declaration. It is not: `swap_na`/`disk_na`/`probes_na` restate it in shell, and `perf_test.sh:28-31` restates a fourth, different subset.

Verified: I copied `doctor/` to a scratch dir, renamed `swapins_per_s` → `swap_in_rate` throughout `perf.sh`, and re-ran `perf_test.sh` — the failure count was unchanged (the same two pre-existing sandbox `ps` failures, no new one). `doctor_test.lua`'s degraded test builds its input *from* `HEADLINE_KEYS`, so it cannot see a producer rename either. The prompt would silently lose the swap row — indistinguishable from a tool with no such section, which is precisely what C1(b) was about.

Fix per the rule: `perf_test.sh` runs the real `perf.sh` (which emits every key even fully degraded, as shown above) and asserts that every key in `doctor.HEADLINE_KEYS` and `doctor.PROBE_KEYS` appears — reading that list from `doctor.lua` via `nvim -l`, not retyping it. That makes the shell a derived consumer instead of a fourth copy.

**I2 — the tests still accept the ambient input where the seam to control it already exists.**

This is the **5th finding in family `untested-shell-surface`.** BR-38's rule generalises to cover both live members: *a test controls the inputs its subject reads — subprocess output and event ordering alike — rather than accepting whatever the ambient environment supplies.* The machinery for both now exists in-tree and was applied to exactly one behaviour each.

1. **`doctor/perf.sh:143-152`** — the ps-redaction (basename + control-byte strip) is the round-10 I4 fix, and no test fails without it. `perf_test.sh:117-133` puts a fake `ps` on PATH but it only `exit 1`s, so it pins the pipeline-exit-status half and nothing about content. Reverting the awk body to `$4` leaves every suite green. Related: `doctor/fixtures/perf_capture.txt:82` still carries full paths (`/System/Library/…/com.apple.geod`), i.e. the recorded fixture is output the current producer no longer emits — so the fixture's stated job (`nvim/doctor.lua:174-180`, "the CONTRACT between perf.sh and delta") is no longer being done. Sweep: promote the fake `ps` to recorded output including a control byte and a long path, assert basename + `?`, and regenerate the fixture from the current producer.
2. **`nvim/init.lua:4157-4168`** — the buffer-changed-mid-flight branch, which the plan calls "the only **data-loss** path in the design", is unexecuted. The new test pins the failed-capture branch and the guard reset but not this one, and not the successful consume either. The seam makes it trivial: the injected runner receives `cb`, so it can rewrite the buffer before invoking it. Without that, a green suite reports one interleaving out of three (ARCH-ORDER at-review).

**I3 — `doctor/SKILL.md:74` and `atlas/index.md:40` name `perf-capture-latest.txt`, which the code no longer writes.**

This is the **3rd finding in family `docs-gate`.** The rule: *a doc that names a runtime artifact names the one the code produces, and the naming is checked in the same window that changes it* — enumerable by grepping docs for `$PAIR_DATA_DIR/<name>` and matching each against the writers in `init.lua`. The rework renamed the sidecar to `perf-capture-<epoch>.txt` (`nvim/init.lua:4128`, verified: the run produced `perf-capture-1788813769.txt`) but left both docs pointing at the old fixed name. SKILL.md's instruction is literally "**Open that file**", and it is the file an agent reads when the prompt arrives truncated — the recovery path `#211` exists for. Fix the two lines to describe the pattern (and, since the pattern is now a fact the docs restate, the same rule as I1 applies: state it once, in `doctor.lua`, and have the payload carry the path — which it already does).

**I4 — `nvim/init.lua:3999-4002`: `time_editor` leaks a scratch buffer on every invocation, and the comment claims a delete that does not exist.**

Family `unreleased-resource` (new): *a resource acquired inside a function is released on every exit path, and a comment claiming teardown is a claim a test should hold.* Measured: three `:PairDoctor` runs under headless nvim took the valid-buffer count from 1 to 4. `grep nvim_buf_delete nvim/init.lua` returns nothing. The plan's ARCH-ORDER table has a row requiring exactly this ("tear it down in a `pcall`-protected finally"), so the plan currently claims delivered behaviour the code does not have. One `pcall(vim.api.nvim_buf_delete, scratch, {force=true})` after the last use, plus a count assertion in `tests/pair-doctor-test.sh` (which already runs `:PairDoctor` three times' worth of paths).

## 4. Minor findings

- **`workshop/plans/…-perf-capture-plan.md:420-424`** claims the missing window-length field and the uncapped `perf-captures.jsonl` are "deferred to `#210`", but `workshop/issues/000210-*.md` records neither — it covers only time-bounding plus the three re-surfaced M1 findings. 4th in family `traceability`; the rule is that a deferral lands in the artifact that survives, and this plan archives to `workshop/history/` at close. Add both to `#210`'s table.
- `nvim/init.lua:3829` adds `_G.PairDoctorCompleteNow` beside `_G.PairDraftCompleteTest.complete_now` (`:3827`) — two globals for one function; the timing site can read the existing table (ARCH-DRY).
- `nvim/init.lua:4128` uses `os.time()`, so two captures in the same second collide on the sidecar name — the failure mode the per-capture rename exists to prevent, at one second's granularity. Sidecars also accumulate one file per invocation forever in a non-tag-scoped dir alongside the uncapped JSONL, both carrying the operator's raw draft text (ARCH-SECURE).
- `nvim/doctor.lua:180` still says "The fixture in `nvim/fixtures/`" (it is `doctor/fixtures/`) — BR-30, previously demoted, noted only because I2 touches that fixture.
- `vim.system` gets `timeout = 30000` against a declared 6 s envelope, so the in-flight guard can hold `:PairDoctor` closed for 30 s on a struggling machine. Defensible as an outer bound; worth naming in the plan's ARCH-CONSTRAINTS rather than leaving the 5× gap implicit.

## 5. Test coverage notes

`nvim -l nvim/doctor_test.lua` and `bash tests/pair-doctor-test.sh` are both green here (9/9 on the latter). `sh doctor/perf_test.sh` fails with 2 grammar-row failures in this shell because `ps` is denied — that is BR-38, already disposed and demoted, not a new defect, but it does mean `make test` is red in a sandboxed agent shell and I could not use `perf_test.sh` as a clean oracle for the I1 revert check (I compared failure counts instead). The pure layer is strong. The gaps are all "the test asserts what the author meant, not what the caller does": no assertion touches the editor legs (C1), the degraded-headline test builds its input from the very constant it validates (I1), the ps-content filter has no test at all (I2.1), and two of the three in-flight interleavings are unexecuted (I2.2).

## 6. Architectural notes

- **ARCH-DRY — flag** (I1). The Lua side is genuinely collapsed; the shell side is three restatements and a test that is a fourth.
- **ARCH-PURE — pass.** Moving the capture parsing out of `init.lua` into `probes_from` removes the last business logic from the IO shell, and `pair_write_data_file` collapses the duplicated write blocks. `init.lua` is now orchestration only.
- **ARCH-PURPOSE — flag** (C1). The Spec names the discriminator "the single most valuable bit"; the shadow-sweep over the single-source key set finds two deferred consumers (I1) and the discriminator still under-delivers while three artifacts claim it in full.
- **ARCH-MOCK — partial pass.** `capture_runner` is a real seam and the wiring can now be driven end to end — the biggest structural improvement in the diff. The `ps`/`top`/`vm_stat` doubles are still stateless `exit 1` stubs, so no test can run the collection path against controlled output (I2.1).
- **ARCH-CONSTRAINTS — pass with a note.** The budget is enforced and pinned (`elapsed ≤ budget` in `perf_test.sh`), shedding is reverse-value-order, and the UI-blocking constraint is honored. Unbounded: the scratch buffers (I4) and the sidecar/JSONL growth.
- **ARCH-SECURE — flag** (I2.1). Filtering at the point of emission is the correct design; it is unpinned, and the committed fixture predates it.
- **ARCH-ORDER — improved, flag.** The guard now resets on a throwing spawn and the spawn is time-bounded, both pinned. The remaining at-review lens applies: the buffer-consume compare — the plan's named data-loss path — is observed in exactly one interleaving (I2.2).

## 7. Plan revision recommendations

Append to `workshop/plans/000208-pairdoctor-perf-capture-plan.md` `## Revisions`:

1. **M2.3 / Revision entry 2 is not yet true.** It states the timing "calls the chain directly against the scratch buffer"; it does, but through `nvim_buf_call`'s autocmd window at col 0, where all three completers return at their column guard (measured: 0.004 ms, verdict `fast`). Either record the reduced scope or, preferably, record the fix — and in the same entry state the rule the enumeration follows, so leg #4 is not the next round's finding.
2. **Revision entry 4's "the key set is now single-sourced" overstates the delivery.** Three shell restatements (`perf.sh:198-200`) and a fourth in `perf_test.sh:28` remain, unenforced; a producer rename is invisible to every suite (verified). Either scope the claim to the Lua consumers or add the producer↔consumer assertion.
3. **The M2.6 deferrals need a home that outlives this plan.** The entry says window-length and the JSONL cap are "deferred to `#210`"; `#210` records neither. Add them there, or drop the claim.
4. **The ARCH-ORDER row "tear it down in a `pcall`-protected finally"** is unimplemented — `time_editor` never deletes the scratch buffer. Fix the code rather than the plan.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan Revisions record M2.6 landing via pair_data_dir() not artifactpath, and the probe as the `pair hoprtt` subcommand; both mechanisms verified reachable in the tree.
findings:
  - id: new
    severity: Critical
    family: failure-reported-as-measurement
    title: |
      The completion leg still measures nothing — the gate moved from mode to cursor column, and `editor: fast` is still emitted as grounds for exclusion
    detail: |
      11th in this family. The rule was stated in round 10 and applied only to the
      precondition that round named. nvim_buf_call runs complete_now in the autocmd
      window at col 0, so path/word/spell_complete all return at `if col == 0 then
      return end` (init.lua:1644, :1817, :1949). Measured under the real init.lua:
      complete_now takes 0.0040 ms and the row records "editor":"fast", while
      doctor/SKILL.md:79 tells the reader that `fast` means the #202 chain ran inside
      a frame and that #201/#203 are excluded on it. Fix the class: assert each leg's
      precondition where it is timed (completion needs a completable token at the
      cursor; redraw needs a real UI), render n/a with the unmet precondition
      otherwise, and add one wiring assertion that the timed chain reached its
      candidate build. Note vim.fn.complete() raises E785 outside Insert mode, so the
      candidate build must be split off the complete() call the way complete_now was
      split off the gate.
  - id: new
    severity: Important
    family: duplicated-logic
    title: |
      The single-sourced key set stops at the Lua boundary — perf.sh and perf_test.sh restate it, and a producer rename breaks nothing
    detail: |
      3rd in this family. Rule: one declaration per fact, and where a second language
      cannot import it, a test asserts the two agree. doctor.lua:398-413 collapsed the
      three Lua sites, but perf.sh:198-200 restates the set in shell and
      perf_test.sh:28 restates a different subset. Verified by renaming
      swapins_per_s to swap_in_rate in a scratch copy of doctor/ — perf_test.sh's
      failure count was unchanged, and doctor_test.lua cannot see it because it builds
      its degraded input from HEADLINE_KEYS itself. The prompt would silently drop the
      swap row, which is the C1(b) defect returning. Fix: perf_test.sh runs the real
      perf.sh and asserts every key read out of doctor.HEADLINE_KEYS/PROBE_KEYS via
      `nvim -l` appears in the output.
  - id: new
    severity: Important
    family: untested-shell-surface
    title: |
      The ps-content filter and the buffer-changed interleaving are both unexecuted, though the seams to control each now exist
    detail: |
      5th in this family. BR-38's rule generalised: a test controls the inputs its
      subject reads — subprocess output and event ordering alike. Two live members.
      (1) perf.sh:143-152's basename + control-byte strip has no test; perf_test.sh's
      fake ps only `exit 1`s, so reverting the awk body to `$4` leaves every suite
      green, and doctor/fixtures/perf_capture.txt:82 still carries full paths the
      current producer no longer emits. (2) init.lua:4157-4168's buffer-changed
      branch — the plan's only data-loss path — is unexecuted; the injected runner
      receives cb, so the test can rewrite the buffer before invoking it. Sweep both:
      recorded-output ps fake plus a regenerated fixture, and the two missing
      interleavings in tests/pair-doctor-test.sh.
  - id: new
    severity: Important
    family: docs-gate
    title: |
      SKILL.md and atlas name perf-capture-latest.txt, which the rework stopped writing
    detail: |
      3rd in this family. Rule: a doc naming a runtime artifact names the one the code
      produces, checked in the window that changes it — enumerable by grepping docs
      for $PAIR_DATA_DIR/<name> against the writers in init.lua. init.lua:4128 now
      writes perf-capture-<epoch>.txt (verified: perf-capture-1788813769.txt), while
      doctor/SKILL.md:74 and atlas/index.md:40 still name the fixed file and SKILL.md
      instructs the agent to open it — the truncated-send recovery path #211 exists
      for.
  - id: new
    severity: Important
    family: unreleased-resource
    title: |
      time_editor leaks a scratch buffer per invocation while its comment claims a guaranteed delete
    detail: |
      New family. Rule: a resource acquired inside a function is released on every
      exit path, and a comment claiming teardown is a claim a test should hold.
      init.lua:3999 creates the buffer; nothing deletes it (grep nvim_buf_delete
      returns nothing) and :4001 asserts otherwise. Measured: three :PairDoctor runs
      took the valid-buffer count from 1 to 4. The plan's ARCH-ORDER table has a row
      requiring the teardown, so the plan claims delivered behaviour the code lacks.
  - id: new
    severity: Minor
    family: traceability
    title: |
      The M2.6 deferrals are recorded only in the plan, which archives at close — issue 210 records neither
    detail: |
      4th in this family. The plan's Revisions say the missing window-length field and
      the uncapped perf-captures.jsonl are "deferred to #210"; #210 covers only
      time-bounding plus three re-surfaced M1 findings. A deferral must land in the
      artifact that survives the archive.
```
