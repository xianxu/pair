# Boundary Review — pair#441 (whole-issue close)

| field | value |
|-------|-------|
| issue | 441 — session-sync: the owner's own ping must not become an owed event |
| repo | pair |
| issue file | workshop/issues/000441-session-sync-the-owner-s-own-ping-must-not-become-an-owed-event.md |
| boundary | whole-issue close |
| milestone | — |
| window | 33dbe6ac447df53a7c945d142c9c11b28c493be7..f01046f120c3ce8be310173389083662feb42070 |
| command | sdlc close --issue 441 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T09:57:22-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the breaker's identity predicate (model.go:282), then trying to build tests.

Identity predicate matches the breaker. Generating runtime assets so I can run the test and a mutation check.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The change is small and correct. It adds one guard in `deliveryRecorder.record` (`cmd/internal/couchcmd/message_deliveries.go:47`) that skips a delivery when its sender binding equals its recipient binding (`m.From == m.To`) and the body starts with `sessionsync.PingPrefix`. This uses the same identity predicate as the #433 breaker (`couchmessage/model.go:282`, `m.From == s.binding`). One table-driven test covers all three Done-when clauses. I generated the runtime bundle assets in the disposable checkout and ran the tests. `TestTheOwnersOwnPingIsNotOwed` passes. It goes red under both mutations: dropping the identity check (so `peer-ping` is lost) and disabling the guard (so `own-ping` is recorded). `sessionsync` passes. The only `couchcmd` failure is `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees`, which fails because of a `PAIR_DATA_DIR` environment conflict and has nothing to do with this diff; the issue log already records it as pre-existing. One thing blocks a clean SHIP: the atlas description of the ledger now gives an incomplete list of what it skips.

1. **Strengths**
   - The skip lives at the single ledger write path, so `sessionsync.Owed` stays a pure reader (ARCH-PURE pass).
   - Identity is checked on the whole `Binding`, not just slot equality. This matches the #433 breaker exactly, so a peer that imitates the prefix is still owed (`message_deliveries_test.go:165`).
   - `PingPrefix` sits next to `SubmitPrefix` in `sessionsync/owed.go:16` as the one source for the protocol string. No other Go code repeats the literal (ARCH-DRY pass).
   - The test covers both modes of each Done-when clause: own ping vs own submit, and owner vs peer for the same ping text.

2. **Critical:** none.

3. **Important**
   - `atlas/couch.md:434-440`: the **Ledger** bullet says the ledger "appends one row per delivery that may have reached the agent … It skips explicit cancel and expiry." That list of skips is now incomplete, because the owner's own `session-sync ping:` is also skipped (#441). The **Owed** bullet at `:453-458` should mention it too, or point to the Ledger bullet. Fix: add a sentence such as "It also skips the owner slot's own `session-sync ping:` (sender binding == recipient, #441), so a ping never seeds the next one." Family sweep for this window: these two atlas bullets are the only prose that lists what the ledger records or skips. The code comment on `record` was updated, and README.md does not describe the ledger's contents.

4. **Minor**
   - `message_deliveries_test.go:168-169`: the test throws away the errors from `Deliver` and `Ledger.Read`. If `Read` failed, `own-ping` would pass vacuously. The positive cases in the same loop would still catch that, so it is low risk, but checking the `Read` error is cheap.

5. **Test coverage notes:** Both mutations were checked and both go red. The test asserts "recorded in the ledger" rather than "listed by `Owed`". That is enough, because `Owed` is a pure filter over ledger rows and is unchanged here.

6. **Architectural notes:**
   - ARCH-DRY pass: the prefix has one source and the identity predicate matches the breaker's.
   - ARCH-PURE pass: the guard sits in the thin recorder, and `Owed` stays pure.
   - ARCH-PURPOSE pass: the loop is broken where it starts, which is the record path, with no follow-up deferred. Only the atlas consumer of the ledger's contract lags; see the Important finding.

   If parley.nvim ever changes the ping text, `PingPrefix` and parley#314 must change together, the same coupling `SubmitPrefix` already has.

7. **Plan revision recommendations:** none. The single plan item is delivered as described.

```findings
findings:
  - id: new
    severity: Important
    family: docs-contract-drift
    title: |
      atlas/couch.md Ledger and Owed bullets omit the owner-ping skip added by pair#441
    detail: |
      atlas/couch.md:434-440 says the ledger skips only explicit cancel and expiry. record() now also skips a delivery whose sender binding equals the recipient and whose body starts with "session-sync ping: ". Add that rule there and reference it from the Owed bullet (:453-458). Those two bullets are the only stale prose in this window.
  - id: new
    severity: Minor
    family: test-ignores-errors
    title: |
      TestTheOwnersOwnPingIsNotOwed discards Deliver and Ledger.Read errors
    detail: |
      message_deliveries_test.go:168-169. A failing Read would make the own-ping case pass vacuously. The positive cases in the same loop would catch it, but checking err is cheap.
```

---

## Re-review — 2026-10-11T09:59:52-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 441 — session-sync: the owner's own ping must not become an owed event |
| repo | pair |
| issue file | workshop/issues/000441-session-sync-the-owner-s-own-ping-must-not-become-an-owed-event.md |
| boundary | whole-issue close |
| milestone | — |
| window | 33dbe6ac447df53a7c945d142c9c11b28c493be7..648509047e31c2f71b39bfd289f5da7c6467563b |
| command | sdlc close --issue 441 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T09:59:52-07:00 |
| verdict | SHIP |

## Review

Tests pass. Next, a quick mutation check: revert the skip and confirm the test goes red.

```verdict
verdict: SHIP
confidence: high
```

**Summary:** This is ready to ship, with no new findings. The fix lives in `deliveryRecorder.record` (`cmd/internal/couchcmd/message_deliveries.go:47`), the only place that writes to the ledger. It skips a delivery only when the full sender binding equals the recipient (`m.From == m.To`) and the body starts with `sessionsync.PingPrefix`. That is the same identity rule the breaker uses for the owner's own submit (`couchmessage/model.go:282`, #433), so a peer can't suppress a real owed event by copying the ping text. Both findings from the last round are fixed. The atlas Ledger and Owed bullets now describe the skip. The test now checks the `Deliver` and `Read` errors. The test itself fails without the fix: I disabled the skip and `own-ping` failed; I dropped only the identity check and `peer-ping` failed. With the code restored it passes.

The checkout couldn't build the package at first because the embed assets under `internal/runtimebundle/assets/runtime/` were missing. To run the tests I added a placeholder `files/` entry and a `manifest.json`, then deleted both afterwards. The checkout is back to its original state.

1. **Strengths**
   - Identity is checked on the whole binding, not just the slot, matching the breaker's predicate (`model.go:282`). So a peer sending ping-shaped text stays owed.
   - One write path and one change: `Owed` still only reads (`sessionsync/owed.go` gains just the `PingPrefix` constant, next to `SubmitPrefix`).
   - The table test (`message_deliveries_test.go:150-183`) covers four cases: own ping, own submit, peer ping and peer report. That exercises every case in the Done-when list.
   - The lesson added to `workshop/lessons.md` names the actual root cause of BR-1: before passing `--no-atlas`, grep the atlas for the rule you changed.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** none new.
   - The Done-when wording says "leaves `Owed` unchanged", but the test asserts on ledger rows. That's equivalent, because `Owed` only reads the ledger.

5. **Test coverage**
   - `TestTheOwnersOwnPingIsNotOwed` passes.
   - Both mutations (no skip, and no identity check) make it fail.
   - The neighbouring ledger tests still pass.

6. **Architecture**
   - **ARCH-DRY: pass.** The prefix is a single constant. The identity check is the same one-line comparison the breaker uses, not a copied block.
   - **ARCH-PURE: pass.** The filter is a pure check at the start of the recorder, before the IO append.
   - **ARCH-PURPOSE: pass.** The skip sits at the one place the ledger is written, so no other path can still record the ping and seed the loop.

7. **Plan revisions:** none. The plan's one checklist item matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      atlas/couch.md:437-440 now states the owner-ping skip (sender binding == recipient + prefix, #433 identity) and Owed bullet :458-459 references it; matches message_deliveries.go:47 and model.go:282.
  - id: BR-2
    disposition: addressed
    note: |
      message_deliveries_test.go:167-173 now fails on Deliver and Ledger.Read errors.
```
