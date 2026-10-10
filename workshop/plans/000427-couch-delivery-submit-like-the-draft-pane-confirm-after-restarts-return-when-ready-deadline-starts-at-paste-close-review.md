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
