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
`SelectNewSlot` chooses the lowest unused positive number; existing directories
remain durable slots even without a usable conversation. Partial or uncertain
candidates require attention instead of being skipped.

`pair#306` connects readiness to ordinary slot open/cold resume and new-slot
creation. Warm reattachment reconnects a running agent without compilation.
Primary :0 setup behavior is unchanged. A create on the primary path starts on
:0 whenever :0's own scope holds no thread -- existing numbered slots and their
threads do not occupy it, so an archived primary is reused (#331). A parked
primary or numbered thread blocks adding another slot; opening or starting
fresh within an existing slot remains available. Preview carries its exact chosen target and has no setup or
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
