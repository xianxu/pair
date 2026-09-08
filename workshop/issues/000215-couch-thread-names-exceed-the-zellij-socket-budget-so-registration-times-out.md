---
id: 000215
status: open
deps: []
github_issue:
created: 2026-09-08
updated: 2026-09-08
estimate_hours:
---

# couch thread names exceed the zellij socket budget so registration times out

## Problem

Starting a thread on `../pair/` from couch fails with:

```
error: await Pair registration {RepoScope:e108517d46ab4575 Tag:couch-675d94857e70ca61}: context deadline exceeded
```

**The session name does not fit zellij's socket path.** Verified against zellij's own validator,
which is the oracle `ProbeSessionName` already uses:

```
$ zellij --session "📁pair-couch-675d94857e70ca61" action list-clients
error: Invalid value "📁pair-couch-675d94857e70ca61" for '--session <SESSION>':
       session name must be less than 0 characters
```

| name | bytes | zellij |
|---|---|---|
| `📁pair-couch-25` | 17 | accepted |
| `📁brain-couch-23` | 18 | accepted |
| **`📁pair-couch-675d94857e70ca61`** | **31** | **rejected** |

The budget is small because the socket directory is macOS's long temp path —
`/var/folders/07/…/T/zellij-501/contract_version_1/` at **78 characters**, against a 104-byte
`sun_path` limit, leaving ~25 bytes for the whole name. `📁` alone costs 4 (it is one rune and four
UTF-8 bytes), so the usable remainder is ~21. Note zellij's own arithmetic underflows and reports
the remaining budget as *"less than 0 characters"*, which is why the message is useless as a clue.

**Why the live threads work and a new one cannot.** Every currently-live thread carries a short
public name — `couch-25`, `couch-23`, `couch-13` — 17–18 bytes, already at the edge. The failing
path composes the name from the **full 16-hex opaque tag** (`couch-675d94857e70ca61`), which is 31
bytes and cannot fit under any `$TMPDIR` this machine will produce.

### The failure chain, and why it is invisible

1. couch mints `couch-<16 hex>`.
2. `ComposeSessionName` produces `📁pair-couch-<16 hex>` — 31 bytes.
3. `LaunchSession` (`launcher/osruntime.go:117-122`) passes it as `--session`; zellij refuses.
4. No `📁` session is ever created, so pair never writes `thread-claim-<tag>.json` with
   `state:"established"`.
5. `awaitThreadRegistration` (`couchcore/couch.go:672-693`) polls `Artifacts.Registration(address)`
   every 10ms for evidence that can never appear, and reports
   `context deadline exceeded` — naming neither zellij, nor the session, nor a length.

Debris observed: **six orphan zellij servers with random names** (`hopeful-tiger`,
`judicious-lemur`, `jumping-pepper`, `zippy-galaxy`, …), created 09:54–11:07 across the failed
attempts, each with **no children and `ppid=1`**, and an empty screen. They accumulate per attempt
and nothing reaps them.

### The guard exists and did not fire

This is the defect, more than the length itself. `ProbeSessionName`
(`launcher/osruntime.go:105-113`) exists to ask zellij whether a name is acceptable, and
`sessionNameFits` / `defaultSessionNameBudget` exist to explain a refusal. The budget constant even
documents its own role:

> a MESSAGE default only — never an acceptance test. zellij's real allowance is its socket path's,
> which varies with username and is a different path entirely on Linux, so the probe stays the
> oracle and this number only makes the refusal quotable.

The probe is the oracle and **the couch launch path does not consult it**. So a check that would
have refused in milliseconds with an accurate message is bypassed, and the operator instead waits
out a registration timeout that names nothing useful.

## Spec

**Names that cannot fit must be refused before launch, and thread names must be short enough to fit.**

Two halves; the second is the fix and the first is the guard that should have caught it.

**1. Probe before launching.** The couch launch path calls `ProbeSessionName` (or the same
`sessionNameFits` decision against a probed budget) before `LaunchSession`, and refuses with a
message naming the composed name, its byte length, and the measured budget. A refusal in
milliseconds beats a 30-second timeout that names nothing.

**2. Give couch threads a short public name.** The live threads already demonstrate the scheme —
`couch-25`, `couch-23` — so the composer must use that short public form rather than the opaque
16-hex tag. Where that short name is allocated, and why the new-thread path is not getting one, is
the thing to find; the tag stays the durable identity (`#149`), and the public name is a display
mapping (`atlas/session-identity.md`).

Budget the name against the **measured** socket path, not a constant. `$TMPDIR` differs per user and
per OS, and this machine leaves ~21 usable bytes after the `📁` prefix.

**3. Reap the debris.** A refused or failed launch should not leave an unnamed zellij server running
with `ppid=1`. Six accumulated in one session of retries.

## Done when

- Starting a thread on a repo whose composed name would exceed the budget **refuses immediately**,
  naming the name, its length, and the budget — asserted by a test at a Linux-sized and a
  macOS-sized budget, since `sessionNameFits` already takes the limit as a parameter for exactly
  this.
- couch can start a thread on `../pair/`; the created session carries a short public name that fits.
- The budget is derived from the actual socket directory, not from `defaultSessionNameBudget`.
- A failed launch leaves no orphan zellij server; existing orphans are reaped or documented as
  manual cleanup.
- The registration timeout's message distinguishes "pair never started" from "pair started and did
  not register" — the current text cannot, which is what made this cost hours.

## Plan

- [ ] Find where the short `couch-<N>` public name is allocated and why the new-thread path misses
      it.
- [ ] Call the probe before `LaunchSession` on the couch path; refuse with the diagnostic triple.
- [ ] Derive the budget from the measured socket dir.
- [ ] Reap orphaned unnamed servers on failed launch.
- [ ] Improve the registration-timeout message to name the session it waited for.
- [ ] Verify: start a pair thread from couch on this machine.

## Log

### 2026-09-08

Operator could not start a pair thread in couch. Diagnosed by composing the name by hand and asking
zellij directly, which is the check the code already knows how to make and does not make here.

Found while chasing an unrelated incident (`#214`), and the two are independent: `#214` is a race
that made one thread unresumable, this is why a *fresh* thread cannot start at all. The archive that
cleared `#214`'s thread is what surfaced this, because it forced a new tag.

Worth recording as method: the operator's error named a timeout, and the cause was a string length.
Nothing in the message pointed at zellij, the session name, or a budget — the six orphan
random-named servers were the only visible clue, and they only became meaningful after noticing they
had no children. A probe the codebase already owns would have said it in one line.
