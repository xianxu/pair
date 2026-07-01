---
id: 000095
status: open
deps: [000094]
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours:
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

- [ ] The nvim/zellij startup-asset strategy is decided and documented (extract
      vs temp-file vs API/flag-driven), with the trade-offs recorded.
- [ ] The chosen strategy is implemented and tested for copied-binary, source,
      and Homebrew layouts.
- [ ] `pair` reaches the native-single-binary endpoint (or the residual gap is
      explicitly documented in atlas as an accepted limitation).
- [ ] `atlas/go-migration-inventory.md` and `atlas/architecture.md` describe the
      final runtime-provisioning shape.

## Plan

- [ ] Survey how `nvim`/`zellij` accept config (files, `-u`/`--config`, env, IPC)
      and what each option costs for determinism/upgrade-safety.
- [ ] Decide extract vs ephemeral-temp vs API-driven; record the decision + why.
- [ ] Implement the chosen provisioning path across copied/source/Homebrew layouts.
- [ ] Tests: startup across all three layouts; upgrade + cleanup behavior.
- [ ] Update atlas to the final native-single-binary shape.

## Log

### 2026-07-01

Created as step 5 (final) of the native-single-binary tracker (#91). This is the
decision point #90's Spec deferred: whether the native `nvim/`/`zellij/` assets
stay extracted or move to ephemeral/API-driven startup. Gated on #94 having
reduced the bundle to native assets only.
