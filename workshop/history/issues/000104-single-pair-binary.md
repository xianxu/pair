---
id: 000104
status: done
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 9
started: 2026-07-06T16:05:29-07:00
actual_hours: 8.53
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

- [x] M1 — Complete + reorganize the surface: fold the remaining families into
  `dispatcher.Families()` with group/leaf nesting (`review|scrollback|changelog|
  clip <leaf>`) + streaming routes + minimal busybox `argv[0]` prefix-strip. Pure
  Go; no consumer changes; standalone binaries still build; backward compatible.
- [x] M2 — Rewrite every owned call site to `pair <sub>` (launcher Go + title-
  poller guard, clipcmd, distiller, nvim, `.claude`, zellij KDL + bundle mirror),
  family-by-family. Helpers stay built → each commit green.
- [x] M3 — `GO_BINS := pair`, drop the `pair-go` twin, stop bundling helper
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
- 2026-07-06: closed — Re-close to re-anchor the publish boundary at HEAD: the only delta since the prior close (41a60e3) is the workshop/lessons.md capture (§4, docs-only, no code change) — all code + tests unchanged, make test still green (142 ok). Single pair binary delivered as previously verified.; review verdict: not-run
- 2026-07-06: closed — Single pair binary delivered: 19 binaries → 1 (pair) + 3 shell shims + pair-slug symlink; bin/pair 81MB→11MB; incremental make build ~1s (was 19 links incl. two 81MB). Every former helper is pair <sub> (nested review|scrollback|changelog|clip); every owned caller rewritten (M2); runtime bundle = config+shims only, pair-on-PATH via launcher exeDir prepend (embedded-runtime smoke). make test green (142 ok, 0 FAIL) across M1/M2/M3; ARCH-PURPOSE shadow-sweep clean. Per-milestone fresh-eyes subagent reviews (M1 SHIP, M2/M3 FIX-THEN-SHIP) with all Critical/Important findings fixed — sdlc claude dispatch E2BIGs on these large windows so reviews ran via subagent. Out-of-repo (user): ~/.zshrc → exec pair scribe; verify pair-slug Stop hook (in-repo caller already uses pair slug).; review verdict: not-run
- 2026-07-06: closed M3 — make test green (142 ok pkgs, 0 FAIL); one binary — bin/pair 81MB→11MB, incremental make build ~1s (one link, was 19); pair-embedded-runtime smoke proves pair-on-PATH via exeDir prepend with ZERO bundled helper binaries; ARCH-PURPOSE shadow-sweep clean (every helper reached as pair <sub>, only pair-slug symlink residual). Fresh-eyes boundary review via subagent (sdlc claude dispatch E2BIGs on the large M3 window): verdict FIX-THEN-SHIP — Critical (scrollback.lua refresh still called the removed scrollback-render alias → fixed to pair scrollback render, test now pins both tokens) + Important (relocated the #59 render→distill seam e2e to cmd/pair-go + 14 changelogcmd.Run behavioral tests to cmd/internal/changelogcmd) all fixed + re-verified green; review verdict: not-run
- 2026-07-06: closed M2 — make test green (146 ok pkgs, 0 FAIL); every owned caller rewritten to pair <sub> (launcher self-exec + title-poller guard, clipcmd, distiller, nvim review/scrollback, zellij copy_command/Run/exec); caller sweep confirms no standalone-name invocations remain; test-review + test-lua + rewritten copy-on-select test pass; review verdict: FIX-THEN-SHIP
- 2026-07-06: closed M1 — make test green (146 ok pkgs, 0 FAIL); 30/30 go pkgs; nested pair<sub> routes + aliases + pair-slug busybox symlink smoke-verified on a real build; fresh-eyes boundary review done via subagent (sdlc claude dispatch hit E2BIG on the mis-computed ancient boundary base a9c32ef/#61, ~19.6k insertions) — verdict SHIP, all correctness areas clean, 2 minor cleanups (dead code + busybox prefix guard) applied in follow-up commit; review verdict: not-run

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
