---
id: 000014
status: done
deps: []
created: 2026-05-03
updated: 2026-05-03
---

# stop forwarding bare BELs by default

`bin/pair-wrap` over-notifies cmux. Live data from `~/pair-wrap.log` (one session, ~2hr): 76 EMITs total, only 8 of them legitimate (OSC 777 `Claude is waiting for your input`). The other 68 are bare-BEL fallback firing on the trailing `\x07` of OSC 8 hyperlinks and OSC 0 title sets that the streaming regex couldn't reconstruct across read boundaries.

Modern TUI agents emit OSC 9 / OSC 777 explicitly when they want attention. The bare-BEL fallback was defensive coding for unfamiliar agents that "might use BEL" — a case we have no concrete example of in the wild. Today its only measurable effect is noise.

Fix: stop forwarding bare BELs by default. Keep the *detection and logging* live so `PAIR_WRAP_LOG` still shows every BEL the wrapper sees, and we can choose to whitelist specific contexts later. Provide an env flag to re-enable forwarding when discovering a new agent's protocol.

## Done when

- The 64 hyperlink/title BEL false positives observed in `~/pair-wrap.log` no longer fire emits.
- Legitimate `OSC 777;notify;Claude Code;...` still forwards.
- `PAIR_WRAP_LOG` still records every BEL event (just labeled `BEL-skip:` when forwarding is off), so the log remains a complete picture for "which kinds should go through later" decisions.
- `PAIR_WRAP_BELL_FALLBACK=1` re-enables BEL forwarding for the discovery workflow.

## Spec

### Symptom (decoded from `~/pair-wrap.log`)

Claude Code emits OSC 8 hyperlinks for clickable file references (`\x1b]8;;file:///.../README.md\x07README.md\x1b]8;;\x07`) and OSC 0 title sets every second for the spinner (`\x1b]0;⠂ Claude Code\x07`). Both terminate with `\x07`.

The wrapper's `OSC_RE` correctly matches well-formed OSC sequences and `is_actionable_osc()` filters titles/hyperlinks as non-actionable. The bug is in the *fallback* path: when an OSC's terminating `\x07` arrives in a read whose preceding bytes were already consumed by a prior match (so the opener `\x1b]8;;` is no longer in `rolling`), the `elif b"\x07" in data:` branch fires and emits.

The fallback can't reliably tell "orphaned tail of an OSC we've already filtered" from "an actual standalone BEL." Six of the seven distinct false-positive BEL contexts in the log are tails of OSC 8 hyperlinks; the seventh (`ure and contents\x07`) is the tail of an OSC 0 title set with the long task-name as content.

### Approach

Stop emitting on bare BEL by default. Keep the detection branch — still log the event, with the snippet — but don't write to the outer TTY.

```python
BELL_FALLBACK = os.environ.get("PAIR_WRAP_BELL_FALLBACK") not in (None, "", "0")

# in the detect path, where bare BEL was previously handled:
elif b"\x07" in data:
    idx = data.index(b"\x07")
    snippet = data[max(0, idx - 16):idx + 16]
    if BELL_FALLBACK:
        debug("BEL", snippet)
        emit_outer()
    else:
        debug("BEL-skip", snippet)
```

Why preserve the log line: the user's mental model is to keep `PAIR_WRAP_LOG` as a forensic record they can review periodically. Seeing `BEL-skip:` lines tells them what *would* have fired, so if a real attention signal ever shows up as a bare BEL in some future agent, they'll see it in the log and can decide whether to flip the flag or special-case the context.

### What we're explicitly NOT doing

- Not introducing a per-agent rule table. The current `is_actionable_osc()` already covers what we know. Defer until a concrete second agent gives us a reason — premature structure ages worst when nobody adds data to the slots you built.
- Not trying to "look back further" in `rolling` to reattach orphaned BELs to their OSC opener. Brittle (file-path URLs blow past any fixed buffer, multiple hyperlinks in one render confuse the heuristic), and the engineering complexity outweighs the benefit when "drop it" works.

### Risk

If some agent in the wild really does signal attention via bare BEL, this change makes pair-wrap silent on that agent until the user discovers the issue and sets `PAIR_WRAP_BELL_FALLBACK=1`. Mitigation: the log still records BEL events as `BEL-skip:`, so the discovery is one `cat ~/pair-wrap.log` away. Document the flag in `atlas/architecture.md` next to the existing `PAIR_WRAP_LOG` discussion.

## Plan

- [x] In `bin/pair-wrap`, add `BELL_FALLBACK = os.environ.get("PAIR_WRAP_BELL_FALLBACK") not in (None, "", "0")` near the top with the other env reads.
- [x] Gate the bare-BEL `emit_outer()` call behind `BELL_FALLBACK`. Log `BEL:` when forwarding, `BEL-skip:` when not.
- [x] Update `atlas/architecture.md` § `bin/pair-wrap`: explain the default-off bare-BEL behavior, point at `PAIR_WRAP_BELL_FALLBACK` for the discovery workflow, link the rationale (data from this issue).
- [x] Synthetic smoke test (`/tmp/claude/wrap_smoke.py`): drives a child agent that emits an OSC 0 title (matches and trims rolling), then a bare `\x07` alone in a separate read (the orphaned-BEL scenario), then OSC 777. Confirmed default produces `BEL-skip` with no EMIT, `PAIR_WRAP_BELL_FALLBACK=1` produces `BEL:` with EMIT, OSC 777 always forwards.
- [x] Manual verification in a real pair session:
  - Replay the symptom: a Claude Code session with hyperlink-heavy output (TODO updates, file mentions). Confirm zero BEL emits, but `BEL-skip:` lines still appear in the log.
  - Idle Claude for ~60s; confirm `OSC 777` still forwards as `OSC777:` + `EMIT:`.
- [x] Truncate `~/pair-wrap.log` after verification so it doesn't carry pre-fix noise into future debugging sessions.

## Log

### 2026-05-03

- Filed from live data analysis of `~/pair-wrap.log` during the issue #13 work session: 76 EMITs, 8 legitimate (OSC 777), 68 spurious (BEL fallback firing on hyperlink/title tails).
- Considered a per-agent rule table; deferred until we have a concrete second-agent divergence to motivate it. Today's `is_actionable_osc()` is sufficient and the noise is entirely in the BEL fallback.
- Implementation: 14-line change to `bin/pair-wrap` (add `BELL_FALLBACK` env, branch on it in the bare-BEL elif, log `BEL-skip` instead of `BEL` when off). Synthetic test confirmed both branches behave as specified.
