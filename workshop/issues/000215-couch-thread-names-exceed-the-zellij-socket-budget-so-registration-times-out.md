---
id: 000215
status: working
deps: []
github_issue:
created: 2026-09-08
updated: 2026-09-08
estimate_hours:
started: 2026-09-08T20:56:16-07:00
---

# couch spends 52 zellij subprocesses assigning a name inside a 5s registration deadline

<!-- The filename's slug preserves the ORIGINAL, refuted root cause ("names
exceed the socket budget"). The name assignment SUCCEEDS; its COST is the defect.
See "root cause corrected" in the Log. Kept as-is because the id, not the slug,
is the address. -->

## Problem

**The registration deadline is too small for the work it fronts. Measured:**

```
REGISTRATION LATENCY: 8.85s     (pair launches, renders, writes thread-claim state:"established")
couch budget:         5.00s     (couchcore/launch_existing.go:111, couch.go:114)
```

Taken by running couch's own invocation — `pair resume <tag> --layout3` — under a pty on a **calm**
machine (load 3.01) and polling for the claim file. Pair does not fail: it launches correctly,
renders its panes, and registers correctly. It is simply **not finished in five seconds**, and couch
reports `context deadline exceeded`.

Five seconds was a bare constant with no stated basis, fronting a full layout3 workbench bring-up —
zellij, three panes, nvim, and a coding agent. It was always marginal; ordinary drift pushed startup
past it.

**Operator decision 2026-09-08: raise the budget to 15 seconds.** That is ~1.7× the measured calm
latency, leaving headroom for the load case below without inventing a large number nothing measured.

### The second consumer of the same budget, which will outgrow any constant

Name assignment spends **one zellij subprocess per candidate**, and the candidate count grows with
every couch thread the repo has ever had. Measured against the real index:

```
index entries: 91
RESULT name="📁pair-couch-26" err=<nil>
PROBES=52  probe-time=1.087s  total=1.088s
```

**Two probes per suffix** — the 16-hex candidate rejected by zellij for length, then the short one
found owned, so the ladder bumps the suffix. The repo owns suffixes 1–25, so reaching the free 26
costs 52 probes. That count *is* the number of couch threads this repo has ever had; nothing
releases a suffix on archive.

Under load it is far worse: `#203` measured a `zellij action` round-trip going from **17.6 ms calm to
145 ms median / 467 ms max** with concurrent agents.

| condition | 52 probes | against 5 s | against 15 s |
|---|---|---|---|
| calm (measured) | 1.09 s | fits | fits |
| loaded, median | ~7.5 s | over | fits, barely |
| loaded, tail | ~24 s | far over | **still over** |

So raising to 15 s unblocks today and does **not** close this: an O(threads) loop whose per-iteration
cost varies 8× with load will re-cross any fixed constant. Both halves need fixing.

**Corrected 2026-09-08.** This issue was first filed claiming the session name was too long for the
socket path. That is wrong — the ladder shortens correctly and `AssignSessionName` **succeeds**,
returning `📁pair-couch-26`. The length matters only because *rejecting* the long candidate costs a
subprocess every time. The original filing verified a string composed by hand rather than the
artifact the code produces; both `BuildSessionNameCandidates` and `AssignSessionName` are exported
and answer it in a six-line test.

### Aggravating properties

**The over-long candidate is probed at all.** Its length is knowable without asking zellij — the
socket directory is a local fact (`/var/folders/…/contract_version_1/`, 78 chars against a 104-byte
`sun_path`, leaving ~25, of which `📁` costs 4). Half of all probes are spent rediscovering
arithmetic.

**The walk restarts at suffix 1 every time.** Nothing remembers that 1–25 are taken.

**The failure names nothing.** `await Pair registration … context deadline exceeded` says neither
what it waited for, nor whether pair started at all. Everything needed was one pty launch away.

### Incidental, found while investigating

Six zellij sessions leaked from `#199`'s own probes — `probes/zellijscrollregion/floodprobe.sh` ×2
and `probes/cursorsaveslots/probe.sh` ×4 — created 09:54–11:07 and still running hours later. They
create bare (unnamed) sessions and nothing reaps them. `probes/couchnestedrows` already does this
correctly with a `defer delete-session`; these two do not.

## Spec

**Names that cannot fit must be refused before launch, and thread names must be short enough to fit.**

Two halves; the second is the fix and the first is the guard that should have caught it.

**1. Stop spending a subprocess to learn a length.** Measure the socket budget once — the directory
is a local fact — and reject over-long candidates arithmetically. That halves the probes
immediately and removes the only reason the 16-hex candidate costs anything.

**2. Do not re-walk the suffix prefix.** Remember the highest suffix assigned per (scope, family) so
a new thread starts near the free end instead of re-probing 1..N. The index already holds every
prior name; the information is present and unused.

**3. Raise the budget to 15 seconds** (operator decision), touching
`couchcore/launch_existing.go:111` and `couchcore/couch.go:114`. Both files were **clean** on
2026-09-08 while `#199` held other files dirty, so this half lands without contention. Requires a
couch **rebuild and restart** — the running couch had been up 1d15h and will not pick it up
otherwise.

**4. Do not leave an unbounded, load-sensitive loop inside a fixed deadline.** Either assign
the name **before** starting the registration clock, or budget the clock against the work — a
fixed constant in front of an O(threads) loop whose per-iteration cost varies 8× with machine load
is a deadline that will keep failing intermittently as the fleet grows (`ARCH-CONSTRAINTS`: the
envelope is unstated, and this is the second time it has bitten — see `#203`).

**5. Reap the debris.** A failed launch should not leave an unnamed zellij server with `ppid=1`.
Six accumulated across one session of retries.

**6. Say what failed.** The registration timeout must distinguish "pair never started" from "pair
started and did not register", and name the session it waited for.

## Done when

- couch starts a thread on `../pair/` again — the 15 s budget covers the measured 8.85 s.
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
