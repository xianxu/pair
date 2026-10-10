# Boundary Review — pair#427 (whole-issue close)

| field | value |
|-------|-------|
| issue | 427 — couch delivery: submit like the draft pane, confirm after; restarts return when ready; deadline starts at paste |
| repo | pair |
| issue file | workshop/issues/000427-couch-delivery-submit-like-the-draft-pane-confirm-after-restarts-return-when-ready-deadline-starts-at-paste.md |
| boundary | whole-issue close |
| milestone | — |
| window | 57932d262e985497c942d0f60f8988ba1ad709b4..6e18ca97788703205fdd8671531d3039945c1a0f |
| command | sdlc close --issue 427 |
| reviewer | claude |
| timestamp | 2026-10-10T12:51:23-07:00 |
| verdict | SHIP |

## Review

Diff and plan read; now checking the restart-readiness goroutine's concurrency and the tests.

Running tests; sandbox blocks some socket tests, so checking which failures are environmental.

Tests pass outside the sandbox. A last check on the atlas, then the verdict.

```verdict
verdict: SHIP
confidence: medium
```

**Verdict: SHIP (medium confidence).** The issue's Spec and Done-when are delivered:

- **Delivery:** the reducer now goes paste → fixed 150ms delay → submit → confirm. The render matcher and its helpers are gone; `git grep` finds no leftover references.
- **Uncertain outcomes:** a deadline or an interrupt after the paste ends `indeterminate` with an `uncertain:` detail and evidence. It never ends as a silent `expired`.
- **Deadline:** it now starts at the paste. `Message.Deadline` means paste-by (90s), and `Message.Horizon()` is the single outer bound everything on the broker side reads.
- **Restarts:** `relaunch` and `reload-context` receipts stay `running` until a *new* session reports Settled. If it doesn't within 2 minutes, the receipt is `failed` with code `unready`. Settled now also waits for the startup orientation, and finishing the orientation re-triggers the settle check.

**Test runs:**
- Sandboxed, many tests fail with "operation not permitted" on `/tmp`. Those failures are from the sandbox, not the code.
- Unsandboxed, `couchmessage` and `wrapcmd` pass in full.
- `couchcmd` has two failures. `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins` passes when run alone, so it is flaky under load. `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` fails because `git push` fails in this environment (exit 128). Neither is caused by this diff.
- `-race` is clean on the new readiness and peer-delivery tests.

Nothing blocks shipping. The findings are all Minor: two places where the code drifted from the plan, and the readiness goroutine's lifetime.

### 1. Strengths
- `couchmessage/delivery.go:81-100`: one `BodyWritten` branch turns every interrupting event after the paste into Indeterminate. That covers the "text left in the box" class of bug as a whole, not one event at a time (ARCH-PURPOSE).
- `Message.Horizon()` (`model.go:67`) is the only bound the broker side reads: model `Tick`, late `DeliveryFinished`, `runActor` job context and the endpoint receipt context. The commit RPC context deliberately stays on `Deadline` (ARCH-DRY).
- `awaitReadiness` and `readinessAfter` keep the 2-minute wait off the console queue, the wait is bounded, and `TestRestartReceiptWaitsForReadiness` exercises it end to end through the receipt the CLI polls. It covers a settled old session, a busy new one, the timeout, a pre-Build wrapper, and other verbs.
- The confirmation evidence is agent-agnostic: the composer was seen occupied and is now empty, or a turn opened after the submit. `peerConfirmEvidence` puts a readable reason on uncertain receipts.
- The `slotOperationOutcome` fix: an empty `ReceiptCode()` no longer overwrites the error's code (`slot_operations.go:165`), and the readiness test catches it.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **Plan drift on pre-readiness wrappers:** the plan says a new session with no `Build` "returns at once with a warning". The code returns `RestartUnready`, i.e. `failed`/`unready` (`live_restart_probe.go:57`). The atlas documents the code's behavior, so only the plan is stale.
- **Plan drift on poll rate:** the plan says the poll runs at 100ms after the paste; the code uses 50ms (`peer_delivery.go:629`).
- **Readiness goroutine lifetime (ARCH-ORDER):** it runs on `context.Background()` (`live_restart_probe.go:85`), so shutting down Couch doesn't cancel it. It is bounded by `readyWithin`, but could use the console's lifetime context.
- **Queue key released during the readiness wait:** the console queue frees the slot's key when the job returns, so a second restart of the same slot is accepted while the first receipt is still `running`. It's harmless, because `before` is re-captured, but it isn't documented.

### 5. Test coverage notes
- The reducer, the wrapper boot simulation, the multi-line collapsed paste, a paste that is never consumed, settle with orientation, and the broker horizon model are all covered with injected clocks and fakes.
- The "a turn was already running at submit" case (`turnAtSubmit` true) isn't covered by a test.
- The live check after landing (`couch --reload-context pair:N` followed by a send) is still pending, as the plan says.

### 6. Architectural notes
- **ARCH-DRY, ARCH-PURE, ARCH-PURPOSE:** pass.
- **ARCH-MOCK:** pass. A fake harness session plus the live conformance run.
- **ARCH-CONSTRAINTS:** pass. Every wait is bounded: 90s to paste, a 30s window after it, and 2 minutes for readiness. The CLI poll budget went from 3 to 5 minutes to cover the readiness wait.
- **ARCH-SECURE:** N/A. Nothing new is parsed from outside the process.
- **ARCH-ORDER:** pass, except the goroutine-lifetime note above. State still moves through `AdvancePeerDelivery`; the new wrapper fields (`sawOccupied`, `submitSequence`, `turnAtSubmit`) are evidence gathered for the reducer, not authoritative state.
- **ARCH-FUNERAL:** pass. Nothing durable is created.

### 7. Plan revision recommendations
- Add a `## Revisions` entry: a new session with no `Build` gives `failed`/`unready` at once, not a warning. The poll after the paste runs at 50ms, not 100ms.

```findings
findings:
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Plan says a pre-Build wrapper returns with a warning; code fails the receipt as unready (and the poll is 50ms, not 100ms)
    detail: |
      live_restart_probe.go:57 returns RestartUnready, which becomes failed/unready; atlas matches the code. Add a Revisions entry to the plan.
  - id: new
    severity: Minor
    family: unbounded-goroutine-extent
    title: |
      awaitReadiness goroutine runs on context.Background, so Couch shutdown does not cancel it
    detail: |
      Bounded by readyWithin (2m), so it cannot leak forever; passing the console lifetime context would end it on shutdown.
  - id: new
    severity: Minor
    family: test-gap-turn-already-running
    title: |
      No test for confirmation when a turn was already running at submit (turnAtSubmit true)
```

---

## Re-review — 2026-10-10T12:53:41-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 427 — couch delivery: submit like the draft pane, confirm after; restarts return when ready; deadline starts at paste |
| repo | pair |
| issue file | workshop/issues/000427-couch-delivery-submit-like-the-draft-pane-confirm-after-restarts-return-when-ready-deadline-starts-at-paste.md |
| boundary | whole-issue close |
| milestone | — |
| window | 57932d262e985497c942d0f60f8988ba1ad709b4..45eb3e60e59fc52035b4191acb6f7d061e58a846 |
| command | sdlc close --issue 427 |
| reviewer | claude |
| timestamp | 2026-10-10T12:53:41-07:00 |
| verdict | SHIP |

## Review

Prior round dispositions focus on the last commit. Let me see it.

Those failures are sandbox /tmp denials, not regressions; checking the non-sandbox subset passes.

```verdict
verdict: SHIP
confidence: high
```

All three prior findings are resolved. The plan now has a `## Revisions` entry that matches the code: a wrapper too old to report readiness fails the restart receipt as `unready`, and the poll after the paste runs every 50ms. The confirmation table has two new cases with `busy` set before the submit write, so `turnAtSubmit` is true. I ran `TestPeerDeliverySubmitsAfterDelayThenConfirms` and it passes, both new cases included. The one advisory the implementer declined (the readiness goroutine on `context.Background`) is minor; their reasoning is recorded and holds up. I re-read the main changes and found nothing that blocks shipping. Some tests failed in my sandbox, but only with `mkdir /tmp/...: operation not permitted` and socket-bind denials. That is the sandbox's `/tmp` restriction, not a fault in the code.

1. **Strengths**
   - `couchmessage/delivery.go:79-100`: once the paste is written, an interruption (input, dialog, deadline or the recipient exiting) ends the delivery as `Indeterminate`, never `Expired` or `Cancelled`. This removes the false-negative receipt.
   - Readiness now comes from signals that don't depend on which agent is running. `peer_delivery.go:374-390` checks only whether the composer is occupied, empty or unknown, plus whether a turn opened since the submit. The old envelope-matching step `peerComposerMatches` is gone, so nothing inspects how the agent drew the pasted text any more.
   - The timing has one clear structure. `Message.Horizon()` is the broker's single outer bound (`model.go:67`), and `deadlineLocked()` (`peer_delivery.go:440`) picks the bound for the current half of the delivery: the paste-by deadline before the paste, the paste-relative window after it.
   - Readiness is pure and testable. `wrapperSettled` (`peer_settle.go:20`) gained an orientation term, and restart readiness keys on a new session (`live.Session != before`), so a stale settled report from the old session can't be mistaken for success.
   - `slot_operations.go:178`: an empty partial-outcome code no longer overwrites the error's `unready` code.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** `AwaitReady` reads `p.service` once at entry. If that read happened before `probe.service.Store` (a narrow startup race), the wait would time out as `unready` instead of panicking. It fails safe, so I'm only noting it.
5. **Test coverage:** the confirmation outcomes are covered as a table, now including the case where a turn was already running at submit. The restart readiness tests cover ready, unready, a wrapper that can't report, and the verbs that don't wait. Those readiness tests need `/tmp` access, so I couldn't run them inside the sandbox.
6. **Architecture:**
   - **ARCH-DRY: pass.** The duplicate composer-matching code was deleted.
   - **ARCH-PURE: pass.** The reducer and `wrapperSettled` are pure.
   - **ARCH-PURPOSE: pass.** Delivery submits like the draft pane, confirms afterwards, and the window starts at the paste; restarts return only when ready.
   - **ARCH-MOCK: pass.** The tests drive harness fakes.
   - **ARCH-CONSTRAINTS: pass.** The 50ms poll runs only while a delivery is in flight, and the 2-minute readiness bound is stated.
   - **ARCH-SECURE: N/A.** No new untrusted input or secrets.
   - **ARCH-ORDER: pass.** The new `Pasted` and `Confirming` phases go through `AdvancePeerDelivery`, and uncertainty is kept as `Indeterminate`, never collapsed into success or failure.
   - **ARCH-FUNERAL: pass.** Nothing durable is created; the readiness goroutine is bounded by `readyWithin`.
7. **Plan revisions:** none beyond the one already added.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The plan's Revisions entry (2026-10-10) records the failed/unready outcome and the 50ms poll, matching live_restart_probe.go:157 and peer_delivery.go:103.
  - id: BR-2
    disposition: withdrawn
    note: |
      Declined in Revisions with a sound reason: readyWithin (2m) bounds the goroutine and process exit ends it, so its lifetime is bounded and this is not a leak.
  - id: BR-3
    disposition: addressed
    note: |
      peer_delivery_test.go adds "queued behind a running turn" and "running turn, text stays" with turnActive set before the submit write; both pass.
```
