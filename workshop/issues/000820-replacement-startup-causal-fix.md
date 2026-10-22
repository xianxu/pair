---
id: 000820
status: open
deps: [pair#806]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: '8c850375d0c5e325a239c825893439716bcb764a' # card fields mirrored from issue-cards; edit via sdlc
---

# Classify retained replacement startup failure

## Problem

The replacement-startup disconnect originally captured under #797 and investigated in #806 remains causally unclassified. Its private server log was removed during cleanup. Passing controls show that Server not ready warnings alone do not establish the cause. #806 now retains bounded server-log tails and launch identities for the next failure.

## Spec

Wait for the first retained replacement-startup failure from #806's diagnostic coverage before claiming or implementing this causal correction. Correlate the exact replacement session, wrapper scope/tag/launch nonce, PID/start generations, public operation chronology and private server tails. Distinguish a server panic, external termination and client-admission failure from downstream registration cancellation. Propose the smallest correction covering the measured class; get hub approval before production code. No speculative one-poller patch, repeated native retry loop, deadline change or weakened ownership/ready proof.

## Done when

- A retained failure identifies the initiating mechanism, with missing/truncated evidence explicitly scoped.
- An approved correction is pinned by a causal regression through the shared startup seam.
- Bounded disposable Claude and Codex public startup/reboot, message delivery and exact-generation cleanup pass.

## Plan

Blocked on evidence, not merely #806 landing. Preserve the first retained failure and return its classification and correction design to the hub before code. Use the shared disposable Couch harness and existing time/frame/output/96MiB write bounds. Independent global readers exist, so any exclusion design must cover its declared complete operating envelope.

## Log

Filed unclaimed per authenticated ops:0 delivery7e2c8fdd-2dc2-4419-96e1-34da5d1eb6df. #806's one native control each for Claude and Codex passed on c90b43e6e99f29a8cd6dbf2546262752745d6972; these do not explain the historical failure or justify a flake registration. #806 carries raw evidence paths/hashes and query-source census. This issue waits for the first retained failure; no new native investigation is dispatched by its filing.
