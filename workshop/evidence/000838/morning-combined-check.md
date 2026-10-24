# Combined morning operator check — 2026-10-24

The operator restarts Couch once from the verified pair:0 build. Pair:3 reports
its exact staged revision and binary provenance separately. Parley #419 is
staged on parley.nvim:0 for the same check. Use disposable documents and cast
feedback. Refresh the worker's Couch skill from the staged candidate before
editorial submissions; existing sessions can retain earlier instructions.

## Candidate inputs

| Input | Qualified revision |
|---|---|
| main, including #856 / PR583 | 7cb0b672ebb6aca081ae63f19c27b0aafc45270f |
| #838 / PR584 | 11ebba30e8e9d15e399f7c2f26d0ac0ad5858f16, full suite PASS13m57s; later evidence is prose only |
| #851 / PR577 | d74a152d8b18978dbcda2c9961067678f5854069, source unchanged since recorded PASS |
| #842 / PR581 | 7b76b99218dccced46a984e41f8bea00d6681af8, source unchanged since recorded PASS |

#857 is excluded: its owner has not supplied green readiness. This check does
not wait for it. The combined merge is staging only; each issue lands separately.

## Switcher — #851

Open the switcher. Press another visible row once: highlight changes, the panel
stays open and no switch happens. Press the selected row: it switches. A rapid
double press switches exactly once; a delayed second press also switches.
Open-panel chips follow the same selection rule; a chip hidden by a filter first
reveals/selects. Outside-panel chips switch on the first press. +N opens its
group. Keyboard arrows and Enter retain their usual behavior.

## Notes panel — #856 on main

In the Chrome cast, check one Notes panel, icon rows and hover glow. Try Show on
page, including a note whose placement is unavailable: the unavailable state is
explicit. Pen colors remain visible before drawing (the approved choice).
Undo, Redo and Done appear when useful.

## Send control — #842

Create a note or finished labelled sketch and a draft. Expect status-bar Send,
even when the preview is stale. Click from agent focus and right-terminal focus:
each click produces one equivalent submission, and the selected feedback
disappears after dispatch. With no eligible feedback, or terminal-only notes,
there is no Send. The Couch LIVE and Chrome bar groups use the existing chord;
there is no browser-column button.

## Collaborative editing — #838 with Parley #419

Follow [the four editorial steps](operator-check.md): a clear command produces
clean changed text with a visible highlight; a question produces a linked side
document; tagged numbered chat asks keep their source and receive answer links;
historical direct/propose rounds keep their rendering. Check the retired-key
notice once and confirm no mode settings are inserted. Use the Couch workflow
for side-document semantics. The native synthetic runs qualify mechanics;
this operator pass qualifies actual agent judgment and Parley rendering.

Report PASS or the first mismatch for each issue, naming the combined revision.
After all checks, pair:3 clears the combined stage through SDLC, coordinating with
the hub and issue owners. Preserve the stage ref until that clear; do not land
the staging-only merge or clear another owner's lease by hand.
