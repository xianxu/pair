# Boundary Review — pair#522 (milestone M1)

| field | value |
|-------|-------|
| issue | 522 — couch: compaction without composer paste: record coordinates, start the new conversation with a launch prompt, no blocking 'continuation failed' state |
| repo | pair |
| issue file | workshop/issues/000522-couch-compaction-without-composer-paste-record-coordinates-start-the-new-conversation-with-a-launch-prompt-no-blocking-continuation-failed-state.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 51ab81e9815ea65e819577cc92d504d2d524f1f5..ca52528bd682dc8c9f8d8427fd34f9d54aa20e46 |
| command | sdlc milestone-close --issue 522 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-13T19:40:27-07:00 |
| verdict | REWORK |

## Review

Reviewing the #522 M1 diff. I've read the plan and the checkpoint, launcher and couchcore changes; next I'm checking the restart path, the switcher UI, the tests, and running the test suite.

Found a likely nil-pointer panic on the coordinates path. Now checking the switcher/console code and running the tests.

Generating the runtime assets that this checkout lacks (gitignored), so I can run the tests that need them.

Tests all pass. Now checking coverage of the new paths: previous-session on compaction, the no-replay checks, and the Failed→Missed migration.

Coordinates panic confirmed by reading the code; the wrapper's fresh start safely drops the prompt. Now sweeping the tree for leftover retired references and checking docs.

```verdict
verdict: REWORK
confidence: high
```

M1 is mostly delivered and well built. The one-time prompt travels beside the argv and is never inside it, `Registered` now completes the request, `checkpoint.Miss` replaces the blocking `Failed` phase with one non-blocking line, the retry/dismiss/status surface is gone from code, README and atlas, and a stored `Failed` request decodes into the line. All affected packages pass (`checkpoint`, `launcher`, `couchcore`, `couchtty`, `couchcmd`, `threadrecord`, `wrapcmd`, `artifactpath`). The checkout first needed the gitignored runtime assets generated via `runtimebundle-generate`; that's an environment step, not a code problem. One crash blocks SHIP. The new coordinates step in `executeContinuation` overwrites `record` with a zero `ThreadRecord` when `RecordPreviousSession` fails, then passes it to `failContinuation`. That function dereferences `record.Continuation.ID` and panics. A revision race with any other writer of the thread record is enough to trigger it. The fix is about one line plus a regression test, and the boundary should be re-run after it.

**1. Strengths**
- **The prompt is structurally one-time.** `wrap.go:2796-2828` keeps `launchPrompt` separate from `argv` and applies it only at exec (`wrap.go:2935`). `freshAgentInvocation` (`wrap.go:2639`) builds a new `AgentCommand` from argv without a Prompt, so an in-pane fresh start can't replay it.
- **Clear, layered admission.** `ValidatePrompt` (`launcher/prompt.go:17`) rejects multi-line, control-character, oversized and `-`-prefixed prompts. It is enforced both at `ValidateTrustedLaunchProfile` and in `RunLaunch`, and only on fresh profiles.
- **Atomic miss.** `ThreadStore.MissContinuation` (`continuation_store.go:62`) clears the request and writes the line in one revision-checked update, as the plan's ORDER note requires. `StartRegistered` (`starttransaction.go:720`) and `PublishContinuation` clear the line.
- **End-to-end acceptance through the real layout stanza and wrapper.** `continuation_acceptance_test.go:447-462` checks the prompt is the last argv element and appears exactly once. `continuation_test.go:89-95` checks the record, `ExtraArgs` and `LatestLaunchProfile` don't carry it.
- **PQ-1 is covered.** `TestAMissWhoseTargetMayExistNeverYieldsASecondAgent` (`continuation_guard_test.go:322`) runs a miss, then relaunch, and checks exactly one incarnation remains.

**2. Critical findings**
- **Nil-pointer panic in `executeContinuation` (`couchcore/continuation.go:357-358`).**
  - The code is `if record, err = c.Threads.RecordPreviousSession(...); err != nil { return c.failContinuation(record, ...) }`.
  - On any error, `updateExistingThread` returns `ThreadRecord{}` (`threadstore.go:389-421`). `failContinuation` then runs `r := record.Continuation; ... r.ID` (`continuation.go:201-203`) and panics.
  - Triggers include a `ThreadRevisionError`, which is realistic: a direct-store writer such as publish-description runs in another process and can bump the revision between the park and this write. A `Validate` refusal also triggers it.
  - The panic happens on the operation goroutine, so it takes Couch down, right after the source was parked and before the new conversation is launched.
  - Fix: assign to a temporary, e.g. `updated, err := ...; if err != nil { return c.failContinuation(record, ...) }; record = updated`.
  - Add a test that makes `RecordPreviousSession` fail, for example by bumping the revision in a hook, and asserts a `Missed` result with no panic.

**3. Important findings**
- **`PromptArgs` appends the positional prompt after user flags (`launcher/prompt.go:41`).**
  - Some flags take several values: Claude's `--add-dir`, `--allowedTools` and `--mcp-config`, and Codex's `-i/--image`. With saved argv such as `--add-dir ../x`, the prompt is parsed as one more directory or image instead of the first message. The new conversation then starts idle, and nothing reports it: completion is at registration, and the plan explicitly doesn't detect a swallowed prompt.
  - Fix: for positional agents, insert `--` before the prompt when the argv has none, after confirming each CLI accepts `--`. Add a table case with a multi-value flag.
- **A plan-claimed test is missing.** The plan says a test checks that "a wrapper fresh start drops it". No `wrapcmd` test feeds `freshAgentInvocation` an environment whose `PAIR_AGENT_COMMAND` carries a Prompt and asserts the new command has none. The behavior is correct by construction today. The test is cheap and guards against a future refactor that copies the incoming command.

**4. Minor findings**
- `resume_route.go:19` and `:110` still name `RetryContinuation` in comments.
- **ARCH-DRY:** the "target may exist" rule, `(SourcePark != "" || SourceAbsence != nil) && len(Incarnations) != 0`, is written twice, at `continuation.go:208-209` and `thread.go:763`. It should be one helper.
- **Downgrade compatibility:** `threadrecord.Record` gains `continuation_missed`, and records decode strictly, so an older binary rejects any record that carries a miss. The comment this diff deleted (the old `DismissFailedContinuation` doc) explicitly valued old-binary readability. Either accept that or note it in the plan.
- `Miss.Path` is the repository checkpoint's `SourcePath`, while the prompt and `PreviousSession.Continuation` use Couch's private copy. The line and the prompt therefore point at different files. Pick one deliberately.
- If `MissContinuation` itself fails (store error), the request stays `Running`, and the console's watcher re-issues `continue-thread` on every scan. This is bounded by `queued`, but it is a retry loop with no backoff.

**5. Test coverage notes**
- Good: per-agent `PromptArgs` table, validation of bad prompts, admission only on fresh profiles, no prompt in the ledger, config or record, Failed→Missed decode (`TestALegacyFailedRequestReadsAsItsLine`), and the line clearing on a registered start.
- Missing: failure injection on the new coordinates step (it would have caught the Critical), a multi-value flag in `PromptArgs`, and the wrapper fresh start receiving a Prompt-bearing environment.

**6. Architectural notes**
- **ARCH-DRY:** flag, Minor (the duplicated target-may-exist rule).
- **ARCH-PURE:** pass. `PromptArgs`, `ValidatePrompt`, `ContinuationPrompt`, `MissOf` and `Miss.Line` are pure and table-tested.
- **ARCH-PURPOSE:** pass for M1. The spec's purpose is met, and the M2 deferral (removing the paste machinery) is a separable cleanup.
- **ARCH-MOCK:** pass. A real wrapper runs a stand-in agent, and the live Claude/Codex check is properly gated on the operator.
- **ARCH-CONSTRAINTS:** pass. No new IO on hot paths, and one fewer record write per compaction.
- **ARCH-SECURE:** pass, with a note. The prompt is Couch-built, validated, and passed as one argv element. The prompt-placement finding and the strict-decode downgrade are correctness and compatibility issues, not trust issues.
- **ARCH-ORDER:** flag. The miss is atomic, but the error path that unwinds the new coordinates step loses the request's state and panics (the Critical). No test injects a failure at that step.
- **ARCH-FUNERAL:** pass. `Miss` is a single overwritten struct, cleared on the next registered start or publish.
- For M2, watch whether `withoutOrientation(env)` in `freshAgentInvocation` still has a reason to exist once `PAIR_ORIENTATION_REQUEST` is retired.

**7. Plan revision recommendations**
- Under Revisions: "`PromptArgs` places a positional prompt after `--`, because user flags that take several values (claude `--add-dir`, codex `--image`) would otherwise swallow it."
- State whether the miss line names the repository checkpoint or Couch's private copy, and say that `continuation_missed` makes records unreadable by pre-#522 binaries.

```findings
findings:
  - id: new
    severity: Critical
    family: error-path-clobbers-state
    title: |
      executeContinuation panics when RecordPreviousSession fails: record is overwritten with a zero record before failContinuation dereferences Continuation
    detail: |
      continuation.go:357 assigns `record, err = c.Threads.RecordPreviousSession(...)`; on error the store returns ThreadRecord{}, and failContinuation (continuation.go:201-203) dereferences record.Continuation.ID (nil). A revision race with any direct-store writer, or a Validate refusal, takes Couch down after the source was parked. Assign to a temp and keep the prior record; add a failure-injection test asserting a Missed result.
  - id: new
    severity: Important
    family: launch-prompt-argv-placement
    title: |
      PromptArgs appends the positional prompt after user flags, so a flag that takes several values swallows it
    detail: |
      launcher/prompt.go:41. With saved argv like claude `--add-dir ../x` or codex `--image f`, the trailing prompt is parsed as another flag value, and the new conversation starts idle with no signal (completion is at registration). Insert `--` before the prompt for positional agents when argv has none, after verifying each CLI accepts it; add a table case.
  - id: new
    severity: Important
    family: plan-claimed-test-missing
    title: |
      No test that the wrapper's in-pane fresh start drops a launch prompt, though the plan claims one
    detail: |
      The behavior is correct by construction (wrap.go:2639 re-encodes AgentCommand without Prompt), but no wrapcmd test gives freshAgentInvocation a Prompt-bearing PAIR_AGENT_COMMAND and asserts the next command has none.
  - id: new
    severity: Minor
    family: stale-retired-references
    title: |
      resume_route.go:19 and :110 comments still name RetryContinuation
  - id: new
    severity: Minor
    family: duplicated-rule
    title: |
      ARCH-DRY: the target-may-exist rule is written in both failContinuation and the Failed-decode migration
    detail: |
      continuation.go:208-209 and thread.go:763 both compute (SourcePark != "" || SourceAbsence != nil) && len(Incarnations) != 0; extract one helper.
  - id: new
    severity: Minor
    family: strict-decode-downgrade
    title: |
      New continuation_missed record field makes thread records unreadable by pre-#522 strict decoders
    detail: |
      The deleted DismissFailedContinuation comment explicitly kept records readable by older binaries. Accept and document this in the plan, or note it in the migration section.
  - id: new
    severity: Minor
    family: miss-path-referent
    title: |
      Miss.Path names the repository checkpoint (SourcePath), while the prompt and PreviousSession.Continuation name Couch's private copy
```

---

## Re-review — 2026-10-13T20:02:58-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 522 — couch: compaction without composer paste: record coordinates, start the new conversation with a launch prompt, no blocking 'continuation failed' state |
| repo | pair |
| issue file | workshop/issues/000522-couch-compaction-without-composer-paste-record-coordinates-start-the-new-conversation-with-a-launch-prompt-no-blocking-continuation-failed-state.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 51ab81e9815ea65e819577cc92d504d2d524f1f5..887a44bf9e28020cf63a597a9188e9d8290bd2b7 |
| command | sdlc milestone-close --issue 522 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-13T20:02:58-07:00 |
| verdict | SHIP |

## Review

Focus on the round-1 fix commit first.

Verifying the prior-round fixes now. Next I'll run the touched packages' tests and revert-check BR-1/BR-3.

Build needs a generated asset; checking how to produce it before running tests.

Tests pass (one unrelated pty flake, green on rerun). Now revert-checking the BR-1 and BR-3 fixes.

Reading each agent CLI's `--help` to check that putting the prompt first is valid.

```verdict
verdict: SHIP
confidence: high
```

All seven round-1 findings are fixed, and I checked each against the code rather than the commit message. The crash (BR-1) is gone. `failContinuation` no longer gets a zeroed record, because `executeContinuation` now keeps the prior one (`continuation.go:371-375`). As a second safeguard, `failContinuation` refuses a record that has no request and returns the cause instead of panicking (`continuation.go:202-207`); a regression test covers that guard. The launch prompt now goes first in argv. I read the live `--help` of all six installed agents, and that placement is valid for each (details under Strengths). The in-pane fresh start is now covered: I made `freshAgentInvocation` carry the prompt, and the new test failed. All touched packages pass. The one exception was `TestRealRunnersExitedPrecedesTheGroup/pty`, which failed once under full-suite load. This range doesn't touch that file, and the test passed 3 of 3 times when re-run alone. One Minor gap remains and blocks nothing: the part of the BR-1 fix that keeps the record has no test. I reverted just that hunk and the continuation suite stayed green.

1. **Strengths**
   - `targetMayExist` (`continuation.go:224-227`) is now the single rule. Both the failure path and the decode migration for old Failed records (`thread.go:196`) use it.
   - `failContinuation` re-reads the request's record through `writeRequestRecord`, and `MissContinuation` checks the request ID (`continuation_store.go:64`). So a stale record passed in by the caller can't end the wrong request.
   - Putting the prompt first is actually safer than the `--` approach BR-2 suggested. `agy --help` prints `Usage of agy:`, which is the usage format of Go's standard `flag` package, and that parser stops reading flags at the first positional argument. Placing `-i P` first is the only position that is safe whatever the saved argv holds.
   - The other agents' usage lines all allow a positional prompt: grok `[OPTIONS] [PROMPT] [COMMAND]`, codex and muse `[OPTIONS] [PROMPT]`, qodercli `[query...]`. Their parsers accept options after it.
   - `launchEnvCommand` (`wrap.go:2676`) makes the split between "argv the wrapper keeps" and "the one-time prompt" a named, testable seam.
   - Nothing stale is left: a sweep for `RetryContinuation`, `DismissContinuation` and `checkpoint.Failed` only finds the migration and tests that assert the retirement.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **Untested part of the BR-1 fix** (`continuation.go:371-375`): if `RecordPreviousSession` fails after park, the fix records a Miss, but no test pins that.
     - Proof: I put back `record, err = …` with the guard still in place, and `go test ./cmd/internal/couchcore -run 'Continu|PreviousSession|Miss'` stayed green.
     - The plan says there is no store seam to inject this failure, which is accurate: `PairLifecycle` and `Threads` are concrete types.
     - Fix: when a seam is added for other reasons, add an injection test that asserts `Missed != nil`.
     - The risk is low, because the guard rules out the crash.

5. **Test coverage**
   - The new BR-1 test pins the nil guard, not the record-keeping (see Minor above).
   - BR-2's table cases (`--add-dir a b`, codex `--image f`) fail against the old code, since the expected outputs changed.
   - The two acceptance tests now assert that the prompt is the first argument and appears exactly once.
   - Environment note: the build needs `make runtimebundle-generate` first, or `artifactpath` and `runtimebundle` tests fail. That's a setup step, not a code problem.

6. **Architecture**
   - **ARCH-DRY: pass.** BR-5 is fixed through `targetMayExist`.
   - **ARCH-PURE: pass.** `PromptArgs`, `ValidatePrompt`, `MissOf` and `ContinuationPrompt` are pure functions with tests that need no IO.
   - **ARCH-PURPOSE: pass.** Every BR-1 call site in `executeContinuation` passes a record that still has its request, and the remaining `record, err =` sites return directly. Nobody reaches `failContinuation` with a cleared record.
   - **ARCH-MOCK: pass, with a note.** There is no live conformance check for the agents' prompt grammar. I confirmed it by hand from `--help` in this review. A cheap scheduled `--help` usage check would catch an upstream CLI changing its grammar.
   - **ARCH-CONSTRAINTS: pass.** The prompt is capped at 4096 bytes and limited to one line.
   - **ARCH-SECURE: pass.**
     - The prompt is a single exec argument and can't start with `-`.
     - Old Failed records read as a Miss. Records carrying the new field can't be read by pre-#522 builds; the plan accepts this and states that downgrading isn't supported.
   - **ARCH-ORDER: pass.** Store transitions go through `MissContinuation` and `AdvanceContinuation` with revision checks.
   - **ARCH-FUNERAL: pass.** `ContinuationMissed` is replaced by the next miss and cleared by the next successful start or compaction.

7. **Plan revisions**
   - Design §1 still says `PromptArgs` places the prompt "positional after the argv, or after an existing `--`". Add a Revisions entry, or edit §1, to say the prompt goes first (`-i P` first for agy). The round-1 log entry records the change, but the Design section now contradicts the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      executeContinuation keeps the prior record (continuation.go:371-375) and failContinuation refuses a request-less record; TestFailContinuationWithoutItsRequestReturnsTheCause panics without the guard. Keep-record hunk itself unpinned (raised Minor).
  - id: BR-2
    disposition: addressed
    note: |
      Prompt now leads argv; table cases fail on old code; live --help of claude/codex/grok/qodercli/muse/agy confirms prompt-first is accepted (agy is Go-flag style, where first is the only safe spot).
  - id: BR-3
    disposition: addressed
    note: |
      Verified by mutation: making freshAgentInvocation encode the prompt turns TestAnInPaneFreshStartDropsTheLaunchPrompt red.
  - id: BR-4
    disposition: addressed
    note: |
      resume_route.go:19/:110 now name Continue; repo sweep finds only migration/retirement-asserting references.
  - id: BR-5
    disposition: addressed
    note: |
      targetMayExist (continuation.go:224) is used by failContinuation and the Failed-decode migration (thread.go:196).
  - id: BR-6
    disposition: addressed
    note: |
      Accepted and documented in the plan's round-1 log (single build, rollback unsupported).
  - id: BR-7
    disposition: addressed
    note: |
      Deliberate choice documented on MissOf (miss.go:26-30): committed checkpoint survives the next compaction, the private copy does not.
findings:
  - id: new
    severity: Minor
    family: fix-unpinned-by-test
    title: |
      BR-1's keep-the-record change in executeContinuation has no test that fails without it
    detail: |
      Reverting continuation.go:371-375 to `record, err = ...` (guard kept) leaves the couchcore continuation suite green; only the nil guard is pinned. Add an injection test asserting a Missed result when a store seam exists.
```
