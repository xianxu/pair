---
id: 000755
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '91cda1e4dea415bcac5f1527b1446591fec8a775' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T22:40:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping)

## Problem

Project product-shaping (ops `teams/devinfra/projects/product-shaping/`, operator-approved 10-20) shapes UI projects
through clickable mocks iterated by sketching. It needs one default way to serve a project's mocks.

## Spec

- `couch mock serve <dir>`: a small Go server for a folder of plain HTML/CSS/JS mocks (no framework), with live
  reload in the browser on any file change.
- **Numbered versions:** each iteration is `mock-1`, `mock-2`, … (a folder or snapshot per version); the server serves
  a current version and can switch (`couch mock use <dir> mock-2`), so "revert to mock-2" is one command.
- **A clickable state machine:** screens link to each other; every screen reads one `fixtures.json`.
- **Self-describing pages:** the server marks each page's root element with `data-mock-version` and
  `data-mock-screen`, so tools such as cast-sketch can scope comments without knowing about the server.
- Local only (127.0.0.1), no auth needed; the web cast reaches it like any page.
- Conventions documented: the state list in the project's `design.md`, empty, error and long-list states, fixtures
  shape as the future API contract.

## Done when

- A sample project with three screens, two versions and fixtures serves, live-reloads on edit, switches versions,
  and exposes the two data attributes; tests cover version switching and the attributes.

## Plan

- [ ] Implement the versioned mock server and browser reload contract, with deterministic and real-browser verification.
- [ ] Wire standalone Couch commands, add the sample and conventions, run the recorded suite, close and land.

Implementation details: [durable plan](../plans/000755-mock-server-plan.md).

## Log

### 2026-10-20

- Claimed in pair:4 after pair#733 landed. Read the operator-approved product-shaping design in ops. The server is standalone: no supervisor, terminal, stored Couch identity or visible browser launch. Surveyed the existing CLI, loopback HTTP lifecycle, browser/CDP qualification and source inventories. A bounded context exploration returned integration points without edits.
- Alternatives: native file watching adds a dependency and recursive watch lifecycle; rescanning per browser multiplies project IO by browser count. Recommend one bounded content scan owned by the server, with browsers polling its cached revision (ARCH-DRY, ARCH-CONSTRAINTS). Use an HTML parser for authoritative root attributes rather than regex or browser-only tagging (ARCH-PURE).
- TL read the committed plan and found it sound, but operator design review is required for product-shaping. Held all implementation pending the operator's answer, relayed through the TL, on whether the highest-numbered version should show by default. No `change-code` or implementation has run.
- During the authorized hold, read pair#757 and pair#756 without edits or commits on either. #757's per-screen development routing, production refusal and fixture-shape contract are separate from #755; #756 consumes the two page attributes. Keep screen IDs stable across version folders and keep one project-root fixture file as planned. Ariadne#512 now contains the published pair#733 cleanup evidence; its landed branch remains preserved.

## Revisions

### 2026-10-20 — concrete server contract for TL approval

- Project layout: root `fixtures.json`, `mock-1/`, `mock-2/`, etc.; HTML pages and their CSS/JS/assets live inside each version. The state list remains human-authored in the project's `design.md`. Screen identity is the HTML path relative to its version, without `.html` (`index`, `runs/detail`); query strings describe scenarios within that screen. Numbered style variants such as `mock-7a` are allowed, matching the approved product design.
- `serve <dir> [--port N]` binds only 127.0.0.1 (default port 0, print actual URL), serves until interrupted, and opens no browser. Initial selection is the highest numbered version unless `.mock-current` explicitly selects one. `use <dir> VERSION` validates an existing version and atomically replaces that small text pointer; last completed replacement wins. A missing/broken selection refuses visibly, never silently picks another version.
- Root navigation redirects into the current version URL. Pages receive server-authored root data attributes and an injected reload client. A version switch moves open pages to the same screen in the selected version, or its index when that screen is absent; queries and fragments are preserved. A content edit reloads the current screen. Old versions remain on disk so switching back restores their identities; sketches remain owned by cast-sketch.
- All screens fetch `/fixtures.json`, the one shared file. Serve version assets and that fixture only; do not expose arbitrary project-root files. Files and paths stay beneath the mock root; reject symlink entries rather than following them outside the project. No writes are accepted over HTTP.
- One 500ms scan hashes bounded project contents, independent of browser count; each browser polls cached state every 500ms. Proposed local-prototype envelope: at most 1,024 files and 32 MiB total, with HTML at most 1 MiB. Over-limit, unreadable and invalid input produces an explicit error with repair guidance; runtime recovery observes the repaired tree. Count file visits and bytes at N/2N, and prove polling adds no scans. No claimed hard wall-clock bound for stalled filesystem IO.
- This exceeds the quick code shell. Request full-flow approval for one atomic close boundary (plain checkboxes, no artificial Mx milestones). Headless browser qualification runs in a temporary profile; operator live look is after landing, as dispatched.
