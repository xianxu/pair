# Boundary Review — pair#451 (whole-issue close)

| field | value |
|-------|-------|
| issue | 451 — Couch tells the hub when a slot is stuck waiting for someone |
| repo | pair |
| issue file | workshop/issues/000451-couch-tells-the-hub-when-a-slot-is-stuck-waiting-for-someone.md |
| boundary | whole-issue close |
| milestone | — |
| window | 536ad9671d39bfb86934fdabd4eae78856ff34da..c451a9f5368d709027ece9a55a985d0bdc5ecc52 |
| command | sdlc close --issue 451 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T13:42:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Tests need generated runtime assets; checking the Makefile to build them, then inspecting `Broker.Send`.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The design holds up and the code works. The watcher runs on the supervisor side and reads facts Couch already keeps in memory. It only reads a slot's screen tail once that slot has been quiet past the threshold. The notice goes through the ordinary `Broker.Send`, so the hub's usual message limits and ledger apply to it. The pure classifier is tested against tails shaped like live `couch --peek` captures. I generated the runtime bundle in the review checkout and ran the new tests three times (`-count=3`); all pass. `Broker.Send` only puts a message in the inbox once admission has succeeded, so a failed send can be retried next tick without sending a duplicate.

Nothing is Critical, but four Important items should be fixed before the issue closes:
1. **Wrong dialog title:** `dialogTitle` takes the first `1.` line from the top of the tail, so a numbered list in the agent's earlier output gets reported as the dialog's title. I confirmed this with a scratch test.
2. **No retry test:** the failed-send retry path has no test.
3. **Live check still open:** Done-when 3 (the live check) is not done and needs the operator.
4. **Plan out of date:** the plan still promises a diagnostic log line for an invalid threshold, and the code doesn't write one.

**Strengths**
- The code is split cleanly: pure `classifyWait`/`needsTail`/`lastProseLine` (`wait_notice.go:37-139`), and `needsTail` keeps the tail read off the path for slots that aren't quiet yet.
- The episode is marked only after the broker accepts the notice (`wait_notice.go:314-317`). The not-waiting cache is keyed on the full set of facts, so an idle slot on its resting branch costs one tail read per episode.
- The integration tests use real sockets, the real broker and wrappers that report Settled. The hub fixture itself sits on a dialog, so every test also checks that the hub is skipped (`wait_notice_test.go:166-199`).
- The episode map is in memory, pruned to the actor listing every tick, and the fact that a Couch restart re-announces a waiting slot is written down. ARCH-FUNERAL is handled properly.
- The atlas section and the TL-facing skill text are accurate, including that Couch writes the notice under the slot's identity and the slot's agent never sees it.

**Critical findings**
None.

**Important findings**
1. **Dialog title picked from the top of the tail, not the bottom** (`wait_notice.go:87-99`). The loop returns at the first `optionLine` match from the top, and `optionLine` accepts plain indented `1. ` lines. A tail of `Plan:` / `1. run tests` / `2. push` followed by a permission dialog returns `"Plan:"` instead of `"Do you want to proceed?"` (confirmed with a scratch test). Agents print numbered lists often, so this breaks Done-when 1's "naming the dialog". **Fix:** scan from the bottom (the last `1.` option line wins) and add a table case with an earlier numbered list.
2. **The failed-send retry path is untested** (`wait_notice.go:314-317`). Nothing checks that a refused send leaves the episode unmarked and that the next tick sends it. If someone moved the `w.seen[b] = …notified` line above `Send`, all tests would stay green and a send that failed once would never be retried. **Fix:** make the hub refuse one send (busy/unavailable), tick, then tick again and assert exactly one notice.
3. **Done-when 3 is unmet at issue close.** The live check is still unchecked ("needs the operator to restart Couch"). Either record the operator's confirmation in the issue log before closing, or explicitly move it to a follow-up and record that in the issue.
4. **The plan and the code disagree on invalid `COUCH_WAIT_NOTICE_AFTER`.** The plan's Threshold rule says "invalid → default with the error in Couch's diagnostic log line". The code (`wait_notice.go:183-188`) falls back silently, and the issue log admits it. Either add a `## Revisions` entry or wire in a log line. Today a typo like `COUCH_WAIT_NOTICE_AFTER=5` quietly becomes 10m.

**Minor findings**
- ARCH-DRY: the "is this the hub's worktree" check (`WorktreeRoot == authority.hub`) now lives in two places, `message_service.go:562` and `wait_notice.go:246`. Extract one predicate.
- ARCH-SECURE: `os.ReadFile(w.optOut)` ignores every error (`wait_notice.go:286`). A permission error on the mute list is treated as "no mutes", with no signal; only a missing file should be silent.
- `hubBinding` iterates over a map. If two bindings ever share the hub's worktree, which one is the hub is nondeterministic.
- The tail-markup parser (`stripTailMarkup`/`dimOnly`) lives in `couchcmd`, while the grammar is defined in `couchmessage/tail.go`. Consider moving it next to the grammar before a second reader appears.
- Each tick runs serially, and each slot can block for up to 2s on the tail read plus 2s on the send. That's fine at the 1-minute tick and current slot counts; the plan doesn't name a bound.

**Test coverage notes**
- Covered: dialog → exactly one notice per episode, and a new episode re-notifies; a question notifies; a normal end on the resting branch doesn't; idle off the resting branch notifies; a muted slot and no hub are silent; the hub-skip check is implicit in every test; summary truncation respects UTF-8 boundaries; threshold parsing.
- Missing:
  - the retry path (Important 2);
  - a test that the not-waiting cache prevents repeated tail reads;
  - a tail-read error path test;
  - an earlier numbered list above a dialog (Important 1);
  - a slot with a nil `Settled` in the integration rig (the classifier unit test covers it).

**Architecture**
- **ARCH-DRY:** flagged, Minor (duplicate hub check).
- **ARCH-PURE:** pass. The classifier, body and opt-out parser are pure and tested without IO; the IO stays in `tick`.
- **ARCH-PURPOSE:** pass on code. Every rule in the spec is implemented; the remaining gap is the live Done-when (Important 3).
- **ARCH-MOCK:** pass. It reuses the `serviceRig` fake endpoints behind the real broker seam; the live conformance check is the pending operator step.
- **ARCH-CONSTRAINTS:** pass, with a note. 1-minute tick, 2s timeouts, 60-line tail, 200-byte summary; per-tick serial latency isn't bounded explicitly.
- **ARCH-SECURE:** flagged, Minor (mute-file errors are swallowed). Tail text is rendered only into the message body, and `ValidateBody` checks it.
- **ARCH-ORDER:** pass. Episode state is a small, explicit struct whose two flags don't overlap; admission is confirmed before marking, so a failure leaves nothing marked. The tests use a single interleaving because `tick` is driven synchronously, which is fine here.
- **ARCH-FUNERAL:** pass. The map is pruned each tick, notices reuse the existing receipt and ledger bounds, and the mute file belongs to the operator.

**Plan revision recommendations**
- `## Revisions`: "Invalid `COUCH_WAIT_NOTICE_AFTER` falls back to 10m silently (no supervisor log seam on this path); the Threshold rule's 'error in Couch's diagnostic log line' is withdrawn", unless a log line is added instead.
- `## Revisions`: "`dialogTitle` takes the last `1.` option line in the tail", once Important 1 is fixed.

```findings
findings:
  - id: new
    severity: Important
    family: tail-reader-scan-direction
    title: |
      dialogTitle returns the line above the FIRST "1." in the tail, so an earlier numbered list in prose is reported as the dialog's title
    detail: |
      wait_notice.go:87-99 scans from the top; tail ["Plan:","1. run tests","2. push","", <permission dialog>] yields "Plan:" (confirmed with a scratch test). Scan from the bottom and add a table case.
  - id: new
    severity: Important
    family: untested-retry-path
    title: |
      No test covers the failed-admission retry path (episode left unmarked, next tick re-sends)
    detail: |
      Moving w.seen[b]=notified above broker.Send (wait_notice.go:314-317) keeps all tests green and silently drops notices after one refused send. Add a rig test where the hub refuses once, then accepts.
  - id: new
    severity: Important
    family: done-when-unmet-at-close
    title: |
      Done-when 3 (live check that the TL receives a notice) is still unchecked at whole-issue close
    detail: |
      Record the operator's live confirmation in the issue log, or explicitly move it to a follow-up, before closing.
  - id: new
    severity: Important
    family: plan-code-drift
    title: |
      Plan promises a diagnostic log line for an invalid COUCH_WAIT_NOTICE_AFTER; the code falls back to 10m silently
    detail: |
      wait_notice.go:183-188. Either add a plan Revisions entry withdrawing the log line, or wire one in.
  - id: new
    severity: Minor
    family: duplicated-predicate
    title: |
      The "binding is the hub" check (WorktreeRoot == authority.hub) is duplicated in message_service.go:562 and wait_notice.go:246
  - id: new
    severity: Minor
    family: swallowed-io-error
    title: |
      Errors reading the wait-notice-off mute list (other than not-exist) are silently treated as no mutes
    detail: |
      wait_notice.go:286 discards the os.ReadFile error; a permission error un-mutes every slot without any signal.
  - id: new
    severity: Minor
    family: nondeterministic-selection
    title: |
      hubBinding picks an arbitrary binding if more than one shares the hub worktree (map iteration order)
```

---

## Re-review — 2026-10-11T13:49:30-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 451 — Couch tells the hub when a slot is stuck waiting for someone |
| repo | pair |
| issue file | workshop/issues/000451-couch-tells-the-hub-when-a-slot-is-stuck-waiting-for-someone.md |
| boundary | whole-issue close |
| milestone | — |
| window | 536ad9671d39bfb86934fdabd4eae78856ff34da..424b80b504e7e29e2dfd6a514175b25b6efd0816 |
| command | sdlc close --issue 451 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T13:49:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Needs the generated runtime bundle; check how the Makefile produces it.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All seven findings from round 1 are fixed. I checked them against the code at the pinned head, and I reverted the two behavior fixes to confirm their tests catch the old bugs. Marking the episode before `Broker.Send` makes `TestWaitNoticeRetriesAfterARefusedAdmission` fail. Scanning for the dialog title from the top makes `TestDialogTitleIgnoresNumberedProse` fail. The `Wait*` tests and the classifier tests pass. The artifact inventory test (`artifactpath`) passes. One new finding stops a clean SHIP: the live check was moved to #452 in Revisions and in the Plan checklist, but the third `## Done when` bullet still lists it as a condition for closing this issue. Fixing it means editing one line.

Done-when: stale — bullet 3 ("Checked live: the TL receives a notice …") still requires this issue to deliver a live check that Revisions moved to #452. The Spec change (a bad threshold refuses the start) is covered by tests and needs no new bullet.

## 1. Strengths
- **Pure classifier with real screen shapes.** `classifyWait`, `lastProseLine` and `dialogTitle` (`cmd/internal/couchcmd/wait_notice.go:47-145`) do no IO. The table tests use tails shaped like real Claude and Codex screens, including recap lines, dim text and escaped markup.
- **Notices reuse normal delivery.** A notice is an ordinary `Broker.Send` from the waiting slot to the hub's exact slot (`wait_notice.go:338`), so it gets the hub's existing breaker, ledger and allowance. No new delivery path was added.
- **Bad threshold fails before any socket is taken.** The check is at `message_service.go:217-221`, and the error reaches `run.go:780-784`, which prints it and exits 1.
- **Integration tests use the real broker.** The `serviceRig` tests run real sockets and broker, with the hub itself sitting on a dialog. A watcher that forgot to skip the hub would therefore fail every test.
- **Opt-out file fails visibly.** If the mute list can't be read, the notice says so instead of failing silently (`wait_notice.go:307-310`), and a test covers it.

## 2. Critical findings
None.

## 3. Important findings
- **The third `## Done when` bullet still requires the live check that was deferred to #452** (issue file, `## Done when`). Revisions and the Plan checklist both say it moved to #452, which exists and has a matching Done-when. But the close gate reads `## Done when`, and that bullet is still unchecked and unqualified.
  - **This is the 2nd finding in family `done-when-unmet-at-close`.** Round 1 fixed one instance by deferring the item without editing the bullet. The rule to apply: any Done-when item deferred in `## Revisions` must be edited in the `## Done when` list itself, in the same commit — struck through, with a pointer to the issue that now owns it. Apply that rule to every bullet here, not just this one.
  - Fix: `- ~~Checked live: the TL receives a notice for a slot left waiting.~~ moved to #452 (operator-owned Couch restart).`

## 4. Minor findings
- `waitEpisode.notified` and `waitEpisode.judgedNot` are two booleans, but only one is ever set. A small tagged state would make that unrepresentable (ARCH-ORDER). Not worth raising as a finding.
- `selectedOption` also matches a Codex draft typed as `› 1. …`, so the draft line could be taken as a dialog's selected option. This only affects the summary text, not whether a notice is sent.

## 5. Test coverage notes
- **Mutation checks I ran:**
  - Marking the episode before the send fails the retry test (BR-2).
  - Scanning from the top with `optionLine` fails the dialog-title test (BR-1).
- **Threshold parsing:** `TestWaitNoticeThreshold` covers it. There is no test of the refused start itself, but the call at `run.go:780` shows the error is passed on.
- **Unrelated failure in the full package run:** `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` fails because this review checkout has no `origin` remote. It has nothing to do with this diff.
- **Timing:** the rig tests wait with fixed `time.Sleep(150ms)` pauses. Delivery order isn't controlled, but the assertions only count deliveries after the wait, so this is acceptable.

## 6. Architecture
- **ARCH-DRY: pass.** `messageAuthority.isHub` is now the single hub check, used by both the allowance code (`message_service.go:572`) and `hubBinding`.
- **ARCH-PURE: pass.** The classifier and body builder are pure; `tick` is thin glue around them.
- **ARCH-PURPOSE: pass.** Prompt, question and idle are all detected and sent. The live check is deferred properly to #452, apart from the Done-when text above.
- **ARCH-MOCK: pass.** The broker and registry are real; only the screen tail is faked, through the same `TailReader` seam production uses.
- **ARCH-CONSTRAINTS: pass.** One tick per minute, a 2 s timeout per tail read, and at most one tail read per episode thanks to the cache.
- **ARCH-SECURE: pass.** Screen text quoted in a notice is capped at 200 bytes and cut on a character boundary. A worker slot could already message the hub, so this adds no new trust. The mute-list read fails visibly.
- **ARCH-ORDER: pass, with the boolean-pair note above.** Episode identity is `LastActivity`. A failed send leaves the episode unmarked, and that is now tested.
- **ARCH-FUNERAL: pass.** The only state is an in-memory map, pruned every tick, which dies with the service. Nothing durable is written.

## 7. Plan revision recommendations
None for the plan. The issue file needs the Done-when edit described in section 3.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      dialogTitle now scans from the bottom for the selected option (wait_notice.go:92-105); TestDialogTitleIgnoresNumberedProse goes red when the scan is reverted to top-down.
  - id: BR-2
    disposition: addressed
    note: |
      TestWaitNoticeRetriesAfterARefusedAdmission goes red when w.seen is marked before broker.Send (verified by mutation).
  - id: BR-3
    disposition: addressed
    note: |
      Live check moved to #452 (exists, matching Done-when) in Revisions and the Plan checklist; the Done-when bullet itself is still unedited, raised separately.
  - id: BR-4
    disposition: addressed
    note: |
      Invalid threshold now refuses the start (message_service.go:217-221, run.go:780-784); plan Revisions records the change; TestWaitNoticeThreshold pins the parse.
  - id: BR-5
    disposition: addressed
    note: |
      messageAuthority.isHub is the one predicate, used at message_service.go:572 and wait_notice.go:256.
  - id: BR-6
    disposition: addressed
    note: |
      Unreadable mute list is reported in the notice body; TestWaitNoticeSaysWhenTheMuteListIsUnreadable covers it.
  - id: BR-7
    disposition: addressed
    note: |
      hubBinding resolves the slot via the registry liveness entry (newest incarnation), not map order.
findings:
  - id: new
    severity: Important
    family: done-when-unmet-at-close
    title: |
      The third Done-when bullet still requires the live TL check that Revisions moved to pair#452
    detail: |
      2nd finding in this family. Rule: any Done-when item deferred in Revisions must be edited in the Done-when list itself, in the same commit, struck through with a pointer to the issue that now owns it. Strike bullet 3 and point it at pair#452.
```

---

## Re-review — 2026-10-11T13:53:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 451 — Couch tells the hub when a slot is stuck waiting for someone |
| repo | pair |
| issue file | workshop/issues/000451-couch-tells-the-hub-when-a-slot-is-stuck-waiting-for-someone.md |
| boundary | whole-issue close |
| milestone | — |
| window | 536ad9671d39bfb86934fdabd4eae78856ff34da..a0a0eb1bb45a9c5476ab94fd468f043af471fb0c |
| command | sdlc close --issue 451 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T13:53:27-07:00 |
| verdict | SHIP |

## Review

Checking how the build generates its embedded runtime assets so I can run the tests.

Almost finished. I'm confirming the threshold-refusal wiring, then writing up the verdict.

```verdict
verdict: SHIP
confidence: high
```

This boundary fixes the one open finding, and I found nothing new that needs fixing before close. BR-8 is fixed: commit a0a0eb1b strikes Done-when bullet 3 and points it at pair#452. That issue file exists (`workshop/issues/000452-live-check-451-waiting-notices-after-a-couch-restart.md`), and the Plan checkbox and Revisions entry say the same thing. The code from earlier rounds still holds up:
- **Pure classifier:** `classifyWait`, `lastProseLine` and `dialogTitle` take no IO.
- **Watcher shell:** `waitWatcher` is thin and runs over the real `serviceRig`.
- **One hub predicate:** `messageAuthority.isHub` is used both by the allowance check and by the watcher.
- **Threshold check:** an invalid threshold refuses Couch's start before any socket is taken (`message_service.go:218`).
- **Tests:** after generating the runtime bundle, the wait-notice tests and the `artifactpath` inventory test pass. One test in the full `couchcmd` package fails, `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees`, but it fails the same way at base 536ad967. So it comes from the environment or was already there, and this diff did not cause it.

1. **Strengths**
   - `wait_notice.go:91` `dialogTitle` scans from the bottom for the selected option, so a numbered list in the agent's prose can't be mistaken for a dialog. This fixes the BR-1 class properly.
   - `waitNoticeThreshold` is pure, takes `getenv` as a parameter, is table-tested, and makes a typo fail loudly instead of silently using the default.
   - The notice goes out as an ordinary `Broker.Send` from the waiting slot. It reuses the existing breaker, delivery and ledger rather than adding a side channel (ARCH-DRY).
   - The episode map lives in memory and is pruned to the current listing every tick (ARCH-FUNERAL).
   - `hubBinding` resolves through the registry's newest incarnation, so the choice is deterministic.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The plan body still has the superseded text: line 37 ("invalid → default with … log line") and the unstruck Task 3 live step at line 102. The Revisions entry at line 106 records both changes, so the plan's audit trail is consistent. Striking them in the body would just be tidier.
   - `tick` ignores a `broker.Actors` error without a word (`return`). It retries on the next tick, so this is acceptable, but a stuck error would be invisible. I'm not raising it as a finding because it is a polling loop that retries, not a swallowed failure in place of a raise.

5. **Test coverage:** the classifier has table tests built from live-shaped Claude and Codex tails. Integration tests cover:
   - a dialog producing one notice per episode;
   - a new episode notifying again;
   - a question notifying, and a normal end on the resting branch staying silent;
   - idle off the resting branch notifying;
   - a muted slot and a missing hub staying silent;
   - a refused admission being retried (BR-2);
   - the threshold parse.

   The only thing left untested is the live check against a real terminal, which is pair#452's job.

6. **Architecture**
   - **ARCH-DRY:** pass (one hub predicate, the existing Send path).
   - **ARCH-PURE:** pass (pure classifier, thin tick shell).
   - **ARCH-PURPOSE:** pass. Both testable Done-when items are delivered. The live check is a separate step that needs the operator, and it is owned by #452.
   - **ARCH-MOCK:** pass. The tests run against real sockets and the real broker, with a tail-serving fake behind the `TailReader` seam.
   - **ARCH-CONSTRAINTS:** pass. It ticks once a minute, reads a tail only for a slot that's been quiet past the threshold, and each read has a 2-second timeout.
   - **ARCH-SECURE:** pass. The mute list is parsed into a typed set, and an unreadable list is named in the notice (BR-6). Tail text only goes out as quoted, length-capped message content.
   - **ARCH-ORDER:** pass. The episode state is a small explicit record (`notified`/`judgedNot` + key), and tests drive `tick` directly with controlled ordering.
   - **ARCH-FUNERAL:** pass. Nothing is written to disk, and the in-memory map is pruned every tick.

7. **Plan revisions:** none required. Optionally, strike plan lines 37 and 102 in the body so they match Revisions.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      a0a0eb1b strikes Done-when bullet 3 in place with a pointer to pair#452 (issue file exists); Plan checkbox and Revisions agree.
```
