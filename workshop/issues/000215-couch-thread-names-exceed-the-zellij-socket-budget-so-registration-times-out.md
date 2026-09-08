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

**Assigning the session name costs one zellij subprocess per candidate, and the candidate count
grows with every couch thread the repo has ever had.** Measured by running the real
`AssignSessionName` against the real index:

```
index entries: 91
RESULT name="📁pair-couch-26" err=<nil>
PROBES=52  probe-time=1.087s  total=1.088s
```

It **succeeds** — and spends **52 zellij invocations** doing it, against a registration budget of
**5 seconds** (`couchcore/launch_existing.go:111`).

### Why 52, and why it grows

`AssignSessionName` walks `for suffix := 1; suffix <= 100`, and for each suffix probes
`BuildSessionNameCandidates` in order. For this repo:

| suffix | cand[0] | cand[1] |
|---|---|---|
| 1 | `📁pair-couch-675d94857e70ca61` (31 B) | `📁pair-couch` (14 B) |
| 26 | `📁pair-couch-675d94857e70ca61-26` (34 B) | `📁pair-couch-26` (17 B) |

**Two probes per suffix**: the long candidate is rejected by zellij (over the socket budget), the
short one is accepted and then found owned, so the ladder breaks and bumps the suffix. The repo owns
suffixes 1–25, so reaching the free 26 costs `26 × 2 = 52` probes.

**That count is the number of couch threads this repo has ever had.** At thread 1 it was 2 probes
(~35 ms). It is now 52. Nothing resets it — the index rows are permanent, and archiving a thread
does not release its suffix.

### Why it broke now rather than earlier

Two independent trends crossing one fixed deadline:

1. **The cost grows linearly with threads created.** 2 probes each, forever.
2. **Each probe got ~8× more expensive.** `#203` measured a `zellij action` round-trip going from
   **17.6 ms calm to 145 ms median / 467 ms max** under concurrent agent load — which is the load
   this operator normally runs.

| condition | 52 probes | vs 5 s budget |
|---|---|---|
| calm (measured) | **1.09 s** | fits, using 22% of it before pair does anything |
| loaded, median | ~7.5 s | **over** |
| loaded, tail | ~24 s | **far over** |

So it did not break; it has been creeping toward the line for 26 threads, and the load regression
pushed it across. On a quiet machine it still works, which is exactly why it looks intermittent.

**Corrected 2026-09-08.** This issue was first filed claiming the session name was simply too long
for the socket path. That is wrong: the ladder handles length correctly, shortening the 31-byte
name to `📁pair-couch-N` at 16–17 bytes. The length only matters because **rejecting the long
candidate costs a subprocess round-trip every time**, which is what makes the walk expensive. The
original diagnosis verified a string composed by hand rather than the one the code produces —
`BuildSessionNameCandidates` and `AssignSessionName` are both exported and answer this directly.

### Two aggravating properties

**The over-long candidate is probed at all.** Its length is knowable without asking zellij — the
socket directory is a fixed local fact (`/var/folders/…/contract_version_1/`, 78 chars against a
104-byte `sun_path`, leaving ~25, of which `📁` costs 4). Probing it is a subprocess spent to
rediscover arithmetic, and it is **half of all probes**.

**The walk restarts at suffix 1 every time.** Nothing remembers that 1–25 are taken, so each new
thread re-walks the whole prefix.

### Debris

Six orphan zellij servers with random names (`hopeful-tiger`, `judicious-lemur`, `jumping-pepper`,
`zippy-galaxy`, …) created 09:54–11:07 across the failed attempts, each with **no children and
`ppid=1`** and an empty screen. They accumulate per failed attempt and nothing reaps them.

### Diagnosis cost

The operator's error named a timeout. The cause was a subprocess count. Nothing in
`await Pair registration … context deadline exceeded` names zellij, the session name, the probe
count, or which of "pair never started" and "pair started and never registered" occurred.

## Spec

**Names that cannot fit must be refused before launch, and thread names must be short enough to fit.**

Two halves; the second is the fix and the first is the guard that should have caught it.

**1. Stop spending a subprocess to learn a length.** Measure the socket budget once — the directory
is a local fact — and reject over-long candidates arithmetically. That halves the probes
immediately and removes the only reason the 16-hex candidate costs anything.

**2. Do not re-walk the suffix prefix.** Remember the highest suffix assigned per (scope, family) so
a new thread starts near the free end instead of re-probing 1..N. The index already holds every
prior name; the information is present and unused.

**3. Do not put an unbounded, load-sensitive loop inside a fixed 5-second deadline.** Either assign
the name **before** starting the registration clock, or budget the clock against the work — a
fixed constant in front of an O(threads) loop whose per-iteration cost varies 8× with machine load
is a deadline that will keep failing intermittently as the fleet grows (`ARCH-CONSTRAINTS`: the
envelope is unstated, and this is the second time it has bitten — see `#203`).

**4. Reap the debris.** A failed launch should not leave an unnamed zellij server with `ppid=1`.
Six accumulated across one session of retries.

**5. Say what failed.** The registration timeout must distinguish "pair never started" from "pair
started and did not register", and name the session it waited for.

## Done when

- Assigning a name for a repo at suffix N costs **O(1) zellij probes, not O(N)** — asserted by a
  test that counts probes at a seeded index of 25 owned suffixes and requires a small constant.
  The current count is 52 and the test should fail against today's code.
- couch starts a thread on `../pair/` on a loaded machine, not only a quiet one.
- Over-long candidates are rejected without a subprocess.
- A failed launch leaves no orphan zellij server; the six existing ones are reaped or documented as
  manual cleanup.
- The registration timeout's message distinguishes "pair never started" from "pair started and did
  not register", and names the session — the current text does neither, which is what made this
  cost hours.

## Plan

- [ ] Reject over-long candidates arithmetically against a once-measured socket budget; keep the
      zellij probe as the oracle for the *budget*, not per candidate.
- [ ] Start the suffix walk from the highest known assigned suffix.
- [ ] Move name assignment outside the registration deadline, or budget the deadline against the
      work; state the envelope.
- [ ] Reap orphaned unnamed servers on failed launch.
- [ ] Improve the registration-timeout message.
- [ ] Verify with the probe-count test, then start a pair thread from couch under load.

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

### 2026-09-08 — root cause corrected

First filed against the wrong cause: "the session name is too long". Measuring the real code
refuted it — `AssignSessionName` **succeeds**, returning `📁pair-couch-26`, and the ladder shortens
correctly. The defect is its **cost**: 52 zellij subprocesses against a 5-second deadline.

The error I made is worth recording, because it is the same one twice in one investigation: I
composed the session name by hand, verified *that string* was rejected, and filed. Both
`BuildSessionNameCandidates` and `AssignSessionName` are exported and answer the question in a
six-line test. Verifying a hypothesis I authored, rather than the artifact the code produces, is
what put a wrong root cause in the tracker — the same shape as the earlier `pair resume … -- --resume`
suggestion, which was reasoned about instead of read.
