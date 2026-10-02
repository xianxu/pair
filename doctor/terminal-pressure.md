# Couch terminal pressure experiment

Use this when investigating delayed child panes while Couch controls remain
responsive. This is an isolated Go test harness, not a recorder attached to a
running session. It exercises the current checkout's code using disposable fake
or real-PTY children. No live Couch/Zellij session is attached or modified.

## Run and retain evidence

Requirements: a Pair source checkout, its Go toolchain/dependencies, and a host
that supports the repository's PTY tests. Run from the repository root. The
original measurements used macOS/arm64; record your platform rather than assuming
its timings transfer. Dependency download/build time precedes the test window.

This shell block creates a unique local evidence directory, records provenance,
preserves the test exit code, and prints the log path even on failure:

```sh
(
  pressure_out=$(mktemp -d "${TMPDIR:-/tmp}/pair-pressure.XXXXXX") || exit 1
  {
    date -u
    git rev-parse HEAD
    git status --short
    go version
    uname -srm
    printf 'GOMAXPROCS=%s\n' "${GOMAXPROCS:-automatic}"
    printf '%s\n' 'Full matrix: fake/pty × baseline/burst/burst-one-cpu/slow-host × 3'
  } > "$pressure_out/environment.txt"
  PAIR_COUCH_PRESSURE=1 go test ./cmd/internal/couchtty \
    -run '^TestCouchOutputPressure$' -count=1 -v -timeout=180s \
    > "$pressure_out/results.log" 2>&1
  pressure_rc=$?
  printf '%s\n' "$pressure_rc" > "$pressure_out/exit-status.txt"
  cat "$pressure_out/results.log"
  printf '\nEvidence: %s\n' "$pressure_out"
  exit "$pressure_rc"
)
```

The matrix takes about 50 seconds after compilation on the original machine.
Each trial uses 12 children, a 191×54 host, a two-second output window and bounded
recovery/teardown. `-count=1` prevents cached results. Avoid concurrent test/build
storms when measuring the baseline. Copy useful evidence out of temporary storage
into the investigation's durable record; delete disposable captures when finished.

For a quick real-PTY burst probe (one repetition):

```sh
PAIR_COUCH_PRESSURE=1 go test ./cmd/internal/couchtty \
  -run '^TestCouchOutputPressure$/^pty$/^burst$/^1$' \
  -count=1 -v -timeout=30s
```

Verify that output contains the intended `=== RUN` subtest and its `mode=...`
measurement line. A skipped test or `no tests to run` supplies no evidence.
Use the full matrix for comparisons; a single trial is only a smoke check.
`PAIR_COUCH_PRESSURE_CHILD` and `PAIR_COUCH_PRESSURE_TRAILING` are internal helper
settings, not user-facing controls.

## Read the measurements

Each trial logs its transport, condition, geometry and effective Go CPU setting.

| Field | Meaning |
| --- | --- |
| `receipt` | Input injection to child receipt; real PTY uses an independent side pipe, fake observes its input buffer. |
| `endpoint_ack` | Input injection to ACK observed in the parsed child screen. |
| `displayed_ack` | Input injection to ACK observed in the fake host display. |
| `menu` | Ctrl+Space injection to rendered switcher, probed while output is active. |
| `*_censored` | ACK was not observed before the menu covered the pane; the logged latency is a lower bound, not an exact value. |
| `emitted_window_bytes`, `ingested_raw` | Workload output versus ingested bytes, including readiness/final-marker framing in the latter. Exact accounting is asserted. |
| `publications`, `host_bytes`, `host_writes` | Delivery and display work; input byte volume is not changed-screen volume. |
| `recovery` | Time beyond the two-second window until all children have in-band completion, all bytes are published, and the selected final marker is displayed. |
| `selective_stall` | Diagnostic criterion: receipt or displayed ACK exceeds one second while menu response stays below 100ms. |

ACK polling is every 10ms, so small latency differences are not precise rankings.
PASS means the fixture completed and its integrity checks passed; it does **not**
mean that no slowdown occurred. Inspect `selective_stall`, censored observations
and actual latencies. A failure/timeout is incomplete evidence, not a healthy run.

## What it can establish

Baseline uses 100ms output ticks; burst uses 10ms ticks with roughly 4KB per child,
up to about 9.375MiB per trial. Missed ticks are skipped. `burst-one-cpu` constrains
only the parent Go runtime, not PTY child CPU or system-wide scheduling.
`slow-host` adds 50ms per host write to baseline traffic as a display control.

The host is an emulator, not Ghostty. Children are synthetic, not Zellij, nvim
or agent CLIs. Repeated writes mostly hit one screen region, stressing parsing
more than full-screen changes. Trials are short and do not emulate a process
storm. A negative result cannot exclude those omitted causes. Compare paired
conditions and repeated measurements before attributing a bottleneck.

For a live incident, preserve current stack/process evidence first using the
[performance investigation procedure](SKILL.md#performance-capture-208). Run
this experiment separately to test a concrete hypothesis. #373 did not reproduce
the reported cross-slot freeze; #375 tracks richer production measurements and
agent guidance. Extend workloads only when evidence identifies what is missing.

If modifying this harness, run its lifecycle and failure-path checks:

```sh
go test -race ./cmd/internal/couchtty \
  -run '^TestCouchPressure(StalledOperation|TrailingPTYOutput|Control)$' \
  -count=1 -v -timeout=60s
```

These cover cancellation, joined teardown, and rejection of recovery while final
PTY output remains withheld. Preserve independent receipt/display observations
and control probes during pressure when extending the experiment.
