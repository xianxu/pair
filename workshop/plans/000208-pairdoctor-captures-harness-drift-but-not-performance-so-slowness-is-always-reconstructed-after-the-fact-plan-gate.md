---
gate: plan-quality
issue: 208
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-06T23:01:13-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-09-06T23:06:41-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Plan names build and path mechanisms that do not work where it puts them
          detail: |-
            M1 adds cmd/hoprtt/main.go with no Makefile change, but build: is
            Makefile.local:80 driven by hand-maintained GO_BINS (:32) and overrides the
            base-layer cmd/*/main.go scan (Makefile:9-10 include order), so bin/hoprtt
            never exists and perf.sh has nothing to call; the plan also never says how
            perf.sh locates the probe or degrades when absent. Same rule at M2.6:
            artifactpath is a Go internal package (cmd/internal/artifactpath) but
            CaptureRecord is placed in nvim/doctor.lua, and Lua reaches paths via
            vim.env.PAIR_DATA_DIR (nvim/init.lua:484). Existing probe homes probes/<name>
            and cmd/probes/<name> are unmentioned. Sweep every named mechanism, not
            these two.
          family: unverified-repo-mechanism
          round: 2
        - id: PQ-2
          severity: Important
          title: Two-sample delta is untested shell; ARCH-MOCK claims a seam no task builds
          detail: |-
            The plan's ARCH-MOCK section says perf.sh emits a line-oriented format
            consumed by pure Lua/Go code tested against recorded fixtures, but no M1/M2
            task creates that parser or consumer. The pid-join over two samples
            (per-process CPU delta, swap rate, vanished/started counts) is pure logic
            the plan's own ARCH-ORDER table calls the most likely to be mishandled, and
            it has no named function and no test. Name it, drive it from recorded sample
            pairs, and give one strategy line covering vanished pids, reused pids, and
            truncated top output.
          family: pure-logic-in-untested-shell
          round: 2
        - id: PQ-3
          severity: Important
          title: M2.3 self-timing names no function, no threshold, and no target buffer
          detail: |-
            The Spec calls the editor-vs-environment discriminator the single most
            valuable bit, but M2.3 is one bullet: time a synthetic keystroke and a
            redraw via vim.loop.hrtime. It promises an "editor: fast / editor: SLOW"
            verdict with no threshold or basis and no test. It also does not say which
            buffer receives the synthetic keystroke -- the draft buffer is both the
            operator's note (read by M2.4) and where #202's TextChangedI chain lives, so
            timing it in place corrupts the note, and no ordering constraint is stated.
          family: discriminator-underspecified
          round: 2
        - id: PQ-4
          severity: Important
          title: Envelope never says whether the 6 s capture blocks the nvim UI
          detail: |-
            ARCH-CONSTRAINTS budgets 6 s hard but omits sync-vs-async. The integration
            table wraps vim.system; a blocking :wait() freezes the draft pane for six
            seconds at the exact moment the operator reports typing lag. State
            async-with-callback and what the operator sees, plus the ARCH-ORDER events
            the caller cannot block that the current table misses: a second :PairDoctor
            during an in-flight capture, and pane teardown mid-window.
          family: ui-path-blocking
          round: 2
        - id: PQ-5
          severity: Minor
          title: Positive-control timing band will flake under the load it exists to study
          detail: |-
            M1.1 asserts /usr/bin/true at 0.5-6.0 ms, and cmd/hoprtt/main_test.go runs
            inside make test's `go test ./... -count=1` (Makefile.local:113-114) in
            parallel with the suite. By the plan's own figures (1.5 ms baseline, 2-3x
            under a spawn storm) that reaches 4.5 ms against a 6 ms ceiling. A ceiling
            near 15 ms still fails the 18.7 ms bug the control was written to catch.
          family: wallclock-assertion-in-default-suite
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-06T23:11:19-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Build half verified against Makefile.local:32,80,336; the artifactpath half of my prior finding was mistaken — artifactpath is the authority for non-Go consumers (paths.go:558).
          round: 3
        - id: PQ-2
          disposition: addressed
          note: doctor.delta named, pure, fixture-driven at M2.2b; perf.sh explicitly does no arithmetic.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: doctor.verdict named with a 16 ms threshold and basis; scratch-buffer and note-before-timing ordering both stated as tests.
          round: 3
        - id: PQ-4
          disposition: not-addressed
          note: Async stated, but the ARCH-ORDER table is unchanged and still claims the only carried state is the sample window — second-invocation and buffer teardown are absent.
          round: 3
        - id: PQ-5
          disposition: addressed
          note: Band is now 0.5-15.0 ms with the reasoning inline.
          round: 3
      blocked: true
---

# Gate ledger — pair#208 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-06T23:01:13-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-09-06T23:06:41-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unverified-repo-mechanism` Plan names build and path mechanisms that do not work where it puts them
  M1 adds cmd/hoprtt/main.go with no Makefile change, but build: is
  Makefile.local:80 driven by hand-maintained GO_BINS (:32) and overrides the
  base-layer cmd/*/main.go scan (Makefile:9-10 include order), so bin/hoprtt
  never exists and perf.sh has nothing to call; the plan also never says how
  perf.sh locates the probe or degrades when absent. Same rule at M2.6:
  artifactpath is a Go internal package (cmd/internal/artifactpath) but
  CaptureRecord is placed in nvim/doctor.lua, and Lua reaches paths via
  vim.env.PAIR_DATA_DIR (nvim/init.lua:484). Existing probe homes probes/<name>
  and cmd/probes/<name> are unmentioned. Sweep every named mechanism, not
  these two.
- **PQ-2** [Important] `pure-logic-in-untested-shell` Two-sample delta is untested shell; ARCH-MOCK claims a seam no task builds
  The plan's ARCH-MOCK section says perf.sh emits a line-oriented format
  consumed by pure Lua/Go code tested against recorded fixtures, but no M1/M2
  task creates that parser or consumer. The pid-join over two samples
  (per-process CPU delta, swap rate, vanished/started counts) is pure logic
  the plan's own ARCH-ORDER table calls the most likely to be mishandled, and
  it has no named function and no test. Name it, drive it from recorded sample
  pairs, and give one strategy line covering vanished pids, reused pids, and
  truncated top output.
- **PQ-3** [Important] `discriminator-underspecified` M2.3 self-timing names no function, no threshold, and no target buffer
  The Spec calls the editor-vs-environment discriminator the single most
  valuable bit, but M2.3 is one bullet: time a synthetic keystroke and a
  redraw via vim.loop.hrtime. It promises an "editor: fast / editor: SLOW"
  verdict with no threshold or basis and no test. It also does not say which
  buffer receives the synthetic keystroke -- the draft buffer is both the
  operator's note (read by M2.4) and where #202's TextChangedI chain lives, so
  timing it in place corrupts the note, and no ordering constraint is stated.
- **PQ-4** [Important] `ui-path-blocking` Envelope never says whether the 6 s capture blocks the nvim UI
  ARCH-CONSTRAINTS budgets 6 s hard but omits sync-vs-async. The integration
  table wraps vim.system; a blocking :wait() freezes the draft pane for six
  seconds at the exact moment the operator reports typing lag. State
  async-with-callback and what the operator sees, plus the ARCH-ORDER events
  the caller cannot block that the current table misses: a second :PairDoctor
  during an in-flight capture, and pane teardown mid-window.
- **PQ-5** [Minor] `wallclock-assertion-in-default-suite` Positive-control timing band will flake under the load it exists to study
  M1.1 asserts /usr/bin/true at 0.5-6.0 ms, and cmd/hoprtt/main_test.go runs
  inside make test's `go test ./... -count=1` (Makefile.local:113-114) in
  parallel with the suite. By the plan's own figures (1.5 ms baseline, 2-3x
  under a spawn storm) that reaches 4.5 ms against a 6 ms ceiling. A ceiling
  near 15 ms still fails the 18.7 ms bug the control was written to catch.

## Round 3 — 2026-09-06T23:11:19-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Build half verified against Makefile.local:32,80,336; the artifactpath half of my prior finding was mistaken — artifactpath is the authority for non-Go consumers (paths.go:558).
- PQ-2 — addressed — doctor.delta named, pure, fixture-driven at M2.2b; perf.sh explicitly does no arithmetic.
- PQ-3 — addressed — doctor.verdict named with a 16 ms threshold and basis; scratch-buffer and note-before-timing ordering both stated as tests.
- PQ-4 — not-addressed — Async stated, but the ARCH-ORDER table is unchanged and still claims the only carried state is the sample window — second-invocation and buffer teardown are absent.
- PQ-5 — addressed — Band is now 0.5-15.0 ms with the reasoning inline.

## Open findings

- **PQ-4** [Important] `ui-path-blocking` Envelope never says whether the 6 s capture blocks the nvim UI
