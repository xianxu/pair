---
name: xx-pair-doctor
description: Use when a pair agent-harness adaptation feels off — Enter leaking a newline instead of confirming a picker, a stale/empty slug, broken Alt+b prompt jumps, a resume that won't reattach — or when proactively checking a harness for integration drift after an agent CLI update. Reads the adaptation flight recorder and proposes fixes.
---

# pair-doctor — diagnose agent-harness integration drift

`pair` adapts each harness (claude/codex/agy) across the integration aspects in
`atlas/how-to-bring-up-a-new-harness-cli.md`. Harnesses update and break those
adaptations *silently* — a renamed picker string or changed transcript shape
doesn't error, the adaptation just stops firing. The **flight recorder**
(`$PAIR_ADAPT_LOG_PATH`) captures one line per adaptation trigger,
including **near-misses** (the harness did something we half-recognized but no
matcher caught). This skill reads that trace and turns it into a fix.

## Operating principle

You (the agent in the user's session) run the aggregator yourself with the Bash
tool, interpret the output against the atlas registry, and propose concrete
fixes for the user to approve. Don't ask the user to run shell by hand. Don't
edit matcher code silently — surface the finding and the proposed edit first.

## Procedure

1. **Run it.** The aggregator uses the current session's exact
   `$PAIR_ADAPT_LOG_PATH` binding; outside a live Pair pane, pass an explicit path
   as `$1` to inspect a specific session's log:

   ```bash
   bash doctor/doctor.sh
   ```

2. **Check emitter health first, then read the tallies + drift findings.** The
   output leads with an emitter-health line: a `[STALE]` `pair-wrap`/`pair-slug`
   binary means the recorder can't log for whole aspects — that's the answer, and
   the fix is `make install` / `pair-dev` (#000046), not a matcher edit. Then the
   tallies group every event by `aspect · signal/outcome · count`, followed by
   deduped `near-miss`/`fail` findings with the literal `detail` string emitted.

3. **Interpret against the registry** in
   `atlas/how-to-bring-up-a-new-harness-cli.md` §3. The Finding → likely-drift →
   fix mapping is tabulated in [`README.md`](README.md) ("Read the findings") —
   use it rather than re-deriving. Anchor to the symptom the user reported when
   they have one; otherwise scan all findings. A `detail` string is usually
   exactly what you paste into the matcher.

4. **Propose the fix** as a specific edit (file + matcher + the new string from
   `detail`), plus a frozen-sample test so the next drift of the *same* kind is
   caught. Get approval before editing.

5. **No findings?** If tallies look healthy with no near-miss/fail lines, say so.
   If the user reported a symptom anyway, the adaptation for it may have no signal
   yet (e.g. aspect 6 is static config) — note that gap as an atlas/issue
   follow-up.

## Notes

- The log truncates at each session launch (`bin/pair`), so it reflects the
  current run. To diagnose a past session, point the script at a saved copy.
- `detail` is capped at 200 bytes and stays local under `$PAIR_DATA_DIR`; it can
  contain a snippet of agent output, so treat findings as session-private.

## Performance capture (`#208`)

`:PairDoctor` also captures machine state, because the question "why is this
slow?" is only answerable **while it is slow** — and the drift half of this
skill is durable while performance is not. A 2026-09-06 investigation produced
three issues and no theory precisely because every measurement was taken on a
healthy machine.

**Read the report in this order. The order is the method.**

**1. The operator's note.** It leads the payload deliberately. Without it you
will explain whatever number is largest instead of what was actually reported —
"typing is slow" and "the build is slow" have different suspects.

**2. `editor:` — the discriminator, and the single most valuable line.**

| reading | what it excludes |
|---|---|
| `editor: fast` | nvim handled a keystroke inside one frame (16 ms at 60 Hz). If typing still *feels* slow, the cause is **at or above the terminal** — transport, rendering, compositing — and the scheduling family (`#201`, `#203`) is excluded for this symptom. |
| `editor: slow` | the cause is **inside nvim**. Look at autocmds and plugins; the environment numbers below are probably noise. |
| `editor: unknown` | timing did not run, or only half of it did. It is **not** a synonym for fast — partial evidence can prove slow but never fast. |

**3. The probes, against the baselines printed beside them.** `pipe_hop` is one
scheduler wake-up, `fork_exec` is process creation, `zellij_action` is
spawn+link+connect+round-trip. They degrade *together* under a process-spawn
storm; a hop staying near 7 µs while typing feels slow points away from
scheduling entirely.

**4. Everything else is context, not verdict.** In particular:

- **`load` does not predict this.** Measured 2026-09-06: degradation at load 9.5
  and none at load 15.5. What correlated was the workload's *phase* — many
  short-lived processes — not its size.
- **`cpu_idle_pct` is routinely high while the machine feels terrible.** Every
  slowdown investigated on this host has had idle CPU.
- **`windowserver_cpu_pct` is the untested candidate.** It has been seen at 45%
  with 66% idle CPU, and its units (tens of ms per frame) are the right order of
  magnitude for visible lag, unlike scheduling's microseconds.

**5. `n/a` means NOT MEASURED. Never read it as zero.** Every collector degrades
to `n/a` with a reason rather than emitting a value, because a fabricated `0` is
indistinguishable from a real reading — the defect class this capture was
hardened against across seven review rounds.

**Comparing readings.** Each capture appends a row to
`$PAIR_DATA_DIR/perf-captures.jsonl` carrying its probes, the verdict, and the
baselines. A single row answers "what is happening now"; the series answers
"what changed", which is what the original investigation lacked.

**Re-running standalone:** `sh $PAIR_HOME/doctor/perf.sh`. It needs no editor.

**Known gap:** no stage is time-bounded, so a hanging collector can exceed the
budget (`#210`).
