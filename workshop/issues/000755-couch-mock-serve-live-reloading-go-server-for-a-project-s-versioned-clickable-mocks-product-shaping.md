---
id: 000755
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 1.69
collaboration-mode: collaboration
card_mirror: '5716d83421c319a421043ff09d5bdb2e59d56166' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T22:40:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

- [x] Implement the versioned mock server and browser reload contract, with deterministic and real-browser verification.
- [x] Wire standalone Couch commands, add the sample and conventions, run the recorded suite, close and land.

Implementation details: [durable plan](../plans/000755-mock-server-plan.md).

## Log

### 2026-10-20

- Claimed in pair:4 after pair#733 landed. Read the operator-approved product-shaping design in ops. The server is standalone: no supervisor, terminal, stored Couch identity or visible browser launch. Surveyed the existing CLI, loopback HTTP lifecycle, browser/CDP qualification and source inventories. A bounded context exploration returned integration points without edits.
- Alternatives: native file watching adds a dependency and recursive watch lifecycle; rescanning per browser multiplies project IO by browser count. Recommend one bounded content scan owned by the server, with browsers polling its cached revision (ARCH-DRY, ARCH-CONSTRAINTS). Use an HTML parser for authoritative root attributes rather than regex or browser-only tagging (ARCH-PURE).
- TL read the committed plan and found it sound, but operator design review is required for product-shaping. Held all implementation pending the operator's answer, relayed through the TL, on whether the highest-numbered version should show by default. No `change-code` or implementation has run.
- During the authorized hold, read pair#757 and pair#756 without edits or commits on either. #757's per-screen development routing, production refusal and fixture-shape contract are separate from #755; #756 consumes the two page attributes. Keep screen IDs stable across version folders and keep one project-root fixture file as planned. Ariadne#512 now contains the published pair#733 cleanup evidence; its landed branch remains preserved.
- Operator approved the design as planned via TL, including highest-numbered default, version folders, shared fixtures, state table and limits. Approval and the non-enforced git feedback convention are recorded in the durable plan's Revisions. Implementation hold is released.
- Plan-quality round 1 blocked on PQ-1 document bootstrap ordering and PQ-2 production-function test strategies; advisory PQ-3 noted unbounded empty-directory traversal. Verified all three against the written plan and appended governing refinements: baseline captured before document read and compared on first poll, named function strategies, and an all-entry traversal cap within the existing file envelope. No implementation started.

- Implementation and sample are complete. Targeted HTTP/CLI, package race, artifactpath and repolint checks passed. Real private-profile Chrome qualified HTML/CSS/JS/fixture edit reload, clicks, switch/revert, invalid-fixture repair and held-first-poll edit/switch orderings, plus the checked-in three-screen/two-version sample and its empty/error/80-row cases. Metadata/script suppression negative controls failed as intended; cached-read rescan and doubled file-read mutations failed their regressions. Node client tests passed all five request/decision cases. Audited new readers as private/literal fixtures, except the actual sample browser reader which uses the conservative non-Go full fallback. Preparing the recorded pushed-head suite and mandatory close review.

- PR507 is open and the recorded pushed-head affected check is started. Implementation checklist is complete; suite completion, close verdict and landing remain machine-gated. Using the TL-worker allowance to overlap the queued recorded suite with boundary review; no landing before a final-head recorded pass.

## Revisions

### 2026-10-20 — concrete server contract for TL approval

- Project layout: root `fixtures.json`, `mock-1/`, `mock-2/`, etc.; HTML pages and their CSS/JS/assets live inside each version. The state list remains human-authored in the project's `design.md`. Screen identity is the HTML path relative to its version, without `.html` (`index`, `runs/detail`); query strings describe scenarios within that screen. Numbered style variants such as `mock-7a` are allowed, matching the approved product design.
- `serve <dir> [--port N]` binds only 127.0.0.1 (default port 0, print actual URL), serves until interrupted, and opens no browser. Initial selection is the highest numbered version unless `.mock-current` explicitly selects one. `use <dir> VERSION` validates an existing version and atomically replaces that small text pointer; last completed replacement wins. A missing/broken selection refuses visibly, never silently picks another version.
- Root navigation redirects into the current version URL. Pages receive server-authored root data attributes and an injected reload client. A version switch moves open pages to the same screen in the selected version, or its index when that screen is absent; queries and fragments are preserved. A content edit reloads the current screen. Old versions remain on disk so switching back restores their identities; sketches remain owned by cast-sketch.
- All screens fetch `/fixtures.json`, the one shared file. Serve version assets and that fixture only; do not expose arbitrary project-root files. Files and paths stay beneath the mock root; reject symlink entries rather than following them outside the project. No writes are accepted over HTTP.
- One 500ms scan hashes bounded project contents, independent of browser count; each browser polls cached state every 500ms. Proposed local-prototype envelope: at most 1,024 files and 32 MiB total, with HTML at most 1 MiB. Over-limit, unreadable and invalid input produces an explicit error with repair guidance; runtime recovery observes the repaired tree. Count file visits and bytes at N/2N, and prove polling adds no scans. No claimed hard wall-clock bound for stalled filesystem IO.
- This exceeds the quick code shell. Request full-flow approval for one atomic close boundary (plain checkboxes, no artificial Mx milestones). Headless browser qualification runs in a temporary profile; operator live look is after landing, as dispatched.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. The source is marked stale by estimate-source; values remain provisional. Derived after plan-quality cleared.

Three bounded concerns: project/document core, HTTP/scanner lifecycle, browser reload. Go net/http, os.Root and x/net/html cover the first two: base design 1h each ×0.5 library availability ×0.2 settled-spec discount =0.10h each. Browser reload uses base design 1h ×0.2=0.20h; no library discount for its document-ordering policy. Base implementations 0.6/0.8/0.6h ×0.4 yield0.24/0.32/0.24h. Familiar Go/HTTP/JS and existing CDP seams justify1.0 familiarity. CLI/sample is a well-specced extension (design0.2×0.2, impl0.4×0.4); docs (design0.1×0.2, impl0.15×0.4) and the single review boundary (design0.1×0.2, impl0.3×0.4) are separate overhead. Thorough approved plan uses15% design buffer; there is no cross-repo propagation. Sum design0.48×1.15 + impl1.14 =1.692h, rounded1.69h. No new API discovery: browser qualification uses the existing CDP transport and private-profile pattern.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.10 impl=0.24
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.20 impl=0.24
item: smaller-go-module design=0.04 impl=0.16
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.12
design-buffer: 0.15
total: 1.69
```
