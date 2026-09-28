---
id: '000019'
status: done
created: 2026-05-10
updated: 2026-05-27
actual_hours: 1.88
---

# Drop Python from pair's runtime path

## Problem

In commit `14dc879`, pair-wrap and pair-scrollback-render were ported
to Go (`cmd/pair-wrap`, `cmd/pair-scrollback-render`), but the Python
originals were kept as fallbacks:

- `bin/pair-wrap.py` — renamed from `bin/pair-wrap`; the Go binary now
  occupies that path. Kept so a broken Go build doesn't ship a wedge.
- `bin/pair-scrollback-render.py` — renamed from
  `bin/pair-scrollback-render` (the pyte-based renderer). The Go
  binary now occupies the prefix-free path.
  `bin/pair-scrollback-open` prefers
  `$PAIR_HOME/bin/pair-scrollback-render` (Go) when present and falls
  back to `python3 bin/pair-scrollback-render.py` otherwise.

Carrying both has costs: it leaves `python3 + pyte` in the dependency
graph (the brew formula vendors pyte into a private venv, see
`pair-bootstrap` and homebrew-pair Formula), and "two implementations"
is a maintenance trap — a future bug fix lands in one and not the
other.

Once the Go binaries have soaked through a few days of real use and
no Alt+/ / Alt+i / agent-output-span regressions have surfaced, drop
the Python.
