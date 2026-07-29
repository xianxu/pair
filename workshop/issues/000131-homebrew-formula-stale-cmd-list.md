---
id: 000131
status: open
deps: []
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours:
---

# homebrew formula cannot build: stale cmd list

## Problem

**`brew install xianxu/pair/pair` has been failing since 2026-06-30.** This is
live breakage for anyone installing or upgrading, not just a release blocker.

The formula (`../homebrew-pair/Formula/pair.rb`) builds six binaries in a loop:

```
pair-go, pair-wrap, pair-scrollback-render, pair-changelog,
pair-context, pair-session-watch
```

while its `url` still points at the **v1.23** tarball. Verified against the real
GitHub release archive — v1.23 contains:

```
cmd/{internal, pair-changelog, pair-continuation,
     pair-scribe, pair-scrollback-render, pair-slug, pair-wrap}
```

So `pair-go`, `pair-context` and `pair-session-watch` are absent and
`go build ./cmd/pair-go` fails outright.

Timeline:

| when | what |
|---|---|
| 2026-06-17 | v1.23 tagged |
| **2026-06-30** | formula commit `3aeb2a6` "build Go public entrypoint" adds `./cmd/pair-go` — while `url` stays at v1.23 |
| 2026-07-06 | `cmd/pair-go` actually appears in the source (#104 M3) |

The formula was edited to match a *future* source tree against a *past* tarball.

It also cannot be repaired in place: the v1.23 tarball needs the *old*
multi-binary list, so reverting the formula would unbreak installs but strand
users two months behind. #104 has since collapsed everything into a single
binary — `cmd/` now holds only `internal/` and `pair-go/`.

Three further staleness bugs in the same file:

- `desc` advertises "Gemini", removed in #40 — should be Antigravity.
- A comment describes `bin/pair-shell` as "the retained shell compatibility
  launcher"; it was deleted in #99 M5c.
- `caveats` says "Run `pair --help` for keybindings" — false, see **#132**.

## Spec

- Cut **v1.24** from current main; the formula's `url`/`sha256` move to it.
- Rewrite the install block for the single binary: build `./cmd/pair-go` only,
  install as `bin/pair`. Drop the whole `%w[...]`-style loop — every former
  helper is a `pair <sub>` since #104.
- Fix `desc`, the `bin/pair-shell` comment, and the caveats line.
- CHANGELOG entry for v1.24 covering the ~54 issues since v1.23 (layout 3,
  single-binary port, repo-scoped tags, review pane, global hotkey routing, the
  #127 terminal-stream fixes).

Release procedure is in the operator's notes: tag `pair`, compute the sha256
from the **GitHub-generated** tarball, bump the separate `homebrew-pair` repo.
Packaging-only formula edits need no re-tag; this one does, because the source
layout changed.

## Done when

- `brew install --build-from-source xianxu/pair/pair` succeeds from a clean
  prefix.
- `brew style Formula/pair.rb` clean.
- `pair --version`/`pair --help` runs from the brew-installed binary, and a
  session launches (the runtime assets resolve from `libexec`).

## Plan

- [ ]

## Log

### 2026-07-29
