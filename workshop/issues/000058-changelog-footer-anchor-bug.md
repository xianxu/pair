---
id: 000058
status: working
deps: []
github_issue:
created: 2026-06-12
updated: 2026-06-12
estimate_hours: 1
---

# pair-changelog incremental fails: volatile-footer anchor → FullRedistill → model timeout

## Problem

Live bug found dogfooding #53: after the first distill, the change log **stops
refreshing** and every `Alt+l` re-ships the whole transcript.

Reproduced on the live session (`changelog-pair-claude.*`):

```
$ pair-changelog --cleaned <live.cleaned> --log t.md --anchor t.anchor --agent claude
pair-changelog: distilling 3110 lines
pair-changelog: model: signal: killed        # 30.017s — hit modelTimeout
# t.md UNCHANGED — distiller died before writeLog/writeAnchor
```

Root cause — a two-bug compound, both from the **anchor landing in claude's
volatile live footer**:

1. The anchor snippet is the last K cleaned lines, which in a live session is the
   **current input box + horizontal rule + status line** (`❯ ` / `───…` /
   `⏵⏵ bypass permissions on · ctrl+t to show tasks · …`). That status text
   changes every render, so `locate` never re-finds the anchor → **`FullRedistill`
   every press** → the *entire* transcript is sent to the model.
2. The full transcript (3110 lines / ~140 KB) **exceeds the 30s model timeout**
   (haiku) → the process is killed → the distiller exits *before* writing the log
   or anchor → log frozen, anchor stuck at `turns:4`. Next press repeats.

Silent to the operator: the viewer's `on_stderr` only matches `distilling N` /
`up to date`, so a killed/errored distill just clears the spinner and reloads the
unchanged log — no failure signal (the reviewer flagged this on #53 as a Minor;
this incident shows it matters).

## Spec

Two robustness fixes in `cmd/pair-changelog` + one viewer fix:

### 1. Trim the volatile live tail before anchoring/slicing/counting (root cause)

A pure `trimLiveTail(lines, agent) []string`: scan the last ~20 lines for the
**empty prompt box** (a line whose trimmed content is exactly the agent's prompt
glyph — `❯` / `›` / `>`) and drop everything from it to EOF (the box + rule +
status are all live chrome below the last committed content). `main.go` applies
it to the cleaned lines first, so the anchor lands on **committed scrollback**
(stable across presses) → `locate` finds it → genuinely incremental. Turn-count
and slice then also exclude the live box. Mid-response (no empty box in the
window) → returns unchanged (degrades to distilling the live region; the cap
below bounds it).

### 2. Cap the model input (safety net)

Bound the slice fed to the model at `maxSliceLines` (~800). On first-run /
`FullRedistill` over a long transcript, take the **last** `maxSliceLines`. This
guarantees no 30s timeout regardless of anchor edge cases; the prior log (fed as
read-only memory) preserves older entries on a capped re-distill. Coverage note:
a fresh first-run on a very long pre-existing session only sees recent activity —
acceptable (it's a *recent*-changes glance; incremental keeps it current; better
than a total-failure timeout).

### 3. Surface distill failure in the viewer

`changelog.lua`'s background-job `on_exit` should detect a non-zero exit and show
a brief error in the bottom virtual line (instead of silently reloading the
unchanged log), so a failed refresh is visible.

## Done when

- Running the distiller on the live `changelog-pair-claude.cleaned` writes the
  log (no `signal: killed`) — verified end-to-end on the real state.
- `trimLiveTail` is pure + unit-tested (empty-box trimmed; rule/status after it
  trimmed; mid-content preserved; no-box → unchanged; per-agent glyphs).
- The model input is capped: an integration test feeds a >cap cleaned and asserts
  the fake model received ≤ cap lines on stdin.
- A no-op test where only the trailing footer changed (content identical) now
  fires the no-op (was the bug: footer churn forced a re-distill).
- The viewer shows a failure signal when the distill job exits non-zero.

## Plan

- [ ] `trimLiveTail` + glyph-char map (pure) + unit tests.
- [ ] Apply in `main.go` (trim first; cap the slice); integration tests for cap +
  footer-churn no-op; verify live against `changelog-pair-claude.cleaned`.
- [ ] Viewer failure signal in `changelog.lua` on_exit.

## Log

### 2026-06-12

- Found + root-caused live (see Problem). Single-pass atomic fix (no milestones).
