---
id: 000272
status: open
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# couch loses live agents: liveness is proved from the launcher pid the agent outlives

## Problem

Split out of `pair#265`'s diagnosis, third member of the same reconciliation
family as `pair#271`.

Measured on the operator's store, 2026-09-16: **all 11 records carrying
`recorded: live` have a dead pid.** Every one classifies
`unusable: stale — helper ownership unresolved`. But three of them have an agent
that is *still running right now*:

| couch row | recorded pid | actually alive |
|---|---|---|
| `pair·6c0d459a` (`couch-95293a9b6c0d459a`) | 81935 — dead | `pair wrap` 82147, `muse-bin` 82165 |
| `pair·b14d495a` (`couch-20b78027b14d495a`) | 90046 — dead | `pair wrap` 90274, `muse-bin` 90294 |
| `pair·93df0741` (`couch-937e6ea093df0741`) | 35749 — dead | `pair wrap` 35793, `muse-bin` 35810 |

Three live `muse` conversations couch shows as dead and cannot reclaim.

The recorded pid is the **launcher** (`pair <agent> --layout3 …`). When couch
exits abnormally the launcher goes with it, but `pair wrap` and the agent are
reparented and keep running. `liveProofMatches`
(`couchcore/actionableinventory.go:342`) correlates
`ProcessIdentity{PID, Identity}` against that launcher pid, so the proof fails
and the thread is "stale" while the work is still there.

The identity is not wrong — it is exact, and deliberately so, "so a recycled PID
cannot pass". The problem is that it names a process whose lifetime is **shorter
than the thing it is proof of**. A liveness proof keyed to a process that dies
first will report every crash as a lost thread.

Two consequences compound:

- Every couch crash orphans its agents. `pair#265` is one such crash and there
  have been many — hence 11 stale rows across 8 repos.
- The orphans are unreclaimable through any gesture. The row offers `archive`
  (it is non-actionable, so `menuThreadActionable` is false), which retires the
  record and leaves the agent running forever with nothing pointing at it.

Related but distinct: `parley.nvim` has two live couch-tagged agents
(`couch-797c45e8e649a9bb` via `pair wrap` 84488, `couch-2583ed61c0ab6ebe` via
`pair wrap` 1130) with **no record in the store at all** — its scope holds a
single record, the healthy parked one. `ClassifyThread` has a
`ReasonUnrecordedChild` ("running but unrecorded") branch for exactly this, but
it is per-record, so a running agent with no record is invisible rather than
reported.

## Spec

- Prove thread liveness from something whose lifetime matches the thread's, not
  from the launcher. `pair wrap` is the obvious candidate — it is the process
  that owns the scrollback log and outlives the launcher — but establish this by
  measurement rather than assumption: enumerate what survives a couch kill and
  what does not, for each supported agent.
- Keep the pid-reuse guard. Whatever process is chosen, the proof stays an exact
  `ProcessIdentity{PID, Identity}` match (ARCH-DRY: reuse the existing
  correlation, do not add a second liveness notion).
- Give an orphaned-but-live thread a reclaim path. Today `archive` is the only
  gesture and it abandons a running agent. Decide whether couch re-adopts it,
  offers to stop it, or reports it for manual attention — and make `archive`
  refuse (or warn) while an agent is provably alive, so the destructive default
  cannot be taken by accident.
- Surface a running agent with no record at all. `ReasonUnrecordedChild` exists
  but is unreachable when there is no record to classify; the inventory needs a
  scan that can report a process the store does not know about.
- Bound the residue with `pair#271` (ARCH-FUNERAL): these two issues generate the
  same garbage from two directions and should agree on what collects it.

## Done when

- A thread whose launcher died but whose agent is running classifies as
  reachable, not `stale`, and the operator can get back into it.
- The liveness proof is keyed to a process measured to outlive the launcher, and
  the measurement is recorded.
- `archive` cannot silently abandon a running agent.
- A running couch-tagged agent with no record is visible somewhere.
- Sequence tests cover: couch killed with the agent surviving; couch killed with
  the agent also dying; a recycled pid at the proof process; and a record
  removed while its agent runs.

## Plan

- [ ]

## Log

### 2026-09-16

- Split out of `pair#265`. Evidence above measured from the live store and
  process table on this date; all pids verified with `ps`. Sibling issues:
  `pair#271` (park wedged in ThreadBusy), `pair#273` (blank viewport).

## Revisions

### 2026-09-16 — observed live, and it compounds with pair#265 into a burn loop

Reason: the operator burned two healthy threads in one sitting. This issue is
not just "couch forgets agents" — combined with `pair#265` it **destroys working
state on every crash**, which raises its priority above the cosmetic reading in
the original Problem.

The loop, each step verified in the store:

1. couch **resumes** a healthy `verified_park` thread. That clears
   `verified_park` and records a live incarnation carrying the **launcher** pid.
2. `pair#265` exits couch — `ctrl+space`, `ctrl+return`, or a focus change with
   the panel up.
3. couch dies without detaching, so the record keeps `state: live` with a pid
   that is now dead.
4. This issue: liveness is proved from that launcher pid, so the row classifies
   `stale — helper ownership unresolved`.
5. The thread is now unreachable, and its agent — if it survived — is orphaned.

A healthy parked thread is converted into an unusable row by one crash. Measured
on `tools·b3458e3244d72aeb`, the operator's own tools thread:

```
record:  inc=[(50192, 'live')]  verified_park=False  last_active=2026-09-15T20:58:35
pid 50192 (launcher):  DEAD
pair wrap 50350:       ALIVE — codex --sandbox danger-full-access,
                       scrollback-couch-b3458e3244d72aeb-codex.raw
```

**That agent is still running right now.** The thread is not lost; couch simply
cannot see it. Which is the argument for this issue: with the liveness proof
keyed to a process that actually outlives the launcher, step 4 does not happen
and steps 1–3 become survivable.

Delta to `## Spec`: add that the fix must be able to **re-adopt** an existing
orphan, not merely classify future ones correctly — there are already several on
this machine, and a fix that only prevents new ones leaves the operator's
current work stranded.

### 2026-09-17 — the same witness fails in the OTHER direction

**Reason:** a live incident showed the launcher pid producing a *phantom* live
thread, which is this issue's defect mirrored. One fix should cover both.

Observed on `astro` after its zellij server panicked at startup (`pair#273`):

```
$ cd ~/workspace/astro && couch --list
astro                  /Users/xianxu/workspace/astro
  address: fcff31946c0ac9d2/couch-48340fd828040287
  recorded: live     pid 63065
  live
```

pid 63065 is `pair resume couch-48340fd828040287 --layout3` — the launcher,
alive. `📁astro-couch-4` does not exist in `zellij list-sessions` at all. The
second line is the **classified** state, which `couchcmd/run.go:753` documents as
"from the same rule the switcher uses" — so this is not a `--list`-only display
bug, and it survives `pair#256`'s M1–M3 (measured with a binary built after M3
landed at 16:54).

The symmetry:

| | witness | reality | couch's answer |
|---|---|---|---|
| this issue | launcher pid dead | agent alive | thread lost (`stale`) |
| astro | launcher pid alive | session dead | thread phantom (`live`) |

Both are the same category error: **the launcher pid is evidence about the
launcher.** It is neither necessary nor sufficient for the session, and it can
fail in either direction because launcher and session have independent lifetimes
(the zellij server is PPID 1 at birth — `pair#275`'s measured basis).

**Delta to `## Spec`:** the liveness proof must be keyed to the session, so it
(a) survives a dead launcher — already this issue's case — and (b) does not
*assert* liveness from a living launcher whose session is gone. A fix that only
re-adopts orphans, without removing the launcher pid as a positive witness,
leaves the phantom half standing.

**Consequence worth stating in `## Done when`:** a phantom-live thread refuses
every lifecycle gesture that guards on liveness, which is how it becomes
unarchivable. That downstream requirement is carried in `pair#275`.
