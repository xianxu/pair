# Boundary Review — pair#522 (whole-issue close)

| field | value |
|-------|-------|
| issue | 522 — couch: compaction without composer paste: record coordinates, start the new conversation with a launch prompt, no blocking 'continuation failed' state |
| repo | pair |
| issue file | workshop/issues/000522-couch-compaction-without-composer-paste-record-coordinates-start-the-new-conversation-with-a-launch-prompt-no-blocking-continuation-failed-state.md |
| boundary | whole-issue close |
| milestone | — |
| window | 386e0b48e13a57bf37f4a0584291015aab630c75..290eb67d30924bd6ade0d66927ac1d22b252baec |
| command | sdlc close --issue 522 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-13T22:41:22-07:00 |
| verdict | SHIP |

## Review

Checking the open findings: BR-12 and BR-13 look fixed; now checking whether BR-8 got a test.

Generating the runtime bundle assets so I can run the wrapcmd/couchtty/couchcmd tests.

```verdict
verdict: SHIP
confidence: high
```

Two of the three open findings are now fixed, and the third is a Minor test gap that doesn't block the close. BR-12 is fixed: `terminalReplies.operatorData`, `orientation.BuildPrompt`, `validateBody`, `literal`, `MaxBodyBytes`, `ReaderIntent` and `TargetAgent` are deleted, and a grep of `cmd/` finds no references left. The plan's Revisions section records a `deadcode` pass and says why the other unreachable functions were left in place. BR-13 is fixed: the Codex 0.154.0 captures now sit in `testdata/peer/codex/0.154.0`, the peer tests point at the new paths, and `.gitattributes` marks `testdata/peer/**/*.raw` as binary. BR-8 is still open. `continuation.go:369-373` correctly keeps the prior record in `recorded` before calling `failContinuation`, but no commit since round 2 adds a test that injects a `RecordPreviousSession` failure. Nothing else turned up that blocks the close.

Packages I ran tests on: checkpoint, readiness, threadrecord, couchcore, launcher, wrapcmd, couchtty and couchcmd. All pass. Two notes on that:
- The first run failed because `runtimebundle/assets/runtime` is missing from the checkout. I generated it with `go run ./cmd/internal/runtimebundle/generatecmd` (the same step as Makefile.local:250) and re-ran.
- `TestParkCoordinatorCoalescesStartupRecoveryAndInteractiveRetry` failed once and then passed 5 times in a row (`-count=5`). Its file isn't in this diff, so it looks like an existing flaky test.

1. **Strengths**
   - `launcher/prompt.go:33-42`: the launch prompt goes before the saved argv. Because `ValidatePrompt` refuses a leading `-`, it can't be read as a flag or swallowed by a flag that takes several values.
   - `checkpoint/miss.go`: a compaction that doesn't finish leaves one record with a `Validate()` check and a pure `Line()`. The comment on `MissOf` explains why it points at the committed checkpoint.
   - The menu path cleans the miss line before showing it (`menu_reattach.go:411`, `rowtext.Sanitize`).
   - The `orientation` package now holds only what `--peek --previous` uses (`switchcontext.go:44`).
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** BR-8 is still open (see below).
5. **Test coverage notes:** BR-8 is the only gap. A fake `Threads` store whose `RecordPreviousSession` returns an error should make `executeContinuation` return a `Missed` result for the original request. That test fails if `record, err = ...` is put back.
6. **Architecture:**
   - ARCH-DRY pass (`targetMayExist` is shared).
   - ARCH-PURE pass (`Miss` and `PromptArgs` are pure).
   - ARCH-PURPOSE pass (the composer paste is fully retired and every consumer moved to the launch prompt).
   - ARCH-MOCK pass (no new external dependency; the live check is recorded in 290eb67d3).
   - ARCH-CONSTRAINTS pass (prompt capped at 4096 bytes, reason capped by `MaxFailureBytes`).
   - ARCH-SECURE pass (the record's `Validate` is called on decode; the menu cleans the line).
   - ARCH-ORDER pass (the miss is set by the continuation store and cleared by the start transaction at `starttransaction.go:110`).
   - ARCH-FUNERAL pass (the next start or compaction clears the miss; the tombstone's end is named, per BR-10).
7. **Plan revisions:** none needed beyond the round-4 Revisions entry already present.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      Still no regression test. No commit after round 2 adds a RecordPreviousSession failure-injection test; continuation.go:369-373 keeps the record correctly but nothing pins it. Minor, does not block.
  - id: BR-12
    disposition: addressed
    note: |
      e9611d97d deletes operatorData (terminal_replies.go), BuildPrompt/validateBody/literal/MaxBodyBytes and the ReaderIntent/TargetAgent fields (orientation/model.go), plus the test that used them. Grep of cmd/ finds no leftovers. The plan's Revisions section records the deadcode pass and why pre-#522 unreachable code was left.
  - id: BR-13
    disposition: addressed
    note: |
      Captures renamed to testdata/peer/codex/0.154.0, peer_composer_test.go paths updated, .gitattributes covers testdata/peer/**/*.raw. wrapcmd suite green.
```
