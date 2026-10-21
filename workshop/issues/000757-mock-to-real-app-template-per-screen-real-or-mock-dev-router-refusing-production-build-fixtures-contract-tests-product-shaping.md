---
id: 000757
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c7addaca11d56918eae5e115f23bcee4773c0e16' # card fields mirrored from issue-cards; edit via sdlc
---

# mock-to-real app template: per-screen real-or-mock dev router, refusing production build, fixtures contract tests (product-shaping)

## Problem

product-shaping (ops) grows the real app out of its mock screen by screen. In development, unbuilt screens should
still be clickable as mocks; nothing half-mocked may ship.

## Spec

A reusable template (Go) for mock-backed apps:
- a development router that serves each screen from real code if it exists, else from the mock, with a visible MOCK
  badge, and a progress count ("12 of 18 screens real") from the state list;
- the production build refuses while any listed screen is mock-backed (or, if a partial release is chosen
  explicitly, those screens render a "not yet" page, never the mock);
- contract tests: the real backend's responses are checked against the shape of `fixtures.json`.

## Done when

- A sample app with two real screens and one mock screen runs in development with the badge and count; the
  production build refuses until the third is real; the contract test catches a shape drift.

## Plan

- [ ]

## Log

### 2026-10-20
