---
id: 000629
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '55f9417d809c37fa966a3650488d672558b647f9' # card fields mirrored from issue-cards; edit via sdlc
---

# Peer messages to a busy hub expire after 1m30s waiting for a recognized composer; landing reports lost

## Problem

## Spec

Peer messages to a busy hub expire undelivered. `couch --message-status` reads
`expired — delivery deadline elapsed: waiting for recognized composer (1m30s)`.

Seen 2026-10-16 after the 17:35 Couch restart:
- ariadne:2 -> ops:0 33f83c87 and its resend 2fd247eb (ariadne#412's landing report): both expired. The hub never saw the landing; it found it by peeking ariadne:2.
- pair:7 -> ops:0 f854048e (pair#604's landing report) expired; pair:7 resent by hand later.

The hub (ops:0) is a Claude pane that is mid-turn most of the time, running tools. Its composer is not "recognized" while a turn runs, and 90 s is shorter than a typical TL turn. A sender can't tell when to retry, so each slot invents its own resend loop.

Expected: a message to a busy agent waits until its composer is next recognized (Claude Code also queues typed input mid-turn, so delivering while busy may be fine). It should never silently expire on a live, connected receiver. Expiry stays for a receiver that is gone.

## Done when

- A message to a connected receiver whose turn runs past the deadline is delivered when its composer next comes back, or queued into the running turn; a test with a fake receiver busy for longer than the deadline shows it delivered once.
- Expiry happens only when the receiver is disconnected or gone, and the receipt says which.

## Plan

- [ ]

## Log

### 2026-10-16
