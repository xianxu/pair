---
id: 000755
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4e4db3e5363974d547dfc9f47ccdf3991ccd8b6d' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-20
