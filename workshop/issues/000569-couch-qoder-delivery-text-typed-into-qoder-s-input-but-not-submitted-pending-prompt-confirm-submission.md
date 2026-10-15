---
id: 000569
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'deb8c85bb8b36e7870c60d40b35e9d80f07483e5' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch → Qoder delivery: text typed into Qoder's input but not submitted (pending * prompt); confirm submission

## Problem

2026-10-15 ~07:55 PDT: a hub message to pair:4 (Qoder CLI) was typed into Qoder's input box but never submitted. The
pane showed it as a pending `* [Couch peer from ops:0; delivery b2fb1254…]` prompt while Qoder sat idle (no generation
running), so the worker never received the hub's answer and stalled. Earlier deliveries to the same Qoder session were
submitted and acted on, so it's intermittent; it likely happens when the delivery lands while Qoder is busy (Qoder
queues typed input as a `*` pending prompt) and Qoder doesn't auto-submit the queue when its turn ends, or the submit
key is dropped by a paste-burst guard (compare #544's Codex paste-burst fix).

## Spec

- Reproduce: deliver to a Qoder slot while it's mid-turn, and while it's idle; record whether the text is submitted.
- Qoder's receiver profile must confirm submission (the prompt leaves the input box, or a turn starts) and retry the
  submit key if not, the same way the Claude and Codex profiles confirm.
- Delivery status must not report "delivered" for text that's only typed.

## Done when

- A live Qoder conformance check: deliveries mid-turn and idle are both submitted and acted on.
- A unit test of the Qoder profile's confirm-or-retry logic over captured screen states (pending `*` prompt vs. empty).

## Plan

- [ ]

## Log

### 2026-10-15
