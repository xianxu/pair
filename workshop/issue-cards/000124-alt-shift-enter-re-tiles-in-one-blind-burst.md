---
id: '000124'
status: done
started: 2026-07-28T08:11:35-07:00
created: 2026-07-28
updated: 2026-07-28
estimate_hours: 0.15
actual_hours: 0.21
---

# Alt+Shift+Enter re-tiles in one blind burst

## Problem

The #123 tiled toggle re-tiles the terminal column via a converge loop:
read geometry → one resize step → 80ms settle (zellij applies resizes
asynchronously) → re-read → repeat until within tolerance. Live it takes
~3 visible steps and ~500ms — the user reports it feels slow.

Zellij 0.44.3's tiled resize step is a fixed fraction of the screen:
5% per `resize increase|decrease left` (RESIZE_PERCENT; measured live
7–8 cols on a 150-col screen — the `* 2.0` source path does not apply
to this action, an earlier misread). The toggle delta (1/2 ↔ ~2/3) is
therefore always exactly three steps — measuring and settling buys
nothing (user decision: "just do 1/3, 2/3 expansion").
