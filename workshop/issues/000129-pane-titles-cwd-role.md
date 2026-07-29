---
id: 000129
status: working
deps: [pair#130]
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 1.83
started: 2026-07-29T10:04:38-07:00
---

# pane titles carry only the pane role

## Problem

The outer terminal tab title is composed by zellij, not pair:

```rust
// zellij-utils/src/shared.rs — hardcoded, no config option exists
format!("\u{1b}]0;{}{}\u{07}",
    get_session_name().map(|n| format!("{} | ", n)).unwrap_or_default(),
    pane_title)
```

So the title is always `<session name> | <focused pane's title>`. The first half
is the zellij session name, which is a **socket filename** capped at **24 bytes**
(verified: 24 accepted, 25 rejected with "session name must be less than 0
characters" — zellij computes its budget minus the long macOS cache path and goes
negative). That budget is why `pair-parley_nvim-parley_nvim` (28 bytes) gets
truncated by `BuildSessionNameCandidates` down to `pair-parley_nv-parley_nv`
(exactly 24). An identifier constrained to 24 bytes makes a poor window title,
and no zellij option suppresses it — every option enumerated, docs checked,
source read. Upstream has had the request open since 2022 (zellij-org/zellij#1495
and #2088, both still open).

The second half, though, is **entirely pair's**: it is the focused pane's title,
settable at runtime via `zellij action rename-pane`, carrying none of the session
name's constraints (verified — a 70-character title containing `/`, spaces, `[]`,
`·` and an em-dash was accepted verbatim). Pair already drives it in two places
and simply never set it for the draft:

| pane | today | where |
|---|---|---|
| agent (startup) | `claude [~/workspace/pair]` | `launcher.PaneTitle` (`format.go:63`) → `PAIR_PANE_TITLE` → `main-3.kdl:54` |
| agent (steady) | `claude (464k) [~/workspace/pair]` | `titlepoller/run.go:197` |
| right terminal | `terminal 1 [a] terminal 3` | `termcmd/run.go:1037` |
| right terminal (renaming) | `[rename: work│] terminal 3` | `termcmd.renamePaneTitleLocked` (`run.go:1057`) |
| draft | *(nothing — falls through to the layout's `name="draft"`)* | `main-{2,3}.kdl` |

**Four** writers, not three, and there are also **two** cwd abbreviators:
`launcher.TildeAbbrev` (`format.go`) and `titlepoller.abbrevCwd` (`run.go:191`).

## Spec

The target tab title is:

```
📁pair          | draft
📁pair          | claude (629k)
📁pair-bugfix   | draft
```

zellij composes `<session name> | <focused pane title>`. Once #130 makes the
session half `📁{repo}-{tag}`, that half carries the folder and the tag — so the
pane half must carry **only the pane's own identity**. Anything more is the same
fact twice.

- Agent pane title becomes `claude (629k)` — the `[~/workspace/pair]` suffix is
  dropped, because `📁pair` says it.
- Draft keeps `draft`, which its layout `name=` already supplies. **No draft
  rename is needed at all.**
- Right terminal keeps its tab strip (`terminal 1 [a] terminal 3`) unchanged.
- **Depends on #130.** Stripping the cwd before `📁{repo}` exists would leave the
  repo identified nowhere: the intermediate state `pair-pair-pair | claude (629k)`
  is worse than either endpoint. This issue is the dependent tail, not the
  driver.

### Accepted consequence

The pane **frame** renders the same string as the tab title, so the agent frame
goes from `claude (629k) [~/workspace/pair]` to `claude (629k)` — the cwd leaves
the frame you look at while working and lives only in the tab title. Confirm at
the live check; if it reads as a loss, the fallback is to keep the frame long and
accept the duplication in the tab.

## Done when

- Agent pane title is `claude (629k)`; with #130 landed the tab reads
  `📁pair | claude (629k)`.
- Draft and right-terminal titles are untouched.
- `go test ./cmd/internal/titlepoller/` green and full `make test` exit 0.
- Live check in both layouts: focus each pane, read the Ghostty tab title.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

Design hours take the ×0.2 spec-quality discount: the investigation is already
done — the seam (`rename-pane`), the writer inventory, the free-form-title
verification and the 24-byte finding were all established before this block was
written. What remains is consolidation plus wiring. The design
buffer is therefore **+15%** (v2.1 Step 6: halved, because the discount already
credits the front-loaded design), not +30%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.08 impl=0.04
item: smaller-go-module design=0.10 impl=0.25
item: smaller-go-module design=0.08 impl=0.20
item: smaller-go-module design=0.10 impl=0.30
item: smaller-go-module design=0.10 impl=0.25
item: milestone-review design=0.05 impl=0.12
item: atlas-docs design=0.03 impl=0.05
design-buffer: 0.15
total: 1.83
```

Item mapping: `issue-spec` = this file; the four `smaller-go-module` rows are
(a) consolidating the two cwd abbreviators + `paneTitlePrefix` + `launcher.PaneTitle`,
(b) `titlepoller.frameTitle` + the `frameCache` invalidation,
(c) `termcmd`'s two title paths + the tag/cwd plumbing + the five exact-string
test sites on the rename path, and
(d) the draft rename across both KDLs + the re-tile survival work + the KDL guard;
`milestone-review` = the one close boundary; `atlas-docs` = the
`atlas/architecture.md` note.

Plausibility note: now comparable to #127 (est 1.40 / actual 1.40). No new
parsing and no concurrency, but four writers, two abbreviators and five
exact-string test sites is more surface than #127's single pure function. The
local trend runs toward under-estimating (#125 est 0.45 / actual 1.43), and the
unbounded risk here is the manual live check across two layouts plus the
swap-layout re-tile — which is also the item that could invalidate the design and
force the `layout_step` fallback. Row (d) and `milestone-review` carry that.

## Plan

- [x] Consolidate the two cwd abbreviators into `titlefmt` — `launcher.TildeAbbrev`
      and `titlepoller.abbrevCwd` were byte-identical. Landed in `97aacb5`, kept
      through the revert: an independent `ARCH-DRY` win, and `TildeAbbrev` still
      has a live caller in `PAIR_PANE_CWD`.
- [x] Verify a pane name survives `next-swap-layout`. **It does** — measured
      directly by renaming the live draft pane, cycling the rung, and reading it
      back. So the `nvim/init.lua` `layout_step` fallback the plan feared is
      unnecessary, and no draft rename is needed.
- [ ] **Blocked on #130.** Drop the `[<cwd>]` suffix from `titlepoller.frameTitle`
      so the agent title is `<agent> (<count>)`. One format string plus its tests
      (`titlepoller_test.go`, `run_test.go`).
- [ ] `atlas/architecture.md`: record that the tab title is
      `<session name> | <focused pane title>`, that the session half is zellij's
      and shaped by #130, and that the pane half deliberately carries only the
      pane role.
- [ ] Live check across both layouts.

## Log

### 2026-07-29

- Split out of the tab-title investigation. Evidence gathered while scoping:
  zellij's session-name limit is **24 bytes, not characters** (`🚧-` + 22 chars =
  27 bytes rejected; 21 chars = 24 bytes accepted). `BuildSessionNameCandidates`
  truncates by `[]rune` against a byte-denominated budget — a latent bug today
  for non-ASCII names, and it would become permanent under an emoji prefix.
  Recorded here because it belongs to the session-naming issue, not this one.

## Revisions

**2026-07-29 — estimate 0.89 → 1.83; scope was under-counted.** The first block
costed three title writers and one cwd abbreviator. The plan-quality gate found
**four** writers (`launcher.PaneTitle` at startup via `PAIR_PANE_TITLE`, and
`termcmd.renamePaneTitleLocked` during an `Alt+r`) and **two** abbreviators
(`launcher.TildeAbbrev`, `titlepoller.abbrevCwd`), plus two mechanisms that can
defeat the Done-when outright: the swap layouts re-declaring `pane name="draft"`
six times, and `frameCache` suppressing the very repair that would fix a reset
title. Five exact-string test sites on the rename path also have to move. Two
new plan items (abbreviator consolidation, cache invalidation) and a
verify-before-building step on the re-tile question. Recorded rather than
absorbed, so the ledger sees the correction and not just an overrun.


**2026-07-29 — scope inverted; now the dependent tail of #130.** The original
spec put `~/workspace/pair [work] · ` in front of every pane title. That was
built and committed (`97aacb5`) before the operator pointed out it duplicates
what `📁{repo}-{tag}` will carry: the target is `📁pair | draft`, not
`pair-pair-pair | ~/workspace/pair [pair] · draft`. Reverted in `ac6b879`.

What that removes: `PaneTitlePrefix` and its wiring through all four writers, the
tag/cwd plumbing into `terminalMux`, the draft self-rename in both KDLs, and the
`frameCache` invalidation item (moot — nothing re-tiles a title away, as the
swap-layout measurement showed).

What survives: the abbreviator consolidation, the swap-layout finding, and
`titlefmt`'s first test file.

Estimate is not re-derived, because the remaining work is one format string and
its tests — the block below is left as the record of what was estimated for the
larger scope, which is the honest comparison for the ledger. Actual will be
recorded against it at close.
