---
name: xx-pair-doctor
description: Use when a pair agent-harness adaptation feels off — Enter leaking a newline instead of confirming a picker, a stale/empty slug, broken Alt+b prompt jumps, a resume that won't reattach — or when proactively checking a harness for integration drift after an agent CLI update. Reads the adaptation flight recorder and proposes fixes.
---

# pair-doctor — diagnose agent-harness integration drift

`pair` adapts each harness (claude/codex/agy) across the integration aspects in
`atlas/how-to-bring-up-a-new-harness-cli.md`. Harnesses update and break those
adaptations *silently* — a renamed picker string or changed transcript shape
doesn't error, the adaptation just stops firing. The **flight recorder**
(`$PAIR_DATA_DIR/adapt-<tag>.jsonl`) captures one line per adaptation trigger,
including **near-misses** (the harness did something we half-recognized but no
matcher caught). This skill reads that trace and turns it into a fix.

## Operating principle

You (the agent in the user's session) run the aggregator yourself with the Bash
tool, interpret the output against the atlas registry, and propose concrete
fixes for the user to approve. Don't ask the user to run shell by hand. Don't
edit matcher code silently — surface the finding and the proposed edit first.

## Procedure

1. **Run it.** The aggregator self-locates the current session's log via
   `$PAIR_TAG`, falling back to the newest `adapt-*.jsonl`; pass an explicit path
   as `$1` to inspect a specific session's log:

   ```bash
   bash doctor/doctor.sh
   ```

2. **Read the tallies + drift findings.** The output groups every event by
   `aspect · signal/outcome · count`, then lists deduped `near-miss`/`fail`
   findings with the literal `detail` string the live harness emitted.

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
