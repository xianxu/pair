---
id: 000435
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '107fbea96d7b55e6b29079ecf1a0dc72521bf730' # card fields mirrored from issue-cards; edit via sdlc
---

# Peer delivery: a hold the operator causes (scrolled up, draft in the composer) doesn't spend the delivery deadline

## Problem

A peer message must start its paste by `Message.Deadline` (`cmd/internal/couchmessage/model.go`). pair#428 made the
recipient's wrapper hold a delivery while the operator has the agent pane scrolled up ("waiting for the composer to
be in view", `wrapcmd/peer_delivery.go`), and it already holds while the composer has a draft in it ("waiting for
empty composer"). Both holds are caused by the operator, and both spend the deadline. On 10-10 (pair#428 live check
on ariadne:2) a test message expired after about 90 s while held for scrollback. The TL slot is where the operator
reads scrollback the longest, so worker reports to the TL are what expire, and a sender learns of it only by
polling the receipt (ops learnings kinks 56, 58). It hurts the TL's effectiveness (operator, 10-10).

## Spec

- **Operator-caused holds stop the clock.** While the wrapper holds only because the composer is out of view or
  holds a draft, the deadline doesn't advance; it resumes when the hold clears. Holds the operator didn't cause
  (paste mode not ready, unrecognized composer, a booting slot) keep today's deadline.
- **A cap still bounds the receipt.** A held message stays pending at most `ReceiptLifetime` (or a stated cap), so
  a pane left scrolled up overnight doesn't keep a sender's message forever; on reaching the cap the status names
  the hold ("expired: held for scrollback").
- **The sender can see the hold.** The receipt reports `held` with the reason, so a TL polling a send sees "held
  for scrollback" rather than "delivering".
- The broker and the wrapper agree on the extended horizon (today `Horizon()` = deadline + `DeliveryTimeout`).

## Done when

- [ ] A message held for scrollback past its original deadline submits when the composer comes back into view (model and wrapper tests).
- [ ] The same for a held draft once the composer empties.
- [ ] A hold not caused by the operator still expires at the deadline (test).
- [ ] The cap expires a long hold with a status that names it; the receipt shows `held` and the reason while held (test).

## Plan

- [ ]

## Log

### 2026-10-10
