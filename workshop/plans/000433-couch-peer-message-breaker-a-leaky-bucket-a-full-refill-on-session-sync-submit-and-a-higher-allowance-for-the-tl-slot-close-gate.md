---
gate: boundary-review
issue: 433
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T00:55:41-07:00"
      dispatched: "2026-10-11T00:54:33-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Self-addressed (session-sync) refill proven only in the pure model, not through broker routing
          detail: 'No broker or service test sends an exact --send-to from a binding to its own slot. The caller check, ResolveRecipient with the sender as a candidate, and Reserve on the sender''s own endpoint are unexercised, so Done-when 2 could be dead in production while tests stay green. Siblings in this window: the hub allowance (next finding).'
          family: proof-at-wrong-layer
          round: 1
        - id: BR-2
          severity: Important
          title: Hub derivation from os.Getwd/ResolveTree in startMessageService is untested glue
          detail: TestMessageHubSlotGetsTheLargerAllowance injects world.hub directly, so message_service.go:229-234 never runs in a test. Extract a small function (cwd plus resolver to hub) and test it inside a slot, in a subdirectory of a slot, and outside a checkout (ARCH-PURE).
          family: proof-at-wrong-layer
          round: 1
        - id: BR-3
          severity: Minor
          title: An agent messaging itself refills its own bucket, so a loop with self-sends is never cut
          detail: Documented as a residual in the Log. The skill text could say explicitly that messaging yourself is not a legitimate refill.
          family: breaker-evasion-surface
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T00:58:59-07:00"
      dispatched: "2026-10-11T00:57:41-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestSelfAddressedSendRefillsThroughTheBroker drives Broker.Send from the owner to its own exact slot after the burst is spent; without the From==binding rule the model returns ErrBudgetExhausted, so the test fails without the fix.
          round: 2
        - id: BR-2
          disposition: addressed
          note: hubWorktree(getwd, resolve) was extracted at message_service.go:184 and is wired with os.Getwd and c.ResolveTree; TestHubWorktreeIsTheStartupCheckout covers the checkout root, a subdirectory, outside a checkout and a cwd error, using a real git init.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The couch SKILL.md now says not to message your own slot to refill it, and that the path exists for the session-sync editor; the Log records the residual.
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: The atlas says spaced reports "never" empty the bucket; true only at intervals of at least RefillInterval (3m)
          detail: The only instance in this window is atlas/couch.md (the leaky-bucket paragraph). The model.go comment already qualifies it as "minutes apart".
          family: doc-claim-overstates-code
          round: 2
      recipe: small-diff-review
      reviewed: c79b08680f27dc6994b26e640e98cb776082c697
      blocked: false
---

# Gate ledger — pair#433 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T00:55:41-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `proof-at-wrong-layer` Self-addressed (session-sync) refill proven only in the pure model, not through broker routing
  No broker or service test sends an exact --send-to from a binding to its own slot. The caller check, ResolveRecipient with the sender as a candidate, and Reserve on the sender's own endpoint are unexercised, so Done-when 2 could be dead in production while tests stay green. Siblings in this window: the hub allowance (next finding).
- **BR-2** [Important] `proof-at-wrong-layer` Hub derivation from os.Getwd/ResolveTree in startMessageService is untested glue
  TestMessageHubSlotGetsTheLargerAllowance injects world.hub directly, so message_service.go:229-234 never runs in a test. Extract a small function (cwd plus resolver to hub) and test it inside a slot, in a subdirectory of a slot, and outside a checkout (ARCH-PURE).
- **BR-3** [Minor] `breaker-evasion-surface` An agent messaging itself refills its own bucket, so a loop with self-sends is never cut
  Documented as a residual in the Log. The skill text could say explicitly that messaging yourself is not a legitimate refill.

## Round 2 — 2026-10-11T00:58:59-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TestSelfAddressedSendRefillsThroughTheBroker drives Broker.Send from the owner to its own exact slot after the burst is spent; without the From==binding rule the model returns ErrBudgetExhausted, so the test fails without the fix.
- BR-2 — addressed — hubWorktree(getwd, resolve) was extracted at message_service.go:184 and is wired with os.Getwd and c.ResolveTree; TestHubWorktreeIsTheStartupCheckout covers the checkout root, a subdirectory, outside a checkout and a cwd error, using a real git init.
- BR-3 — addressed — The couch SKILL.md now says not to message your own slot to refill it, and that the path exists for the session-sync editor; the Log records the residual.

### Raised

- **BR-4** [Minor] `doc-claim-overstates-code` The atlas says spaced reports "never" empty the bucket; true only at intervals of at least RefillInterval (3m)
  The only instance in this window is atlas/couch.md (the leaky-bucket paragraph). The model.go comment already qualifies it as "minutes apart".

## Open findings

- **BR-4** [Minor] `doc-claim-overstates-code` The atlas says spaced reports "never" empty the bucket; true only at intervals of at least RefillInterval (3m)
