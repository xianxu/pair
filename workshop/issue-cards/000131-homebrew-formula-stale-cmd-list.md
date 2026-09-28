---
id: '000131'
status: done
started: 2026-08-16T20:13:46-07:00
created: 2026-07-29
updated: 2026-08-16
estimate_hours: 2.65
actual_hours: 0.79
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
