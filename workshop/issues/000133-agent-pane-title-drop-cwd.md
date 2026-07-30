---
id: 000133
status: codecomplete
deps: []
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 1.60
started: 2026-07-29T15:42:55-07:00
actual_hours: 0.90
---

# agent pane title drops the cwd suffix

## Problem

zellij composes the outer terminal tab title as `<session name> | <focused pane
title>` (hardcoded in `zellij-utils/src/shared.rs`; no config option, upstream
request open since 2022 — zellij-org/zellij#1495, #2088).

#130 made the session half carry the folder and the tag: `ComposeSessionName`
(`session_index.go:75`) emits `📁{repo}` plus any residual tag tokens. Verified
live — `PAIR_SESSION_NAME=📁pair`, and `zellij list-sessions` shows `📁pair`,
`📁ariadne`, `📁kbench`.

The agent pane still appends `[<tilde-cwd>]`, so the tab reads:

```
📁pair | claude (629k) [~/workspace/pair]
```

The repo is named twice. Three of the five Pair-owned surfaces already match the
target — only the agent pane is off:

| pane | writer | title today | on target? |
|---|---|---|---|
| agent (startup) | `launcher.PaneTitle` `format.go:61` → `PAIR_PANE_TITLE` → `main-{2,3}.kdl` | `claude [~/workspace/pair]` | no |
| agent (steady) | `titlepoller.frameTitle` `titlepoller.go:72` | `claude (629k) [~/workspace/pair]` | no |
| draft | layout `name="draft"` | `draft` | yes |
| right terminal | `termcmd.paneTitleLocked` `run.go:1042` | `terminal 1 [a] terminal 3` | yes |
| right terminal (`Alt+r`) | `termcmd.renamePaneTitleLocked` `run.go:1057` | `[rename: work│] terminal 3` | yes |

**Two** writers emit the suffix, not one. The startup writer matters on its own:
`PAIR_PANE_TITLE` is what the pane is named at launch, so leaving it in place
shows the old shape from launch until the poller's first pass.

## Spec

Target tab title:

```
📁pair        | claude (629k)
📁pair-bugfix | claude (629k)
📁pair        | draft
```

The session half carries the folder and tag, so the pane half carries **only the
pane's own identity**. Anything more is the same fact twice.

- `titlepoller.frameTitle` → `<agent> (<count>)`, or `<agent>` when no count
  resolved. Drop the `[%s]`.
- `launcher.PaneTitle` → `<agent>`. Drop the `[<tilde-cwd>]`.
- **Delete BOTH cwd abbreviators, and the whole `cwd_display` chain with them.**
  No pane title shows a cwd any more, so no cwd *formatting* should exist
  anywhere — that is the whole `ARCH-DRY` resolution, not a rehousing (see the
  Revisions entry for how this replaced "consolidate into `titlefmt`").
  Traced end to end:
  - `titlepoller.abbrevCwd` (`titlepoller.go:57`) — sole non-test caller is
    `run.go:194`, the empty-`cwd_display` fallback feeding `frameTitle`.
  - `cwd_display` in `pane-*.json` — sole reader is `PaneInfo.CwdDisplay`
    (`run.go:32`, decoded `runtime.go:74`), consumed only at `run.go:192`.
  - `PAIR_PANE_CWD` — sole writer `createflow.go:469`; sole readers the two KDL
    printfs (`main-2.kdl:45`, `main-3.kdl:54`) that emit `cwd_display`. Nothing
    else in the tree references it: no Go, no lua, no shell, no atlas or docs
    contract.
  - `launcher.TildeAbbrev` (`format.go:47`) — sole non-test caller is that same
    `createflow.go:469` line. So it dies too; the earlier claim that it "keeps a
    live caller" was true only because its caller existed to feed the thing being
    deleted.
- **The raw `cwd` field STAYS.** It has two live readers that are nothing to do
  with titles, and both want the unabbreviated path: `contextcmd.paneCwd`
  (`contextcmd.go:74`) and `launcher.legacyPaneAgentForScope` (`legacy_live.go:10`,
  which feeds `pathWithinRoot(meta.CWD, scope.Root)`). Removing `cwd_display`
  therefore costs no capability — any consumer wanting the pane's cwd reads `cwd`
  (or `$PWD`, which the pane has anyway); what goes is a redundant pre-formatted
  copy.
- Out of scope: the draft pane (already correct), the right terminal (already
  correct), and the session half (zellij's, and #130 settled it).

### Accepted consequence

The pane **frame** renders the same string as the tab title, so the agent frame
goes from `claude (629k) [~/workspace/pair]` to `claude (629k)` — the cwd leaves
the frame you look at while working and lives only in the tab title. Confirm at
the live check; if it reads as a loss, the fallback is to keep the frame long and
accept the duplication in the tab.

## Done when

- Agent pane title is `claude (629k)` at startup AND in steady state; the tab
  reads `📁pair | claude (629k)`. **Steady state observed live in layout 3**
  (read back via `zellij action list-panes --json`). Layout 2 and the startup
  title are covered mechanically rather than by observation — see the Revisions
  entry recording why that is sufficient.
- Draft and right-terminal titles are unchanged, including mid-`Alt+r`.
- No cwd-formatting function remains in the tree: `abbrevCwd`, `TildeAbbrev` and
  both their tests are gone, and `grep -rn 'TildeAbbrev\|abbrevCwd'` returns
  nothing. `PAIR_PANE_CWD` and `cwd_display` are gone from both KDLs and from
  `PaneInfo`.
- After a `make build`, `grep -rn 'cwd_display\|PAIR_PANE_CWD' zellij/
  cmd/internal/runtimebundle/assets/` is empty — i.e. the **regenerated embedded
  bundle** is clean too, not just the source layouts. This is the one surface the
  live check structurally cannot cover (the dev session reads the repo KDL via
  `defaultPairHome`), so it gets its own assertion. Use `grep -rn --no-ignore` or
  `ls` the asset path: the assets dir is gitignored, and an ignore-respecting
  grep silently reports zero hits for a file that still contains the string.
- The raw `cwd` round-trip is proven by **executing the KDL producer**, not by
  hand-written JSON: the new conformance test (Plan item 1) runs the real
  `args "-c"` line from both layouts and asserts `contextcmd.paneCwd` returns the
  cwd from what it wrote. Existing tests staying green is explicitly NOT accepted
  as proof here — all of them hand-write `pane-*.json` and never execute the
  producer.
- `pane-<tag>-<agent>.json` contains exactly **one** JSON object after the edit
  (the printf-recycling regression is pinned, not just avoided).
- `go test ./cmd/internal/titlepoller/ ./cmd/internal/launcher/` green, and full
  `make test` exit 0 (run with `env -u PAIR_SESSION_ID -u PAIR_TAG` — the
  in-session env leaks into review-target and changelog tests).

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale (recalibration tracked in #127), so the number is provisional but
uses the required method.

Design hours take the ×0.2 spec-quality discount across every primitive: the
investigation is done and in the Problem/Spec above — the two writers are
enumerated, the whole dead chain is traced to its last reader with file:line, and
the conformance-test approach is specified down to the PATH-stub seam. What
remains is execution. Per v2.1 Step 6's rule of thumb, applying ×0.2 broadly means
the spec is already credited for that front-loading, so the design buffer is
halved to **+15%** rather than +30%. Implementation hours are written at 40% of
the v2 primitive table per v3.1 rule 5; `familiarity: 1.0` (same package set
touched twice today).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.15 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.03 impl=0.10
item: cross-cutting-refactor design=0.10 impl=0.14
item: cross-cutting-refactor design=0.10 impl=0.16
item: atlas-docs design=0.06 impl=0.12
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 1.60
```

Arithmetic: design 0.52 + impl 1.00 + buffer (0.52 × 0.15 = 0.078) = **1.598 → 1.60**.

Item mapping, in Plan order:
- `issue-spec` — this file, including three `## Revisions` entries and **three**
  plan-quality rounds (see the gate ledger). Design 0.75×0.2 (upper-mid of the
  0.5–1.5 band: the file carries a writer-inventory table and a full chain trace,
  not a thin spec); impl 0.2×0.4 = 0.08, mid band rather than the 0.1 floor, for
  the same reason.
- `smaller-go-module` (0.06/0.20) — Plan item 1, the KDL producer conformance
  test. Impl at the **top** of the 0.2–0.5 band before scaling: it shells out,
  stubs two binaries on `PATH`, and tables over two layouts. The single largest
  item, and the only one adding code rather than removing it.
- `smaller-go-module` (0.03/0.10) — Plan item 2, `frameTitle` plus its two test
  sites. A one-line format change; low band.
- `cross-cutting-refactor` (0.10/0.14) — Plan item 3, `abbrevCwd` and the dead
  thread behind it across 5 files (`run.go`, `runtime.go`, `runcli.go`,
  `titlepoller_test.go`, `run_test.go`).
- `cross-cutting-refactor` (0.10/0.16) — Plan items 4+5, `TildeAbbrev` +
  `PAIR_PANE_CWD` + the degenerate `PaneTitle` + both KDL printfs + the inlined
  `SetEnv`. Slightly higher impl than item 3 because it touches non-Go surface
  (KDL) where the compiler cannot catch a mistake — which is exactly why item 1
  exists.
- `atlas-docs` — three stale lines at `:274`/`:276`/`:278` plus the new content,
  the side-quest fix, and the `Makefile.local:79` comment that also documents the
  old frame shape. Priced as **two** touches at mid band (0.15×0.2 = 0.06 design,
  0.15×0.4 = 0.12 impl), following the v2 table's own anchor row, which multiplies
  the 0.05–0.2 band per touch (charon #13 `M7 (docs ×3) | 0.15–0.6 | 0.15–0.6`).
- `milestone-review` — the single close boundary AND the manual live check in both
  layouts. Impl at the top of the 0.2–0.5 band: 0.5×0.4 = **0.20**. The live check
  needs a session relaunch to observe the *startup* title, not just the steady
  one, and repeats in layout 2.

Plausibility note: lands next to #127 (est 1.40 / actual 1.40), which is the right
neighbour — comparable file count, no new parsing, no concurrency. It is well
below #129's 1.83 for the reverted prefix scope, which is correct: four writers
gaining a prefix is more work than two writers losing a suffix.

Signals in both directions, including the one that cuts hardest against this
number. **Against (over-estimate):** the immediate predecessor in this same
tab-title thread is `pair#130` — est 4.31, actual 1.78, **ratio 2.42**
(`calibration-ledger.tsv:333`, closed today, same packages, same v3.1 model). That
is the most adjacent row in the ledger and it says this method currently
over-estimates work in exactly this area. **For (under-estimate):** #125 (est 0.45
/ actual 1.43), and the general local trend. #129's measured 0.22h is a weak
signal in the over-estimate direction — undercounted by construction, so it is
cited but not leaned on. Net: 1.60 is retained rather than trimmed toward the #130
ratio, because #133 is a far smaller issue than #130 and one adjacent row is not a
recalibration; but if the actual lands near 0.6–0.7 that is the #130 ratio
repeating, and it should feed #127's recalibration rather than be explained away.

The two named risks — the `PATH`-stub conformance test possibly needing a second
shape, and the live check's session relaunches — both sit on the **impl** side,
which by construction gets no buffer (v2 Step 6 buffers design only). Top-of-band
is the primitive's ceiling, not slack: if item 1 needs a second iteration it exits
the primitive rather than eating headroom. This issue is also a clean v3.1
validation row, since nothing here fans out — a sequential deletion sweep plus a
manual live check means estimate and `sdlc actual` should converge tightly.

## Plan

Single review boundary — no `Mx` tags: one branch, one close.

- [x] **Guard the KDL producer FIRST, before editing it** —
      `cmd/internal/contextcmd/panejson_kdl_test.go`, table-driven over
      `main-2.kdl` and `main-3.kdl`. Extract the agent pane's `args "-c"` line
      from the layout (the `termcmd/run_test.go:612` idiom already reads a KDL
      from a Go test), then run it under `sh -c` with a temp `PAIR_DATA_DIR`,
      a set `ZELLIJ_PANE_ID`/`PAIR_TAG`/`PAIR_AGENT`, and stub `zellij` + `pair`
      executables on `PATH` so the trailing `exec pair wrap` is a no-op. Assert:
      (a) `contextcmd.paneCwd(dataDir, tag, agent)` returns the cwd — the real
      consumer, closing the loop the current Done-when only claimed;
      (b) the file is **exactly one** JSON object with a non-empty `pane_id`.
      This is what catches the printf failure mode: drop a `%s` and leave its
      argument and shell printf **recycles the format**, emitting two
      concatenated objects — `Unmarshal` then fails, the poller skips the pane,
      `paneCwd` returns `""`, and every existing test stays green because they
      all hand-write the JSON. Write it against the CURRENT printf (it must pass
      before the edit), so it is a guard and not a post-hoc rationalisation.
      `ARCH-MOCK`: this is the producer's missing conformance check — the fake
      binaries sit on the same `PATH` seam the real launch uses.
- [x] Drop `[%s]` from `titlepoller.frameTitle` (`titlepoller.go:72`) → `<agent>
      (<count>)` / `<agent>`. Update `TestFrameTitle` (`titlepoller_test.go`) and
      the `run_test.go` title assertions.
- [x] Delete `abbrevCwd` + `TestAbbrevCwd` and the `run.go:192-194` cwdDisp
      block. Then follow the now-dead thread all the way out, since Go does not
      flag unused struct fields: `PaneInfo.Cwd` **and** `.CwdDisplay`
      (`run.go:31-32`), their decode (`runtime.go:73-74, :84-85`), and the
      `Home` thread that only fed `abbrevCwd` — `Options.Home` (`run.go:15`),
      `runcli.go:34`, `run_test.go:91`. The JSON `cwd` **key** stays; it is
      `PaneInfo`'s copy of it that goes.
- [x] Delete `launcher.TildeAbbrev` + `TestTildeAbbrev` and the `PAIR_PANE_CWD`
      export at `createflow.go:469`; drop `cwd_display` from the printf in
      **both** `main-2.kdl:45` and `main-3.kdl:54` — removing the `%s` *and* its
      `"${PAIR_PANE_CWD:-$PWD}"` argument together — keeping `pane_id` and `cwd`.
      Re-run item 1's guard: it must still pass. Verify no launcher test asserts
      `PAIR_PANE_CWD` is set.
      **Four files carry this printf, but only two are the source.**
      `zellij/layouts/main-{2,3}.kdl` is the source of truth;
      `cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-{2,3}.kdl`
      are generated copies (`Makefile.local:92-95`) embedded via `embed.go:8`.
      Never hand-edit the generated pair. Correcting the gate's advisory: they are
      **not** committed — `.gitignore:34` blanket-ignores
      `/cmd/internal/runtimebundle/assets/` and `git log --all` shows they were
      never tracked — so there is nothing to commit and no stale-asset-in-git
      failure mode. The real gap is that `make build` regenerates them while the
      **live check cannot see them**: `defaultPairHome=$(CURDIR)` makes the dev
      session read the repo KDL, so a stale bundle would pass every check we run.
      Hence the explicit Done-when grep below covers the generated tree too.
- [x] **`launcher.PaneTitle` becomes an identity function — delete it rather than
      keep it.** With the suffix gone it would return `agent` from
      `PaneTitle(agent, cwd, home)` with two unused params, and `TestPaneTitle`
      would assert identity. Inline `rt.SetEnv("PAIR_PANE_TITLE", agent)` at
      `createflow.go:468` and delete both function and test — the same `ARCH-DRY`
      reasoning applied to `TildeAbbrev` above, applied consistently.
- [x] `atlas/architecture.md`, naming the stale lines: `:274` lists "cwd abbrev"
      among the poller's pure decisions (now none exist); `:276` documents the
      frame as `<agent> (<count>) [<cwd>]`, the record as
      `{pane_id, cwd, cwd_display}`, and still says `main.kdl`. Rewrite all
      three, and record that the tab title is `<session name> | <focused pane
      title>` with the session half zellij's (shaped by #130's `📁{repo}-{tag}`)
      and the pane half carrying only the pane role, and that `pane-*.json`'s raw
      `cwd` exists for `pair context` + legacy scope matching, not display.
- [x] `side-quest:` fix `atlas/architecture.md:278`'s reference to
      `cmd/internal/titlefmt` — pre-existing #130 drift pointing at a package it
      deleted. One line, in this branch, called out as a side quest rather than
      left for someone to trip over.
- [x] Live check — done against the running session via
      `zellij action list-panes --json` (a real read-back, not an eyeball claim).
      Operator confirmed the shortened frame is acceptable, so the
      keep-the-frame-long fallback is not needed.

## Log

### 2026-07-29
- 2026-07-29: closed — Agent pane title drops the cwd; verified live via zellij action list-panes --json after respawning the poller from the new binary: agent "claude (246k) [~/workspace/pair]" -> "claude (255k)", draft and terminal unchanged, tab reads "📁pair | claude (255k)". New KDL producer conformance test (cmd/internal/contextcmd/panejson_kdl_test.go) written to pass pre-edit then mutation-tested: dropping the %s while leaving its argument made shell printf recycle the format and emit two concatenated JSON objects, caught by the exactly-one-object assertion. PAIR_PANE_TITLE had zero test coverage before; added a mutation-checked assertion. Whole dead cwd chain removed (abbrevCwd, TildeAbbrev, PaneTitle, PAIR_PANE_CWD, cwd_display in both KDLs, PaneInfo.Cwd/.CwdDisplay + decode, Options.Home); raw cwd key kept for contextcmd.paneCwd and legacy scope matching, with TestPaneCwdToleratesLegacyCwdDisplayField pinning the pre-#133 upgrade path. Regenerated embedded bundle verified clean with --no-ignore. make test exit 0 (termcmd/wrapcmd pty tests need the sandbox disabled; both packages untouched).; review verdict: FIX-THEN-SHIP

- Supersedes #129, which is `wontfix`. #129 opened with the inverse spec — put
  `~/workspace/pair [work] · ` in FRONT of all four pane titles — built it
  (`97aacb5`), then reverted (`ac6b879`) once the operator pointed out it
  duplicates what `📁{repo}-{tag}` carries. Its re-scope commit (`33845c7`)
  already described this issue's spec. Branch abandoned rather than rebased;
  history preserved at tag `abandoned/000129-prefix-scope`.
- **Two findings carried forward from #129, both worth not re-deriving:**
  - A pane name **survives `next-swap-layout`** — measured directly by renaming
    the live draft pane, cycling the rung with `Alt+↑`/`Alt+↓`, and reading it
    back. So the six `pane name="draft"` re-declarations in `main-3.kdl` do not
    reset a runtime rename, no draft self-rename is needed, and the
    `nvim/init.lua` `layout_step` fallback #129's plan feared is unnecessary.
  - Consequently `frameCache` (`titlepoller.go:114-124`) needs no invalidation:
    nothing re-tiles a title away, and the poller respawns per session entry, so
    there is no long-lived stale-cache window.
- Why #129's `titlefmt` consolidation is not carried forward: #130 **deleted**
  `cmd/internal/titlefmt` outright when it retired `EmojiTitle` (that function
  was the package's only content). Reviving the package to host `TildeAbbrev`
  would add an indirection for a single caller — hence deletion of the duplicate
  instead.

- **Implemented.** All plan items done except the live check (below). Evidence:
  - The producer guard was written FIRST and verified to pass against the
    unmodified KDL, then **mutation-tested**: dropping the `cwd_display` `%s`
    while leaving its argument made printf recycle the format, and the test caught
    it exactly as predicted — `paneCwd = ""` plus a second JSON object
    (`{"pane_id":"7","cwd":"…"}{"pane_id":"…","cwd":""}`). The KDL was then edited
    correctly (`%s` **and** argument removed together, leaving 2 verbs / 2 args)
    and the guard passes again.
  - `PAIR_PANE_TITLE` had **no test at all** in the tree, so the startup title —
    half the Done-when — was unguarded. Added an assertion in
    `TestRunLaunchForcedCreateClaude` covering both the new value and the absence
    of `PAIR_PANE_CWD`, and mutation-checked it (injecting `claude [~/oops]` fails
    it).
  - Dead thread removed in full: `abbrevCwd`, `TildeAbbrev`, `PaneTitle`,
    `PaneInfo.Cwd`, `PaneInfo.CwdDisplay`, their JSON decode, `Options.Home`, the
    `runcli.go` wiring, and `TestUpdateFrameTitlesCwdFallback` (whose subject no
    longer exists). `grep -rn 'TildeAbbrev|abbrevCwd'` is empty.
  - Embedded bundle verified regenerated and clean: the generated copies were
    stamped 14:44 carrying `cwd_display`, and after the build they are 16:14 with
    `printf '{"pane_id":"%s","cwd":"%s"}'`. Checked with `--no-ignore`, since the
    assets dir is gitignored and a plain grep reports zero hits misleadingly.
  - `make test` **exit 0**. Note two tests (`termcmd` pty, `wrapcmd` re-exec) fail
    under the agent's command sandbox with `pty.Start: operation not permitted` and
    pass outside it; both packages are untouched by this change.
- **Decision recorded — `cwd_display` chain deleted, raw `cwd` kept.** Deleting it
  costs no capability: `contextcmd.paneCwd` and `legacyPaneAgentForScope` both read
  the raw `cwd`, which stays. New test `TestPaneCwdToleratesLegacyCwdDisplayField`
  pins the upgrade path — a pane record written by a pre-#133 session still
  resolves, since such a file survives on disk when the binary updates under it.
- Kept `tests/copy-on-select-test.sh`'s agent-pane title as a legacy/hostile
  fixture rather than modernizing it: its whole point is that nvim detection keys
  on `terminal_command`, not the title, so a bare `claude` would make the case pass
  trivially. Comment updated to say so.
- Pre-existing, not touched: `gofmt -l` flags `launcher/lifecycle_test.go` and
  `launcher/pick_test.go`; both are unmodified by this branch.

- **Live check (layout 3, this session).** Killed the running poller and
  respawned it from the new binary (`pair title <tag> <agent> <session>` — the
  real spawn shape), then read titles back with
  `zellij action list-panes --json`:

  | pane | before | after | |
  |---|---|---|---|
  | agent | `claude (246k) [~/workspace/pair]` | `claude (255k)` | changed |
  | terminal | `[terminal 1]` | `[terminal 1]` | unchanged |
  | draft | `draft` | `draft` | unchanged |

  Tab title now `📁pair | claude (255k)`. This exercised the real code path
  (`updateFrameTitles` → `RenamePane` → zellij), not a simulation. Pidfile
  correctly re-pointed to the new poller (49647), so `ensure_title_poller` keeps
  working on the next entry.

  **Scope of that check, stated precisely rather than over-claimed:** layout 3
  only. Layout 2 is covered transitively — `tests/term-pane-shortcuts-test.sh`
  pins that both layouts share the same agent+draft launch commands (re-verified
  by hand here), and the new conformance test runs BOTH layouts' agent lines. The
  *startup* title (`PAIR_PANE_TITLE`) was not observed in a fresh session, because
  that needs launching a new one. It is covered instead by the mutation-tested
  `TestRunLaunchForcedCreateClaude` assertion plus — after the close review's I-2 —
  an **argv assertion** on the KDL's `rename-pane` hop. The original wording here
  credited the conformance test with "executing the KDL line that consumes
  `${PAIR_PANE_TITLE:-agent}`", which was an over-claim: executing is not
  asserting, and the `zellij` stub discarded argv. The stub is now a recording
  fake, verified by mangling the expansion in `main-3.kdl` — argv came back as
  `agent` instead of `claude` and the test failed, where previously the whole tree
  stayed green. The `Alt+r` rename path
  is untouched code with existing table tests.
- `zellij action …` needs the command sandbox disabled — inside it the socket
  connect fails as "There is no active session!" while `zellij list-sessions`
  (a directory read) still works, which is a misleading pair of symptoms worth
  remembering.
- Operator decisions this turn: `PAIR_PANE_CWD` removal **approved**; the
  shortened agent frame **approved**.

## Revisions
**2026-07-29 — close review (FIX-THEN-SHIP): two Important findings fixed in the
close commit.** `README.md:91` was the last hand-maintained restatement of the old
frame shape — the shadow-sweep had covered `createflow.go`, `frameTitle`, both
KDLs, the generated bundle, `PaneInfo`, `atlas/architecture.md` and
`Makefile.local:79`, but missed the surface a *user* reads (`ARCH-PURPOSE`). And
the conformance test's `zellij` stub discarded argv, so the `rename-pane` half of
the KDL line ran unasserted (`ARCH-MOCK`): mangling `${PAIR_PANE_TITLE:-agent}`
kept every test in the tree green. Both fixed, the second mutation-verified. Also
took the cheap Minors: the `titlepoller` package doc contradicted `frameTitle` 58
lines below it; the decoded-but-unasserted `Cwd` field now asserts; and the legacy
`claude [~/workspace/parley.nvim]` fixtures in `zellijpane_test.go` and
`clipcmd/run_test.go` are now labelled deliberate rather than reading as stale.

Two documentation corrections rather than re-verification, per the review's
recommendation: Done-when asked for a live check in **both** layouts, but only
layout 3 was observed. Layout 2's agent line is byte-identical to layout 3's —
pinned by `tests/term-pane-shortcuts-test.sh` and re-verified by hand — and the
conformance test executes **both** layouts' lines, so the mechanical coverage is
equivalent for the pane-title path; the difference between the layouts is the
presence of a terminal pane, which this change does not touch. Done-when now says
that instead of implying an eyeball that did not happen.

**Estimate outcome: est 1.60 / actual 0.90, ratio 1.8× over.** The plausibility
note called this: it flagged `pair#130`'s 2.42× over-estimate as the most adjacent
ledger row, predicted that "if the actual lands near 0.6–0.7 that is the #130 ratio
repeating", and said it should feed #127's recalibration rather than be explained
away. 0.90 is close to that prediction, and it is the second consecutive
same-area over-estimate under v3.1 — evidence the impl-hour scale is still high
for deletion-shaped work in this repo, which is exactly the signal #127 needs.


**2026-07-29 — the cwd chain is deleted whole; both abbreviators go, not one.**
Authored spec said "delete `abbrevCwd`; `TildeAbbrev` stays, it has a live caller
at `createflow.go:469` (`PAIR_PANE_CWD`)", and left the `cwd_display`
disposition as an open decision for implementation time. Tracing the chain before
running `change-code` settled it and inverted the conclusion: `PAIR_PANE_CWD`'s
only readers are the two KDL printfs that emit `cwd_display`, whose only reader
is the `frameTitle` call being deleted. So `TildeAbbrev`'s "live caller" existed
solely to feed the thing this issue removes — it is dead by transitivity, and the
honest `ARCH-DRY` end state is that **no cwd-formatting function survives**.

The raw `cwd` field is the load-bearing part and stays: `contextcmd.paneCwd` and
`legacyPaneAgentForScope` both read it, both want it unabbreviated. So the
deletion costs no capability — it removes a redundant pre-formatted copy.

Scope delta vs authored: two more deletions (`TildeAbbrev`, `PAIR_PANE_CWD`) and
two KDL edits. Recorded before the estimate is derived, so the estimate costs the
real scope rather than being revised after.

**2026-07-29 — plan-quality round 1: producer guard added, three dead threads
followed further.** The gate blocked on a real silent-corruption path the plan had
walked into: the Done-when credited three existing tests with proving the raw-`cwd`
round-trip, but all three hand-write `pane-*.json` and never execute
`main-{2,3}.kdl`, so a mis-edited printf (format `%s` dropped, argument left)
would recycle the format, emit two concatenated objects, and fail decoding
**silently with `make test` green**. Plan item 1 is now an executable conformance
test of the producer, written to pass BEFORE the edit. Also folded in: the dead
thread runs further than `opts.Home` (it reaches `PaneInfo.Cwd`, its decode, and
`Options.Home`/`runcli.go`, none of which Go will flag); `PaneTitle` degenerates
to an identity function and should be deleted by the same `ARCH-DRY` argument used
against `TildeAbbrev`; and the atlas step now names `:274`/`:276`/`:278` with the
pre-existing `titlefmt` drift called out as an explicit side quest.

**Operator-visible consequence worth a veto:** `PAIR_PANE_CWD` is currently
exported into every agent pane's environment. It is undocumented and unused by
anything in the tree, but a personal script outside the repo could read it. Any
such script can read `cwd` from `pane-<tag>-<agent>.json`, or just `$PWD`. Flagged
rather than assumed.
