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

## Revisions

### 2026-10-13: plan-quality round 1

- **PQ-1, a target that may still be alive.** The duplicate-agent guard is the thread's incarnation invariant,
  not the continuation phase. Two existing pieces already enforce it:
  - **Starts:** every start goes through `CommitStartClaim`, which refuses while any incarnation remains
    (`threadstore.go:606`).
  - **Relaunch:** it parks the one exact incarnation before it resumes (`relaunch.go:117-121`: no incarnation →
    "not running; Enter resumes it"; otherwise `soleParkableIncarnation`, then park).

  So the Missed line has two kinds, and each takes the switcher's existing action for its row state:
  - **No target:** the failure came before launch, or `rollbackTrackedStart`/`AbortStarted` removed the target's
    incarnation. The line reads "…; continuation at <path>", and the row's action is **resume** (Enter), which
    brings back the parked old conversation. Old conversation running (a pre-park failure): the row is live, and
    the action is compact again.
  - **Target may exist:** the target's incarnation is still recorded (an `AbortStarted` cleanup error, or the
    unresolved start at `continuation.go:254`). The line adds "the new conversation may be running", and the action
    is **relaunch**, which parks that exact target first. Any other start is refused by the incarnation guard, not
    by the continuation phase.

  Tests:
  - an `AbortStarted` cleanup failure, then relaunch, never gives two incarnations or two agents;
  - a post-park launch failure, then resume, gives back the old conversation (`VerifiedPark` survives
    `rollbackTrackedStart`); this replaces what `RetryAbsent` did.
- **Non-goals, stated:**
  - Completion at registration doesn't prove the agent consumed the prompt. That is deliberate: the paste's
    composer recognition is the fragile part this issue removes. A swallowed prompt (for example under Codex's
    trust screen) shows as an idle new conversation whose previous-session coordinates and continuation are
    recorded. The live check includes a trust-screen case.
  - Resume with a prompt (offered in the Spec) is not built: compaction always starts a fresh conversation, so no
    caller needs it (YAGNI).
- **Pure transition changes (`checkpoint/request.go`):**
  - `Advance` at `Registered` completes the request (Running → Complete); `Submitted` is removed.
  - `Fail`, `RetryAbsent` and `RetryObserve` are removed; `Failed` is kept only as a decode-time phase.
  - `CheckDismissible` goes.
  - The store gains `ThreadStore.MissContinuation(address, requestID, attempt, miss)`, which clears
    `Continuation` and writes `ContinuationMissed` in one revision-checked update. The decode migration maps a
    stored `Failed` the same way.

### 2026-10-13: the milestone boundary moves (during M1)

Completion at registration makes `checkpoint.Failed` unreachable. Keeping
retry/dismiss alive until M2 would have meant keeping dead events in the pure
machine just to serve them. So the boundary follows the layers instead:

- **M1:** the request's whole lifecycle and its operator surface.
  - The prompt, the coordinates, completion at registration, and the line.
  - Retire `Failed` as an event target, along with `RetryContinuation`,
    `DismissContinuation`, `ReconcileContinuation`, the three operations, the
    switcher's retry/dismiss items, and the copy-orientation item.
  - Rework the tests and docs that described them.
- **M2:** the composer-paste machinery that no longer has a producer.
  - The wrapper's orientation delivery, `PAIR_ORIENTATION_REQUEST`,
    `readiness.ReadyRecord.Orientation`, `OSOrientationStatusReader.Read`, the
    `orientation-status` operation and the `orientation` delivery model.
  - The composer recognizers stay, for peer delivery.
- **One test was weakened.** `TestSingletonCommandHeldHostLeaseRefusesBeforeStoreConstruction`
  used the internal `retry-continuation` as its lease-taking command. Every
  owner command is now public and adoption-gated first, so on its unmigrated
  fixture it accepts either refusal; the untouched-authority snapshot is
  unchanged. `TestHeldSupervisorRefusesBeforeStartingActor` keeps the lease
  refusal itself.

### 2026-10-13: M1 review round 1 (REWORK)

- **BR-1 (Critical):** an error from `RecordPreviousSession` replaced the
  record with a zero one before `failContinuation` dereferenced it.
  - The class is error paths that clobber the caller's state.
  - `executeContinuation` now keeps the prior record, and `failContinuation`
    refuses a record without its request (cause returned, no panic).
  - No store seam fails that one write inside a live compaction, so the pin is
    the direct test plus the code fix.
- **BR-2:** a trailing positional prompt is swallowed by a saved flag that takes
  several values. `PromptArgs` now puts the prompt FIRST (agy: `-i P` first),
  so nothing precedes it. It can't be read as a flag (`ValidatePrompt` refuses
  a leading `-`). The table cases include `--add-dir a b` and codex
  `--disable … --image f`.
- **BR-3:** `launchEnvCommand` splits the decoded command. A test feeds a
  prompt-bearing command through it and `freshAgentInvocation`, and asserts
  that the fresh start carries no prompt.
- **Minor:**
  - **DRY:** `targetMayExist` is the one rule (failure and decode).
  - **Stale comments:** fixed.
  - **`Miss.Path`:** it is the committed checkpoint on purpose (the private copy
    is replaced by the next compaction); the reason is on `MissOf`.
  - **Strict decode:** a pre-#522 binary can't read a record carrying
    `continuation_missed`. That is accepted, as for #517's `previous_session`.
    Couch and Pair ship as one build (`make build`), and rolling back to an
    older build is not a supported path while threads hold the line.

### 2026-10-13: M2 review round 3 (FIX-THEN-SHIP, BR-9)

- **The retirement sweep starts from a written list.** These identifiers, paths,
  env vars, operation names and fixture directories were grepped across every
  tracked file type (Go, Lua, Markdown, test names, testdata, `.gitattributes`;
  `workshop/history` and dated continuations excluded):
  - **Wrapper and its delivery:** `wrapcmd/orientation.go`, `orientationDelivery`,
    `PAIR_ORIENTATION_REQUEST`, `orientation.Env`, `DeliveryState`/`DeliveryPhase`/`AdvanceDelivery`.
  - **Operations and Couch's reader:** `orientation-status`, `OrientationStatus`,
    `ReadOrientationStatus`, `OSOrientationStatusReader`, `retry-continuation`,
    `dismiss-continuation`, `continuation-status`, `RetryContinuation`,
    `DismissContinuation`, `DismissFailedContinuation`, `ReconcileContinuation`.
  - **Switcher:** `copy-orientation`/`copyOrientation`/"Copy orientation",
    `setOrientationLocked`.
  - **Composer gate:** `orientationPromptCol`, `orientationRuleCellTolerant`,
    `orientationPromptOK`, `promptGlyphAuthorities`, `orientationComposerActive`,
    `codexOrientationStartupStatus`, `orientationRowText`.
  - **Renamed:** `orientationReplies`/`orientationEscape` (now `terminalReplies`/`terminalEscape`).
  - **Other:** "continuation failed", `ContinuationRefuses`,
    `RetireProvedDeadIncarnations`, `checkpoint.Exits`, `CheckDismissible`,
    "stuck orientation", "orientation delivery", "orientation auto-submit",
    `automaticInputOrientation`, `orientationPending`, `testdata/orientation`.

  Every hit was cleared in one pass. The Codex 0.154.0 captures peer tests still
  read moved to `testdata/tty/codex/0.154.0`, and the orphaned agy captures and
  their `.gitattributes` line went. Test names now follow their APIs
  (`TestReadyReader…`, `TestTheContinuationGuardsReachMatchesEveryRowAction`).
  Mentions that remain are deliberate history: the atlas line naming what #522
  retired, the #280 symptom comment, the legacy-decode test and tombstone, and
  the retired-name assertions.
- **The tombstone's end:** `ReadyRecord.DeprecatedOrientation` may be deleted
  once every slot has been relaunched on a #522+ build. Encode refuses a
  non-empty value, so never-written is structural.
- **Settled without the orienting gate (Minor, disposed):** a launch-prompted
  session is unsettled for the same reasons as any other: Settled needs
  `SettleInterval` (3 s) of byte-quiet, no open turn, no overlay, and an empty
  composer. An agent consuming its first message is drawing output or holding a
  turn. If it never takes the prompt (swallowed by a trust screen), it is
  idle, and Settled is the truth. A gate that waits for a first turn would hold
  such a slot unsettled forever and block peer delivery to it. No test can pin
  an agent's start-up timing; the live check exercises it.
