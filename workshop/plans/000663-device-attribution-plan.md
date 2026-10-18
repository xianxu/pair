---
collaboration-mode: discussion
---

# Remaining Suite Device Writes: Attribution Plan

> **For agentic workers:** Consult AGENTS.md Section 3 for bounded delegation. Execute the approved measurement steps before proposing production changes.

**Goal:** Identify the producer of the remaining final-Go device traffic and assess whether suite concurrency changes total bytes per completed validation.

**Architecture:** Start with a controlled existing workload, separating observer cost from workload cost. Add a bounded native host census only if needed. Stop at a demonstrated permission or attribution limit rather than repeating full suites without a new question.

**Tech stack:** macOS disk0 IORegistry counters, procutil native resource counters, existing couchcore tests, the normal machine test lock, and retained JSON evidence.

## Revisions

- 2026-10-18: Hub approved only section 2's single observation comparison (receipt f22cead8-9b80-4ee6-a1e4-8e4d9b2e7a3d). Section 3's host census is cancelled: if attribution remains open, provide a scoped 60-second fs_usage command for the operator. Section 4's concurrency experiment is deferred until the writer is known. These decisions supersede the unchecked alternatives below. The comparison uses five complete invocations per arm, full observation first, endpoints only second, with 60 seconds per arm. This is investigation using temporary instrumentation and unchanged tests; any repository implementation still requires change-code.

## Core concepts

This is an attribution plan; no production entity or public interface is selected.

| Pure entity | Lives in | Status |
|---|---|---|
| Measurement comparison | workshop/plans/000663-device-attribution-report.md | new report |

Each comparison owns its source head, counter identity, endpoints, completed operation count, duration, coverage errors and observation mode. Process generations are keyed by PID and start identity. Existing processes require an interval baseline; lifetime totals cannot be charged to the interval. Counter resets and missing samples make a comparison incomplete rather than zero. ARCH-PURE: arithmetic and validity checks stay separate from OS observation.

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| NativeResourceReader | cmd/internal/procutil/resource*.go | existing, reuse | bounded enumeration and identity-fenced resource counters |
| Device counter reader | retained #658 diagnostic driver, adapted only after approval | existing diagnostic seam | disk0 driver identity and Bytes (Write) |
| Owned workload | cmd/internal/couchcore existing tests | unchanged | real filesystem and Git operations |
| Suite admission | scripts/test-lock.py in the slot's ariadne | unchanged | ordinary machine tokens |

ARCH-DRY: reuse native observation and admission instead of introducing another production sampler. ARCH-PURPOSE: any temporary diagnostic exists only to answer a named attribution question. No product lifecycle or vocabulary changes are proposed.

## 1. Commit the evidence and obtain approval

- [ ] Save the retained-data findings in the issue and commit this plan.
- [ ] Send the hub the control, bounds, escalation rule and concurrency question; wait for approval before diagnostic code or fresh measurements.

Both prior complete measurements ran one Make while holding both tokens. Final Go wrote 15.554 GB / 358.452 s before and 13.312 GB / 325.877 s after. Traffic falls near the tail while observer cadence remains steady. This weakens a fixed-cost observer hypothesis; it does not identify the writer. APFS counters overlap and cannot partition device bytes.

## 2. First control: identical work, different observation

- [ ] Pin the current issue head and compile couchcore once outside the measurement interval. Reuse the existing TestSlotPreferencesIndependentAcrossRestartResumeAndFresh workload, which exercises real fixture and durability paths. Confirm the precise test selection from source before execution.
- [ ] Prepare two isolated owned temporary roots with identical normal cache conditions. Run the same fixed number of complete test invocations in each arm; select that count once before either arm and retain it. Do not disable fsync, change assertions, or alter test timing.
- [ ] Reserve both ordinary suite tokens for the device comparison and notify the hub at start/end. Record 30-second quiet windows around the pair, without claiming they permit exact background subtraction.
- [ ] Compare full #658 observation against device endpoints only: no in-loop ps, ioreg, native process queries or FD scans in the endpoint-only arm. Cap each arm at 60 seconds; an unfinished invocation makes the comparison incomplete. One pair only, no automatic reruns.
- [ ] Record bytes per completed invocation, duration, driver identity, failures and observer output/process costs. Flush diagnostic buffers outside measured intervals where practical. Similar totals falsify a dominant observer contribution for this workload only; a large difference motivates isolating observer components.

ARCH-CONSTRAINTS: fixed operations and explicit time bounds prevent a diagnostic loop. ARCH-ORDER: counter endpoints bracket actual work, and every failed invocation remains recorded.

## 3. Bounded host attribution, only if the control leaves the gap

- [ ] First run one 60-second native census feasibility window, at most 1 Hz and 4,096 PIDs per sample. Reuse NativeResourceReader's identity fences. Record permission failures, vanished generations, counter decreases and overflow explicitly; stop and mark incomplete on a bound violation.
- [ ] Baseline pre-existing processes, charge only interval deltas, and separate observer, owned workload and other visible host writers. Do not collect argv or environment. Resolve executable names only as needed for the largest visible writers; no initial whole-host FD scan.
- [ ] If feasible, combine this census with one bounded repeat of the same owned workload, using the previously fixed operation count and a 60-second bound. Keep device and process views distinct; counters are coverage clues, not an additive filesystem partition.
- [ ] If brief children, permissions or filesystem activity still prevent attribution, stop. Send the hub a ready-to-paste 60-second operator fs_usage command with exact process/path scope and cleanup, rather than running more unprivileged probes.

ARCH-SECURE: record process identity and counters without unrelated command arguments. ARCH-FUNERAL: supervisors own newly created process groups, cancel only those groups, reap direct children and release the admission lease on exit. Partial evidence survives cancellation; do not claim detached descendants were all reaped without proof.

Any new diagnostic comparison logic must first check synthetic traces for PID reuse, reset, denied reads and incomplete work. Real native reads supply live conformance. Keep diagnostic scripts and exact source hashes with the report; choose their repository location at the implementation gate, not an invented public API now.

## 4. Concurrency: equal completed work, not instantaneous rate

- [ ] Report what existing evidence proves: excluding another full suite does not eliminate the gap. It does not measure the effect of running two suites concurrently.
- [ ] If a cheap workload comparison is useful after attribution, compare two completed copies serially with the same two copies concurrently, using isolated roots and unchanged checks. Label it a pilot, not bytes per landing.
- [ ] Before any whole-suite comparison, ask the hub to approve its cost and window. The smallest matched experiment is two complete suites serially versus two complete suites concurrently, on one immutable head in independent owned checkouts with normal caches and real admission. Retain warm-cache/order/background confounds; cap each suite at the prior 45-minute diagnostic bound and stop on a failed arm.
- [ ] Compare total device bytes divided by successful suite completions, with failed-attempt bytes separate. Duration and MB/s are secondary. Do not edit WF_TEST_SUITES, protected gate configuration, or other slots' processes.

## 5. Decision and fix boundary

- [ ] Commit a report naming the largest measured producer/path, uncertainty, and the next evidence needed. Preserve before/after endpoints, quiet slopes, source hashes and coverage counts in durable prose/tables; temporary raw files are supporting evidence only.
- [ ] Send the attribution and concurrency conclusion to the hub. If a production fix is supported, append its Spec/Plan revision and counted byte/file regression budget, obtain approval, then run sdlc change-code before implementation. Retain every test and durability guarantee.
- [ ] If an operator trace is required, provide the concrete bounded command and state the blocker. Do not choose a convenient low-impact writer as the cause of the unexplained majority.

No operator UI, pair:0 staging, dependency installation, system-wide configuration change or production optimization is part of this measurement plan. No estimate is entered before the implementation scope clears change-code's gates.
