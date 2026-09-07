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
