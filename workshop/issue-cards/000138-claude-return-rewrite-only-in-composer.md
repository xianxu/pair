---
id: '000138'
status: done
started: 2026-08-19T19:25:36-07:00
created: 2026-08-16
updated: 2026-08-20
estimate_hours: 3.32
actual_hours: 3.13
---

# Claude Return rewrite only in composer

## Problem

Claude currently gets pair-style multiline input by rewriting plain Return in
the agent pane by default, with overlay detection as the escape hatch. Codex
now uses the safer rule: rewrite only when Pair positively identifies the live
composer/input box. Claude should move to the same integration contract so
permission prompts, pickers, and future Claude UI variants do not depend on
enumerating every non-composer menu.
