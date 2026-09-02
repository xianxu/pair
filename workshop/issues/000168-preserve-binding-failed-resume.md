---
id: 000168
status: open
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
---

# Preserve parked binding after failed resume

## Problem

Resuming a verified parked Couch thread can fail after Pair appends a new
launch row but before it appends the corresponding binding. The session ledger
then treats that newest, incomplete launch as current and no longer projects
the earlier established binding. The thread disappears from Couch's resumable
inventory even though its verified park record and native agent transcript are
still intact and resumable.

Observed on `couch-71d653a6fc8f615d`: two attempts rebound Codex session
`01a05f9d-76cf-7ca3-8133-f21eeb3e0798`; a later attempt left only a launch
row. The exact owner query became provisional/empty and hid the parked thread.

## Spec

- A failed resume attempt must not destroy or shadow the last established
  binding for a verified parked thread.
- Retry bookkeeping must distinguish a pending launch attempt from committed
  current binding authority.
- After rollback/reconciliation, Couch must continue to expose the parked
  thread as resumable when the prior native session remains authorized and
  resumable.
- Recovery must remain fail-closed when prior binding evidence is invalid,
  disputed, unauthorized, or no longer resumable.

## Done when

- A regression test starts from a verified park plus established binding,
  injects failure after the next launch append but before binding publication,
  and proves the same thread remains actionable and resumable.
- Repeated failed attempts do not progressively remove the thread from the
  resumable inventory or replace its last established native ID.
- A successful retry commits the new launch/binding pair without retaining
  ambiguous provisional authority.
- Negative tests prove invalid prior evidence is not resurrected.

## Plan

- [ ] Model pending resume attempts without replacing established binding authority.
- [ ] Reconcile failed attempts back to the prior verified parked projection.
- [ ] Add boundary regressions through the authoritative ledger inventory and Couch startup path.

## Log

### 2026-09-01

Captured from a live failure investigation. The ThreadStore record remained
verified parked with no incarnation, and the Codex transcript remained
authorized and resumable; only the ledger's newest incomplete launch caused
the actionable projection to disappear. This violates `ARCH-PURPOSE`: attempt
history is being used as current resumability authority.
