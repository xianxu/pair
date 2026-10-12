---
id: 000453
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'b2f09a38ebafc8955bb90fe284d67412e187e050' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T20:08:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "471f3cb6", done: "866c3d8a"}
---

# broadcast-web: decide the recorder base by prototype (rrweb vs our own on CDP)

## Problem

couch-broadcast-web (project: ops `teams/devinfra/projects/couch-broadcast-web/project.md`) casts a local Chrome tab as a DOM stream and records it for viewers and agents. Its recorder and replayer are the core of the product, and the base is undecided: rrweb (a mature in-page recorder and replayer, injected through CDP) or our own built on Chrome's DevTools protocol (the DOM domain streams the tree and every change for all frames, including cross-origin iframes and closed shadow roots, without injection). The operator wants the decision made from evidence, not argued.

## Spec

Build two throwaway prototypes against the same pages and compare them. Both drive a Chrome started with its own `--user-data-dir` over CDP; neither needs Couch integration.

- **A, rrweb:** inject rrweb's recorder through CDP (`Page.addScriptToEvaluateOnNewDocument`, main world, every frame via auto-attach); replay with rrweb's replayer in a sandboxed, script-free, scaled frame.
- **B, own on CDP:** `DOM.getDocument` (pierce) plus DOM mutation events per target; `CSS` and `Network`/`Fetch` domains for stylesheets and assets; an observer script for state CDP doesn't report (form values set by script, scroll offsets, selection, CSSOM changes if the CSS domain doesn't report them). Where hooking the page's own functions is the only way (as rrweb does), inject into the main world; record which items needed it. A minimal replayer that rebuilds the DOM in a sandboxed frame and applies changes by node id.

Test pages: the startup's vibe-coded app on a test account (if access is ready), one of our own web apps, and a stress page with heavy re-renders, a cross-origin iframe, a closed shadow root, CSS-in-JS, and inputs set by script.

Compare on:
1. **Fidelity:** a low-res screenshot of the real tab every few seconds vs a render of the replay at the same moment (a pixel-diff score per moment), plus the known fidelity list item by item.
2. **Volume and cost:** events per second and bytes per minute on the busy page; recorder overhead on the page (frame times).
3. **Coverage gaps:** what each misses, and what closing each gap would take.
4. **Fit with the agent format:** how directly each maps to "DOM at t, events in a window, notes on elements".
5. **Build estimate** for the MVP recorder + replayer on each base.

The prototypes stay out of the product tree (a `prototypes/broadcast-web-recorder/` directory or a scratch branch that is not merged); only the write-up lands.

## Done when

- Both prototypes recorded and replayed the same test pages, and the comparison (the five points above, with numbers) is written in this issue's Log and summarized in the project's Log.
- A recommendation, rrweb or own-on-CDP or a hybrid, with its reasoning; the operator decides.
- No prototype code merged into the product.

## Plan

- [ ] Stress page and test harness (Chrome with its own profile, CDP, screenshot sampler)
- [ ] Prototype A (rrweb via CDP)
- [ ] Prototype B (own on CDP)
- [ ] Fidelity scores, volume, overhead, gaps on each page
- [ ] Write-up and recommendation

## Log

### 2026-10-11
