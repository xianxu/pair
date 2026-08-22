---
id: 000149
status: open
deps: []
github_issue:
created: 2026-08-21
updated: 2026-08-21
estimate_hours:
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

## Done when

- A couch-launched session gets a generated durable tag, and a revival of the
  same space reuses it — verified by the draft and ledger surviving a restart.
- An operator can name a space after the fact and rename it, with no file moved
  and no state lost.
- pair's picker shows the human name where one exists and the hex string where
  none does, and resolves a name to its space with couch not running.
- `pair claude` standalone still asks for a tag exactly as it does today.
- `couch list` no longer leads with the system id.
- Two spaces in one tree (via `--same-tree`) keep separate drafts and ledgers.

## Plan

- [ ] Make the tag opaque and durable; `Spawn` resolves rather than mints.
- [ ] Promote the name↔space mapping in `launcher/session_index.go` to a rename
      layer; pair resolves standalone.
- [ ] Picker shows names, falling back to the hex string.
- [ ] couch reads pair's index instead of keeping its own naming table.
- [ ] Drop the system id from `couch list`'s common output.
- [ ] Reconcile `#135`'s tag-as-work-identity with space.

## Log

### 2026-08-21

Split out of a design conversation during `pair#145`'s close. The trigger was
noticing that `couch-ab50125e` and pair's tag are two names for something that
felt like one thing — and the resolution is that they *are* one thing, once the
generated id stops being per-incarnation.
