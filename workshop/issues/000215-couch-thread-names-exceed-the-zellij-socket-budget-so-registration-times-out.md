---
id: 000215
status: codecomplete
deps: []
github_issue:
created: 2026-09-08
updated: 2026-09-09
estimate_hours:
started: 2026-09-08T20:56:16-07:00
actual_hours: 4.16
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

- [x] Reject over-long candidates arithmetically against a once-measured socket budget; keep the
      zellij probe as the oracle for the *budget*, not per candidate.
- [x] ~~Start the suffix walk from the highest known assigned suffix.~~ **WON'T DO** — the
      premise died with the fix. This was on the list because each iteration cost a zellij
      subprocess. It is now pure in-memory work: **66µs at 25 owned suffixes, 86µs at 60,
      144µs at 99**, with probes flat at 8. Tracking a highest-suffix-per-base-name would
      add state to save ~100 microseconds, and would change behaviour (a freed suffix would
      never be reclaimed). Reopen only if the 100-suffix ceiling is approached.
- [x] Budget the deadline against the work and state the envelope (15s, named constant with the
      measurement and the bounded-work condition written down). Assignment stays inside the
      window because it is now O(1); the constant says so, and says that raising it is the wrong
      fix if anything unbounded moves back in.
- [x] Reap orphaned unnamed servers on failed launch — **already fixed by #199**, verified
      here rather than assumed: both leaking probes are now `func main() { os.Exit(run()) }`
      with `defer session.Close()`, and `zellijprobe.Session.Close` runs
      `zellij delete-session --force`. `TestNoProbeExitsPastItsOwnCleanup` keeps it that
      way. The six live orphans predate that fix; they are UNNAMED, and production pair
      always names sessions `📁…`, so an unnamed server is probe debris by construction.
      Removing the six is an operator action, not a code change — see below.
- [x] Improve the registration-timeout message.
- [x] Verify with the probe-count test. Operator started threads from couch repeatedly and both
      cold start and relaunch now succeed; under sustained *load* is still unverified.

## Log


- 2026-09-09: closed — Unblock achieved and operator-confirmed: couch cold-starts a pair thread and relaunches a parked one, repeatedly, where before it failed with "context deadline exceeded". Round 3 fixed BR-18 at the class, and the fix is smaller than what it replaced: a session name is a socket filename so acceptance is MONOTONE IN LENGTH, and a two-sided bracket learns from an acceptance as readily as from a refusal. Previously the arithmetic path started only after a rejection, so on a host whose socket dir is short enough that the longest candidate fits (Linux ~/.cache/zellij vs macOS temp) nothing was refused and it paid one probe per suffix -- O(threads) restored on the machines with the most headroom, while code, atlas and Done-when claimed O(1) unconditionally. Now: tight budget 8->4 probes, generous budget O(N)->3, both INVARIANT across 25 and 60 owned suffixes; resume still exactly 1 probe. Pinned by TestProbeCountIsInvariantInBothBudgetRegimes, mutation-verified (27 at 25 owned, 62 at 60 without it). discoverSessionNameBudget and defaultSessionNameBudget deleted -- the acceptor needs no constant, and a production symbol only tests reach is its own smell (#192); measureAcceptedLimit replaces it for refusal MESSAGES through the caller own acceptor, so one encoding of "does this fit" and the answers cannot contradict, which closes the 3rd finding in family fallback-guess-used-as-oracle (the old fallback could call a name fitting that the probe had just refused, printing an EMPTY message on exactly the machine BR-1 is about). Two tests replaced rather than quietly deleted: BR-1 probe-count proxy became a soundness test asserting the acceptor agrees with a raw probe on every candidate across five budgets, which subsumes BR-1 and pins the monotonicity the bracket rests on; the live conformance test now drives the production acceptor so what is pinned is what runs. atlas/session-identity.md rewritten to describe the bracket and both regimes. make test, test-smoke and test-couch-zellij-live all exit 0; binaries rebuilt.; review verdict: FIX-THEN-SHIP
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

### 2026-09-08 — implemented (branch `000215-…`, 4 commits)

**Frontmatter first.** `4663fb70` rewrote this file's body and clobbered its head,
taking the YAML frontmatter and H1 with it, so every `sdlc` verb refused the issue
with "no YAML frontmatter" — it could not be claimed, shown or closed. Restored,
with the H1 stating the corrected root cause since the filename's slug still
asserts the refuted one.

**The cost (52 → 8 probes).** `sessionNameAcceptor` (createflow.go) judges a
candidate's length arithmetically against a budget the probe measures once.
`TestAssignSessionNameCostsABoundedNumberOfProbes` asserts O(1) as **invariance
across index sizes**, not as a bound — a bound alone still passes something that
grows slowly, and this index only ever grows. Against the old code it reports 52
at 25 owned suffixes and 122 at 60.

**A regression I shipped and the operator caught.** The first cut discovered the
budget *eagerly*, which made the cold path O(1) and the **resume** path 7× worse:
the ledger short-circuit asks about ONE name — every resume takes it — and it was
paying for a binary search it had no use for. 1 probe → 7, ~45ms → ~315ms.
Operator reported "relaunch worked, but it takes a lot longer than before" and the
measurement confirmed it was mine. Discovery is lazy again — probe directly until
a probe says NO, then measure once and go arithmetic — which is cheaper than both
the old code and my first cut:

| path | before #215 | first cut | now |
|---|---|---|---|
| resume (ledger hit) | 1 | 7 | **1** |
| cold (ladder walk, 25 owned) | 52 | 7 | **8** |

`TestResumingAKnownThreadCostsOneProbe` pins the warm path, because nothing did:
the bounded-cold-path test passed happily while the warm path got 7× worse.
"Bounded" and "cheap" are different claims, and a fix for either can quietly pay
for it out of the other.

**The deadline.** 5s → a named `pairRegistrationTimeout = 15s` carrying the 8.85s
measurement and the condition that makes it valid.

**The message.** `diagnoseRegistrationFailure` reports liveness, and liveness is
the ONLY discriminator: `PairSession` returns a missing index entry as an *error*
rather than `Present=false`, and that is the "Pair never recorded a name" case —
the most diagnostic one there is. Branching on the error first swallowed it; the
test caught exactly that ordering.

**Startup latency is still unexplained, and two suspects are dead.** #215 raised
the deadline against a measured 8.85s without measuring where those 8.85s go:

| suspect | measured | verdict |
|---|---|---|
| nvim + pair's `init.lua` | 117 ms (`nvim --startuptime`) | not it |
| `zellij list-sessions` | 43 ms at 26 sessions | not it alone |
| one `ProbeSessionName` | ~45 ms, creates no session | bounded now |

`probes/zellijcalls/` traces every zellij subprocess via a PATH shim rather than
instrumented call sites — pair spawns zellij from eight places, so instrumenting
"the seam" means instrumenting eight and missing the ninth. Not yet run against a
real couch thread start.

**Note for whoever closes this.** couch's shell function rebuilds `bin/couch` on
every launch but NOT `bin/pair`, and couch execs `pair` from PATH
(`launch_existing.go:52`). The deadline fix lives in couch and arrives free; the
probe fix lives in the launcher and needs `make build`. Half of what looks like
couch's behaviour is in a binary couch does not build.

**Still open:** the suffix walk still restarts at 1 (cheap now, but still O(N)
in-memory); orphaned unnamed servers are not reaped — six from #199's probes plus
a `zellij action rename-pane` stuck 8h are live on the operator's machine and
awaiting an OK to kill; verification under sustained load.

### 2026-09-08 — the last two plan items, resolved by measurement rather than code

Neither remaining item needed implementing, and both are worth recording because
the reasoning is the deliverable:

**The suffix walk (won't do).** Its entire justification was that each iteration
cost a subprocess. With judging made arithmetic, the walk is 66-144µs across
25-99 owned suffixes and the probe count does not move (8). Optimising it would
add state and change reclamation behaviour to buy microseconds. A plan item can
be made obsolete by an earlier item in the same plan; ticking it by writing code
anyway would be busywork with a behaviour change attached.

**Orphan reaping (already done, by #199).** Verified at the source rather than
assumed. What remains is six live servers on the operator's machine plus a
`zellij action rename-pane` stuck 8h14m — historical debris awaiting an operator
OK to kill, not a code change. `ps` shows all six at ppid=1 and 0.0% CPU, so they
cost ~430MB RSS and inflate `list-sessions` to 26 entries; they are NOT the
startup latency (one `list-sessions` is 43ms).

**Scope, corrected 2026-09-08 (operator).** *"#215 is about unblock so that I
can continue to use (and thus dogfood) couch."* That is achieved: cold start and
relaunch both work, repeatedly, on the operator's machine. The unexplained 8.85s
startup is **split out as `#218`**, which carries the measurements taken here —
four suspects already dead — and the `probes/zellijcalls/` tracer built here and
not yet run. Latency is not a reason to hold this issue open.

Verification under *sustained* load remains unverified, and is noted rather than
claimed: the operator's repeated successful starts were on an ordinarily-loaded
machine, not a deliberately loaded one. The O(1) probe count is what makes the
load case survivable, and that is asserted by test rather than by a live run.
