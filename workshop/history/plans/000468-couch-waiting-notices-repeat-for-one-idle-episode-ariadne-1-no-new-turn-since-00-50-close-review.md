# Boundary Review — pair#468 (whole-issue close)

| field | value |
|-------|-------|
| issue | 468 — couch: waiting notices repeat for one idle episode (ariadne:1, no new turn since 00:50) |
| repo | pair |
| issue file | workshop/issues/000468-couch-waiting-notices-repeat-for-one-idle-episode-ariadne-1-no-new-turn-since-00-50.md |
| boundary | whole-issue close |
| milestone | — |
| window | 245325d8047031799f06251026f34f98779ac7c3..9fa31ab46ccae193bce10d2d7f354a8e69e458bd |
| command | sdlc close --issue 468 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:15:31-07:00 |
| verdict | SHIP |

## Review

Tests pass, and the new regression test fails when the fix is reverted. Writing up the verdict now.

```verdict
verdict: SHIP
confidence: high
```

This is a small fix aimed at the root cause. The watcher used to treat every new `LastActivity` as a new episode. Claude Code's update check redraws its footer every 30 minutes, which moved `LastActivity` and re-armed the episode each time. Now the watcher re-reads the screen after new output, and if the slot shows the same reason and summary it already reported, it stays in the same episode and sends nothing (`wait_notice.go:337-341`).

I checked the evidence myself. I generated the runtime assets and ran the `couchcmd` WaitNotice, DialogTitle and Classify tests: they pass. Then I put the base version of `wait_notice.go` back and the new `TestWaitNoticeOutputWithoutATurnKeepsTheEpisode` failed at `wait_notice_test.go:416` with two notices, the same repeat seen on ariadne:1. So the test does catch the bug.

Both Done-when clauses are met: the hour-long test is there, and the Log explains the cause with measured bytes and timestamps. The atlas is updated, and no README change is needed because nothing new is user-facing. Nothing blocks shipping.

**Strengths**
- **Root cause measured, not guessed:** the Log gives byte counts at 01:10:26 and 01:40:26 for the update check. The fix reads the screen rather than matching the "Checking for updates" text, so the rule that Pair only renders still holds (`wait_notice.go:334`).
- **Failed sends still retry:** when a send fails, the old episode is kept, so a real new wait is tried again on the next tick (`wait_notice.go:349-352`).
- **Old test corrected:** the #451 dialog test had treated a bare `Update` as a new episode, which encoded the bug. It now changes the screen first (`wait_notice_test.go:286-290`).
- **Both directions tested:** the new test checks that repeated output with the same screen gives one notice, and that new words on screen give a second one.

**Critical:** none.

**Important:** none.

**Minor**
- **A brief "not waiting" verdict wipes the episode memory.** At `wait_notice.go:333`, `w.seen[b] = waitEpisode{key: key, judgedNot: true}` drops `notified` and `judged`. If the slot is quiet past the threshold but briefly reads as not waiting (for example a message to it is still in flight), and then flips back with the same screen and no turn, a second notice goes out. That is the exact "message delivered but no turn started" case the Spec lists. This is the only place in the window that overwrites the episode. Fix: keep `notified` and `judged` on the judgedNot entry, or keep the last notified wait in a separate field.
- **The test leaves out the settle flip.** In the real cause, each burst of output also unsettles the wrapper and re-settles it 3 seconds later. The test only pushes `LastActivity`. This is harmless today because quiet is under the threshold during the flip, but it isn't pinned.
- **Atlas formatting:** the reflowed line at `atlas/couch.md:448` ("pending), so an idle slot costs…") is much longer than the lines around it.

**Test coverage:** both Done-when clauses are covered, and reverting the fix turns the new test red. The "Known limit" in the Log (a real turn that ends on the same last line of prose doesn't notify again) is documented but not tested. That's acceptable for a limit the Log owns.

**Architecture**
- **ARCH-DRY: pass.** No duplicated logic. The `judged` key is built once and reused at both use sites.
- **ARCH-PURE: pass.** The new check is a pure comparison inside the existing tick shell, and classification is still in the pure `classifyWait`.
- **ARCH-PURPOSE: pass.** The fix addresses what re-arms the episode, not just the update-check symptom. It also covers idle auto-compaction (one extra notice, as the Log says). The one leftover sibling is the judgedNot reset above, which is Minor.

**Plan revisions:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: episode-identity-lost-on-rejudge
    title: |
      A not-waiting verdict overwrites the notified episode, so a brief not-waiting state re-arms the notice
    detail: |
      wait_notice.go:333 replaces seen[b] with a judgedNot entry, dropping notified and judged. A slot quiet past the threshold that briefly reads as not waiting (a message in flight, Pending=true), then flips back with the same screen and no turn, sends a second notice. This is the only place in the window that resets the episode. Fix: keep notified and judged on the judgedNot entry.
  - id: new
    severity: Minor
    family: test-models-partial-signal
    title: |
      The 468 test pushes LastActivity only and leaves out the wrapper's unsettle and re-settle around each burst
    detail: |
      In the real cause, each update check also flips Settled false, then true 3 s later. The test keeps Settled true, so the flip is not pinned. It is harmless today because quiet is under the threshold during the flip.
  - id: new
    severity: Minor
    family: doc-reflow
    title: |
      atlas/couch.md Episode bullet has one overlong reflowed line
    detail: |
      The line starting "pending), so an idle slot costs one tail per output burst. The map is pruned..." is much longer than the lines around it.
```
