---
id: 000686
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'cefe35e6e4d94e94259eb5563944d81fb78c85fa' # card fields mirrored from issue-cards; edit via sdlc
---

# session-watch busy-loops on freshly started Codex threads (45-80% CPU each)

## Problem

On 2026-10-18 at 18:47, the `pair session-watch codex` processes of the two Codex slots that were just started fresh were busy: pid 70442 (`session-watch codex 1-pair-69`, pair:0, started 18:17) at 45–81% CPU and pid 84747 (`session-watch codex 1-parley-nvim-70`, parley.nvim:0, started 18:22) at 34–70% CPU, continuously. Older Codex slots' session-watch processes sit at 0%. The machine load average was 55 at the time (suites and builds too), and slot launches were timing out on zellij pane birth.

Likely shape (unverified): a fresh Codex thread has no bound rollout yet, and session-watch polls or rescans ~/.codex/sessions in a tight loop looking for it.

## Spec

session-watch must idle at about 0% CPU while it waits for a fresh thread's transcript: bounded, backed-off polling or file-system notifications, with work that doesn't grow with the size of ~/.codex/sessions per tick.

## Done when

- A regression with a fresh thread and a large sessions tree shows bounded work per tick (counted), and no busy loop.
- On the live machine, a freshly started Codex slot's session-watch stays near 0% CPU (ps sample).

## Plan

- [ ]

## Log

### 2026-10-18
