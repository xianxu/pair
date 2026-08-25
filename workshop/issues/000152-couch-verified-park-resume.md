---
id: 000152
status: open
deps: [149]
github_issue:
created: 2026-08-25
updated: 2026-08-25
estimate_hours:
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
Pair-owned zellij session is absent, every Pair-recorded agent/wrapper/editor
PID is dead or has a mismatched process identity, and all durable writers have
acknowledged their final flush. Until then the incarnation remains occupied.
Unknown liveness, partial shutdown, timeout, or flush failure is a failed park;
the thread remains live/unknown and its admission slot stays consumed.

The coordinator owns one ordered transaction: request graceful shutdown,
collect flush acknowledgement, verify the complete recorded process set,
persist the incarnation removal plus a monotonic `last_active_at`, then publish
the parked result. It never records the timestamp or frees capacity before
verification. Forced kill is a separately named recovery action and does not
silently masquerade as park.

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

## Done when

- A live Pair thread parks only after zellij absence, recorded-process death,
  and durable-flush acknowledgement are all observed.
- Partial, unknown, timeout, and interrupted transitions retain the admission
  slot and give an exact retry/recovery state.
- Successful park preserves every tag-scoped artifact and records one monotonic
  `last_active_at`; resuming uses the same composite address and latest exact
  successful launch profile.
- `#135` can consume the same quiescence proof without a second coordinator.
- Stateful fake and real process-level conformance cannot claim cleanup the
  production stack did not perform.

## Plan

- [ ] Model the park transaction, process inventory, acknowledgements, and
      recovery states as a pure transition system.
- [ ] Put Pair/zellij/process shutdown and observation behind one stateful seam.
- [ ] Implement graceful park, verification, atomic slot release, and recency.
- [ ] Exercise resume-after-park and the shared #135 quiescence boundary.
- [ ] Document operator recovery and wire the lifecycle into #151.

## Log

### 2026-08-25

Split from #149 after fresh spec review showed that “stop every process that can
write and flush durable output” is a process-coordination protocol, not a field
on the durable thread record.
