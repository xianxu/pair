---
id: 000149
status: working
deps: []
github_issue:
created: 2026-08-21
updated: 2026-08-26
estimate_hours: 17.80
started: 2026-08-25T14:21:34-07:00
---

# couch: opaque tags and a human naming layer

Project: `workshop/projects/couch.md`. Successor to `pair#145`.

## Problem

pair's **tag does two jobs at once**: it is the durable storage key
(`draft-<tag>.md`, `ledger-<tag>.jsonl`, `log-<tag>.md`,
`scrollback-<tag>-<agent>.raw`) *and* the human handle typed into
`pair <tag>`. Three symptoms follow from that one conflation:

- **Naming is demanded upfront.** A space cannot exist before it has a name,
  so the operator must know what a thread is before starting it. That is
  backwards — the constitution's own flow has an issue crystallise mid-thread,
  which is why `sdlc claim` is cheap and the estimate comes later.
- **Renaming is not offered**, because a rename would be a filesystem move.
- **Tags accumulate with no cleanup.** Nothing distinguishes a thread worth
  keeping from an abandoned one, because everything has a name and nothing
  records whether the name was deliberate.

couch made the same split two days ago for its own identity (opaque system id,
mutable human labels over it — see the project's 2026-08-21 scope event). This
applies it one layer down, to pair.

## Spec

**Tag *is* the space.** No new noun: the generated id becomes the tag, and the
tag stays with the space for its whole life. `couch-ab50125e` is therefore
**durable**, not per-incarnation — which reverses `#145`'s framing of ActorID.

**The path is an attribute of the space, not its identity.** This supersedes
`#145`'s tree-as-key model rather than extending it. `#145` keyed the registry
on the worktree and enforced one agent per tree; here the key is the tag and
the absolute starting path is one of its attributes.

The inversion matters because a path is *where work happens and where artifacts
are stored*, which is what comes to mind first — "I want to work in pair", "I
want to work on arc-agi-3". `couch start <file-path>` says *start me somewhere*.
The repo portion of that path is merely a container that supplies git as a
facility: history tracking and work isolation. It is not identity.

**Concurrency becomes a configurable repository policy**, replacing "one agent
per tree" plus an escape hatch. The policy derives a conflict key and live limit
from the repository's work shape:

- `pair`, `ariadne`, `parley` — the checkout is the installation, one branch and
  one index, so the repository root is the conflict key and its limit is 1.
- `brain` — a capture repo where threads append to different files, much like
  separate chat threads; the policy is unbounded (realistically fewer than five
  live threads), and no override is involved on the normal path. Under `#145`'s
  model this case needed `--same-tree` every time, which is a smell: an escape
  hatch on the ordinary path.
- `kbench` — each competition directory is a conflict key with a limit of 1;
  distinct competitions may run concurrently in the shared checkout.
- worktree-managed application repositories — each generated worktree path is a
  conflict key with a limit of 1; repository policy also owns creation and later
  garbage collection of those worktrees.

`--same-tree` therefore stops being a special flag and becomes "exceed the
configured limit", which is a cleaner thing to announce loudly.

**`couch start <path>` always creates a new space.** With the path an attribute
rather than a key, `start` cannot mean "resume whatever is there" — there may be
zero or several. Resumption is explicit: by tag, or by a name once one is
attached. This is a real UX change from what `#145` shipped and is easier to
decide now than to discover later.

**Different surfaces lead with different attributes.** One record, several
views: the picker leads with the path or the name, `couch list` with the name,
a log line with the hex tag. That is the point of attributes rather than a
single canonical display string.

**`Spawn` looks the id up; it does not mint one per run.** Minting per spawn
fragments the draft, ledger and scrollback across revivals, which is exactly
the continuity the tag exists to hold.

**A simplification falls out:** with the id durable, no separate incarnation id
is needed. `{PID, Identity}` already identifies one run, which is what
`pair#147` needs to drop a reply from a dead incarnation.

**Human names are a mutable attribute layer over the opaque tag** — assignable
whenever, renamable freely, never a filesystem key.

**Names live in pair's session index, not couch's.** couch failing must not
block work, so pair has to resolve names standalone; couch reads that index
rather than keeping a second store. Two stores of human names would drift
immediately. `launcher/session_index.go` already keeps
`SessionNameEntry{scopeKey, tag, sessionName, superseded}` with collision
ladders — the indirection is half-built, used today only to compose zellij
display names.

**Opaque tags and the picker ship together.** Once tags are opaque, pair's
picker is a list of hex strings unless names land in the same change. An
intermediate state is strictly worse than today.

**An unnamed space displays its hex string.** Decided deliberately: no
inference from draft contents or recency. Simple, and unnamed is the default
state rather than the exception.

**`pair claude` standalone keeps today's behaviour** — it still asks the
operator to name a tag. Only couch-launched sessions mint a hex tag for the
repo.

**Naming becomes the retention signal.** A space that is unnamed, cold, and
holds a trivial draft is a cleanup candidate; one the operator bothered to name
is not. That is a defensible GC criterion without adding a separate keep-this
flag — worth designing in now even if the sweep itself lands later.

**Stop rendering `couch-ab50125e` in `couch list`'s common output.** `#145`
put the system id in the first column of every row, contradicting the project's
own decision that the system id need not be legible. Lead with the human name
and tree; keep the id for `show` and diagnostics.

### Relationship to existing work

- **`#135` (live cross-agent handoff)** treats the tag as the durable work
  identity. Under this model that stays true and gets sharper: "one live driver
  per tag" becomes "one live driver per space", which composes with couch's
  one-agent-per-tree rather than competing.
- **`--same-tree`** yields two spaces in one tree — two drafts, two ledgers,
  two ids. That falls out rather than needing a special case.

The limit counts live actor incarnations only. Parked work threads remain in the
inventory but consume no live concurrency slot. These policies are the couch
consumer of the repository strategy model tracked in `ariadne#200`; they are not
hard-coded actor subclasses.

## Revisions

### 2026-08-24 — “space” becomes the durable work thread

**Reason:** designing `#146`'s panel actions exposed that an actor/process is the
wrong durable row. The operator returns to a thread of work whose transcript,
draft, ledger, continuation, human name, and description survive after the
harness stops. A path may host several such threads (ordinary behavior in
brain), and one thread may be inactive without ceasing to exist.

**Delta:** **work thread** supersedes **space** as the human-facing noun in this
issue. The opaque Pair tag is the work thread's durable ID. Its starting/current
path is an attribute, and the system maintains the conceptual index
`path → [work threads]`; identity and human metadata belong to each work thread,
not to that index edge. A thread has zero or one live actor incarnation. The
actor is the runtime—deterministic couch actor plus agent harness/native LLM
session—not the continuity record. `{thread ID, process identity}` is sufficient
to reject replies from an obsolete incarnation; no second durable actor ID is
introduced.

The lifecycle vocabulary follows the identity:

- **park** succeeds only after every process in the live incarnation that can
  modify the workspace has stopped and durable output has been flushed. It then
  frees the configured concurrency slot while preserving the work thread and
  all durable context. A surviving zellij session or agent process means the
  thread is still live; a partial stop is a failed park and retains its slot;
- **resume** creates a new live incarnation using the same opaque tag and may
  reattach a native agent session whose durable resume identity belongs to that
  tag;
- **archive/forget** is a later retention/garbage-collection decision about the
  durable work thread, not a synonym for stopping its process;
- **kill** may remain a low-level recovery action for a wedged harness, but is
  not the normal thread-menu verb.

The eventual couch panel lists work threads, including inactive historically
active ones. Enter attaches to a live thread and resumes a parked one. Tab opens
thread-level actions; rename and description therefore target the selected
thread without ambiguity. Multiple threads at one path are distinct rows even
when unnamed. The hierarchical menu is sequenced in `#151`, which depends on
this issue; `#146` keeps its flat transitional worktree panel rather than
building an actor submenu that would immediately be discarded.

Thread summaries expose exact live/parked state and a durable `last_active_at`.
A live thread presents as active now. When successful park or reconciliation
verifies that the entire incarnation is no longer able to modify the workspace,
couch monotonically records the time of that observation in pair's
thread/session index; a child-client exit alone, failed park, or unknown
liveness does not advance it or free the slot. The timestamp survives couch
restart and is never supplied by the agent. The panel may map its age to
progressively dimmer terminal grays, but color is only a secondary cue:
live/parked state and relative age remain readable in text and on terminals
without grayscale.

The concurrency questions are also settled by repository policy rather than one
global granularity: singleton local-tool checkouts key at repo root, brain is
unbounded in place, kbench keys by competition directory, and worktree-managed
repos key each generated worktree. Only live incarnations count. This records
the operator decisions already captured in `ariadne#200` and removes the stale
per-repo-versus-per-path open question.

### 2026-08-24 — park and recency become observed lifecycle facts

**Reason:** spec review found that “end or suspend” could call a detached but
still-running zellij/agent process parked and release its collision guard. It
also found no authoritative event behind the proposed recency display.

**Delta:** park is now an all-or-fail transition to no workspace-writing process;
only its success frees capacity. Resume always creates a new couch incarnation,
though the underlying agent may use its persisted native resume ID. A monotonic
`last_active_at` is persisted on an observed live→parked transition and remains
unchanged for failures or unknown liveness. Done-when now enumerates every
repository-policy case rather than accepting one generic limit test
(ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-25 — split durable thread identity from process quiescence

**Reason:** fresh spec review found that verified parking is not a storage
detail. Proving that a Pair-owned zellij session, agent harness, editor, and
every recorded workspace-writing process have stopped—and that durable output
has flushed—needs a coordinator, observable acknowledgements, timeout and
recovery behavior. That is an independent subsystem shared with `#135`'s live
handoff problem. Worktree provisioning and garbage collection are likewise a
separate lifecycle from consuming a resolved admission policy.

**Delta — current #149 boundary:** this issue delivers the durable work-thread
record, opaque tag allocation, mutable human metadata, legacy migration,
explicit create/resume semantics, picker integration, and the `ariadne#200`
policy consumer. Verified park plus authoritative `last_active_at` move to
`#152`; generated-worktree provisioning and garbage collection move to `#153`.
`#151` depends on both this issue and `#152`, so its menu never presents a park
action whose lifecycle contract does not exist.

**Authoritative record:** Pair owns one portable record per work thread under
the global Pair data directory. The record is keyed by the opaque Pair tag and
contains a schema version, tag, immutable canonical physical working path,
repo-scope identity, optional human name, optional description, and creation
time. It is the source for thread inventory and human resolution whether couch
is running or not. Couch's registry contains only optional live incarnations,
each referencing a thread tag plus `{PID, process identity}` and the normalized
policy result used for its admission. The existing session-name index remains
the zellij socket-name binding; a mutable human name is not overloaded onto its
`SessionName` field (ARCH-DRY, ARCH-PURE).

One atomic file per thread avoids a read-modify-rewrite race between standalone
Pair and couch. Rename/describe replace only that thread's file atomically.
Names need not be unique: an exact tag wins; an exact or fuzzy name/path that
matches several threads returns all candidates and refuses to guess. Empty name
clears it. An unnamed row leads with the opaque tag, then its path; “do not lead
with system id” applies to named rows, whose common rendering leads with name.

**Path and allocation:** creation resolves symlinks and persists the exact
canonical physical requested path, including a nested kbench competition path;
it is not silently replaced by the repository root and does not drift with the
caller cwd. The path is immutable in this issue. A missing/moved path leaves the
thread enumerable but resume refuses with an exact recovery diagnostic; an
explicit relocate operation is future scope. IDs come from cryptographic
randomness and are accepted only after collision checks against both thread
records and tag-scoped artifacts; bounded retry exhaustion or entropy failure
is an error and never reuses a tag.

**Commands and transitions:** `couch start [path]` always reserves a new thread
record before forking and rolls back that exact still-pristine reservation if
the fork fails. `couch resume <ref>` resolves one existing parked thread and
spawns `pair resume <tag> --layout2` in its recorded path. The existing
one-shot repo-default handoff remains authoritative for agent arguments and
native-session restoration; #149 adds no second picker. Resume refuses an
ambiguous ref, missing path, known-live incarnation, or unknown liveness. A
live child already hosted by the current console attaches through the existing
console-local target rather than resume; cross-process attachment remains
`#147`.

**Provider boundary:** couch calls `sdlc fleet policy --path <path> --json`
through an injected `PolicyResolver` before forking. It persists the returned
repo identity, admission key, tagged bounded/unbounded capacity, and optional
bounded `on_capacity` action on the incarnation. Occupancy counts known-live
and unknown incarnations conservatively by normalized key; dead records are
pruned first. Missing, invalid, or outside-scope diagnostics fail closed. A
bounded `reject` refuses; `provision-worktree` reports that `#153` is required
without fabricating a path. Pair does not parse declarations or retain its
temporary repo-name `PolicyTable`; a stateful fake and live provider conformance
exercise the same seam (ARCH-DRY, ARCH-MOCK, ARCH-PURPOSE).

This is a coordinated cross-repo boundary rather than a circular whole-issue
dependency. `#149 M1` lands and boundary-reviews the normalized policy consumer
against ariadne#200's locally built provider; ariadne#200 then closes and merges;
later #149 milestones finish durable identity and naming against the merged
provider before #149 closes.

**Legacy migration:** existing path-derived Pair tags remain the durable tag,
so draft, ledger, log, scrollback, saved native-session identity, and detached
zellij session are not renamed. A legacy tree-keyed name/description migrates
onto that tree's single path-derived thread. Legacy `--same-tree` co-tenants
already share one Pair tag and therefore cannot be truthfully split; they are
retained as multiple incarnations of one migrated thread and fail closed for
new admission/resume until reconciliation leaves at most one. Registry schema
migration is versioned and atomic; an unreadable or ambiguous legacy record is
reported and preserved, never dropped.

### 2026-08-25 — make the thread store transactional and remember launch profiles

**Reason:** a second fresh review found that atomic replacement of one record
does not serialize concurrent start/name/migration operations, that a global tag
alone cannot address legacy tags repeated in different repositories, and that
start spans several failure-prone effects. The operator also specified that a
new actor should default to the LLM and parameters last used at that path.

**Delta — address and authority:** the durable address is the composite
`{repo_scope, tag}`. Newly generated tags remain globally improbable, but
legacy repeated tags resolve within their repository scope and never overwrite
one another. A Pair-owned `ThreadStore` is the sole authority for work-thread
records, migration-era incarnations, admission evidence, lifecycle
transactions, and launch preferences. All cross-record mutations take one
cross-process store lock; every record also carries a monotonic revision so a
stale writer fails compare-and-swap rather than losing another process's edit.

Tag allocation uses `couch-` plus 16 lowercase hexadecimal digits from
`crypto/rand`. Creation is an atomic no-replace claim checked against both the
composite record and tag-scoped artifacts. Eight collisions are retried;
entropy failure and exhausted retries are distinct errors and neither reuses an
existing tag.

**Start transaction:** one durable nonce makes start recoverable and idempotent:

1. canonicalize the requested path and resolve its current provider policy;
2. under the store lock, reconcile stale policy evidence, prune only proven-dead
   incarnations, count live/unknown/creating occupants, and atomically claim the
   thread plus a `creating` reservation;
3. release the lock and fork Pair with the resolved launch profile;
4. reacquire the lock and register `{PID, process identity}`, the normalized
   admission result, transaction nonce, and launch profile.

A pre-fork failure removes only its matching pristine reservation. If the child
forks but registration fails, couch stops and verifies that child before
rollback; when verification is unavailable it leaves an occupied recovery
record, never a free slot beside an untracked writer. Startup reconciliation
uses the nonce, owner PID/identity, and child identity to finish or report the
same transaction without duplicating it (ARCH-PURE, ARCH-PURPOSE).

Persisted admission evidence includes the provider schema version and canonical
declaration digest. A digest change re-resolves every live, unknown, or
`creating` incumbent before a new admission decision; unresolved or invalid
incumbents conservatively occupy capacity. A creating transaction whose target
no longer resolves to the same policy remains occupied and enters recovery
rather than being rekeyed beneath its blocked helper. Pair never treats cached
policy as authority.

**Launch profiles:** each thread records the agent and exact argument vector of
its latest successfully registered incarnation. Separately, the store keeps a
path preference keyed by normalized repository identity plus canonical physical
path: `last_agent` and the last exact argument vector used by each agent at that
path. A successful incarnation registration updates both facts atomically;
selection, cancellation, failed fork, or failed registration updates neither.

For a new thread the agent resolves in this order: explicit selection, the
path's `last_agent`, then the root actor's agent. Its arguments resolve from the
path's last arguments for that selected agent, falling back to Pair's declared
repository default for the agent. Agent choices come from Pair's shared agent
inventory, not a couch-specific enum. Thus changing from Claude to Codex does
not apply Claude arguments to Codex, and returning to Claude restores Claude's
last path-specific arguments. `#151` owns the Ctrl-Space selector and visibly
reports both resolved agent and argument source (ARCH-DRY).

**Command and migration corrections:** #149 does not expose couch resume until
`#152` can prove a thread parked. `couch name` and `couch describe` mutate the
selected composite thread record. The advisor's publish operation receives
`COUCH_THREAD_SCOPE` and `COUCH_THREAD_TAG` and stores its published summary in
a distinct field. Standalone Pair launches upsert the same store; its picker
resolves names only within repository scope and refuses ambiguity.

Legacy migration runs under the store lock with a schema version and durable
nonce journal. Each step is idempotent after interruption and preserves all
legacy files. Repeated tags in different repositories become distinct composite
records; same-tree co-tenants that cannot be separated remain multiple
incarnations of one thread and block admission until reconciled. `#152` writes
verified park, `last_active_at`, and incarnation removal through the same store
transaction. `#153` may explicitly rebind mutable `working_path` after
deterministic reprovisioning; `created_at` and `starting_path` remain immutable.

### 2026-08-25 — close composite-artifact and pre-exec recovery boundaries

**Reason:** review of the transactional revision found two remaining holes: a
composite metadata key did not yet guarantee composite artifact access, and a
parent could crash after fork before learning enough about the child to recover
it. It also found that dependent resume/worktree outcomes remained in #149's
acceptance boundary.

**Delta — composite means every durable lookup:** `repo_scope` selects Pair's
existing repository-scoped data namespace; `tag` selects the thread within it.
Every draft, ledger, log, scrollback, saved config, native-session identity,
picker, and resume API accepts or derives this same composite address and
validates that the artifact lives below the selected scope. `pair resume <tag>`
derives `repo_scope` from its canonical working repository; couch passes both
`COUCH_THREAD_SCOPE` and `COUCH_THREAD_TAG`. Cross-scope lookup is never a
tag-only fallback. Two legacy repositories may therefore retain the same tag
and original filenames without sharing a namespace or opening one another's
artifacts (ARCH-DRY, ARCH-PURPOSE).

**Delta — no untracked post-fork child:** start uses a close-on-exec handshake.
The child is forked into a tiny Pair launch helper with the durable transaction
nonce and blocks before it can start zellij, the agent, an editor, or any other
workspace writer. The parent durably records the child's PID and process
identity under that nonce, then acknowledges the pipe/socket and permits exec.
If the parent dies first, the helper observes EOF or a bounded timeout and exits
without exec. Reconciliation can therefore identify every surviving child from
the journal; an occupied `creating` record with no matching live helper is
recoverable as a failed pre-exec transaction, never an unknowable writer.

**Delta — acceptance boundary:** #149 ends with durable identity, transactional
start, metadata, inventory/picker, launch-profile storage, legacy migration,
and normalized policy consumption. It may report a typed
`provision-worktree` action but does not create that tree. Couch resume and
verified parked state are #152 outcomes; managed missing-path reprovision and
working-path rebind are #153 outcomes. Older start/resume and generated-tree
acceptance text is superseded by this boundary.

### 2026-08-26 — align the first boundary with transactional admission

**Reason:** implementation planning made a dependency in the milestone shorthand
explicit: M1 cannot count cross-process `creating` occupants safely if the sole
locked/revisioned store does not exist until M2. A temporary admission store
would create exactly the shadow authority this issue removes. Plan review also
made the provider-IO and one-file-per-thread persistence protocols explicit.

**Delta:** M1 introduces the final `ThreadStore` lock/revision transaction
kernel and the minimal per-thread admission/reservation schema. No legacy
registry writer participates in admission after M1. M2 widens that same store
for composite identity, tag claims, journaled start, and the blocked pre-exec
helper; it does not replace or rebuild the M1 authority.

Provider subprocess IO never runs while holding the global store lock. Admission
captures relevant record revisions under lock, resolves the candidate and stale
incumbents outside it, then reacquires the lock and either applies a pure
prune/group/claim mutation against unchanged revisions or retries. Resolution
failure remains occupied and no child forks.

The store retains one atomic file per composite work thread. Store-level
journals/manifests coordinate only mutations spanning multiple records (for
example thread plus path preference or migration), with idempotent recovery;
there is no monolithic all-thread snapshot in the final schema. Launch-profile
resolution records agent provenance (`explicit`, `path`, or `root`) separately
from argument provenance (`path` or `repo-default`), because those axes combine
independently.

M1 cuts admission over under the global lock: before the new authority can
admit, it idempotently imports every legacy registry actor into minimal
per-thread records, retains same-tree co-tenants as separate conservative
incarnations, and persists a manifest cutover marker. Corrupt or ambiguous
legacy input refuses the cutover. M5 later enriches these records and migrates
metadata/artifact access; it does not first discover live admission occupants.
During stale-policy refresh, every successful result for one repository must
share the candidate's provider version and declaration digest. A mixed epoch
retries the whole cohort; bounded exhaustion fails closed without forking.

### 2026-08-26 — make the couch store the singleton namespace

**Reason:** exposing thread naming to the root advisor raised the ownership
question directly: if two couch consoles can supervise the same durable store,
each can create a child the other cannot attach to, and “root” stops naming one
place. Conversely, making every couch process a new namespace would orphan
durable threads whenever couch restarts.

**Delta:** the canonical `COUCH_STORE_DIR` is the durable couch namespace. The
default Pair data store is the one ordinary namespace for this OS user. A
thread's complete address inside it remains `{repo_scope, tag}`; if multiple
stores are ever promoted to product behavior, the global address naturally
becomes `{couch_namespace, repo_scope, tag}` without changing stored records.
An explicit alternate store remains a test/isolation facility, not v1 UX.

Pair resolves the namespace once at process entry, before constructing the
lease, store, or local endpoint: make an explicit relative value absolute
against the startup cwd, normalize it, create the directory if absent, resolve
all physical symlinks, and reject any failure. That exact absolute physical path
is used for the lifetime lease, endpoint, ThreadStore, and inherited
`COUCH_STORE_DIR`; no child reinterprets a relative path from its own cwd.

Exactly one couch process holds the namespace's supervisor lease. The lease is
an OS advisory lock held for the owner's lifetime on a close-on-exec,
non-inheritable descriptor, so no spawned or execed child retains ownership.
Only after acquisition does the owner atomically publish its PID and
process-start identity for diagnostics; a refusal reports that identity only
after verifying it still denotes the lock owner. A crash therefore releases
authority without relying on stale file deletion, even while children survive.
Restarting couch acquires the same namespace and adopts its
durable inventory—it does not mint a new namespace. A second supervising
`couch start`, including consoleless mode, refuses with the current owner
identity. Read and metadata clients may transact against the locked ThreadStore,
but an operation that creates or attaches an actor must execute in the owner so
the resulting child belongs to the console that can route its terminal. Until
#147 supplies owner routing, an external actor-creating client refuses rather
than starting an invisible second supervisor (ARCH-PURPOSE, ARCH-MOCK).

The root actor and every child inherit the canonical namespace location plus
their composite thread address. A couch process incarnation may have an
ephemeral diagnostic ID, but it is never part of thread identity or artifact
addressing.

### 2026-08-26 — opaque identity joins admission in M1

**Reason:** the implementation gate found that normalized Brain policy would
admit two same-path starts in M1 while both still launched Pair's path-derived
tag. Those nominal threads would attach to one native session and share
artifacts until M2, violating the identity this issue exists to establish.

**Delta:** M1 allocates and atomically claims the final composite
`{repo_scope, couch-<16 hex>}` address for every new couch start before policy
admission/fork. It launches Pair with that exact tag and scoped environment, so
unbounded same-path starts are distinct from the first milestone. The M1 legacy
bootstrap alone retains path-derived tags for already-existing actors. M2 no
longer introduces identity; it widens the same record into the journaled
blocked-helper start transaction and restart reconciliation (ARCH-PURPOSE).

## Done when

- A couch-launched work thread gets a generated composite durable address;
  restarting couch reloads the same record, draft, ledger, and other scoped
  artifacts without reminting or crossing repositories.
- An operator can name a work thread after the fact and rename it, with no file
  moved and no state lost.
- pair's picker shows the human name where one exists and the hex string where
  none does, and resolves a name to its work thread with couch not running.
- `pair claude` standalone still asks for a tag exactly as it does today.
- `couch list` no longer leads with the system id.
- Two work threads in one tree keep separate drafts and ledgers.
- Local-tool policy rejects a second live work thread under the same repository
  root; brain policy admits multiple live in-place threads without an override;
  kbench admits distinct competition directories but rejects two live threads
  in one competition; and full worktree capacity returns the provider's typed
  `provision-worktree` action without allocating a path in #149.
- `couch start <path>` twice creates two distinct work threads where the limit
  allows it, and a rejected or interrupted start cannot leave an untracked
  workspace writer or falsely free reservation.
- A thread records the exact agent/arguments of its latest successful
  incarnation, while new-thread defaults remember the last successful agent and
  per-agent arguments at that canonical path and fall back to the root agent and
  Pair repository defaults as specified.
- Thread inventory distinguishes multiple threads at one path and exposes exact
  live versus non-live/unknown state; verified parked state, resume, and
  historical-age presentation follow `#152`.
- Relative and symlinked spellings resolve to one physical couch namespace; one
  console or consoleless supervisor owns it at a time, a concurrent supervisor
  refuses with verified owner identity, and killing the owner releases the
  lease immediately even if its child remains alive.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so this is provisional but uses the required method. The
service-scale item is the lock/revision/WAL-backed ThreadStore and recoverable
start authority; the remaining items separate its OS, policy, UI, migration,
and integration boundaries.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: smaller-go-module design=0.06 impl=0.16
item: greenfield-go-module design=0.40 impl=0.32
item: api-integration design=0.60 impl=0.60
item: greenfield-service design=2.00 impl=2.80
item: greenfield-go-module design=0.40 impl=0.32
item: cross-cutting-refactor design=0.20 impl=0.20
item: smaller-go-module design=0.06 impl=0.16
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: smaller-go-module design=0.06 impl=0.16
item: smaller-go-module design=0.06 impl=0.16
item: skill-or-dispatcher design=0.40 impl=0.16
item: tui-screen design=0.40 impl=0.40
item: smaller-go-module design=0.06 impl=0.16
item: smaller-go-module design=0.06 impl=0.16
item: smaller-go-module design=0.06 impl=0.16
item: greenfield-go-module design=0.40 impl=0.32
item: cross-cutting-refactor design=0.80 impl=0.20
item: cross-repo-refactor-small design=0.06 impl=0.12
item: real-api-discovery design=0.00 impl=0.24
item: atlas-docs design=0.20 impl=0.16
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
design-buffer: 0.15
total: 17.80
```

## Plan

- [x] M1 — introduce the final locked/revisioned `ThreadStore` kernel and its
      minimal per-thread admission/reservation schema; allocate and atomically
      claim final composite opaque addresses for new starts; consume
      ariadne#200's versioned/digested normalized policy through an injected
      resolver, reconcile stale evidence, account for live/unknown/creating
      occupants, remove every legacy admission writer and the shadow policy
      table, enforce one supervisor lease per store namespace, and close this
      exact milestone before ariadne#200 closes.
- [ ] M2 — widen that same `ThreadStore` for the blocked pre-exec handshake,
      journaled start transaction, and restart reconciliation.
- [ ] M3 — add mutable name/description/published-summary operations, scoped
      standalone Pair resolution, shared inventory, and common rendering without
      a leading system id.
- [ ] M4 — persist per-thread and per-path/per-agent launch profiles; resolve
      explicit/path/root agent defaults and path/repository argument defaults,
      updating preferences only after successful registration.
- [ ] M5 — migrate legacy tags, artifacts, sessions, registry state, and
      same-tree co-tenants idempotently under the store lock, proving every
      artifact lookup is scoped by the composite address.
- [ ] Reconcile `#135` with composite work-thread identity; leave verified park,
      couch resume, and `last_active_at` to `#152`, and managed path rebinding to
      `#153`.

## Log

### 2026-08-26 — M2 recoverable-start record boundary

The store schema now groups each nonce with the exact supervising process
identity that created it; the incarnation's existing PID/start-token pair names
the blocked helper before exec and remains stable across exec. Validation permits
the pre-fork and helper-recorded creating states, rejects partial identities,
unsafe nonces, live records carrying an unfinished start, and more than one
tracked start per thread, while continuing to read M1 and legacy incarnations.
`launcher.ScopedPaths.Validate` and shared repo-scope/tag validators establish
the same composite boundary for the upcoming helper instead of duplicating path
rules (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

The initial tests failed to compile before the new model existed. Focused and
race runs now pass for `./cmd/internal/couchcore ./cmd/internal/launcher`.

### 2026-08-26 — M2 blocked pre-exec helper

`pair-launch-helper` now owns the only pre-exec wait: descriptor 3 carries one
exact acknowledgement byte, is close-on-exec, and is closed before replacing
the helper with Pair. EOF, a wrong byte, or a bounded timeout exits without
target exec. Both stdio and PTY runners pass the same pipe capability and retain
their existing handle/terminal behavior; `FakeRunner` models blocked, acked,
cancelled, and exact exec-count state across calls (ARCH-PURE, ARCH-MOCK).

Subprocess regressions observe no target marker before acknowledgement, exactly
one afterward, and none after cancel/EOF. Focused, race, and command-build tests
pass for couchcore, couchcmd, ptychild, and `cmd/pair-launch-helper`.

The pure `StartTransaction` projection and `AdvanceStartTransaction` now own
the legal claim → helper-recorded → registered sequence. Generated interruption
cases drive `ReconcileStart`: absent evidence rolls back only when the relevant
owner/helper is proven dead; unknown process or registration evidence remains
occupied; established Pair evidence promotes a live exact helper to live and a
gone/unverifiable helper to conservative unknown. The same transition sequence
runs against `FakeRunner`, pinning zero target execs before durable helper state
and exactly one after acknowledgement (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-26 — M1 admission kernel integrated
- 2026-08-26: closed M1 — make test; go test ./... -count=1; go test -race ./cmd/internal/couchcore ./cmd/internal/launcher -count=1; make test-couch-policy-live SDLC_BIN=../ariadne/bin/sdlc; zellij config and main-2/main-3 layout validation; git diff --check; review verdict: SHIP

Added the locked/revisioned composite ThreadStore, strict normalized fleet
policy consumer, and optimistic admission reconciliation. New Couch starts now
claim their final opaque `couch-<16hex>` address before admission, fork only
after a durable creating incarnation exists, pass the composite scope/tag to
Pair, and conservatively retain uncertain post-fork state. Proven-dead
incarnations alone are pruned; unknown and creating occupants fail closed under
bounded policy. Focused admission tests and the full couchcore suite pass
(ARCH-DRY, ARCH-MOCK, ARCH-PURE, ARCH-PURPOSE).

Removed the complete local-policy shadow class: `PolicyTable`, repository
`Mode`, `policy.json`, registry admission, and the public same-path bypass.
A source sweep now prevents any of those decision surfaces from returning.
Capacity refusals render normalized provider evidence; provision-worktree
refusals name #153 and create no path. Removing the bypass also exposed and
fixed acceptance of undeclared CLI flags. `go test ./... -count=1` passes.

The M1 integration boundary adds a live conformance target against a caller-
supplied Ariadne `sdlc`, plus weekly/manual and resolver-change CI. It exercised
bounded→unbounded declaration changes and exact typed refusal using a freshly
built sibling binary. Full `make test`, focused Couch packages, layout/config
validation, real supervisor crash/exec/contender probes, exact physical
namespace inheritance, and `git diff --check` pass. `actionlint` is not
installed locally; the workflow is also exercised command-for-command outside
GitHub Actions.

### 2026-08-26 — M1 boundary review corrections

Boundary review corrected a lifecycle conflation in the preceding entry: a
dead Pair client is not proof that its zellij session is quiescent. M1 now
retains that incarnation and its capacity until #152 supplies
whole-incarnation proof. Opaque allocation also rejects collisions with every
current scoped Pair artifact family and the detached-session binding before
claiming ThreadStore state. Policy churn makes exactly three attempts before
returning typed `PolicyUnstableError`; the public README and live conformance
interface now match provider-owned admission (ARCH-DRY, ARCH-PURPOSE).

Verification passes with `go test ./... -count=1`, `go test -race
./cmd/internal/couchcore -count=1`, the full `make test`, relative-path
`make test-couch-policy-live SDLC_BIN=../ariadne/bin/sdlc`, both zellij layout
dumps, zellij config validation, and `git diff --check`. The first full-suite
rerun exposed a test-only synchronization race in the panel fixture: focus is
published before the panel model, while the helper waited for focus alone. The
helper now waits for both facts, and 50 consecutive couchtty package runs pass.

### 2026-08-26 — M1 atomic address authority

The second boundary review demonstrated a scan/claim interleaving, so the
sequential artifact checker became a durable O_EXCL address claim shared by
Couch and native Pair. A Couch reservation blocks direct Pair until the exact
child establishes it; direct Pair claims before its first artifact; ThreadStore
failure rolls back only its matching marker. The artifact-family inventory now
lives beside `ScopedPaths`, including the claim marker, while malformed session
indexes fail closed. Concurrency, historical adoption, Couch-child adoption,
all constructor families, and zero-write refusal paths have stateful tests
(ARCH-DRY, ARCH-MOCK, ARCH-PURPOSE).

The current M1 `Operation` is explicitly integration because it still contains
effectful `Invoke` closures; a plan-contract test protects every current kind,
and M3 retains the declaration/executor split. README no longer advertises
owner-required stop as a second-process command before #147 routing, and M1's
project close date remains unset until the boundary accepts it.

Verification passes with the full `make test`, `go test ./... -count=1`, race
tests over Couch and launcher, relative-path live provider conformance, 100
immediate SIGUSR2 wrapper restarts, zellij config/layout validation, and `git
diff --check`. The full gate exposed a separate pidfile readiness race: the
wrapper published its PID before registering SIGUSR2. The handler now owns the
signal before the pidfile becomes visible (side-quest commit `7dbd8ac`).

### 2026-08-26 — M1 complete collision-domain rule

The third boundary review swept beyond `ScopedPaths` and found active Go/Lua/UI
families such as `draft-pane`, `image-capture`, parked scrollback, pane memory,
slug, and review sidecars. Collision recognition now uses a structural rule over
the Pair-owned scope directory: the exact tag must have a hyphen boundary on
the left and end/dot/hyphen on the right. This covers every current and future
family without a duplicated prefix inventory while rejecting neighboring opaque
tags. Integration tests precreate both current out-of-`ScopedPaths` families and
an unknown future family and require allocation refusal (ARCH-DRY,
ARCH-PURPOSE).

A reusable project-state contract now walks every project artifact and rejects
closed metadata beside an unchecked milestone row. Restoring M1's premature
`**closed:**` line makes that repository test fail exactly; successful SDLC
acceptance remains the only point that may check the row and record closure.

### 2026-08-25 — session summary

Fresh spec review split verified park into #152 and managed worktree lifecycle
into #153, then drove the remaining durable identity contract to an approved
transactional shape: repository-scoped composite artifact addressing, a locked
revisioned ThreadStore, recoverable pre-exec start handshake, provider
version/digest reconciliation including creating reservations, and exact
per-path/per-agent launch-profile memory. #151 owns the selector presentation;
#152 resumes with the thread's exact latest profile rather than new-thread
defaults (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-21

Split out of a design conversation during `pair#145`'s close. The trigger was
noticing that `couch-ab50125e` and pair's tag are two names for something that
felt like one thing — and the resolution is that they *are* one thing, once the
generated id stops being per-incarnation.

### 2026-08-22 — path demoted from identity to attribute

Folded in the model that supersedes `#145`'s tree-as-key design. Identity is the
durable hex tag; the absolute starting path is an attribute of it; the repo is a
container that supplies git as a facility rather than an identity.

The trigger was noticing that `#145`'s one-agent-per-tree guard forces brain-like
repos onto `--same-tree` for ordinary use, which is an escape hatch on the normal
path. Making the concurrency limit a recorded per-repo number turns that from a
bypass into configuration, and it generalises the three worktree-strategy modes
`#145` stubbed into one question: does work at this path typically conflict.

Two consequences recorded above rather than discovered later: `couch start
<path>` always creates rather than resuming, since a path may name zero or
several spaces; and the limit's granularity and whether it counts live sessions
or all spaces both need settling before implementation.

### 2026-08-22 -- inherited from `#146` M2's smoke: the config picker

`#146` made `couch start` spawn `pair resume <tag>`, which removes the name
prompt and `DecideLaunch`'s session picker. One prompt survives and lands inside
couch's own pty: `runConfigPicker` (`launcher/createflow.go:646`), the
saved-config restore choice -- "use saved params + session / use saved params /
use new params".

It fires only on a COLD start of a tag that has a saved config; once a session
is live, `couch start` attaches and prompts nothing. The operator's call on
2026-08-22 was to leave it, because choosing fresh-vs-resume at a cold start is
a reasonable thing to be asked.

Why it belongs here rather than there: the picker is skipped only when argv
already pins an explicit resume (`extractExplicitResume`, `createlogic.go:57`),
which needs the **agent session id**. couch has no way to know one today. This
issue's model -- the tag as the space's durable identity, with its draft, ledger
and session surviving revival -- is what would let a couch-launched session
resume without asking. Whoever implements that should decide whether a
non-interactive restore is part of it.
