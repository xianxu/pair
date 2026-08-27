# pair-doctor — diagnose agent-harness integration drift

`pair` adapts each harness (claude/codex/agy) across the integration aspects in
[`../atlas/how-to-bring-up-a-new-harness-cli.md`](../atlas/how-to-bring-up-a-new-harness-cli.md).
Harnesses update and break those adaptations *silently* — a renamed picker
string or a changed transcript shape doesn't error, the adaptation just stops
firing. The **flight recorder** (`$PAIR_DATA_DIR/adapt-<tag>.jsonl`) captures one
line per adaptation trigger, including **near-misses** (the harness did something
we half-recognized but no matcher caught). `doctor.sh` turns that trace into a
fix.

## Run it

```bash
doctor/doctor.sh            # current session ($PAIR_TAG), else newest adapt-*.jsonl
doctor/doctor.sh <path>     # a specific session's log
```

It first prints an **emitter-health** check, then per-`aspect · signal/outcome`
tallies, then the deduped `near-miss`/`fail` findings with the literal `detail`
string the live harness emitted. Always exits 0; prints `NO-DATA` if no log
exists yet.

**Emitter health** (`emitter-health.sh`) guards the failure that *looks* like
drift but isn't: a `pair-wrap`/`pair-slug` binary built before the flight
recorder existed has no logging code, so it emits nothing and the log goes
silent with no error. The probe greps each binary (preferring the *running* one
via its pidfile) for its adapt signal string and flags `[STALE]` with the fix —
`make install`, or launch via `pair-dev` (#000046). A `[STALE]` line explains an
otherwise-mysterious empty/thin tally below it.

## Read the findings

The output maps to the signal registry in
[`../atlas/how-to-bring-up-a-new-harness-cli.md`](../atlas/how-to-bring-up-a-new-harness-cli.md)
§3. Anchor to the symptom you hit when you have one; otherwise scan all findings.

| Finding | Likely drift | Fix |
|---|---|---|
| `overlay-detect/near-miss` (aspect 2) | harness renamed its picker; the `detail` holds the new wording | add that string to `codexPickerMarkers` / `agyPickerMarkers` / `musePickerMarkers` (or the OSC body for claude) in `cmd/internal/wrapcmd/wrap.go` |
| `return-remap` all `bypass`, no `fired` (aspect 1) | remap stopped engaging | check the agent's entry in `harnessTTYProfiles` (`cmd/internal/wrapcmd/harness_tty.go`) via `profileForHarness`; a `composer unknown` reason means a positively gated profile had no snapshot or no registered recognizer, `composer inactive` means the recognizer ran and declined — all four harnesses now have one, so this reason is reachable for `claude` too. No profile at all logs nothing — plain Return simply passes through, which is also what `PAIR_WRAP_REMAP_RETURN=0` produces (it disables overlay detection and its `overlay-detect` telemetry too) |
| `return-remap` reason `composer inactive` on a real composer (aspect 1) | the harness repainted its composer and the recognizer no longer matches | recapture evidence with `PAIR_LIVE_HARNESS=<agent> go test ./cmd/internal/wrapcmd -run TestHarnessTTYLive -count=1 -v`, then fix the recognizer in `composer_recognizers.go` against the new fixture |
| `session-id/fail` or `near-miss` (aspect 3) | session file moved or id format changed | update the watch-dir / find / id-extract logic in `cmd/internal/sessionwatch` |
| `slug-parse/near-miss` (aspect 4) | transcript schema changed | update the parser in `cmd/pair-slug/slug.go` |
| `output-filter` *absent* for codex (aspect 5) | a sync-output sequence was renamed (no `fired` line where you'd expect one) | update `codexSyncOutputMarkers` in `cmd/internal/wrapcmd/wrap.go` |
| `prompt-search/near-miss` (aspect 7) | prompt glyph changed | update `PROMPT_PATTERN_BY_AGENT` in `nvim/scrollback.lua` |

A `detail` string is the literal text the live harness emitted — usually exactly
what you paste into the matcher. When you fix a drift, add the new string as a
frozen sample in the matcher's test so the *same* drift is caught next time.

## Notes

- The log truncates at each session launch (`bin/pair`), so it reflects the
  current run. To diagnose a past session, point the script at a saved copy.
- `detail` is capped at 200 bytes and stays local under `$PAIR_DATA_DIR`; it can
  contain a snippet of agent output, so treat findings as session-private.
- No findings + a real symptom? The relevant adaptation may have no signal yet
  (e.g. aspect 6 is static config). That gap is a candidate follow-up.

## Entry points

The primary, agent-agnostic entry is **`:PairDoctor`** in pair's nvim (#000048):
it hands whatever agent is running (claude / codex / agy / vanilla) a
`$PAIR_HOME`-absolute instruction to run `doctor.sh` and propose fixes — so it
works under any agent and from any cwd, unlike a `.claude/skills/` entry (claude
only) or a bare `doctor/doctor.sh` (resolves only in the pair repo). The
procedure it points at is single-sourced in [`SKILL.md`](SKILL.md).

[`SKILL.md`](SKILL.md) also makes `doctor/` a self-contained Agent Skill
(AGENTS.md §11) over the one `doctor.sh` here (no bundled copy). Registering it
as a Claude skill — by linking `doctor/` into `.claude/skills/` — is an optional
convenience for claude users; `:PairDoctor` is the path that works everywhere.
