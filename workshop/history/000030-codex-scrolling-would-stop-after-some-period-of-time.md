---
id: 000030
status: done
estimate_hours: 1.0
deps: []
created: 2026-05-31
updated: 2026-06-01
actual_hours: 0.8
---

# codex: scrolling would stop after some period of time

on a new session, scrolling in agent pane works, but then it stops. observed so far only in codex. anecdotally, this seems to happen if I scroll the screen during codex's generating.

when such thing happen, there seems to be some minor corruption in the screen.

## Done when

- The root cause of Codex pane scrolling stopping is identified from captured output or a focused reproduction.
- The fix preserves Codex inline mode and normal pair scrollback capture.
- Verification covers the relevant terminal sequence.

## Spec

Codex runs under `--no-alt-screen` so conversation output should flow through the
normal screen and zellij scrollback. The user observed that scrolling works early
in a new Codex session, then later stops, possibly after scrolling while Codex is
generating. When it happens, the agent pane appears to have minor screen
corruption.

The current machine does capture screen output by default through
`pair-wrap --scrollback-log` from the zellij layout. For tag `pair`, the live raw
capture is `/Users/xianxu/.local/share/pair/scrollback-pair-codex.raw`, with
resize events in `scrollback-pair-codex.events.jsonl`. `PAIR_WRAP_LOG` is a
separate debug log and is not enabled by default.

Inspection of the raw stream showed Codex was not using alternate screen and was
not enabling mouse reporting. It did emit thousands of balanced DEC synchronized
output markers (`ESC[?2026h` / `ESC[?2026l`) while redrawing a fixed screen
region. That is a plausible zellij live-scroll interaction: the markers are
terminal presentation hints, not content, and pair's raw scrollback replay remains
usable without them.

## Plan

- [x] Inspect the captured Codex raw stream for terminal modes/sequences around the observed corruption.
- [x] Compare Codex inline-mode output against pair/zellij scroll assumptions.
- [x] Identify the smallest fix or logging improvement needed for a reliable reproduction.
- [x] Verify with a focused test or a live reproduction.

## Log


- 2026-06-01: closed — raw scrollback showed balanced Codex ESC[?2026h/l sync markers, no alt-screen/mouse toggles; added Codex-only stdout filter with tests; env GOCACHE=/private/tmp/pair-go-cache go test ./cmd/pair-wrap -count=1; env GOCACHE=/private/tmp/pair-go-cache make test; make pair-wrap
### 2026-05-31

### 2026-06-01

- Confirmed the current Codex session has default raw scrollback capture enabled:
  `scrollback-pair-codex.raw` and `scrollback-pair-codex.events.jsonl` are live.
  No default `PAIR_WRAP_LOG` artifact was present; that log is opt-in.
- The capture had 3,473 balanced `ESC[?2026h` / `ESC[?2026l` synchronized-output
  marker pairs, no alternate-screen toggles, and no mouse-reporting toggles.
  Replaying with `bin/pair-scrollback-render` produced a sane 2,037-line
  scrollback, so the raw capture path is intact.
- Added a Codex-only stdout filter that strips synchronized-output markers before
  bytes reach zellij, while preserving the original raw stream for capture and
  detection.
- Verified with `env GOCACHE=/private/tmp/pair-go-cache go test ./cmd/pair-wrap
  -count=1`, `env GOCACHE=/private/tmp/pair-go-cache make test`, and `make
  pair-wrap`.
