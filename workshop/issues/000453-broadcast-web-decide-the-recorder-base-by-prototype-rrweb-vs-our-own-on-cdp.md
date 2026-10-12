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

- [x] Stress page and test harness (Chrome with its own profile, CDP, screenshot sampler)
- [x] Prototype A (rrweb via CDP)
- [x] Prototype B (own on CDP)
- [x] Fidelity scores, volume, overhead, gaps on each page
- [x] Write-up and recommendation

## Log

### 2026-10-11 — prototypes, measurements, recommendation
- 2026-10-11: closed — Prototype A (rrweb 2.1.6 over CDP) and B (own on CDP + isolated-world observer) recorded and replayed the stress page, an own React+Tailwind app, Wikipedia, HN and GitHub in one headless session; fidelity by pixelmatch vs real-tab screenshots every 2s; volume, overhead (2 runs per mode), gaps, agent fit and estimates are in the Log. Prototype code is on the unmerged branch 453-recorder-prototypes (afe773a4, pushed); the product diff is the issue file only. Project Log summary: ops commit 2dc3159.; review verdict: SHIP

**Setup.** Chrome 154, headless (`--headless=new`, own `--user-data-dir`, `--site-per-process`), 1280×800, driven
over CDP from Node. Both recorders ran **in the same session** on each page, so both replays are scored against the
same screenshots. Overhead was measured in separate runs (none / A / B, two runs each). The prototypes live outside
the product tree, on the unmerged branch `453-recorder-prototypes`
(`prototypes/broadcast-web-recorder/`, about 580 lines, with a README for re-running them).
- **A, rrweb 2.1.6:** `Page.addScriptToEvaluateOnNewDocument` (main world) in the page and in every auto-attached
  out-of-process iframe target. Child frames forward their events to the top frame through rrweb's
  `recordCrossOriginIframes`, and the top frame hands batches to the driver through `Runtime.addBinding`. Replay
  uses rrweb's `Replayer` (`pause(t)`).
- **B, own on CDP:** `DOM.getDocument` (pierce, `includeWhitespace: all`) plus DOM mutation events for each target;
  `DOM.requestChildNodes` for inserted subtrees; `CSS` for sheets and fonts; `Network.getResponseBody` for
  stylesheet bodies. An isolated-world observer (invisible to the page) covers form values and scroll. The
  replayer (~160 lines) rebuilds the DOM in a `sandbox="allow-same-origin"` frame with no scripts, and applies
  changes by node id.
- **Pages:** the stress page (100-row table with 20 cells changed every 100ms and the whole body rebuilt every 1s, a
  cross-origin iframe, closed and open shadow roots, `insertRule` CSS-in-JS, `adoptedStyleSheets`, inputs set by
  script, a script-scrolled inner container, SVG, `<template>`, `<details>`, a FontFace-API font, a CSS animation);
  our own React 18 + Tailwind-v4-runtime app, standing in for a vibe-coded app (scripted navigation, typing, a
  modal, a toast, a cross-document navigation); Wikipedia, Hacker News and github.com/rrweb-io/rrweb (scrolling).
  The startup's app had no test access, so it was not used.
- **Fidelity metric:** a low-res (640×400) screenshot of the real tab every 2s, compared with the replay at the
  same moment. The score is the % of pixels pixelmatch marks as different (threshold 0.1). A screenshot shows the
  last painted frame, so the replay is scored at t−600…t+200ms and the best match is taken. Both sides are
  captured identically.

**1. Fidelity (% pixels different per moment; median / max)**

| Page | A rrweb | B own-on-CDP | What the difference is |
|---|---|---|---|
| stress (10 moments) | 4.31 / 5.86 | 0.34 / 1.79 | A: the closed shadow root is missing, so everything below it shifts; the FontFace font is not rendered. B: animation phase and timing only. |
| own React+Tailwind app (13) | 0 / 4.56 | 0 / 4.56 | Both equal. The 4.56 is the same shared moment (focus ring and timing). |
| Wikipedia (8) | 0 / 2.78 | 0 / 1.83 | Scroll timing. |
| Hacker News (8) | 0 / 9.88 | 0 / 8.80 | Scroll lag of up to ~0.5s while the page scrolls (one moment for A, three for B). Both settle at 0. |
| GitHub repo page (8) | 1.25 / 95.81 | 0.37 / 0.53 | A: a blank replay at 2s, because rrweb takes its first snapshot at the `load` event (3.4s after navigation, vs 0.44s for B), and a 1px offset in the README region. |

Fidelity list, item by item:

| Item | A | B |
|---|---|---|
| CSSOM `insertRule` | ✓ (hooks) | ✓ — CDP fires `styleSheetChanged`, but `getStyleSheetText` returns the source text, so the rules are read through the isolated world |
| `adoptedStyleSheets` | ✓ | ✓ pure CDP (`adoptedStyleSheetsModified` + `getStyleSheetText`) |
| Form values set by script | ✓ | ✓ observer (100ms poll, then `DOM.requestNode`); not inside closed roots |
| Inner scroll | ✓ | ✓ observer (scroll events). The replay must re-apply scroll after the CSS loads (found and fixed) |
| Open shadow DOM | ✓ | ✓ |
| Closed shadow DOM | ✗ (invisible to rrweb; needs a main-world `attachShadow` hook and an rrweb patch or fork) | ✓ pure CDP |
| Cross-origin iframe | ✓ (injected per target, stitched by rrweb) | ✓ (one session per target, stitched by `frameId`) |
| FontFace-API font | ✗ in replay (recorded with a relative URL; fixable in our layer) | ✓ pure CDP (`CSS.fontsUpdated` gives family + src) |
| Hover/focus | ✗ | ✗ (both need stand-ins) |
| CSS animation timing | ≈ | ≈ (neither syncs phase) |
| Text nodes over 10k characters | ✓ | ✗ by default: CDP truncates node values at 10,000 characters (Tailwind's runtime `<style>` broke). Fixed for `<style>` via `CSS.getStyleSheetText`; other huge text nodes need a read through the isolated world |
| Browser-internal shadow trees | n/a | `pierce` also streams user-agent shadow DOM (input editors), which the recorder must drop |
| First snapshot after navigation | waits for `load` by default (`recordAfter: 'DOMContentLoaded'` is an option) | at `documentUpdated` |
| Visible to the page | yes (main-world globals, patched CSSOM) | no (isolated world only) |

**2. Volume and cost**

| Page | A events/s | A KB/min raw → gz | B events/s | B KB/min raw → gz |
|---|---|---|---|---|
| stress (busy) | 57 | 9,680 → 947 | 926 | 17,970 → 1,398 |
| own app | 3 | 209 → 27 | 7 | 386 → 31 |
| Wikipedia | 3 | 3,884 → 487 | 64 | 9,161 → 999 |
| GitHub | 1 | 15,372 → 2,009 | 43 | 24,358 → 3,273 |

- B's raw stream is the verbose CDP node format (on the stress page 53% of the bytes are `setChildNodes`, the
  subtree push for inserted rows). It also carries stylesheet bodies, which rrweb inlines only when they are
  same-origin. A compact format of our own is the plan in any case. After gzip, the fairer comparison, B is
  1.15× A on the own app, 1.48× on stress, 1.63× on GitHub and 2.05× on Wikipedia.
- Recorder overhead on the busy stress page, main-thread task time over 33.6s: none 1.05s, **A 1.56s (+0.51s,
  +1.5% of the main thread)**, **B 1.39s (+0.34s, +1.0%)**. No frames dropped in any mode (p95 16.7ms). B's
  round trips (`requestChildNodes` for each inserted subtree, `requestNode` for each observed value) cost the
  driver, not the page.

**3. Coverage gaps and what closing each takes**
- **A:** closed shadow roots (a main-world hook plus a patch to rrweb's serializer); the late first snapshot (a
  config option); FontFace URLs (absolutize them, part of the asset proxy anyway); asset bodies (rrweb keeps URLs
  for cross-origin CSS and all images and fonts, so the CDP `Network` capture is needed regardless); hover/focus.
- **B:** CSSOM rule content and form values need the isolated-world observer (built, ~30 lines); form values
  inside closed shadow roots are out of the observer's reach (reading them takes `DOM.resolveNode` on each input
  CDP shows there, or a main-world hook); the 10k text
  truncation (a general read path); dropping user-agent shadow trees; hover/focus; a replayer that keeps meeting
  edge cases rrweb has already solved (top layer/`<dialog>`, `srcset`/`<picture>`, CSS `url()`, selection, custom
  elements). None was a blocker on these five pages.

**4. Fit with the agent format** ("DOM at t, events in a window, notes on elements")
- **A:** rrweb's format is compact and well specified, but its ids exist only inside rrweb's mirror. To get "the DOM
  at t", an agent (in Go) must either re-implement rrweb's apply semantics or run rrweb's replayer in a headless
  browser. We would store two formats (rrweb's, plus a derived index) and depend on a ~266KB (83KB gz) script
  injected into every frame of the product.
- **B:** the stream is already one we define. The Go recorder normalizes CDP events into our compact operations
  (insert/remove/attr/text/value/scroll/sheet), and the viewer's replayer and the agent query CLI apply the same
  operations. `DOMSnapshot` boxes are available for notes in the same session. Couch has no CDP client today
  (no chromedp in go.mod), so both options add one: the Chrome-in-Couch item needs it in any case.

**5. Build estimate for the MVP recorder + replayer**
- **A:** ~6–7 days. Injection and stitching per target, buffering and segments ~1.5d; rrweb live replay in the
  viewer ~1d; deriving the agent index (a Go port of the apply semantics, or a headless query path) ~2–3d;
  closed-shadow patch and font/first-snapshot fixes ~1.5d. After that we carry a forked or patched vendored
  dependency.
- **B:** ~7–7.5 days. A Go CDP recorder with event normalization, the observer and asset bodies ~3d; hardening the
  replayer from the prototype, using rrweb's code as a reference for edge cases, ~2.5–3d; agent queries on the
  same format ~1.5d. We own all of it from then on.

**Recommendation: B, own on CDP, with a small isolated-world observer (a hybrid in the narrow sense).**
- Its median fidelity equalled or beat A's on every page. It was worse only at mid-scroll moments on HN, where B's
  scroll lagged at 3 moments and A's at 1, because B's observer round trips are slower than rrweb's in-page
  sampling. It was better where it matters for our targets: closed shadow roots, FontFace fonts, and the first
  snapshot on a heavy SPA (rrweb's replay was blank for the first ~3s on GitHub).
- It is invisible to the product page, and it uses the asset bodies the proxy needs anyway.
- It is one format from browser to agent, applied by the same code in the viewer and the CLI.
- It had the lower page overhead on the busy stress page, the only page where overhead was measured (+1.0% vs +1.5%
  of main-thread time).
- Its costs are known and bounded: about half a day more of build, 1.15–2.05× the gzipped bytes before a compact encoding,
  and a replayer we harden ourselves.
- rrweb remains the reference for the replayer's edge cases, and the fallback if replayer hardening runs long.
- The operator decides.

Project Log summary: ops `teams/devinfra/projects/couch-broadcast-web/project.md`, entry "2026-10-11 — pair#453
recorder base: prototypes compared" (commit in ops main).
