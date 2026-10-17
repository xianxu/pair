---
id: 000632
status: open
deps: [pair#625]
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'ad311d1fa6bdfb6436596e48451fddb77f7d0f59' # card fields mirrored from issue-cards; edit via sdlc
---

# Keep background-command waits bright without blocking peer admission

## Problem

#625's admitted-wrapper work timestamp fixes open and completed peer-driven turns. A separate background-command case cannot use Settled: a free composer must continue receiving messages while a 10–30 minute suite runs in the background.

Read-only live evidence on 2026-10-16: ariadne:2, Claude 2.1.296, shows an empty composer, "1 shell still running" and a "1 shell" footer while waiting for its recorded suite. The authenticated resource snapshot at 19:33:16 PDT reports idle_known=true, idle=true, implying Settled=true. CPU/rate evidence was partial and is not the work proof.

The existing SessionFrame/SlotLiveness snapshot has no independent background-work field. Detached threads do not give Couch an agent-pane snapshot; peek reads it on demand via messageService.handleTail -> TailReader.Tail. Polling that endpoint would add an IO probe. Checked-in peer/claude/2.1.296 shell-footer.raw and idle-suggestion.raw contain the same positive shell counter.

## Spec

Provide display activity for running background commands independently of message/restart admission. Reuse an already-owned observation where available; if a new wrapper field is necessary, explicitly design its negotiated compatibility, identity/lifetime and bounded cache semantics. Do not set Settled=false solely because background commands exist. Preserve message receipt to a free composer while the command runs.

## Done when

- A background-command-only wait remains bright despite file activity older than a day, including a detached thread.
- The free composer continues receiving peer messages during that wait, with unchanged admission behavior.
- Completed/zero-work state eventually ages normally; quoted body/composer text and stale/disconnected/replaced wrapper evidence do not count as work.
- Recorded positive footer and negative controls precede code; live conformance checks both brightness and successful message receipt under a running background suite.

## Plan

- [ ] Design the independent display signal and its lifecycle/compatibility before implementation.

## Log

2026-10-16: filed from #625 under TL decision19e11f39-5e0e-46cf-8fe8-a1fbc76d223b; current #625 lands its existing work-state projection plus approved LastActivity reconnect seed. This independent surface follows that landing; no stacked code or live rollout performed here.
