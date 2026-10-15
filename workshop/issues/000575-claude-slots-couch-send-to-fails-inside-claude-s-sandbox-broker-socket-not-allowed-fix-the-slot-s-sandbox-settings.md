---
id: 000575
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '807a642152cd1b2d88c128b4e14f81e54f20cfa0' # card fields mirrored from issue-cards; edit via sdlc
---

# Claude slots: couch --send-to fails inside Claude's sandbox (broker socket not allowed); fix the slot's sandbox settings

## Problem

On Claude Code slots, `couch --send-to …` run inside Claude's default sandbox times out ("outcome uncertain; no
recipient retains it"), although inbound delivery to the same slot works; it succeeds only when the agent runs it with
the sandbox disabled (reported by ariadne:2 on 2026-10-15; the TL on ops:0 also runs every couch call with the sandbox
off). So the per-call sandbox exemption pair installs for Claude slots doesn't match `couch` sends, and every worker
report from a Claude slot either fails or needs a sandbox override, which the agent must justify each time.

## Spec

- Find what the sandbox blocks for `couch --send-to` (likely the Couch broker's unix socket, absent from the sandbox's
  `allowUnixSockets`, or the command not matched by the exemption pattern pair writes into the slot's Claude settings).
- Fix it at the source: pair's Claude slot settings allow the broker socket path (exact path, not a wildcard), or the
  exemption matches all `couch` verbs that talk to the broker.
- No broader sandbox loosening.

## Done when

- In a Claude slot with the default sandbox, `couch --send-to hub --message …` succeeds with a receipt, and
  `couch --message-status` works; a conformance test or a documented live check covers it.

## Plan

- [ ]

## Log

### 2026-10-15
