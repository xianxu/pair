---
id: 000462
status: open
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '4d1d1def47a8a16f57f4d8f8299f25e622b4374f' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: agent queries — DOM at t, events in a window, notes with context

## Problem

Wave 1 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`; read its Direction, Development approach and Principles). Depends on pair#457 (recording format and component seams): build against its format, Go interfaces, fakes and fixtures, not against the other wave-1 components, which are being built at the same time. Your component sits behind its #457 interface; test it against fakes and the fixtures, and add contract tests the other side can run. If the format or a seam is wrong for you, don't change it locally: report it to the TL (ops:0), who revises #457's contract. Browser runs are headless and never take focus from Couch. The real test app is the operator's blog (`~/workspace/xianxu.dev`, `npm run dev`).

## Spec

Implement #457's query surface as a CLI agents use on a recording (built on the reference reducer):
- The DOM at time t, optionally under a selector, as compact readable text (not raw JSON by default; `--json` for structure).
- Events between t1 and t2 (navigation, input, scroll, DOM changes summarized).
- Every note with its element, the DOM around it at that time (the notes model's focused context, pair#461, through its #457 interface/fake until it lands), and the user actions in the N seconds before it.
- Command shape and name fit Couch's CLI conventions (`couch --help`); outputs sized for an agent's context.

## Done when

- The three queries run on every #457 fixture with golden-output tests.
- An agent given only a fixture recording and the CLI answers what a fixture note refers to (element, time, preceding action): a scripted check, or a recorded manual run in the Log.

## Plan

- [ ] DOM at t (text and --json)
- [ ] Events in a window
- [ ] Notes with context
- [ ] Golden tests; agent check

## Log

### 2026-10-11
