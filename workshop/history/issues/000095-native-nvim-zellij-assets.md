---
id: 000095
status: done
deps: [000094]
github_issue:
created: 2026-07-01
updated: 2026-07-04
estimate_hours: 2.4
started: 2026-07-03T14:17:31-07:00
actual_hours: N/A
---

# native nvim and zellij startup assets

Tracking: #91 (native single binary) — step 5 of 5. Depends on #94.

## Problem

After #94, the embedded runtime bundle carries only the native assets —
`nvim/*.lua` and `zellij/*.kdl` — which are still extracted to
`$PAIR_DATA_DIR/runtime/<digest>/pair-home` on a copied binary's first run.
That extraction step is the last thing standing between Pair and a *true* native
single binary: an executable that provisions its runtime without writing a
Pair-owned tree to disk. Whether extraction is even removable is an open
question — `nvim` and `zellij` are external processes that read config from
files/paths — so this step is a decision as much as an implementation.

## Spec

Revisit how `nvim/` and `zellij/` startup assets reach the external processes,
and reach the endpoint: **one Pair executable, with external platform tools
(`zellij`, `nvim`, `fzf`, `jq`, clipboard tools, agent CLIs) still supplied by
the system.**

Evaluate the options (this issue includes the design decision, not just code):

- **Keep extracting native assets** to a versioned runtime dir (status quo from
  #90) — simplest, but still writes a Pair-owned tree.
- **Generate ephemeral temp files** at launch (write config to a temp path, pass
  it to `nvim`/`zellij` via `-u` / `--config` / env, clean up after) — no
  persistent extracted tree.
- **API/flag-driven startup** where the external tool supports it, minimizing or
  eliminating on-disk config.

Constraints:

- Do not port `nvim`/`zellij` logic into Go — they stay native assets
  (invariant from #72/#90). This is about *how the assets reach the process*,
  not rewriting them.
- Whatever path is chosen must keep the runtime deterministic, idempotent, and
  upgrade-safe, and must keep working for the source/Homebrew adjacent layouts,
  not only the copied binary.
- `ARCH-PURPOSE`: this closes only if it reaches (or consciously documents the
  final gap to) the true native single binary — one executable, no Pair-owned
  tree extracted, platform tools external.

## Done when

- [x] The nvim/zellij startup-asset strategy is decided and documented (extract
      vs temp-file vs API/flag-driven), with the trade-offs recorded. (Log + atlas.)
- [x] The chosen strategy is implemented and tested for copied-binary, source,
      and Homebrew layouts. (Extraction kept; the PATH fix is layout-agnostic — the
      copied-binary smoke exercises it end-to-end; source/Homebrew share the same
      `prependBinToPath(resolvedRoot)` path, differing only in the resolved root.)
- [x] `pair` reaches the native-single-binary endpoint (or the residual gap is
      explicitly documented in atlas as an accepted limitation). (Residual
      zero-tree gap documented — unreachable with external nvim/zellij.)
- [x] `atlas/go-migration-inventory.md` and `atlas/architecture.md` describe the
      final runtime-provisioning shape.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.2 impl=0.8
item: atlas-docs design=0.2 impl=0.6
item: milestone-review design=0.0 impl=0.4
total: 2.4
```

Small: the decision is documentation (extraction is unchanged), the only code is a
pure `prependBinToPath` helper + one `RunLaunch` SetEnv line + two tests (unit +
fake wiring) + the copied-binary smoke assertion; `atlas-docs` covers the
cache/residual-gap reframe + the three stale-PATH-doc fixes (incl. the Homebrew
peer `pair.rb`); one boundary review (atomic single-pass, no `Mx`). Design is
weighted 1.6× (raw item sum 2.0 → total 2.4). Durable plan:
`workshop/plans/000095-native-nvim-zellij-assets-plan.md`.

## Plan

- [x] Survey how `nvim`/`zellij` accept config (files, `-u`/`--config`, env, IPC)
      and what each option costs for determinism/upgrade-safety. (See Log survey.)
- [x] Decide extract vs ephemeral-temp vs API-driven; record the decision + why.
      → **keep extraction, reframe as a content-addressed cache** (Log decision).
- [x] Fix the PATH-prepend regression: `RunLaunch` prepends `$PAIR_HOME/bin` to
      PATH (pure `prependBinToPath` + `SetEnv`), so zellij resolves the bundled
      bare-name helpers across copied/source/Homebrew layouts.
- [x] Tests: `prependBinToPath` unit + `RunLaunch` fake-wiring + the copied-binary
      smoke asserting bare-name helper resolution (verified it fails without the fix,
      exit 21).
- [x] Update atlas to the final shape (extraction = runtime cache; documented
      residual zero-tree gap) + fix the three stale "launcher prepends PATH" docs
      (`atlas/architecture.md` ×2, `zellij/config.kdl`, peer `pair.rb`).

## Log

### 2026-07-03 — survey + decision
- 2026-07-03: closed — #95 closes the native-single-binary roadmap (#91 step 5). DECISION: keep the digest extraction (reframed as a content-addressed cache; true zero-tree unreachable with external nvim/zellij — documented residual gap). FIX: restored the PATH-prepend regression #99 M5c dropped — RunLaunch prepends $PAIR_HOME/bin via pure prependBinToPath so a copied/Homebrew pair resolves zellijs bare-name helpers (pair-wrap, copy-on-select, pair-help). Verified: full make test green (MAKE_EXIT=0); prependBinToPath 5-case unit; RunLaunch fake-wiring test (no pollution); copied-binary smoke asserts bare-name resolution + PROVEN to fail exit 21 without the fix. Atlas reframed + 3 stale PATH docs fixed (peer homebrew pair.rb committed+pushed). --no-actual: sdlc actuals auto-window matched an UNRELATED 2026-06-15 "#95 M5" commit (a different issue-number-95 in history — mention-fallback collision), scoping to 8.46h across ~80 issues instead of the 5-commit #95 branch (~40min actual); no --base override exists to correct it, so recording N/A rather than the collision-inflated figure.; review verdict: SHIP

**Decision: keep the digest-versioned extraction; reframe it as a content-addressed
runtime *cache*; document that true zero-tree is unreachable with external
nvim/zellij.** Durable plan: `workshop/plans/000095-native-nvim-zellij-assets-plan.md`.

**Why the endpoint ("zero Pair-owned tree extracted") is physically unreachable:**
`nvim`/`zellij` are external processes that read config from real filesystem paths
(`nvim -u init.lua`, `zellij --config-dir`). Go's `embed.FS` is in-memory, not a
path — so *some* on-disk materialization is unavoidable. Worse for nvim:
`nvim/init.lua` `dofile()`s ~5 siblings by absolute path (`review/seam.lua`,
`slug.lua`, `doctor.lua`, `zellij_trace.lua`) → it needs a real **directory** tree,
not one file; and viewers (scrollback/changelog/review) spawn fresh
`nvim -u $PAIR_HOME/nvim/*.lua` **mid-session**, so the tree must persist for the
whole session, not just launch.

**Options weighed (trade-offs, Done-when #1):**
- **(a) Keep extracting** — already deterministic (content digest), idempotent
  (skip-unchanged), upgrade-safe + self-pruning (keep-2). A *cache*, not scratch.
  Confined to the copied-binary layout (source + Homebrew point their asset root at
  an adjacent real tree and never extract). **Chosen.**
- **(b) Ephemeral temp dir** — writes the *same* 27-file nvim tree to `/tmp`; the
  mid-session viewer spawns force session-lifetime persistence (else detach→reattach
  breaks), so it degenerates into a worse, non-shared re-implementation of the
  digest cache that re-writes every launch. **Rejected.**
- **(c) API/flag-driven** — no nvim API to load a config tree from memory; `-u`
  needs a file, `--cmd` can't carry the dofile-tree through argv. **Not viable.**

**Discovered regression (folded into #95's scope):** the Go launcher never prepends
`$PAIR_HOME/bin` to PATH, though `atlas/architecture.md` (~207, ~803),
`zellij/config.kdl:39`, and the Homebrew `pair.rb` all claim it does — describing
the *retired* shell `bin/pair`, whose prepend was dropped in #99 M5c. zellij execs
`pair-wrap` (the agent pane) + `copy_command "copy-on-select"` + `Run "pair-help"`
by **bare name**, so a copied/Homebrew `pair` (whose `bin/` isn't on the user's
PATH) **can't launch a session**. Masked in dev because dev-aliases already put
`bin/` on PATH. This sits directly on #95's "works across copied/source/Homebrew"
Done-when, so #95 restores the prepend (in `RunLaunch`) + a regression-guarding
smoke. Operator confirmed folding it into #95 (vs a separate bug).

**Endpoint reached (as documented):** one Pair *executable* + a self-provisioned
content-addressed config *cache* for the two external tools + system platform
tools. Not literally zero bytes on disk — that's the accepted, documented residual
gap (ARCH-PURPOSE permits documenting the final gap).

### 2026-07-01

Created as step 5 (final) of the native-single-binary tracker (#91). This is the
decision point #90's Spec deferred: whether the native `nvim/`/`zellij/` assets
stay extracted or move to ephemeral/API-driven startup. Gated on #94 having
reduced the bundle to native assets only.
