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

Snapshot callers use `internal/testoracle.Run`: stdout remains frame JSON, while
stderr records synchronous startup (Node version, executable, platform and
architecture), import, write, snapshot and dispose markers. Each call enables
Node fatal-error and signal reports in its own Go artifact directory; environment
and network information are excluded. On a process failure or a later frame
assertion failure, the complete `request.json`, `stdout.raw` and `stderr.txt`
are retained, together with `node-report.json` if Node produced one. Successful
calls remove their evidence. Reports are not guaranteed for a raw SIGSEGV;
`--report-on-signal` uses SIGUSR2, and a missing report leaves the last phase and
exact request as evidence rather than establishing a cause.

For retained files on a manual Go run, add `-artifacts -outputdir <owned-dir>`.
The native CI runner uses those flags under invocation scratch and publishes
complete failed oracle files into the retained job log before cleanup. Replay
only the exact retained request against the recorded runtime/dependency lock,
for example `node tests/terminal-oracle/driver.cjs < request.json` from the repo
root. A passing replay does not resolve the historical crash (#668).
Run `node --test tests/terminal-oracle/driver_test.cjs` for the headless phase and
report controls; native CI runs these after installing the pinned dependencies.
