---
id: '000215'
status: done
started: 2026-09-08T20:56:16-07:00
created: 2026-09-08
updated: 2026-09-09
actual_hours: 4.16
---

# couch spends 52 zellij subprocesses assigning a name inside a 5s registration deadline

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
