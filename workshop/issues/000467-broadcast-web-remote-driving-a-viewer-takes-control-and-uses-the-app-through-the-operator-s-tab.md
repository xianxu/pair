---
id: 000467
status: working
deps: [000466, 000458, 000460]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 6.17
card_mirror: 'b3ce5b18eaf59c4cc8243ceca5f562446cfc5a20' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T06:23:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab

## Problem

Build remote driving for internal viewers (the operator, 10-12; MVP), against the remote-driving contract filed alongside this issue.

## Spec

- Viewer: request control; while driving, capture pointer and keyboard on the scaled, script-free frame, map them back to element ids and page coordinates, and send them over the reverse channel; give control back.
- Couch: grant/revoke from the `Chrome ⏸` cell (and its menu); show who is driving in the tab bar and in the page; inject input through the browser-control seam into the cast tab only.
- Recorder: actions recorded with their source; control changes recorded.
- Works from a desktop browser and an iPad (touch maps to pointer).
- While a viewer drives: file choosers are cancelled, downloads are denied, and new targets the cast tab opens are closed. Revoke works from any state. (TL, 10-12)
- A real recorded driven session replaces the hand-written `driven` fixture. (TL, 10-12)
- Design, decisions 1–7 and milestones: `workshop/plans/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab-plan.md`.

## Done when

- A second person (desktop Chrome) requests control, the operator grants it, they click, type and scroll in the blog through the cast, then give control back; the operator can revoke at any time; nothing outside the page is reachable.
- The recording attributes their actions to them; the agent queries show it.
- The operator's two-device smoke test runs after landing, in the session the TL batches; anything it finds becomes a follow-up (see Revisions).
- Couch tells the operator that uploads and downloads are off while a viewer drives; a JS dialog gives the driver a fixed sentence, not a 500.
- Tests: `RunInjectorContract` over the real `TabInjector` rules; a live headless-Chrome injection test (attribution, file chooser, download, new target, stale, refused); the drive routes; the viewer's drive module (unit + live, mouse and touch); Couch's span and menu; the recorded `driven` fixture passes the coverage checks.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.04 impl=0.2
item: smaller-go-module      design=0.02 impl=0.1
item: smaller-go-module      design=0.06 impl=0.25
item: greenfield-go-module   design=0.1 impl=0.35
item: smaller-go-module      design=0.04 impl=0.25
item: greenfield-go-module   design=0.1 impl=0.3
item: greenfield-go-module   design=0.1 impl=0.32
item: smaller-go-module      design=0.06 impl=0.25
item: tui-screen             design=0.2 impl=0.45
item: tui-screen             design=0.15 impl=0.4
item: cross-cutting-refactor design=0.1 impl=0.4
item: atlas-docs             design=0.1 impl=0.08
item: milestone-review       design=0.0 impl=0.4
item: milestone-review       design=0.0 impl=0.4
item: milestone-review       design=0.0 impl=0.4
item: milestone-review       design=0.0 impl=0.4
design-buffer: 0.15
total: 6.17
```

Items, in order: `Tab`/`TabInjector` + fake tab; `MemController` rename; observer hold/take + `Observe(by)`;
`Recorder.WithTab` + `cdpAct`; live headless injection test; `Chrome.SetDriving` guards, dialog signal, banner; drive
routes + Attach + Cast wiring; `drive.js` pure parts; shell mount + live test (mouse, touch, keyboard); Couch span +
menu + notice; recorded `driven` fixture + consumers deriving from data; atlas; four milestone reviews.

## Plan

- [x] M1 — the Go core: `Tab`/`TabInjector`, `MemController`, observer hold/take + `by`, `Recorder.WithTab` (CDP), `Chrome.SetDriving` guards + banner, drive routes + Attach + Cast wiring
- [x] M2 — the viewer shell: `drive.js` (seat, ask/give back, pointer/touch/keyboard capture, batches), notes authored by the seat
- [x] M3 — Couch: drive span after `Chrome ⏸`, drive menu (grant/deny/revoke), Ctrl+click revoke
- [ ] M4 — a real recorded `driven` fixture replacing the hand-written one; consumers derive from data; atlas

## Log

### 2026-10-12
- 2026-10-12: closed M3 — couchtty TestDriveFromCouch/TestWebcast -race green; copyWebcastDriveLink takes one snapshot (BR-29); all prior M3 evidence stands; review verdict: SHIP
- 2026-10-12: closed M2 — node tests green (drive.js 16); go test -race ./cmd/internal/webcast/... green incl. release tests (revoke, transfer, refusal, page change, late release names its hold, auto-repeat once); WEBCAST_LIVE_CHROME=1 DriveLive/LiveDrive/AnnotateLive green; atlas: letting go + per-act segments; review verdict: SHIP
- 2026-10-12: closed M1 — go test ./cmd/internal/webcast/... -race green; WEBCAST_LIVE_CHROME=1 full webcast live suite green (135s); TestLiveDrive 3/3 real headless Chrome incl. intercepted file chooser via event tap, download deny vs positive control, restore; TestChromeCastTabDialog for BR-1; FuzzDriveInput 20s; review verdict: SHIP

- Claimed at pair:6 on the TL's dispatch (ops:0, receipt 7dd881f0 verified). Read #466's atlas sections plus the recorder, Chrome loop, server, viewer and Couch seams. No real Controller or Injector existed; the recorder never stamped `by`; `Presence` was unused in production.
- Attribution design: hold/take the observer's queue inside the recorder-queue input that dispatches, not an arrival-time window (which would race a revoke and get a real click refused by the reducer).
- TL approved decisions 1–7 (ops:0, 2efb3d52). It added two asks: a Couch notice and menu line that uploads and downloads are off while someone drives, and a fixed dialog sentence. The smoke test moves after landing.
- M1 built: `drive.Tab`/`TabInjector` (the rules once, fake and real), `MemController` (renamed fake), observer hold/take + `Observe(by)`, `Recorder.WithTab`/`Opened`/dialog signal, `cdpAct`, `Chrome.SetDriving` (downloads deny, file-chooser intercept on every page session, opener-matched new targets closed before the tracker sees them, banner), drive routes + `admitBody` shared with the board, `liveGate`, `Cast.Drive()`.
- Deviation from plan: the file-chooser intercept goes on every page session (and every page attached while driving), not just the cast tab — decision 5's "one rule"; the cast tab is the only one input reaches anyway. `castTab` reads the tracker's active tab while live rather than `recTab` (which is only set with a recorder).
- Live (real headless Chrome) caught a bug the fakes couldn't: `DOM.getNodeForLocation` with `includeUserAgentShadowDOM:true` returns an `<input>`'s inner editor, which the recording doesn't have, so every click on an input read as stale. Now false.
- Live confirms the dialog residual: an alert opened by the driver's click returns ErrDialog in < 3 s; the click itself is released by the expired hold after the operator answers, without `by` (documented class). Download deny: the request reaches the server once, and the transfer is denied by the CDP setting (asserted in the loop test; no download dir to read in headless).
- Verified: `go test ./cmd/internal/webcast/... -race` green; `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast` green (135 s, all live suites); `TestLiveDrive` 3/3; FuzzDriveInput 20 s.
- M2 built: `drive.js` (`joinSeat`, `seatPoster`, `batcher`, `gestures`, `keyAct`, `driveView`, `mountDrive`), `drive.css`; `viewer.js` takes the seat before the stream (the stream holds it) and mounts annotate and drive after; `identityStore.adopt` makes notes authored by the seat; the in-page control's banner names the driver (inject test). Verified: node tests all green (drive 10, inject 8, …); `TestDriveLiveTwoViewers` 5/5 (mouse click/typing/Enter, touch tap, grant transfer, revoke, give back); full live suite green (224 s).
- M3 built: the drive span (🕹/✋) after Chrome ⏸ 💭N, the drive menu as the notes menu's third source ("Remote driving": left on driver = revoke, left on asker = grant, right on asker = deny), Ctrl+click span = revoke in any state, notices (asks, grant turns uploads/downloads off, control back); `webcast.FakeDriven` exported as Driven's fake (replaces the cast test's private one). Test races fixed in the test, not the product: a click must wait for Couch's delivered control and the drawn menu to match (lesson #473 shape). Verified: TestDriveFromCouch 40/40 and 20/20 under -race with neighbours; couchtty, couchcmd, webcast suites green unsandboxed.
- M3 review round 1 + two links (operator via TL, ddc0fc5a): drive token and `Update.DriveLink`; drive routes 404 under the view token; Alt+right-click copies the drive link (TL told: Shift+click is the terminal's selection); `--broadcast-list` lists shares by kind; `driveFailure` classes control failures; FakeDriven `FailEmits`. A 1-in-25 hang was a lock-order inversion in the test (pnotes.mu is a leaf). Verified: TestDriveFromCouch 25/25 -race; couchtty, couchcmd, couchmessage, webcast/..., broadcast/... green unsandboxed; live drive/annotate/viewer suites 3/3 (one unexplained live failure while broadcast's suite ran alongside, not reproduced in 3 runs).
- M4 built: `rec-driven` recorded through the production pieces in headless Chrome and adopted (the hand-written `driven` removed with `fixturegen -driven`); a driver note via fixturegen's spec (author v1). The recording found the navigating-click gap; the held observer now tags and sends in order (TestLiveDriveNavigation). Agent queries show attribution: `2188-2493 input @12 <input#q> value "invoice 42" by Ana (v1)`. Consumers derive from the fixture's records.

## Revisions

- 2026-10-12 (TL, ops:0, 2efb3d52): Done when's "the operator smoke-tested it with a second device before close" becomes "after landing, in the TL's batched session; findings become follow-ups". Close and merge on SHIP after M4.
- 2026-10-12 (operator via TL, ops:0, ddc0fc5a): two links. The view link watches, points and notes, with no seat and no request button. A separate drive link (its own token) offers the seat; the drive routes accept only its token. Couch copies the drive link with Alt+right-click on the Chrome cell, and `couch --broadcast-list` shows each share's kind.

