---
id: 000052
status: working
deps: []
github_issue:
created: 2026-06-11
updated: 2026-06-16
estimate_hours: 1.5
---

# pair continue: deferred minors (verb + writer polish)

## Problem

`#50` (pair continue) shipped with several **Minor** findings deferred fix-forward across
its milestone + close reviews. Capturing them here so they survive `#50`'s Log being
archived to `history/`. None are blocking; pick up opportunistically.

## Spec

Grab-bag of polish for the `continue` verb + the writer (each independent):

- **`normalize_tag()` shell helper (ARCH-DRY).** The `pair-*`-strip + `*[!A-Za-z0-9_-]*`
  charset validate is duplicated between the `resume` and `continue` blocks in `bin/pair`
  — extract a shared shell fn.
- **`continue` list one-liner truncation.** `pair continue` (bare) prints the *whole* first
  NEXT-ACTION line, which floods the row — truncate to ~N chars (observed in the `#50` dogfood).
- **`parked-scrollback-*` reaper / lister.** The Alt+x park-nudge renames captures to
  non-recyclable `parked-scrollback-<tag>-<ts>.*`; they accumulate with no cleanup or
  surfacing. A `pair park-render <tag>` helper (render + offer to distill) would close the loop.
- **NEXT-ACTION guard: non-empty, not just heading-presence.** The writer rejects a body
  with no `## NEXT ACTION` heading, but a heading with no content passes; assert a non-blank
  line follows.
- **`pair --help` should list `continue`** alongside `resume` (README usage already does).
- **`tests/continue-list-test.sh`** fixture for the bash list path (awk/glob regressions).
- **Fresh-vs-attach on slug collision.** `pair continue <slug>` onto a *live* same-tag
  session attaches rather than starting a fresh seeded session — reasonable, but the Spec's
  "fresh session" isn't guaranteed when the slug collides with a live tag; decide + document.
- **park-nudge shell vars → `local`** (`_sb_base`/`_parked`/`_ans`/`_pts`/`_pbase` leak to
  caller scope; harmless since the script exits after, but inconsistent).
- **Slug glob suffix-greedy.** `*-"$slug".md` also matches `…-deep-<slug>.md`; `sort|tail -1`
  picks one. Low-impact; tighten if it bites.

(The repo-root / cross-repo guard is tracked separately in `#51`.)

## Done when

- Each item is either done (with its own commit) or explicitly punted with a reason. This is a
  checklist issue, not an atomic deliverable — close when the list is drained or re-scoped.

## Plan

- [ ] `normalize_tag()` extraction
- [ ] `continue` list NEXT-ACTION truncation
- [ ] NEXT-ACTION non-empty guard + test
- [ ] `pair --help` lists `continue`
- [ ] `parked-scrollback-*` reaper / `pair park-render` helper
- [ ] `tests/continue-list-test.sh`
- [ ] fresh-vs-attach decision + doc
- [ ] park-nudge vars `local` + slug-glob tightening

## Log

### 2026-06-11
