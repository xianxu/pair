---
id: 000891
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'a6ef845b06ad5c37776ffa7bafb5648ea0fa93d1' # card fields mirrored from issue-cards; edit via sdlc
---

# Codex 0.164's shared background server prompt stalls every wrapped start; launch so it can't appear

## Problem

## Spec

**Codex 0.164 stops every Couch-launched start at a "Background server has incompatible feature settings" prompt
(10-25).** Codex 0.164 runs a shared background server for every session on the machine. Couch launches Codex with
`shell_snapshot` disabled; once a server is running with other settings (here, a Codex started outside Couch right
after the update), each Couch launch stops at a three-way prompt: run without daemon this time / restart the shared
server with these settings (which can interrupt other sessions) / cancel. Couch can't answer prompts (`couch --keys`
refuses), so the slot sits stalled until the operator presses a key; pair:11 did this twice on 10-25. Every Codex
slot will meet it on its next start.

`pair wrap` launches Codex so the prompt can't appear: either opt out of the shared server for wrapped sessions (the
equivalent of "run without daemon", via a supported flag or config if 0.164 has one), or launch with settings that
match the shared server's. Never restart the shared server from a slot. Also: Couch records the Codex version per
launch and surfaces a version change to the hub, so the next upstream change is noticed before slots stall.

## Done when

- A wrapped Codex 0.164 start with a mismatched shared server reaches its composer with no prompt (native test, or a
  documented manual check if the server can't be faked), and the old-version path is unchanged.
- A Codex version change since the last launch is reported to the hub once.

## Plan

- [ ]

## Log

### 2026-10-25
