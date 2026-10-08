---
id: 000407
status: open
deps: [pair#395]
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: 'e3020de52510d5981df3877f97b836c16fbe282e' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch web front end: a browser as a pluggable Couch parent terminal (remote control)

## Problem

#395 makes a browser a view-only *output* parent for Couch: the hub taps
the Presenter's composed frames and an xterm.js page renders them. The operator's
framing (2026-10-07): a Couch web front end is just a remote-controlled
Couch. Adding the input half gives remote help (customer service, debugging
someone's setup) and, eventually, a browser as Couch's primary front end.

Overlaps #121 (remote Pair control relay), which has been claimed since
2026-07-26. Reconcile with it before designing.

## Spec

Open questions raised in #395's discussion:

- **Separate endpoint and grant.** Control runs on its own URL (WebSocket), with
  its own separately issued grant and explicit approval from the operator. A #395
  view-only link can never be upgraded to control. A capability link alone is too
  weak, because control means running code as the operator; it needs SSO or
  Cloudflare Access.
- **Geometry.** Couch has one host size. A helper takes the operator's grid
  (auto-fit font, as in #395). A browser that is Couch's primary front end would
  own the size, so the host becomes pluggable rather than a second viewer.
- **Keys.** Couch relies on enhanced keyboard reporting (Alt+d, Ctrl+Return;
  #279). Browsers capture some chords, and xterm.js has limited support for kitty
  keyboard reporting. Measure this before promising parity.
- Reuse the #395 hub. A control transport adds only the input direction.

## Done when

- A remote helper granted control (a separate grant from any view-only link,
  with the operator's explicit approval) can type into the operator's Couch
  from a browser, and the operator can revoke it at once.
- A #395 view-only link can never send input, shown by a test.
- Control requires authentication stronger than a capability link (SSO or
  Cloudflare Access), shown by a test of the refusal without it.

## Plan

- [ ] Reconcile with #121 (remote Pair control relay) and settle the trust
      model: grant, approval, revocation, authentication
- [ ] Measure browser keyboard fidelity for Couch's enhanced chords (#279)
- [ ] Design the control transport (WebSocket on its own URL) over the #395 hub

## Log

### 2026-10-07

- Filed from #395's design discussion.
