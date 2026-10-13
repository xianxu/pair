---
id: 000506
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 3.79
card_mirror: '74abdbdc1afc1b9b237d0aef63c66a89d1b5a10b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:49:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 4.06
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* The source calibration is marked stale, so this is a provisional ship-wall-clock estimate, derived after plan-quality cleared.

Conventions follow #306's estimate:
- A thorough plan exists, so design takes ×0.2 of the v2 table, and the design buffer is 15%.
- Implementation is v3.1's ×0.4 of the v2 table.
- Familiarity is 1.0: the web cast's server, board and viewer were read for this plan, and the patterns (token classing, admission, live headless tests) exist.
- No library applies.

The rows:
- The issue/spec dialogue (short, already settled with ops:0) is 0.5/0.08.
- Each of the seven focused changes is a smaller module, 0.3/0.5 before scaling: castLink+linkOf, board refusal, palette answer, cast pointer token, viewer read-only mount, live view-only test, Couch copies.
- 👆/👽 may need new spans if #501 lacks them, so it is a TUI piece, 1.0/1.0 before scaling.
- Docs are 0.2/0.2.
- Two milestone reviews are 0.04/0.20 each.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.50 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 3.79
```

## Plan

Durable plan: `workshop/plans/000506-broadcast-web-three-links-like-the-terminal-cast-view-only-right-click-pointer-drive-plan.md`.

- [x] M1 — Cast side: three tokens, the link kind of a request, Board refusal under the view token, palette.json with annotate, the viewer's read-only mount (moved from M2, see the plan's Revisions).
- [x] M2 — Couch: the copy commands and the 👆/👽 mapping (after #501 lands); docs. (The operator smoke test is a Done-when item, checked at the issue close after #495 lands.)

## Log

### 2026-10-12
- 2026-10-12: closed — operator waived smoke; pair#523 follows. Full make -k test on ddb76d210 (branch + origin/main with #495, #501): exit 0, 103 packages ok, no FAIL. Close review round fixes (84a3d0cc1): README cast-row table and web cast comments name the pointer link (git grep for view link/viewer link clean outside the terminal cast), Update.LinkFor is the one kind-to-URL map (TestUpdateLinkFor) used by the copy table and both unlock helpers, lessons section moved; couchtty, webcast, couchmessage green after. Server: TestBoardRoutesRefuseTheViewOnlyLink, TestPaletteSaysWhetherTheLinkAnnotates, TestCastPublishesThreeLinks (each link its own lock and code). Viewer: readPalette node table; live Chrome view-only viewer. Couch: TestWebcastCopiesEachLink, TestWebcastLinkButtons, TestWebCastCodesOnTheChromeCell.; review verdict: FIX-THEN-SHIP
- 2026-10-12: closed M2 — Full make -k test on 86bd312c9: 101 packages ok, the one failure (artifactpath inventory missing webcast/link.go) fixed in b41e45347. Review round 1 fixes in this commit: atlas/README swept for the old two-link wording (git grep clean), copy commands folded into copyWebcastLinkOf, notices say view-only link; couchtty green after. TestWebcastCopiesEachLink and TestWebcastLinkButtons (recorded cast: 👆 copies the pointer link, not the view-only one). 99 node tests and live Chrome tests green. Operator smoke test pending (after #495 lands).; review verdict: SHIP
- 2026-10-12: closed M1 — webcast (all subpackages), couchtty, couchcmd green; 99 webcast node tests green; live TestAnnotateLiveTwoViewers green under WEBCAST_LIVE_CHROME=1 with a view-only viewer C that sees Bs note read-only, has no rail, no Resolve, and gets 403 This link is view-only. Five mutations each fail a test (board refusal, palette annotate flag unit+live, pointer stream counted as view, Resolve hidden, rail hidden). Couchs right-click notice still describes the old link; M2 relabels it.; review verdict: SHIP
- Plan sent to ops:0 before building; approved as written, with point 3 (view-only viewers see others' strokes and notes read-only) confirmed by the TL, operator may veto.
- M1 built: castLink + linkOf, board refusal before admission (403 "This link is view-only."), palette.json {palette, annotate}, the cast's pointer token and Update.PointerLink, the viewer's readPalette and read-only mount, live view-only viewer C. Viewer work moved into M1: the palette shape change broke every viewer's rail until the viewer read it (live test). Webcast package green; 99 node tests green; live annotate test green with WEBCAST_LIVE_CHROME=1. Five mutations each caught.
- M2 built: Couch copy commands (right-click view-only, `copyWebcastPointerLink`), merged #501 and re-pointed its 👆 to the pointer link (span renamed `WebPointerLink`), README and atlas. Full make -k test on 86bd312c9: 101 packages ok; the one failure was this branch's (`webcast/link.go` missing from the artifactpath inventory), fixed in b41e45347.
