---
id: 000523
status: open
deps: [pair#506]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '6879f93e43d0767fa60138c4c480143fb782d738' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast: 👆 and 👽 are capability switches on both casts (👆 off: no one annotates; 👽 off: no one drives from outside)

## Problem

The cast row's emoji mean different things on the two casts. On the terminal cast, left-clicking `👆` toggles pointing (first click makes and copies the pointer link; off clears marks and leaves the link view-only; README, #412). On the web cast (#506), left-clicking `👆` or `👽` only copies a link. The operator's click rule (#495) says left click acts or toggles, right-click copies.

## Spec

The operator (2026-10-12): `👆` means "marking-up, here" on both casts, and the emoji are capability switches:
- **`👆` off: no one can annotate.** Left click toggles it (amber when on). On the web cast, off means pointer-link viewers become view-only (the Board refuses their strokes and notes, as for a view-only link) and the operator's cast-Chrome overlay (#494) is unavailable; existing marks clear as on the terminal. The first click makes the pointer link and copies it, as the terminal does.
- **`👽` off: no one can take control from outside.** Left click toggles it. Off means the drive link can't ask to drive; if someone is driving when it turns off, control returns to the operator.
- **Right-click copies** that control's link on both casts, whatever the state. Hover and Alt+click on `LIVE ⏸` / `Chrome ⏸` stay as #495 made them.
- The terminal cast already behaves this way for `👆`; keep it, and share one rule in code and README.
- Server-side enforcement for both switches (never only hidden UI); viewers see a clear state ("annotation is off", "driving is off").

## Done when

- On the web cast, `👆` off makes pointer-link viewers view-only and disables the operator overlay; `👽` off refuses drive requests and ends an active drive; right-click copies on every control; the terminal `👆` is unchanged; tests cover the server refusals and the clicks; the operator approves on pair:0.

## Plan

- [ ]

## Log

### 2026-10-12
