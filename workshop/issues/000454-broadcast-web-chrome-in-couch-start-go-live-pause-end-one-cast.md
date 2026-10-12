---
id: 000454
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '3ed97d5e7e4da5577a3c48febbb95351f5c3b562' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: Chrome in Couch — start, go live, pause, end; one cast

## Problem

couch-broadcast-web (project: ops `teams/devinfra/projects/couch-broadcast-web/project.md`, read its Operator flow and Principles first) casts a Chrome tab from Couch as a DOM stream. Before the recorder exists (its base is being decided in pair#453), Couch needs the operator's controls: start a dedicated Chrome, hold a CDP connection, go live, pause, resume, end, and show the state in the tab bar. This is the first piece of the product and sets its component boundaries.

## Spec

The operator flow, from the project:

1. **Start:** Shift+Ctrl+Alt+B with no Chrome running starts Chrome with a dedicated profile (its own `--user-data-dir` under pair's data dir), a normal window with its address bar, on a blank start page, with a CDP endpoint Couch connects to. One cast Chrome at a time.
2. **Go live:** a "Go live" button injected through CDP into every page in an isolated world (never the page's own script world), or from Couch. A blank page can go live.
3. **Indicator:** while Chrome runs, Couch's tab bar shows `Chrome ▶` (not live) or `Chrome ⏸` (live).
4. **Controls:** click, or Shift+Ctrl+Alt+B while Chrome runs: go live / pause / resume (the browser stays open). A stop control inside the page also pauses. Right-click: copy the public viewer URL. Ctrl+click: end the cast and close Chrome. Chrome closed by the operator: Couch notices and clears the indicator.
5. **Active tab:** the cast follows the active tab of that Chrome. CDP has no tab-activated event; find it from page visibility and window focus (with two windows, the last focused). New tabs opened later are attached as they appear.
6. **Stub stream** until the recorder lands: live/paused state, the active tab's URL and title, and the viewer URL served over the broadcast's transport, showing just that. The recorder (pair#453, then the Recorder task) plugs into the same seam.

**Boundaries (the operator's direction for web work: service-oriented, components with explicit seams):** a browser-control component (Chrome process, CDP connection, tabs, active tab, go-live state, events) with a small interface the recorder and Couch's UI use; Couch's UI only talks to that interface. Keep CDP types from leaking past it. Test it against a fake of that interface, and run a live conformance check against a real Chrome where practical.

**Check first:** whether Shift+Ctrl+Alt+B is distinguishable from Ctrl+Alt+b (screen broadcast) in Couch's key decoding for the operator's terminal; legacy encodings send the same bytes. If it isn't, stop and ask the operator for the key.

## Done when

- Shift+Ctrl+Alt+B starts the dedicated Chrome; a second press goes live; the tab bar shows `Chrome ⏸`; click pauses (`Chrome ▶`) and resumes; right-click copies the viewer URL; Ctrl+click ends the cast and closes Chrome; closing Chrome by hand clears the indicator.
- The viewer URL shows the live/paused state and the active tab's URL and title, updating when the operator navigates or switches tabs (including a tab opened after start).
- The browser-control component has a fake and tests; a live check drives a real Chrome.
- The operator smoke-tested it in Couch before close.

## Plan

- [ ] Key decoding check (Shift+Ctrl+Alt+B vs Ctrl+Alt+b)
- [ ] Browser-control component: launch, CDP connect, tab tracking, active tab, go-live state, events; fake + tests
- [ ] Injected Go-live and in-page pause controls (isolated world)
- [ ] Tab-bar indicator and its click / right-click / Ctrl+click
- [ ] Stub stream and viewer page over the broadcast transport
- [ ] Live check against real Chrome; operator smoke test

## Log

### 2026-10-11
