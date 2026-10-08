---
id: 000411
status: open
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: 'eee516bb6361d7e9cc5781c4bbdeff382686ee3e' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch peer-delivery receiver profile for grok

## Problem

Couch delivers peer messages (`couch --send-to`) only to agents with a Pair
receiver profile: today Claude Code and Codex CLI (`peerReceiverAgents`,
`cmd/internal/wrapcmd/peer_runtime.go`). pair#410 brought Grok to harness parity
but deliberately left peer delivery out, so a Grok slot cannot receive Couch
messages.

## Spec

Add a Grok receiver profile, following how the Claude and Codex profiles were
qualified (`atlas/couch.md`, "Receiver profiles"). Composer recognition, the
Return remap (ESC CR newline, CR send) and picker detection already exist from
pair#410. What remains is the delivery path: bracketed paste into the live
composer, then submit, with the composer, menu and image-state refusals the
other profiles have.

## Done when

- `grok` is in `peerReceiverAgents`, with tests mirroring the Codex profile's.
- Live: `couch --send-to <grok slot>` lands a message in a Grok composer and
  submits it; an occupied picker refuses rather than force-pasting.

## Plan

- [ ]

## Log

### 2026-10-08
