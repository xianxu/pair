---
id: 000467
status: working
deps: [000466, 000458, 000460]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '56b70ba3c86c078b5824fe3a98694b860f93723d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T06:23:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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
- The operator smoke-tested it with a second device before close.
- Tests: `RunInjectorContract` over the real `TabInjector` rules; a live headless-Chrome injection test (attribution, file chooser, download, new target, stale, refused); the drive routes; the viewer's drive module (unit + live, mouse and touch); Couch's span and menu; the recorded `driven` fixture passes the coverage checks.

## Plan

- [ ] M1 — the Go core: `Tab`/`TabInjector`, `MemController`, observer hold/take + `by`, `Recorder.WithTab` (CDP), `Chrome.SetDriving` guards + banner, drive routes + Attach + Cast wiring
- [ ] M2 — the viewer shell: `drive.js` (seat, ask/give back, pointer/touch/keyboard capture, batches), notes authored by the seat
- [ ] M3 — Couch: drive span after `Chrome ⏸`, drive menu (grant/deny/revoke), Ctrl+click revoke
- [ ] M4 — a real recorded `driven` fixture replacing the hand-written one; consumers derive from data; atlas

## Log

### 2026-10-12

- Claimed at pair:6 on the TL's dispatch (ops:0, receipt 7dd881f0 verified). Read #466's atlas sections plus the recorder, Chrome loop, server, viewer and Couch seams. No real Controller or Injector existed; the recorder never stamped `by`; `Presence` was unused in production.
- Attribution design: hold/take the observer's queue inside the recorder-queue input that dispatches, not an arrival-time window (which would race a revoke and get a real click refused by the reducer).
