---
id: 000149
status: working
deps: []
github_issue:
created: 2026-08-21
updated: 2026-08-25
estimate_hours:
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

## Done when

- A couch-launched work thread gets a generated durable tag, and a revival of
  that thread reuses it — verified by the draft and ledger surviving a restart.
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
  in one competition; and worktree policy gives each generated worktree a
  singleton live slot.
- `couch start <path>` twice creates two work threads where the limit allows it,
  and resuming a specific one is an explicit act.
- Each durable work thread can be parked and resumed under the same opaque tag;
  parking frees the live concurrency slot without deleting its history.
- Thread inventory distinguishes multiple threads at one path and exposes
  live/parked state plus persisted observed last-active time for terminal
  presentation.

## Plan

- [ ] Make the tag opaque and durable; `Spawn` resolves rather than mints.
- [ ] Promote the name↔work-thread mapping in `launcher/session_index.go` to a rename
      layer; pair resolves standalone.
- [ ] Picker shows names, falling back to the hex string.
- [ ] couch reads pair's index instead of keeping its own naming table.
- [ ] Drop the system id from `couch list`'s common output.
- [ ] Reconcile `#135`'s tag-as-work-identity with work thread.
- [ ] Implement repository-policy conflict keys and live-slot accounting.
- [ ] Make park/resume and persisted last-active explicit lifecycle transitions.

## Log

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
