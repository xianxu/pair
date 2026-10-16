---
id: 000569
status: done
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 1.924
github_issue:
started: 2026-10-15T16:14:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 3.28
tracker:
    version: 1
    completion:
        token: close-48db62f6845b
        repository: github.com/xianxu/pair
        reviewed_head: 1d117dfd648001b3dfda370a0bde9cf0e30f97db
        evidence_commit: 9a943c887932fb44aa69d311829a80600278d73e
        landed_commit: 704529cfab53d4fd1378687b594828110f5fecae
---

# Couch → Qoder delivery: text typed into Qoder's input but not submitted (pending * prompt); confirm submission

## Problem

2026-10-15 ~07:55 PDT: a hub message to pair:4 (Qoder CLI) was typed into Qoder's input box but never submitted. The
pane showed it as a pending `* [Couch peer from ops:0; delivery b2fb1254…]` prompt while Qoder sat idle (no generation
running), so the worker never received the hub's answer and stalled. Earlier deliveries to the same Qoder session were
submitted and acted on, so it's intermittent; it likely happens when the delivery lands while Qoder is busy (Qoder
queues typed input as a `*` pending prompt) and Qoder doesn't auto-submit the queue when its turn ends, or the submit
key is dropped by a paste-burst guard (compare #544's Codex paste-burst fix).
