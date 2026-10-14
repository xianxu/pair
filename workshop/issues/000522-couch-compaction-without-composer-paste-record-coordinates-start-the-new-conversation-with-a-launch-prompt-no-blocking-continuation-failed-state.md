---
id: 000522
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-13
estimate_hours:
card_mirror: 'dd003d5c78c1a8d6cdcc28b3bba19ce9e4ee166a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T17:50:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couch: compaction without composer paste: record coordinates, start the new conversation with a launch prompt, no blocking 'continuation failed' state

## Problem

Compaction (Alt+Shift+C, and agents' continuation writer) saves a continuation, replaces the conversation, then pastes an orientation prompt into the new agent's composer. On 2026-10-12 pair:2 (Codex) restarted twice with an empty composer: the paste never landed, the new session sat idle until the TL noticed, and the thread was left in a hidden 'continuation failed' state that blocked the next continuation, relaunch and agent switch until the operator found dismiss-continuation in the switcher (it shows nowhere in the pane). Likely cause: the paste is gated on recognising the agent's composer state, and a fresh agent's startup screen (tips, warnings, trust prompts) isn't recognised, so delivery waits, times out and fails. Related: pair#414 (orientation lost on a Grok switch), pair#517.

## Spec

The operator (2026-10-12): record the last conversation's coordinates and let the new conversation decide; decouple compaction from the agent switch (pair#517 owns the switch and the coordinates record).
1. Compaction records the coordinates (pair#517's previous-session descriptor) plus the continuation document's path.
2. The new conversation starts with a launch prompt instead of a composer paste: `claude --resume`/`codex resume`/`grok --resume` (or a fresh start) with PROMPT = one line pointing at the continuation. pair#517's audit (2026-10-12) found all three accept a prompt argument and Pair passes AgentArgs through; never save the one-time prompt into the thread's relaunch argv (it would replay).
3. No blocking state: if the launch fails, the slot shows a visible line with the continuation path and the next compaction or relaunch simply proceeds. Retire 'continuation failed', retry-continuation and dismiss-continuation, or reduce them to that one visible line.

## Done when

- A Claude and a Codex slot compact and the new conversation picks up its continuation with no paste; a forced launch failure shows the visible line and blocks nothing; the one-time prompt never reappears on a later relaunch; tests cover each.

## Plan

Durable plan: [compaction launch prompt plan](../plans/000522-couch-compaction-without-composer-paste-record-coordinates-start-the-new-conversation-with-a-launch-prompt-no-blocking-continuation-failed-state-plan.md).

- [ ] M1 — Launch prompt (one-time, never persisted), coordinates recorded on compaction, completion at registration, one non-blocking line.
- [ ] M2 — Retire the composer paste and the retry/dismiss surface; sweep references.
- [ ] Live: a Claude and a Codex slot compact with no paste, after the operator's OK through the TL.

## Log

### 2026-10-12
- 2026-10-13: claimed at pair:3 (TL dispatch). Measured the pipeline; plan written and sent to the hub before building.
