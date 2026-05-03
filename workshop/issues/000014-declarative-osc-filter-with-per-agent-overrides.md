---
id: 000014
status: open
deps: []
created: 2026-05-03
updated: 2026-05-03
---

# declarative OSC filter with per-agent overrides

`bin/pair-wrap` over-notifies cmux. Live data from `~/pair-wrap.log` (one session, ~2hr): 76 EMITs total, only 8 of them legitimate (OSC 777 `Claude is waiting for your input`). The other 68 are false positives — bare BEL fallback firing on the trailing `\x07` of OSC 8 hyperlinks and OSC 0 title sets that the streaming regex couldn't reconstruct.

Mental model we want: a declarative table of OSC sequences with explicit `forward`/`skip` rules, optionally namespaced per agent (claude / codex / gemini). Adding a new agent or refining behavior becomes a data change, not code.

## Done when

- The 64 hyperlink/title BEL false positives observed in `~/pair-wrap.log` no longer fire emits when reproduced.
- Legitimate `OSC 777;notify;Claude Code;...` still forwards.
- Filter rules live in a single declarative table at the top of `bin/pair-wrap`, easy to read and edit.
- Per-agent override hook in place (driven by `$PAIR_AGENT`); claude's table is the default, others can be added incrementally.
- `PAIR_WRAP_LOG` discovery workflow still works for unknown agents.

## Spec

### Symptom (decoded from `~/pair-wrap.log`)

Claude Code emits OSC 8 hyperlinks for clickable file references (`\x1b]8;;file:///.../README.md\x07README.md\x1b]8;;\x07`) and OSC 0 title sets every second for the spinner (`\x1b]0;⠂ Claude Code\x07`). Both terminate with `\x07`.

The wrapper's `OSC_RE` correctly matches well-formed OSC sequences and `is_actionable_osc()` filters titles/hyperlinks as non-actionable. The bug is in the *fallback* path: when an OSC's terminating `\x07` arrives in a read whose preceding bytes were already consumed by a prior match (so the opener `\x1b]8;;` is no longer in `rolling`), the `elif b"\x07" in data:` branch fires and emits.

Six of the seven distinct false-positive BEL contexts in the log are tails of OSC 8 hyperlinks; the seventh (`ure and contents\x07`) is the tail of an OSC 0 title set with the long task-name as content.

### Approach

Two layers of fix:

**Layer 1 — drop the bare-BEL fallback by default.** It's a relic. Modern TUI agents that want attention use OSC 9 or OSC 777 explicitly. The fallback existed to "catch unknown signals from unfamiliar agents" but in practice it just catches stream-fragmentation noise. Make it opt-in via `PAIR_WRAP_BELL_FALLBACK=1` for the discovery workflow.

**Layer 2 — declarative filter table.** Replace `is_actionable_osc()` with a table:

```python
# (ps, body_predicate) -> action
DEFAULT_RULES = [
    Rule(ps="777", match=any, action="forward"),
    Rule(ps="9",   match=lambda b: not b.startswith(b"4;"), action="forward"),
    Rule(ps="9",   match=lambda b: b.startswith(b"4;"),     action="skip"),    # iTerm progress
    Rule(ps="0",   match=any, action="skip"),  # window title
    Rule(ps="1",   match=any, action="skip"),  # icon name
    Rule(ps="2",   match=any, action="skip"),  # window title (alt)
    Rule(ps="8",   match=any, action="skip"),  # hyperlinks (claude uses these for file refs)
    Rule(ps="1337", match=any, action="skip"), # iTerm proprietary
]

AGENT_RULES = {
    "claude": [],   # uses defaults
    "codex":  [],   # populate as we discover
    "gemini": [],   # populate as we discover
}
```

Resolution: `$PAIR_AGENT` selects an override list; rules are matched in order, agent-specific first then defaults. First match wins. An OSC that matches no rule is treated as `skip` (conservative — silence is better than spam, and the discovery log will show `OSC<N>-unmatched` so we know to add a rule).

### Per-agent: how does this play out?

Claude — table above is sufficient. Live data confirms it covers everything we see.

Codex — empty `[]` means defaults apply. Use the `PAIR_WRAP_LOG` discovery workflow to populate as needed. If codex emits some OSC family we don't recognize, `OSC<N>-unmatched` lines tell us what to add.

Gemini — same.

The agent override list is *prepended* to defaults, so it can either tighten a default rule (e.g. forward an OSC family that defaults skip) or add a new one. There's no need for a "remove default" verb yet — defer until we hit a case where it's needed.

### Why not just look further back in `rolling` to reattach orphaned BELs to their OSC opener?

Considered and rejected:
- "How far back?" is brittle. OSC 8 with a long absolute path can be hundreds of bytes; a 512-byte rolling buffer won't always cover it.
- Multiple hyperlinks in one render produce a sequence like `<OSC 8 open><text><OSC 8 close><OSC 8 open><text><OSC 8 close>` — re-matching across reads where some openers were trimmed gets gnarly.
- The miss case (one false BEL leaks through) isn't catastrophic, but the engineering complexity to mostly-fix it is high. Dropping the fallback altogether trades a hypothetical future signal we'd recover via fallback for the concrete current noise it causes.

### Why opt-in `PAIR_WRAP_BELL_FALLBACK` rather than removing it entirely?

Two reasons:
1. The discovery workflow benefits from it — when pairing with a brand-new agent that uses bare BEL (rare but possible), the user can enable it and at least see *something* in the log.
2. Reversibility. If we discover an agent in the wild that genuinely uses BEL, flipping a flag is faster than re-introducing the code.

## Plan

- [ ] Add `Rule` (small dataclass or namedtuple) and `DEFAULT_RULES` table at the top of `bin/pair-wrap`.
- [ ] Add `AGENT_RULES` dict (empty lists for `claude`/`codex`/`gemini` initially).
- [ ] Replace `is_actionable_osc(ps, body)` with `classify_osc(ps, body) -> "forward" | "skip" | "unmatched"`. Resolution: agent-specific rules (from `$PAIR_AGENT`) prepended to defaults; first match wins; no match → `unmatched` (treated as skip, logged distinctly).
- [ ] Update the OSC-match path in `main()` to dispatch on the classify result. Log `OSC<N>:`, `OSC<N>-skip:`, `OSC<N>-unmatched:` accordingly.
- [ ] Gate the bare-BEL fallback behind `PAIR_WRAP_BELL_FALLBACK` env var (default off). Log `BEL-skip:` when off.
- [ ] Update `atlas/architecture.md` § `bin/pair-wrap` to describe the table-driven model, the per-agent hook, and the `PAIR_WRAP_BELL_FALLBACK` opt-in.
- [ ] Update README "Notifications" section if any user-visible behavior shifts (probably just one line).
- [ ] Manual verification:
  - Replay the `~/pair-wrap.log` symptom: open a Claude Code session, perform actions that emit hyperlinks and title spinners; confirm `BEL-skip:` lines (or no BEL handling at all) and zero false EMITs.
  - Idle Claude for ~60s; confirm the OSC 777 still forwards as `OSC777:` + `EMIT:`.
  - Set `PAIR_WRAP_BELL_FALLBACK=1`; confirm bare BELs come back.
  - `pair codex` / `pair gemini` smoke test: no crashes, defaults apply, log shows `OSC<N>-unmatched` for anything we haven't seen before.

## Log

### 2026-05-03

- Filed from live data analysis of `~/pair-wrap.log` during the issue #13 work session: 76 EMITs, 8 legitimate (OSC 777), 68 spurious (BEL fallback firing on hyperlink/title tails).
