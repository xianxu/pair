# Independent terminal renderer oracle

Run `sh tests/terminal-oracle/run.sh` from the repository root. It installs the
lockfile-pinned test-only `@xterm/headless` interpreter and requires the renderer
wire tests. Normal Go tests skip the oracle when npm dependencies are absent;
`PAIR_TERMINAL_ORACLE=1` makes missing dependencies a test failure. M2 and M4
verification require this target, not just the Go-only suite.

`driver.cjs` accepts one JSON object on stdin with `Cols`, `Rows`, and `Chunks`.
It returns observations after each completed write: visible lines, cell width,
foreground/background and attributes, cursor coordinates, and parsed OSC 8
payloads. Go assertions use literal expectations. No expected cell state comes
from Pair's backend. OSC observations verify wire framing and parameters;
they do not claim that xterm exposes hyperlink identity through its cell API.

This independent engine complements native Zellij/nvim conformance; its Unicode
width behavior is not used to redefine the production terminal profile.
