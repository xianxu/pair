---
id: '000251'
status: done
started: 2026-09-14T08:57:57-07:00
created: 2026-09-14
updated: 2026-09-14
estimate_hours: 1.74
actual_hours: 1.31
---

# Keep Ctrl+Return notification jumps working across Couch thread switches

## Problem

On 2026-09-14 the operator reported that Ctrl+Return stopped jumping to the
newest pending notification: another thread remained yellow in the status bar,
no Couch notice appeared, and Return was inserted into the agent. The recent
thread shortcut still worked. The operator confirmed Ctrl+Space then Return
successfully reached the notification. Thus notification targeting works; the
shortcut is failing before its recognized handler.

`newestPageSequence` accepts `ESC[13;5u` only. The existing tests feed those bytes
directly and pass, so they cannot detect loss of terminal keyboard mode.
Couch currently depends on Zellij's output to enable Kitty disambiguation.
`switchTo` replays a bounded raw output tail; it can omit an aged-out enable or
replay an unmatched pop. `ptychild.Screen` tracks no keyboard flags, and
`hostty.RepaintFor` does not restore them. Child resets and primary/alternate
buffer changes can also remove the mode. The symptom is consistent with the
terminal producing legacy CR; the historical triggering bytes were not captured.
