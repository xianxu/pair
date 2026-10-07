# couch — the session supervisor

`couch` is a second binary in this repo (`cmd/couch`) that supervises agent
sessions: it registers them, spawns them, and knows what is running where. It
is **not** an extension of `pair`. pair is what the operator sits inside, so a
supervisor bug must not break the ability to fix it; the fallback is always to
launch pair the old way.

Project: `workshop/projects/couch.md`. Registry/spawn shipped in `pair#145`;
the pty console, actor panel, notices, and complete local lifecycle shipped in
`pair#146` M1-M4.

## What exists today

Couch has one production supervisor and one selected inventory per local OS
account. `couchcmd` resolves the real UID's account home independently of
`HOME`/XDG overrides; `couchsingleton` owns `.local/share/pair-host/singleton` there. A durable
selection fixes the physical Couch store, Pair data root and identity authority.
The host lifetime lease is acquired before the selected store's existing
non-inherited advisory lease. A second launch refuses with owner/store evidence;
the kernel releases both leases after a crash. The store lease continues to
protect against older binaries using that namespace, but cannot fence older
binaries on other stores: stop and upgrade them before cutover. This contract
requires machine-local filesystems; distributed/shared homes are unsupported.

`couchcore.ThreadStore` is
the shared lifecycle interface. Primary and arbitrary-path records use the global
store; numbered slots use `<environment>/.couch/` with one current conversation
record, separate preferences, and retained history. Each backend uses the existing
lock, revision checks and recoverable journal. The supervisor lease still belongs
to the selected global namespace beneath the account lease.

### Singleton adoption and isolation (#366)

`couchsingleton.Manager` separates read, preview, adoption and lifetime ownership.
Normal selected reads validate the selection without a global ownership scan.
Absent selection permits an unambiguous fresh launch to adopt; read-only commands
instead print the explicit adoption command. Existing selection is never silently
replaced, and missing selected resources refuse rather than recreating identity.

`couch --adopt-store /absolute/store` emits a JSON preservation report. Optional
`--pair-data /absolute/path` and `--identity-dir /absolute/path` identify the
store's existing companions; `COUCH_IDENTITY_DIR` also supplies legacy identity
authority. Repeated `--legacy-store /absolute/path` adds custom sources beyond
the bounded identity/retention registry union. Repeat the complete request with
`--apply <digest>` to revalidate the report's 64 lowercase hexadecimal digest and
publish the selection under host then store leases. Preview never initializes
source directories or repairs journals. The source remains in place: C/N/M,
conversation keys, preferences and worktrees are preserved.

Retention evidence or an explicit store/Pair-data tuple establishes the companion
artifact root; identity registration alone does not. `COUCH_PAIR_DATA_DIR` carries
the selected Pair root to hosted children independently of repo-scoped
`PAIR_DATA_DIR`. Couch clears the previous actor's scoped directory before launch;
Pair derives the new scope from its checkout under the selected global root. Global
claim/read paths and installed runtime extraction use that same selection. Existing
hosted helpers may retain the matching scoped directory. Inspection has a five-second context and limits of 4096 stores,
65536 filesystem entries, 64 MiB total source payload and 4 MiB per file; exceeding
a limit leaves unresolved evidence and refuses adoption.

Multiple populated stores or unavailable evidence report **UNMIGRATED**. This
release does not merge inventories. Repeated `--exclude-store /absolute/path`
records an operator's decision to retire another inventory from
active use; it preserves files and registry entries and cannot exclude live or
unknown owners. Unregistered custom stores require explicit disclosure. Digest
validation excludes transient supervisor metadata; owner observation remains a
separate admission predicate. Changed source evidence requires a new preview.

Adoption evidence includes numbered-slot `.couch` metadata as well as the global
namespace. The final inspection holds their existing transaction locks through
selection publication; unreadable slots, pending recovery and changed payloads
refuse. A free supervisor lease does not establish that recorded wrappers are
gone: live or unknown incarnations in other stores also block exclusion. Surviving
wrappers in the selected store keep their namespace and reconnect normally.
Selection serialization and reading share a 64 KiB encoded-payload limit, checked
at preview and again before publication.

`COUCH_ISOLATED_ROOT` names an explicit absolute canonical test/diagnostic root,
with authority at `singleton` and defaults at `data/pair`, `data/pair/couch`, and
`pair-host` beneath it. Every effective root must remain inside it, including
after symlink resolution. This mode needs no account lookup. Production ignores
HOME as an authority selector; alternate store/XDG roots alone do not isolate.
Derived child HOME, XDG data and temporary directories are also resolved and
checked before selection publication or directory creation. Escaping or unresolved
symlinks refuse; children receive only validated physical paths.
The resolved runtime propagates selected Pair/Couch/identity/isolation roots to
children and feeds the same roots to listing, messaging and artifact readers.
Existing socket addressing remains derived from the selected namespace.


### Durable numbered slots (#306)

The thread action menu's **Add slot** entry (#313) opens the existing start form
with the family's starting directory projected into the primary checkout and agent focus. `menuRepositoryRoot`
uses the validated slot primary root or scope-matched ordinary root; unknown
roots do not offer the shortcut. The same StartCreate preview/fingerprint and
repository admission checks apply as when entering a path manually, so Add slot
also fills the lowest free number (`:0` included) before adding one, and the
form lists the repository's parked threads when it adds one (#332).

Directory/Git identity supplies the durable slot; native scope/tag identifies its
current conversation. Slot rows stay selectable when current metadata is missing
or damaged. Explicit fresh conversation replaces the current reference and retains
old evidence without retiring the directory or preferences. Unknown process/session
ownership still refuses launch.

Global manifest schema 2 stores enrolled primary roots and their verified common
Git directories, not slot lifecycle facts.
Catalog enumeration rebuilds the slot inventory. Enrollment stages local metadata
before publishing the root and retiring global copies; interrupted journals replay
idempotently. Retention independently enumerates local stores and preserves native
owners referenced by current or archived records. Missing/corrupt ownership blocks
GC. Archive deletion receipts carry the backing slot location.

Repository families (#355) reserve one relative starting directory per physical
Git common directory in the root manifest. `repository_family.go` owns conflict
resolution and safe checkout projection; `repository_family_store.go` previews
read-only and reserves under the existing journal before provisioning or launch.
Park/archive retain the descriptor. Legacy records infer a coherent directory;
conflicting legacy directories preserve existing conversations but refuse new
admission. Checkout roots remain separate from starting/working directories in
launches, local-store routing, inventory, and menu actions. Missing directories
and paths escaping the checkout refuse before the agent starts.
Storage membership also requires matching checkout scope/common Git identity,
so nested independent repositories keep their own conversations and preferences.
Enrollment retains common-directory identity independently of the family's
starting-directory reservation, including separate Git directories. Older
metadata without that authority remains readable by exact checkout scope;
identity-dependent routing requires verified re-enrollment. Conventional paths
locate slots but never supply repository identity.
Each store admits at most 4096 families; existing families remain usable at the
limit. Reservations are retained on park/archive and are not silently recycled.

Key seams: `slotcatalog.go`, `threadstore_layout.go`, `threadstore_location.go`,
`slotmigration.go`, `threadstore_preview.go`, `slotinventory.go`, and the shared
retention adapters. Slot row keys use host paths; process/terminal maps retain
native addresses. Creation admission and launch recovery are described in
[workspace provisioning](workspace-provisioning.md).

### Slot references and repository aliases (#360)

Every `repo:N` reference — operation refs, the switcher filter, and
`--send-to` — resolves its repository part with `ResolveRepositoryName`
(`couchcore/repositoryname.go`): exact directory name or alias, then a unique
prefix of either, counted per repository so two repositories sharing a name
are refused rather than picked. Misses and ambiguity list bounded candidates
(`FormatRepositoryCandidates`, ≤12 entries / 1 KiB). Operations
(`Couch.repositoryPrimary`, `slotcontext.go`) resolve over enrolled
repositories, after giving an existing sibling directory precedence (an
un-enrolled sibling opens as before; one whose existence is undecidable keeps
its own error). Messaging (`couchmessage.ResolveRecipient`) resolves over the enrolled
families (`Broker.SetFamilies`) plus live bindings' families, so an offline
repository's exact name misses rather than prefix-routing to a live one; it
applies `--agent` only after the family identity check, and keeps its response
codes on a miss, listing live slots when the repository has none. The switcher
filter matches by prefix without the uniqueness rule, since it lists every
candidate.

Aliases live in the root store's `repository-aliases.json`, not the strictly
decoded manifest, so older couch builds can still read the store. One entry per
enrolled repository; clearing removes it; entries for un-enrolled roots are
ignored and dropped on the next write; a stored alias that would shadow a
directory is withheld on read. `ApplyRepositoryAliases` labels slot rows (by
primary root) and the `:0` row (repository scope, starting at the primary root);
subdirectory threads keep their labels, and `PresentThreads` names the
group by its alias, so the tabs and switcher show `alias:N`. The `alias`
operation is a switcher action on live `:0` rows only; it refuses an alias that
an existing sibling directory would shadow. `couch --actors` shows each slot's
alias and agent.

### Live peer messages (#353)

The supervisor owns an ephemeral `couchmessage.Broker` under its existing
namespace lease. Committed live pane observations validate exact wrapper
bindings: repository, scope/tag, native session, launch nonce and PID/start
identity. A slot label is an address, not authority to reuse a replaced wrapper.
No message body, mailbox or receipt is persisted across supervisor restart.

Cold resumes mint a new wrapper launch nonce just like fresh conversations;
the resumed native conversation ID does not identify a wrapper incarnation.
Couch-launched Codex disables shell snapshots for that process so tool shells
inherit the current slot environment instead of restoring an older slot's
identity. The override is not persisted into saved agent arguments.

`couch --actors`, `--send-to repo[:N] [--agent NAME] --message TEXT` and `--message-status ID`
use bounded Unix stream RPC, without constructing another mutable supervisor.
`--actors` and `--message-status` support JSON. One pending delivery per actor,
eight inbound admissions between genuine operator submissions, and bounded
receipt retention limit the runtime. Family selection requires fresh resting
branch and quiet-wrapper observations; exact sends allow occupied-slot
coordination. Quietness is never acceptance of repository work.

**Registration follows lifecycle events, not polling (#365).** Each wrapper
holds one long-lived session on the namespace's `registry` socket
(`couchmessage.SessionClient` / `SessionServer`). The hello carries the binding,
and the kernel's peer PID must equal `Binding.PID`. The connection closing *is*
the death or exec event: Go sockets are close-on-exec, so a SIGUSR2 re-exec
reconnects as a new session even when its binding is byte-identical. The
Console posts each thread's current pane to a coalescing
`couchmessage.PaneMailbox`. `SubscribeMessageLifecycle` replays the panes
attached before the service started (the startup pane and the reattach pass).

One pure `couchmessage.Registry` decides which bindings may receive:
- **Admission:** a binding is connected when its newest session is admitted
  against the thread's current pane. `messageService` executes the registry's
  effects: one full authority check per admission, then broker
  `Register`/`Disconnect`.
- **Ordering:** the registry owns same-slot displacement, the newest session
  wins, and late admission results for a closed session or a replaced pane are
  discarded.
- **Failed checks:** retry on a bounded ladder (0.5–8 s), then the session goes
  dormant until something touches it: an attach, a submit, a reconnect, or a
  send that targets its slot.
- **Exact recipient:** the newest admitted session wins. A request already
  committed to the old incarnation may still finish there, or end
  `Indeterminate`; it is never redirected to the new one.

A full check (`messageAuthority.live`) runs the ps/zellij ownership probe, so it
runs only at admission. Observe, Reserve, Deliver and a sending caller use
`messageAuthority.current` instead, which spawns nothing. It checks the pane is
live, the PID file still names the wrapper, and the launch's recorded ready file
and session index still name its nonce and session.

The wrapper pushes observation changes coalesced to at most one frame a second,
submissions at once, and nothing while idle. `--actors` reads that memory. An
admitted idle actor stays known. The resting branch comes from the Console's
slot-git cache (`MessageSlotGit`), and only family admission runs git, per
candidate per send. The pre-#365 1 s heartbeat, 1 s reconcile and 10 s
verification window are gone. Wrappers from older binaries still send
`register` and are answered `unsupported` at no cost; their slots receive again
after a relaunch.

**What delivery does and does not promise (#365).** Messaging promises
at-most-once input to the wrapper's PTY, not exactly-once task execution. The
remaining uncertainty is stated, never papered over:
- **Wrapper receipt.** A wrapper's receipt says the envelope was pasted and
  submitted to the agent's composer. It says nothing about whether the agent
  read or acted on it. Check the transcript or the issue state for that.
- **Death mid-delivery.** A wrapper, broker or Zellij crash during delivery
  ends the receipt `Indeterminate`. The paste may or may not have landed, and
  it is never retried.
- **Replacement mid-delivery.** If a wrapper is replaced while delivering, the
  message completes at the old incarnation or ends `Indeterminate`. It is never
  redirected to the replacement.
- **Couch restart.** Receipts are memory-only. After a restart,
  `--message-status` asks the connected recipients for their retained receipts:
  each wrapper keeps its last 64. A recipient that does not answer makes the
  answer `uncertain`, not absent. One that has since exited or exec'd has
  forgotten. A receipt recovered while still in flight is reported
  `Indeterminate`, because no broker job watches it any more.
- **Retries.** The CLI mints a fresh ID per send and tells the sender not to
  retry an uncertain one. A reused ID that reaches a wrapper that remembers it
  adopts the recorded outcome and is not pasted again.

Each wrapper endpoint conditionally reserves its observed input generation,
then accepts one delivery commit. The broker polls outcome receipts; it never
retries PTY input after uncertainty. Pair's input owner arbitrates ordinary
typing, image admission and automatic paste/submit. Unknown or occupied
composers wait within the delivery deadline. Interference cancels automatic
submission and leaves visible text for inspection. Receiver profiles exist for
Claude Code and Codex CLI at any installed version (`peerReceiverAgents`; the
exact-version allowlist was removed in #360 after auto-updates silently dropped
slots). Fixtures under `wrapcmd/testdata/peer/` and `TestPeerLiveConformance`
were captured on Claude Code 2.1.286 and Codex CLI 0.159.2; per-version
evidence from daily use is #368. Short-message submission
has live evidence for both; deterministic wrapping is matched conservatively.
Collapsed paste summaries remain unsubmitted and expire. Human Couch acceptance
remains a separate step.

Single-line suggested prompts in recognized agent composers use shared ANSI faint styling
and the cursor at the input origin, independent of wording or RGB color.
Delivery reads the current composer, not a sticky human-draft flag. Buffered
input/output blocks inspection; newly forwarded human input gets one second to
settle before inspecting the screen. Visible draft text still blocks delivery,
while an erased draft can become eligible without submitting it. A one-second
polling timer exists only while a message is pending; events also wake checks.
After the input deadline, the broker allows two seconds for read-only receipt
collection; this never extends the wrapper's paste/submit deadline. Waiting
receipts report the blocking guard and retain that reason on expiry. Senders
query `--message-status` for final outcomes; admission returns before delivery.
Orientation and peer delivery share automatic-input ownership across the entire
paste/render/submit transaction. The next automatic writer waits for a fresh,
empty composer after that transaction terminates, including cancellation.

Transport frames are bounded at 577 KiB to carry all 128 actors with bounded
4 KiB encoded bindings, or a fully escaped 8 KiB message. Receipt diagnostics
are capped at 1 KiB. Wrapper socket names retain the incarnation hash and name
the owner PID; runtime server startup collects only sockets with proven-dead
owners, preserving live/uncertain owners and replaced inodes.

Key files: `couchmessage/{model,routing,broker,transport,protocol,endpoint}.go`,
`couchcmd/{messages,skill}.go`, and the wrapper delivery adapter. The canonical
skill is `couchcmd/skills/couch/SKILL.md`, embedded for `couch --skill` even
outside Couch. Loading or installing it is explicit; runtime does not change
agent configuration. Ordinary messages carry follow-up context in their body;
there is no reply protocol or availability toggle.

### Recover plan after a restart (#367)

`couch --recover-plan-from-sdlc` (operation `recover-plan`, `ExecuteDirectStore`,
`EffectRead`, `PresentationRecoverPlan`) runs in the CLI process like `--list`
and never touches the console. Flow:

1. `Couch.RecoverPlan` (`recoverplan_source.go`) reads the enrolled repositories,
   resolves each primary's `fleet_root` through the slot catalog's
   `sdlc workspace --json` probe, and runs `sdlc fleet inventory --json --path
   <first primary>` once per fleet (sorted, at most 8) through
   `SDLCFleetSource` over `ProvisionIO` (process group, 90 s, 1 MiB).
2. `DecodeFleetInventory` (`recoverplan_fleet.go`) is the only reader of those
   bytes: `schema_version` must be exactly 1 and `slots`, `machine`,
   `dangling_claims` and every `rows[].claims_state` present, else
   `ErrFleetSchemaUnsupported` (a pre-ariadne#288/#289 build never reads as
   zero slots). Duplicate keys refuse; additive fields are accepted.
3. Couch's side is `ActionableThreadInventoryContext(ctx, nil)`, the same
   evidence gather as `--list`. An unreadable store makes every agent unknown.
4. For an unavailable or unsupported fleet only, the shell lists its slots by
   Couch's layout (enrolled primaries + `EnumerateSlotCandidates`) and reads
   each with the switcher's `ProbeSlotGit` under the shared
   `SlotGitProbeTimeout` (3 s). That probe has no ahead or operation facts, so
   those stay unknown.
5. Pure `DeriveRecoverPlan` (`recoverplan.go`) builds the slot universe (fleet
   slots, candidates, Couch slot rows, dangling claims on a conventional slot
   path), joins Couch rows by `Target.Slot.WorktreeRoot` or, for `:0`,
   `IsPrimaryRow` (the one definition, shared with `ApplyRepositoryAliases`),
   dedupes by address (first fleet wins), and reads each slot into closed
   `SlotEvidence` dimensions where unknown is a value. `classifyRecover` is the
   first-match rule table; steps come only from rule A over
   `ActorActions(ActorRowFactsOf(row))` plus the literal `ask-agent-restore`
   (`RestoreWorkspaceMessage`). `recoverReason` authors every row's text.
   Every member of a slot is judged on its own facts (M1 review BR-4/5/14):
   the host's branch, dirt, unlanded commits, operation, claims and claim-read
   quality fill the host dimensions; each dependency is judged by
   `judgeDependency` on its own branch, dirt, unlanded commits, operation and
   claims (`judgeMemberClaim`: active, resting, other, unknown) and folded by
   `foldDependencies` into `DepClaims` (none, active, resting, resting-dirty,
   conflict, work, unknown) plus `DepTree` (clean, dirty, unknown, operation:
   every dependency's working tree folded worst first). Reboot safety is a
   slot-level property, since a reboot replaces the whole slot's agent: rule
   A's reboot guard and its `inspect-uncommitted-first` note read only the
   `slotOperation` / `slotGitUnknown` / `slotDirty` helpers, which fold the
   host's tree with `DepTree`; every other judgment reads host facts alone
   (M1 review round 4). The union across members appears only in the row's
   `evidence` list. A dependency claim on another branch is
   `conflict:dependency-claim`; an unread or gone dependency (a dangling claim
   on a missing member) is note `dependency-unread`, never absence; a
   dependency's own unclaimed work is `dependency-work`; a dependency claim
   beside a host-decided row is listed as `inactive-claims`.
   `TestHostJudgmentsReadOnlyHostFacts` proves over the whole evidence domain
   that dependency facts never change a host judgment. The JSON row
   shows them under `claims.dependency` (`ref`, `checkout`, `state`), and
   `RestoreWorkspaceMessage(ref, address, checkout)` names the checkout that
   holds the claim, and the dirt that blocks a restore is that checkout's own.
   `hostAtRest` (resting or landed branch, no dirt, nothing unlanded, no
   operation, all the host's own) is the one reading of an idle host, shared by `idle`,
   `landed` and the dependency-claim decision; `recoverReason` derives each
   row's text from that row's own steps and notes.
   A clean, unclaimed slot on a done issue's branch is `landed` (note
   `issue-done-branch`, no step), not a conflict; `conflict:issue-terminal`
   needs dirt, unlanded commits, an operation or a claim on that branch.

| Entity | Lives in | Kind |
|---|---|---|
| `FleetInventory` / `DecodeFleetInventory` / `FleetSchemaVersion` | `recoverplan_fleet.go` | pure |
| `RecoverPlanInput`, `FleetObservation`, `CouchObservation`, `RecoverSlotCandidate`, `RecoverLocalGit` | `recoverplan.go` | pure |
| `RecoverPlan` / `RecoverRow` / `RecoverNext` / `RecoverStep`, `RecoverClass` / `RecoverHold` / `RecoverNote` and their `All*` vocabularies | `recoverplan.go` | pure |
| `SlotEvidence`, `slotEvidenceOf`, `classifyRecover`, `consistent` (test-domain pruning only) | `recoverplan.go` | pure |
| `ActorRowFacts` / `ActorRowFactsOf` / `ActorActions` (the switcher's resume/reboot arms) | `actor_actions.go` | pure |
| `IsPrimaryRow` | `actionableinventory.go` | pure |
| `FleetInventorySource` / `SDLCFleetSource`, `Couch.RecoverPlan`, `Couch.Fleet` | `recoverplan_source.go`, `couch.go` | IO shell |
| `FakeFleetSDLC` / `FakeFleet` | `recoverplan_fake.go` | stateful fake behind `ProvisionIO` |

`FakeFleetSDLC` models fleets of slots whose members carry branch, dirt,
operation, unlanded commits, issues and claims, plus dangling claims, off-slot
rows, claim-read quality, per-fleet schema and failure modes (exit, hang,
garbage). It judges members with sdlc's `JudgeCheckout` precedence and refuses
any argv but the one `SDLCFleetSource` builds. The golden capture
(`testdata/sdlc_fleet_inventory_v1*.json`) and `TestFleetInventoryLiveConformance`
(skipped without `sdlc` on PATH or under `-short`) pin its vocabulary to the
real producer. `TestDeriveRecoverPlanIsTotalOverTheEvidenceDomain` crosses every
evidence dimension and proves the defensive `no-rule` class unreachable. The
report writes nothing. Each step's `command` is `SlotOperationCommand` (adds
`--confirm` exactly when the declaration requires it) or `SendToCommand` with
the shell-quoted restore message; tests shell-split and parse every one.

**Slot operations through the running Couch (M2).** `couch --resume repo:N`
and `couch --reboot repo:N --confirm` act on one slot from a live Couch slot:

1. The CLI (`runSlotOperationCLI`, `messages.go`) admits `resume`/`reboot` on
   the broker socket with a request ID, then polls `operation-status` every
   500 ms within 3 minutes, each exchange on a fresh `AdmissionTimeout`
   context. After admission any lost outcome (dial failure, timeout, an
   unavailable caller after a restart, a receipt no longer held, the budget)
   prints one uncertain line pointing at the report.
2. `messageService.handle` intercepts the three ops before the broker
   protocol: `ValidateRequest` (one exact `repo:N`, `Confirmed` only on
   resume/reboot), then `liveCaller`, which authenticates by Couch's own
   liveness rather than messaging registration (operator decision after the
   smoke test): the thread named by the request's scope and tag has a live
   Couch pane (`authority.thread`), and the launch its record names
   (`authority.agent` reads the agent) records this shell's session and nonce
   (`authority.recorded`). No broker binding or wrapper PID is needed, so a
   slot whose wrapper's peer setup failed can still recover others. A
   malformed request, including one with no caller identity, is
   `invalid-request`; a caller that fails the check is `unavailable` ("caller
   is not a live Couch slot …"). Nothing is enqueued in either case. The CLI
   turns an older Couch's "unknown message operation" into a restart hint.
3. `slotOperations` (`slot_operations.go`) owns the in-memory receipts behind
   one mutex: reboot without `Confirmed` is `confirmation-required`
   (`OperationConfirms`); the queue key comes from the resolved repository
   (`remote\x00<primary key>:N`, so `pair:1` and `pa:1` share one pending
   key); a duplicate admission returns the held receipt and never enqueues;
   an enqueue refusal drops the reservation and answers `busy`/`overloaded`;
   more than 64 held receipts is `overloaded`. The pure receipt machine is
   `couchmessage.ApplyReceiptEvent` (`queued → running →
   succeeded|refused|failed`, a closed table). Receipts are created on
   admission, removed 5 minutes after they turn terminal (swept on every
   admit and status) or when Couch exits; status answers only the slot that
   admitted them, and a receipt not held answers `unknown`.
4. `consoleSlotOperations` wires `Console.EnqueueRemoteOperation`
   (`console_remote.go`) with `Couch.PrepareSlotOperation`: the job rides the
   switcher's `operationQueue`, `c.ops` and `finishOperation`. Prepare runs on
   the queue against fresh inventory: `ParseWorkspaceReference`,
   `WorkspaceReferencePath`, `SelectSlotRow` (`IsPrimaryRow` for `:0`), then
   `not-offered` unless `ActorActions` offers the op, and the call's args from
   `ActorOperationArgs` (slot path plus its scope; exact tag plus warm-only on
   a detached `:0`), the one mapping the switcher also reads. The origin is
   `{op, PreserveFocus}` with Attempt 0: adopted with `background=true`, never
   touching the operator's `InFlight`. A remote resume is recognized at
   completion as `resume`, Attempt 0, no `ContinuationID`, and clears the
   row's reattach-failure mark. `finished` fires once after adoption; an
   enqueue error never reports through it.

| Entity | Lives in | Kind |
|---|---|---|
| `OperationReceipt`, `ReceiptStatus`, `ReceiptEvent`, `ApplyReceiptEvent` | `couchmessage/operation.go` | pure |
| `SelectSlotRow`, `ActorOperationArgs`, `SlotOperationCommand`, `SendToCommand`, `SlotOperationError` | `couchcore/slot_operation.go` | pure |
| `Couch.PrepareSlotOperation` | `couchcore/slot_operation.go` | IO shell |
| `Console.EnqueueRemoteOperation`, `remoteResumeCompletion` | `couchtty/console_remote.go` | console queue |
| `slotOperations`, `consoleSlotOperations`, `handleSlotOperation` | `couchcmd/slot_operations.go`, `message_service.go` | socket handler |
| `runSlotOperationCLI` | `couchcmd/messages.go` | CLI |

`TestRecoverPlanStepsConverge` (couchcore) runs the loop: report, every
automatic step through `PrepareSlotOperation` and the live-owner dispatch,
report again, and a resend refused `not-offered`.
`TestSlotOperationSocketAcceptance` (couchcmd) drives `--resume repo:0`
through the real socket, wiring and a running Console. The skill's
"Recovering slots after a restart" section is the agent procedure.

**Slot reconciler in the report (#387).** `RecoverPlanInput.SlotPlans` carries
the slot reconciler's report per `:1+` host path, observed by
`Couch.recoverSlotPlans` for each slot `Discover` knows, with the row's agent
evidence. `RecoverSlotClass` reads a report into the `SlotEvidence.Reconcile`
dimension:
- `needs-zero` → class `slot-needs-zero`, hold `workspace-handoff`, ranked right after
  ambiguous threads. It is decided through `SlotOutcome`, the callers' own
  decision, so the report never holds a slot they would open
  (`TestRecoverSlotClassAgreesWithTheCallers`).
- `reconcilable` → a `reconcile` step, only on an otherwise idle row or a missing
  directory with leftovers, keeping earlier notes.
- `held`, `degraded` (a non-blocking hand-off) and `unknown` → notes.

`recoverReason` is total over the classes, and the new classes carry the
reconciler's advice or plan.

The mechanics are mapped in [workspace provisioning](workspace-provisioning.md#slot-resources-387).

### Grouped workspace display (#307)

`couchtty.PresentThreads` derives repository grouping, numeric slot order, full
labels and display paths from the existing row targets and repo scopes. A slot
maps to its primary checkout scope; an ordinary starting subdirectory can recover
its checkout root by matching ancestor scopes without filesystem IO. Unknown
legacy roots retain recorded path context. Same-name checkout groups stay distinct.

The menu installs this order at inventory ingestion, then overlays reattachment
state through its existing viewed lookups. Rendering uses stable row keys, including
addressless recovery slots. `Console.statusModelLocked` joins attached/pending
members through the same projection and keeps a pane fallback while inventory
catches up; attachment order still serves internal process bookkeeping.
`RenderStatusRow` shortens slot labels only after the visible group's anchor and
creates click spans in the same pass that clips text. Parked slots stay in the
switcher, and a missing primary tab gives the first slot a full `repo:N` label.

Key files: `couchtty/thread_presentation.go`, `console_presentation.go`,
`menu_reattach.go`, `menu_render.go`, and `reserve.go`. Rendered examples are in
`couchtty/testdata/slots_grouped_*.txt`.

### Checkout quick-status glyph (#317, #339)

Each checkout, including standalone repositories and `:0`, carries a glyph after its label in
both the switcher and the tabs, built from two independent parts
(`couchcore.SlotGlyph`, #319). The branch part is `` (U+E0A0) off its resting
branch (`main` / `main-slotN`, from `couchcore.RestingBranch`), or, on it,
divergence from its upstream: `±` both ways, `+` ahead only, `-` behind only.
The dirty part `*` follows on any branch. Behind reads the local
remote-tracking ref; the probe never fetches. Colour is decided per glyph
character by `slotGlyphSGR` (`reserve.go`), shared by both views: `±` and `*` in
the one attention amber (#321), the rest in the row's style; the switcher's selected row stays plain.
`PresentThreads` derives `ThreadPresentation.Glyph`
once from `MenuState.SlotGit`, so the two views cannot disagree.

The data is one `git --no-optional-locks status --porcelain=v2 --branch` per
checkout (`couchcore.ProbeSlotGit` / `ParseSlotGitStatus`; the recover-plan
report reuses it for slots sdlc could not describe, under the same
`couchcore.SlotGitProbeTimeout`). `Console.Run` owns a
single-flight refresh (`console_slotgit.go`, reusing `RefreshSchedule`): a 10s
ticker, every landed inventory (so opening the switcher), and every switch
request a pass. A worker probes the inventory's checkouts one at a time,
3s each, outside `c.mu`; `MenuEventSlotGit` rebuilds the map over the probe set,
keeping the last value for a failed probe. Render and keystroke paths only read
the map. A glyph can be stale; a git failure never reaches chrome. Standalone
roots are recovered from the immutable starting path and repository scope via
`presentationRoot`, not the child's current working directory. They use `main`
as their resting branch, just like a slot group's primary checkout.

### Idle fading of live threads (#247)

Live thread labels recede as they go idle, in both views: under 1 day normal,
from 1 day faded, from 3 days more faded (`IdleLevelFor`, `couchtty/idle_shade.go`).
`FadeStyle` blends the label's colour, or the amber of its `±`/`*` glyphs,
toward the terminal's reported background, so fading darkens on a dark scheme
and lightens on a light one. It falls back to ANSI 90 when the terminal has not
reported its colours, and changes nothing under `NO_COLOR`. Fading is the
weakest cue: the active chip / selected row, a pending notification (bell or
attention lines) and a placeholder keep their own styling. Non-live switcher
rows keep their own `AgeBand` ramp. The switcher reads `MenuState.Activity`
and `MenuState.Palette`; the tab bar reads the same two through
`statusModelLocked`, which fills `StatusActor.Idle` and `StatusModel.Palette`.
A thread with no activity entry is unfaded. Each slot glyph fades from its own
base colour (`slotGlyphBase`).

The data: activity is `threadactivity.Latest`, the one definition the title
poller's heat ramp also uses. It is the newest of the bound agent transcript's
mtime, the Pair log's (sends), and the current launch's pane-birth evidence
(rewritten on create or resume, never on attach). The draft is excluded: its
autosave writes on every focus loss. `console_activity.go` runs it like the slot
git pass: a 60s ticker, every landed inventory, and every switch request a
single-flight pass over live threads only, 2s per thread, outside `c.mu`, merged
with the shared `mergeObservations` (a failed probe keeps the last value). The
palette comes from one OSC 10/11 query `Run` writes before the first frame
(`console_palette.go`); the replies arrive as input `Reply` events, which
`routeInputEvent` records instead of dropping. `ensureMenuLocked` keeps a palette
that arrived before the menu was built. Measured cost: ~114ms per thread
(56 real threads in 6.4s), all on the background worker. A mid-session theme
change is not re-queried.

`registry.json` remains as a transitional live-handle cache for the shipped
console. It is not a metadata or display authority, and its records are claims,
not liveness: nothing removes one when its child exits. A reader that treats a
record as hosting proof (`classifyForAction`, behind archive and switch-agent)
probes `Couch.Liveness` first, so a dead record is residue and an unknown one
joins the evidence's Unproven side. Every launch reaps the known-dead records
before inserting its own (`withoutDead`, pair#378). The one-time journal import
of its actors into ThreadStore went with `pair#170` M4: every store that needed
it was cut over years of commits ago, and the manifest keys that recorded the
cutover survive only as decode tombstones. CLI diagnostics read the raw
one-row-per-composite-thread inventory; the ordinary switcher reads the
actionable projection described below.

That raw `ThreadInventory` remains the diagnostic/recovery view: persisted
incarnation states are shown even when Couch cannot prove a usable terminal.
M1 exposes `ActionableThreadInventory`, a pure fail-closed projection over the
same snapshot plus owner observations. It emits `live` when couch has POSITIVE
evidence it is hosting the thread's process -- a console pty child, or a recorded
process the OS still vouches for by exact PID and start token; since #256 M1 that
is a union with no match required between the two, and its absence proves nothing
(a record's incarnation names the launcher, which dies with couch) -- or
`parked` when its LEDGER resolves a conversation to resume into and nothing is
running on it — the park receipt is not the authority and has not been since #256
M2, and neither is the incarnation: two of the four `parked` shapes carry one
(a dead launcher, a driverless start claim), which is why the rule is "no LIVE
evidence" rather than "no incarnation". Contradictory and undecodable records stay available to
diagnostics. Since #151 M3, Console refreshes this projection asynchronously
from exact hosted PID/start observations and never promotes raw persisted
lifecycle state into a user-visible `live` or `parked` row.

#151 M2 added the pure core and M3 wired it into Console. One
immutable-by-copy `MenuState` stack owns the root filter/selection plus exact
thread-bound action, confirmation, and text frames; a global start frame
overlays the preserved originating stack. `ReduceMenu` is the only transition
authority for semantic keys, exact-address operation effects, inventory
refreshes, preview results, notices, and ephemeral per-thread bells. It
allocates monotonic menu-lifetime attempt and frame-instance identities,
captures both before dispatch, and rejects a mismatched attempt before
accepting its returned inventory. Existing-thread
operations correlate both outcomes with the captured request address; a failed
start needs no created address, while start success does. Effects that assert
success, such as clearing a switched thread's bell, commit only after that
correlated success. It reconciles
completion-owned stack prefixes against the captured frame instance and
preserves a newer global start overlay opened after dispatch; an asynchronous
completion does not own unrelated later UI. It reconciles
refreshed identity root-to-leaf independently from filtered selection and
discards the first invalid thread frame plus descendants; hidden-target notices
retain the prior human label and composite address, while a global start frame
survives with its saved origin reduced to the valid prefix. Every list frame,
including park confirmation, filters displayed labels while retaining internal
operation identities (`switch coding agent` presents `switch-agent`). Inputs
are byte-bounded at 1 KiB for filters/aliases and 4 KiB for paths.
The input seam decodes horizontal arrows in both CSI and application-mode SS3,
so the start form's agent selector is reachable in either terminal mode. Root
rows clip variable label/path text around a protected state/age/bell suffix at
the 40-column minimum. Generated key traces keep stack depth, UTF-8 ownership,
and effects bounded
(ARCH-DRY, ARCH-PURE, ARCH-CONSTRAINTS).

#338 adds normal/focus mode to the root frame. Space toggles it only with an
empty root filter; returning from a child frame and reopening retain that mode
for the console lifetime. `visibleMenuRows` applies focus membership after the
existing inventory overlay and typeahead: `Live()` plus a nonempty sanitized
`DisplaySummary()` -- the agent's published summary only, since #363 stopped
displaying the stored operator description. This keeps keyboard selection, mouse
extents, and rendering on the same rows. Focus rendering uses the full-inventory
labels and presents `label ◆ summary ◆ slug` on one line, clipped to terminal-cell
width; a row with no slug ends at its summary. The slug (#372) is the thread's latest
pair-slug suggestion (`slug-proposed-<tag>`, never the draft-mirrored
`slug-<tag>`), unfenced to `<branch> | <focus>` by `cmd/internal/slugline`,
the one definition of that line format. `ApplySlugs` fills
`ActionableThreadSummary.Slug` for live rows after the projection, through the
`Couch.Slug` seam (`OSSlugReader`: bounded, no-follow, display-only, so errors
read as no slug). The existing inventory refresh supplies new descriptions and
slugs, which is why a slug change redraws without ever becoming attention; no
additional storage or polling is added.

Both CLI resolution and in-memory menu filtering derive from
`ClassifyThreadReferenceFields`/`MatchThreadReferenceFields`: exact opaque tags
win set-wide over case-insensitive name/path containment. The menu also supplies
the sanitized displayed description as an optional fuzzy field for both root
views; CLI callers leave it empty. Slot rows search that same display text unless
the filter is an explicit slot reference. Default description matches add a
shared-row detail line; focus keeps its inline summary. No store read occurs on
the keystroke path. `RenderMenu` consumes only state, terminal dimensions,
clock input, and the 256-color capability. It keeps the selected row inside a
bounded viewport, anchors wide children beside the selected parent row and
narrow children below the measured parent list, keeps the current frame
operable at 40x10, asks for resize below that, strips controls, clips by
terminal columns, and renders live state without historical age while parked
rows retain text age plus an optional three-band grayscale. `DecodePanelKeys`
maps legacy HT and unmodified Kitty CSI-u Tab to the same semantic key; modified
Tab remains a dropped chord.

Start preview scheduling is also pure: `AdvancePreviewSchedule` admits one
running identity and one replaceable latest identity. `MenuState` allocates
those identities monotonically across edits and start-form lifetimes. A newer
request asks for cancellation once, but only a terminal outcome for the running
identity retires it. The start frame binds accepted `PreparedStart` and one
armed submit to the same nonzero identity; edits, Escape/reopen, stale results, failures, and
duplicate results cannot allocate or reuse authority incorrectly. An unchanged
accepted generation reuses its one grant, non-sticky fallback agents remain
omitted from the preparation request so path history can resolve them, and the
accepted agent plus agent/argv provenance are rendered from the shared
resolution. Console opens and navigates from the last-good in-memory
projection. One single-flight refresh plus one dirty follow-up owns inventory
I/O; one running and one replaceable-latest preview bound start resolution.
Lifecycle operations run on the existing capacity-one queue with exact
attempt/frame correlation, so input and repaint never wait for store, process,
or harness work.

#160 extends the same reducer/effect boundary with directory-only path
completion. `SplitCompletionPath` preserves editable relative/absolute spelling;
the Console reads at most 128 entries per filesystem batch behind
`DirectoryBatchReader`, while `CompletionAccumulator` retains the lexical top
200. One active scan and one replaceable pending request share the generic
latest-wins scheduler. Exact frame/generation identity makes canceled or late
results inert, and rendering reserves start-form controls before allocating a
selected-candidate viewport (ARCH-DRY, ARCH-PURE, ARCH-MOCK, ARCH-CONSTRAINTS).

`cmd/internal/artifactpath` is the sole constructor for Pair's tag-bearing
files. Standalone Pair selects its own `{repo_scope, tag}`; Couch allocates the
same address shape for a hosted start and Pair establishes the pre-reserved
claim. The launcher then exports exact paths to Go helpers, shell, Neovim, and
both Zellij layouts.
Each resolved-consumer family is tied to a named resolver/member witness;
closed vocabulary allowances separately cover exact non-path protocol and CLI
uses. Every production source is exhaustively inventoried as one of those
classes or as a non-artifact source; new files have no implicit default, and a
Go source that imports `artifactpath` cannot remain in the non-artifact class.
Current resolved consumers have positive family-specific resolver/member (or
direct resolver) bindings. Exact vocabulary and direct literal or constant-
expression checks are bounded defense in depth; they do not claim semantic
provenance through arbitrary helper, package, control-flow, or string-building
programs. The Core concepts contract derives the artifact
authority's type/catalog inventory from its exported declarations rather than
copying a second expected list.
Generated-runtime coverage builds a temporary mirror from declared source
inputs. The clean-bootstrap regression starts without `.git` or that mirror and
proves the public test target generates it before every consumer.

`couchcore.Operations()` is the closure-free capability schema: typed
argument/result family, effect, confirmation, execution owner, and presentation.
`list`, `show` and `archived` project as public `--list`, `--show` and
`--archived`. `peek` (pair#362) projects as `--peek repo:N [--lines N]
[--json]`, a read-only look at another slot. It returns the plain-text tail of the
thread's live terminal recording (`scrollbackcmd.RenderOwnedLines`, under the same
retention lease `pair scrollback render` takes), the Pair sent-prompt log, and
native transcript paths from the switcher's `OSSwitchContextResolver`. Transcripts
are paths, never parsed, and every unreadable source is named in `unavailable`.
Any operation that declares a `json` flag prints its result as JSON.

The hosted-agent hook `publish-description` projects only through hidden
`couch --internal publish-description <text>`, which pair's draft calls for a `!`
tag line (#337), a `!!` describe line (#358), and a bare `!` clear line (#357),
which publishes an empty summary. `prepare-start`, `start`,
`attach`, `switch`, `park`, `resume`, `relaunch`, `prepare-switch-agent`,
`switch-agent`, `leave`, `stop`, `alias` and `reboot` are TUI/in-process
operations. `orientation-status` is an internal owner operation for one launch
attempt. #363 removed `open-slot`, `fresh-slot`, `name`, `describe`, `archive`,
`recover-thread` and `recover-checkpoint` from the declarations: resume and
reboot replace them, and their couchcore internals (`OpenSlot`,
`StartFreshSlot`, `Couch.ArchiveThread`, `RecoverThread`, `ApplyThreadMetadata`)
stay as what resume and reboot compose. The stored name and description fields
stay on the record, written by nothing in the switcher, and are neither displayed
nor matched: `ThreadReferenceFields` is `{Address, Label, WorkingPath, Summary}`,
store resolution leaves `Label` and `Summary` empty (tag and path only), and the
switcher passes what a row shows.

**Resume and reboot are the actor operations for a row that is not live (#363).**
`resume` (`Couch.ResumeTarget`, `couchcore/resume_route.go`) gets the old
conversation back by whatever path works: a pure `ChooseResumeRoute` hands the
row to `OpenSlot` (warm, cold, or adopting a conversation whose pointer couch
lost -- on resume's guessed agent only through the native ledger, which binds
per agent; a record-less detached survivor is refused `resume-survivor-unproven`
because the detached proof cannot tell which agent runs), `RetryContinuation`,
`RecoverThread` or `ResumeContextWith`. It takes a slot `path` or a thread
address; `warm-only` (the background reattach pass) stays a direct warm-only
resume. A refusal whose transcript cannot come back names reboot; the
`ResumeRebootAdvice` map classifies every `ResumeDiagnosticCode`, so transient
refusals say retry instead. `reboot` (`Couch.Reboot`, `couchcore/reboot.go`)
archives the old record with its evidence and starts a fresh agent with a new
tag in the same slot or path, without touching the worktree; a pure
`DecideReboot` picks archive-and-start, archive-only (directory missing, or an
unreadable `:0`), start-only, or refuse, and `RebootableState` is archive's
admission rule. The fresh profile resolves before `prepareRetirement`
(`ArchiveThread`'s admission and quiesce half) stops anything, and the retire
plus the fresh claimed record commit in one store journal: `replaceSlotCurrent`
for `:1+`, `ThreadStore.ReplaceThreadExpected` for `:0`, which composes the
same `archiveJournalEntries` / `createJournalEntries` builders `archiveThread`
and `CreateThread` use. Fresh records start without the stored name and
description; the archived record keeps them. Both kinds report the retirement
half through one `retiredResult`, including a session left unstopped.

**The switcher's per-row actions are one pure table (#363 M2).**
`menuActionItems(row)` is `menuRowActions(menuRowFactsOf(row))`
(`couchtty/menu_actions.go`): `menuRowFacts` reduces a row to its kind (`:0`
primary or `:1+` slot) and phase (live, live with a failed / running / pending
continuation, resumable, unusable, unknown, busy) plus `ResumeOffered`,
`DirectoryMissing`, `AliasOffered` and `AddSlotOffered`, and `menuRowActions` is
the Spec table over them. Live rows get detach, relaunch, park and switch coding
agent (`:0` adds alias and add slot); rows that are not live get resume and
reboot, and a `:0` among them also gets add slot unless its directory is missing
(pair#402: a new slot needs the primary checkout, not `:0`'s agent); unknown,
busy and live-pending rows get nothing, and `menuRowAdviceOf`
reads the same facts to say why ("state could not be checked", "starting
elsewhere", `RebootDirectoryMissing` on a `:1+`, `RebootCheckoutMissing` on a
`:0`). It is the one home of every row-facing next step -- the status
explanation, Enter's way forward (`Tab → reboot`) and the no-directory reboot
cost -- so each names only an action that row's kind can reach (a step taken on
another row says so: `OnPrimary`, `:1+` only); `TestRowAdviceNamesOnlyReachableActions`
sweeps every derived row shape and every confirmation it offers. Enter (`enterOperationFor`) switches a
live row and resumes a row that offers resume; every confirmation is re-checked
by one rule -- its row still offers its action, or that action is the one in
flight on that row (`menuFrameOperationInFlight`). A slot row's resume and
reboot send the host checkout as `path` and key their in-flight by row; reboot's
result may name a new tag, so it matches by attempt (`menuOperationReplacesAddress`)
and keeps its own frames while its row is replaced. `couchtty`'s sweeps
(`TestRowActionTableMatchesTheSpec`, the both-direction declaration sweep,
reachability and offered-implies-permitted) all iterate one derived domain,
`everyMenuRowShape`: kinds × `AllThreadStates` × `AllThreadReasons` ×
`AllPhases`. The console's continuation watch and focus-on-landing key on the
routed `resume`, which can return `ContinuationResult`.

Numbered directory preparation uses `couch --internal provision-workspace <primary>
--slot=N [--remote=R]` without taking a supervisor lease or launching a thread.
See [workspace provisioning](workspace-provisioning.md) for readiness, repeat-call
recovery and the #306 lifecycle integration boundary.

Continuation has five internal operations (`pair#249`, `pair#280`):

- `request-continuation`, invoked as `couch --internal request-continuation <absolute-path>`, durably accepts the hosted source's exact checkpoint. The inherited scope, tag, agent, session, launch ordinal, and expected digest bind publication to the writer's validated bytes and current source generation. This metadata operation can run in another worktree without becoming a second supervisor.
- `continue-thread`, invoked in process through `couch --internal continue-thread`'s declared operation, executes or reconciles an accepted request under the live owner.
- `retry-continuation`, exposed in the switcher's thread actions, reconciles a retained failure. After Couch has exited, `couch --internal retry-continuation <tag>` in the thread's repository acquires the normal singleton lease and opens a Console for recovery. It refuses a competing owner.
- `dismiss-continuation`, offered beside Retry on a row whose request FAILED, deletes the retained request. It records the operator's decision that the thread has moved on, typically because they took over the target before automatic orientation finished and a retry would re-deliver a stale handoff into a live conversation. It is a direct-store record write with no process effect and no supervisor lease. After or during Couch, the CLI form is `couch --internal dismiss-continuation <tag>`. It deletes rather than adding a terminal phase because records decode strictly (`DisallowUnknownFields`), so a new phase would make every pre-change binary, including long-running `pair` helpers, reject the whole record. Couch's private checkpoint copy is left for the next publish or archive to replace, and the repository's checkpoint file is never touched.
- `continuation-status`, represented by `couch --internal continuation-status`, reconciles the exact launch attempt's orientation receipt under the live owner. The Console supplies the address, request ID, and attempt through the typed operation arguments.

**A retained request composes with its thread; it does not replace it**
(`pair#280`).
- **State text, in every phase:** `<state> · continuation queued|continuing…|continuation failed`,
  for example `live · continuation failed`. In-flight phases are NOT bounded
  in time: a request whose owner died reads `continuing…` until someone retries
  it. So the state always shows.
- **Actions of a live row with a failed request:** the row's own actions minus
  exactly those `continuationGuard` refuses (`couchcore.ContinuationRefuses`:
  relaunch, switch-agent, cold resume, start), plus retry and dismiss. Park and
  detach stay, because neither reads the request.
- **Actions of an in-flight request:** kept restricted on purpose, because the
  continuation owns the thread mid-replacement. On a live row `Running` offers
  only `retry-continuation` (which reconciles a stalled request), and `Pending`
  offers nothing until it runs. A row that is not live offers resume (which
  routes the request to `RetryContinuation` or `RecoverThread`) and reboot.
- **Refusals:** every refusal a retained request causes names its exits
  through one wording, `checkpoint.Exits`, which names each row's own actions:
  retry/dismiss on a live thread, resume/reboot on one that is not. The guard and publish call it
  directly. Each other check wraps its refusals ONCE, at its boundary, in
  `withContinuationExits`: `RecoverThread`, `prepareAbsentContinuation`
  (through `admitRetainedRecovery`), `archiveContinuationVacant` and
  `validateContinuationWarm`. `TestEveryRefusalARetainedRequestCausesNamesBothExits`
  drives one row per call site and scans that every call site has a row.
  `pair continue --retry` does not know the phase and uses the phase-neutral
  wording (`Exits("", tag)`).
- **Orientation prompts carry their producer.** `menu.Orientation` is written
  by continuation delivery and by switch-agent's orientation watch. Each entry
  records which (`setOrientationLocked`), and a prune removes only its own
  producer's entry (`dropOrientationLocked`). A new switch-agent launch
  supersedes any prompt. Without that, the continuation scan deleted
  switch-agent's Copy orientation prompt on every tick.

Before #280, a failed request replaced the state text, the action set and
relaunch's admission indefinitely.
The switcher's Retry also never reached its thread: it sent `ref` and `tag`,
which `resolveOperationThread` refuses. Both continuation exits now address the
row by its exact implicit tag alone.

### Stale-thread recovery

`RecoverThread` reobserves the selected address and prefers warm attachment to
an exactly owned detached session; with no survivor, a retained checkpoint
starts a new conversation through the existing continuation executor. Since
#363 it is reached through resume's route (`ResumeRouteRecover`), not its own
switcher entry, and the explicit-checkpoint-path form has no caller left but
the live conformance test. Resume and reboot use the Console operation queue
and acquire the namespace supervisor lease from the CLI too; the CLI resolves
their repository scope from the caller.

`RecoveryDecision` is the common UI result shape. The snapshot projection adds
no refresh IO and offers inspection; execution gathers process-start identity,
exact session presence and detached ownership again. Unknown/active/ambiguous
observations and open start/park transactions refuse effects. Settled dead
helpers retire through `RetireIncarnation`, retaining `LastActiveAt` and never
fabricating `VerifiedPark`.

Source-gone checkpoint execution records explicit `SourceAbsence` authority,
including generation/revision proof. Legacy import may omit an unavailable
retired helper identity, but keeps the original checkpoint bytes, path and
digest. Actual park and absence authority cannot coexist. Recovery remains a
fresh conversation, distinct from native parked resume. A prior target's
attempt-bound readiness generation can authorize retry after that target is
proved absent; unrelated newer generations refuse admission.

Archive uses the same reconciliation, then checks that the continuation's source
and target are proved absent before quiescing.
A final record revision check prevents archiving a concurrently replaced
request. An empty record may be archived with its pending/failed continuation
intact; a live source or target cannot. The existing store journal preserves
the request in the archive and removes only its derived materialized file.
Disposable helper/session fault fixtures exercise warm recovery, checkpoint
recovery across worktrees, and the missing-checkpoint archive escape. Real
operator threads are not fault-injection fixtures.

### Continuation ownership and recovery

The continuation writer commits the document before requesting replacement and
passes its absolute path plus digest. `checkpoint.Checkpoint` validates a bounded
256 KiB UTF-8 document and stores its body, original path, and digest. The
revisioned ThreadRecord embeds one `checkpoint.Request`, including source
launch generation, request ID, attempt, phase, and process evidence. Repeated
publication of the same source and digest is idempotent. Warm attachment changes
the owning helper without changing the native pane's source launch ordinal.

The Console's one lifetime-bound worker observes only its hosted and accepted
request addresses; it performs no native-session scan on each poll. Accepted
requests survive removal of their source pane, keeping the recovery panel open
even when a failure and the last child's exit arrive in either order. The
existing operation queue owns process effects. Couch parks the exact source,
materializes the saved body at `continuation/<scope>/<tag>.md` inside its store,
and starts a fresh conversation through the existing blocked-helper claim and
registration protocol. The Pair scope and tag stay unchanged, preserving prompt
history. An unsubmitted checkpoint needs no native conversation binding.

Acceptance is not completion. Registration proves a fresh target exists;
`complete` requires the matching orientation `submitted` receipt. A failed,
canceled, or unconfirmed delivery remains recoverable, and text may already be
present in the target. Retry observes or reattaches an existing matching target
instead of automatically submitting again. Another fresh attempt requires proof
that the previous target is absent; unknown ownership refuses. Inspect the
existing agent and use the available copy-orientation action before manually
sending text whose delivery is uncertain.

The embedded snapshot remains authoritative if the original file is edited,
removed, or saved in a sibling worktree. It is retained through failure and
completion until superseded, and remains in the archived ThreadRecord. Archive
removes the derived materialized file. Active requests prevent unrelated cold
resume, agent switching, or relaunch from bypassing their ownership. Explicit
archive may retain an incomplete request only after proving its source/target
unoccupied; it never marks an unfinished request complete.
Hosted inner `pair restart` and address-changing rename routes refuse before
teardown; use Couch's tracked relaunch or name action. Standalone Pair retains
its outer restart-loop ownership and draft-seeding workflow.

`make test-couch-zellij-live` exercises continuation seed transport alongside
real park teardown. A deterministic pane under real Zellij reads the exact
materialized snapshot from the launch profile's orientation prompt and publishes
waiting/submitted readiness records. The production reader verifies session,
agent, tag, attempt, and live PID; registration alone cannot complete the request,
and deleting the session invalidates its receipt. This fixture uses temporary
stores and no paid agents. It uses the stateful fake for source parking and the
blocked launch helper; real composer recognition and actual agent submission
remain operator smoke tests. The conformance workflow runs on relevant changes
and weekly.

`relaunch` uses the current ledger resume target. A confirmed UUID or a requested
UUID under probation is sufficient for cold resume; transcript parsing does not
gate startup. An early Alt+n retries that target while current-launch observation
continues. A fresh Pair-chosen UUID whose root transcript has not materialized
instead restarts fresh with a new UUID. A fresh launch without any known UUID must wait for correlation;
ambiguous confirmed identities remain unavailable. See [Session identity](session-identity.md).

`Couch.ArchiveThread` (reached through reboot's `prepareRetirement` since #363;
`archive` is no longer a declared operation) is COMPLETE: it stops the thread's
zellij session first (`Artifacts.Quiesce` -> `zellij delete-session --force`,
polled until the session is verifiably gone), then removes the thread from the
working set and KEEPS its record, moving `threadstore/records/<scope>/<tag>.json` to
`threadstore/archive/<scope>/<tag>.json` and dropping the address from the
manifest in one journal entry, so a crash cannot leave a record in both sets or
neither. Restoring is that move reversed plus a manifest re-add -- `Snapshot`
walks the manifest, so a restored file the manifest does not list stays
invisible. Two layers refuse, and they ask different things (#256 M3). `Couch.ArchiveThread`
asks the classification (`ArchivableState`) whether couch is hosting the thread or cannot tell --
archiving a hosted thread would leave the console owning a record the store no longer lists, and a
hosted thread can carry no incarnation at all. The store asks only what a decoded record proves on its
own (`archivableRecord`): an open park or an outstanding start claim. Exact helper-death proof permits
reconciliation; unknown ownership still refuses destructive effects.

Park cannot do the stopping and that is why Quiesce does: park drives a
transaction through `PairLifecycle` and needs a live incarnation, which the
debris archive exists for does not have. Quiesce runs FIRST and its failure
refuses the archive -- the other order produces a record in the archive with a
live session behind it, which is the forgotten thread the action removes. It is
idempotent (nil when no session is bound), so a refused archive is safe to
retry.

What the archive does NOT touch is Pair's per-tag artifacts: the append-only
`repos/<scope>/ledger-<tag>.jsonl`, its `agent-*`, `config-*` and
`workbench-layout-*` files. That is deliberate -- it is what keeps an archived
thread inspectable, since the ledger still maps the Pair tag to every native
session id it ever bound. The ledger keeps ALL generations; only
`CurrentLaunch`'s projection is latest-only. Adding a typed operation cannot expose argv
without assigning a presentation. `DispatchOperation` validates a call and
invokes exactly one injected direct-store or live-owner executor; missing owner
capability returns the typed cross-actor routing refusal and never falls back
to a second process. No caller produces that refusal today: cross-actor routing
was punted with `pair#147`.

Switch agent (`pair#184`) uses the highlighted thread's action menu. The form
selects claude, codex, agy, muse, or qoder, then edits parameters loaded from the shared
starting-path preference. Confirming `switch-agent` revalidates the accepted
`prepare-switch-agent` fingerprint, parks the exact outgoing incarnation and
starts a fresh context at the same thread address and working path. Selecting
the current agent also starts fresh. Successful registration changes the path's
default agent and its argv; other agents' parameters remain available.

The outgoing Pair TTY archive is carried through cleanup completion into
`VerifiedPark`, using the exact collision-safe artifact token. Retry recovery
checks at most 32 prior completions, newest first, and honors cancellation; an
unresolved history beyond that budget fails explicitly before cleanup. The context
resolver captures the native session before park and adds readable supporting
transcript and sent-prompt paths. Missing evidence is reported in the generated
orientation prompt. The target renders and reads the TTY log, summarizes the
work and waits. No draft or operator-authored log is seeded by the switch.

Fresh registration requires a ready record matching the new launch nonce,
agent, session and process. Until then a failed launch cannot establish the path
default or authorize deletion of a session that raced for the same name.
`orientation-status` separately reports prompt delivery after adoption. The
wrapper owns paste and submit alongside normal input; operator input cancels
automatic delivery. Failed or uncertain delivery offers manual recovery without
resending. The panel retains focus for a panel-origin switch.

Start is a two-operation owner contract. Agent-facing `prepare-start` resolves
canonical path, selected agent/argv and provenance, preference revision,
repository-default digest, and repository identity into one explicit
length-delimited fingerprint. `start` then commits by fingerprint: it re-resolves
from the same inputs the preview used and refuses if the answer moved
(`ErrStartResolutionChanged`), so an operator never launches a resolution they
did not see. `StartResolution.CommitArgs` is the single owner of those inputs;
every caller renders them through it rather than restating the map.

The capability token that used to sit between the two operations went with
`pair#170` M4. A 256-bit one-shot grant with a TTL and a capacity bound defends
a prepared start against *another owner*; couch has none, so it only ever
guarded the start form against itself -- which the form's own armed-submit
identity already does. At-most-once now lives where the double-submit is, in
the reducer.

Acquisition of owner authority remains separate from the Console/PTY decision.

**couch hosts `pair` whole.** The stack is couch → pair → zellij → agent+nvim.
couch starts `pair resume <tag> --<couch's layout>` inside a child pty and owns
the operator tty until the console exits. Verified by operator smoke; the
alternative (couch absorbing zellij's role) was considered and rejected because
the agent child is never spawned by Go — zellij spawns it from a KDL layout, and
`entrypoint.ValidRootMarkers` *defines* a valid pair install as having those
layouts.

**Couch launch IS the console (`pair#146` M2).** It allocates a pty per child,
puts the operator's terminal in raw mode, and routes bytes -- so it no longer
hands the child its own stdio and blocks. The mechanism is shared with `pair term`
rather than written twice: `cmd/internal/ptychild` (a child on a pty, its
endpoint, bounded diagnostic capture and acknowledged output publication) and
`cmd/internal/hostty` (the operator's terminal: size, raw mode, coalesced
resizes). The escape sequences production writes to it belong to
`terminal.Presenter`, not hostty (#289). See [Terminal ownership](terminal.md).

Public launch requires terminal stdin and stdout before store, lease, or
actor work. The stdio runner remains an injected domain seam and live
conformance target, but is no longer selected by public argv.

**The pty is a CAPABILITY on a handle, not a second Runner signature.**
`Runner.Start` is unchanged; a handle from `PtyRunner` additionally satisfies
`TerminalHandle`. `ExecRunner`'s does not, and a test asserts that -- a
capability check no runner can fail is vacuous. `Terminal()` returns the
concrete `*ptychild.Child` rather than an interface, because `FakeRunner`'s
double IS one, so a test takes the branch production takes.

## The reserved row and terminal ownership

Couch gives its child one fewer row and composes the child publication with its
status row through `terminal.Presenter`. The endpoint interprets all child output
before presentation; child escape sequences never surround or interrupt chrome
writes. Child cursor, margins, modes, alternate screen and synchronized output
are virtual terminal state. The presenter alone owns the physical parent writes
and admits child input only after the complete selected frame is written.

Snapshots and typed normal history replace raw replay and resize nudges. See
[Terminal ownership](terminal.md) for bounds, decoding, effect policy and teardown.

**Placeholders** (`pair#206`). While the reattach pass runs, each pending
thread is drawn after the attached chips as a greyed placeholder
(`placeholderSGR`). Each thread starting now carries the spinner (up to the
pass's `Limit` at once, `pair#205`; drawn in attempt order), from the
`spinnerGlyph` table the switcher shares. A placeholder records no `ChipSpan`,
so it cannot be clicked, and the attached chips keep their columns. A thread
that attaches takes the column its placeholder held, because attached chips
are drawn in attach order. The spinner's tick is a Run-loop timer, armed only
while a thread is loading.

## Navigation

### Where input is allowed to go (#265)

Couch and the presenter are two state machines over the same question, and they
can legitimately disagree. `Focus` (couchtty) says whether the operator is
pointed at an actor or at couch's own panel. `View` (terminal) says whether the
presenter currently holds an admitted endpoint. The panel's healthy shape is
`State=Ready, Admitted="", selected=nil` -- `Presenter.Panel` clears the
endpoint on purpose -- so "no admitted endpoint" is a *description of the panel*,
not an error.

`terminal.ErrNoDestination` is how the presenter says that, and it is distinct
from a write failure (which latches `View` into `Failed` and closes `Failed()`).
Four sites answer with it: `Presenter.Input`, `Presenter.mouseInput`,
`Presenter.UpdateChrome` and `Presenter.resizeLayout`. The enumeration is "every
refusal that reports the ABSENCE of an endpoint", not "every caller of `Input`";
the by-caller reading is what missed `UpdateChrome` in planning and
`resizeLayout` at the close boundary.

The classification is `terminal.IsRoutingAnswer`, not an `errors.Is` per site:
the set has more than one member (`ErrNoDestination` and `ErrInputEnded`, the
latter because a child's input closes the moment its agent exits while the
console learns of that asynchronously), and widening one `errors.Is` at a time
is how this class survived five review findings. `ErrBackpressure` is
deliberately NOT a member — a full queue is a capacity answer, and dropping
input under load is its own decision.

Every console path to `Presenter.Input` goes through `deliverPresenterInput`,
which asks that question instead of handing the error to `terminalError` -- pinned
by `TestConsoleReachesPresenterInputOnlyThroughItsDoor`. Child-bound events
additionally go through `deliverChildInput`, which drops them when the panel is
focused. `paintNow` and `onResize` classify it too, because `showMenu` clears
the endpoint before it flips focus, and because `installObservedThreadActor`
sets focus to an actor WITHOUT selecting it when no pane is active. That second
state is DURABLE, not transient -- the operator sits on a pane the presenter
does not hold until they switch away -- so every drop is recorded on the
`no-destination` trace event. That is the only channel that can report it
without repainting, and repainting is what re-enters the escalation
(`publishNotice` is "push and paint are one operation").

This matters because couch *asks* the terminal for the events that exposed it:
the first mode delta writes `\x1b[?1004h` (focus reporting) and `\x1b[>3u`
(kitty flags 1|2, where flag 2 is "report event types", i.e. key release) on the
very first paint, panel or not. Before #265 those three kinds bypassed the panel
check, so a focus change or a key release with the switcher open exited couch.

`ErrBackpressure` is deliberately NOT in this scheme: it is a capacity answer,
still fatal, and changing that is its own decision.

`ctrl-space` is intercepted before the child sees it. It arrives in TWO
encodings and both are recognised: the legacy `0x00`, and CSI-u
`\x1b[32;5u` under the Kitty keyboard protocol, whose disambiguation Couch maintains -- so the
legacy byte is the one a real session almost never sends. The interceptor
returns a SPLIT (bytes for the focus being left, bytes for the focus landed on),
because a concatenated buffer cannot say which child the tail belongs to. It
suspends inside a bracketed paste: a pasted NUL that switched actors and ate a
byte would be untraceable data loss.

`ctrl-space` means one thing: **open the switcher**, from any actor, focused on
the actor with the latest notification -- or, with nothing pending, on the
thread being left, reconciled through `reconcileRootSelection` so a stale
`ActiveAddress` degrades to the first visible row rather than to no selection.
The child -> root-actor -> panel ladder and the root-actor/home concept are gone
(`pair#170`); `Up`, `Console.root` and `actorAlive` went with them, and #146's
Core-concepts contract was revised at its source rather than loosened. Inside
the panel `ctrl-space` still opens the global start form -- that is the panel's
own binding, not a rung of the deleted ladder, and it remains the only route to
starting a thread.

`ctrl+backspace` is **previous**, in both encodings: the legacy bare byte `0x08`
(a branch beside `couchkeys.SwitchLegacy`, since it is not an escape sequence) and the Kitty
`\x1b[127;5u` (an ordinary `knownSequences` row). In legacy encoding `0x08` is
`^H`, so ctrl-h is taken from the child too -- deliberate, and harmless under
the Kitty protocol zellij pushes. `panelkeys.go` computed a `modified` flag and
then ignored it for backspace, so the CSI-u form decoded as a plain backspace;
that is fixed as defence in depth, since the interceptor claims both encodings
before the panel sees them but forwards paste content verbatim.

`ctrl+return` **answers the newest page** (`pair#221`). From an actor it lands on
`attention.NewestActor()` -- the thread `ctrl-space` would have opened the
normal switcher on -- with no switcher in between. Focus-view filtering does
not constrain this jump. In code it is `ctrl+backspace`'s
mirror image: `onNewestPageHotkey` computes a target from console-local state and
calls `switchTo` directly. It does not dispatch the switcher's queued `switch`
operation, which would add the queue hop, a dependency on the inventory having
loaded, and `ctrl-space`'s first-visible-row fallback for a thread missing from
a stale inventory. The arrival is `arrivalNotification`, because the target is
paging by construction, and that is exactly what the switcher's Return derives
from a non-zero capture. So the jump is non-pinning: a second press answers the
next page, and `ctrl+backspace` still goes home. One test,
`TestNewestPageLandsWhereCtrlSpaceThenReturnWould`, drives both paths from one
attention state and compares the whole landing, `SwitchTracker` included.

Three edge cases:

- **Nothing paging.** It stays put and says so on the status row. There is no
  `ActiveAddress` fallback: a jump to where you already are reads as a dropped
  key.
- **Paging thread's child done, exit not yet reduced.** It refuses the same way.
- **The newest pager is the actor already in use.** Reachable only through the
  `focusedAtDelivery` race. `switchTo(..., force=false, ...)` acknowledges and
  stays, with no takeover. It also shows a notice, because the row never draws
  the active actor's bell, so the acknowledgement alone would be invisible.

The chord uses Kitty keyboard disambiguation (`couchkeys.NewestPageSequence`,
`\x1b[13;5u`); explicit press and repeat forms also jump, while release does
not. The presenter owns one keyboard-protocol stack entry per screen it presents on
and restores both at release (`atlas/terminal.md`, #279).
Child protocol negotiation stays in the endpoint and determines child input
encoding. Plain Return still reaches the agent; terminals without enhanced keys
retain Ctrl+Space then Return as the fallback. In the switcher, Return keeps its
panel behavior. Selection and input share one acknowledged presentation order.

`SwitchTracker` (`couchtty/switchrule.go`) is the whole rule: one `previous`
slot and one boolean carried on the CURRENT actor. `Console.switchTo` is the
funnel, and it owes two rules on every landing -- record it in the tracker, and
acknowledge the landed actor's pending notifications, because an actor does not
notify while the operator is attached to it. The rules key off `arrival`
differently: only a notification hop is non-pinning, but every landing clears
the bell. Two sites land without passing through `switchTo` and are handled
explicitly: the first attach seeds the tracker, and an exit `Drop`s rather than
records, because the operator lands on the panel and a dead thread must never
become the return target. Returning home twice is a no-op by construction, and
that is intended.

The stdin pump does not treat a `Read` boundary as an event boundary: after it
finds a hotkey, it waits for the Run loop to acknowledge the focus transition
before routing the suffix. The same stream rule holds for legacy Escape in the
panel — a bare ESC is held briefly because it may be the first byte of a split
arrow sequence; the Run loop's ambiguity timer turns it into an Escape key only
when no continuation arrives.

The hierarchical switcher is Couch's own single terminal surface. It owns input
while visible and suppresses background-child painting. The root lists only
actionable durable threads; Tab/Right opens the selected thread's actions,
Enter switches a proven live row or resumes a row whose conversation resolves, and
Escape/Left restores the preserved parent frame. Printable keys filter the
current list from memory. Breadcrumb plus one local banner identify nesting,
progress, validation failures, and operation errors without a second status
channel. Ctrl-Space opens the global path/agent start form from any list frame;
its cursor follows the active text row and its preview uses Pair's shared
token-bound preference resolution.

`start` and `resume` return a load-bearing `StartResult`, and `relaunch`
returns its own result carrying the same child. Adoption is therefore a PROPERTY
of the result (`couchcore.StartedChild`), not a concrete type: asserting
`StartResult` alone left relaunch's child spawned and never adopted, so the
record said live, the switcher rendered `live`, and no pane existed to switch
to. The separately declared typed `attach` operation must join that exact
terminal before success can select or land on it; attach failure aborts the exact newly started actor
and retains the form plus local error. Park, leave, reboot and alias use
the same declared operation surface. Each accepted slow action paints an
identity-owned spinner before dispatch, and stale completions cannot mutate a
replacement frame.

**Mouse ownership (#255).** Couch's parent presenter requests any-motion SGR
reports. It owns the status row and panel; the admitted child's endpoint receives
only the mouse events its virtual tracking mode asks for. An ongoing child drag
keeps that owner when crossing chrome. Switching, resize, EOF or teardown settles
the gesture once. Tracking and coordinate encoding remain separate endpoint
facts, so a legacy-mode child receives its requested encoding. The policy avoids
reasserting click-only mode over a child that needs drag motion.

The existing Ctrl+vertical-wheel product policy remains explicit in the typed
input adapter; it is separate from terminal mode ownership and tracked by #226.

A click dispatches the SAME declared `switch` (or `resume`) operation Enter
dispatches, chosen by the same `enterOperationFor` rule — one authority, because
a restatement had already diverged on its first day. The one difference is that a
click is always MANUAL: it suppresses the attention capture, so the landing is
`arrivalOrdinary` and `ctrl+backspace` undoes it even on a paging actor, where
Enter would be a non-pinning notification hop.

The shared incremental decoder recognizes mouse reports before product routing.
It bounds incomplete frames; raw read boundaries never authorize a partial event.

**A click maps to an ACTOR, and the geometry comes from the render** (`pair#172`
M1). `RenderStatusRow` returns `RenderedStatusRow{Body, Chips}`: each chip's
column span is recorded by the same pass that CLIPS chips to width, so a chip the
width dropped contributes no span and a clipped one contributes the columns it
actually drew. A caller re-deriving spans from `StatusModel` would agree at
comfortable widths and disagree at exactly the narrow ones, which is where a
mis-mapped click is least catchable by eye.

In the switcher the unit is the ACTOR, never the line: an actor occupies its own
row plus one per pending attention message. Notifications have `├─`/`└─`
connectors aligned to the owning row's label,
including numbered slots' indentation. Empty messages draw no row; the final
nonempty message gets `└─` before clipping. An oversized selected group keeps
its owner visible and clips trailing messages. `RenderMenuView` returns
`ActorExtent` runs derived from the `actorStart` boundary the scroll window
already uses. They are re-based there rather than in `renderRootMenuFrame`,
because the notice is inserted at index 1 and shifts every actor row down — an
extent computed before that shift is right by one line and wrong by one. Both
maps are TOTAL: a gap between chips, the breadcrumb, the notice, and anything
past the drawn rows are nobody, which is how "clicking bare space does nothing"
is a value rather than a branch at each call site.

The SGR decoder lives in `cmd/internal/mouseinput`, moved out of `termcmd` rather
than copied: one parser means one answer to "where does this sequence end", which
is the decision `#127`'s dead keyboard came from making twice. A caller that holds
on `IsPrefix` must bound the wait (`MaxReport`) — an unbounded hold parks every
following keystroke.

**A row states an age only when it has one** (`pair#187`). `LastActiveAt` was
written by park alone, so a thread that was DETACHED had never recorded activity
— and `now.Sub(time.Time{})` does not compute a large age, it overflows int64
nanoseconds and saturates, which the switcher then rendered as `detached ·
106751d ago`. Detach now records the time too (read once, before its
revision-conflict retry loop, so the value does not drift with contention), and
both readers ask `hasRecordedActivity` first: `rootStateText` omits the ` · <age>`
clause and `ageColor` paints nothing rather than claiming `AgeOld`. Only the ZERO
VALUE means absent — `time.Unix(0, 0)` is 1970 and must still render a real age,
which is what stops the guard being written as "very old implies absent".

`SelectResumableRoot`'s recency ranking deliberately does NOT guard: the zero
time is `Before` everything, so a never-active row cannot displace a better one,
and ties are deterministic because the projection sorts by `(RepoScope, Tag)`.
Correct as written, recorded here because the next reader would otherwise
re-derive it.

**The projection is TOTAL** (`pair#181`): every record in the manifest becomes a
row, and `ClassifyThread` returns a state plus, when the row cannot be acted on,
a `ThreadReason` from one closed vocabulary -- `binding-lost`, `session-gone`,
`never-started`, `invalid`, `unreadable`, `path-missing`, `profile-missing`,
`unsupported-agent`, `unknown`, `orphaned-server` (#399). (`stale-incarnation` and `unrecorded-child` were
retired by #256; see "Recoverability is a fact about the session" below.)
Failing closed is unchanged -- an unproved row is not actionable and startup
never selects it -- but it is expressed as a state rather than as absence. The
IO shell (`gatherThreadEvidence`) resolves evidence and decides nothing;
`ThreadEvidence` keeps "we asked and the answer was no" distinct from "we could
not ask" -- as a `ProofStatus` for the parked proof, and as
`SessionObservation`'s three-valued state for the session. Without that
distinction one failed zellij query would assert `session-gone` on every
detached row, and `session-gone` is a reason retirement acts on. `couch --list`
and `--show` classify through the same function over the same evidence, with
OS-derived liveness in place of the console's pty proof, so one store cannot
produce two stories. Ambiguous and legacy-unverified records now appear in both
views, named rather than hidden. Ephemeral console targets bind only to rows classified `live` --
couch's own hosting or an OS-vouched recorded process, never the record's say-so alone --
so a stale child handle cannot turn an inactive row's Enter into switch. If
Park removes the final actor while the switcher owns focus, the console remains
available for the refreshed resumable row. Lifecycle shortcuts are panel-only
(#245): Alt+x opens the typed `leave` confirmation in its park disposition;
Alt+d performs the detach sweep without confirmation. Individual thread actions
remain in the switcher. While an actor is displayed, those raw chords reach
Zellij and the receiving pane. No inner-pane focus cache or key-time query exists.
The agent consumes only Shift+Alt+T/Left/Right; Couch consumes its three navigation
chords. Couch's chords are declared once, in `couchkeys` (#282), each with a
scope: every pane (the navigation chords) or switcher only (the lifecycle
chords). The interceptor frames from that table, and `actorReserved` reads the
scope, so routing and the help's context agree. `couch --help` and Pair's
Alt+h page render the same `couchkeys.HelpSections`. Couch does not take Alt+h;
Pair's page shows Couch's keys when the attached client was launched by Couch.
`Interceptor` frames candidates, then Console routes the preceding bytes
before resolving focus and authorizing or forwarding the raw candidate. This
preserves ordering when a read contains navigation followed by a lifecycle key.
That confirmation is a **global frame** -- `menuFrameBindsThread` is false for
it -- because it names couch rather than a thread. It used to ride the root
actor's live address, so five thread lookups passed by accident; one of them,
`reconcileMenuFrames`, fires on the next inventory refresh rather than on a
keypress, so a keystroke-only test would watch the confirmation appear and then
vanish. With `leave` reachable from a couch with no live thread at all, none of
the five applies to it. Leaving is unconditional for the same reason: a switcher
holding nothing live must still have a way out, and making the exit conditional
on there being something to act on is exactly how the operator got stranded in
it. Any failure leaves Couch open and occupied for recovery (ARCH-PURPOSE).

The park trigger writes the exact typed quit intent and then deletes only the
indexed Pair/Zellij session. That deletion returns Pair's blocking handoff so
Pair can consume the intent and execute its shared full-quit cleanup. Couch
polls under a 15-second operation deadline for both the matching durable
completion and death (or PID-identity replacement) of the exact recorded Pair
child; completion alone never finalizes. Construction performs no active-park
session observation or reconciliation. After the live owner is composed, one
context-bound worker serially performs durable reconciliation plus external
Pair/Zellij recovery; blocked observation or teardown therefore cannot delay
startup or fan out across pending parks
(ARCH-PURE, ARCH-MOCK, ARCH-CONSTRAINTS).

There is no numbered jump or `:` command state. Colons and digits are ordinary
filter text; actions are discoverable from the selected durable thread.

A panel row carries three non-interchangeable addresses: `ThreadAddress`
(`{repo scope, tag}`) is durable identity, working path is a displayed/start
attribute, and the console-local child id routes terminal bytes and bells.
Filtering delegates to launcher's portable thread matcher; target joins and
selection use only `ThreadAddress`. Two Brain threads at one path therefore
remain distinct rows and cannot steal each other's local target.

Rows start at `Couch.ActionableThreadInventory()` plus exact Console-owned TTY
observations for live threads. Structurally eligible parked records additionally
require one context-bearing `NativeBindingResolver` result backed by session
inventory's exact established-root query; provisional, ambiguous, unbound, or
canceled resolution emits no parked row. Human name leads, the opaque tag is
the unnamed fallback, and
operator description remains separate from the agent-published summary. A
failed authoritative refresh preserves the complete last-good menu state and
renders the error locally; it never turns corruption into an authoritative
empty inventory or no-match result. A successful mutation remains visibly
refresh-pending until a successful actionable snapshot whose generation was
admitted after that mutation; a pre-mutation result may update last-good rows
but cannot present them as current. CLI `list` remains name-first over the raw
diagnostic inventory, while `show` always includes the full immutable composite
address.

## Switcher operating envelope

The primary UI is keystroke-critical: 100 actionable rows at 120x40 are the
supported fixture, with a 50 ms open budget, 16 ms filter/navigation/render and
refresh-apply budgets, and 100 ms first-progress budget. The committed
`BenchmarkMenu100` records all six paths and portable tests bound allocations,
input sizes, queue topology, and minimum 40x10 behavior. The opt-in
`TestMenuTargetPerformance` runs 20 warmups plus 200 samples for each path in
one baseline and two trials beside exactly four joined SHA-256 CPU workers on
the target M2 Max. No load process or unbounded goroutine fan-out is introduced
(ARCH-CONSTRAINTS).

## Exit, detach, and terminal lifecycle

Pair Alt+x and Couch Park share one typed full-quit cleanup implementation.
Couch persists a nonce-bound park transaction before publishing or triggering
the request; only a matching durable completion plus final ThreadStore CAS
removes the incarnation. Timeout, stale evidence, replacement, and child exit
remain occupied. Couch derives both Alt+x terminal encodings from Pair's
canonical chord table, renders confirmation first, and submits confirmed work
through the `PairLifecycleController`'s bounded worker (capacity
`LifecycleParallelism`, `pair#205`). Startup
recovery, Park, Retry, Recover, Abandon, and Leave all enter that same boundary;
same-address/same-nonce overlap shares one future; at capacity, other work
waits for a free unit (bounded by its ctx) rather than being refused.

**Bounded parallelism** (`pair#205`). One bound,
`couchcore.LifecycleParallelism`, is half the CPU cores and at least one. It
caps three things:
- **`Leave`'s fan-out:** quit detaches or parks that many threads at once. A
  failing thread does not stop its siblings, and the report keeps snapshot
  order.
- **The park worker.**
- **The startup reattach pass's in-flight set** (`ReattachPass.Limit`, fixed
  when the pass is armed).

The console drains its operation queue with `Limit+1` workers, so the pass
alone can never occupy all of them. Remote and continuation jobs share the
spare worker. Results still reach the console goroutine through `q.results`. Every
bound makes callers wait; none refuses on load. The actor registry is guarded
by `regMu` (`registry()`/`mutateRegistry()`), because writers no longer share
one goroutine.

**One lifecycle operation per thread** (`pair#205`, `couchcore/threadgate.go`).
An in-memory `ThreadGate` on `Couch` is held by every entry that changes a
thread's lifecycle:
- **Refuse** a held thread with `ThreadBusyError`, naming what is running:
  resume (all roads), relaunch, detach, `Couch.Park`, switch-agent, the three
  continuation entries, reboot, `RecoverThread`, `Stop`, and #399's `Reap`
  and `Recover` (which holds once for all its steps). The canonical list is
  `TestEveryLifecycleEntryRefusesAHeldThread`; a new lifecycle entry joins
  that table.
- **Wait** for the holder instead: the drains `Leave`, `RecoverActiveParks` and
  `AbortStarted`. `Leave` decides each thread from the record it reads *after*
  waiting.

Composites acquire once at the top and hand their context down, so the inner
entries re-enter instead of refusing their own caller. Re-entry matches the
hold's identity token, so a context that outlived its release cannot slip into
a later holder's hold.

Two existing guards cover what the gate does not:
- An **open park transaction is its own lock**: `Couch.Park` joins it (the
  worker coalesces by nonce) rather than refusing, and every launch refuses a
  thread carrying one.
- **`AbortStarted` touches the session by address only while the thread's
  incarnation is still its own.** The console runs it through
  `Console.GoTracked`, so a gate wait never blocks rendering.

A busy refusal reaches the console as `MenuEvent.Busy`, never as a resume
diagnostic. The reattach pass skips it silently, a refused continuation is
re-armed for the next scan, and an operator gesture shows it as a notice.

**Alt+n / Ctrl+Alt+n relaunch a thread from every pane** (`pair#182`,
`pair#284`). They are `couchkeys.ScopeEveryPane`: from a displayed Pair pane
`onRelaunchHotkey` targets the thread on screen, and in the switcher the
highlighted row. Couch replaces the helper with the current binary and keeps the
conversation. #245 had passed them inward, but Pair's own reload cannot run under
Couch, so Couch owns these keys outside the right terminal. The composition root
wires `rightTerminalFocusProbe`: resolve the exact thread's session, query
`list-clients`, and match the sole client's pane against Pair's live terminal
registry. Both restart chords then reach `pair term`, which applies its TUI
policy. Outer alternate-screen state is not inner focus evidence. The query is
bounded to one second and runs only for actor-focused restart candidates;
missing, malformed, multiple-client or failed observations display a notice
without forwarding or relaunching. The switcher needs no probe. Pair refuses an in-session restart before writing anything whenever
`launcher.CouchOwnsRestart` holds: the session env names Couch, or Couch presents
the client. That covers a session Couch presents but did not create, where a
Couch-launched client would refuse the marker only after quit cleanup had
already ended the thread. There is no whole-Couch relaunch; leave the switcher,
rebuild and run Couch again.

**Detach is available in the switcher** (`pair#170`, `pair#245`). Alt+d there
detaches all live threads; a thread's Detach action operates on that thread.
Use this route for durable Couch retirement. Pair's own draft/right-pane detach
only detaches its Zellij client. Detach is park's warm counterpart:
`Couch.Detach` SIGTERMs the actor's process group (never SIGKILL -- it does not
reuse `handleCleanup`, whose own comment calls that path rollback rather than
graceful shutdown), waits bounded for exit, proves the zellij session is still
there before AND after, and only then retires the incarnation by CAS through
`ThreadStore.RetireIncarnation` -- FinalizePark's removal half without the park
transaction, because nothing was torn down and writing a verified park would
claim a teardown that never happened. A client that ignores SIGTERM makes detach
FAIL rather than escalate; nothing was destroyed, so failing is safe. It needs no
confirmation at either scope, and that asymmetry with park is why both exist.
Detaching an actor moves focus to the switcher exactly as park does, which is
also what keeps Couch alive when the LAST actor detaches: an actor-focused
console exits with its final child, so without the focus move the safe gesture
would end the session.

**Relaunch replaces a thread's Pair process and keeps its agent conversation**
(`pair#182`). It is park-then-resume composed as one operation, and the design is
entirely in the ORDER: park is destructive and resume can refuse, so a relaunch
that parks and then finds the resume cannot run has traded a working session for
a cold one. Every refusal a check can see is raised BEFORE the park —
`CheckResumePreconditions` (the resume rules a park cannot change, shared with
`DecideResume` so the two cannot drift) plus `soleParkableIncarnation`, which is
park's own precondition and not one of resume's.

Four outcomes, and which state each leaves behind is the point:

| outcome | thread after | recovery |
| --- | --- | --- |
| `Relaunched` | one live incarnation, same address, same conversation | — |
| `RefusedBeforePark` | unchanged, still live | nothing happened |
| `ParkIncomplete` | OPEN park transaction; Pair already sent its quit intent | park's `retry`/`recover`/`abandon` — NOT `Enter`, which refuses `ResumeParking` |
| `park-ok-resume-failed` | verified park, no incarnation | `Enter` on the row |

Two preconditions cannot be checked early and are named rather than hoped over:
the native binding is validated against artifacts the agent is still writing, so
established-before does not imply established-after (a change lands in the
park-ok row, recoverable); and a Pair CLEANUP failure is not a park failure at
all — `CleanupAttempt` runs after the durable completion and the final CAS
(`park.go:642-643`), so its error lands in `ParkResult.CleanupError` while the
park returns nil, and relaunch proceeds.

**The axis that will otherwise be confused.** Pair's `Alt+Shift+N` restarts the
*conversation* and keeps the code; relaunch restarts the *code* and keeps the
conversation. They are inverses, and both exist.

`Leave` is the whole-couch form of that same pair, carrying a
`LeaveDisposition`: `LeaveDetach` applies `Detach` to every live thread,
`LeavePark` applies `Park`. Detach is the default and the safe one -- quitting
Couch does not kill a running agent unless the operator asked for park by name.
An unknown disposition is refused rather than defaulted, since guessing either
way silently contradicts the key that was pressed. A thread already mid-park is
driven to completion under both, and one carrying an `unknown` incarnation is
SKIPPED and reported under both -- Couch cannot vouch for that state, so neither
killing it nor claiming to have safely detached it is honest.

The detach sweep rechecks each recorded-live process by exact identity (#291).
Confirmed-dead bookkeeping is retired through `clearLifecycleDebris`, without
signalling, tearing down a surviving session, or adding a leave-report line.
Its saved conversation can still classify as parked. Unknown processes and
live processes without a session are preserved and reported as skipped;
observation failures and detach failures still stop the sweep.

**A failed start ends only what it created (`pair#230`).** Every failure after
a start's helper is acknowledged runs `quiescePostAckStart`, which ends the
helper and -- only when the start OWNS the session -- quiesces it, meaning
`zellij delete-session --force` plus a kill of that session's server. A warm
reattach owns nothing: it attached to a session that predates it, and that
session is running the agent the reattach exists to preserve. Deleting it was
the defect. Ownership is three-valued (`StartShape`: spawn, cold resume, warm
reattach) rather than a warm boolean, because spawn and cold resume already had
different durable tails and merging them would have changed spawn.

Two questions, answered at different moments. Whether the session may be ended
is `StartShape.OwnsSession`, consumed once by `quiescePostAckStart`; an
unrecognised shape answers no, so the failure direction leaves a session behind
rather than killing an agent. What happens to the RECORD is the pure
`DecideStartCleanup`, over `(shape, helper-dead, session-presence,
claim-vs-live-record)` -- rollback, retire, mark-unknown, or the pre-existing
reconcile tail -- and the session's absence after cleanup is one of its inputs,
which is why it cannot also decide the first question. Its whole input space is
table-tested, and the session is observed only where that answer reads it -- a
spawn reconciles regardless, so its cleanup asks zellij nothing. At the live-record phase every exit that has not retired the incarnation falls
through to the mark-unknown disposition as one structural fallback, so no path
leaves a live incarnation behind a dead helper. A failed rollback at claim phase
leaves only a claim, which `reconcileInterruptedStarts` settles on the next
startup. Two properties hold everywhere: a start never ends a session it
did not create, and nothing durable is undone while the helper is unaccounted
for. A warm reattach that fails therefore leaves its thread **detached and
reattachable**, using the same `retireDetachedIncarnation` rule `Detach` uses.
Ownership is read from couch's own registry record (`ActorRecord.Shape`), never
from a `StartResult` a caller relays back, which carries whatever that caller
believes about a start it did not make.

**Detached is a derived actionable state, not a persisted one.** `launcher`
already classifies a live zellij session with zero clients as `SessionDetached`,
and `pair resume` already reattaches onto one, so `ProjectDetachedSessions`
consumes that rather than teaching Couch a second way to ask. It fails closed
both ways -- two addresses claiming one session name, or two zellij rows sharing
one name -- and the projector's detached branch requires ZERO incarnations, which
is what keeps a crashed Couch's stale `IncarnationLive` from masquerading as a
clean detach. `DetachedSessions` takes **candidates** rather than returning the
whole set, because the session-name index is per repo scope. Each candidate
carries its address and saved agent profile; the observation adds its uniquely
owned live client-free session name. `detachedResumeProofMatches` is shared by
resume execution and its post-claim recheck (`pair#248`). None of these warm
paths resolves or requires a native conversation binding.

**The INVENTORY no longer calls it**: since #256 M1 the refresh asks session
PRESENCE — one host-wide `list-sessions`, no client counting — and the
candidate gate it used to pass (no incarnation, no verified park, a usable
profile and path) went with it, because deciding what to observe from the
bookkeeping is what hid a surviving agent behind a dead launcher. The
`list-clients` fan-out `pair#228` bounded now belongs entirely to the ACTION
path, where it is one call for the one thread the operator chose.
Before that it asked every live pair session, **measured at 1.49 s** on a
13-live-session host (2026-09-02). `list-clients` is the expensive call, about
250 ms against a real detached session, so the whole reattach path now asks it
of two sessions: the detached proof and its re-proof. `pair resume`'s launcher
and couch's registration poll read liveness only (`SessionLive`). See
`cmd/probes/reattachcost` for before and after.

**A cold resume's registration waits for the pane's birth first (`pair#287`).**
A cold resume starts a new zellij server, and zellij 0.45.1 panics when a
connection it accepted before the first client initialized the session closes.
The liveness poll is a `list-sessions` every 10 ms, which connects to every
socket, and under it every cold launch died
(`probes/zellijbirthrace -hammer 10ms`: 10/10).

- **The baseline.** While the helper is still blocked, and so before Pair can
  have touched anything, `coldResumeBirthBaseline` asks once whether the
  thread's session is already live. If not, it snapshots the thread's agent
  pane sidecars (`PaneBirthIO.PaneSidecars`, a `PaneMarks` of path → mtime).
  - *Already live* means no birth is coming, so there is nothing to wait for.
    A live but *attached* session fails the detached proof and reaches the
    cold path, and Pair refuses that resume. Waiting there would run out the
    deadline, and the cold-resume cleanup, which owns the session, would
    delete a live agent (#287 close review).
  - *Unobservable* fails the start before release.
- **The wait.** `awaitResumeRegistration` makes no session probe until
  `PaneMarks.BornIn` sees a sidecar appear or change, polled through the
  shared `panebirth.Await` (#288). It compares equality only, so no clock is
  involved. Neither the launcher clearing the sidecar
  nor a stale twin left by another agent counts as a birth. A failed
  observation mid-wait is "not yet": ending the wait would make the cleanup
  delete the session this launch just created.
- **What skips it.** A warm reattach takes no baseline: its pane was born
  long ago. A spawn and a fresh start already wait on in-session evidence
  (the claim, the ready record).
- **The fake.** It births a pane only on the edge a create launch produces:
  the session coming up (`SetPairSession`'s live edge), or an explicit
  `SetPaneSidecar`. A launch released against an already-live session writes
  none. An earlier version invented a birth there, and that hid the
  deletion path above.

Couch's menu refresh after a start completes stays a machine-wide
`list-sessions`. So do other threads' pollers. That residual risk has no
cure short of upstream zellij.

**A launch that dies at birth ends its helper (`pair#288`).** When such a
birth kills the server, the zellij client sometimes hangs instead of exiting.
Before #288 the launcher then waited forever, and the thread stayed live
with archive refused until Couch restarted.
- **Cold create.** Its claim is established before zellij starts, so the
  registration deadline never covered it. Now the launcher's birth watch ends
  the hung client about 10 s in (see architecture.md, "Launcher birth
  watch"), and the helper exits 1.
  - The pane closes with `exited (1)`, and the thread reads `session-gone`
    or `parked`. `ArchiveThread` accepts either and retires the stale `live`
    incarnation. No Couch state was added.
  - The launcher's own message, which names zellij's log, is not shown in
    Couch.
- **Cold resume.** Its wait still runs to its 15 s deadline even after the
  helper has exited at about 10 s, then rolls back to `parked` as before.

Where that lands differs by caller, and both matter:

- **Switcher refresh: not blocking.** Refreshes are event-driven and run on the
  single-flight worker while the menu renders its last-good projection, so the
  50 ms open and 16 ms keystroke budgets are untouched; rows simply converge
  later. Each query carries `zellijQueryTimeout`, because a hung zellij would
  otherwise wedge that worker and the switcher would render last-good forever
  without ever noticing.
- **Startup: blocking** (`pair#170` M3). `StartInteractive` must decide
  resume-vs-new before it attaches anything, so a detach candidate adds that
  cost before the first frame -- and `leave` detaching rather than parking makes
  a detach candidate the normal case.
- **Startup proves only the threads its readers consume** (`pair#206` M1).
  The readers of those rows are listed on `startupAsks` in `startup.go`, which
  is the list's one home. Each filters before it reads -- to this repository's
  scope (`pair#363`; the cwd's exact path before it), or to rows whose layout
  differs -- so `startupAsks` resolves exactly that union and leaves every other candidate
  `ProofUnresolved`, which classifies `unknown`: a row no reader here can act
  on. Those rows never leave `StartInteractive` (`StartResult` carries none),
  so unasked state cannot reach the switcher.

  The predicate gates `ResolveEstablished` as well as the zellij query, because
  each resolution reads that thread's own ledger -- narrowing only the
  `list-clients` calls would leave time-to-first-frame growing with the store
  while looking fixed. `TestNarrowedStartupAnswersAsAFullProofWould` computes
  the inventory both ways and asserts every listed reader agrees; a new reader
  joins that list and that test.
- **Every other detached thread comes back in the background** (`pair#206`
  M2). A start arms the reattach pass once its own thread has attached; a
  resume of one named thread does not. The pass is pure state in `MenuState`
  (`menu_reattach.go`), so the switcher's one transition authority orders it
  against every operator operation. Its specification is the transitions table
  in the #206 plan. The decisions:
  - **Seeded once**, from the first successful inventory: threads whose agent
    still runs behind a client-less session, plus threads whose proof could not
    be asked (`unknown`), minus the startup thread, most recently active first.
    The queue is never pruned or extended afterwards, because a failed refresh
    reads as "no sessions at all".
  - **One attempt at a time**, as a `resume` with the implicit `warm-only` arg.
    It re-proves the thread at its turn, and refuses one that stopped being
    warm, parked meanwhile or its session gone, before any effect. The pass
    skips such a thread silently. So it can only reattach an agent, never start
    one. Any other failure marks the row `reattach failed:` with its code, or
    with the error's first line when it has none. The row stays selectable,
    and resuming it by hand clears the mark.
  - **Behind the operator.** An attempt never takes the operator's in-flight
    slot, a background attach moves neither focus nor the tracker, and no
    background success steals focus.
  - **It yields.** The pass holds while the operator has an operation in
    flight, so a leave mid-pass is never followed by one more reattach. A
    successful leave ends it, and Stop cancels the attempt in flight and drops
    the queue, which leaves the rest detached.
  - **Pending rows are not ready.** Queued and loading threads show as
    placeholders on the reserved row, and as greyed `queued` or `reattaching…`
    rows in the switcher. The cursor, auto-select and clicks all skip them
    (`menuRowSelectable`). A thread the pass attached reads live until an
    inventory newer than its attach lands. Menu code reads rows only through
    `menuRows`, `menuThread` and `visibleMenuRows`, which apply that view, and
    `TestMenuCodeReadsTheInventoryOnlyThroughTheViewedLookups` fails any other
    read.

Resume accepts a **resolvable conversation** (cold) or **proved detachment**
(warm). It used to read the park receipt for the cold half; #256 M2 replaced that
with the ledger, because a receipt names a ParkIdentity and no conversation. A
detached thread has neither receipt nor need of one — nothing was torn down and
its authority is the surviving session. Both `DecideResume` (Enter's gate) and
`ProjectActionableThreads` (the switcher's list) carry that second authority --
widening only one would list a row whose Enter fails, or hide a row that would
have worked. A third gate used to sit between them, `ReconcileResumeAdmission`,
which re-checked fleet capacity before relaunching; it went with admission in
`pair#170` M4. Since #256 M2 the tombstone scan is no longer a VETO at all: it runs only where
nothing resolves, to say *why* in better terms than "unbound". Before that it
refused on any tombstoned entry with no break, and the detached branch had to be
checked first, because a thread once abandoned mid-park and later detached would
otherwise be permanently
unreattachable. `DecideResume` has not refused on an occupied incarnation since #256 M1:
the incarnation names a launcher that dies with couch, so that refusal contradicted
the `detached` classification it was fed. Resume rests on the two facts the classifier
uses -- a surviving session (warm) or a ledger that resolves a conversation (cold).
`DeleteStart` no longer deletes a record carrying a `LatestLaunchProfile`: the
verified park used to be the only rollback authority, and an unnamed detached
thread has none, so a post-claim failure would have deleted the agent and argv
needed to reattach while its session kept running.

**Resume**, continued: it atomically records a creating/start claim
on the same `{repo_scope, tag}`, reuses the exact saved working path, agent argv,
and established #155 native root, and read-only validates Pair's existing
established address marker. It rechecks that root immediately before child
effects. Verified park is cleared only after the exact Pair session registers;
ambiguous execution remains occupied/unknown. TUI Resume, alongside new-thread
`start`, is a singleton-owner operation: after a later Couch launch the
switcher resumes that exact thread and makes it the root console. It never
creates an intervening actor that would occupy the parked thread's address
(`ARCH-PURPOSE`, `ARCH-PURE`).

Interactive `couch [<repo>]` startup resolves the requested repository scope,
then applies `SelectResumableRoot` to the same proof-bearing actionable
inventory used by the switcher, over that scope's primary (ordinary-target) rows
at any working path -- one primary per repository (`pair#363`). It RANKS: detached
before parked, most recently active within each class, and starts a new thread
only when nothing matches. Inventory failure or a Resume refusal stops startup
without creating a fallback actor, and `startupResumeRefusal` wraps that refusal
with the thread it names and the ways forward -- no-fallback was right, refusing
mutely was not. The selector is bounded O(n) with no fleet scan or remembered
root identity (`ARCH-DRY`, `ARCH-CONSTRAINTS`).

**The ranking reverses `SelectUniqueResumableRoot`'s documented refusal to have
one** ("Preferring warm over cold would be a policy, and this selector
deliberately has none", `pair#170` M3). Exactness turned out to be a ratchet:
two resumable rows at one path were two matches, so startup created a third,
which guaranteed the next startup created a fourth. One repository reached six
threads that way. Warm beats cold because a detached agent is already running,
so reattaching preserves what it was doing; recency because that is the thread
the operator was last in, and a wrong guess costs one `ctrl-space`
(`pair#181`).

**A record couch cannot READ is a third answer, not a verdict.** `Snapshot`
carries such addresses as `ThreadSnapshot.Unreadable` rather than raising, so one
corrupt file cannot fail the whole inventory -- a store with 13 threads and one
bad record has 13 threads. They project as `unusable/unreadable`, which is
deliberately NOT `invalid`: a decode failure can mean the record is corrupt, or
that this binary is older than the store that wrote it, and conflating them
would make an old couch call every thread debris and offer to archive live work.
It is the same distinction `ProofStatus` draws about evidence, one layer down.

Unknown is also CONSERVATIVE. `PathHoldsUnreadableThread` blocks a start
anywhere in the repository scope of an unreadable record: reading it is what
would have supplied its working path, so it cannot be matched by path, and
treating it as absent would create a second thread in a tree that may hold live
work -- silently, where the old code failed loudly. An unreadable record CAN be archived by the operator -- that escape is what stops
a corrupt record locking its repository -- and `resolveThreadForArchive`
addresses a thread without decoding it so the gesture reaches the one record
class that most needs it. But archiving one never stops its session: the
classification that says couch is not hosting it needs a decoded record, so
quiescing would kill an agent on the strength of a record couch just failed to read. The archive
returns `ArchiveResult.Warning()` saying so.

Both projections take one `ThreadProjectionInput` (records + evidence +
unreadable). The three used to travel separately with the unreadable set as a
trailing variadic, which meant omitting it compiled cleanly and silently
restored "some records get no row" -- the regression the total projection exists
to prevent. One value makes the omission named and visible at each construction site;
`FromSnapshot` is the form that cannot forget.

`ScopeHoldsUsableThread` is the other half: **one primary thread per
repository** (`pair#363`, widened from one thread per path, `#181`), enforced at
the single site every creation entry funnels through (`spawnResolved`), refusing
a start anywhere in a repository -- a subdirectory of its primary checkout
included -- whose primary (`:0`, an ordinary target; slot rows are `:1+` and
never count) is live, detached or parked. Startup reaches the same rows first
and resumes instead, so the refusal is the start form's: "couch keeps one
primary slot per repository", naming the thread's label and a fresh step its row
can take (`primaryFreshStep`: `Tab → reboot` where `RebootableState` permits it,
otherwise `Alt+Shift+N` inside the live thread). Known debris does not block --
a repository whose only primary rows are unusable-but-classified must stay
startable (resolved ambiguity 9). A linked worktree that is not a slot has its
own scope, so its own primary. Couch starts only inside a Git repository:
`Resolve`'s `rev-parse --show-toplevel` refuses anything else before any record
exists (`TestStartInANonGitDirectoryRefuses`), so no non-Git row kind exists. An UNREADABLE record is the
deliberate exception, and it accepts the hazard this sentence used to warn
against ("one corrupt record locks its repo out permanently"): couch cannot tell
which path such a record holds, so it blocks the scope rather than risk a second
thread over live work. The lockout is bounded by naming the record's file in the
refusal, and by the switcher reached from another repository -- which is the
recovery path, and is stated in the refusal because an unstated escape is no
escape. In total version skew every record is unreadable and no repository
starts; the file path is then the only honest next step. Co-tenant primaries in
one tree from a store that predates `pair#363` still resolve and can be stopped
by actor ID. There is deliberately no opt-in flag: `StartArgs.SameTree` looks
like one and is documented as inert legacy serialization, so reading it would
resurrect a dead field as policy.

A row's label is `Label()`: a slot's `repo:N` (or `alias:N`), else the
repository alias, else `threadLabel` -- the working directory's last segment,
else the tag. The stored human name is never read (#363). `PresentThreads`
labels an ordinary row with its group's alias or scope-proven root name, which
is how a `:0` started in a subdirectory reads `repo` on the tab bar and in the
switcher alike; the console no longer transports pane labels through `Name`. `DisambiguateLabels` appends the tag's tail to
labels that collide, computed over the whole inventory rather than the filtered
view so a name does not change as the operator types.

Automatic startup never adopts two neighbouring states, though both are
listed. A session **attached elsewhere** yields no detached observation, so
couch cannot steal it. A record whose recorded helper is no longer hosted is **no longer a state of its
own**: since #256 the classifier does not consult the incarnation, so such a
record shows as `detached` when its session survived -- which is the case #272
was filed for -- and `session-gone` when it did not. Explicit recovery rechecks
helper/session ownership before effects (`pair#250`).

**Warm attachment and cold conversation resume use different evidence**
(`pair#248`). Warm access requires the surviving session; cold resume requires
the established native conversation binding. Warm success does not establish
that binding or promise transcript-dependent recovery. Foreground Enter and
startup preserve the selected detached row's intent with `WarmOnly`, as the
background pass already does. If the row becomes parked before execution, the
attempt refuses instead of creating a cold replacement. The final recheck
requires the same session name as the initial execution proof, then existing
tracked-start registration and cleanup deliver the helper to Console. Failed
warm starts never quiesce a session they did not create.

Zellij snapshot queries have a five-second per-query timeout. Query failures
propagate as errors, leaving inventory unknown and preventing execution; a
failed client count cannot become proof of zero clients. The exact Zellij
empty-inventory diagnostic remains an empty result. Contradictory warm proof
reports unknown, while binding-lost describes missing cold conversation proof.

Parked and detached candidates are physicalized alike, which the selector
depends on rather than merely benefits from: it compares paths by exact string,
so resolving one kind and not the other would match an alias path against a
parked row and miss an identical detached one — a bug visible only on a
symlinked checkout.

Every hosted pane retains three identities with separate jobs: the pty handle
routes bytes inside this console, `ActorID` addresses registry persistence and
notices, and the canonical worktree drives transitional human resolution.
They are not interchangeable: both real and fake runners mint a handle ID that
differs from the actor ID.

Each attached child publishes its own exit. If the focused child exits while
others remain, the operator lands on the panel; an inactive exit records the
cause without stealing focus. Either way the dead pane is removed and
`Couch.Forget` removes the registry-cache incarnation. A dead Pair client does
not prove its zellij session quiescent, so M1 retains durable occupancy until
#152 supplies whole-incarnation quiescence evidence. Exit and bell notices
share one bounded `Feed`
over `couchcore.Enqueue`: keys include the actor (`exit:<id>`, `bell:<id>`), so
repeated bells from one actor collapse while two actors remain two obligations,
and exit controls are never discarded for capacity.

Detach inside a live console means focus moved, not process stopped. The child
keeps running and filling its bounded replay ring; returning from the panel and
switching between children use the same clear-and-replay attach path. Beyond a
console process, warmth belongs to zellij's server session plus couch's forced
Pair tag: the console hosts a zellij client, so losing the client loses the view
and a new couch deterministically reattaches.

Console teardown has one owner. Normal stop, last-child exit, SIGTERM, and
SIGHUP all release the presenter, whose `parentReleaseControls` revoke
mouse/focus/paste/extended-keyboard modes, close synchronized output (a frame
write that failed mid-bracket would otherwise hold the display, #262), and reset
the scrolling region. Release leaves an alternate screen only if the presenter
entered one, and restores/shows the cursor, leaving the last frame's pixels in
place. Teardown then restores raw mode, stops host event sources, closes the
blocking input seam, and joins console workers before returning. This explicit reset is required because restoring termios does not
disable terminal-emulator private modes; otherwise mouse movement after Leave
types SGR reports into the returned shell.
`hostty.TerminationHost` is optional because couch consumes process termination
while the other `hostty.Host` consumer, `pair term`, owns lifecycle elsewhere.

## Spawning: `pair resume <opaque-tag> --<couch's layout>`

Every new start first atomically claims a final composite address
`{repo_scope, couch-<16 lowercase hex>}`. `CommitStartClaim` then performs, in
one revision-checked write, what admission used to do around its capacity
decision: clear the reservation, append the `creating` incarnation, and record
the start claim. It commits before the fork, so a resolution that drifted
starts no child. The creating record then gains one
`start-<16 hex>` nonce plus the exact supervisor identity. Couch forks the
internal `pair-launch-helper`, which cannot exec Pair until Couch durably adds
the helper's PID/process-start identity and sends one acknowledgement byte over
an inherited close-on-exec descriptor. EOF, cancellation, or timeout exits the
helper without starting any workspace writer.

After acknowledgement, Pair changes the same composite address claim from
`reserved` to `established`; that is the registration oracle, not PID liveness
or a successful pipe write. A Couch child may inherit the supervisor's zellij
ancestry; Pair lets that launch reach the claim check instead of applying the
ordinary nested-session rejection, but the exact reserved marker remains the
authorization gate. Only then does Couch clear the transaction and mark
the incarnation live. Any post-ack error before `Spawn` successfully transfers
the handle—including an acknowledgement error after the byte may already have
been delivered, registration read failure, promotion conflict, or legacy-
registry save failure—treats exec as possible. Both stdio and PTY runners make
the Pair client the leader of one actor-owned process group; Couch-launched
session-watcher and title-poller sidecars inherit that group rather than
detaching. Couch sends TERM and then unconditional KILL to the group, reaps the
client, and proves the group empty. The remaining process class is the zellij
server and its panes: Couch resolves the exact `{scope, tag}` session-name
binding, observes its record and exact server PID set, deletes and escalates,
then requires two stable observations with both absent. Query, deletion, and
kill errors fail closed rather than becoming quiescence evidence. Only after
whole-incarnation quiescence is proven does Couch reconcile durable state: an
unfinished transaction remains creating or becomes conservative unknown,
while an already-promoted exact incarnation is marked unknown. No error return
can leave an unowned workspace writer. A failed cleanup attempt does not return:
the start call stack retains the handle and one reusable wait-result channel,
then retries until it proves absence. Retry does not create another goroutine
blocked on the same process handle.
Server escalation carries PID plus kernel start identity and reauthorizes the
identity and exact server argv immediately before signalling.

On supervisor restart, the pure `ReconcileStart` decision
uses exact owner/helper identities plus that registration evidence: dead and
unregistered is proven free and rolls back by nonce+revision, established and
live promotes live, established but gone promotes conservative unknown, and
any unknown evidence stays occupied. The ThreadStore is therefore always
occupied or proven free across every interruption point.

Composite allocation and Pair artifacts share one durable address authority:
`thread-claim-<tag>.json` is created with O_EXCL before either Couch commits the
ThreadStore record or native Pair writes a sidecar/session binding. Couch writes
a reserved claim; only the child carrying the exact scope/tag establishes it.
That reserved → established transition writes and fsyncs a sibling temporary
file, atomically renames it, then syncs the directory, so concurrent recovery
readers observe one complete state and a crash cannot leave truncated evidence.
Direct Pair creates an established claim before its first artifact and adopts
historical tags into the same scheme. Collision detection uses the exact
structural tag boundary, while actual access goes only through
`artifactpath.Paths`; no consumer scans its way to a selected file. The session
binding index now lives in the same selected repository scope; strict reads
merge the former global file for upgrade compatibility, and malformed or
unreadable present state fails closed.

The child receives `COUCH_TREE`, `COUCH_STORE_DIR`, `COUCH_THREAD_SCOPE`, and
`COUCH_THREAD_TAG`, and launches as `pair resume <opaque-tag> --<couch's
layout>`.

`COUCH_INPUT_TRACE=<path>` (`pair#182`) is an env var couch reads
for ITSELF rather than passing down: it appends every operator keystroke couch
receives to that file. It exists because "the chord had no effect" has two
indistinguishable causes — couch consumed it and dispatched nothing, or the
terminal never sent the bytes couch watches for — and only the wire separates
them. It is non-visual by necessity (the console hosts a child terminal, so a
probe that painted anything would corrupt it), off unless set, and the file is
created 0600.

**Read what it captures before enabling it.** The tap is in `pumpStdin`, BEFORE
the Interceptor splits anything, so the file holds everything typed or pasted
into the hosted agent — prompts, pasted secrets, credentials. It is a debugging
instrument for a session you own, not something to leave on. If it cannot open
its file it says so on the status row at control priority rather than tracing
nothing: an empty trace would otherwise read as "no bytes arrived", which is the
exact ambiguity the probe exists to remove.

`COUCH_TRACE=<path>` (`pair#206`) is a timing trace of startup and the
reattach pass. It writes one line per event,
`<unix-ms>\t<event>\t<scope>/<tag>\t<detail>`, with `-` for an absent field.
The events:
- `startup`, stamped with the process start;
- `first-frame`;
- `inventory`, with `rows=N` or `error`;
- `pass-seeded`, with `pending=N`;
- `slot-git` (`pair#317`), with `ok=N failed=M` for one slot quick-status pass
  that probed at least one checkout;
- `activity` (`pair#247`), with `ok=N failed=M` for one idle-fading activity pass
  that probed at least one live thread;
- `reattach-start`, with `attempt=N`;
- `reattach-done`, with `ok`, a resume diagnostic code, or `error`;
- `no-destination` (`pair#265`), with the abandoned operation and the
  presenter's refusal: `panel`, `input`, `chrome` or `resize`. It records a drop
  that has no other channel — a notice would repaint, and repainting is what
  re-enters the escalation `#265` removed. Its detail carries the refusal text
  rather than a bare code, unlike `reattach-done`; the text is a static reason
  plus `%q`-quoted endpoint ids, which the TSV framing permits because `%q`
  escapes tab and newline.

`TestAtlasNamesEveryTraceEvent` pins this list against `trace.go`, so a new
event cannot ship undocumented (`pair#265` BR-8).

Unlike the keystroke trace, it records addresses, counts and timings, never
content. The traces write through one `traceFile` (`trace.go`): opened 0600,
at a path the composition root passes in, and reported on the status row when
it cannot open. `PAIR_PROBE_SAMPLE_SECS=N make test-reattach-cost` samples
`zellij action` latency and prints its window in unix ms, so the sampler's
output lines up with the trace.

**Crash files** (`pair#397`, `cmd/internal/crashreport`). A Go panic writes its
stack only to stderr, and couch's stderr is the terminal it redraws over, so the
console-owning couch also routes its crash output (`debug.SetCrashOutput`) to
`<couch store>/crash/<UTC yyyymmddThhmmssZ>-<pid>.log`. It installs this right
after taking the singleton lease; the lease is what proves every older file there
belongs to a dead incarnation. CLI invocations don't install it, since their
stderr is readable. On a clean exit `cmd/couch`'s `main` calls
`crashreport.Finish` after `Run` returns, which removes the empty file. This is
never done from a defer: Go runs defers while a panic unwinds, before the runtime
writes the panic, so a deferred close would delete the file empty (BR-1). On the
next start, files whose pid is still alive are skipped (a dying owner drops its
lease before its panic is written), and every previous ending is folded into one
standing notice:
- a non-empty `.log` is a crash: a control notice
  `previous couch crashed — see <path>` stands on the status row, and the file is
  renamed `.crash`, so it is reported once;
- an empty `.log` means the process died without a panic (SIGKILL, power loss, the
  memory killer): notice `previous couch ended abruptly (no panic recorded)`, and
  the file is removed. SIGTERM and SIGHUP shut down in order and don't count.

`pair gc` ages crash files out with the diagnostics retention period through its
own sweep of each registered store's `crash/` (`gcruntime.crashRows`). The
inventory walk excludes stores, and the runtime writes these files outside any
`diagnosticlog` writer registration, so neither existing path reaches them.
Except for matching the scope/tag to establish Pair's reserved address claim,
Pair treats these Couch-owned values as opaque pass-through context for the
hosted child: it does not resolve Couch names or paths and never reads or
mutates Couch's manifest or records.
Distinct starts at one path therefore use distinct Pair
sessions and artifacts.

**Layout is couch-wide and never mixed** (`pair#198`, reversing the 2026-08-22
pin). `couch` defaults to `--layout3`, giving every thread pair's own
right-hand terminal (`pair#242`); `--layout2` explicitly opts out. It is a property of the couch PROCESS,
chosen at startup and immutable for its lifetime -- not a per-thread setting.

Two rules carry it:

- **The flag reaches argv only at a COLD boundary.** A warm reattach sends no
  layout flag at all, because a running session already has its layout and
  asking for a different one sends pair down a path that offers to DELETE it
  (`pair#179`). This is the safety property; the guard below is not.
- **A startup guard refuses to mix.** `ThreadRecord.Layout` witnesses what each
  thread's session is, and couch refuses to start when a thread already holds a
  session in the other layout. The blocking set is the states that hold a
  session -- live, detached, busy. A *parked* thread never blocks: park ends its
  session, so its next cold resume takes couch's layout freely, which is what
  makes "park them first" a reachable remedy rather than a dead end.

A record written before `#198` has no layout field, and those are layout2 with
certainty; `ProjectActionableThreads` normalizes them. An unreadable value
becomes `LayoutUnknown`, which conflicts with every request rather than
defaulting to something the guard would trust.

The pin this reverses read "couch owns terminal switching, so layout3's third
pane is the layer couch replaces" -- an actor-cluster-era claim that `#170`'s
rescope to couch-lite invalidated: couch-lite switches agent sessions and never
took over handing the operator a shell at their cwd.

`ResolveLaunchProfile` keeps two provenance axes independent. Agent precedence
is explicit start selection → the path preference's `last_agent` → the root
actor's `$PAIR_AGENT`; argv precedence is that selected agent's path entry →
its Pair-owned repository default. Agent choices derive from
`launcher.AgentInventory`, so Couch has no harness enum and can never apply one
agent's argv to another.

Primary and ordinary-path preferences are strict revisioned records below
`threadstore/path-preferences/`, addressed by a digest of normalized repository
identity plus canonical physical path while retaining both values in the
record for validation. Numbered workspaces route that same API to their
`<environment>/.couch/preferences.json`, keyed to the nested main checkout;
sibling dependency clones are not enrolled as threads. `repoLaunchDefault`
resolves numbered workspaces' missing per-agent values from the primary checkout
across create, fresh and Switch agent. It retains the explicitly known primary
root during first creation before enrollment. First-use agent selection remains
the current Couch agent, not a copy of :0's preference; registered launches then
establish independent per-workspace values. Resume consumes its incarnation's
exact recorded profile, while fresh replacement preserves per-agent history and
validates parameters before replacing the current record. The resolved profile travels to Pair as a strict
tag-bound `PAIR_COUCH_LAUNCH_PROFILE`. `PAIR_USE_REPO_DEFAULT=1` accompanies it
only for matching repo-default provenance; path provenance supplies one
authoritative empty value. `ExecRunner` overlays supplied child keys after
removing inherited duplicates, so stale launch policy cannot cross the process
boundary. Pair consumes both keys before launch and does not persist
Couch-resolved argv back as a new repository default (`AgentArgsFromCouch` guards
ordinary launches as well as fresh/resume paths).

The pending start claim carries the exact profile across Couch failure, but it
does not count as history. Established registration promotes that profile onto
the incarnation and journals the thread record, per-path/per-agent history, and
manifest generation as one recoverable transaction. Failed fork,
acknowledgement, or registration paths write neither preference. A restarted
Couch therefore selects the last successful agent and exact argv at that path
without reopening Pair's saved-config picker (ARCH-DRY, ARCH-PURE,
ARCH-PURPOSE).

## Couch metadata and resolution

Name, operator description, and agent-published summary are independent mutable
fields on the revisioned ThreadRecord. `couch --internal publish-description` is run by a
session with its exact `$COUCH_THREAD_SCOPE` and `$COUCH_THREAD_TAG`; it cannot
resolve a mutable path/name or overwrite operator prose.

`cmd/internal/threadrecord` owns the persisted Couch record wire schema, strict
structural validation, and persisted address/generation checks. Couch alone
reads and mutates those records through ThreadStore. Its inventory, human-name
and path resolution, metadata edits, and lifecycle transitions all
stay on that authority; none are projected into standalone Pair.

Pair independently owns exact scoped tag claims, sidecars, ledgers, public
session bindings, and its tag-only resume/picker flows. Pair's strict claim
decoder rejects duplicate keys, unknown fields, malformed identity, and invalid
states, but that marker is only the Couch↔Pair registration handshake—not a
second metadata store. `SessionNameEntry` remains only Pair's stable zellij
socket binding; Couch's mutable human thread name or working path never renames
that socket, decorates Pair's picker, or becomes valid `pair resume` input
(ARCH-DRY, ARCH-PURPOSE, ARCH-PURE).

## Identity

The durable address is `ThreadAddress{RepoScope, Tag}`. `RepoScope` is Pair's
existing hidden repository scope; `Tag` is the opaque Pair thread tag. The
canonical physical requested path is an attribute, not the thread identity, so
Brain-style repositories can host several independent threads in one directory.

**Admission is gone** (`pair#170` M4). It normalized Ariadne's versioned
`sdlc fleet policy --path P --json` result into a per-incarnation capacity
decision, reconciled cohorts under compare-and-swap, and refused starts over
capacity. Capacity and incumbency across a *fleet* is the multi-owner case
exactly, and couch-lite is one operator on one host: the whole subsystem, its
cross-repo provider dependency, its stateful fake and its live conformance
target went together.

One field survived it. `advanceSuccessfulStart` keyed the path launch
preference by the policy record's `repo_identity`, which is just the Git common
directory -- so `ThreadIncarnation.RepoIdentity` now carries it, resolved
locally through couch's own `GitRunner` seam. The value is byte-identical, so
every existing `path-preferences/` file stays readable and the operator's
per-path agent+argv memory survives the deletion. The old `policy` object
remains as a decode tombstone.

`Worktree`, `ActorID`, and `registry.json` remain transitional live-console
data. Working path is a start/display attribute and `ActorID` identifies one
registry incarnation; neither selects a durable row or addresses Pair
artifacts.

## Seams

Everything touching the world is injected, so the domain tests without
processes, disk, wall-clock or randomness. The seam set itself lives in
`Couch`'s struct fields in `cmd/internal/couchcore/couch.go` -- read it there
rather than from a list here, for the same reason the operations are not
enumerated.

The property that matters: each seam has a fake, and the fakes that model
*behaviour* rather than data are compared against the real thing by
`conformance_live_test.go`. `PAIR_LIVE_COUCH=1` checks process/git/pty behavior.
(`make test-couch-policy-live` checked couch's policy consumer against Ariadne's
real provider; it went with admission in `pair#170` M4, along with the weekly
workflow that ran it.) The process check found a
real bug -- `Alive()` reporting a zombie as running -- which no test against the
fake could have. `TestSessionQuiescenceLive`, run by both `make test-live` and
the focused `make test-couch-zellij-live`,
creates and deletes an ephemeral real zellij session through the production
observation seam and explicitly requires real server discovery, session-delete
dispatch, and an underlying OS kill dispatch against an exact-argv sentinel
that ordinary zellij deletion does not own before accepting verified absence.
A separate macOS workflow runs it on relevant changes and weekly/manual cadence.

`Runner` was genuinely new — pair has no async process-exec seam.
`launcher.ProcOps` is named for pair's own sidecars, and `wrapcmd` spawns its
child inline and unseamed.

## Recoverability is a fact about the session (#256)

**The classifier reads the world, not couch's bookkeeping about the world.**
`ClassifyThread` does not consult `Incarnation` liveness fields or
`record.Park`.

The measurement it rests on: the zellij **server is PPID 1 at birth** — it
daemonizes, it is not reparented — so `pair wrap`, `pair term`, nvim and the
agent are its children, not couch's. `Detach` SIGTERMs the **launcher**, which is
couch's own child and dies with couch anyway. Therefore:

> A clean `alt+d` detach and a couch crash leave **identical external state**.
> The only difference is whether the bookkeeping ran.

Before #256 that difference decided everything: with the record, `detached`
(recoverable, ranked highest at startup); without it, `stale — helper ownership
unresolved` (debris). Measured on the operator's store, all 11 records carrying a
`live` incarnation had a dead pid and three had an agent still running.

### What durable state earns its place

Durable state is justified only when it records something **not derivable from
the world**:

| Kept — not observable | Not read for classification — a shadow of what you can look at |
|---|---|
| native session id (the conversation) | `Incarnation{PID, Identity, State}` |
| address → session-name binding | `ParkTransaction{Phase, Attempts, …}` |
| launch profile, paths, name | — |

Neither field is deleted from the record; they stop being **read** by the
classifier. Replacing the park transaction itself is `#275`.

### `SessionObservation` — four values, not a boolean

`couchcore/sessionevidence.go`. `SessionUnresolved` is the **zero value**, so an
observation nobody populated fails closed: if absence were the zero value, a
gather branch that silently stopped running would assert "no session" for every
thread it skipped — the anonymous refusals `#181` removed. `session-gone` is
archive-eligible, which is what makes the distinction load-bearing.

**`SessionOrphaned` (#399)** is the fourth value: the session's exact zellij
server is alive but its socket is gone. Nothing can reach it, `list-sessions` no
longer lists it, and its agent may still be writing. On 2026-10-06 a test deleted
`$TMPDIR` and orphaned all 20 servers; the inventory read them `parked`, offered
`resume`, and resume failed with a raw `list-panes` exit status.

| Fact | Where it is read | Rule |
|---|---|---|
| server argv `zellij --server <socket>` | `launcher.ParseServerProcesses` | one parser; the socket's base name is the session |
| socket state | `launcher.ObserveSocket` | only ENOENT is `gone`; any other Lstat answer is `unknown` and never makes an orphan |
| per-session verdict | `launcher.ClassifyServers` | lone server + gone socket → orphaned; unknown socket or two servers for a name → unresolved |
| refresh evidence | `SessionPresence` + `launcher.ServerStates` | one `ps` + one Lstat per server per refresh; consulted only for names `list-sessions` doesn't report live; a snapshot error fails the refresh closed |
| one session's owner | `SessionOwnerProbe.Probe` | asks the socket before `list-panes`; gone → `SessionOwnerOrphaned` |

The classifier turns it into `unusable/orphaned-server`, ahead of every
"no session" reading and the record's own faults (an orphan is a running process
whatever the record says). It is not archivable, rebootable or resumable. Rows
carry `Orphan` (pid, session) so every surface prints one sentence,
`launcher.OrphanDiagnostic`: "<session>: server PID N lost its socket — Tab →
recover". Resume refuses with `resume-orphaned-server`. Startup refuses rather than
starting a second primary beside it (`ScopeHoldsOrphanedThread`; unusable rows are
otherwise debris to the one-primary rule). The recovery report shows agent
`orphaned` with its server; a lone orphaned row's steps are `couch --reap repo:N
--confirm` then `couch --resume repo:N` (never reboot, which would archive a
running conversation), and an orphan among several threads holds the slot.

**The live orphan** (2026-10-07 acceptance): a thread Couch still hosts keeps
working over its open connection when its server loses the socket, so
`ClassifyThread` keeps it `live` — but `orphanOf` sets `Orphan` on it too (never
on busy or unknown rows). `ActorActions` offers `reap` on any row carrying
`Orphan`, `agentOf` reads it as `orphaned` (steps reap → resume), `Couch.Reap`
admits it, and the switcher offers `[recover, reap]` instead of detach, relaunch,
park and switch-agent, which would all refuse an orphan. Reap and recover are in
`endsItsOwnChild`, so the hosted client's exit is expected, not a notice.

**`reap`** (M2) is the confirmed operation that ends an orphaned server's tree,
offered by `ActorActions` (only) on an orphaned row, so the switcher shows it
too. `Couch.Reap` admits a row only if the orphan verdict holds again for the
same server identity a second later: a starting server is in `ps` before its
socket exists. `launcher.PlanReap` orders one snapshot of the tree, deepest
descendant first and the server last; `launcher.Reaper` sends SIGTERM in that
order, SIGKILLs survivors by pid after a bound (a child that reparented to PID 1
is still found), re-reads every pid's start identity before every signal, refuses
a changed server before any signal, and names a SIGKILL survivor instead of
waiting. `launcher.OSOrphanReaper` then ends the thread's helpers that live
OUTSIDE the server's tree, the title poller (whose parent is the launcher, Couch's
hosted client) and the editors, through the quit path's own pidfiles
(`ReapTagHelpers`), and finally proves zellij's leftover EXITED record gone with
the shared quiescence loop. The snapshot reads pid, parent and start identity in
one read (`procutil.Table`). Order matters because on 2026-10-06 the servers were
killed first: `pair wrap`, which ignored SIGTERM, survived, and the launcher-owned
`pair title` pollers were left at PPID 1.

**`recover`** (M2, `couchcore/recover_action.go`) is the switcher's default Tab
action, offered wherever `ActorActions` offers anything. It runs exactly the
actor steps the recovery report computes for that row — `[resume]`,
`[reap, resume]` or `[reboot]` — or refuses with the report's hold (a typed
`RecoverRefusal`, no effect); a thread no report row stands for falls back to
the same rule over `ActorActions`. It never asks (`ConfirmNone`, operator
2026-10-07): choosing it is the consent, since within its envelope it only stops
a server nothing can reach or reboots a conversation that cannot be resolved,
and never changes the slot's files. `reap` alone keeps its confirmation.
`Recover` runs each step through `DispatchOperation`, re-reading the row before
each and waiting briefly (`recoverSettle`) for it to admit the next step — a
reaped live orphan's hosted client exits a moment after the reap; the last
step's result is returned unchanged so a resume's child is adopted as a plain
resume's is. `couch --recover repo:N [--json]` reaches it through the
slot-operation socket.

There is deliberately no "held elsewhere" value. The refresh never counts
clients — `list-clients` costs ~250 ms per live session (`#228`) — and the
reattach path re-observes attach state before committing. **Optimistic inventory,
strict action:** the expensive question is asked for the one thread the operator
pressed Enter on, so cost is proportional to what you *do*, not what you *have*.
`DetachedSessions` remains the action path's authority, guarded by
`RequireAttachState`.

### One class, four sites — and one function (M2)

A guard reading bookkeeping the classification no longer trusts is one defect
with several homes. Each was found by fixing the one before it — which is the
point: the enumeration is the deliverable, not any single site.

1. `ClassifyThread` — session-first.
2. `DecideResume` — stopped vetoing on the incarnation and the open park, or a
   row the switcher advertised as `detached` could not resume.
3. `CommitStartClaim`'s caller — **re-adoption**: retire the dead launcher's
   incarnation before claiming a new one. One-incarnation-at-a-time is a store
   invariant, not a lifecycle opinion, so the *caller* clears it — gated on
   confirmed `Dead`, never on an unobservable process, since retiring a live one
   would abandon a running agent.
4. `RetireIncarnation`'s open-park precondition, which became reachable **because
   of** site 3. An orphaned park is abandoned alongside the dead incarnation, and
   every precondition is screened before that write — `AbandonPark`'s tombstone
   is permanent, and a failure after it leaves a thread that can be neither
   resumed nor archived.

The sweep is written as a **predicate**, not a list: *every guard refusing on
`record.Incarnations` or `record.Park`*. Writing it as four sites is what let two
further shapes through — a park owned by a process that is not the incarnation,
and an open park with **zero** incarnations. Both are the same
`replacementUnknown` escape in `threadrecord/lifecycle.go`, read at different
incarnation counts, and both wedged `couch` in the whole tree. The clearing pass
is therefore **total over the shapes `validateLifecycle` accepts** — which is the
domain, because ARCH-SECURE treats a record written by another version as
untrusted input — and `TestReAdoptionExitsAreTotalAndCoded` enumerates it.

Four rules outlive the sweep:

- **An irreversible step never precedes a revocable check**, and its precondition
  is proved about **the exact entity the step acts on**. The park's owner and the
  incarnation's process are not always the same process
  (`TestForeignOwnedParkIsRepresentableAndRefused`).
- **A guard omitted as "unrepresentable" cites the validator clause that makes it
  so, read including its exceptions**, and is pinned by a test that builds the
  fixture through the real store. That test exists here because the claim was
  made twice and was wrong twice.
- **Guidance belongs at the consumer that needs it**, not as a marker every
  producer must carry: `startupResumeRefusal` decorates any failure. An earlier
  attempt at the latter changed what `ResumeDiagnosticCode` *meant* — from "is a
  structured refusal" to "came out of resume" — and broke every reader that used
  the distinction.
- **A diagnostic code is a claim.** `ResumeNotRunning` must not be emitted where
  couch could only establish ignorance, or the switcher renders "not running"
  over a live conversation.

`hasOccupiedIncarnation` survives for `relaunch` and `switch-agent`, which ask a
different question: not "is this recoverable" but "is couch itself already
operating on this thread", where couch's own record *is* authority.

**M2 gave the sweep one home.** `clearLifecycleDebris`
(`couchcore/lifecycledebris.go`) is that rule as a function, and **archive calls
it too**. The alternative — re-deriving it inside `DecideRecovery` — would have
been the same rule in a second place, which is the shape that produced four
rounds of findings in the first place.

Nothing was deleted to get there. `DecideRecovery`'s park and incarnation-shape
gates and `archivableRecord`'s unfinished-transaction rule each protect a real
precondition downstream, so they stay; they stop being the **operator's** wall
because the debris is gone before they run. (M3 narrowed `archivableRecord`
itself: the occupancy half moved up to `Couch.ArchiveThread`, which classifies.) The operator's second wedged row needed no new
rule at all: the cleared record reaches the reconciler carrying no incarnation,
which the existing binding-absent hatch already admits.

Three processes can be named by one record — the park's owner, the start's owner,
and the helper the start forked — and any can outlive the others, so each gets
its own probe. A start claim **rolls back** rather than retiring, because
`RetireIncarnation` takes only a live incarnation; when that rollback removes a
husk carrying nothing else, `ErrThreadRolledBack` reports the fact and each
caller decides what it means (resume: a refusal; archive: success, the row is
gone).

**Order, in archive, is the guard.** Clearing debris is a durable write, so the
session — the one refusal that is nobody's fault and may change on its own — is
asked first, through `RecoverySessionRefusal`, with nothing written. Getting this
backwards was a regression caught by an existing test: archive refused for an
unanswerable session *after* retiring an incarnation. `observeRecovery` also
stopped skipping the session probe for records carrying a park or a claim — the
same structural defect M1 removed from `gatherThreadEvidence`, where bookkeeping
decided whether the world got asked.

### `ThreadBusy` has exactly one producer, and it is bounded by the world

A `ThreadStartClaim` — couch's record of its **own** in-flight operation, not a
claim about an external process. Without it, the window between claiming a start
and the launcher acquiring a pid would classify `session-gone`, an
archive-eligible reason, for a thread starting normally.

**M2 finished the sentence.** A claim carries `{OwnerPID, OwnerIdentity}` — the
couch that began the transaction — precisely so "recoverable on its own terms"
can be checked, and nothing checked it. A couch dying mid-start left a row
reading `starting…` forever: no timeout, no owner check, and `busy` offers
neither archive nor resume. So:

> A start is in flight while its claiming couch is **alive or unprovable**. A
> claim whose owner is provably `Dead` has no driver, and the row reports what
> the world shows.

`Unknown` keeps the row busy, the same direction `ReconcileStart` already fails
("Unknown evidence always keeps capacity occupied"). `ThreadEvidence.StartOwner`
carries the answer and its zero value is `Unknown`, so a record no gather branch
reached cannot be declared driverless — the same fail-closed-by-construction
shape as `SessionUnresolved`.

Releasing decides nothing on its own: the row falls through and reads live,
detached, parked or session-gone like any other record with the same evidence.

### Cold-resume authority is the ledger, not the park receipt (M2)

`VerifiedPark` carries a `ParkIdentity` and **no native conversation id**. It can
attest that a park happened; it can never say there is something to resume into.
That lives in `ledger-<tag>.jsonl`, and `ResolveEstablished` is the only thing
that reads it.

The read used to be gated on `record.VerifiedPark != nil`, which made the receipt
the authority. Two consequences, one rule:

- A thread whose session died **without** a park was never asked about its
  ledger, so it read `session-gone` — archive-eligible — with a live
  conversation still recorded. Archive could discard it without a word.
- A receipt with no resolvable conversation behind it still read `parked`.

So the ledger is asked for **every resume-shaped record whose session is not
present**, and `VerifiedPark` leaves the classification path the way
`record.Park` and the incarnation liveness fields did in M1. It survives as a
*diagnostic*: a receipt with no ledger entry reads `binding-lost` rather than
`session-gone`, because "couch parked this deliberately and the conversation is
gone" is a different story for the operator.

The cost stays bounded by the same optimistic-inventory rule: a row couch is
hosting, and a row whose session outlived its launcher, pay nothing
(`TestWarmRowsAskNoLedgerQuestion`). An unreadable ledger is `ProofUnresolved`
→ `unusable/unknown`, never `session-gone`.

**The guard moved with the classifier**, because a guard that contradicts the
classification it feeds is the defect, not a detail. Four sites read the receipt
as cold authority: `DecideResume`'s admission gate, its `RequiredSessionID`
branch, the `CheckResumePreconditions` binding exception, and `Resume` itself —
which resolved the binding only for receipt-holders, so a row the classifier now
calls `parked` arrived with an empty binding and was refused `unbound`. `Resume`
now asks the **session** first (warm is cheaper, safer, and already preferred by
the classifier) and the **ledger** only when warm did not answer.

Two deletions fell out. `ResumeLegacyUnverified` — "thread has no verified park
completion" — lost its producer and is retired; the produced-by guard forced it.
And the `ParkHistory` tombstone scan stopped being a **veto**, surviving only as
the better explanation where there is nothing to resume into: archive abandons
orphaned parks as a matter of course now, so a veto would have made "couch
crashed mid-park once" a permanent cold-resume ban.

**`parked` has FOUR producers now, and its consumers are the enumeration.** A
park receipt whose session is absent; a ledger that resolves with no receipt at
all (M2's new one); a receipt whose session could not be asked about, which the
classifier keeps deliberately; and a driverless start claim whose ledger still
resolves. `everyThreadShape` carries all four, and
`TestEveryParkedProducerIsAcceptedByResumeSwitchAndArchive` derives its rows from
it rather than listing them — an earlier version hand-wrote two and its totality
check compared a map filled from that same literal, so it could not fail.

The consumers are the enumeration. That
is the rule the M2 boundary review extracted, after the sweep re-derived the
consumers of the evidence FIELD that widened (`ThreadEvidence.Parked`) and missed
the consumers of the STATE. `ThreadParked` has seven readers — `startup.go`'s
rank and `PathHoldsUsableThread`, `ActionableThreadSummary.Resumable`, two
renderers, `menuThreadActionable`, and the switcher's action list. The last one
offers `switch-agent`, and `PrepareAgentSwitch` still demanded a park receipt, so
a ledger-parked row was offered an action that always failed — the thing
`menu.go` names in its own words as how a switcher teaches an operator to
distrust it. Two derived tables make the class checkable rather than re-derived per
milestone: `TestEveryParkedProducerIsAcceptedByResumeSwitchAndArchive` takes its
producers from `everyThreadShape`, and `TestSwitchAgentOfferedImpliesPermitted`
takes its domain from `AllThreadStates() × AllThreadReasons()`.

**An action guard CONSUMES the classification; it does not re-derive one.** This
took two attempts to get right and both failures are worth keeping. The first
guard read `record.VerifiedPark != nil`, which refused M2's new ledger-parked
producer. The second asked the session directly — a second derivation, drifting
toward the cases its author thought of — and admitted a row couch is HOSTING
whenever the session index held no binding, while `SwitchAgent` parks a source
only when the record names an incarnation: two agents on one tree.

`SwitchableState(state, reason)` is now a pure predicate both the switcher and
`PrepareAgentSwitch` call, and the guard classifies through `classifyForAction`,
which runs the same evidence pass and the same rule the rows come from with
couch's own registry as live proof. The rule it encodes is *nothing is running
that a switch would orphan* — a switch launches fresh and resumes no
conversation. `TestSwitchAgentOfferedImpliesPermitted` derives its domain from
`AllThreadStates() × AllThreadReasons()`, so offered-implies-permitted is
checkable rather than remembered.

The enumeration that catches this class must itself be **derived**. The first
version compared a map filled from its own literal and so could not fail;
deriving it from `everyThreadShape` immediately surfaced two further producers of
`parked` — a receipt whose session could not be asked about, and a driverless
start claim whose ledger still resolves. The second of those is why `SwitchAgent`
now clears lifecycle debris before its target claim, as resume and archive do.

**A diagnostic code needs a producer reachable from a production entry point.**
The tombstone-as-better-explanation above was true of `DecideResume` and false of
couch: `ResumeContextWith` bailed on a binding diagnostic before `DecideResume`
ever saw the resolution. A binding diagnostic is EVIDENCE and travels with the
resolution it describes, so the resolver reports and `DecideResume` decides —
guidance at the consumer, the same rule M1 round 3 wrote. The emitted-somewhere
guard passed the whole time; reachability needed its own.

One asymmetry is kept deliberately. An unresolved session refuses a cold verdict
**unless** `record.VerifiedPark` says couch tore the session down itself. For a
deliberately parked thread the session answer is uninformative, so demoting every
parked row because one `list-sessions` failed is strictly worse. For every other
thread the session may be **alive**, and `parked` invites a relaunch that would
put a second agent on a live conversation. The receipt is a fact about what couch
did — that is all it is still read for.

### Admission is a predicate over the classification (M3)

Every action the switcher offers has a **pure predicate over `(state, reason)`**,
consumed by the guard that would otherwise re-derive one:

| Action | Predicate | Guard that consumes it | Permits |
|---|---|---|---|
| switch-agent | `SwitchableState` | `PrepareAgentSwitch` | `live`, `parked`, `binding-lost`, `session-gone` |
| archive | `ArchivableState` | `Couch.ArchiveThread` (via `classifyForAction`) | `detached`, `parked`, every unusable reason **except** `unknown` |
| reboot | `RebootableState` (= `ArchivableState`) | `Couch.Reboot` (`rebootAdmit`, then `prepareRetirement`) | same as archive |
| resume | `ResumableState` | `SelectResumableRoot` | `detached`, `parked` |

The switcher's **offer** is written separately, in the action table
(`menuRowActions`), and `couchtty`'s `TestActionOfferedImpliesPermitted`
compares the two over the table's derived domain. Resume on an unusable row is
admitted by `ResumeTarget`'s route rather than `ResumableState`, so the test
pairs it with that statement (resumable, or unusable). It is deliberately
not filtered through the predicate: a filter makes offered-implies-permitted true
by construction, and a guard that cannot fail is not a guard.

A live row with a failed continuation IS filtered, through
`ContinuationRefuses` (`pair#280`), and it stays falsifiable because the
list's oracle is the guard's BEHAVIOUR, not another predicate:
`TestContinuationRefusesMatchesTheGuardForEveryRowAction` drives every
declared row action through the production dispatcher on a live thread holding a
failed request. A listed operation must be refused by the guard itself, and an
unlisted one must succeed. A list that drifts from the guard sites fails there.

**What the resume row does NOT cover.** `ResumableState`'s consumer is startup's
`SelectResumableRoot`, not the Enter path — `ResumeContextWith` gathers its own
strict evidence and would pay a second evidence round to classify. So the table's
resume row proves the menu and startup agree; that pressing Enter on an offered
`resume` then succeeds rests on M1 having removed every bookkeeping read from
`DecideResume`, which now reads only the two facts the classifier used (a
surviving session, a resolvable ledger). Change `DecideResume` and this table will
not notice.

**Cost of the archive admission.** One host-wide `list-sessions` per archive,
added by M3. A row whose session is present also pays **three** `list-clients`
(~750 ms, `#228`) — `observeRecovery`'s three looks, which predate M3 — and a
sessionless row pays none. `TestArchiveEvidenceCostIsBoundedByItsMaximisingShape`
pins both, so a fourth look fails rather than costing another 250 ms unseen.

Two layers, not one. The predicate answers *what is this thread*; the record-shaped
guards below it (`archivableRecord`, `DecideRecovery`, `clearLifecycleDebris`)
answer *is there bookkeeping to act on*, which a classification cannot see.
`archivableRecord` narrowed to exactly that in M3: an open park and an
outstanding start claim — couch's own unfinished transactions, which a decoded
record proves by carrying them — and no longer an occupied incarnation, which is
a claim about a process the store cannot probe.

### Unknown survives the projection (M3)

`ObserveRecordedProcesses` returns `RecordedProcessObservation{Address, Process,
Liveness}`. It used to return only the positive answers, so a probe that could
not answer and one that proved the process dead reached the classifier as the
same silence — and since M1 that silence falls through to the session, which can
say `session-gone`. `ThreadEvidence.Unproven` is the negative side of `Live`, and
the classifier fails closed on it ahead of every durable refusal, because
`session-gone` is archive-eligible and `unknown` is the one reason archive
declines.

Narrow on purpose: `Exists` answering **Dead**, and an identity token that reads
and **differs** (a recycled pid), are confirmed answers and keep falling through.
That is `#272`'s fix and it must not regress.

### Lifecycle changes go through named transitions (M3)

`ThreadStore.updateExistingThread` is **unexported**, and inside `couchcore` it
is reachable only from a `*ThreadStore` method —
`TestArbitraryLifecycleMutationHasNoDoor` derives that from the receiver rather
than a file list, because all three leaking callers lived in the store's own
package and two in its own directory.

CAS, immutable-field checks and final validation protect a *coherent* record;
none of them requires an *authorized* change. The transition's **name** is where
the caller's authority is recorded:

| Transition | What the caller is asserting |
|---|---|
| `RetireIncarnation` | this live incarnation's process was just stopped |
| `RetireUnprovenIncarnation` | this exact `{PID, identity}` was observed **Dead** |
| `RetireProvedDeadIncarnations` | every recorded process was observed Dead, and none carries a start claim |
| `ClearVerifiedPark` | the thread is attached again, so the receipt describes an undone teardown |
| `BeginContinuationFromRetiredIncarnations` | the request and the retirement are ONE write, because a record must never carry both |

`RetireUnprovenIncarnation` exists because `RetireIncarnation` refuses an
`unknown` incarnation on purpose — detach holds no death proof, and retiring one
there would let an unproven thread present as cleanly detached. Archive does hold
the proof. Without the split, a record marked unknown by `markLiveRecordUnknown`
could never be archived once its helper died, which it always does.

### Archive stops a session, not necessarily an agent (M3)

`Quiesce` runs `zellij delete-session --force`, and **that reaps a pane by
SIGHUP**. Measured 2026-09-17 on zellij 0.45.1 with a throwaway session: a pane
child with the default disposition dies; one whose launching shell had `SIG_IGN`
survives and is reparented to init. Same fixture, one variable. `#274`'s 106
orphaned `pair term` trees are that regime in production, and this measurement
proves its standing hypothesis.

So reboot's confirmation on a **detached** row (archive's, before #363) names the
agent and says it *may* survive, rather than promising a stop couch cannot deliver. `#274` owns making it
a promise. The agent name reaches the confirmation through
`ActionableThreadSummary.Agent`, projected from the saved launch profile.

### Retired reasons

`stale-incarnation` and `unrecorded-child` both named a *disagreement* between
the record and observation, one per direction. There are no longer two sides to
disagree. `unrecorded-child` returns with `#276`, which gives it a producer — a
couch-tagged session with no record at all.

## Liveness is recomputed, never stored

Because Couch owns the console, diagnostic flags run in a **second process**
with no `Handle`. So `ActorRecord` persists `{PID, Identity}` where `Identity`
is `procutil`'s kernel start token, and a reader recompares it: a recycled PID
reports not-live because the token differs.

That correlation is still exact, and still positive-only: after #256 the
**absence** of a live observation proves nothing and falls through to the
session. The `Live` union — console pty children plus OS-vouched recorded
processes — stays a union, because the CLI passes no observations of its own and
narrowing it would make every running thread read `detached` there (`#181`'s
"one store, two stories").

Within one process, `ExecRunner` reaps its children in a background goroutine
and liveness is a closed channel — **not** `kill -0`, which succeeds for a
zombie and would report an exited-but-unreaped child as running.

## The bounded mailbox

`Enqueue` (`mailbox.go`) is a pure function: collapse by kind, drop the oldest
non-control entry over capacity, never drop control. `couchtty/notice.go` uses
it for the exit/bell feed.

**`Control` also decides how long a notice STANDS** (`pair#185`). Nothing used
to retire one -- no timer, no clear on keystroke, no expiry -- so a momentary
refusal like `previous: nowhere to return to` sat on the status row until an
unrelated notice displaced it, reading as current state. The type already
carried the distinction: an exit is an OBLIGATION (it says why a pane
disappeared, and is still true later), a refusal is an EVENT about the keystroke
just pressed. So a transient notice carries a lifetime and a control notice does
not. `Feed.Row()` walks from the tail skipping what has expired, which is why an
expired transient can uncover a control notice underneath but never an older
transient -- an older transient is staler, so it is never a better answer.

The row's expiry is also a REPAINT obligation: nothing else is guaranteed to
happen when a notice stops being true, so `Run` arms one timer for the row's own
deadline beside `syncSpinner`. That dedup is an OPTIMISATION and not a
correctness rule: re-arming every iteration would still fire at the right
moment, because the remaining duration shrinks with the deadline. Recorded that
way deliberately -- the first version of this note claimed the notice would
never retire without it, which was false, and a false rationale outlives the
line it justifies. `Feed` takes both its clock and its lifetime, because the two
are exercised at different levels: pure expiry tests hand-advance a fake clock,
a console test must let a real timer fire.

Pushing a notice and painting it are ONE operation (`publishNotice`). They were
two until a lifetime existed, when "the sentence appears whenever something else
next paints" stopped being merely late: on an idle console nothing else paints,
so a notice could expire entirely unseen, which is worse than one that
overstayed.

The goroutine loop that used to wrap it (`Actor`) was groundwork for
`pair#147`, where messages between actors would begin to exist. That scope is
punted, so the loop was built, unit-tested and never instantiated -- deleted in
`pair#170` M4. The mailbox stayed, because it has a real consumer.

Its shape is worth keeping on record: a mutex-guarded queue rather than two
channels with a priority select, because the bounded/collapse policy must apply
at insertion and a buffered channel cannot collapse a duplicate already in it.
That is exactly why `Enqueue` is pure and survived on its own.

## Terminology

- **namespace** — one canonical physical Couch store and its single live
  supervisor lease.
- **thread** — one durable composite `{repo_scope, opaque tag}` record.
- **path** — canonical starting/working location; not identity.
- **incarnation** — one creating/live/unknown run attached to a thread, with
  verified process identity and the repository identity that keys its saved
  launch preference.
- **actor / ActorID** — a hosted child/cache identity; routing and notices use
  it, while every switcher action uses the durable thread address.
- **parked thread** — a durable thread whose LEDGER resolves a conversation to
  resume into, with nothing running on it (no live evidence, no surviving
  session). It may still carry a dead launcher's incarnation or a driverless
  start claim. The park receipt (`VerifiedPark`) records that a park happened; it
  is not the authority, and `parked` has four producers (see `everyThreadShape`).

## Planned, not built

`pair#170` rescopes couch to **couch-lite**: a switcher over a group of live
coding sessions whose unit is a pair session. It adds resume of a live session,
`alt+d` detach with detached sessions listed and reattachable,
notification-focused switching, and a single `previous` slot whose one rule
(`entered_via_notification`) keeps a notification hop from costing the operator
their place. It also decides what of the machinery above is deleted.

**Punted by that rescope, not rejected:** `pair#153` managed-worktree lifecycle,
`pair#147` cluster transport and queries, `pair#148` brain as advisor, and the
cross-repo enabler `ariadne#199` exposing the query API. The reasoning is the
scope event in `workshop/projects/couch.md`.

Ariadne #200's normalized policy provider is implemented and consumed at the
#149 M1 boundary.


### Mouse diagnostic trace (#207, #255)

`COUCH_MOUSE_TRACE=<path>` enables `cmd/internal/couchtty/mousetrace.go`.
Records carry endpoint mode observations and presenter admission/gesture state,
with active handle, actor and durable thread identity. Presentation errors are
reported rather than hidden behind a raw-output scanner's belief. These are local
state and write-result observations, not terminal queries or proof of pixels.
The existing opt-in 0600 append sink closes at Console teardown and records no
child body or keystrokes. The operator removes the temporary trace after diagnosis.

### Fresh slot conversation launch (#315)

`StartFreshSlot` allocates a new conversation address and uses the same launch
profile, Pair registration and failure cleanup as ordinary creation. It validates
saved arguments before claiming an address or replacing the current record.
History and preferences stay with the slot; existing live-owner checks still
apply. Same-address agent switching and continuation keep their existing flows.

`TestSpawnComposesProductionPairRegistrationBoundary` runs ordinary and fresh-slot (reboot's `:1+` half)
creation through the real Pair launcher and claim files, with the special fresh
readiness observer unset. The slot needs no additional launch protocol.

## Unavailable retention namespaces (#346)

The global retention registry keeps namespace membership even when an auxiliary
store disappears or becomes unreadable. Registering an intact selected store
checks the registry structure and that selected directory; it does not require
every other namespace to be mounted. Couch startup/listing can therefore proceed.
GC inventory and migration acknowledgment still require all registered stores
to be readable, so missing references never become implicit deletion permission.

For a permanently abandoned, missing store, use
`pair gc --forget-missing-store /exact/canonical/registered/path`. This removes
only that registration and resets migration acknowledgment. Restore/remount a
temporarily unavailable store instead. To re-enable collection, independently
acknowledge every remaining namespace with `pair gc --complete-migration --store
PATH` (repeat `--store`). Existing paths, aliases, permission failures and mixed
mutation flags are refused. The coordinator owns the mutation under its lock.

Production-boundary coverage: `couchcmd.TestListWithMissingAuxiliaryStoreUsesIsolatedRoots`;
registry/collection recovery: `storagegc/stale_store_test.go`. Smoke invocations
must isolate HOME, XDG_DATA_HOME, PAIR_DATA_DIR and COUCH_STORE_DIR together and
clear inherited explicit artifact overrides. The original scratchpad producer
has not been identified.

### Isolated terminal pressure experiment (#373)

The [debugging runbook](../doctor/terminal-pressure.md) owns repeatable commands,
evidence capture, prerequisites, metric interpretation and extension checks.

`cmd/internal/couchtty/terminal_pressure_test.go` exercises production Console
input, endpoint ingestion, publication and presentation using bounded fake and
real-PTY children. `TestCouchPressureControl` runs in ordinary tests; opt in to
24 paired trials with `PAIR_COUCH_PRESSURE=1 go test ./cmd/internal/couchtty
-run '^TestCouchOutputPressure$' -count=1 -v -timeout=180s`.
It separates child receipt, endpoint ACK, displayed ACK and rendered switcher
latency under bursts, one Go CPU and delayed host writes. Two-second trials
have bounded output and joined teardown. It does not emulate Zellij, Ghostty,
system-wide scheduler pressure or sustained full-screen redraws; a negative
result cannot rule out those causes of selective pane freezing.

## Live display capture

For an explicitly opted-in session, `COUCH_CAPTURE_DIR` enables both display
boundaries with exact bytes, geometry and timing. Regular Couch is supported;
`COUCH_ISOLATED_ROOT` remains optional. Capture stays disabled by default. See the
[live capture runbook](couch-live-capture.md) for launch, limits and extraction.
