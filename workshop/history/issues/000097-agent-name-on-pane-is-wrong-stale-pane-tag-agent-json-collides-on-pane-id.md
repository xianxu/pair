---
id: 000097
status: done
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-05
estimate_hours: 0.5
started: 2026-07-05T21:11:05-07:00
actual_hours: 0.25
---

# agent name on pane is wrong — stale pane-<tag>-<agent>.json collides on pane_id

## Problem

The zellij agent-pane frame title shows the wrong agent: the pane is running
`claude`, but the frame reads `codex`. (`PAIR_AGENT`, `PAIR_PANE_TITLE`, and
`$PAIR_DATA_DIR/agent-<tag>` all correctly say `claude` — only the frame label is
wrong.)

## Spec

### Root cause (diagnosed)

`pair-title.sh` → `update_frame_titles()`
(`cmd/internal/runtimebundle/assets/runtime/files/bin/pair-title.sh:95-116`)
derives each pane's agent label from the **filename** of
`$PAIR_DATA_DIR/pane-<tag>-<agent>.json` and renames the zellij pane to
`<agent> (<count>) [<cwd>]`. It loops over **every** `pane-<tag>-*.json` for the
tag. When two of them point at the **same `pane_id`**, it renames that one pane
once per file, and — because the glob expands alphabetically — the last write
wins.

Observed on tag `ariadne`, both files carrying `pane_id: "0"`:

| file | mtime |
|------|-------|
| `pane-ariadne-claude.json` | Jul 1 00:22 (current session) |
| `pane-ariadne-codex.json`  | **Jun 29 12:27 (stale)** |

`claude` sorts before `codex`, so each 60s poll renames pane 0 to `claude …`
then immediately to `codex …` → the pane settles on **codex**. The stale
`pane-ariadne-codex.json` is left over from a codex session that used pane 0 on
Jun 29 and was never removed when claude later took the same pane_id.

**Not tag-specific.** The same stale twins exist for other tags:
`pane-2-codex.json`, `pane-pair-codex.json`, `pane-parley_nvim-codex.json` all
sit beside their `-claude.json` counterparts, so any tag where codex previously
held a pane_id now reassigned to claude will mislabel.

### Fix direction

Root cause is the pane-file lifecycle + the blind all-files loop. Options
(pick during design):

1. **Label only the active agent.** `update_frame_titles` already knows the live
   agent for the tag (the `$AGENT` arg it was spawned with, and/or
   `$PAIR_DATA_DIR/agent-<tag>`); skip any `pane-<tag>-<agent>.json` whose
   `<agent>` isn't the active one (or, when multiple panes legitimately coexist,
   dedupe by `pane_id` preferring the active agent).
2. **Clean up on reassignment.** When a pane is (re)assigned to an agent, remove
   the prior agent's `pane-<tag>-*.json` for that `pane_id` so no stale twin
   survives.

Add a regression test: two `pane-<tag>-*.json` sharing a `pane_id` must yield a
single frame rename to the active agent (mirror the existing
`PAIR_TITLE_TEST_CALL` / `_test_frame_titles_twice` harness).

### Decision (2026-07-05) — design closed

The poller was **ported shell→Go in #93** (`bin/pair-title.sh` is now the Go
`cmd/internal/titlepoller` package); the bug rode along verbatim. Current site:

- `updateFrameTitles` (`cmd/internal/titlepoller/run.go:170-184`) loops over every
  `pane-<tag>-*.json` returned by `PaneFiles` (`runtime.go:62`, glob → alpha
  order) and renames each. The unchanged-skip cache (`frameCache.changed`,
  `titlepoller.go:120`) is keyed on `pane_id`, so two files on one `pane_id`
  each render a *different* title for the same pane every tick → last (alpha)
  wins + churn. Same last-wins mechanism as the shell.

**Blast radius is the frame poller only.** `pair context`
(`contextcmd.go:75`) reads `pane-<tag>-<agent>.json` by **exact agent name**, not
a glob, so a stale twin never confuses it; `pair list` doesn't read pane files.
The glob at `runtime.go:63` is the sole consumer that hits the collision.

**Two-pane invariant + reliable active agent (traced).** A pair tag is exactly
one agent pane + one nvim pane (`atlas/architecture.md:5,292`); two live agents
never coexist in a tag. The poller is respawned on every entry as
`pair-title <tag> <agent>` with the agent resolved fresh from `agent-<tag>` via
`InferAgent` (`createflow.go:162-167,280`; `lifecycle.go:38`), so `opts.Agent` is
authoritatively the active agent. Any `pane-<tag>-<other>.json` is therefore
stale by definition. The staleness source: `runCleanup` (`lifecycle.go:82-92`)
removes every per-(tag,agent) sidecar **except** `pane-<tag>-<agent>.json` — an
oversight, nothing intentionally preserves it (single writer: the zellij layout
`main.kdl:45`).

**Chosen fix = both halves (consume defensively + stop the leak):**

1. **(a) Poller honors the invariant (primary, self-healing).** In
   `updateFrameTitles`, skip any pane whose `pane.Agent != opts.Agent` — render
   exactly the active agent's one frame. Fixes the symptom immediately even for
   stale files already on disk (and any left by a crash that bypasses
   `runCleanup`), and kills the per-tick flip-flop. Pure filter, unit-tested via
   the existing `updateFrameTitles` fake harness (ARCH-PURE).
2. **(b) Stop producing the cruft (root cause, ARCH-DRY).** Add
   `pane-<tag>-<quitAgent>.json` to `runCleanup`'s sidecar-removal list — the
   list already resolves `quitAgent` (`lifecycle.go:58`) and cleans every sibling
   sidecar; the pane file was simply omitted. One line, consistent.

Skipping the "one-time cleanup of existing twins" migration as YAGNI: (a)
neutralizes existing twins and (b) prevents new ones on clean quit. Immediate
manual remediation (`rm` the stale twins) stays available but is now cosmetic.

### Immediate remediation (symptom, no code change)

`rm` the stale `pane-<tag>-<agent>.json` whose `pane_id` collides with a newer
twin (e.g. `pane-ariadne-codex.json`). Corrects the frame on the next poll;
recurs on the next agent switch until the code fix lands.

## Done when

- [x] A tag with a stale `pane-<tag>-<other-agent>.json` on the same `pane_id`
      renders the frame as the **active** agent, not the alphabetically-last one.
- [x] Stale `pane-<tag>-*.json` twins are prevented (cleaned on reassignment) or
      ignored by the frame updater. *(both: ignored by (a); cleaned on quit by (b))*
- [x] Regression test: two pane files sharing a `pane_id` → one rename, active
      agent.

## Estimate

Method A (primitive decomposition). Small, fully-specced Go fix — design
pre-resolved above, so design hours are near-zero.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.1 impl=0.15
item: smaller-go-module design=0.0 impl=0.1
item: milestone-review design=0.0 impl=0.15
total: 0.5
```

- item 1: (a) `updateFrameTitles` active-agent filter + regression test.
- item 2: (b) `runCleanup` sidecar-list one-liner.
- item 3: mandatory boundary review at `sdlc close`.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Design +30% buffer absorbed (design subtotal
0.1 → negligible against rounding).*

## Plan

Atomic single-pass (≤3 files, <100 lines) — plain checkboxes, one `sdlc close`.

- [x] (a) `updateFrameTitles` (`titlepoller/run.go`): skip panes whose
      `pane.Agent != opts.Agent` so only the active agent's pane frame renders.
- [x] Regression test (`titlepoller/run_test.go`): two `PaneInfo` sharing a
      `pane_id` (active `claude` + stale `codex`), `opts.Agent=claude` → exactly
      one rename, to `claude`. Glob-order note added in the test comment.
- [x] (b) `runCleanup` (`launcher/lifecycle.go`): add
      `pane-<tag>-<quitAgent>.json` to the sidecar-removal list; extended the
      `TestRunLaunchQuitCleanup` removal-set assertion.
- [x] `go test ./...` green (env-scrubbed); verified via real-data run on the
      live colliding `pane-pair-{claude,codex}.json` (both `pane_id:0`).

## Log

### 2026-07-01

- Moved from `ariadne#151` (filed in ariadne's inbox; the bug + fix live here in
  pair). Root cause diagnosed during that session — see the evidence above:
  `pane-ariadne-{claude,codex}.json` both on `pane_id 0`, alphabetical last-wins
  in `update_frame_titles`.

### 2026-07-05
- 2026-07-05: closed — go test ./... green (env-scrubbed) + vet clean; frame poller now renders only the active agent — real-data run over this session's live colliding pane-pair-{claude,codex}.json (both pane_id:0) renders claude only, stale twin ignored; regression test reproduces the pre-fix claude/codex flip-flop (last-wins) and passes post-fix; runCleanup removes pane-<tag>-<agent>.json on quit.; review verdict: SHIP

- Claimed + start-plan. Discovered the poller was **ported shell→Go in #93**
  (`cmd/internal/titlepoller`); the bug rode along verbatim into
  `updateFrameTitles`. Traced the pane-file lifecycle (single writer
  `main.kdl:45`; no cleanup on reassignment; two-pane invariant; `opts.Agent`
  authoritative). Confirmed blast radius is the frame poller alone (`pair context`
  reads by exact agent name; `pair list` reads no pane files). Design closed to
  the two-part fix (a: poller filter, b: `runCleanup` sidecar line) — see
  Spec ▸ Decision.
- Implemented (a) `run.go` one-line filter + `run_test.go`
  `TestUpdateFrameTitlesIgnoresStaleAgentTwin` and (b) `lifecycle.go` sidecar +
  `lifecycle_test.go` assertion. `go test ./...` green (exit 0, env-scrubbed for
  the `PAIR_SESSION_ID`/`PAIR_TAG` leak), `go vet` clean. atlas/architecture.md:244
  updated (poller renders the active agent, not a blind loop; twin cleaned on quit).
- **Verified real-data.** The bug is live on this session's own tag: `PAIR_DATA_DIR`
  holds `pane-pair-{claude,codex}.json`, both `pane_id:"0"`. A throwaway driving
  the shipped `titlepoller.OSRuntime.PaneFiles` glob + the new filter over the real
  files rendered **only** `claude` (stale `codex` twin ignored). The regression
  test independently reproduces the pre-fix flip-flop
  (`claude→codex→claude→codex`, last-wins) and passes post-fix.
- Binaries (`bin/pair-title`, the runtimebundle asset) are gitignored build
  artifacts — regenerated on `make build`; the running poller adopts the fix on
  next build + respawn. Existing on-disk twins are now harmless (a) and clear on
  each tag's next clean quit (b); manual `rm` remains available but cosmetic.
