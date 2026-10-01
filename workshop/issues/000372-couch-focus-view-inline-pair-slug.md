---
id: 000372
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'b5628768b5a3da54f2e9518e41f79a86ea4c90ee' # card fields mirrored from issue-cards; edit via sdlc
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
- **Kept current, read from memory:** couch reads the slug files during the
  inventory refresh it already runs, and re-reads one only when its
  mtime/size changes. Nothing is read while drawing or on a keystroke. The
  renderer gets the slug as state, like the description.
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
- Test showing the refresh re-reads a slug file only when it changed.
- Operator smoke test in a live couch (after `make build`): the focus view
  shows each live agent's current slug, and it updates after the agent's next
  turn without any notification.

## Plan

- [ ] Add a pure parser `=== L | R ===` → `L | R` (next to slugcmd's
      `slugRE`, sharing that one format definition per ARCH-DRY)
- [ ] Couch: resolve each live thread's `Paths` (data dir + tag), read
      `SlugProposed` on refresh with an mtime/size check, carry the slug on the
      menu's thread state
- [ ] `menu_render.go`: focus rendering appends `◆ slug` when present
- [ ] Tests from Done when; `make test`
- [ ] Update the focus-view paragraph in `atlas/couch.md` (#338 section)

## Log

### 2026-10-01
