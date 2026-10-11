# Boundary Review — pair#438 (whole-issue close)

| field | value |
|-------|-------|
| issue | 438 — Peer delivery to a busy agent: confirm a message the agent queued mid-turn, and report the real blocking condition |
| repo | pair |
| issue file | workshop/issues/000438-peer-delivery-to-a-busy-agent-confirm-a-message-the-agent-queued-mid-turn-and-report-the-real-blocking-condition.md |
| boundary | whole-issue close |
| milestone | — |
| window | b70c52f79a677c00e5b52beb3339c1d7f7cf31d2..9f2c7f0b3e1fcddd02cadb4b2c0102c5fd538def |
| command | sdlc close --issue 438 |
| reviewer | claude |
| timestamp | 2026-10-11T01:55:02-07:00 |
| verdict | SHIP |

## Review

Sandbox failures. Run the targeted tests and grep properly.

```verdict
verdict: SHIP
confidence: high
```

The change delivers all three Done-when clauses with tests that actually test something. The new `FrameComplete` check on `outputPending` (`peer_delivery.go:442`) is the measured root cause from the Log. The framed and unframed fixtures together act as a mutation check. `TestPeerBusyAgentQueuedMessageSubmits` and `TestPeerBusyAgentStreamingMinutesLands` deliver only because a synchronized frame is complete. `TestPeerUnframedStreamExpiresNamingOutput` uses the same screens with the frames stripped, and it expires naming `rendered output`. That shows `outputPending` stays above zero throughout, so the new gate is what lets the framed runs through. The wait accounting is bounded and in memory, and it resets at enqueue and at the paste. The targeted peer and frame tests pass (`go test -run 'PeerBusy|FrameComplete|RecordedTurn|Unframed|LongestWait|PeerHold|PeerDelivery|PeerInput|PeerPoll'` → ok). In the full package run, every failure was a sandbox `/tmp` mkdir or socket-bind denial, none in the changed logic. Nothing blocks SHIP; only Minor notes remain.

1. **Strengths**
   - `terminal_model.go:227-232`: frame tracking lives in the existing control observer, behind its `?`-prefix and `h`/`l` filter. A DECRQM query (`?2026$p`) can't mark the stream as framed. The new code is agent-agnostic, as planned.
   - `FrameComplete` fails closed: if the stream never framed, or the model is closed, it reports false, so non-framing agents keep today's rule.
   - The queued-message confirmation reuses the existing evidence (the composer was occupied after the paste and is empty now) instead of adding a Claude-specific recognizer. `turn-queued.raw` reads as Empty (`TestPeerComposerReadsRecordedTurnScreens`).
   - `TestPeerExpiryNamesTheLongestWait` mixes half-drawn frames into a run of composer-less frames. It shows the longest wait comes first even when the last check was a different one, which is exactly the clause in the Spec.
   - The fixture README is honest about how the screens were made: re-rendered, not raw slices, with the reason given.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `peer_delivery.go:639` (doc comment for `peerConfirmEvidence`): on expiry after the submit, the detail is now a duration-ranked history rather than "the composer and turn as the wrapper last saw them". The last reading is dropped if it stood under 1s or ranks below third. The joined form is also hard to read, because the evidence string itself contains `; `, the same separator the summary uses.
   - `atlas/couch.md:304`: the claims that "Claude Code and Codex frame each repaint" and that Codex queues mid-turn input rest on no Codex measurement or fixture. Only Claude Code was replayed.
   - `waitSummaryLocked` caps the list at 3 entries and drops anything under 1s. The atlas doesn't mention either rule.
   - The `framed`/`frameOpen` flags survive RIS (`ESC c`, which `handleESC` already sees for visibility) and a later program that doesn't frame its output. The effect is that output in flight is treated as complete after a hand-off. This is low risk.

5. **Test coverage notes:** each Done-when clause is covered:
   - The queued message ends `submitted` (`TestPeerBusyAgentQueuedMessageSubmits`).
   - A not-ready composer under continuous output is named on expiry (`TestPeerExpiryNamesTheLongestWait`).
   - Streaming for 3+ minutes lands (`TestPeerBusyAgentStreamingMinutesLands`).
   - The observer has its own unit test (`TestTerminalModelFrameComplete`).
   - Gap: nothing pins the confirming-half expiry detail in its new summarized form beyond the existing `Contains(evidence)` check.

6. **Architectural notes**
   - ARCH-DRY: pass. Frame state lives in the one control observer, and no logic is duplicated.
   - ARCH-PURE: pass, with a note. `chargeWaitLocked` and `waitSummaryLocked` read `d.now()` through the injected clock, and the summary logic is a near-pure ranking that tests can reach. If the summary grows, extracting it as `func(map[string]time.Duration) string` would let it be tested on its own.
   - ARCH-PURPOSE: pass. The fix goes after the real blocker (the starving gate), not the symptom alone, and all three Spec bullets are delivered without deferral. The issue's Spec allowed either busy-agent policy; the plan chose paste-mid-turn and justified it.

7. **Plan revision recommendations:** none. The Plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: doc-claim-matches-code
    title: |
      peerConfirmEvidence doc says the last-seen reading; the expiry now reports a ranked history
    detail: |
      peer_delivery.go:639 says the uncertain receipt reports the composer and turn as last seen. waitSummaryLocked replaces that with the longest waits, can drop the final reading (under 1s, or below third), and joins with the same separator the evidence uses internally. Other instances in this window: atlas/couch.md:355 does not mention the top-3 cap or the 1s floor.
  - id: new
    severity: Minor
    family: unmeasured-claim
    title: |
      Atlas and Plan say Codex frames repaints and queues mid-turn input; only Claude Code was measured
    detail: |
      atlas/couch.md:304 and the Plan's policy bullet both name Codex, but there is no Codex recording or fixture. Either measure it or scope the wording to Claude Code. The safe fallback (an unframed stream keeps the old rule) limits the impact.
  - id: new
    severity: Minor
    family: terminal-reset-clears-mode-state
    title: |
      RIS does not clear the framed and frameOpen flags
    detail: |
      handleESC resets visibility on ESC c but leaves the frame state. After a reset, or a hand-off to a program that does not frame, output in flight still counts as complete. Low risk.
```
