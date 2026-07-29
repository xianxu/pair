---
id: 000129
status: working
deps: []
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 1.83
started: 2026-07-29T10:04:38-07:00
---

# pane titles carry cwd and role

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

- Every Pair-owned pane's title leads with the abbreviated cwd and the tag, then
  its role: `~/workspace/pair [work] · <role>`.
  - agent → `~/workspace/pair [work] · claude (464k)`
  - draft → `~/workspace/pair [work] · draft`
  - right terminal → `~/workspace/pair [work] · terminal 1 [a] terminal 3`
- The tag is included even though it is currently redundant with the session-name
  prefix. That prefix is the half we do not control; if zellij#1495 ever lands,
  the pane title becomes the WHOLE title and the tag has to already be there.
  Transitional redundancy is the deliberate trade.
- Every pane must carry a correct title at all times, not only the focused one —
  zellij emits on focus change, so a stale title anywhere is user-visible the
  moment focus lands on it.
- **One** cwd-shortening rule and **one** prefix builder for all four writers.
  `launcher.TildeAbbrev` and `titlepoller.abbrevCwd` are today two
  implementations of the same idea; consolidate before adding a third consumer,
  or this change institutionalises the drift (`ARCH-DRY`).
- The right terminal shows the **session's repo root** (`PAIR_PANE_CWD`, static),
  not the shell's live cwd. A user who `cd`s inside that pane does not move the
  title — it identifies the workbench, not the shell.
- The draft's cwd is the repo root and never changes, so a one-shot rename at
  startup is the right shape *if* it survives the swap-layout re-tile — which
  must be verified, not assumed (see Plan). Precedent for the idiom: both the
  terminal pane (`termcmd/run.go:44`) and the agent pane (`main-3.kdl:54`)
  already self-rename from their layout command lines.
- Out of scope, deliberately: the session-name half. Fixing it needs either
  upstream zellij#1495 or a session-naming change (drop the redundant repo
  segment, byte-based rather than rune-based truncation), and that also touches
  `pair list`, cmux and `zellij list-sessions` — a different blast radius, so a
  different issue.

## Done when

- Every Pair-owned pane shows `~/workspace/pair [work] · …` — agent (at startup
  AND in steady state), draft, and the right terminal (idle AND mid-`Alt+r`
  rename) — verified live by focusing each in turn and reading the Ghostty tab
  title.
- The draft pane's title survives a draft-height rung change (`Alt+↑`/`Alt+↓`),
  which re-tiles through swap layouts and could otherwise reset a
  layout-supplied name.
- Layout 2 (no terminal pane) is covered as well as layout 3.
- `go test ./cmd/internal/titlepoller/ ./cmd/internal/termcmd/` green, and full
  `make test` exit 0.

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

Single review boundary — no `Mx` tags: one branch, one close.

- [ ] **Consolidate the cwd abbreviators first, in `cmd/internal/titlefmt`.**
      `launcher.TildeAbbrev` and `titlepoller.abbrevCwd` both shorten a cwd.
      `titlefmt` is the right home: it already exists and is **already imported
      by both** (`launcher/format.go:7`, `titlepoller/titlepoller.go:20`).
      Putting it in `launcher` would force `termcmd` to import a heavy IO-ish
      package for a string function. Doing this *before* adding consumers is the point — adding
      a third caller to two implementations is how the divergence in #127 got
      made (`ARCH-DRY`).
      Test: one table, both former call sites' cases folded in.
- [ ] **`paneTitlePrefix(cwdDisplay, tag) string`** — the single place the
      `~/workspace/pair [work] · ` shape is built. Pure, no IO (`ARCH-PURE`).
      Test: `TestPaneTitlePrefix`. Adversarial input is degenerate — empty tag,
      empty cwd, both empty — where the risk is a dangling separator or a
      leading `·`, so assert exact strings. Include an **empty role**: both
      `paneTitleLocked` (`run.go:1043`) and `renamePaneTitleLocked`
      (`run.go:1058`) return `""` on zero tabs and callers skip the rename on
      `""` (`run.go:1029`), so the prefix must be applied AFTER that guard or a
      tab-less mux writes a bare `~/workspace/pair [work] · `.
- [ ] **All four writers take the prefix**, so no pane can show the old shape:
      - `launcher.PaneTitle` (`format.go:63`) — the **startup** title exported as
        `PAIR_PANE_TITLE`. Missing this leaves the agent pane on the old shape
        for up to one 60s poll on every launch — visible at exactly the moment
        the user first looks at the tab.
      - `titlepoller.frameTitle` — steady state.
      - `termcmd.paneTitleLocked` (`run.go:1042`) — the tab strip. The active-tab
        bracketing (`[a]`) must be unchanged; that is the regression risk.
      - `termcmd.renamePaneTitleLocked` (`run.go:1057`) — the rename-mode title
        goes through the same `setPaneTitle`, so without the prefix the tab
        title *loses* the cwd for the duration of an `Alt+r`, contradicting
        "correct at all times".
      Test: extend the existing tables. **Eleven** exact-string sites must move,
      not five: `termcmd/run_test.go:484, 492, 595, 606, 668, 698, 800, 853`,
      plus `launcher/format_test.go:61` (`TestPaneTitle`) and
      `titlepoller/titlepoller_test.go:41, 51` (`TestAbbrevCwd`,
      `TestFrameTitle`). This is the bulk of the churn.
- [ ] **Plumb tag + cwd into `terminalMux` at the boundary** — read once in
      `runShell`/`OSRuntime` from `PAIR_TAG` / `PAIR_PANE_CWD` and store as
      struct fields. NOT `os.Getenv` inside `paneTitleLocked`: that would make
      the title path env-dependent and force the existing table tests to mutate
      process state (`ARCH-PURE`).
      Test: a named assertion that the title path reads the struct fields, not
      process env — that testability IS this item's justification.
      Note `PAIR_PANE_CWD` is **already** tilde-abbreviated (`createflow.go:445`
      sets it via `TildeAbbrev`), so do not run the abbreviator over it again.
- [ ] **Draft self-rename**, mirroring the agent pane's existing idiom at
      `main-3.kdl:54` (`zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID"
      …`), in **both** `main-2.kdl` and `main-3.kdl`.
- [ ] **Survive the re-tile — this is the item that can invalidate the design.**
      Every swap layout re-declares `pane name="draft"` (6 occurrences in
      `main-3.kdl`). If zellij reapplies that name on `Alt+↑`/`Alt+↓`, a
      startup-only rename *structurally cannot* meet the Done-when. Verify this
      first, before writing the rename; if the name is reapplied, the fallback is
      a re-rename from `nvim/init.lua`'s `layout_step` (`init.lua:3485`), which
      already fires on every rung change and already shells to `pair layout …`.
      Decide from observation, not assumption.
- [ ] **Invalidate `frameCache` on rung change.** `titlepoller.go:114-124` skips
      the rename when the computed title equals the last one *written*. If a
      re-tile resets the agent pane to `agent`, the cache still believes nothing
      changed and never repairs it — a stale title on the pane the user looks at
      most. The old spec tolerated this; "correct at all times" does not.
- [ ] `atlas/architecture.md`: record which process owns each pane title, that
      there is now one prefix builder and one cwd abbreviator, and that the
      session-name half of the tab title is zellij's and unreachable
      (zellij#1495).
- [ ] Live check: focus each pane and read the Ghostty tab title, in **both**
      layouts; then `Alt+↑`/`Alt+↓` and confirm every title survives the
      swap-layout re-tile.

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
