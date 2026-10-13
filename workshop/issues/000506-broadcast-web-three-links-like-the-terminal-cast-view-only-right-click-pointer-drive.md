---
id: 000506
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'f77e917c373ea2ea64b31543b5f123016c262a9f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:49:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# broadcast-web: three links like the terminal cast — view-only (right-click), pointer (👆), drive (👽)

## Problem

The web cast has one viewer link that grants watching, pointing and notes, plus the drive link (#467). The terminal cast has three levels: a view-only link (right-click on LIVE), a pointer link (👆) and, later, remote control. The operator wants the web cast to mirror that, so a link can be handed out for watching alone. This revises the 10-12 morning decision of "one link for watching, pointing and notes".

## Spec

- **Three tokens per web cast:** view, pointer and drive. Each link grants a superset of the one before: view watches; pointer watches, points, leaves and resolves notes (today's view link); drive adds a seat to ask for driving (unchanged, #467).
- **The server enforces it.** Under the view token, the Board routes (stroke, note, note-state) answer 403 "This link is view-only." before the body is read or anything reaches the Board. The drive routes stay drive-only.
- **The viewer follows the server.** `palette.json` answers `{"palette":[…],"annotate":bool}`, with annotate false under the view token. A view-only viewer still draws other people's strokes, note pins and note text read-only, like the terminal's view link, but shows no swatches, no ✏️/💭 mode buttons and no Resolve. (TL 10-12: approved; the operator may veto read-only notes.)
- **Couch:** right-click on the Chrome cell copies the view-only link; 👆 copies the pointer link; 👽 (and Alt+right-click) copies the drive link. The 👆/👽 web-cast controls come from #501 (pair:1); this issue points them at the right links after #501 lands, merging main first (no stacking, #272). If #501 lands without them, this issue adds them after the Chrome cell.
- **#495 (pair:4)** gives every link its own code; its codes map one to one onto these three tokens. Whichever of #495 and #506 lands second wires the pointer link's code.

## Done when

- A view-only link's viewer watches and sees others' strokes and notes, but cannot draw, leave or resolve a note: the server refuses (403, nothing reaches the Board) and the page shows none of those controls.
- The pointer link behaves as today's view link; the drive link is unchanged.
- Right-click, 👆 and 👽 in Couch copy the view-only, pointer and drive links, each with a notice that says what it grants.
- Tests: a route table per token, the palette answer per token, the viewer's pure palette reading (node), the live two-viewer test with a view-only viewer, and Couch's copy per control. Full suite green.
- The operator smoke-tested it.

## Plan

Durable plan: `workshop/plans/000506-broadcast-web-three-links-like-the-terminal-cast-view-only-right-click-pointer-drive-plan.md`.

- [ ] M1 — Server: three tokens, the link kind of a request, Board refusal under the view token, palette.json with annotate.
- [ ] M2 — Viewer and Couch: read-only annotate for view-only; Couch's copy commands and the 👆/👽 mapping (after #501 lands); docs; operator smoke test.

## Log

### 2026-10-12
- Plan sent to ops:0 before building; approved as written, with point 3 (view-only viewers see others' strokes and notes read-only) confirmed by the TL, operator may veto.
