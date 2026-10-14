---
id: 000522
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-13
estimate_hours: 4.83
card_mirror: '08865b9a7aa1a3403d944005c59e24fd3d8dfbfd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T17:50:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 3.07
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

- [x] M1 — Launch prompt (one-time, never persisted), coordinates recorded on compaction, completion at registration, one non-blocking line.
- [x] M2 — Retire the composer paste and the retry/dismiss surface; sweep references.
- [x] Live: a Claude and a Codex slot compact with no paste, after the operator's OK through the TL.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module        design=0.3 impl=0.6
item: smaller-go-module        design=0.4 impl=0.8
item: cross-cutting-refactor   design=0.3 impl=1.0
item: tui-screen               design=0.1 impl=0.3
item: atlas-docs               design=0.0 impl=0.3
item: milestone-review         design=0.0 impl=0.4
design-buffer: 0.30
total: 4.83
```

- smaller-go-module 1: the one-time prompt end to end (PromptArgs, AgentCommand.Prompt, profile admission, no-replay tests).
- smaller-go-module 2: the request state machine (completion at registration, MissContinuation, Failed decode, guards) and coordinates.
- cross-cutting-refactor: M2 retires the paste machinery and retry/dismiss across wrapcmd, couchcore, couchtty, couchcmd and the tests.
- tui-screen: the switcher row's line and the notice.
- milestone-review ×2: M1 and M2 boundaries.

## Log




- 2026-10-13: closed — Live check PASSED (TL, Couch restarted 21:15 onto 7775cb260): pair:4 Claude and pair:5 Codex compacted; first input is the launch prompt (argv[1] in ps), nothing pasted; --previous shows the replaced conversation; thread records: request complete, previous_session.continuation set, prompt in no record/ledger/config; relaunch of pair:4 resumed with --resume and no prompt, idle. M1 SHIP (2 rounds), M2 SHIP (4 rounds), advisories fixed after (fixture home, orphaned symbols). Full make -k test green on e9611d97d except TestRealRunnersExitedPrecedesTheGroup (pty load flake, untouched; 3/3 alone); merge with main 7775cb260: continuation/resume/relaunch/prompt tests green.; review verdict: SHIP
- 2026-10-13: closed M2 — Round 3 fixes in the latest commit: BR-9 rule applied -- a written list of every retired identifier/path/env/op/fixture dir (in plan Revisions) grepped across all tracked file types; all hits cleared (nvim/draft_send.lua, wait_notice.go, launcher/args.go comments; switchcontext/guard test names; codex 0.154.0 captures moved to testdata/tty, orphaned agy captures and .gitattributes line removed); tombstone: removal condition stated and Encode refuses it (test); Settled-without-gate disposed with reasoning in plan. wrapcmd, couchcore, readiness, artifactpath, couchcmd green unsandboxed. Earlier full make -k test on M2 was green apart from the render-epoch regression that test caught (fixed).; review verdict: SHIP
- 2026-10-13: closed M1 — Round 1 REWORK fixed in 887a44bf9: BR-1 executeContinuation keeps its record on a RecordPreviousSession error and failContinuation refuses a record without its request (TestFailContinuationWithoutItsRequestReturnsTheCause); BR-2 PromptArgs puts the prompt first (agy -i first) so no multi-value saved flag swallows it (table incl --add-dir a b, codex --image; real wrapper/layout acceptance asserts argv[0]==prompt, once); BR-3 launchEnvCommand split + TestAnInPaneFreshStartDropsTheLaunchPrompt; minors: targetMayExist single rule, stale comments, Miss.Path rationale, strict-decode note in plan. Packages checkpoint/launcher/wrapcmd/couchcmd/artifactpath green unsandboxed; couchcore green except TestFleetInventoryLiveConformance (pair:0 staged: the #539 'staged' grammar gap, unrelated). Full make -k test was green on ec93571ae+ca52528bd apart from a docs check since fixed.; review verdict: SHIP
### 2026-10-12
- 2026-10-13: claimed at pair:3 (TL dispatch). Measured the pipeline; plan written and sent to the hub before building.
- 2026-10-13: TL approved the plan. Plan-quality cleared in 2 rounds (PQ-1: a target that may still be alive is guarded by the incarnations; sequence test added). M1 built:
  - the prompt rides beside argv (`AgentCommand.Prompt`, `launcher.PromptArgs`);
  - `PreviousSession.Continuation`;
  - Registered completes the request;
  - `checkpoint.Miss` with `ThreadStore.MissContinuation`;
  - Failed decodes into the line;
  - retry/dismiss/status and copy-orientation retired.
  The boundary moved: see the plan's Revisions.
- 2026-10-13: **live check PASSED**: staged 7775cb260 on pair:0 (`sdlc stage --build`); the operator restarted Couch at 21:15.
  - pair:4 (Claude) and pair:5 (Codex) compacted. Each new conversation's first input was the launch prompt, which `ps` showed as argv[1]. Nothing was pasted.
  - `--previous` showed the replaced conversation.
  - The thread records show the request complete, `previous_session.continuation` set, and the prompt in no record, ledger or config.
  - The relaunch of pair:4 at 22:33 was `--dangerously-skip-permissions --resume <id>` with no prompt, and the agent sat idle.
