# Boundary Review — pair#433 (whole-issue close)

| field | value |
|-------|-------|
| issue | 433 — Couch peer-message breaker: a leaky bucket, a full refill on session-sync submit, and a higher allowance for the TL slot |
| repo | pair |
| issue file | workshop/issues/000433-couch-peer-message-breaker-a-leaky-bucket-a-full-refill-on-session-sync-submit-and-a-higher-allowance-for-the-tl-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8d7c50ef44e16b94bd5275230dd80fd32430c2c1..7afc4faaa843a4e4073c0200c0c26032b58f1cb4 |
| command | sdlc close --issue 433 |
| reviewer | claude |
| timestamp | 2026-10-11T00:55:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Model tests pass. One last check: are any callers still reading the stale `Remaining()` instead of `RemainingAt`?

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The leaky bucket is right. It lives in the pure `Advance` model and covers the edge cases: refill is capped at capacity, the refill clock restarts when the bucket is full, and a partial interval carries over (`refilledAt += n*RefillInterval`). Every Done-when row has a model test, and `go test ./cmd/internal/couchmessage/` passes. The listing uses the time-aware `RemainingAt`, and no non-test caller still reads the raw `Remaining()`. The atlas and the skill text are updated. Two things keep this from SHIP. First, the session-sync path and the hub detection are each proven only at the layer below the one that actually runs: there is no broker or service test that sends a message to the sender's own slot. Second, the code that works out the hub from `os.Getwd` is never exercised. Neither is a correctness bug I could show, so this is FIX-THEN-SHIP.

**Strengths**
- `cmd/internal/couchmessage/model.go:141-154`: `refill` is pure. `RemainingAt` uses the same arithmetic, so the listing and admission can't disagree (ARCH-DRY pass).
- `model.go:259`: refill only runs when `e.At` is after `refilledAt`, so an event arriving out of order can't credit a negative interval.
- `model_test.go`: the ping-pong test runs in both modes, with the hub and without it. That covers the hub+worker loop the plan names.
- `broker.go:141`: `Register` now delegates to `RegisterWithAllowance`, so existing callers are unchanged. A reconnect keeps its bucket, as the comment says.
- Hub match: `Resolve` and `identity.WorktreeRoot` both return physical paths, so symlinked worktrees compare correctly.

**Critical**: none.

**Important**
1. **Done-when 2 is proven only in the model.** `TestSelfAddressedMessageRefillsFully` calls `Advance` directly. Nothing checks that a real `--send-to` to one's own exact slot gets through the broker. That path includes `b.caller(from)`, `ResolveRecipient` with the sender as a candidate (only family routes exclude the sender, `broker.go:504`), and `Reserve` on the sender's own endpoint. If any of those refuses a self-send, the feature is dead even though every test is green.
   - Fix: add a `message_service_test` or `broker_test` case. Drain a slot's bucket, send an exact `--send-to` from that same binding, and assert it is admitted and the bucket is full again.
2. **The hub derivation glue is untested.** `message_service.go:229-234` (`os.Getwd` → `ResolveTree`) is never run in a test: `TestMessageHubSlotGetsTheLargerAllowance` injects `world.hub` directly.
   - Fix: lift the lookup into a function that takes the cwd and a resolver, and test the three cases (inside a slot, in a subdirectory of a slot, outside a checkout). This also keeps the IO thin, per ARCH-PURE.

**Minor**
- Self-send refill is a way around the breaker. An agent in a two-agent loop that also messages itself each turn keeps its own inbound bucket full, so the loop is never cut. The Log documents this as a residual, and the skill says not to evade. Even so, the new skill sentence ("a session-sync submit refills it") tells agents how the refill works without saying "not by messaging yourself." Consider adding that explicitly.
- `Event.Allowance` treats `<= 0` as the default. A negative allowance passed by mistake silently becomes 8; that's harmless.
- `RemainingAt` on a disconnected actor still credits refills. That's harmless because listings skip disconnected actors.

**Test coverage**: all four Done-when rows have model tests. Rows 2 and 3 have no test at the layer where the behaviour actually happens (Important 1 and 2).

**Architecture**
- ARCH-DRY: pass.
- ARCH-PURE: pass for the model; the hub-root lookup is IO left inline (Important 2).
- ARCH-PURPOSE: pass. All three spec bullets are delivered, and the remaining allowance is visible via `--actors`.

**Plan revisions**: none needed.

```findings
findings:
  - id: new
    severity: Important
    family: proof-at-wrong-layer
    title: |
      Self-addressed (session-sync) refill proven only in the pure model, not through broker routing
    detail: |
      No broker or service test sends an exact --send-to from a binding to its own slot. The caller check, ResolveRecipient with the sender as a candidate, and Reserve on the sender's own endpoint are unexercised, so Done-when 2 could be dead in production while tests stay green. Siblings in this window: the hub allowance (next finding).
  - id: new
    severity: Important
    family: proof-at-wrong-layer
    title: |
      Hub derivation from os.Getwd/ResolveTree in startMessageService is untested glue
    detail: |
      TestMessageHubSlotGetsTheLargerAllowance injects world.hub directly, so message_service.go:229-234 never runs in a test. Extract a small function (cwd plus resolver to hub) and test it inside a slot, in a subdirectory of a slot, and outside a checkout (ARCH-PURE).
  - id: new
    severity: Minor
    family: breaker-evasion-surface
    title: |
      An agent messaging itself refills its own bucket, so a loop with self-sends is never cut
    detail: |
      Documented as a residual in the Log. The skill text could say explicitly that messaging yourself is not a legitimate refill.
```

---

## Re-review — 2026-10-11T00:58:59-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 433 — Couch peer-message breaker: a leaky bucket, a full refill on session-sync submit, and a higher allowance for the TL slot |
| repo | pair |
| issue file | workshop/issues/000433-couch-peer-message-breaker-a-leaky-bucket-a-full-refill-on-session-sync-submit-and-a-higher-allowance-for-the-tl-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8d7c50ef44e16b94bd5275230dd80fd32430c2c1..c79b08680f27dc6994b26e640e98cb776082c697 |
| command | sdlc close --issue 433 |
| reviewer | claude |
| timestamp | 2026-10-11T00:58:59-07:00 |
| verdict | SHIP |

## Review

Still reviewing #433; one service test failed, so I'm checking whether that's the sandbox or a real failure.

```verdict
verdict: SHIP
confidence: high
```

The window does what the issue asks. The leaky bucket, the self-addressed refill and the larger allowance for the slot Couch started from (the "hub", the TL slot) all live in the pure `ActorState`/`Advance` model in `model.go`. The broker and service layers only add a register-time allowance and a `Capacity` field, and both are exercised. All three prior findings are fixed with real evidence, and I found nothing new worth blocking on.

Inside the sandbox, every test that creates a socket under `/tmp` fails with "operation not permitted". That is the sandbox, not the code. Unsandboxed, the full `couchmessage` package passes, and so do the `couchcmd` tests `-run 'TestMessage|TestHub'`.

1. **Strengths**
   - `model.go:141-154`: `refill` is pure arithmetic shared by `Advance` and `RemainingAt`. The listing and admission paths cannot disagree about how many messages a slot has left, so there is one source of truth (ARCH-DRY, ARCH-PURE pass).
   - `model.go:279-287`: a self-addressed message refills instead of spending, as a single rule in the model. Routing still sees `c.Remaining = InboundAllowance` (`broker.go:499`), so the model's budget check stays the authoritative one.
   - `TestPingPongIsCutOffWithinSeconds` checks both the ordinary and the hub case. It proves the worker's burst of 8 still cuts off a ping-pong even when one side is the hub. That is the safety property most at risk from this change.
   - `hubWorktree(getwd, resolve)` is a thin, injected seam. Its test covers the checkout root, a subdirectory, outside any checkout, and a failing working-directory lookup.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The atlas says "spaced reports never do" empty the bucket. That holds only for reports at least 3 minutes apart (one `RefillInterval`). The code comment says "minutes apart"; the atlas could say the same.
   - The edited paragraph in `SKILL.md` has one very long unwrapped line ("…editor. Do not manufacture…"). Cosmetic.

5. **Test coverage**
   - Every `## Done when` clause has a model test.
   - Clause 2 (self-send refill) is now also proven through `Broker.Send`. If the self-send rule is removed, the owner's send fails with `ErrBudgetExhausted`, because routing passes but the model refuses. So the regression test would go red without the fix.
   - The hub comparison at `message_service.go:555` is covered by `TestMessageHubSlotGetsTheLargerAllowance`. Both sides of that comparison use physical paths: `Resolve` resolves physical paths, and the identity uses a canonical path.

6. **Architecture**
   - **ARCH-DRY:** pass. The refill logic is shared, and `Register` now delegates to `RegisterWithAllowance`.
   - **ARCH-PURE:** pass. The budget policy sits entirely in the pure model, and the service only picks which allowance to register with.
   - **ARCH-PURPOSE:** pass. All four parts of the spec are delivered: the bucket, the session-sync refill, the hub allowance, and the listing showing `remaining/capacity`.

7. **Plan revisions:** none needed. The plan matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestSelfAddressedSendRefillsThroughTheBroker drives Broker.Send from the owner to its own exact slot after the burst is spent; without the From==binding rule the model returns ErrBudgetExhausted, so the test fails without the fix.
  - id: BR-2
    disposition: addressed
    note: |
      hubWorktree(getwd, resolve) was extracted at message_service.go:184 and is wired with os.Getwd and c.ResolveTree; TestHubWorktreeIsTheStartupCheckout covers the checkout root, a subdirectory, outside a checkout and a cwd error, using a real git init.
  - id: BR-3
    disposition: addressed
    note: |
      The couch SKILL.md now says not to message your own slot to refill it, and that the path exists for the session-sync editor; the Log records the residual.
findings:
  - id: new
    severity: Minor
    family: doc-claim-overstates-code
    title: |
      The atlas says spaced reports "never" empty the bucket; true only at intervals of at least RefillInterval (3m)
    detail: |
      The only instance in this window is atlas/couch.md (the leaky-bucket paragraph). The model.go comment already qualifies it as "minutes apart".
```
