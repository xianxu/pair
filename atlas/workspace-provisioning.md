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

`WorkspaceProvisioner.Ensure` validates SDLC workspace JSON v2 and Git linkage.
For a new host it fetches the selected remote's main, captures the full SHA from
`git fetch --verbose --porcelain` output, records creation intent, and creates the
branch/worktree with remote/main upstream. Git must support fetch porcelain;
unsupported Git fails visibly. A later fetch cannot change that captured value.

A verified existing host retains its active branch, dirty files, commits and
upstream. Without a valid setup-success marker, Ensure runs `weave compile` from
that host. Weave owns dependency setup, exclusion and partial-clone recovery.
Couch records success only after exit 0 and final identity validation.

Repeat the same operation after an error or interruption. There is no retry flag.
A missing marker runs compile again even if the prior compile actually completed.
A valid marker skips fetch and compile. The marker records initial success; it is
not a check of current build output or dependency contents. Later source changes
and missing dependency files require an explicit Weave/build operation.

`ProvisionResult` is JSON on stdout: schema version 1, address, host path,
resting branch, baseline SHA, and disposition `created`, `prepared`, or `reused`.
Progress and diagnostics go to stderr; failure returns nonzero and no result.

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
- **Not converged yet.** M1 delivers the model, the observation and the plan. The
  converge loop that replaces `Ensure`'s repair logic is M2 of #387.

## Ownership and interruption

- One close-only inherited flock at `<common-git-dir>/couch-workspaces/creation.lock`
  covers host Git creation. Contention returns busy; no silent queue or renumbering.
  Git children retain exclusion if their Couch parent dies.
- `<common-git-dir>/couch-workspaces/<N>/creation.json` retains a captured baseline
  and branch/directory ownership evidence while Git creation or setup is incomplete.
  Repeated calls finish only provable missing steps. It is removed after success;
  a cleanup error retains one bounded file for a later call.
- `<host-git-admin-dir>/couch-setup-success.json` stores initial setup success.
  Publication and owned temporary cleanup briefly reacquire the same creation lock.
  A concurrent valid marker wins. Weave runs outside this lock under its own lock.
- Commands run sequentially with bounded cancellation. Local probes have 5-second
  ceilings, fetch 120 seconds, compile 20 minutes. Structured output is capped at
  1 MiB, records at 16 KiB, in-memory diagnostic tails at 64 KiB.

Couch never resets, prunes, deletes or takes over an unverified worktree, branch,
clone or directory. If ownership cannot be established after interruption, inspect
`git worktree list --porcelain`, the named path/ref and creation intent. Resolve
the discrepancy deliberately before repeating the operation; never delete a lock
file to override a live writer. Malformed records also require inspection.

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
