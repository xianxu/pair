---
id: '000058'
status: done
created: 2026-06-12
updated: 2026-06-14
estimate_hours: 2
actual_hours: 8.99
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
