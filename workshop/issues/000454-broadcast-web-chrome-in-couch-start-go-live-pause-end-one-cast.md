---
id: 000454
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours: 4.69
card_mirror: '8036005b121ecb59e324522b9cbfc82711814909' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T20:08:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.1 impl=0.24
item: greenfield-go-module   design=0.1 impl=0.2
item: greenfield-go-module   design=0.2 impl=0.32
item: real-api-discovery     design=0.0 impl=0.24
item: cross-cutting-refactor design=0.2 impl=0.3
item: smaller-go-module      design=0.04 impl=0.2
item: smaller-go-module      design=0.06 impl=0.2
item: smaller-go-module      design=0.06 impl=0.2
item: tui-screen             design=0.2 impl=0.4
item: atlas-docs             design=0.1 impl=0.08
item: milestone-review       design=0.0 impl=0.4
item: ux-rename-iteration    design=0.5 impl=0.12
design-buffer: 0.15
total: 4.69
```

Items, in order: cdp pipe client; tab tracker + translate; real Chrome + inject.js; CDP/Chrome visibility discovery (live check); shared `broadcast.Host` (post-PQ-1: ref-counted, mounts by token); Browser seam + FakeBrowser + `latest[T]`; conformance harness (fake + live) incl. Task 9b; webcast server + viewer + Cast; Couch key/indicator/state machine; README + atlas; two milestone reviews (M1, M2); smoke-test iteration round. Design ×0.2 where the plan pre-resolves decisions (all but the UX round). Impl at 40% of v2 ranges.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* (Calibration source is flagged stale; numbers provisional.)

## Plan

Durable plan: `workshop/plans/000454-broadcast-web-chrome-in-couch-start-go-live-pause-end-one-cast-plan.md`.

- [x] Key decoding check (Shift+Ctrl+Alt+B vs Ctrl+Alt+b): distinct; see Log
- [x] M1 — browser-control component: `cdp` pipe client; `webcast` State/Tracker/translate; Browser seam + FakeBrowser; in-page control (isolated world); real Chrome; conformance suite run against fake and live Chrome
- [ ] M2 — Couch integration and stub stream: shared `broadcast.Host` (one tunnel, casts mounted by token); webcast viewer server + page; Cast; Shift+Ctrl+Alt+B, `Chrome ▶/⏸` cell with click / right-click / Ctrl+click; launcher wiring; README + atlas; operator smoke test

## Log

### 2026-10-11
- 2026-10-11: closed M1 — go test -race -count=10 webcast + -count=5 cdp green; mutation-checked WaitDelay/ErrProtocol class/drain (all caught); WEBCAST_LIVE_CHROME=1 live suite green unsandboxed (headless Chrome 154); review verdict: SHIP

- **Key decoding check: passed.** Couch pushes kitty flags 3; measured through `terminal.Decoder` + `productKey`: Ctrl+Alt+b → `\x1b[98;7u`, Shift+Ctrl+Alt+B → `\x1b[98;8u` (also `66;8u`, `98;8:1u`, `98:66;8u` canonicalise distinctly). Legacy `ESC ^B` was never a broadcast binding (#395). Operator's Ghostty config binds nothing on ctrl+alt+shift. Real-terminal confirmation goes in the smoke test.
- **CDP spike (Chrome 154):** `--remote-debugging-pipe` works; isolated-world binding invisible to the page's main world; `runImmediately` injects into the current document; URL/title from `Target.targetInfoChanged`; closing the pipe makes Chrome exit (owner death collects it — ARCH-FUNERAL). Chrome can't launch inside the agent sandbox (exit 21); live checks run unsandboxed.
- Design choices: pipe over port (no network surface, no websocket dep — ARCH-SECURE); `broadcast.Endpoint` extracted so both casts share one tunnel lifecycle (ARCH-DRY); viewers get URL/title only while live.

- **M1 live findings (Chrome 154, measured):** (1) `targetInfoChanged` carries a new `<title>` only much later (>15s, at detach) → the page reports its own title. (2) Setup after attach raced the first navigation → browser-level `Target.setAutoAttach{waitForDebuggerOnStart, filter: page}` (also covers the already-open page) and release with `runIfWaitingForDebugger`, always. (3) Without `Page.enable`, `addScriptToEvaluateOnNewDocument` never reaches the next document. (4) Headless second launch on a held profile exits 21 (profile in use), windowed prints the handoff. Live runs switched to headless per the operator (never steal focus from Couch); real tab switching moves to the smoke test.
- **M1 review round 1 (FIX-THEN-SHIP, 5 Important):** all fixed — BR-2 `endError` precedence; BR-3 run loop tested via scripted CDP peer + fake process (mutation-checked: release, grace cancel, conn error, SetLive-while-ending, Page.enable); BR-4 `WaitDelay`; BR-5 `DocumentChanged` clears the page title; BR-6 profile-in-use read from output, measured text `SingletonLock: File exists`. Minors: deterministic tie-break, SetLive while ending → ErrEnded, plan Revisions entry. Fake fidelity (PageTitle / grace) left: the fake models titles through Navigate and closes at once on the last tab; the scripted-peer tests cover the real paths.
- **M1 review round 2 (3 Important):** fixed by rule — BR-11 cdp reader drains after any failure (test: peer writes never block); BR-12 `cdp.ErrProtocol` class, endError allow-lists it (peer mode closing the reader first); BR-4 `startProcess` WaitDelay tested with a background child holding the pipe. Mutation-checked (WaitDelay, class, drain). lost-connection test now 10/10.
