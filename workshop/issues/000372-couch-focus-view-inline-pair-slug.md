---
id: 000372
status: working
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '109a5eaa3268f3e535b754d5366a7da9c4af50ff' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-01T15:09:24-07:00
flow: {kind: quick, provenance: inferred, spec: "127580d7", done: "1f629565"}
---

# couch focus view: show pair-slug inline

## Problem

After jumping to a notifying agent (ctrl+Return), it takes a while to work out
what that agent is doing. The switcher's focus view (#338) could orient the
operator *before* the jump. Each row there shows only `name ◆ description`, and
the description is static: the operator sets it with the `!` notation
(`PublishedSummary`, which falls back to the typed `Description` via
`ThreadSummary.DisplaySummary`).

pair-slug already tracks what each agent is doing. At every turn end,
`pair-wrap` starts `pair-slug` in the background. It feeds the last ~12
transcript turns plus the previous slug to a small model. The model returns a
new `<focus>` or `KEEP`. The result is written atomically (write a temp file,
then rename) to `slug-proposed-<tag>` (`artifactpath.Paths.SlugProposed`), in
the form `=== <branch-left> | <focus> ===`. nvim is currently its only
consumer: it writes the slug into draft line 1 when that line is free, and
mirrors any edits back into `slug-<tag>`. Couch never reads either file.

## Spec

- In the focus view, a root row renders on **one line**:
  `name ◆ description ◆ slug`. It is clipped to terminal-cell width by the
  existing `clipMenuLine`, so a long line is truncated, not wrapped.
- **Source:** the slug comes from the suggestion file `slug-proposed-<tag>`
  (`Paths.SlugProposed`). Never from `slug-<tag>`, and never from the draft's
  line 1, so the operator's local edits are ignored.
- **Format:** drop the `=== ` / ` ===` fence and keep the branch prefix.
  `=== main-slot3 | couch switcher click-select feature ===` becomes
  `main-slot3 | couch switcher click-select feature`. The text goes through the
  menu's existing sanitizer for agent-written text (strip control characters).
  A file that is missing, empty, or malformed means no slug: the row stays
  `name ◆ description` with no trailing `◆`.
- **Focus membership is unchanged:** a row must be `Live()` and have a
  non-empty `DisplaySummary()`. A row that has a slug but no description stays
  out of focus view.
- **Kept current, read from memory:** couch reads each live row's slug file
  (bounded, ≤ 1 KiB) during the inventory refresh it already runs. That refresh
  is off the render path and fires when the switcher opens and after every
  operation. Nothing is read while drawing or on a keystroke. The renderer gets
  the slug as a row field (`ActionableThreadSummary.Slug`), like the
  description.
- **Not a notification:** a slug change only redraws the row. It never creates
  attention, paging, a bell, or idle-fade activity. Attention lines keep
  rendering below the row as they do today.
- Out of scope: normal view (unchanged), and matching the slug in the
  typeahead filter.

## Done when

- Unit tests:
  - focus view renders `name ◆ description ◆ slug` with the fence removed;
  - with no slug or a malformed one, it renders `name ◆ description` with no
    trailing `◆`;
  - a row with a slug but no description is not in focus view;
  - a long line is clipped at the terminal width;
  - control characters in the slug are stripped;
  - a slug change produces no attention or notice;
  - the slug is read from `SlugProposed`, never from `Slug`.
- Inventory test: a live row carries its slug from the injected reader; a
  non-live row gets none, and the reader is not called for it; a reader error
  leaves the slug empty without failing the inventory.
- Operator smoke test in a live couch (after `make build`): the focus view
  shows each live agent's current slug, and it updates after the agent's next
  turn without any notification.

## Plan

- [x] New pure package `cmd/internal/slugline`: `Format(left, focus)`,
      `Valid`, `Focus`, `Unfence`. slugcmd switches to it, so the
      `=== L | R ===` format has one definition (ARCH-DRY)
- [x] couchcore: `ActionableThreadSummary.Slug`; a `Couch.Slug` seam (like
      `OrientationStatus`); `ApplySlugs` runs after the projection, in the same
      place as `ApplyRepositoryAliases`, for live rows only. Display-only:
      errors leave the slug empty
- [x] `OSSlugReader{DataDir}`: `artifactpath.Resolve` → `SlugProposed`, a
      bounded regular-file read, then `slugline.Unfence`. Wired in
      `couchcmd/run.go` next to `OrientationStatus`
- [x] `menu_render.go`: focus rendering appends ` ◆ slug` when present
      (sanitized)
- [x] Tests from Done when; `make test`
- [x] Update the focus-view paragraph in `atlas/couch.md` (#338 section)

## Revisions

- 2026-10-01 — design pass. Dropped the mtime/size change check: one bounded
  read of a ≤ 1 KiB file per live row costs about the same as the stat, and the
  inventory refresh already runs off the render path. The slug is a row field
  filled in after the projection (the `ApplyRepositoryAliases` precedent), not a
  third probe pipeline beside activity and slot-git. Done-when's
  "re-read only on change" test became the inventory seam test.

## Log

### 2026-10-01

- Built per Plan (6e20f14d). `slugline` owns the `=== L | R ===` format;
  slugcmd's `validateSlug`/`rightOf` and their tests moved there (ARCH-DRY).
- `couchcore/slug.go` registered in the artifactpath inventory as a
  ResolvedConsumer (binding `composite-slug-proposed`), `slugline.go` as a
  non-artifact source. The suite's `TestProductionArtifactReferencesAreExactlyClassified`
  still fails, with the 54 violations byte-identical to an `origin/main` worktree.
- Mutation check: dropping `menuFocusSlug` from the focus render fails the two
  new slug tests (5 assertions).
- Verification (five-var scrub, sandbox off): `make -k test` → only
  `test-changelog` (known, default TMPDIR); `TMPDIR=<scratchpad> make
  test-changelog` → pass; `go test ./... -count=1` → only the three known
  failures that also fail on main (`TestBareCouchInstalledCommand`,
  `TestProductionArtifactReferencesAreExactlyClassified`,
  `TestCouchReferencesLocalArchiveLocatorRoundTrip`).
