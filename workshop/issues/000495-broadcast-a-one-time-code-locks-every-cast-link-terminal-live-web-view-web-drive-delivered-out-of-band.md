---
id: 000495
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 8.05
card_mirror: 'acbec659f5f9cb872e330c525f71f1813476cb1d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T12:59:16-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band

## Problem

A cast link (the terminal broadcast's LIVE link, the web cast's view link, its drive link) is a bearer secret in a URL: it ends up in chat logs and browser history, and chat apps fetch it to build link previews. Anyone holding the link sees the cast (and, with the drive link, can ask for control of the operator's browser). The operator (10-12) wants a one-time code delivered out of band (said aloud on the call) as a second factor.

## Spec

Decided with the operator, 10-12:
1. **Locked first:** a cast link opens a lock screen asking for the code; it combines the code with the name prompt #470's viewer already shows on a viewer's first note, asking both at once.
2. **The server enforces it:** entering the right code issues a session token; without it the server sends nothing (no record or event stream, no assets, no note/stroke/drive routes). The lock screen is never just a cover over streaming content.
3. **`URL#NNNN`:** the code may ride in the URL fragment for convenience (browsers never send fragments; link-preview bots don't run the page's script); the page reads it and submits it like typed entry.
4. **Guessing:** one budget per code (about 8 wrong tries across all clients; a client can't be identified behind the tunnel), then the code rotates and Couch alerts; existing sessions survive a guess-rotation. Unlock attempts are rate-limited per lock. 4 digits for view and pointer links, 6 for the drive link.
5. **Separate codes per link:** view and drive codes differ; the view code never grants driving. The terminal LIVE link has its own.
6. **Rotation:** each new cast gets random codes. Rotating a code from Couch's Codes menu (Alt+click on `LIVE ⏸` / `Chrome ⏸`: a link row's `rotate`, or `rotate all`) cuts off everyone holding the old one (sessions revoked). The operator announces the new one out of band.
7. **Display:** no codes inline in the tab bar. Hovering a cast cell shows its codes as a tooltip (`view 4821 · pointer 7730` on LIVE, `view 5512 · drive 903114` on Chrome); Alt+click on the cell opens a Codes menu (one rotate item per link, plus rotate all); on rotation the new code flashes three times. All of these draw in an operator-only presenter layer that the LIVE broadcast never carries. One lock page asks for the code and the viewer's name for every link; terminal notes gain the author.
8. Atlas: the threat model in one paragraph (what the code protects against: leaked links, previews, history; what it doesn't: a viewer who already has the code).

## Done when

- Every cast link (terminal LIVE, terminal pointer, web view, web drive) is locked until the right code; the server refuses streams and routes without a session; the per-code wrong-try budget rotates with a Couch alert; codes are separate per link; rotating from the Codes menu disconnects that link's viewers; codes show only in the operator-only layer (tooltip, menu, flash), never in the broadcast frame; the lock page's name attributes notes on every link. Replay links are a filed follow-up.
- Tests for each guard (including a direct stream request without a session); the operator smoke-tested it on desktop and iPad.

## Plan

Durable plan: `workshop/plans/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band-plan.md` (decisions D1–D6 and their answers are under Revisions).

- [x] M1 — `castlock` core + Host gate (session seam) + lock page (code + name) + unlock route; terminal LIVE and pointer links locked; revoke ends live streams; every route refuses without a session (direct `GET events` included).
- [x] M2 — web cast view and drive locks (6-digit drive); lock-page name replaces #470's note-dialog prompt; terminal notes gain the author; headless-Chrome tests unlock first.
- [x] M3 — Couch: operator-only presenter layer (never tapped); hover tooltip with codes, Alt+click Codes menu with per-link rotate and "rotate all", guess-rotation alert, the three-flash on rotation — all in that layer; atlas threat model; follow-up issue for replay links; operator smoke test on desktop and iPad.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* The calibration doc is marked stale (#127), so the numbers are provisional.

- **castlock core:** a greenfield Go module.
- **Host gate:** lock page, unlock route and cookie; a greenfield Go module plus JS.
- **Terminal LIVE and pointer wiring:** a smaller Go module.
- **Web cast view and drive wiring, plus the name merge:** a smaller Go module.
- **Operator-only presenter layer:** a greenfield Go module.
- **Couch tooltip, Codes menu, flash and alert:** a TUI screen. It mirrors the notes menu, so design sits mid-range.
- **Atlas threat model.**
- **Reviews:** three milestone reviews.
- **Smoke-test iteration:** one UX round.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=1.0 impl=0.22
item: greenfield-go-module   design=1.0 impl=0.24
item: smaller-go-module      design=0.2 impl=0.16
item: smaller-go-module      design=0.3 impl=0.2
item: greenfield-go-module   design=0.8 impl=0.2
item: tui-screen             design=1.2 impl=0.28
item: atlas-docs             design=0.1 impl=0.04
item: milestone-review       design=0.1 impl=0.14
item: milestone-review       design=0.1 impl=0.14
item: milestone-review       design=0.1 impl=0.14
item: ux-rename-iteration    design=0.5 impl=0.08
design-buffer: 0.15
total: 8.05
```

## Log

### 2026-10-12
- 2026-10-12: closed M3 — BR-16 fixed (rotations bound to the cast's start attempt; TestLateRotationFromAnEndedCastIsDropped). BR-17: README section with per-control table. BR-18: operator smoke test on desktop and iPad PASSED ('tested the code, it's working well'), re-test of the ergonomics revision and menu PASSED ('looks good for those changes'); both recorded in the Log. Revisions folded in: the cast row's rule (hover and Alt+click on LIVE/Chrome cells, right-click copies), Codes menu as a pure model with per-word click targets and keyboard, popup stays open with a press flash and in-place rotation. make -k test exit 0 on the merged head 3bf5f83b under the suite lock.; review verdict: FIX-THEN-SHIP
- 2026-10-12: closed M2 — BR-10: node tests for the web viewer's relock on a lost session (relock, disconnected when the session holds, no probe after end or while reconnecting); revert of the branch fails the relock test. BR-1: plan summary current (three web links, LinkLocks registry, tooltip anchors, live-test substitution). make -k test exit 0 under the suite lock before these test-and-plan-only changes.; review verdict: FIX-THEN-SHIP
- 2026-10-12: closed M1 — Round 1 fixes: BR-4 Rotation.Gen + TestConcurrentRotationsNewestGenIsCurrentCode; BR-3 plan records D1 Strict and revocation as built; fail-closed spent code (TestSpentCodeThatCannotRotateFailsClosed), Failed outcome; FuzzUnlockBody (525k execs); castClient replaces DefaultClient mutation. castlock+broadcast -race green; couchtty cast tests green. Earlier: make -k test green except artifact inventory, fixed.; review verdict: FIX-THEN-SHIP
- D4 follow-up filed by the hub from pair:0 as pair#509 (replay share links; unclaimed).
- Operator smoke test (desktop and iPad), build bd801297: the codes passed ("tested the code, it's working well"). The ergonomics revision (the cast row's rule: hover and Alt+click on LIVE ⏸ / Chrome ⏸, right-click copies, blue copy actions in the Codes menu, no Alt+right-click on Chrome) is recorded in the plan's Revisions and built before landing. The terminal pause is filed as #518 (unassigned).
- Codes menu reorganized per the operator (one row per link: label, then `rotate · copy-link` as blue words; keyboard Up/Down/Left/Right/Enter/Escape).
  - **Decision: a link that doesn't exist yet** (the screen pointer link before pointing first starts) shows its label alone, `pointer (not made yet)`, with no actions; it does not get a `copy-link` that creates it.
  - **Why:** making the pointer link turns pointing on, which changes what LIVE viewers get. A copy should never have that side effect, and 👆 already does exactly that on purpose.
- Found while building: the menu's first cell writer counted runes as cells, so an emoji in a label made an invalid frame and the menu never drew. Fixed by laying out and drawing by grapheme width; pinned by `TestCodesMenuDrawsWideCharacters`.
- Operator re-test (build f7e6cc7a): passed for the menu layout, click rules and hover ("looks good for those changes").
- Last tweak, built before landing: actions in the Codes popup keep it open (Escape or × closes it). The pressed word draws inverted for 180 ms, and a rotation refreshes the menu in place, keeping the selection. Pinned by `TestCodesMenuStaysOpenAndShowsThePress`.

