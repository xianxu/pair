---
id: '000262'
status: done
started: 2026-09-17T18:51:31-07:00
created: 2026-09-15
updated: 2026-09-17
estimate_hours: 2.45
actual_hours: 2.51
---

# Screen flicker: the compositor re-emits global terminal state every frame (#255)

## Problem

The screen sometimes flickers subtly during fast Neovim input, without visible
corruption. Holding Delete and deleting one character at a time can trigger it;
running a program or producing heavy output in the right pane does not show the
same symptom.
