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
    - "n": 3
      timestamp: "2026-10-13T20:34:01-07:00"
      dispatched: "2026-10-13T20:25:44-07:00"
      agent: claude
      findings:
        - id: BR-9
          severity: Important
          title: 'M2 sweep incomplete: comments, test names and orphaned fixtures still reference the retired orientation paste'
          detail: 'This is the 2nd finding in family stale-retired-references; fix the rule, not the instances. Instances: nvim/draft_send.lua:12 (cites deleted wrapcmd/orientation.go), couchcmd/wait_notice.go:55 ("a stuck orientation"), launcher/args.go:10 ("optional orientation delivery"), couchcore/switchcontext_test.go:84 and :256 test names after removed APIs, and orphaned wrapcmd/testdata/orientation/agy/1.1.25/* (no consumers) plus codex fixtures still under testdata/orientation used only by peer tests. Rule: a retirement sweep starts from a written list of every retired identifier, path, env var, op name and fixture dir, and greps it across all file types (Go, Lua, Markdown, test names, testdata). Record the list in the plan Revisions and clear every hit in one pass.'
          family: stale-retired-references
          round: 3
        - id: BR-10
          severity: Minor
          title: ReadyRecord.DeprecatedOrientation names no removal condition and never-written is not enforced
          detail: 'ARCH-FUNERAL: say when the decode-only field can be deleted (every slot restarted on a #522+ build); have Encode or Validate reject a non-empty value so "never written" holds structurally.'
          family: tombstone-end-unnamed
          round: 3
        - id: BR-11
          severity: Minor
          title: No test pins that a launch-prompted fresh session does not report Settled before its turn opens
          detail: 'ARCH-ORDER: wrapperSettled dropped the explicit orienting gate (peer_settle.go:20) and now depends on the agent opening its turn from the launch prompt within the settle window. The atlas states this, but no test pins it. This is the 2nd finding in family plan-claimed-test-missing; the class is a removed explicit gate whose replacement invariant is only stated in prose.'
          family: plan-claimed-test-missing
          round: 3
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 4
      timestamp: "2026-10-13T20:47:14-07:00"
      dispatched: "2026-10-13T20:40:00-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: Written list recorded in plan Revisions; grep of every listed name is clean outside deliberate history; draft_send.lua:12, wait_notice.go:55, args.go:10 and test names fixed; agy fixtures and .gitattributes line removed.
          round: 4
        - id: BR-10
          disposition: addressed
          note: record.go:13-18 names the removal condition; Encode refuses non-empty value (record.go:27) and orientation_test.go asserts Encode(decoded legacy) errors, which fails if the guard is removed.
          round: 4
        - id: BR-11
          disposition: withdrawn
          note: 'Rebuttal accepted: Settled reduces to turn/picker/composer inputs, which TestWrapperSettledRule pins exhaustively; agent start-up timing is external and covered by the live check, and a first-turn gate would wedge a slot whose prompt is swallowed.'
          round: 4
      findings:
        - id: BR-12
          severity: Minor
          title: 'Symbols orphaned by the retirement survive: terminalReplies.operatorData, orientation.BuildPrompt/validateBody/MaxBodyBytes/ReaderIntent/TargetAgent'
          detail: '3rd finding in this family. terminal_replies.go:96 has zero callers; orientation/model.go BuildPrompt is test-only. Rule: a retirement list enumerates what was deleted, not what the deletion orphaned, so close every retirement with a mechanical unused-symbol pass (staticcheck U1000 or deadcode -test=false) over the touched packages and delete or justify every symbol left with test-only references.'
          family: stale-retired-references
          round: 4
        - id: BR-13
          severity: Minor
          title: Codex 0.154.0 peer captures moved into testdata/tty, whose contract requires metadata.json, so the conformance replay skips them
          detail: testdata/tty/codex/0.154.0 has no metadata.json, so TestHarnessTTYFixtureConformance skips it, yet it looks like Codex's newest conformance version. Peer-only README captures already live in testdata/peer/<agent>/<version>; move them there and extend the .gitattributes binary rule to that tree.
          family: fixture-home-convention
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 5a9eddbf10942a2e07da7f94b333e81e98cfaf16
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

## Round 3 — 2026-10-13T20:34:01-07:00 (claude) — BLOCKED

### Raised

- **BR-9** [Important] `stale-retired-references` M2 sweep incomplete: comments, test names and orphaned fixtures still reference the retired orientation paste
  This is the 2nd finding in family stale-retired-references; fix the rule, not the instances. Instances: nvim/draft_send.lua:12 (cites deleted wrapcmd/orientation.go), couchcmd/wait_notice.go:55 ("a stuck orientation"), launcher/args.go:10 ("optional orientation delivery"), couchcore/switchcontext_test.go:84 and :256 test names after removed APIs, and orphaned wrapcmd/testdata/orientation/agy/1.1.25/* (no consumers) plus codex fixtures still under testdata/orientation used only by peer tests. Rule: a retirement sweep starts from a written list of every retired identifier, path, env var, op name and fixture dir, and greps it across all file types (Go, Lua, Markdown, test names, testdata). Record the list in the plan Revisions and clear every hit in one pass.
- **BR-10** [Minor] `tombstone-end-unnamed` ReadyRecord.DeprecatedOrientation names no removal condition and never-written is not enforced
  ARCH-FUNERAL: say when the decode-only field can be deleted (every slot restarted on a #522+ build); have Encode or Validate reject a non-empty value so "never written" holds structurally.
- **BR-11** [Minor] `plan-claimed-test-missing` No test pins that a launch-prompted fresh session does not report Settled before its turn opens
  ARCH-ORDER: wrapperSettled dropped the explicit orienting gate (peer_settle.go:20) and now depends on the agent opening its turn from the launch prompt within the settle window. The atlas states this, but no test pins it. This is the 2nd finding in family plan-claimed-test-missing; the class is a removed explicit gate whose replacement invariant is only stated in prose.

## Round 4 — 2026-10-13T20:47:14-07:00 (claude) — passed

### Disposed

- BR-9 — addressed — Written list recorded in plan Revisions; grep of every listed name is clean outside deliberate history; draft_send.lua:12, wait_notice.go:55, args.go:10 and test names fixed; agy fixtures and .gitattributes line removed.
- BR-10 — addressed — record.go:13-18 names the removal condition; Encode refuses non-empty value (record.go:27) and orientation_test.go asserts Encode(decoded legacy) errors, which fails if the guard is removed.
- BR-11 — withdrawn — Rebuttal accepted: Settled reduces to turn/picker/composer inputs, which TestWrapperSettledRule pins exhaustively; agent start-up timing is external and covered by the live check, and a first-turn gate would wedge a slot whose prompt is swallowed.

### Raised

- **BR-12** [Minor] `stale-retired-references` Symbols orphaned by the retirement survive: terminalReplies.operatorData, orientation.BuildPrompt/validateBody/MaxBodyBytes/ReaderIntent/TargetAgent
  3rd finding in this family. terminal_replies.go:96 has zero callers; orientation/model.go BuildPrompt is test-only. Rule: a retirement list enumerates what was deleted, not what the deletion orphaned, so close every retirement with a mechanical unused-symbol pass (staticcheck U1000 or deadcode -test=false) over the touched packages and delete or justify every symbol left with test-only references.
- **BR-13** [Minor] `fixture-home-convention` Codex 0.154.0 peer captures moved into testdata/tty, whose contract requires metadata.json, so the conformance replay skips them
  testdata/tty/codex/0.154.0 has no metadata.json, so TestHarnessTTYFixtureConformance skips it, yet it looks like Codex's newest conformance version. Peer-only README captures already live in testdata/peer/<agent>/<version>; move them there and extend the .gitattributes binary rule to that tree.

## Open findings

- **BR-8** [Minor] `fix-unpinned-by-test` BR-1's keep-the-record change in executeContinuation has no test that fails without it
- **BR-12** [Minor] `stale-retired-references` Symbols orphaned by the retirement survive: terminalReplies.operatorData, orientation.BuildPrompt/validateBody/MaxBodyBytes/ReaderIntent/TargetAgent
- **BR-13** [Minor] `fixture-home-convention` Codex 0.154.0 peer captures moved into testdata/tty, whose contract requires metadata.json, so the conformance replay skips them
