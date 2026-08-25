---
id: 000145
status: done
deps: []
github_issue:
created: 2026-08-20
updated: 2026-08-25
estimate_hours:
started: 2026-08-21T12:40:25-07:00
actual_hours: 8.51
---

# couch: spawn and registry

Project: `workshop/projects/couch.md` — architecture, non-goals and execution
order live there; this issue is task 1.

## Problem

Nothing knows what agent sessions exist. Pair drives one session; the outer
shell is cmux or ghostty tabs, which have no notion of what an agent session
*is*, so there is no way to ask what threads are open, bring a dormant one back,
or stop two sessions from silently sharing one working tree.

Before any switching UI is worth building, couch needs the thing underneath it:
a registry of named actors and a deterministic way to bring one up.

## Spec

**Actors are goroutines inside couch**; the agent harness (claude/codex, or a
directly-driven loop) is a child process the actor spawns, feeds, and lets die.
One OS process stays up, mailboxes are structs, and the name registry is
authoritative because it lives in one address space.

**Spawn takes a peer repo, not an issue.** What the agent works on is decided
inside the session — an issue crystallizes mid-thread, it is not a precondition.
Start-args stay **structured, not a bare string**: repo, tree (checkout or
worktree path), agent stack, prompt/skill configuration, and optionally an issue
ref as metadata. couch must know how to bring up a *properly equipped* agent from
that record.

**The durable key is the working tree** (repo + checkout/worktree path). It
survives restarts, exists whether or not there is an issue, and is what two
agents actually collide over. The system-level id need not be legible.

**Actor ids are generated and opaque** — `couch-ah8d`. They identify an actor
*instance* and are not meant to be legible or typed.

**Human addressing is a mutable mapping above them** — short names the operator
assigns at runtime, plus a one-line description of what the agent is doing,
supplied by the agent and free to change mid-session. Nothing structural depends
on either: a label may be wrong, duplicated or stale without corrupting anything.
Resolution stays fuzzy-in/exact-out, so a duplicate label asks which.

**Names and descriptions attach to the tree, not to the actor id.** The id is
per-instance and dies with it; the tree is the durable key. If naming hung off
the id, every revival would re-impose exactly the memory load the naming layer
exists to remove.

**Session selection in the root actor** simplifies away from pair's tag: load the
last active non-subagent thread in this tree, else start fresh. Clearing context
is `alt+shift+n`.

**Name registration IS the collision guard.** Refusing a second agent on a
working tree is `register(repo, tree)` failing because the tree is taken — not a
separate feature. The invariant is one agent per tree, which is more accurate
than one issue per repo: the issue was only ever a proxy for the tree. On
refusal, offer worktree-or-switch, which turns today's silent collision into a
decision at the moment the operator has context to answer it.

**Per-repo concurrency policy is recorded, not inferred per spawn** —
`in-place-serial` where the checkout is the installation (pair, ariadne, parley),
`worktree-parallel` otherwise, plus a third case for workspaces with heavy local
state where worktrees are expensive for unrelated reasons. Read from fleet
metadata (`ariadne#200`) once that exists; a local stub until then.

**Queryable state and callable operations from the first commit.** Every
operation carries structured args and every state is readable — the LLM-drivable
surface the project depends on.

**Descriptors come from the agent, with a cached fallback.** couch asks a live
agent for its one-line description and caches the last answer, because a cold
agent cannot answer and cold is exactly where forgetting lives. This is not the
published-status artifact returning: that failed because it described *state* and
went stale confidently. A stale *label* still finds the right tree, and then you
ask it. Labels tolerate staleness; state does not.

**Bounded mailbox with collapse-by-kind.** Ephemeral, not durable. Two channels —
`control` (stop, deadline) and `normal` — with a priority select. A send is a
non-blocking attempt that may fail; a full mailbox is a loud bug signal, not flow
control.

No switching in this issue: spawned sessions land in the current terminal, the
way they do today.

## Done when

- couch registers, lists and spawns actors against a peer repo, bringing up a
  properly equipped agent for at least the claude stack, with no issue required.
- An agent started on a tree that already holds one is refused by name
  registration, with worktree-or-switch offered.
- An operator-assigned short name and an agent-supplied description both resolve
  to the right actor, and both can change mid-session without breaking anything.
- A name assigned before an actor dies still resolves after it is revived, since
  naming hangs off the tree rather than the instance id.
- Per-repo concurrency policy is read from a recorded source, not inferred.
- Every operation is invocable with structured args and every state is
  queryable — audited over the operation set, not spot-checked.
- `pair-go` standalone is unaffected; couch failing does not block work.

## Plan

- [x] Registry keyed on the working tree; structured start-args; durable across
      couch runs.
- [x] Actor goroutine: bounded mailbox, priority drain (mutex-guarded queue,
      not two channels -- deviation recorded below).
- [x] Spawn: bring up a claude-stack child from start-args in the right tree.
- [x] Name registration on (repo, tree) as the collision guard; worktree-or-switch
      on refusal.
- [x] Runtime naming layer: operator short names + agent-supplied description,
      cached; neither load-bearing.
- [x] Per-repo concurrency policy source (recorded file, not a constant).
- [x] Queryable-state + callable-operation surface, with the operation-set audit.
- [x] Operator smoke: hosting a real `pair` child (settled the layering fork),
      the second-shell read path, `show` filtering to one tree, and the refusal
      against a live incumbent. The kbench-subdirectory case remains unrun by
      the operator, though it is covered by unit and live tests.

## Log

### 2026-08-21

Split out of the former root ticket when couch was promoted to a project. The
architecture narrative, v1 non-goals and cross-repo enablers moved to
`workshop/projects/couch.md`; this issue keeps only task 1.

Rekeyed from issue-as-identity to tree-as-identity the same day (see the project
Log scope event). Spawn now takes a repo; the issue is metadata, not the key.

### 2026-08-21 — session summary

Design settled through two rounds of fresh-eyes plan review, then implemented.
The plan (`workshop/plans/000145-couch-spawn-and-registry-plan.md`) was
respecified after the second review built a scratch module from its own specs
and ran them: it found a test that deadlocked and a test that passed with the
seam it named deleted, neither visible on inspection. Hand-written Go in a
markdown document cannot be trusted without executing it, and executing it is
implementing it -- so the plan now carries contracts, per-test "what bug does
this catch", and a **deletion check** per task.

Landed: `cmd/internal/couchcore` (path canonicalisation, worktree resolution,
registry, naming, policy, store, runner seam + stateful fake, proc seam,
composition root, spawn, operation surface) and `cmd/internal/couchcmd` +
`cmd/couch`. `couch` is in `GO_BINS` with its own recipe; `make build` produces
both binaries and `./bin/pair --help` still works.

Deletion checks paid immediately. On the very first function, removing
`filepath.Clean` left the suite green -- `filepath.Abs` already cleans, so the
line was dead on the success path. Every seam call since has its own failing
test: both `PathOps.Physical` calls in `Resolve`, `FakeGit`'s `dir` key, the
registry map copy, `SameTree`, the pre-fork guard, and the CLI dispatch audit.

Two bugs found by writing rather than by review. `NamingTable.Lookup` returned
`Worktree(key)` -- the folded key -- so on a case-insensitive volume it handed
back a lowercased path, losing the case that `ResolveRepoScope` hashes.
`Registry.All()` exposed folded map keys and invited the identical mistake; it
is now `Records()`.

Deviations from the Spec, recorded rather than silent: the mailbox is a
mutex-guarded queue rather than two channels with a priority select, because
the bounded/collapse policy must be applied at insertion and a bare buffered
channel cannot do that. `list`/`show` return per-tree summaries and keep a
named tree with no running agent visible (dimmed) -- a parked thread is exactly
what this project exists to stop losing.

Still open: the actor loop and mailbox (the last domain pieces, and what
`#147`'s transport sits on), live real-vs-fake conformance, and the operator
smoke that settles whether couch hosts `pair` whole or absorbs zellij's role.

## Side quests

Unbudgeted but shipped -- all surfaced by running `go test ./cmd/... -race`,
which was not previously part of the loop. Full tree is now green under `-race`
(38 packages).

- wrapcmd test race: buffer polled across goroutines, ~0.3h, `10e136a`
- launcher test race: fake's map written from two goroutines, ~0.3h, `10e136a`
- **scribecmd production bug**: pty use-after-close -- the SIGWINCH goroutine
  outlived `Run` and could `ioctl` a recycled descriptor -- plus two leaked
  goroutines per call, ~0.7h, `e528671`
- scrollbackcmd: emulator left unclosed so its drainer parks, since
  `vt.Emulator.closed` is an unsynchronised bool and the reverse order
  deadlocks (verified), ~0.5h, pending commit

### 2026-08-21 — layering fork settled

Operator ran `./bin/couch start ../pair`: **pair starts up correctly as a couch
child.** So couch hosts `pair` whole -- couch -> pair -> zellij -> claude+nvim,
three layers of terminal management -- and does **not** need to absorb zellij's
role. That was the open fork recorded against `#146`, and it is the cheap
answer: a zellij inside a couch-owned pty is just a child that redraws on
SIGWINCH.

Scope of the result: this milestone hands the child couch's own stdio, so what
is proven is that the whole stack comes up under a couch-spawned process. It
says nothing about attach/detach or multi-child routing, which need the pty
`#146` builds. Not yet exercised by the operator: the second-shell `couch list`
read path, the refusal offer, and the kbench-subdirectory case.

### 2026-08-21 — actor loop landed

`Message`/`Enqueue` (pure) and `Actor` (the goroutine). All 8 plan items ticked.

**Deviation from the Spec, with the reason.** The Spec suggested two channels
with a priority select; this is a mutex-guarded queue instead. The bounded and
collapse-by-kind policy has to be applied AT INSERTION, and a buffered channel
cannot collapse a duplicate already sitting in it. Putting `Enqueue` behind the
mutex keeps the whole decision pure and unit-testable without goroutines; the
channel version would have pushed that logic into the receive loop, where it is
much harder to test. Priority survives as a control-first drain.

Control messages are never dropped, even over capacity. Control carries stop and
deadline, and trading a real obligation for a bookkeeping number is the wrong way
round -- so an all-control overflow keeps everything and reports the violation
instead. `Send` returns false and `OnDropped` names what was lost, because the
Spec asks for a full mailbox to be a loud bug signal rather than flow control,
and a signature that cannot fail makes that impossible to honour.

`QueueLen` is a direct call behind the same mutex, not a message: Go shares
memory, so message passing here buys ordering and decoupling, not fidelity to
Erlang. Depth is also one of the advisor's two staleness signals -- it says
somebody is waiting on this agent, where git staleness says the thread has gone
cold.

Deletion checks: swapping `Enqueue` for a plain `append` turns the collapse and
drop tests red (so the pure policy is genuinely wired in, not orphaned beside
the loop), and taking the oldest instead of control-first turns the priority
test red.

One fixture note worth keeping: the priority test uses three *distinct* normal
kinds. Identical ones would collapse to one and the expected count would never
arrive -- a fixture that fights the policy it sits on deadlocks rather than
fails, which is exactly how an earlier draft of this test hung.

### 2026-08-21 — live conformance found a real bug in ExecRunner

Task 16 landed (`conformance_live_test.go`, gated on `PAIR_LIVE_COUCH=1`, no
build tag per the house pattern), and it earned its keep on the first run.

`ExecRunner.Alive()` was wrong for an exited-but-unreaped child. It called
`procutil.Alive`, which is `kill -0` -- and **`kill -0` succeeds for a zombie**.
So a child that had already died reported as running until somebody called
`Wait`. The fake reported it dead, correctly; the real implementation did not,
and the two diverged. That is exactly the drift a live check exists to detect,
and no unit test against the fake could have found it.

Fixed by reaping in a background goroutine at `Start`: liveness is now a closed
channel rather than a syscall, which is also the shape `FakeRunner` already
models. The fatal-signal scenario went from a 5s timeout to 0.01s.

Two things worth keeping from the diagnosis. First, my initial reading was that
the *scenario* was wrong -- that signalling `sh` was not reaching `sleep` -- and
adding `exec` changed nothing, which is what pointed at `Alive` rather than at
the shell. Second, `Couch.IsLive` shares the same `kill -0` hazard for
out-of-process reads; the window is now documented in place, and it needs a
couch that spawned a child, is not waiting on it, and is still running, which
`couch start`'s blocking Wait rules out today.

Conformance also required teaching the fake a signal *disposition*
(`SetDiesOn`), because the fake cannot know whether a child catches a signal.
The default remains "nothing kills", and the two dispositions are now each
checked against a real process rather than assumed.

### 2026-08-21 — close notes on estimate and actuals

**`estimate_hours` is empty, and deliberately stays empty.** `sdlc change-code`
was never run: this went start-plan → implement, skipping the gate that owns the
plan-quality check and the estimate. Deriving an estimate now would be hindsight
with the answer already known, and a guessed value pollutes velocity calibration
far worse than a missing one -- which is the exact hazard the gate exists to
prevent. So this issue contributes an actual with no estimate to compare it
against, and the calibration ledger should treat it as unpaired rather than as a
1.0x hit. The plan itself did get two rounds of fresh-eyes review, which is
stronger than the plan-quality judge, but it is not the same gate and does not
substitute for the estimate.

**What the 3.96h covers.** The window is the claim commit to HEAD, i.e. today's
implementation, two plan-review rounds, and the four side-quest race fixes. It
does **not** cover the design: the pensive, the project file, the identity
rekey and the UI model were all worked out on 2026-08-20 and 2026-08-21 *before*
the claim at 12:40, so roughly two sessions of design sit outside the measured
window. Claiming early is supposed to prevent exactly that and I claimed late.

Attribution warnings on the measurement: ~44m attributed to #145 and ~15m to
#146 by mention-fallback, plus ~58m unattributed, all in the 15:35→16:34 band.
The number is measured rather than typed, so it is adopted as-is, but it is
softer than a clean commit-boundary attribution would be.

### 2026-08-21 — close review REWORK, addressed

Boundary review returned REWORK with 12 blocking findings. It reproduced them
against the built binary, injected an undeclared `couch nuke` branch to prove
the operation audit vacuous, and restored the pre-fix `ExecRunner` to prove the
liveness fix undefended. All 12 are addressed, each with a regression test whose
failure was verified by reverting the fix.

The three Criticals were real and related. **The guard never consulted
liveness**, so since `couch start` blocks and nothing unregisters on exit, the
ordinary end of a session left a dead record refusing its own tree forever;
`Spawn` now prunes records whose identity no longer matches before the guard
runs. **`couch stop` never signalled anything** -- it called `Forget`, which
freed the tree while the agent kept running, opening the exact hazard the
registry closes; `Stop` now signals SIGTERM through a new `ProcOps.Signal`,
re-checking the identity token first because SIGTERM to a recycled PID is not
recoverable, and only then forgets. **`show <ref>` ignored its argument** --
`Summarize` took a filter and then folded in every record, so it printed what
`list` printed; the old test passed only because its fixture had one tree.

The finding that matters most for how the rest shipped is BR-7: `couchcmd`
constructed production seams inline, so no test could reach start, stop or the
refusal rendering. That is why three Criticals got through. `Runtime` now
supplies the domain, and driving the CLI over fakes immediately surfaced a
second design fact -- `couch start` blocks on `Handle.Wait`, so a fake child
that never exits hangs a test rather than failing it. `FakeRunner.AutoExit`
models completion.

BR-6 deserves naming. The audit compared `Dispatch()` against `OperationNames()`,
both derived from `Operations()` -- two views of one source, structurally unable
to fail. I had run a deletion check on it and got red, but only by hand-injecting
an entry that real code cannot produce. Having written a lesson this session
titled "a test that survives deleting the seam it names tests nothing", I then
shipped exactly that and called it structural. It now drives the CLI itself and
catches the reviewer's `couch nuke` attack; its comment states plainly what it
does and does not guarantee.

Also fixed: `Store.Load` fabricated `same_tree=true` on replay and the next Save
persisted it, so no reader could tell who used the escape hatch (`Registry.Insert`
now replays without the guard); an unreadable registry read as a first run and
was then destroyed (`fs.ErrNotExist` is now distinguished from real IO errors);
the zombie fix was pinned only by an opt-in gate (a default-suite test now polls
`Alive()` after `sh -c 'exit 0'` without calling `Wait`); the description cache
had no source (`publish-description` plus `COUCH_TREE`/`COUCH_STORE_DIR` on the
child give the agent a way to write its own line); the atlas described an actor
loop nothing instantiates; the README still claimed a single binary.

And BR-12, which is the one to remember: the operator smoke was ticked after
step 1 of 5. **Two of the four unrun steps are exactly where BR-1 and BR-3
live** -- a second-shell `couch list` and a repeat `couch start` would have
surfaced both. The checkbox is now unticked and says which steps ran.

### 2026-08-21 — smoke testing caught a fail-open bug in the BR-1 fix

The operator ran the smoke steps and the second-shell read path worked: `couch
list` showed the live actor with its pid, and `couch show ../pair` returned one
tree rather than everything (BR-3 confirmed against the real binary).

Then the refusal step **did not refuse**. It admitted a second agent onto a tree
that already had a live one, and in doing so destroyed the running session's
registration.

Root cause is in the BR-1 fix itself. `PruneDead` dropped every record where
`IsLive` was false -- but `IsLive` was two-valued, and both probes underneath it
collapse "the process is gone" with "I could not check": `procutil.Alive` forks
`kill -0`, so a restricted fork reads as dead, and `procutil.Identity` returns
`""` on any error. So a probe that merely failed pruned a live actor and then
let a second agent in. Pruning failed **open**, which is worse than not pruning
at all -- it deletes exactly the evidence the guard depends on.

Fixed by making liveness three-valued. `Liveness` returns Live, Dead or Unknown;
`PruneDead` prunes only on **Dead**; `Stop` signals on Live *or* Unknown, since
declining to signal because we could not confirm would free the tree while
leaving a running agent behind. `OSProcOps.Exists` now uses `syscall.Kill(pid, 0)`
directly instead of forking -- ESRCH is dead, EPERM is alive-but-not-ours,
anything else is Unknown -- which also fixed the probe outright in restricted
environments, since nothing forks any more. Rendering shows `unknown` distinctly
from `dead`.

The lesson is not "the fix was wrong" so much as **a repair aimed at a stale
record introduced a way to delete a live one**, and only running the operator
steps found it. This is the second time in this issue that BR-12's shape has
bitten: the value was in the steps I had ticked without running.

### 2026-08-22 — refusal confirmed against a live child
- 2026-08-22: closed — go test ./cmd/... -race green across 38 packages, green in a pristine worktree of the same commit (couchcmd tests are hermetic; no test can reach a production seam), and `make test-live` runs the opt-in conformance suites. All blocking findings from three review rounds fixed, each with a regression test verified RED against the reverted fix. Round 3 addressed by rule and swept for the shape: a guard bypass is FlagOnly so `couch start /repo true` can no longer disable the refusal (the first, broader rule broke `couch describe <ref> <text>` and the sweep caught it); persisted cwd is canonical since StartArgs exists for replay; the real-probe guard pin is hermetic and ungated because a gated-only pin is not a pin; one seam instance per Runtime so CLI tests can hold two distinguishable actors. Operator smoke confirmed: pair+zellij+nvim hosts as a couch child, second-shell couch list shows the live actor with pid, couch show returns one tree not everything, and a second start against a live incumbent is refused.; review verdict: FIX-THEN-SHIP

Operator re-ran the refusal after the fail-closed fix: a second `couch start`
on a tree with a live incumbent is refused. That was the step that failed
before and surfaced the fail-open pruning bug, so it is the one that needed
re-running rather than assuming.

Smoke coverage now: hosting a real pair child, the second-shell read path
(`couch list` showing the live actor with its pid), `couch show <ref>` returning
one tree rather than everything, and the refusal. The kbench-subdirectory case
is still unrun by hand; it is covered by a unit test and by live git
conformance against a real linked worktree, and is recorded as such rather than
ticked as operator-verified.
