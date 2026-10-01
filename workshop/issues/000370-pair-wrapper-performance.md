---
id: 000370
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '624e74e1f31050f7ab6ed2d949195dd5aef63bab' # card fields mirrored from issue-cards; edit via sdlc
---

# Investigate Pair wrapper CPU and diagnostic logging overhead

## Problem

Pair itself consumes appreciable CPU even when visible terminal traffic looks modest. During the 2026-10-01 workbench investigation, `pair wrap` PID 58109 used 0.87 CPU-seconds over a five-second window (about 17.4% of one core). We need a measured explanation and a reproducible way to evaluate improvements, rather than assuming the Couch messaging polling fix also resolves this cost.

## Spec

Investigate Pair wrapper CPU using profiles first, then focused source inspection and controlled comparisons. Diagnostic logging amplification is a leading hypothesis, not an established attribution: about 12.9 KB of raw terminal output generated 224 KB / 1,073 diagnostic records in one five-second observation. A separate 4.84-second window contained 366 `master-chunk`, 367 `stdout-queue`, 367 `scrollback-write`, and 49 stdout flush records.

The ordinary sample includes terminal snapshots/parsing, diagnostic-log frames, and substantial open/rename/write/close activity. It also includes blocked reads and sleeping threads; sample proportions must not be presented as CPU shares. `traceWrap` writes synchronously through `diagnosticlog.Writer.Write`, which loads/validates metadata, saves append intent, appends and verifies payload, and saves final metadata; metadata publication stages and renames JSON files. Determine the actual cost of this path relative to terminal processing, snapshotting, polling and other work.

Use repeatable idle, modest-output, and burst-output workloads. Measure process CPU-time deltas, output bytes, diagnostic record/byte volume, and relevant filesystem operations over the same window. Record binary/source identity, configuration and workload so results survive process restarts. Isolate suspected contributors with controlled comparisons in a disposable test environment; preserve production diagnostic evidence and terminal behavior.

Related: #365 removes messaging-related global ownership/liveness polling in the cross-slot-work-scheduling project. This investigation is independently actionable and does not depend on #365 or expand that project's eight-task MVP. Distinguish shared causes from independent wrapper overhead; do not attribute improvements from a different binary or workload to the hypothesized mechanism.

The deliverable is a causal investigation and a concrete remediation proposal. Any implementation follows its own reviewed design and SDLC gates; do not silently trade away log recovery guarantees or composer/input correctness for lower CPU.

## Done when

- Reproducible commands and workload/configuration details establish CPU, terminal-byte and diagnostic-volume baselines for idle and active Pair wrappers.
- Profiles and a controlled comparison confirm or reject diagnostic logging amplification as a material contributor; report absolute CPU costs and measurement limits, separating waiting time from CPU time.
- Results distinguish messaging polling (#365), terminal/snapshot work and logging overhead sufficiently to choose the next change, without claiming unmeasured contributions are zero.
- A durable report records findings and proposes the smallest justified remediation, including the purpose and failure behavior of any logging integrity checks it would alter. Concrete follow-up implementation tasks have acceptance criteria and link to the evidence.
- A repeatable regression workload or test captures the confirmed expensive behavior and states how a later fix will be evaluated, including terminal/composer correctness and diagnostic recovery checks where affected.

## Plan

- [ ] After claim/start-plan, capture repeatable baselines and a suitable CPU profile before deeper code investigation.
- [ ] Compare suspected contributors under a controlled workload and preserve the evidence with its limitations.
- [ ] Record the causal findings, regression workload and remediation tasks for review.

## Log

### 2026-10-01 — filed from live performance observations

The operator requested a separate Pair performance investigation after reviewing cross-slot-work-scheduling. Initial local evidence: `/tmp/pair-wrap-58109.sample.txt` (five-second ordinary sample, including waits). PID 58109 was the Pair wrapper for the active Codex pane; PID and temporary path are historical evidence, not portable prerequisites. CPU/log-volume observations above came from bounded read-only measurements; no controlled causal comparison or performance fix has been performed. Created as open, without a claim or implementation assignment.
