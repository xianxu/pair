---
collaboration-mode: discussion
---

# Couch teams: hubs as team folders Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution
> approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or
> superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** a Couch hub becomes a *team*: a folder `teams/<team>/` with a committed `.hub` in an operations checkout,
addressed by the team's name, and run through the same lifecycle as a slot. Several teams can share one checkout.

**Architecture:** today a hub is a checkout identified by its worktree root and its `repo:N` slot. This plan makes the
hub's identity a *name* backed by either a team folder (new) or a checkout (today's form, kept for plain repo paths).
Ownership, addressing, registration, the switcher and session-sync key on that name. The team's agent is an ordinary
thread whose scope is the team folder, carrying an additive `team` field. Its lifecycle comes from the
`LifecycleAddress` team arm and proven teardown of pair#695 (split from #684).

**Tech Stack:** Go (`cmd/internal/couchcore`, `couchcmd`, `couchmessage`, `couchtty`, `sessionsync`, `wrapcmd`).
The attention finder lives in parley.nvim; this plan gives it data and files the parley change.

**Umbrella design:** `ops:teams/devinfra/projects/introduce-team/design.md`, sections 1–3. Section 2 (the folder
contract) is shared with ariadne#456. This plan reads it as written, except for one clarification sent to the hub
(§2 below: what an "unknown line" in `.hub` is).

---

## 1. Today, measured (2026-10-18)

From a read of the code (file refs under `cmd/internal/`):

- **Startup** (`couchcmd/cli.go:110`, `hub_cli.go:11`, `hubs.go:29`): `couch p1 p2 …` already starts up to 128 hubs.
  Each path must be a checkout's worktree root (`sdlc workspace --json` gives `Slot != nil && WorktreeRoot == root`,
  `hubs.go:51`). `paths[0]` starts through `start` and attaches; the rest queue `start-hub`. `.hub` is read from
  `<worktree root>/.hub` **only when there are two or more paths** (`hubs.go:54`).
- **Identity:** `HubDefinition{Root, Workspace, Slot HubSlot, Claims}`, keyed by `Root`. `NewHubOwnership` rejects two
  hubs with the same `HubSlot`, so two teams in one checkout cannot exist today.
- **Ownership** (`hub_ownership.go:82` `Resolve`): the hub's own slot first; then a runtime override; then claims
  (exact `repo:N` beats repo-wide; the first hub on the command line wins a tie, and later ones become `Conflicts`);
  unclaimed slots go to `Main()` (the first hub).
- **Addressing** (`couchmessage/routing.go`, `message_service.go:967`): recipients are family names (repo directory or
  alias, unique prefix allowed), exact `repo:N`, or the reserved `hub`. `hub` resolves to the caller's owning hub
  binding, found by `WorktreeRoot` (`wait_notice.go:300,313`).
- **Registration** (`wrapcmd/peer_runtime.go:202`, `couchmessage/model.go:38`): a wrapper registers
  `Binding{Slot: "repo:N", …}`. Admission requires the slot address and worktree root to match (`message_service.go:839`).
  An agent in `ops/teams/echo` would register as `ops:0`, the same as every other team in that checkout.
- **The hub's thread:** a `:0` hub is an ordinary thread, scope `sha256(cleanRoot)[:16]` (`launcher/scope.go:23`),
  in the global store. There is at most one primary per repo scope (`startup.go:38`).
- **Switcher** (`couchtty/thread_presentation.go:174,273`, `menu_render.go:508`): with more than one hub, the rows are
  sectioned per hub in command-line order. Each section is led by the hub row, labelled with its ordinary label.
- **Session-sync** (`sessionsync/ledger.go:62`, `file.go:78`): `couch-owner:` must be a canonical `repo:N`.
- **Attention finder:** in parley.nvim (`lua/parley/couch.lua:103`, `attention_model.lua:69`). Its scope is the current
  slot or all slots.
- **`.hub` files on this machine:** none (`find ~/workspace -maxdepth 6 -name .hub` is empty). The ops checkout has
  `teams/devinfra/` and `teams/ariadne/` without one.

## 2. Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `TeamIdentity` | `couchcore/team.go` | new |
| `HubName` | `couchcore/team.go` | new |
| `HubDefinition` | `couchcore/hub_ownership.go` | modified |
| `HubOwnership` | `couchcore/hub_ownership.go` | modified |
| `HubClaimDocument` | `couchcore/hub_claim.go` | modified |
| `ResolveTeamNames`, `TeamNamesLine` | `couchcore/namespace.go` | new |
| `WorkspaceReference` (`team:` references) | `couchcore/workspaceref.go` | modified |
| `LifecycleAddress` (team arm) | `couchcore/lifecycleaddress.go` (from #695 M1) | modified |
| `couchmessage.Binding` | `couchmessage/model.go` | modified |
| `PresentThreads` sections | `couchtty/thread_presentation.go` | modified |
| `sessionsync.ValidOwner` | `sessionsync/ledger.go` | new (replaces `ValidSlot` at the owner seam) |

- **TeamIdentity** — `{CheckoutRoot, TeamDir, Name string}`. `CheckoutRoot` is the physical worktree root of the
  operations checkout. `TeamDir` is clean and relative (`teams/devinfra`). `Name` is `filepath.Base(TeamDir)`, derived
  and never stored separately, so it cannot drift (design §2). It is the *default* name only; the name a team runs
  under is resolved per startup (`ResolveTeamNames`, Revision 2). `Validate()` refuses an empty or unclean
  `TeamDir`, `..` segments and an absolute path. It never refuses a name.
  - **Relationships:** 1:1 with a hub of kind team. N:1 with a checkout (several teams per checkout).
  - **DRY rationale:** #695's `LifecycleAddress` team arm *is* this type (designed under #684, moved to #695). That arm was written as `HubIdentity
    {OpsRoot, TeamDir, Name}`; it is renamed `TeamIdentity` with kind `team`, matching the umbrella design's word. A
    Revision on #684's plan records the rename (the #684 code hasn't started, so it's free).
- **HubName** — the address a hub is reached by. For a team, `team:<name>` as resolved by `ResolveTeamNames`. For a checkout hub (today's form), its
  existing label (`repo` for `:0`, `repo:N` otherwise). Today `HubOwnership` keys on `Root` (a path); it keys on
  `HubName` instead, and `Root` stays as the hub's working directory.
- **HubDefinition** — becomes `{Name HubName; Kind HubKind /*team|checkout*/; Root, Workspace string; Team
  TeamIdentity /*kind team*/; Slot *HubSlot /*kind checkout only*/; Claims HubClaimDocument}`. `HubSlot` stays: it is
  the identity of a *claimed* slot (`{PrimaryRoot, N}`), which teams still need. It stops being the hub's own identity.
  `NewHubOwnership` refuses duplicate names and duplicate roots. That is an internal invariant: names arrive already
  resolved and unique, so the user never sees this refusal. It no longer refuses two hubs in one checkout when
  both are teams.
- **HubOwnership** — the same rules, keyed by name. "The hub's own slot first" applies only to checkout hubs (a team
  owns no slot). `Main()` becomes `Steward()`: the first hub on the command line (design §6.2), returning a name.
  `Apply(HubMove)` takes names.
- **HubClaimDocument** — design §2 says unknown lines are preserved (ARCH-EVOLVE). Today any line that is neither a
  comment nor a claim refuses the whole file. A bare word *is* a claim (`echo`), so "unknown" needs a shape no claim
  can have. The clarification sent to the hub: **a directive line is `<key>=<value>` or `<key> = <value>`, where `key`
  matches `[a-z][a-z0-9-]*`.** The claim grammar forbids `=` and whitespace inside a claim, so the two can't collide.
  Directive lines are kept byte for byte through `PlanHubClaimEdits` and ignored, and each distinct unknown key gets
  one startup note. Malformed claim lines still refuse, because a typo in a claim must not silently re-route a slot.
  `Claims()` is unchanged.
- **Team addresses** (Revision 3, replacing Revision 2's forms). A team's canonical address is `team:<name>`
  (`team:devinfra`), in the same style as `pair:2` for a slot and `pair#444` for an issue. A bare name is always a
  repo family, `echo:2` is a slot, and `team:echo` is the team. Bare names never resolve to teams, so there is no
  team-versus-repo or team-versus-alias clash, and `hub` stays the caller's own team. The word `team` is a reserved
  prefix only before a non-numeric name: a repo called `team` still has slots `team:N`.
- **ResolveTeamNames** — never refuses a startup over a name. It is pure:
  `ResolveTeamNames(teams []TeamStart) []ResolvedTeam`, where `TeamStart{Identity TeamIdentity; Repo string /*the
  checkout's repo name*/; Address string /*the checkout's repo:N*/; Alias string}` and `ResolvedTeam{Name, Address,
  Reason string}`. The rules are:
  1. The name is the startup alias (`teams/devinfra:devinfra2`), else the folder name. A folder name outside the
     name grammar is normalized with the existing `launcher.NormalizeDisplayComponent`, and an all-digit or empty
     result becomes `team-<result>`. Team names are never numeric, so a `<path>:<name>` argument can't be read as
     `repo:N`.
  2. The only clash left is two teams with one name. The first in command-line order keeps `team:<name>`, the same
     tie rule as ownership today. A later one is qualified inside the team namespace as `team:<repo>/<name>`
     (`team:ops2/devinfra`), and `Reason` says why.
  3. If that still clashes (two checkouts of one repo), the checkout's address qualifies it: `team:ops:2/devinfra`.
     That is unique, because one checkout can't hold two folders at one path.
  4. The same team folder given twice starts once, with a note. It is not refused.

  `TeamNamesLine([]ResolvedTeam) string` prints the startup line, for example: `couch teams: team:devinfra
  (steward, ops/teams/devinfra) · team:devinfra2 (ops2/teams/devinfra, alias) · team:ops2/finance
  (ops2/teams/finance; team:finance is taken)`. `WorkspaceReference` parses `team:<rest>`. If `rest` is all
  digits, it is slot N of a repo named `team`. Otherwise it is a team reference, `[<repo>[:N]/]<name>`. Every form
  round-trips. `ResolveRecipient` resolves a team reference exactly, with no prefix matching across kinds. The
  address lives on `HubDefinition.Name`. It is resolved per startup and never persisted: the team's record and lease
  key on `CheckoutRoot`/`TeamDir` (#695's `Key()`), so renaming a team never orphans its thread. A session-sync
  `couch-owner:` that names a team not running under that address is handled like any unreachable owner today. The
  report fails visibly and the file is untouched.
- **LifecycleAddress team arm** — `Key()` for a team is `team/<scope(CheckoutRoot)>/<TeamDir>`, distinct per team
  and per checkout (#695 `TestLifecycleAddressKeyDistinctPerTeam`).
- **couchmessage.Binding** — gains `Team string` (`json:"team,omitempty"`). `Validate`: exactly one of `Slot` and
  `Team`. A team binding needs everything else a slot binding needs.
- **PresentThreads sections** — teams are a section above slots, named by team. Each team's slots are grouped under
  it (design §3). A checkout hub keeps today's section and label.
- **ValidOwner** — `repo:N` (today) or a team name (`HubName` grammar). The ledger path escapes either.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `prepareHubs` (was `prepareExplicitHubs`) | `couchcmd/hubs.go` | modified | `sdlc workspace --json`, filesystem |
| `ReadHubClaimFile` | `couchcmd/hub_claim_files.go` | modified | filesystem (`.hub`, O_NOFOLLOW) |
| team thread start | `couchcore/startup.go`, `startargs.go` | modified | launcher, thread store |
| wrapper team registration | `wrapcmd/peer_runtime.go` | modified | Couch message socket |
| team admission and `hub` resolution | `couchcmd/message_service.go`, `wait_notice.go` | modified | live bindings |
| `couch --actors --json` team fields | `couchcmd` actors output | modified | stdout JSON |
| session-sync owner | `sessionsync/file.go`, `ledger.go`, `couchcmd/session_sync.go` | modified | ledger files |

- **prepareHubs** — classifies each startup path. A directory containing a regular `.hub` is a **team**. Any other
  directory is a **checkout hub** and must be a worktree root, as today. A team is resolved with the same `sdlc
  workspace --json` call on its folder, which gives `WorktreeRoot` (→ `CheckoutRoot`) and `FleetRoot` (→ `Workspace`,
  for claim resolution). The folder must be physical, under `WorktreeRoot`, and not the root itself (`TeamDir` must
  not be `.`). Couch doesn't check that `.hub` is committed: it reads only `.hub`, and commits are tl-lead's (design
  §2). The plan says so instead of adding a git call. A team's `.hub` is always read. A checkout hub's `.hub` is read
  as today (only with two or more paths), so plain `couch` and a single repo path behave exactly as today.
  - **Injected into:** `HubOwnership` construction. Tests use the existing workspace-resolver fake.
- **team thread start** — `StartArgs` gains `Team *TeamIdentity`. A team start resolves an ordinary target whose
  scope is `ResolveRepoScope(<CheckoutRoot>/<TeamDir>)`, so each team folder has its own scope and the
  one-primary-per-scope rule (pair#363) holds per team. The working directory is the team folder. The launch env adds
  `COUCH_TEAM=<name>` beside `COUCH_THREAD_SCOPE`/`TAG`. The record carries an additive `team` object
  (`{checkout_root, team_dir}`). An older build reads it as an ordinary thread at that scope, which is what it already
  is, so no version bump is needed (#695 §2a: tolerant and preserving decoders).
- **wrapper team registration** — with `COUCH_TEAM` set, the wrapper registers `Binding{Team: name, …}` instead of
  refusing "peer delivery needs an addressable Couch slot". Admission accepts a team binding only when the name is a
  configured team, the wrapper's physical cwd is that team's folder, and its scope and tag match that team's record.
- **`hub` resolution** — `hubBindingFor(from)` resolves the caller's owner *name*, then that hub's live binding by
  name (a team binding, or a checkout hub's binding by worktree root, as today).
- **actors JSON** — additive fields: `team` on a team-hub row, and `owner_team` on every slot row. These are the data
  the attention finder needs.

## 3. Behaviour

### Startup (`couch [path …]`)

| Invocation | Meaning |
|---|---|
| `couch` (no args) | today: the start folder is the one hub (legacy `authority.hub`). Unchanged, including in ops. |
| `couch <repo>` | today: one checkout hub, `.hub` not read. Unchanged. |
| `couch teams/devinfra` | one team hub, `devinfra`; its `.hub` is read. |
| `couch teams/devinfra teams/echo ../ops2/teams/finance` | three team hubs; `devinfra` is the steward. |
| `couch teams/devinfra ~/workspace/foo` | a team and a checkout hub may mix; order decides the steward. |
| `couch teams/devinfra ../ops2/teams/devinfra:devinfra2` | the second team is named `devinfra2` (a startup alias). |
| `couch teams/devinfra ../ops2/teams/devinfra` | the second is auto-qualified `team:ops2/devinfra`; the startup line says so. |

**A name never refuses a startup** (Revision 2): names resolve by `ResolveTeamNames`, and one line prints every
team's resolved name before launch. Startup still refuses, before launching anything, when a team folder is
invalid (missing, not physical, the checkout root itself) or a `.hub` fails to parse, because those are not names.
Each refusal names the path and the fix. A path argument is first taken whole. If no such directory exists and it
ends in `:<name>` with a non-numeric name, the suffix is the alias. The first hub attaches in the
foreground, and the rest queue `start-hub` (today's mechanism, now carrying `Team`).

### Addressing

- `couch --send-to team:devinfra` delivers to the team's live binding. `couch --send-to echo` is the repo family,
  as today, even when a team `echo` runs.
- `couch --send-to hub` from a slot delivers to the team whose `.hub` claims it. Exact slot beats repo, the first team
  breaks ties, and unclaimed goes to the steward. These are today's `Resolve` rules, by address.
- `couch --send-to hub` from a team refuses with "you are the hub" (as today). A team reaches another team by its
  `team:` address.
- `--move-slot echo:2 --to-hub team:echo [--persist]` accepts a team address. `--persist` writes the team's `.hub`,
  preserving directive lines.

### Switcher and tab bar

Teams first, one section each, headed by the team's row. Under each team come the slots it owns, indented as
today's hub sections are. Slots owned by a checkout hub follow in that hub's section. The `hub <label>` suffix on
other rows shows the team's display name.

**A team displays as its name in capitals, with no `:N`** (operator, Revision 1). A team is like a repo without
worktrees: a singleton. The tab bar groups by repo family (`tools :1 :2`), and a team is a group of its own
(`ECHO`). **When a team and a repo with the same name are adjacent in the bar, they merge into one group:** the
team, then that repo's slots (`ECHO :0 :1`). Otherwise they stay separate (`ECHO tools :1 :2 echo :1 :2`). The
merge is a pure function over the ordered bar items:

```go
// couchtty/bargroups.go (pure)
type BarItem struct { Team string /* set for a team */; Family string; Slot int /* -1 for a team */ }
type BarGroup struct { Label string; Items []BarItem } // Label "ECHO", "tools", or "ECHO" for a merged group
func GroupBar(items []BarItem) []BarGroup
func TeamDisplayName(name string) string // strings.ToUpper; the only place a team's capitals are made
```

`GroupBar` groups consecutive items of one family, and merges a team group with the immediately following family
group when the family's name equals the team's name. Adjacency is in the bar's existing order (sections first),
so a team's own same-named repo, owned by the team, merges naturally. The switcher rows use `TeamDisplayName` for
team rows. Keys, addresses and session-sync stay lower case: capitals are display only.

### Session-sync and attention

- `couch-owner: team:devinfra` is valid, and a write report goes to that team. `--owner` accepts a team address.
- Couch publishes `team` and `owner_team` in `--actors --json`. The parley.nvim change is filed as a parley issue
  in M2: the attention finder **partitions by team or by slot** (a team's partition holds the team and the slots
  whose `owner_team` is that team), with the team shown in capitals as in Couch. It lands there, not here
  (layer direction: parley reads Couch's output).

### Lifecycle

A team's agent goes through the same mechanism as a slot (#695's primitives, then #684's Converge): `LifecycleAddress{Kind: team, Team}`, the same lease
(keyed by `Key()`), the same transcript binding on its record, the same failure records (#683), and the same three
outcomes. A team never holds an sdlc claim. Nothing in this plan calls sdlc for a team beyond `sdlc workspace --json`.

## 4. Migration of today's hubs

- **Types:** `HubDefinition` and `HubOwnership` change in place (by name). Every caller listed in §1's digest moves to
  names: the `owner`/`isHub` users in `message_service.go`, `wait_notice.go`, `supervision.go`, `hub_guards.go`,
  `hub_move.go`, `lifecycle_notices.go`, `resources_alert.go`, `notify_service.go`, `agent_drive_service.go`,
  `agent_switch_service.go`, `menu.go`, `menu_reattach.go`, `menu_render.go`, `console_presentation.go` and
  `couchmessage/protocol.go`. A compile-time sweep: `Root`-keyed lookups become `Name`-keyed, and the type change
  makes a missed caller a build error (ARCH-PURPOSE: every consumer derives from the name).
- **Files:** no `.hub` exists on this machine today. A checkout hub's `<worktree root>/.hub` keeps working unchanged.
  Moving an ops checkout hub's claims into a team is a copy of its lines into `teams/<team>/.hub`. That is
  `tl-lead team new`'s job (ariadne#456), so this plan adds no converter.
- **Runtime:** the legacy single-hub mode (`authority.hub`) stays for bare `couch`. Underneath it becomes a one-hub
  `HubOwnership` with a checkout hub, so there's one code path (ARCH-DRY), not two. The guard is
  `TestBareCouchUnchanged`.

## 5. Principles

- **ARCH-DRY:** the team identity is #695's `LifecycleAddress` arm. Ownership rules are today's, re-keyed. The start
  path is today's `start-hub`. No second hub mechanism: the checkout hub and the team are kinds of one definition.
- **ARCH-PURE:** `TeamIdentity`, `HubOwnership`, `ResolveTeamNames`/`TeamNamesLine`, the `.hub` parser and the section builder are pure
  and unit-tested. The IO is `prepareHubs`, the wrapper registration and admission.
- **ARCH-PURPOSE:** every listed consumer moves to names in this issue (§4), and so do session-sync and actors. The
  only thing left to another repo is the parley scope, which reads Couch's published data.
- **ARCH-MOCK:** the external surfaces are `sdlc workspace --json` (existing resolver fake) and the filesystem (temp
  dirs). There's no new external binary.
- **ARCH-CONSTRAINTS:** startup path, at most 128 hubs (`MaxHubSlots`). It costs one `sdlc workspace` call and one
  `.hub` read per hub, the same as today. `.hub` is at most 64 KiB and 1024 lines (unchanged). Admission adds one
  name lookup.
- **ARCH-SECURE:**
  - **`.hub`:** it is untrusted input even though it's committed. It is parsed into a typed document at the boundary,
    opened with O_NOFOLLOW under a physical folder (unchanged), and refuses when malformed. Directive lines are kept
    and never interpreted.
  - **`COUCH_TEAM`:** it is an env claim. Admission checks it against the configured team's folder, scope and tag, so
    an agent in another folder can't register as a team. The trust level is the same as today's `repo:N` slot
    admission: same user, local socket.
  - **No secrets** are involved.
- **ARCH-ORDER:**
  - **Ownership:** `HubOwnership` stays an immutable value with `Apply` as its only transition (unchanged). The new
    durable state is the team record field, which is written once at creation and never changes.
  - **Events the caller can't block:**
    - A team folder is deleted or renamed while Couch runs. Its `start-hub` (or relaunch) fails, and the failure is
      recorded (#683). Its row shows the reason and is never re-pointed to another folder.
    - `.hub` is edited while Couch runs. It is re-read on `--move-slot --persist` (compare-and-swap on the before
      bytes, as today) and otherwise at the next start.
    - Two wrappers claim the same team name. The second refuses, the same as a duplicate slot binding today.
  - The one most likely to be mishandled: **a team agent registering as its checkout's `ops:0`**. The wrapper must
    prefer `COUCH_TEAM` over the sdlc slot whenever it is set. `TestTeamWrapperNeverRegistersAsCheckoutSlot` guards
    this.
- **ARCH-FUNERAL:** new durable items:
  - A team thread record: the same family and removal as any thread (archive and GC).
  - The `team` field: it lives and dies with its record.
  - The lease file under `Key()`: #695 owns it, and it goes with the record's archive.
  - A team's session-sync ledger file: the existing per-owner family, one per team, with the same retention.
- **ARCH-EVOLVE:**
  - `.hub` directive lines are preserved (above).
  - Record `team`: an additive object, preserved by #695 §2a's decoders.
  - `Binding.Team`: `omitempty`, so a slot binding's wire bytes are unchanged. Only team bindings carry the field, and
    only a Couch that knows teams launches a team.
  - actors JSON: additive fields.
  - Tests: `TestHubDirectiveLinesSurviveMove` and `TestTeamRecordFieldSurvivesOlderRewrite` (the latter via #695's
    unknown-field preservation).

## 6. Milestones

This plan depends on pair#695 being on main (the `LifecycleAddress` type, the binding, the lease and teardown) for M3 only.
M1 and M2 need none of it, so they can start from main as soon as this plan is approved. That is the parallel window
the umbrella design asks for. Branch rule (#272): no stacking. M3 starts after #695 has landed on main (the hub's
split, §7.1).

### M1 — The team model (pure)

**Files:**
- Create: `couchcore/team.go` and `team_test.go`, plus `couchcore/namespace.go` and `namespace_test.go`.
- Modify: `couchcore/hub_ownership.go`, `hub_claim.go` and their tests.

- [ ] Test (red) `TestTeamIdentityValidate`: clean relative folder, name from base, refuses `.`, `..` and absolute paths;
  never refuses a name.
- [ ] Test (red) `TestHubClaimDirectiveLinesPreserved` and `TestHubDirectiveLinesSurviveMove`: a `steward=true` line
  survives `PlanHubClaimEdits` byte for byte; `echo :2` still refuses.
- [ ] Test (red) `TestHubOwnershipByName`: two teams in one checkout are both accepted; duplicate names refuse;
  `Resolve` follows today's rules returning names; a team owns no slot; `Steward()` is the first.
- [ ] Test (red), Revision 3: `TestTeamStartupAliasNamesTeam` (`teams/devinfra:devinfra2` → `team:devinfra2`),
  `TestTeamClashAutoQualified` (two `devinfra` → `team:devinfra`, `team:ops2/devinfra`, with the reason),
  `TestAliasResolvesClash` (the alias removes the qualification), `TestQualifiedClashFallsBackToCheckoutAddress`
  (`team:ops:2/devinfra`), `TestNumericSuffixIsNotAlias`, and `TestTeamNamesLine` (the startup line: every team,
  the steward marked, qualifications explained).
- [ ] Test (red) `TestWorkspaceReferenceParsesTeam`: `team:echo` is a team, `team:3` is slot 3 of repo `team`,
  `team:ops2/devinfra` and `team:ops:2/devinfra` are qualified teams, `echo` and `echo:2` are unchanged; all
  round-trip.
- [ ] Test (red) `TestBareNameNeverResolvesToTeam`: with team `echo` and repo `echo` both live, `echo` reaches the
  repo family and `team:echo` the team.
- [ ] Implement until green; the existing `hub_ownership_test.go`/`hub_claim_test.go` cases are re-keyed and still
  pass.
- [ ] Commit `#694 M1: …`, then `sdlc milestone-close --issue 694 --milestone M1`.

### M2 — Teams run: startup, addressing, registration, switcher, session-sync

**Files:**
- Modify: `couchcmd/hubs.go`, `hub_cli.go`, `hub_claim_files.go`, `hub_move.go`, `hub_guards.go`,
  `message_service.go` and `wait_notice.go`, plus the callers in §4.
- Modify: `couchcore/startup.go`, `startargs.go`; `wrapcmd/peer_runtime.go`; `couchmessage/model.go`, `routing.go`.
- Create: `couchtty/bargroups.go` and `bargroups_test.go`.
- Modify: `couchtty/thread_presentation.go`, `menu_render.go`; `sessionsync/ledger.go`, `file.go`;
  `couchcmd/session_sync.go`; the actors JSON.

- [ ] Test (red) `TestPrepareHubsClassifiesTeamAndCheckout`: a `.hub` folder is a team; a plain repo path is a
  checkout hub; a folder at the checkout root refuses; the same team twice starts once with a note; a
  `path:alias` argument splits only when the whole path doesn't exist.
- [ ] Test (red) `TestStartupPrintsTeamNamesLine`: the line is printed once, before the first launch.
- [ ] Test (red) `TestSendToQualifiedTeam`: `team:ops2/devinfra` reaches the qualified team.
- [ ] Test (red) `TestBareCouchUnchanged`, `TestSingleRepoPathUnchanged`: no `.hub` read, the same hub, the same
  sections.
- [ ] Test (red) `TestTeamStartScopeIsTeamFolder`: two teams in one checkout get distinct scopes and records; the
  working directory is the folder; `COUCH_TEAM` is set.
- [ ] Test (red) `TestTeamWrapperNeverRegistersAsCheckoutSlot`, `TestTeamAdmissionChecksFolderScopeTag`,
  `TestBindingTeamOmitEmptyKeepsSlotBytes`.
- [ ] Test (red) `TestSendToTeamByAddress` (`team:devinfra`), `TestSendToHubResolvesOwningTeam`, `TestMoveSlotToTeamPersists`.
- [ ] Test (red) `TestBarMergesAdjacentTeamAndSameNameRepo` (`ECHO :0 :1`),
  `TestBarKeepsTeamAndRepoSeparateWhenNotAdjacent` (`ECHO tools :1 :2 echo :1 :2`), `TestTeamDisplayNameCapitals`.
- [ ] Test (red) `TestSwitcherTeamsSectionFirst`, `TestSessionSyncOwnerAcceptsTeam`, `TestActorsJSONTeamFields`.
- [ ] Implement until green, then the full suite.
- [ ] File the parley.nvim issue for the "my team" attention scope (reads `owner_team`).
- [ ] Atlas: Couch hubs page becomes teams and hubs (terminology: team, checkout hub, steward).
- [ ] Commit, then `sdlc milestone-close --issue 694 --milestone M2`.

### M3 — Team lifecycle (after #695 lands on main)

**Files:** `couchcore/lifecycleaddress.go` (team arm's `Key`/`Validate` wiring), `couchcore/startup.go`,
`couchcmd/hubs.go`.

- [ ] Test (red) `TestTeamAgentTakesLeaseByKey`: two teams in one checkout hold distinct leases.
- [ ] Test (red) `TestTeamRecordBindsTranscript`: binding stored, resume by exact ID.
- [ ] Test (red) `TestTeamRecordFieldSurvivesOlderRewrite`.
- [ ] Test (red) `TestRestartResumesBothTeams`: a fake-backed Couch restart resumes `devinfra` and `echo` through the
  lifecycle path, not a fresh start.
- [ ] Implement until green, then the full suite.
- [ ] Live check (Done-when): from ops, `couch teams/devinfra teams/echo` starts two named hubs. An echo slot's
  `--send-to hub` reaches `echo`, and another slot's reaches `devinfra`. A Couch restart resumes both. The operator
  runs it, and I ask before touching the live Couch.
- [ ] `sdlc close --issue 694`.

## 7. Open points for the hub

1. **#684 landing shape.** Resolved (hub, delivery 51ef8069): #684's M1–M2 split into pair#695, landed from main
   like any issue (deps: #683). This plan's M3 depends on #695 only.
2. **`.hub` directive lines** (§2, `HubClaimDocument`): this reading clarifies design §2 without changing it. Accepted by
   the hub, which carries it to ariadne#456 and design §2.
3. **Kind and arm naming:** #684's address kind becomes `team` (`TeamIdentity`), matching the design. Done: #684's
   plan has the Revision (227faef80), and the type now lives in #695's plan.

## Revisions

- 2026-10-19 00:10 — Revision 1, operator display input via the hub (delivery d7c631c0; the session file's
  Operator instructions item 8). A team displays as its name in capitals with no `:N` (a singleton, like a repo
  without worktrees). A team and a same-named repo that are adjacent in the bar merge (`ECHO :0 :1`); otherwise
  they stay separate (`ECHO tools :1 :2 echo :1 :2`). Added the pure `GroupBar`/`TeamDisplayName`
  (`couchtty/bargroups.go`) and three tests in M2. Consequence for the namespace: a team may now share its name
  with a repo. The bare name addresses the team and the repo's slots are `echo:N`; teams still refuse a clash with
  another team, an alias or `hub`. This relaxes design §3's "clash refuses at startup" for team/repo pairs, so it
  goes to the hub. The parley issue's attention scope partitions by team or by slot.
- 2026-10-19 00:10 — the hub split #684's M1–M2 into pair#695 (delivery 51ef8069). M3's dependency is now #695, and
  every `#684 M1–M2` reference points there.
- 2026-10-19 00:40 — Revision 2, operator decision on name clashes (hub delivery f3aab0d2). A startup never refuses
  over a name, because such refusals leave the user unclear about what is running. Startup aliases come first
  (`path:name`; team names are never numeric). A remaining clash with an earlier team, an alias or `hub`
  auto-qualifies the later team as `repo:team` (`ops2:devinfra`), and both forms address it. A team may share a
  plain repo's name (Revision 1). One startup line shows every resolved name. `AddressNamespace`/`CheckNamespace`
  became `ResolveTeamNames` plus `TeamNamesLine`, and `WorkspaceReference` learns the qualified team form. Three
  details were added by me: the first team keeps the bare name; two checkouts of one repo fall back to
  `ops:2:devinfra`; and a folder name outside the grammar is normalized, not refused. Names are per startup and
  never persisted, since the record and lease key on the folder. Tests are in M1 (aliases, auto-qualification, an
  alias resolving a clash, the line, the parse) and M2 (the printed line, the qualified send).
- 2026-10-19 01:00 — Revision 3, operator simplification (hub delivery db41cdb8). The canonical team address is
  `team:<name>`, like `pair:2` and `pair#444`. Bare names never resolve to teams: `echo` is the repo, `echo:2` a
  slot, `team:echo` the team. This removes team-versus-repo, team-versus-alias and team-versus-`hub` clashes, and
  `hub` from a slot still means its own team. The only clash left is two teams with one name. The startup alias
  resolves it, and otherwise the later team is auto-qualified as `team:<repo>/<name>` (with the fallback
  `team:<repo>:<N>/<name>`), never refused. `team` is a reserved prefix only before a non-numeric name, so repo
  `team` keeps `team:N`. Dropped from Revision 2: the `repo:team` form, the bare-name-means-team rule, and the
  alias/`hub` clash rules. Display is unchanged: capitals in the bar and the `ECHO :0 :1` merge. Tests were swapped
  accordingly in M1 and M2.
