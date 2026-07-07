---
id: 000104
status: working
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 9
started: 2026-07-06T16:05:29-07:00
---

# Fold pair repo binaries into a single pair Go program

## Problem

Dev builds compile ~19 Go binaries; `pair-dev` re-runs `make build` on every
launch and every in-session restart, so the whole link cost is paid constantly.
`bin/pair` and `bin/pair-go` are the same program linked twice into two 81 MB
files, and `pair` embeds 17 helper binaries in its runtime bundle (~55–65 MB of
its 81 MB). The many `pair-*` names on PATH also create "which do I call?"
confusion. Only one caller is external — `pair-scribe`, wired into the user's
`~/.zshrc` — and that stays maintained separately.

## Spec

Collapse **every** pair-repo Go binary into a single `pair` program reached as
`pair <subcommand>`. The only other artifacts are the three shell shims
(`pair-dev`, `pair-help`, `pair-notify`). `pair-scribe` folds in too — the
user's `~/.zshrc` moves from `exec pair-scribe` to `exec pair scribe`. The
subcommand surface already exists (`dispatcher.Families()` + streaming seam,
argv[0] dispatch in `entrypoint.ClassifyInvocation`) and covers 8 helpers; every
remaining helper is a thin shim over a ready `RunXxxCLI` seam. Finish + nest the
surface, rewrite every call site **we own** (launcher Go, clipcmd, distiller,
nvim Lua, `.claude` allowlist, zellij KDL + bundle mirror) to `pair <sub>`, then
collapse the build and stop bundling helper binaries.

Reorg (nest families, keep member names): `pair review target|open|readiness`,
`pair scrollback render|open`, `pair changelog render|open`,
`pair clip copy-on-select|clipboard-to-pane|flash-pane`; the rest stay flat.

The runtime bundle keeps expanding **config/assets only** — never binaries,
because a binary is always reachable as `pair <sub>`. `pair` is already on the
session PATH (inherited from the launching shell; `pathenv.go` prepends, doesn't
replace) + a `dir(os.Executable())` prepend for robustness — no symlink bridge,
nothing written into the content-addressed store. Single-sources the binary set
(today restated in 5 places) onto `dispatcher.Families()` — ARCH-DRY.

Design: `workshop/plans/000104-single-pair-binary-plan.md`.

## Done when

- `make build` links one `pair` binary; `bin/pair-go` is gone.
- `pair <sub>` reaches every former helper (nested where grouped); every in-repo
  call site invokes `pair <sub>` (or self-exec), not a standalone `pair-*` name.
- The runtime bundle embeds no helper binaries (config + shims only); `bin/pair`
  measurably shrinks from 81 MB (target ~20–25 MB).
- Only remaining artifacts: `pair`, `pair-dev`, `pair-help`, `pair-notify`
  (+ at most a `pair-slug` symlink pending Stop-hook verification).
- `~/.zshrc` runs `pair scribe`; `make test` green; live-session smoke of agent
  pane, copy-on-select, `pair scribe`, and the scrollback/changelog keybinds.

## Plan

Durable plan: `workshop/plans/000104-single-pair-binary-plan.md` (3 milestones).

- [ ] M1 — Complete + reorganize the surface: fold the remaining families into
  `dispatcher.Families()` with group/leaf nesting (`review|scrollback|changelog|
  clip <leaf>`) + streaming routes + minimal busybox `argv[0]` prefix-strip. Pure
  Go; no consumer changes; standalone binaries still build; backward compatible.
- [ ] M2 — Rewrite every owned call site to `pair <sub>` (launcher Go + title-
  poller guard, clipcmd, distiller, nvim, `.claude`, zellij KDL + bundle mirror),
  family-by-family. Helpers stay built → each commit green.
- [ ] M3 — `GO_BINS := pair`, drop the `pair-go` twin, stop bundling helper
  binaries (pair shrinks), guarantee `pair`-on-PATH, keep only the `pair-slug`
  symlink, delete `cmd/<helper>` dirs + old flat aliases, measure. Atlas/README at
  close. (`~/.zshrc`→`pair scribe` and the `pair-slug` hook are out-of-repo.)

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Thorough plan doc → +15% design buffer; familiarity 1.0 (this codebase, and the #93/#96/#99 arc built the exact machinery). Itemized by milestone: M1 = entrypoint busybox + the group/leaf dispatch fold; M2 = the caller rewrites (launcher+guard, clipcmd, distiller, nvim, zellij/bundle); M3 = build collapse + bundle + cleanup; plus one milestone-review per boundary.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
design-buffer: 0.15
item: skill-or-dispatcher design=0.3 impl=0.3
item: smaller-go-module   design=0.3 impl=1.0
item: smaller-go-module   design=0.2 impl=1.0
item: smaller-go-module   design=0.1 impl=0.5
item: smaller-go-module   design=0.1 impl=0.3
item: lua-neovim          design=0.2 impl=0.8
item: smaller-go-module   design=0.1 impl=0.7
item: smaller-go-module   design=0.2 impl=0.8
item: smaller-go-module   design=0.1 impl=0.5
item: atlas-docs          design=0.1 impl=0.6
item: milestone-review    design=0.0 impl=0.2
item: milestone-review    design=0.0 impl=0.2
item: milestone-review    design=0.0 impl=0.2
total: 9.06
```

## Log

### 2026-07-06

- Designed via brainstorm + code sweep. Key finding: the #93/#96/#99 milestones
  already ported every helper into `cmd/internal/*cmd` with `RunXxxCLI` seams and
  built the `dispatcher`/`entrypoint` subcommand machinery — this issue is
  finishing that arc, not new architecture. Consumer map + durable plan written.
- ARCH: `ARCH-DRY` (5-way binary-list restatement → one `Families()` source),
  `ARCH-PURPOSE` (Full Phase 2 — every owned consumer derives from `pair <sub>`;
  symlinks-only would be the deferred-purpose "easy subset"), `ARCH-PURE`
  (`RunXxxCLI` seams already pure; argv[0] map kept a pure function).
- Fresh-eyes plan review (subagent, verified against code): 3 blocking gaps
  found + folded into the plan — (1) `titlepoller.pollerArgvMatches` matches the
  literal `"pair-title "` argv, so the self-exec would break the single-instance
  guard → duplicate pollers (guard rewrite co-located, plan Task 2.2); (2) dev
  sessions run `make build` not `install` (dogfood path); (3) `pair-go-install-
  layout-test.sh` asserts `bin/pair-go` exists → migrate to the `pair` launcher.
  Corrections: copy-on-select is streaming for **stdin** (not lifetime); 16 (not
  17) bundled helper binaries; `session-watch` already a family; `.claude` also
  grants `bin/pair-wrap`.
- v2 decisions (this session): (a) **fold `pair-scribe` in** → truly one binary
  + 3 shell shims; `~/.zshrc`→`exec pair scribe`. (b) **Reorg** = nest families,
  keep names (`review|scrollback|changelog|clip <leaf>`). (c) **PATH simplified**
  — stop expanding binaries; `pair` reached via inherited session PATH (pathenv
  prepends, doesn't replace) + `dir(os.Executable())` prepend; no store symlink,
  drift/prune concern gone; content-addressed caching already skips re-extract.
  (d) **zellij 0.44.3 verified** — `copy_command` + `Run` accept two-token
  `pair <sub>` (whitespace-split, no shell); no zellij symlink needed. Milestones
  restructured: M1 surface+reorg · M2 rewrite callers (green per commit) · M3
  collapse+stop-bundling. Plan `## Revisions` v2 records the deltas.
