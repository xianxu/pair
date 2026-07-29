---
type: continuation
slug: start-130-session-names
agent: claude
created: 2026-07-29T10:57:22
branch: main
issues: [000130]
---

## NEXT ACTION

**Implement #130** (`workshop/issues/000130-session-name-folder-prefix.md` — claimed,
planned, not started). Run `sdlc change-code --issue 130` to branch, then work the
plan in the issue.

**Start with the riskiest item, not the first one.** The plan's order is logical,
but `session_blocks_reuse` calls `zellij delete-session --force` on `EXITED` rows,
and it decides "is this mine?" purely by the `pair-` name prefix. Get that
predicate wrong while introducing a second prefix and pair deletes *someone
else's* zellij session. Write the "a foreign session is never a deletion
candidate" test first — `fabulous-aardvark` was genuinely present in the live
session list while scoping this, so the case is real, not theoretical.

The single most important structural call in the plan: **one `isPairSessionName`
predicate shared by all four filters** (`zellij.go:27`, `zellijparse.go:60`,
`legacy_live.go:19`, `lifecycle.go:158`), not four edited string literals. Four
independent copies of one rule is exactly the shape that caused #127's SGR
terminator bug this same session.

## State of play

**#130 — the work.** Target `📁{repo}[-{residual tag tokens}]`:
`pair-pair-pair` → `📁pair`; `pair-parley_nvim-parley_nvim` → `📁parley-nvim`.
Full spec, worked examples, accepted trade-offs and plan are all in the issue file.

Facts established empirically this session (do not re-derive):
- zellij's session-name limit is **24 BYTES, not characters** — 24 chars accepted,
  25 rejected with the nonsense message `session name must be less than 0
  characters` (it computes budget minus the long macOS cache path and goes
  negative). Emoji test: `🚧-`+22 chars = 27 bytes rejected; 21 chars = 24 bytes
  accepted.
- `📁` is 4 bytes with no separator, vs `pair-` at 5 → budget for the rest is **20
  bytes**.
- Emoji session names work end to end: create, list, address by name, on-disk
  `…/session_info/📁pair`, delete. Tested with both `🚧-pair` and `📁pair`/`📁pair-1`.
- `BuildSessionNameCandidates` truncates by `[]rune` against that **byte** budget.
  Harmless while everything is ASCII; a prerequisite fix once the prefix is
  multi-byte.
- The tab title is `<session name> | <focused pane title>`, composed by
  `zellij-utils/src/shared.rs::make_terminal_title` — **hardcoded, no config
  exists**. Verified by enumerating every zellij option, reading the docs, and
  reading the source. Upstream asks are open since 2022:
  zellij-org/zellij#1495 and #2088. `#1495` had activity 2026-07-10, so commenting
  there beats filing a third duplicate.

**#129 — the dependent tail.** Branch `000129-pane-titles-cwd-role` exists with
three commits; `make test` green. Its remaining work is one format string (drop
`[<cwd>]` from `titlepoller.frameTitle` so the agent title is `claude (629k)`),
and it **must not land before #130** — the intermediate state
`pair-pair-pair | claude (629k)` identifies the repo nowhere. An earlier design
(cwd+tag prefix on every pane title) was built and then reverted in `ac6b879`
because it duplicated what the session name will carry; the abbreviator
consolidation from that work was kept and is live. Also measured and recorded: a
pane name **survives** `next-swap-layout`, so no re-rename hook is needed.

**Also open, and #131 is arguably higher priority than either title issue:**
- **#131 — `brew install xianxu/pair/pair` has been broken since 2026-06-30.** The
  formula builds `./cmd/pair-go` while its `url` still points at the v1.23
  tarball, which does not contain it (verified against the real GitHub archive).
  This is the only item actively broken for anyone who isn't the operator. Fixing
  it means cutting v1.24, which needs a CHANGELOG covering ~54 issues.
- **#132** — `Alt+h` shows CLI usage whose last line says "In-session keybindings
  are on Alt+h". Regression from #99 M5c; the old text is recoverable from
  `git show 308d314^:bin/pair-shell`.
- **#128** — share escape *framing* between `termcmd` and `wrapcmd` (policy tables
  stay separate; they are in one case opposed).
- **ariadne#187** — tune the change-code gate. **ariadne#188** — allocate issue IDs
  from origin/main, plus the one-shot-sync gap.

## Process gotchas learned the hard way this session

- **Create issues from `main`, never from a branch.** `sdlc issue new` allocates
  the ID from the working tree and its sync-to-main needs a main worktree, so from
  a branch the file silently lands on that branch. Bit twice today (ariadne#188).
- **`sdlc claim` publishes a one-shot snapshot.** Anything authored after the claim
  never re-syncs, so `main` can show a stub while the branch holds the real spec.
  Happened to #129 — the operator noticed, not the tooling.
- **The plan-quality gate is stateless.** Every `change-code` re-review starts
  fresh with no memory of prior findings, so plans converge by exhaustion rather
  than agreement: 6 rounds on #127, 3 on #129. Budget for it, or use `--no-judge`
  deliberately once findings drop to Minor/Advisory (ariadne#187).
- **`make test` needs `env -u PAIR_SESSION_ID -u PAIR_TAG -u PAIR_PANE_CWD`** when
  run inside a pair session, or ambient env leaks into assertions.
- **The agent sandbox blocks `zellij action` and `pty.Start`.** Both need
  `dangerouslyDisableSandbox`; `go test ./cmd/internal/termcmd/` fails in-sandbox
  on PTY allocation alone, which is not a real failure.

## Thread arc & user model

Started as "improve user-facing docs and do a release", and the release never
happened — each step surfaced something more urgent. README was updated and landed;
then a live bug report pulled us into #127 (merged: two terminal-stream defects);
then a tab-title question opened into the zellij investigation that produced #129
and #130.

The operator is a systems thinker who **pushes back on premises rather than
details** — twice redirecting away from a workaround toward the actual constraint
("does the title and the file name need to be the same?", "we should invest if
there's other ways to customize zellij"). Both redirections were correct and
changed the design. They also caught two of my errors: an empty issue file on main,
and a pane-title prefix that duplicated the session name.

They are **cost-sensitive about SDLC ceremony** — they challenged 6 gate rounds for
126 lines of code, which produced ariadne#187. Do not silently loop on gates;
surface the cost. They prefer being told the honest trade-off and deciding
themselves over being handed a pre-narrowed option.

Design decisions they made explicitly, do not re-open: `📁` over `🚧`, no separator
after the emoji, repo = first alphanumeric token, tag in full, drop redundant tag
tokens, refuse-early rather than silently truncate.
