---
gate: boundary-review
issue: 522
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T19:40:27-07:00"
      dispatched: "2026-10-13T19:32:38-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: 'executeContinuation panics when RecordPreviousSession fails: record is overwritten with a zero record before failContinuation dereferences Continuation'
          detail: continuation.go:357 assigns `record, err = c.Threads.RecordPreviousSession(...)`; on error the store returns ThreadRecord{}, and failContinuation (continuation.go:201-203) dereferences record.Continuation.ID (nil). A revision race with any direct-store writer, or a Validate refusal, takes Couch down after the source was parked. Assign to a temp and keep the prior record; add a failure-injection test asserting a Missed result.
          family: error-path-clobbers-state
          round: 1
        - id: BR-2
          severity: Important
          title: PromptArgs appends the positional prompt after user flags, so a flag that takes several values swallows it
          detail: launcher/prompt.go:41. With saved argv like claude `--add-dir ../x` or codex `--image f`, the trailing prompt is parsed as another flag value, and the new conversation starts idle with no signal (completion is at registration). Insert `--` before the prompt for positional agents when argv has none, after verifying each CLI accepts it; add a table case.
          family: launch-prompt-argv-placement
          round: 1
        - id: BR-3
          severity: Important
          title: No test that the wrapper's in-pane fresh start drops a launch prompt, though the plan claims one
          detail: The behavior is correct by construction (wrap.go:2639 re-encodes AgentCommand without Prompt), but no wrapcmd test gives freshAgentInvocation a Prompt-bearing PAIR_AGENT_COMMAND and asserts the next command has none.
          family: plan-claimed-test-missing
          round: 1
        - id: BR-4
          severity: Minor
          title: resume_route.go:19 and :110 comments still name RetryContinuation
          family: stale-retired-references
          round: 1
        - id: BR-5
          severity: Minor
          title: 'ARCH-DRY: the target-may-exist rule is written in both failContinuation and the Failed-decode migration'
          detail: continuation.go:208-209 and thread.go:763 both compute (SourcePark != "" || SourceAbsence != nil) && len(Incarnations) != 0; extract one helper.
          family: duplicated-rule
          round: 1
        - id: BR-6
          severity: Minor
          title: New continuation_missed record field makes thread records unreadable by pre-#522 strict decoders
          detail: The deleted DismissFailedContinuation comment explicitly kept records readable by older binaries. Accept and document this in the plan, or note it in the migration section.
          family: strict-decode-downgrade
          round: 1
        - id: BR-7
          severity: Minor
          title: Miss.Path names the repository checkpoint (SourcePath), while the prompt and PreviousSession.Continuation name Couch's private copy
          family: miss-path-referent
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T20:02:58-07:00"
      dispatched: "2026-10-13T19:47:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: executeContinuation keeps the prior record (continuation.go:371-375) and failContinuation refuses a request-less record; TestFailContinuationWithoutItsRequestReturnsTheCause panics without the guard. Keep-record hunk itself unpinned (raised Minor).
          round: 2
        - id: BR-2
          disposition: addressed
          note: Prompt now leads argv; table cases fail on old code; live --help of claude/codex/grok/qodercli/muse/agy confirms prompt-first is accepted (agy is Go-flag style, where first is the only safe spot).
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Verified by mutation: making freshAgentInvocation encode the prompt turns TestAnInPaneFreshStartDropsTheLaunchPrompt red.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: resume_route.go:19/:110 now name Continue; repo sweep finds only migration/retirement-asserting references.
          round: 2
        - id: BR-5
          disposition: addressed
          note: targetMayExist (continuation.go:224) is used by failContinuation and the Failed-decode migration (thread.go:196).
          round: 2
        - id: BR-6
          disposition: addressed
          note: Accepted and documented in the plan's round-1 log (single build, rollback unsupported).
          round: 2
        - id: BR-7
          disposition: addressed
          note: 'Deliberate choice documented on MissOf (miss.go:26-30): committed checkpoint survives the next compaction, the private copy does not.'
          round: 2
      findings:
        - id: BR-8
          severity: Minor
          title: BR-1's keep-the-record change in executeContinuation has no test that fails without it
          detail: Reverting continuation.go:371-375 to `record, err = ...` (guard kept) leaves the couchcore continuation suite green; only the nil guard is pinned. Add an injection test asserting a Missed result when a store seam exists.
          family: fix-unpinned-by-test
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 887a44bf9e28020cf63a597a9188e9d8290bd2b7
      blocked: false
---

# Gate ledger — pair#522 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T19:40:27-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `error-path-clobbers-state` executeContinuation panics when RecordPreviousSession fails: record is overwritten with a zero record before failContinuation dereferences Continuation
  continuation.go:357 assigns `record, err = c.Threads.RecordPreviousSession(...)`; on error the store returns ThreadRecord{}, and failContinuation (continuation.go:201-203) dereferences record.Continuation.ID (nil). A revision race with any direct-store writer, or a Validate refusal, takes Couch down after the source was parked. Assign to a temp and keep the prior record; add a failure-injection test asserting a Missed result.
- **BR-2** [Important] `launch-prompt-argv-placement` PromptArgs appends the positional prompt after user flags, so a flag that takes several values swallows it
  launcher/prompt.go:41. With saved argv like claude `--add-dir ../x` or codex `--image f`, the trailing prompt is parsed as another flag value, and the new conversation starts idle with no signal (completion is at registration). Insert `--` before the prompt for positional agents when argv has none, after verifying each CLI accepts it; add a table case.
- **BR-3** [Important] `plan-claimed-test-missing` No test that the wrapper's in-pane fresh start drops a launch prompt, though the plan claims one
  The behavior is correct by construction (wrap.go:2639 re-encodes AgentCommand without Prompt), but no wrapcmd test gives freshAgentInvocation a Prompt-bearing PAIR_AGENT_COMMAND and asserts the next command has none.
- **BR-4** [Minor] `stale-retired-references` resume_route.go:19 and :110 comments still name RetryContinuation
- **BR-5** [Minor] `duplicated-rule` ARCH-DRY: the target-may-exist rule is written in both failContinuation and the Failed-decode migration
  continuation.go:208-209 and thread.go:763 both compute (SourcePark != "" || SourceAbsence != nil) && len(Incarnations) != 0; extract one helper.
- **BR-6** [Minor] `strict-decode-downgrade` New continuation_missed record field makes thread records unreadable by pre-#522 strict decoders
  The deleted DismissFailedContinuation comment explicitly kept records readable by older binaries. Accept and document this in the plan, or note it in the migration section.
- **BR-7** [Minor] `miss-path-referent` Miss.Path names the repository checkpoint (SourcePath), while the prompt and PreviousSession.Continuation name Couch's private copy

## Round 2 — 2026-10-13T20:02:58-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — executeContinuation keeps the prior record (continuation.go:371-375) and failContinuation refuses a request-less record; TestFailContinuationWithoutItsRequestReturnsTheCause panics without the guard. Keep-record hunk itself unpinned (raised Minor).
- BR-2 — addressed — Prompt now leads argv; table cases fail on old code; live --help of claude/codex/grok/qodercli/muse/agy confirms prompt-first is accepted (agy is Go-flag style, where first is the only safe spot).
- BR-3 — addressed — Verified by mutation: making freshAgentInvocation encode the prompt turns TestAnInPaneFreshStartDropsTheLaunchPrompt red.
- BR-4 — addressed — resume_route.go:19/:110 now name Continue; repo sweep finds only migration/retirement-asserting references.
- BR-5 — addressed — targetMayExist (continuation.go:224) is used by failContinuation and the Failed-decode migration (thread.go:196).
- BR-6 — addressed — Accepted and documented in the plan's round-1 log (single build, rollback unsupported).
- BR-7 — addressed — Deliberate choice documented on MissOf (miss.go:26-30): committed checkpoint survives the next compaction, the private copy does not.

### Raised

- **BR-8** [Minor] `fix-unpinned-by-test` BR-1's keep-the-record change in executeContinuation has no test that fails without it
  Reverting continuation.go:371-375 to `record, err = ...` (guard kept) leaves the couchcore continuation suite green; only the nil guard is pinned. Add an injection test asserting a Missed result when a store seam exists.

## Open findings

- **BR-8** [Minor] `fix-unpinned-by-test` BR-1's keep-the-record change in executeContinuation has no test that fails without it
