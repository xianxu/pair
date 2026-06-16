---
id: 000052
status: done
deps: []
github_issue:
created: 2026-06-11
updated: 2026-06-16
estimate_hours: 1.5
actual_hours: 0.66
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

- [x] `normalize_tag()` extraction — shared helper; DRYs resume/continue/rename **+ the
      name prompt** (4 sites). Returns (not exits) so it composes inside `$( )`.
- [x] `continue` list NEXT-ACTION truncation — capped at 80 chars + `…`.
- [x] NEXT-ACTION non-empty guard + test — pure `HasNextAction()` (heading must be
      followed by a non-blank, non-heading line); wired into the writer; unit-tested.
- [x] `pair --help` lists `continue`.
- [~] `parked-scrollback-*` reaper / `pair park-render` helper — **PUNTED**: this is a
      new verb (render + distill-offer), feature-sized, not polish. Real but minor
      (disk accumulation); file its own issue if/when it bites.
- [x] `tests/continue-list-test.sh` — **covered in `pair-continue-test.sh`** (bare-list +
      a new long-line truncation assertion); no separate file needed.
- [x] fresh-vs-attach decision + doc — decided **attach on live same-tag is intended**
      (never clobber a running session; "fresh" holds for a new tag); documented in the
      `continue` block comment.
- [x] park-nudge vars `local` (done: `_sb_base/_parked/_ans/_pbase`; `_pts` didn't exist).
- [~] slug-glob suffix-greedy tightening — **PUNTED**: issue says "tighten if it bites";
      no evidence it has. Note kept in Spec.

## Log

### 2026-06-16
- 2026-06-16: closed — 7 relevant minors done (normalize_tag DRY across 4 sites; writer non-empty NEXT-ACTION guard via unit-tested HasNextAction; bare-list truncation; --help lists continue; park-nudge locals; fresh-vs-attach documented; list-test coverage). Verified: bash -n + go test + pair-continue-test.sh (incl new truncation assertion) + pair-rename.sh 57/0 all green; manual resume path checked (invalid→exit1, pair-foo→FORCED_TAG=foo). Punted: park-render (feature, not polish) + slug-glob (cosmetic) → --no-plan-check; polish/DRY, no new architectural surface → --no-atlas; review verdict: SHIP
- Drained the relevant polish in one pass. **Done:** `normalize_tag()` (DRYs 4 sites —
  resume/continue/rename/name-prompt; returns-not-exits so it works in `$( )`); bare-list
  NEXT-ACTION truncation (80 + `…`); writer non-empty NEXT-ACTION guard via pure
  `HasNextAction()` (unit-tested, TDD); `pair --help` lists `continue`; park-nudge vars
  `local`; fresh-vs-attach documented (attach-on-live is intended); list-path test coverage
  via a new truncation assertion in `pair-continue-test.sh`.
- **Punted (re-scoped):** park-render reaper (#3 — feature, not polish; file its own issue
  if disk accumulation bites) and slug-glob tightening (#9 — cosmetic, "if it bites"; no
  evidence it has).
- **Verified:** `bash -n bin/pair` clean; `go test ./cmd/pair-continuation/...` ok;
  `pair-continue-test.sh` (incl. new truncation assertion) + `pair-rename.sh` (57/0) pass;
  manual resume path checked (invalid→exit 1, `pair-foo`→`FORCED_TAG=foo`).
- Closed with `--no-plan-check` (2 items deliberately punted, above) and `--no-atlas`
  (polish/DRY — no new architectural surface; `architecture.md` already accurate).
- **Boundary review: SHIP** (high confidence; no Critical/Important). Addressed its one
  actionable Minor: `HasNextAction` now uses proper ATX-heading detection (`#`+ space) so a
  `#NN`-style issue ref as the first NEXT-ACTION line counts as content, not an empty
  section — amended into the code commit with a regression test case.

### 2026-06-11
