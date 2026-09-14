---
id: 000253
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
started: 2026-09-14T13:56:39-07:00
---

# Record attachment disconnect causes

## Problem

Astro lost its Couch attachment helper and Zellij client while its Zellij server
and Claude survived. Existing notices are transient; procutil.WaitCode and the
launcher handoff collapse signal termination into an integer. PTY read errors
are discarded. The available logs cannot identify this incident's trigger.
The operator requests durable evidence for the next disconnect alongside #250.

## Spec

Record attachment lifecycle events by default in local, bounded diagnostic logs.
Correlate Couch owner, thread scope/tag, helper PID/start identity, Zellij session
and client PID/start identity. Capture start, PTY read termination, child wait
result (normal exit code versus signal, raw wait status where available), and
application-requested signal/close/detach/park/quit with reason and result.
Record requests before their effects and keep observed results separate from
inferred causation. A process disappearing without a wait result is unknown,
never a fabricated clean exit. Do not add a background process per thread.

Use a shared structured wait-result decoder behind existing integer-returning
compatibility APIs (ARCH-DRY), and an injected event sink at PTY/launcher/owner
boundaries. Events contain metadata, not terminal output, keystrokes, prompts,
environment dumps or arbitrary argv. Distinguish PTY EOF/EIO/local close from
unexpected errors; a PTY EOF alone does not prove why the child exited.

The production sink uses private local JSONL files with a fixed writer-enforced
retention bound. Prefer an existing suitable bounded writer; current optional
Couch trace files have neither rotation nor structured process results, and
the agent lifecycle journal has a different contract. Resolve diagnostic paths
through the artifact-path authority. Logging failure must not prevent recovery
or shutdown, but must produce a visible rate-limited diagnostic. Do not perform
disk writes while holding Console state locks or make per-output-chunk events.
Specify exact rotation/concurrency and write-failure behavior in the plan
(ARCH-CONSTRAINTS, ARCH-FUNERAL, ARCH-ORDER).

Application-originated shutdown requests can establish local intent. A signal
from an external sender remains unattributed unless independent evidence names
the sender. This instrumentation cannot reconstruct Astro's past exit or claim
that every future disconnect will yield a definitive external trigger.

Alternatives considered: extending only COUCH_TRACE misses launcher-side signal
information and requires advance opt-in; an OS-wide process monitor adds scope
and platform dependence without fixing the discarded wait results. Recommend
structured events at the existing process ownership boundaries.

## Done when

- Default diagnostics distinguish clean/nonzero exits, signal death, PTY read
  failure, and application-requested teardown, preserving exact identities.
- Real disposable-process tests verify normal exit, SIGTERM/SIGKILL, and local
  close; stateful tests verify ordered request/result correlation, startup and
  early-exit races, bounded retention, concurrent writers and failed writes.
- Existing launch/attach exit semantics and #250 recovery behavior remain valid.
- Documentation names log location, retention, evidence limits and a practical
  incident-inspection procedure. SDLC boundary review passes before shipping.

## Plan

- [ ] Review the telemetry design and record the approved implementation plan.
- [ ] Implement process results and correlated bounded diagnostics with tests.
- [ ] Verify disposable disconnects, document inspection, and close through SDLC.

## Log

### 2026-09-14

Created and claimed following the operator's explicit telemetry request. Read
ptychild pump/Signal/Close, procutil.WaitCode, launcher runBlockingHandoff and
Couch onExit. Both wait paths currently discard signal distinctions; PTY pump
discards its terminal read error. This is evidence collection, not a speculative
fix for Astro's unknown trigger. #250 recovery implementation continues in
parallel; telemetry design has not yet crossed change-code.
