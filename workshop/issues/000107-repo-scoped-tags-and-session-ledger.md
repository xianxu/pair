---
id: 000107
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 8.2
started: 2026-07-07T11:23:27-07:00
---

# repo-scoped tags and session ledger

## Problem

pair started as a thin wrapper around claude, grew agent-neutral over time, but
never fully committed to it — so its identity model still **conflates two
independent axes**: *which piece of work* (the tag) and *which agent runs it*.
The picker exposes only the first and silently resolves the second, which
produces concrete bugs and a fragile repo binding.

**Originating bug.** `pair-dev codex -- --sandbox danger-full-access` launched
**claude** with codex's `--sandbox` flag and failed to start. Root cause:
`DecideLaunch` ignores the agent entirely (`decision.go:24`); with history
present it returns `ActionPick`. Picking an existing tag is resume-by-name, so
`runOnce` resets the agent to `""` and re-infers it from disk
(`createflow.go:158-171`) — landing on the tag's last agent (claude) — but the
CLI-forwarded `AgentArgs` (codex-only `--sandbox danger-full-access`) ride along
onto claude. The zellij layout then runs `pair wrap … claude --sandbox …` →
claude chokes. Confirmed by a failing repro test
(`TestRunLaunchPickInferredAgentMustNotInheritCliArgs`): `PAIR_AGENT="claude"`
while `PAIR_AGENT_ARGS` still carried `--sandbox danger-full-access`.

**Structural gaps behind it:**

- **Agent is not a picker axis.** The session picker shows one row per *tag*
  (`pair-<tag>`) with no agent shown; the agent is only inferred *after*
  selection via `InferAgent(tag)`. So the picker can't tell you `pair-work` is a
  *claude* tag — you can't avoid the mismatch.
- **Repo is not a real scope dimension.** History is filtered only by *tag-name
  prefix* where `base = DefaultTag(cwd)` = the cwd **basename**
  (`history.go:74`) — a fragile proxy for "same repo" (`~/work/pair` and
  `~/other/pair` collide). Live detached sessions aren't filtered at all: any
  detached `pair-*` session, from any repo, both triggers the picker and shows
  as a row (`decision.go:34`, `pick.go:48`). The cwd *is* recorded
  (`pane-<tag>-<agent>.json`, claude's transcript path) but nothing uses it to
  scope.
- **Only the last session per (tag, agent) is retained** — a single
  `config-<tag>-<agent>.json` overwritten each launch. There's no record of
  prior sessions or of cross-agent activity within a tag.

## Spec

### Reframe

A **tag is a repo-local unit of work**; an **agent is a resource used within
it**. A tag can span multiple agents over its life and can touch peer repos, but
it *homes* in the repo it started in. This decouples the two axes the current
model conflates.

**Keystone simplification (no hand-off DAG).** Sessions under a tag have no
strong lineage — there are no cross-agent hand-offs, and typically none even
between same-agent sessions. The durable thread is the **repo/work state** (git,
`sdlc state`) and the **continuation doc**, not a session graph. So the model
stays *flat*: a tag owns a time-ordered ledger of independent sessions, no
parent/child edges. `pair continuation` is the explicit transition primitive
that reconstitutes state into a fresh session regardless of agent.

### Decisions made (this discussion)

1. **Repo becomes a real scope dimension; tags become repo-local.** The
   `pair-<repo>-work-item` naming convention is retired — you no longer encode
   the repo into the tag by hand. Scope = **the repo, identified by its repo
   name** for anything user-facing.
2. **No hash in anything presented to the user.** A collision-safe internal key
   (e.g. a hash of the repo's absolute path) is acceptable *only where it's
   hidden* (the data-dir subdir key). Session names, the picker, and titles show
   the **repo name + tag**, never the hash.
3. **Per-tag append-only session ledger is the new source of truth.** It records
   `{agent, session_id, started, last_active, repo}` per session (at least the
   last per agent; keep a few for recovery). `agent-<tag>` and the single-
   overwrite `config-<tag>-<agent>.json` become *derived caches*. Cross-agent
   ordering (a single-threaded human view) comes from `last_active` — derivable
   from native-transcript mtimes without the agents' cooperation.

### Identity model (two layers)

- **Display tag** — local to the repo, defaults to the repo name, never carries
  a hash. What the user types and sees.
- **Scope key** — hidden, collision-safe (hash of the repo abs path). Lives only
  in the data-dir subdir name; a `meta` inside records the abs path + display
  repo name.

Why two layers — the **technical reason** repo-scoping was skipped before, still
live:
1. **zellij session names are one global namespace per user** — `pair-<tag>`
   must be globally unique, so the tag *was* the global key (hence the
   `pair-<repo>-…` convention).
2. **The data dir is flat and global** (`~/.local/share/pair`, sidecars keyed by
   bare `<tag>`) — `config-work-claude.json` is shared by any repo using `work`.
3. **Session names have a hard length budget (#54)** — `ProbeSessionName` asks
   zellij, which rejects names that overflow the socket path. So you can't just
   prepend a full repo path/slug.

Resulting shape:
- **Data-dir per-repo subdir**: `~/.local/share/pair/<scope>/{config,draft,log,
  ledger,…}`. The repo filter stops being a name-prefix guess and becomes a
  **filesystem fact** — "this repo's tags" is a listing of the subdir; history
  glob, picker, and ledger scope naturally.
- **Session name** carries the scope for global uniqueness while displaying the
  repo name; same-repo-name collisions (rare) fall back to pair's existing
  numeric-disambiguation. (Exact session-name encoding is a spec detail — must
  respect #54 and keep the hash out of what the user reads.)
- **`parley.nvim` dot dissolves for free**: the scope is a hash of the *path*,
  not a normalized basename, so `parley.nvim` vs `parley_nvim` can never collide
  across repos. The dot only remains a per-repo cosmetic concern where the tag
  appears in a filename/session name (display literal, normalize the key).

### Startup modes (target UX)

- **1.1 — explicit agent + params → new session (no picker).** e.g.
  `pair codex -- --sandbox …` creates a codex session directly. Within a session,
  a "continue last recorded session" command: if last session is the **same
  agent**, offer to extend it (saved params or new ones — ~today's config
  picker); if **agent mismatch**, offer `pair continuation` (start the new agent,
  seed from the previous session's transcript). *This subsumes the originating
  bug — explicit agent never routes through the resume-picker.*
- **1.2 — bare `pair` → continue last, inherit its agent + params.** The picker
  survives but is reframed as a **repo-scoped, agent-annotated work-item
  picker**; "continue the most-recent tag" is the default row.
- **1.3 — (depends on future per-repo default-agent config, cf. #83)** an extra
  option to start a fresh session with the repo's default agent/params.

### Immediate crash guard (decouple from this redesign)

Independently ship a small guard so users aren't exposed while this bakes: a
picked/inferred agent must never inherit another agent's CLI-forwarded args
(drop them on agent mismatch), optionally plus showing the agent in the picker.
Repro test already staged.

## Done when

- A tag is scoped to its repo: two repos can each have an independent `work` tag,
  live simultaneously, without session-name or sidecar collision.
- The session picker is filtered to the current repo and annotates each row with
  its agent, using repo name + tag only (no hash surfaced anywhere).
- A per-tag session ledger records sessions across agents (≥ last per agent) and
  is the source of truth; `agent-<tag>`/config are derived from it.
- `pair codex -- <codex args>` starts codex with those args and never launches a
  different agent (originating bug fixed at the model level).
- Bare `pair` offers "continue last session" for the current repo, inheriting its
  agent + params.
- Existing flat sidecars + live unscoped sessions migrate/grandfather without
  data loss.

## Open questions

- Session-name encoding that keeps the scope hidden yet globally unique within
  #54's length budget (repo name + tag + disambiguator vs. a hidden key).
- Ledger granularity: how many sessions per agent to retain, and what counts as
  a session boundary (native transcript file? pair launch?).
- "Touch multiple repos": a session run in a peer repo records its own cwd in the
  ledger, but the tag stays under its origin scope — confirm this is the wanted
  behavior for the peer repo's picker.
- Interaction with #83 (per-repo default agent) for mode 1.3.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: cross-cutting-refactor design=1.4 impl=3.0
item: greenfield-go-module design=0.7 impl=1.2
item: lua-neovim design=0.3 impl=0.6
item: atlas-docs design=0.1 impl=0.25
item: milestone-review design=0.0 impl=0.25
design-buffer: 0.15
total: 8.18
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only. Calibration source is currently
marked stale by `sdlc estimate-source`, so the per-primitive values are
provisional but derived from the required source.

## Plan

- [x] Brainstorm → durable plan (`superpowers-writing-plans` → `workshop/plans/`)
      before implementation; resolve open questions above.
- [x] Add scoped identity/path/session-name pure model with failing tests first.
- [ ] Add per-tag session ledger as source of truth, while emitting derived
      `agent-<tag>` and `config-<tag>-<agent>.json` caches.
- [ ] Migrate launcher decisions, history, picker, list, rename, restart, and
      cleanup to current-repo scoped snapshots and sidecar paths.
- [ ] Update zellij/nvim/shell consumers so inherited scoped `PAIR_DATA_DIR` is
      authoritative.
- [ ] Grandfather existing flat sidecars and live unscoped sessions without data
      loss.
- [ ] Add acceptance tests for same-tag multi-repo isolation, picker agent
      annotation, explicit-agent arg safety, bare `pair` ledger continuation,
      and legacy recovery.
- [ ] Update atlas docs and run final verification.

## Log

### 2026-07-07

Created from a live design discussion. Started from the `pair-dev codex --
--sandbox danger-full-access` crash (codex args leaked onto claude; root-caused
to `DecideLaunch` ignoring the agent + resume-by-name re-inference carrying the
CLI args). Discussion widened to the underlying identity model. Decisions
recorded in Spec: repo as a real scope dimension with repo-local tags (retire the
`pair-<repo>-…` convention), no hash in anything user-facing, and a per-tag
append-only session ledger as source of truth. Not #83 — #83 (per-repo default
agent) layers on top (mode 1.3). Next: decouple + ship the crash guard; claim
this issue and run it through brainstorming into a durable plan.

Claimed issue via `sdlc claim --issue 107`, entered planning via
`sdlc start-plan --issue 107`, and wrote durable plan
`workshop/plans/000107-repo-scoped-tags-and-session-ledger-plan.md`.
Architecture decisions in the plan cite ARCH-DRY (centralized scoped identity
and paths), ARCH-PURE (pure model with thin IO seams), and ARCH-PURPOSE (all
consumers and legacy data must migrate, not just the crash guard).

Plan-quality gate returned FAILURE: session-name encoding, same-name repo
mapping, and ambiguous legacy ownership were underspecified. Revised the durable
plan to make public zellij names concrete (`pair-<repo>-<tag>[-N]`), add a
global `SessionNameIndex` mapping public names back to hidden scope keys, define
live-session mapping order (index → scoped pane metadata → legacy recovery), and
define ambiguous flat sidecars as explicit manual-import recovery rather than
silent ownership. Increased estimate to 8.2h to reflect the clarified registry
and migration work.

Second plan-quality gate found the length-budget fallback and ambiguous legacy
predicate still too loose. Revised the plan with a deterministic session-name
candidate algorithm: full readable name first, then trim the longer repo/tag
component while preserving at least 4 normalized chars each and the numeric
suffix; abort before sidecar writes if no probed candidate passes zellij. Also
made the ambiguous legacy recovery predicate exact:
`tag == DefaultTag(currentRepoRoot)` or `strings.HasPrefix(tag,
DefaultTag(currentRepoRoot)+"-")`.

Implemented the first pure identity slice with TDD: `RepoScope`,
`ScopedPaths`, `SessionNameIndex`, public session-name candidate assignment, and
optional assigned session names in `DecideLaunch`. Verified with:
`go test ./cmd/internal/launcher -run
'TestRepoScope|TestNormalizeDisplayComponent|TestDefaultTag'`,
`go test ./cmd/internal/launcher -run
'TestScopedPaths|TestCanonicalConfigPath|TestConfigPaths'`,
`go test ./cmd/internal/launcher -run
'TestScopedSessionName|TestSessionNameIndex|TestAssignSessionName|TestDecideLaunch'`,
and `go test ./cmd/internal/launcher`.

Implemented the pure ledger slice with TDD: `LedgerEntry`, JSONL
parse/render, latest-entry selection, latest-per-agent selection, and
compaction that keeps recent rows plus latest per agent. Verified with:
`go test ./cmd/internal/launcher -run
'TestSessionLedger|TestLatestLedger|TestCompactLedger'` and
`go test ./cmd/internal/launcher`.
