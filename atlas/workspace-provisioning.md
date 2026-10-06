# Numbered workspace provisioning

`pair#305` adds the repeatable preparation operation used by the slots v2
lifecycle work. It prepares a working directory and launches no agent.

```sh
couch --internal provision-workspace /absolute/fleet/pair --slot=1
# Select a configured remote when main's upstream does not resolve ambiguity:
couch --internal provision-workspace /absolute/fleet/pair --slot=1 --remote=upstream
```

The host is `/absolute/fleet/worktree/pair-slot1/pair`, with resting branch
`main-slot1`. Dependencies are ordinary sibling clones prepared by Weave.
The primary checkout's local commits and dirty files stay untouched.

## One operation, including recovery

`WorkspaceProvisioner.Ensure` is the slot reconciler (#387). It validates the
primary checkout through `sdlc workspace --json`, then converges the slot's
resources (below) to a working slot. Creating a slot, repairing one after a
crash or an interrupted setup, and re-adding a deleted one are the same
operation, so repeating it after any error or interruption finishes what is
provably missing. There is no retry flag.

A new resting branch starts at the selected remote's `main`, captured from
`git fetch --verbose --porcelain` output with a zero-OID compare and swap. A
branch, directory or registration Couch did not itself create is adopted, never
refused: its commits and files are kept. A verified host keeps its active branch,
dirty files, commits and upstream.

`ProvisionResult` (JSON for `couch --internal provision-workspace`) carries the
address, host path, resting branch, baseline SHA and disposition (`created`,
`prepared` when a compile ran, `reused`), plus `warning` when the slot is usable
but something did not converge. A blocking failure returns nonzero, with the
advice text described below.

## Slot resources (#387)

A `:1+` slot's state is spread over git, the filesystem, weave and Couch, so it
is modeled as a table of resources rather than one thing. The table is
`SlotResources()` in `cmd/internal/couchcore/slotresource.go` and is the single
source: each row has an owner kind, `DependsOn` edges (converge order is
`SlotResourceOrder()`), its `SlotLayout` locations, and whether reconcile may
remove it.

| Resource | What | Kind |
|---|---|---|
| `env` | `<fleet>/worktree/<repo>-slotN` | derived |
| `store` | `<env>/.couch` (incl. `saved-work/`) | Couch internal, preserved |
| `intent` | legacy `couch-workspaces/N/creation.json` | derived, desired absent |
| `branch` | `main-slotN` | user data (adopted, never deleted) |
| `upstream` | `branch.main-slotN.{remote,merge}` | derived |
| `registration` | `<common>/worktrees/<name>` | derived |
| `host` | `<env>/<repo>` | derived checkout (set aside, never deleted) |
| `deps` | the host's `construct/deps` graph | external (read through ariadne `layergraph`) |
| `dep:<rel>` | each dependency clone in the env | derived checkout (set aside, never deleted) |
| `setup` | weave compile + `couch-setup-success.json` | derived |
| `agent` | the slot's agent session and record | runtime, observed only |

- **Paths.** `SlotLayout` (`slotlayout.go`) spells every slot-state path and name.
  `TestEverySlotStateSiteIsAResource` reads the production AST, so a literal or join
  that builds slot state elsewhere fails the build.
- **Observation.** `ObserveSlot` (`slotobserve.go`) reads each resource as
  present / absent / broken / unknown, with a reading (stale, unreadable,
  lock-held …), and changes nothing:
  - A failed probe is unknown, never absent.
  - A checkout is broken only on positive evidence: git answers "not a git
    repository".
  - A resource waiting on an unconverged dependency is pending.
- **Plan.** `PlanSlot` (`slotplan.go`) is pure, and its rules are checked over every
  single and pair perturbation of the table:
  - unknown blocks its dependents;
  - a checkout is only ever set aside, never deleted, and never under a running
    or unknown agent;
  - a stale registration is re-added with `git worktree add --force` on the branch
    it records;
  - `:0` is refused.
- **Display.** `couch --show repo:N` (or a slot path) prints the resources in table
  order and the plan, including for a slot with no thread left.
- **Converge loop.** `reconcileLoop` (`slotreconcile.go`) runs observe → plan → apply
  until the plan is empty or holds only stops. It keeps no state between runs, so
  recovery from any interruption is running it again.
  - A step the observation still calls for after the run executed it is "no
    progress", never an empty plan.
  - Git and filesystem steps run under the host creation lease, which is released
    for `weave compile`.
  - Steps (`slotconverge.go`) re-check their own precondition: the zero-OID compare
    and swap on `main-slotN`, existing upstream kept, `git worktree add --force` on
    the stale registration's own branch, `git worktree repair` as an attempt judged
    by the next pass.
- **Saved work.** A checkout broken on positive evidence (git: "not a git
  repository") is never deleted. `SetAside` (`slotsave.go`) moves it whole into
  `<env>/.couch/saved-work/<name>-<time>/tree`, with a manifest written pending
  before the move and complete after. The move happens under weave's setup lock,
  after re-checking the agent; it is capped at 16 entries per slot. The reconciler
  that writes saved work also collects it: every run removes entries older than
  `storagegc.RetentionPeriod` (by `saved_at`, else the entry's age). The result's
  `SetAside` lists what a run moved and how to restore it.
- **Failures.** `slotfailure.go` classifies each failure:
  - retryable: weave's "setup is active", a busy lease, a timeout, cancellation;
  - hand-off: everything else, with weave's own `Error:` line as the cause;
  - unknown: a probe failed;
  - hold: a live or unknown agent, or saved work full.

  `ReconcileAdvice` is the one operator text. Only retryable says "run it again",
  and a hand-off goes to the repository's `:0` agent. A live-agent hold names
  reboot, whose first reconcile pass gets past exactly that hold, stops the agent,
  and repairs in its post-stop pass. An unknown-agent hold says to look again
  later, because reboot cannot act on an agent it cannot observe either. Every
  converge-step refusal is a typed error that classifies into the class the plan
  gives it.
- **Outcome.** `OutcomeSeverity` reads the final observation. It is blocking (the
  caller refuses) exactly when the host is not present or setup never completed;
  everything else is degraded (the caller proceeds and prints the warning). Open,
  resume, reboot and a fresh start reconcile first through `selectedSlot`, with the
  agent evidence from `ObserveSlotSessions`. Reboot's first pass holds under the
  live agent, and its post-stop pass repairs.
- **Known failures.** A hand-off setup failure is remembered beside the marker
  (`couch-setup-attempt.json`, keyed by a digest of `HEAD`, the
  `construct/deps` files and the weave on `PATH`: its resolved file, size and
  modification time, so an upgraded weave retries). Later opens do not recompile until an input changes or
  `couch --reconcile repo:N` asks for it explicitly.
- **Add slot.** `Discover` unions environment directories, registrations and
  `main-slotN` refs. A number known only from leftovers is reusable
  (`ErrSlotNeedsReconcile`), and any other broken slot skips only its own number.

## Ownership and interruption

- One close-only inherited flock at `<common-git-dir>/couch-workspaces/creation.lock`
  covers host Git creation. Contention returns busy; no silent queue or renumbering.
  Git children retain exclusion if their Couch parent dies.
- The legacy `<common-git-dir>/couch-workspaces/<N>/creation.json` is retired
  (#387): reconcile removes one it finds, and nothing writes it any more. Recovery
  after an interruption is observation, not a recorded intent.
- `<host-git-admin-dir>/couch-setup-success.json` stores initial setup success.
  Publication and owned temporary cleanup briefly reacquire the same creation lock.
  A concurrent valid marker wins. Weave runs outside this lock under its own lock.
- Commands run sequentially with bounded cancellation. Local probes have 5-second
  ceilings, fetch 120 seconds, compile 20 minutes. Structured output is capped at
  1 MiB, records at 16 KiB, in-memory diagnostic tails at 64 KiB.

Couch never resets, prunes or deletes a worktree, branch, clone or directory. It
adopts what it finds at a slot's conventional place (#387), removes only its own
stale registration (`git worktree add --force` over it), and moves a checkout that
git cannot read into saved work instead of deleting it. Never delete a lock file to
override a live writer. Malformed records require inspection.

The operator owns retained worktrees and their disk use. Discover hosts through
`git worktree list`; remove them with ordinary deliberate Git worktree removal,
then inspect sibling dependency clones before removing their environment.

## Lifecycle boundary

#305 supplies directory readiness only. It has no thread reservation or launch
side effects and does not acquire the singleton Couch supervisor lease.
`SelectStartSlot` chooses the lowest free number, including :0; existing
directories remain durable slots even without a usable conversation, so an
archived or threadless checkout can be reused in place. Partial or uncertain
candidates require attention instead of being skipped.

`pair#306` connects readiness to ordinary slot open/cold resume and new-slot
creation. Warm reattachment reconnects a running agent without compilation.
Primary :0 setup behavior is unchanged. A create fills the lowest number, :0
included, whose checkout holds no thread (`SelectStartSlot`, #331/#332):
archived threads free their number, and a threadless numbered checkout is
reused as-is with a fresh conversation (`StartResolution.ReuseSlot`, in the
fingerprint). Only when every number is taken is a new directory provisioned.
Parked threads occupy their number but no longer block a new slot; the start
preview names them as `open-slot` reuse suggestions. Lost bindings are named as
`fresh-slot` suggestions, and unreadable threads still refuse. Preview carries its exact chosen target and has no setup or
migration effects; submission refuses changed selection instead of renumbering.

Couch state lives beside the Git host at `<environment>/.couch/`: `thread.json`,
`preferences.json`, derived `continuation.md`, retained `archive/` and archive
clocks, plus existing journal/lock files. The environment remains after failed
starts and conversation replacement. Native transcripts and Pair sidecars stay in
their existing stores. [Couch](couch.md) maps enrollment, local storage and GC.

## Code and verification

- `cmd/internal/couchcore/provision.go`: Ensure and Git reconciliation.
- `workspace_identity.go`, `provision_request.go`, `provision_host.go`,
  `slotallocation.go`: checked transport and pure decisions.
- `provision_io.go`, `provision_lock_unix.go`, `provision_store.go`: bounded
  process execution, inherited lease and strict atomic metadata.
- `ops.go`, `operationdispatch.go`, `cmd/internal/couchcmd/run.go`: internal
  operation, injected runtime, progress/JSON and scoped CLI cancellation.
- `provision_*_test.go`: real Git, stateful external outcomes, interrupted effects,
  process death and concurrency. CLI tests prove no supervisor or agent launch.

The live conformance test uses installed SDLC/Weave with isolated local Git
transport and minimal manifests; it installs no packages or tools:

```sh
PAIR_LIVE_WORKSPACE=1 go test ./cmd/internal/couchcore -run '^TestProvisionConformance$' -count=1 -v
```

Run this live check whenever provisioning or the consumed SDLC/Weave identity or
setup contract changes, and during pair#309 acceptance. Ensure rejects corrupt
or conflicting observations before consulting NextHostAction. The creation lock
is nonblocking, including after compile: contention can require another readiness
call and repeat compile if no success marker was published.

## Console output ownership

Couch's CLI streams provisioning diagnostics to stderr only when no terminal
console is active. A console-bound Couch uses a discard progress writer for both
initial setup and later menu operations; its renderer owns the screen and existing
operation notices show progress. `OSProvisionIO` still retains the bounded command
diagnostic tail and includes it in setup failures. Raw Homebrew/weave output must
never paint over the switcher. (`pair#312`)
