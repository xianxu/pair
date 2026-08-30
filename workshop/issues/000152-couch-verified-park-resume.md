---
id: 000152
status: working
deps: [149, 155]
github_issue:
created: 2026-08-25
updated: 2026-08-29
estimate_hours:
started: 2026-08-27T12:04:27-07:00
---

# couch: verified park and resume lifecycle

## Problem

`#149` makes a durable work thread distinct from its optional live actor
incarnation. The thread menu needs a normal way to make a live thread inactive
without deleting its draft, ledger, transcript, native-session identity, name,
or description. Signalling couch's top child is insufficient: Pair hosts a
zellij server session whose panes can outlive that client, so freeing the
concurrency slot at that point would admit a second writer beside the first.

The same quiescence proof is needed by `#135` before transferring a live tag to
another agent. This issue owns that reusable lifecycle protocol; the menu in
`#151` is only a client.

## Spec

**Park is an observed all-or-fail transition.** It succeeds only after the
Pair-owned zellij session is absent and every Pair-recorded
agent/wrapper/editor PID is dead or has a mismatched process identity. Until
then the incarnation remains occupied. Unknown liveness, partial shutdown, or
timeout is a failed park; the thread remains live/unknown and its admission
slot stays consumed. Park does not wait for an LLM turn or a separate semantic
flush acknowledgement: ariadne-style work is durable in the repository, and
#155's identified native transcript tree is the reconstruction substrate.

The coordinator owns one ordered transaction: request graceful shutdown,
verify the complete recorded process set and exact Pair session absence,
preserve #155's native session-tree binding, persist the incarnation removal
plus a monotonic `last_active_at`, then publish the parked result. It never
records the timestamp or frees capacity before verification. Forced kill is a
separately named recovery action and does not silently masquerade as park.

The process inventory and shutdown seam are shared with `#135`; impossible
fake-only cleanup is forbidden. Tests use a stateful Pair/zellij/process fake
whose sessions and PIDs change only in response to modeled operations, plus a
real process-level conformance probe. Recovery after interruption is explicit:
re-running park resumes verification from durable transaction state rather than
starting a second transition or guessing success (ARCH-MOCK, ARCH-PURPOSE).

The coordinator commits verified park, `last_active_at`, and incarnation
removal through #149's locked `ThreadStore` transaction; it does not own a
parallel lifecycle file. This issue enables `couch resume <ref>` once that fact
exists. Resume accepts only a uniquely resolved, verified-parked thread. It
refuses live, unknown, partially parked, legacy-unverified, missing-path, and
ambiguous records with an exact recovery diagnostic; it never guesses that
absence means quiescence.

Resume uses the thread's latest successfully registered launch profile—the same
agent, exact argument vector, and saved native-session identity. It does not
consult the path's new-thread preference, root agent, or current repository
default. A missing/unsupported agent or invalid saved profile refuses with an
exact recovery diagnostic rather than silently changing harnesses. #153 may
later intercept a missing managed `working_path`, deterministically reprovision
and rebind it, then call this unchanged resume operation on the now-valid path;
until that integration lands, missing paths remain a refusal.

## Revisions

### 2026-08-28 — recovery uses deterministic native session trees

**Reason:** an LLM may be unavailable or overloaded during park, while the
repository and native transcripts already preserve reconstruction material.
The actual missing evidence is a reliable mapping from Pair tags to complete
agent session trees, including subagents.

**Delta:** remove the semantic flush-acknowledgement requirement. Park proves
writer and Pair-session absence before releasing capacity and preserves the
session-tree binding from #155. #152 now depends on #155's deterministic,
fail-closed forest inventory and correlation instead of treating a newest or
first transcript probe as recovery truth (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-29 — Couch park is Pair full quit, not detach

**Reason:** design review exposed a noun mismatch. Pair already has the lifecycle
the operator means by “park”: Alt+x fully quits the Pair/Zellij session while
retaining the durable information needed by `pair resume <tag>`. Alt+d merely
detaches a client while the session remains live; Couch switching already covers
that useful behavior, so exposing detach would add a confusing duplicate state.

**Delta:** this revision supersedes the earlier independent process-coordinator
design. Couch does not build a second PID inventory, shutdown protocol, or
forced-cleanup implementation. Pair's existing Alt+x path remains the sole owner
of full-session teardown: quit marker, exact Zellij-session quiescence, embedded
editor cleanup, live-sidecar cleanup, durable session/config preservation, and
the established #155 native binding. The lifecycle core gains a typed,
durable completion result so direct Alt+x and Couch-driven park execute the same
behavior and Couch never infers success merely from a client disconnect.

Couch park first commits one `parking` transaction against the exact composite
thread address and expected live incarnation, then requests Pair full quit with
that transaction's stable nonce. The Couch menu's explicit confirmation replaces
Pair's local Alt+x confirmation; it does not synthesize keystrokes or show a
second modal. Because the Couch action is explicitly named park, it selects the
existing preserve-scrollback cleanup branch without another nudge; direct Pair
Alt+x retains its current optional scrollback prompt. Pair publishes exactly one
nonce-bound completion after its full quit cleanup succeeds. A client exit,
missing completion, cleanup warning/error, timeout, replacement incarnation, or
revision conflict leaves the thread occupied with an exact retryable diagnostic.
Reissuing park joins or resumes the durable transaction rather than spawning a
second teardown.

Only the matching completion may atomically remove the expected incarnation,
record a monotonic `last_active_at`, and mark the durable thread verified parked.
Every durable resumability artifact remains owned by Pair and #155; Couch stores
only lifecycle state and the latest exact successful launch profile needed to
re-enter the same address. Restart recovery consumes the same nonce-bound result
and cannot convert absence alone into proof.

Resume accepts only one exact verified-parked thread. It reuses the same
`{repo_scope, tag}`, latest successful agent and argument vector, current working
path, and established native-session binding through Pair's existing
`resume <tag>` flow. It never allocates a new tag or consults new-thread path,
root-agent, or repository defaults. Live, parking, failed/unknown, ambiguous,
missing-path, invalid-profile, unsupported-agent, and non-established native
bindings refuse before starting a child. There is no Couch detach operation
(ARCH-DRY, ARCH-PURPOSE).

The operating envelope is explicit (ARCH-CONSTRAINTS): park confirmation must
publish `parking` within 100 ms on the target development machine and must never
hold the UI beyond 1 second; full quit continues asynchronously under Pair's
existing bounded cleanup. One coordinator may exist per thread, repeated calls
coalesce by nonce, and no corpus/process-tree scan enters the interaction path.
Timeout or unavailable evidence fails closed with capacity still occupied.
Resume uses the existing local start path and adds no full native-inventory scan;
its startup tolerance is the same as ordinary `pair resume`.

The revised acceptance contract is:

- Couch park and direct Pair Alt+x share one full-quit lifecycle implementation;
  a regression in either path fails the same stateful lifecycle tests.
- The UI returns promptly with `parking`, while only a matching successful Pair
  cleanup completion releases the incarnation and records `last_active_at`.
- Interrupted, failed, timed-out, duplicated, or stale park attempts remain
  occupied and are recoverable without repeating teardown.
- Resume recreates the same composite thread through the exact saved profile and
  established #155 binding; no fallback may create a different conversation.
- Couch exposes park and resume through its shared operation schema and exposes
  no detach operation.

## Done when

- A live Pair thread parks only after zellij absence and recorded-process death
  are both observed, without depending on an LLM response.
- Partial, unknown, timeout, and interrupted transitions retain the admission
  slot and give an exact retry/recovery state.
- Successful park preserves every tag-scoped artifact and #155 native
  session-tree binding, and records one monotonic `last_active_at`; resuming
  uses the same composite address and latest exact successful launch profile.
- `#135` can consume the same quiescence proof without a second coordinator.
- Stateful fake and real process-level conformance cannot claim cleanup the
  production stack did not perform.

## Plan

- [ ] Model the park transaction, process inventory, and recovery states as a
      pure transition system consuming #155's session-tree binding.
- [ ] Put Pair/zellij/process shutdown and observation behind one stateful seam.
- [ ] Implement graceful park, verification, atomic slot release, and recency.
- [ ] Exercise resume-after-park and the shared #135 quiescence boundary.
- [ ] Document operator recovery and wire the lifecycle into #151.

## Log

### 2026-08-25

Split from #149 after fresh spec review showed that “stop every process that can
write and flush durable output” is a process-coordination protocol, not a field
on the durable thread record.

- 2026-08-29: unblocked after #149/#155 completion. Operator clarified that
  Couch park is Pair Alt+x full-quit semantics, while Alt+d detach has no Couch
  operation; revised the design around one shared Pair lifecycle path.
