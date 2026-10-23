---
id: 000162
status: codecomplete
created: 2026-09-01
updated: 2026-10-23
estimate_hours: 1.97
github_issue:
started: 2026-10-22T18:09:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-6b670fd9e3e4
        repository: github.com/xianxu/pair
        reviewed_head: 51f0a6e0cdaf9847904dff59821237bb8bb3e926
        evidence_commit: e8ea9c785b62743e4dc4d6d4df6b59fd26813f1d
---

# Enter sends when the composer line is addressed to the harness

## Problem

In the **agent pane** — the Claude/Codex tty itself, not the nvim draft —
sending `/login` costs Alt+Enter. So does `/clear`, `/model opus`, `/context`,
`/resume`. Natively, every one of these agents submits on plain Enter; Pair is
what took that away.

That was deliberate. `cmd/internal/wrapcmd/wrap.go:127-143` documents it: the
draft uses Enter = newline / Alt+Enter = send, the agent's native TUI uses
Enter = send, "that mismatch is jarring when the user moves between panes", so
pair-wrap rewrites stdin to give the agent the inverted mapping too
(`sendKeymap`, `PAIR_WRAP_REMAP_RETURN`). For prose the inversion is right and
should stay. For a slash command it is pure cost: it charges a modifier for the
shortest, most frequent input, and it is *subtracting* behavior the agent
shipped with.

The path is already traced end to end:

- `decidePlainReturn` (`cmd/internal/wrapcmd/harness_tty.go:90`) is the single
  place a plain Enter is resolved.
- Its first branch is the overlay bypass, which does **not** cover this. Claude's
  detector keys on OSC 777 with body `notify;Claude Code;Claude needs your
  permission` (`wrap.go:650-688`) — a permission prompt. The slash-command
  dropdown is not that, so `overlayActive` stays false.
- So the `composerGatePositive` branch runs, `claudeComposerActive` correctly
  reports the cursor is inside the composer's ruled box, and the decision is
  `remap()` → `plainCR` = `\<CR>` → **a newline in the composer.**

Nothing is malfunctioning. The gate answers the only question it is currently
able to ask — *is a composer active?* — and for a slash command the right
question is a different one.
