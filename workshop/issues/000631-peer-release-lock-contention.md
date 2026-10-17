---
id: 000631
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '2399fb273250dd64c842826f2c6ae66168aa7a5c' # card fields mirrored from issue-cards; edit via sdlc
---

# Peer promotion holds broker lock during wrapper release RPC

## Problem

During #629 investigation, Broker.promote was found calling the wrapper endpoint's Release RPC while holding the shared broker mutex on its error path. A slow Release can delay independent recipients' Send and Status calls until the RPC timeout.

This is a separate static finding. No retained evidence ties this lock contention to the 2026-10-16 17:37–18:22 hub outage; #629's confirmed mechanism is serial pre-paste expiry.

## Spec

Move wrapper release IO outside the shared broker lock while preserving exact actor/reservation ownership and promotion retry pacing. Use a gated stateful endpoint to prove another recipient's admission and status can progress while Release is withheld.

## Done when

- A gated Release test fails on the original locking path and passes after the change.
- Another recipient completes Send/Status while the first receiver's Release is blocked.
- Existing promotion/reconnect/duplicate-ID tests and targeted race checks pass.

## Plan

- [ ]

## Log

- Filed separately at the TL's direction during #629 planning; unclaimed for later dispatch.
