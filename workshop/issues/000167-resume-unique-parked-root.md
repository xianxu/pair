---
id: 000167
status: open
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
---

# Resume unique parked root on Couch startup

## Problem

`Leave Couch` deliberately parks every active thread, including the home/root
actor. Starting `couch` again at the same repository currently creates a new
root, leaving the former root available only as a parked child in the panel.
The native conversation is recoverable, but its home role is not, and the
operator must create a disposable conversation just to reach it.

## Spec

On interactive `couch [<repo>]` startup, before preparing a new root, resolve
the requested path to the same normalized repository scope and physical working
path used by Couch admission, then inspect the existing proof-bearing
actionable thread inventory for that exact target. Physicalize candidate paths
in the thin inventory/startup shell and feed only normalized values to the pure
unique-candidate decision; symlink or alias spellings of the same path must not
split identity.

- If **exactly one** matching thread is in the verified `parked` state and has
  exact Resume authority, Resume that thread and install its returned handle as
  the console's root/home actor.
- If there are zero matching candidates, retain today's behavior and create a
  new root.
- If there is more than one matching candidate, treat the set as non-resumable
  and retain today's behavior: create a new root. Do not add a picker, prompt,
  ranking rule, warning workflow, or remembered “last root” identity.
- Never infer eligibility from raw ThreadStore records or labels. Reuse the
  same verified-park and exact native-binding proof consumed by the existing
  Resume operation (`ARCH-DRY`, `ARCH-PURPOSE`).
- A failure to read the authoritative ThreadStore snapshot or actionable
  inventory is a startup error and creates no root. A single record whose path
  cannot be physicalized or whose Resume binding cannot be proven is omitted as
  non-resumable, matching the existing conservative actionable projection; it
  does not invalidate otherwise authoritative inventory.
- Keep the decision pure and the launch/console effects in the existing thin
  startup shell (`ARCH-PURE`). A selected candidate that becomes invalid
  during Resume must fail conservatively through the existing Resume
  diagnostics; it must not silently fall through to creating another root.
- Startup remains bounded by the existing local inventory and Resume work; it
  must not add a fleet-wide scan, prompt, or unbounded retry to the interactive
  path (`ARCH-CONSTRAINTS`). No new external dependency or double is introduced
  (`ARCH-MOCK`: N/A).

## Done when

- `Leave Couch`, followed by `couch` at the same path, resumes the sole exact
  verified-parked thread with its saved native conversation and makes it the
  root/home actor.
- Zero matching parked threads creates a new root as today.
- Two or more matching parked threads create a new root without selecting or
  resuming either candidate.
- Equivalent symlink/alias spellings of the requested and stored working path
  still match after physical normalization.
- Live, ambiguous, unverified, wrong-repository, and wrong-path records are not
  startup-resume candidates.
- Whole-snapshot/inventory failure is surfaced and creates no root; per-record
  path or binding failure conservatively excludes only that record.
- A Resume refusal after selection is surfaced and does not create a fallback
  root.
- Automated pure-decision, CLI/startup integration, and restart-level tests
  cover the success, fallback, and race/refusal paths.

## Plan

- [ ] Design the pure unique-candidate startup decision over actionable
  inventory and pre-normalized path identity, and add its decision-table tests.
- [ ] Route interactive startup through Resume when the decision returns one
  candidate, preserving the existing new-root path otherwise.
- [ ] Prove console root installation, exact native-session preservation,
  multiple-candidate fallback, and conservative Resume failure end to end.

## Log

### 2026-09-01

Captured from operator experience after using Leave Couch: the desired narrow
behavior is automatic adoption only for one exact safe candidate. Multiple
candidates intentionally receive no picker or heuristic; startup creates a new
root as it does today.
