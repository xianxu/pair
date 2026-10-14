# #522 plan: compaction starts the new conversation with a launch prompt

Issue: `workshop/issues/000522-…md`.

## What's there (measured from the tree, 2026-10-13)

- **Flow:** Alt+Shift+C (the draft pane's `PairConfirmCompact`) and agents' own `pair continuation` both run the
  continuation writer. Its `pair continue --checkpoint` reaches `Couch.RequestContinuation`, which stores a
  `checkpoint.Request` (Pending) in `ThreadRecord.Continuation`. `Couch.Continue` → `executeContinuation` checks
  the source, parks it, writes a private copy (`materializeContinuation`) and launches a **fresh** conversation
  (`launchTrackedThread{Fresh: true, Orientation: &orient}`).
- **Delivery is a paste:** the orientation travels in `TrustedLaunchProfile.Orientation` → `PAIR_ORIENTATION_REQUEST`
  → `wrapcmd/orientation.go`. That code waits up to 30 s for each agent's composer recognizer, pastes and submits.
  Couch's `ReconcileContinuation` reads the result from the ready file and fails the request on anything but
  Submitted. It also fails it if nothing is confirmed within 30 s (a second clock).
- **Failure blocks:** `Phase == Failed` stays in the thread record. `continuationGuard`/`PublishContinuation`
  refuse relaunch, the agent switch, resume, reboot, start, archive, recovery and the next compaction until retry
  or dismiss (switcher items, `couch --internal retry-|dismiss-continuation`).
- **Coordinates are lost:** only the agent switch calls `RecordPreviousSession`. Compaction parks but never
  records it, so `couch --peek repo:N --previous` has nothing after a compaction.
- **No one-time argument exists:** anything in `Argv`/`ExtraArgs` is persisted three ways (the ledger,
  `LatestLaunchProfile`, the actor's `ExtraArgs`) and is also re-derived by the wrapper's in-pane fresh start
  (`freshAgentInvocation` builds from the running argv). A prompt there would replay.
- **Agents' prompt argument** (#517's audit, plus `--help` today):
  - `claude 'P'`, `codex 'P'`, `grok 'P'`, `qodercli 'P'`, `muse 'P'` (positional);
  - `agy -i 'P'` (`--prompt-interactive`).
- **Orientation users:** only compaction sends one; the agent switch starts with an empty composer (#517).

## Design

### 1. The one-time launch prompt (ARCH-PURE, SECURE)

- `TrustedLaunchProfile` gains `Prompt string`, allowed only with `FreshRequired`. It replaces `Orientation`.
- The prompt is never in `Argv`.
  - `launcher.AgentCommand` gains `Prompt`. The wrapper appends it only when it builds the exec argv
    (`launcher.PromptArgs(agent, argv, prompt)`, pure, a per-agent table: positional after the argv, or after an
    existing `--`; `-i P` for agy).
  - The ledger, `writeConfig`, `LatestLaunchProfile` and `StartArgs.ExtraArgs` never see it. The wrapper's in-pane
    fresh start re-derives from `Argv`, so a later fresh start, relaunch or resume can't replay it.
- **Text:** one line, built by a pure function. It names the continuation's private copy, the NEXT ACTION, and
  `couch --peek repo:N --previous` for the previous conversation.
  - The path is Couch's own, under its store, so it carries no shell metacharacters.
  - It reaches the agent as a single exec argument, never a shell string.

### 2. Coordinates (#517's descriptor)

- After a successful park, `executeContinuation` calls `RecordPreviousSession`, as the switch does, and adds the
  continuation copy's path (`PreviousSession.Continuation`).
- `--peek repo:N --previous` then works after a compaction. The record is the coordinates the issue asks for.

### 3. Completion without a paste

- The request completes when the target registers, which is the existing FreshNonce readiness: Registered →
  Complete. The prompt was part of the launch, so there is nothing to confirm.
- `ReconcileContinuation`'s orientation read, and its 30 s submission clock, go away.

### 4. One visible line, no blocking state (the operator's rule: few visible states, one action each)

| Visible state | When | The one action |
| --- | --- | --- |
| `compacting` | Pending/Running (seconds) | none: wait. It still serializes competing transitions, as today. |
| `compaction didn't finish: <reason>; continuation at <path>` | any failure | **relaunch**, which already exists; the next compaction also simply proceeds |

- A failure clears the request and leaves `ThreadRecord.ContinuationMissed{Path, Reason, At}`.
  - It shows on the slot's switcher row and as one console notice.
  - It blocks nothing, and the next successful start or compaction clears it.
- **Before park** (source changed, digest mismatch): the old conversation keeps running, and the line says so.
- **After park** (launch failed): no agent is running. The existing relaunch resumes the old conversation.
- **Retired:** `checkpoint.Failed` as a blocking phase, `RetryContinuation`, `DismissContinuation`, the two
  internal CLI verbs, the switcher's retry/dismiss items and "continuation failed" label, the "copy orientation
  prompt" fallback, and `ContinuationRefuses`'s Failed branch.
- **Migration (FUNERAL):** a stored Failed request decodes into `ContinuationMissed`, so an existing blocked slot
  unblocks on the first Couch start with the new build.

### 5. Retire the paste (M2, the retired-surface sweep)

- **Removed:** `wrapcmd/orientation.go` delivery, `PAIR_ORIENTATION_REQUEST`, `readiness.ReadyRecord.Orientation`,
  `OSOrientationStatusReader`, the `orientation` delivery model, and the peer-composer and automatic-input waits on
  orientation.
- **Kept:** the composer recognizers, which peer delivery uses.
- **Swept:** README, atlas (`couch.md`'s continuation section and the ops table) and comments; env-gated live tests
  are included.

### Standalone Pair (no Couch)

- Unchanged: it seeds the draft and never pastes. Out of scope.

## ARCH notes

- **PURE:** `PromptArgs` (per-agent argv placement), the prompt text, and the Failed → Missed decode are pure and
  table-tested.
- **ORDER:**
  - The request is Pending → Running → Complete, or cleared into Missed.
  - Missed is written in the same record update that clears the request, so no window shows both or neither.
  - Couch dies mid-Running:
    - on restart, a Running request whose target registered completes;
    - otherwise it becomes Missed, and nothing blocks.
- **SECURE:**
  - The prompt is Couch-built from a Couch-owned path and passed as one argv element.
  - `Prompt` is admitted only on fresh profiles, beside FreshNonce, and is never persisted.
- **MOCK:**
  - The existing lifecycle doubles carry Prompt.
  - A real-helper acceptance test checks that the encoded command execs `argv + prompt` and that a wrapper fresh
    start drops it.
  - A live check on a Claude and a Codex slot is done after TL/operator approval.
- **FUNERAL:** Missed is one small struct, overwritten or cleared, never accumulating. The private continuation
  copy keeps its existing lifetime.
- **CONSTRAINTS:** no new IO on any hot path. One record write fewer per compaction (no Submitted step).

## Plan

- [ ] M1 — Launch prompt, coordinates, one non-blocking line:
  - `PromptArgs` and prompt text (pure, tests per agent);
  - `AgentCommand.Prompt` / `TrustedLaunchProfile.Prompt` end to end, with no-replay tests (ledger, config,
    LatestLaunchProfile, ExtraArgs, wrapper fresh start, relaunch);
  - `RecordPreviousSession` on compaction, and `--peek --previous` after it;
  - completion at Registered;
  - `ContinuationMissed`, the Failed → Missed decode, the row line and notice;
  - guards no longer refuse on a failure;
  - a forced launch failure shows the line and blocks nothing.
- [ ] M2 — Retire the paste and the retry/dismiss surface (code, ops table, switcher, CLI, README, atlas, tests);
  sweep the references.
- [ ] Live, after the operator's OK through the TL: a Claude and a Codex slot compact and pick up their
  continuation with no paste; a later relaunch shows no replayed prompt.
