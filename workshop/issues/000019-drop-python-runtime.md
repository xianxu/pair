---
id: 000019
status: open
deps: []
created: 2026-05-10
updated: 2026-05-10
---

# Drop Python from pair's runtime path

## Problem

In commit `14dc879`, pair-wrap and pair-scrollback-render were ported
to Go (`cmd/pair-wrap`, `cmd/scrollback-render`), but the Python
originals were kept as fallbacks:

- `bin/pair-wrap.py` — renamed from `bin/pair-wrap`; the Go binary now
  occupies that path. Kept so a broken Go build doesn't ship a wedge.
- `bin/pair-scrollback-render` — the pyte-based renderer, also kept
  as a fallback. `bin/pair-scrollback-open` prefers
  `$PAIR_HOME/bin/scrollback-render` (Go) when present and falls back
  to `python3 bin/pair-scrollback-render` otherwise.

Carrying both has costs: it leaves `python3 + pyte` in the dependency
graph (the brew formula vendors pyte into a private venv, see
`pair-bootstrap` and homebrew-pair Formula), and "two implementations"
is a maintenance trap — a future bug fix lands in one and not the
other.

Once the Go binaries have soaked through a few days of real use and
no Alt+/ / Alt+i / agent-output-span regressions have surfaced, drop
the Python.

## Spec

Three drops, ideally in one commit per repo:

**In this repo (`pair`):**

1. Delete `bin/pair-wrap.py`.
2. Delete `bin/pair-scrollback-render` (the Python pyte renderer).
3. Simplify `bin/pair-scrollback-open` to invoke the Go binary
   directly — remove the python3 + pyte preflight + fallback branch.
   Hard-fail with a clear "build the Go renderer: make
   scrollback-render-install" message if `bin/scrollback-render` is
   missing.
4. Drop the `pair-bootstrap` target's pyte-install step in
   `Makefile.local` (and the comment paragraph explaining why pyte
   was needed). The target either becomes a no-op or gets removed —
   double-check no other runtime Python deps need it before
   deleting outright.
5. Update `cmd/scribe/README.md` and `cmd/scrollback-render/README.md`
   (if it gets one) — drop any "Python fallback" language.
6. Update `atlas/architecture.md` — the scrollback section currently
   mentions "Downstream pair-scrollback-render reads both files and
   replays through pyte" / similar phrasing. Re-anchor on the Go
   binary.

**In `homebrew-pair`** (separate repo):

7. Drop the pyte venv from the Formula. The `pair-bootstrap` target
   no longer needs it; the formula's `install` should:
   - Remove the `libexec/venv` creation
   - Remove the `pip install pyte` step
   - Remove any `PAIR_HOME/venv/bin/python3` references
8. Bump version (release-worthy: drops a runtime dep entirely).

**In `ariadne`** (separate repo, separate session):

9. `scripts/close-issue.py` is the only Python left across the
   ariadne-styled repos that pair inherits. Port to bash — close-issue
   isn't perf-sensitive, but consistency wins. (Lower priority than
   the pair drops above; can ship anytime.)

## Plan

- [ ] Soak Go binaries in real sessions for ~3-5 days; track any
      regressions in this issue's Log section.
- [ ] Drops 1-6 (pair repo) in one commit. Subject suggestion:
      `pair: drop python from runtime path`.
- [ ] Drops 7-8 (homebrew-pair).
- [ ] Drop 9 (ariadne) — separate, lower priority.

## Log

- 2026-05-10: filed. Go ports landed in commit `14dc879`; Python
  fallbacks intentionally retained for the soak window.
