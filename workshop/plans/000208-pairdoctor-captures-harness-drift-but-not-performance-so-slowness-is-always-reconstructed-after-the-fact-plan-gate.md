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
    - "n": 4
      timestamp: "2026-09-06T23:13:56-07:00"
      agent: claude
      dispose:
        - id: PQ-4
          disposition: addressed
          note: |-
            ARCH-CONSTRAINTS names UI-blocking as the binding constraint and commits to
            vim.system+on_exit; ARCH-ORDER adds the re-invoke and buffer-teardown events.
          round: 4
      findings:
        - id: PQ-6
          severity: Minor
          title: M2.6 routes the Lua rolling-file write through artifactpath, a Go internal package Lua cannot call
          detail: |-
            2nd finding in this family, so the rule is the deliverable, not the site: a
            plan may name a repo mechanism only with a file:line AND a check that it is
            reachable from the language and layer the calling code sits in. Prevalence
            2/2 rounds that named a mechanism — PQ-1 failed build-path reachability,
            this fails language reachability. artifactpath is cmd/internal/artifactpath
            with no CLI surface; CaptureRecord lives in nvim/doctor.lua. Use the
            existing idiom, pair_data_dir() at nvim/init.lua:484 plus pair_tag()
            (nvim/init.lua:4012). Sweeping the plan's other named mechanisms under the
            same rule found one more: the binary is named `hoprtt`, but
            Makefile.local:5-7 requires the `pair-` prefix on every GO_BINS entry
            because make install (:84-87) puts each on PATH — the collision that
            comment records already fixing once for `scribe`. Name it `pair-hoprtt`.
          family: unverified-repo-mechanism
          round: 4
      blocked: false
content_hash: 6367981e5dc4c02c0d97030d1b9e7007d85f0895c976d7b6a95fb7e2a36a0e2d
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

## Round 4 — 2026-09-06T23:13:56-07:00 (claude) — passed

### Disposed

- PQ-4 — addressed — ARCH-CONSTRAINTS names UI-blocking as the binding constraint and commits to
vim.system+on_exit; ARCH-ORDER adds the re-invoke and buffer-teardown events.

### Raised

- **PQ-6** [Minor] `unverified-repo-mechanism` M2.6 routes the Lua rolling-file write through artifactpath, a Go internal package Lua cannot call
  2nd finding in this family, so the rule is the deliverable, not the site: a
  plan may name a repo mechanism only with a file:line AND a check that it is
  reachable from the language and layer the calling code sits in. Prevalence
  2/2 rounds that named a mechanism — PQ-1 failed build-path reachability,
  this fails language reachability. artifactpath is cmd/internal/artifactpath
  with no CLI surface; CaptureRecord lives in nvim/doctor.lua. Use the
  existing idiom, pair_data_dir() at nvim/init.lua:484 plus pair_tag()
  (nvim/init.lua:4012). Sweeping the plan's other named mechanisms under the
  same rule found one more: the binary is named `hoprtt`, but
  Makefile.local:5-7 requires the `pair-` prefix on every GO_BINS entry
  because make install (:84-87) puts each on PATH — the collision that
  comment records already fixing once for `scribe`. Name it `pair-hoprtt`.

## Open findings

- **PQ-6** [Minor] `unverified-repo-mechanism` M2.6 routes the Lua rolling-file write through artifactpath, a Go internal package Lua cannot call
