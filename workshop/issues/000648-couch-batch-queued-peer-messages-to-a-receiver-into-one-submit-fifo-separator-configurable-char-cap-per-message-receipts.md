---
id: 000648
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9790ac12fb6b30eb3bebd1565ef45d6bfdacec67' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: batch queued peer messages to a receiver into one submit (FIFO, separator, configurable char cap), per-message receipts

## Problem

## Spec

Operator decision, 2026-10-17, prioritized because the TL thread is under pressure. When a receiver's composer becomes available and two or more peer messages are queued for it, Couch submits them as one paste instead of one turn each.

Today (pair#629) each queued message drains as its own submit, so each starts a new agent turn. For the hub that means a full turn of context and overhead per report. Every message already carries its own `[Couch peer from SLOT; delivery ID]` header, which is how the receiver tells senders apart, so turn boundaries carry no information the receiver needs.

Rules:
- **FIFO:** messages keep queue order inside a batch, with a clear separator line between them, and each keeps its header exactly.
- **A size cap:** a batch holds messages up to N characters. N is configurable, for example `COUCH_PEER_BATCH_MAX_CHARS`, with a safe default under the smallest known composer paste limit. pair:3 measures Claude Code's and Codex's paste limits to choose the default. A message larger than N goes alone, as today. Remaining messages wait for the next free composer.
- **Receipts:** every message in a batch gets its own receipt. All of them go to submitted on the one confirmation, and all of them go to indeterminate or refused together if the paste fails. No message is reported delivered unless its batch was.
- **Scope:** the hub first. Any receiver, if it is as safe; say which.
- pair#629's note that "same-receiver input cannot be batched safely": say whether it was about receipts, which the rule above covers, or something else, and handle it.

## Done when

- A fake busy receiver with five queued messages gets one paste with all five in FIFO order, separated, and five submitted receipts. A test covers it.
- A cap smaller than the total splits the messages into ordered batches, and a single over-cap message goes alone. Tests cover both.
- A failed batch paste marks every message in it as not submitted.
- The live check after a restart: the hub, made busy with ten queued reports, receives them in one or two turns, not ten.

## Plan

- [ ]

## Log

### 2026-10-17
