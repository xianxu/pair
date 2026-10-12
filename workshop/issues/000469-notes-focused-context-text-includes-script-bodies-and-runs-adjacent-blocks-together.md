---
id: 000469
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '040ab4f70e106aa9f06d297c2dc3e9dc09362b21' # card fields mirrored from issue-cards; edit via sdlc
---

# notes: focused-context text includes script bodies and runs adjacent blocks together

## Problem

pair#462 (agent queries), switching onto pair#461's `notes.Structural`, found two defects in `Context.Text` (golden: `cmd/internal/webcast/query/testdata/golden/app.notes.txt`): (1) the landmark text includes `<script>` bodies (app fixture, note app-n3: the text ends with `const r=document.getElementById…`); (2) adjacent blocks run together with no separator between text nodes (`tasksAddShip onboarding flowFix billing`). An agent reading a note's context gets code and run-on words instead of what the user saw.

## Spec

Context text is the visible text a person would read: skip `script`, `style`, `template`, `noscript` and other non-rendered content (and hidden elements where the recording says so); separate block-level boundaries (and `br`, list items, table cells) with whitespace, collapse runs of whitespace the way rendering does. Keep the 500-character window. Regenerate #462's golden output after the fix.

## Done when

- The app fixture's note app-n3 context has no script text, and block boundaries are separated, in a test.
- `notes.RunModelContract` and #462's goldens pass with the corrected text.

## Plan

- [ ]

## Log

### 2026-10-12
