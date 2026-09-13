---
type: project
name: "couch-slots"
goal: "Let one repo host several concurrent couch threads — a small, rigid set of numbered worktree slots per work repo, co-tenancy for brain — without the two-agents-in-one-repo gaps that today's tooling tolerates only because they are rare."
done_when: "The operator runs two threads in pair (primary + slot1) through a full issue lifecycle each — claim, plan, change-code, close, merge — in parallel, with no silent double-claim, no multi-minute stall on issue new, slot1's tree intact and reset after merge, and both threads distinguishable in the status row and switcher; and brain admits a second thread at its primary path."
status: defined
mvp_scope: [ariadne#223, ariadne#222, ariadne#214, pair#197, pair#236]
explicitly_out: [pair#153]
created: 2026-09-11
updated: 2026-09-12
sources: [brain/workshop/pensive/2026-09-11-01-pensive-couch-slots.md, pair/workshop/projects/couch.md, pair/workshop/history/issues/000153-couch-managed-worktree-lifecycle.md]
---

# couch-slots

A repo gets a few numbered desks. `slot1`, `slot2` — rigid, reusable, always
there — and a thread sits at one until it is archived. **The headline
omission: nothing is ever cleaned up automatically.** No garbage collection, no
reprovisioning a vanished tree, no free-form worktrees inside couch, no manual
slot names. That is the whole of `pair#153`, punted on 2026-09-02, and it stays
punted. Also out: a mechanism for binding a slot to another repo's slot
(`pair:1 → ariadne:2`) beyond what a symlink already does, and any per-tree
override of `construct/deps`.

Most of this project is not couch. It is the surrounding tooling that assumed
one tree per repo, made explicit and fixed before the second tree exists.

## PRD

**The problem is a rule that no longer matches how the operator works.** couch
keeps one thread per working path (`couchcore/couch.go:397`, "ENFORCED for
now… until per-repo policy is modelled", pair#181). Meanwhile three live
sessions share `~/workspace/parley.nvim` outside couch, six brain threads were
what prompted the rule, and the operator hand-built the shape this project
formalizes: a worktree with `worktree/pair/ariadne -> ~/workspace/ariadne`
beside it. Today's per-issue worktrees (`sdlc change-code --worktree=yes`) are
the wrong shape for couch: `sdlc merge` removes the tree, and a parked thread
whose path is gone cannot resume.

**A slot is a desk, not a branch.** `~/workspace/worktree/slot1/<repo>`, on a
branch named `slot1`. The numbering is the point — a small fixed set, two or
three per repo in practice, so the operator can hold "pair :1 :2" in their head
the way they hold five repos today. Rules:

- **Occupancy.** A slot holds exactly one thread — live, detached, or parked.
  A parked slot is *resumed*, never skipped: starting in a repo whose slot1
  holds a parked thread sits back down at slot1. A new thread in an occupied
  slot requires archiving the old one first. This is couch's existing
  resume-before-spawn rule (`startup.go:148`) applied per slot.
- **Capacity.** The per-repo count is policy, already declared in
  `.sdlc/fleet.json` (`capacity: bounded, limit: N`): pair's `limit: 1` today
  becomes primary + two slots. brain's `capacity: unbounded` means co-tenancy
  at the primary path with no worktree — the declaration exists; nothing reads
  it (ariadne#214).
- **Primary.** Inferred, never hardcoded: the thread's cwd → its repo root →
  the parent is the workspace, where every primary lives. Only the primary is
  ever on `main`; git enforces that.
- **Resting branch.** `slot1` points at `origin/main` between issues. Work is
  committed on issue branches cut from it, exactly as from `main`. After
  merge, the slot branch resets to the new `origin/main` and the tree stays.
  `sdlc push` from a slot stays refused — that refusal is the safety property.
- **Peers.** Beside a slot, each peer repo is either a real worktree on the
  same slot number or a symlink to the primary; `ls -l worktree/slot1/` is the
  map. The parent directory is the binding: the 28 lowered symlinks are
  literal `../ariadne/…`, and weave resolves `substrate ../ariadne` against
  the parent (`layergraph/walk.go:110-117`, physicalizing the parent and
  keeping the peer's logical name — measured 2026-09-11: link text stays
  stable through a symlink). Default is symlink; promote on demand.
- **Ownership over freshness.** Three trees means three copies of
  `workshop/issues/`; the trunk is truth. Only `claim`, `issue new`, and
  `start-plan`'s contention check need trunk-fresh data. Once a claim is
  published, the owner's copy is authoritative for that issue and nobody else
  writes it.
- **Naming.** Status row `pair :1 :2`; `:n` is the nth secondary thread of the
  repo, backed by a slot for work repos and by a co-tenant for brain. Same
  grouping in the switcher. Session name `📁pair-1-couch`, inside the socket
  budget. No manual rename.

**Why the surrounding work comes first.** Two gaps exist today and are
tolerable only because two agents in one repo is rare: a stale double-claim
overwrites silently (`claim.go:186-193` reads only the local file;
ariadne#222), and every mutating `sdlc` verb in every worktree of a repo waits
on one `.git/sdlc.lock` that `change-code` holds across a multi-minute judge
(`repolock.go:92-118`, `changecode.go:394,542`). With slots those are daily.
Shipping slots before those fixes would make the feature feel broken while
every slot mechanism works. Separately, a family of `filepath.Base(repoTop)`
and `filepath.Dir(repoTop)` calls misidentify the repo and the fleet root from
any worktree today (`close.go:460,630`, `--brain-dir ../brain`), and
change-code's issue publish is `branch == "main"` — both already bite the
existing issue-worktree flow.

**What this is not.** couch does all provisioning; `sdlc` only recognizes the
convention (branch `slotN`, path under `worktree/slotN/`) and changes its
merge ending, its off-main publish, and its refusal of nested worktrees.
Outside couch, `sdlc` is unchanged.

## Estimate

Not yet derived. Required before `defined` → `committed`, with `deadline` and
`planned_finish`. Per-issue `estimate_hours` derive after each plan clears
change-code's plan-quality gate; the project figure assembles from those.

Sizing basis when it is done: the ariadne items are each a bounded change to
one verb with a known shape (`close` already has the release-around-judge
pattern to copy; `fleet.NormalizeVantage` already derives the primary
correctly; `claim.go:156` already publishes via trunk CAS off main). The couch
items are an offer at an existing refusal site, a label derivation pair#197
already owns, and re-reading a policy file with an existing schema. The unknown
is not code size but how many places assume one tree per repo that no grep
finds — which is why the sequence leaves the core flow working after every
task.

## Breakdown

**Do it in one dedicated block, and do the spine first.** Tasks 2–7 modify
`claim`, `change-code`, `close`, and `merge` — the verbs every session uses.
They are the destabilizing part and the part with gotchas nobody has grepped
for. Run them with no other spine work in flight, one at a time, each merged
and used for a day before the next. Everything after task 7 is additive:
couch's refusal path changes only when policy says so, merge's ending changes
only on a `slotN` branch.

**Every task is a stopping point.** After task 1 the derivatives stop churning
on `weave compile`. After task 3, two agents in one repo stop silently
overwriting each other and stop stalling behind judges — worth having even if
slots never ship. After task 7 today's issue worktrees work correctly end to
end. After task 9 the two measurements say whether the couch half is worth it.

**Two measurements gate the couch half** (tasks 8–9). If slot start is more
than a few seconds, the offer must say what it is about to do; if parked
threads mostly leave dirty trees, the occupancy rule needs a "what's dirty"
reason in the row before it needs anything else. Neither outcome kills the
design; both change what ships first.

**Abort criterion.** If task 3 or task 5 turns up a second undocumented
one-tree-per-repo assumption that is not a local fix, stop, bank tasks 1–3,
and re-plan. That is the "shape unknown" signal pair#170 measured at 0.3×
estimate ratios, and the answer is to stop deciding while building.

Issues for tasks without a ref are filed when the project moves to
`committed`, one per task; a task line then gains its ref.

- [ ] untrack weave-lowered substrate symlinks [ariadne#223]
- [ ] claim reads the trunk before it flips: no silent double-claim [ariadne#222]
- [ ] change-code releases the repo lock around the judge; judge hold time measured
- [ ] repo identity and fleet root from the vantage, not the basename; `--brain-dir` follows
- [ ] issue publish off main: change-code sync, peerwrite, start-plan clean-check
- [ ] slot-aware merge: keep the tree, reset `slotN` to origin/main, refuse on unpublished commits
- [ ] refuse `--worktree=yes` from a non-primary tree; `sdlc state` names the slot
- [ ] measure slot start: worktree add + peer symlink + weave compile
- [ ] survey parked-thread trees: how many are dirty today, and with what
- [ ] one label derivation for status row and switcher: `pair :1 :2` [pair#197]
- [ ] one thread order for switcher and status row: repo load order, groups, Alt+Up/Down, projection [pair#236]
- [ ] couch reads `.sdlc/fleet.json` again: bounded / unbounded / offer-slot [ariadne#214]
- [ ] the slot offer at the refusal site: resume the slot's parked thread first, else provision the lowest free slot (v0 prints the commands)
- [ ] AGENTS.md: the ownership rule, and how an agent learns which tree it is in
- [ ] brain co-tenancy: per-thread naming, `Unregister` per thread, unreadable-block scoped to the thread
- [ ] measure `.git/index.lock` contention: two agents committing under `nous serve`

<a id="ariadne-223"></a>
### ariadne#223 — untrack weave-lowered substrate symlinks

**status:** filed 2026-09-11, open

Independent of everything else; safe anytime. 28 manifest-lowered symlinks are
tracked in each of five derivatives since `#38 M4` converted vendored copies in
place. weave prunes and refreshes them as its output; git sees churn. One
dependency: `bootstrap.sh`'s `exec make bootstrap` handoff resolves through the
tracked `Makefile` link. Not a prerequisite for slots — a symlinked peer keeps
the link text stable — but it is what stops `weave compile` from being a
diff-producing operation in a slot.

<a id="ariadne-222"></a>
### ariadne#222 — lost update on the same issue

**status:** open, undesigned (three sketched designs in the issue)

The project needs only the narrow case: `claim` must read the trunk copy of
the issue before flipping `open → working`, and refuse if it is not open there.
The full lost-update design (publish provenance refs, blob-in-history) is not
required for MVP and should not be pulled in. Record the narrowing in the
issue's Log when claiming it.

<a id="pair-197"></a>
### pair#197 — actor label: repo-qualified, one derivation for status bar and switcher

**status:** open

Slots make this mandatory: the status row (`couchtty/reserve.go:88-128`)
paints raw `Worktree.Repo()` with no disambiguation pass, so two same-repo
threads render identically, and `Worktree.Repo()` is `filepath.Base(git
toplevel)` — which for a slot is the slot's own directory, not the repo. The
derivation this issue owns is where `pair :1 :2` lives.

## Log

### 2026-09-11 — created

Distilled from the brain advisor session and its pensive
(`brain/workshop/pensive/2026-09-11-01-pensive-couch-slots.md`), after a
brainstorm that changed the model three times: slot-first layout for the
parent-directory binding; primary inferred rather than hardcoded; occupancy as
resume-not-skip on the operator's correction that slots are rigid by design.
Measured on the way: 1042 orphaned test processes holding 11 GB (unrelated,
parley#220); the repaint nudge's 87 ms window (unrelated, pair probe); that
`weave compile` beside a symlinked peer does not rewrite lowered links (this
project); and that the 28 lowered links are tracked (ariadne#223).

Not committed to a timeline. The operator wants a dedicated block for tasks
2–7 because they touch the spine and will surface gotchas.

[ariadne#223]: #ariadne-223
[ariadne#222]: #ariadne-222
[pair#197]: #pair-197
