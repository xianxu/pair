---
collaboration-mode: discussion
---

# Shared agent startup preferences and safe switching

> **For agentic workers:** Follow AGENTS.md §3 for task delegation. Execute approved work with the executing-plans skill and TDD; SDLC owns boundary reviews.

**Goal:** A successful explicit agent configuration is available to Pair, the Couch form and Couch CLI throughout one repository; editing parameters and restarting the current agent work for live or proven-vacant sources.

**Architecture:** One leaf `launchprefs` package owns current preferences and repository/path resolution. Historical thread and per-tag invocation records retain execution/resume evidence. Existing core source classification, lifecycle holds and fresh launch transactions remain the only process authority. Registration commits before preference publication; a preference failure is a started-with-warning result, never a failed launch or rollback.

**Tech stack:** Existing Go launcher/Couch/core/TTY, strictjson raw-preserving codecs, filesystem fake, real Git worktree fixtures, current supervisor/terminal fake, existing live acceptance harness.

**Status:** Proposed to ops:0; no implementation approval yet. No implementation changes.

## Evidence and alternatives

- `launcher/createflow.go` skips repository-default learning for Couch-owned/fresh launches. Couch `advanceSuccessfulStart` instead journals exact-path preferences with the thread. `PrepareKnownAgentSwitch` reads that path and a separate primary-folder default, so another slot has neither.
- `switchServiceGuard.ConfirmAgentSwitch` requires a connected wrapper before core classification, although `SwitchableState` already supports proven-vacant parked/session-gone sources.
- `prepareAgentSwitch` rejects the current agent; the TTY field requires a successful `SwitchPrepared`, so that preview error also disables typing.
- The CLI parser/protocol have no explicit startup-argument intent.

Recommended: consolidate preference authority and preserve core switching. Merely copying a successful form's arguments into the old primary default leaves independent stores/codecs/writers, races and stale precedence. A cross-root journal would preserve the old thread-plus-preference atomicity, but adds recovery/lock ordering to a user-preference update; use the already committed registration profile plus an explicit warning on preference-publication failure instead. The latter tradeoff requires hub approval below.

## Product contract requiring approval

1. Same-agent confirmation is **Restart with new parameters**: a fresh conversation in the same thread/workspace, preserving the prior conversation for existing resume/history recovery. It does not resume the current native conversation with altered flags. The form states this before confirmation. CLI same-agent operation has the same meaning.
2. Argument precedence: explicit input (including explicit empty) > the selected agent's exact physical-path override > that agent's canonical repository fallback. Never inherit another harness's argv. Agent selection keeps existing explicit/path/root precedence.
3. Successful operator-authored configuration updates exact-path and repository fallback together. The most recent serialized successful publication for that agent defines the repository fallback; unrelated agents and path overrides survive. Concurrent launches are ordered by publication, not a fictitious comparison of unrelated path revisions.
4. Preferences are learned after observed successful registration/readiness. Warm attach, native resume and recovery do not learn a new preference. Failed launch/registration preserves previous preferences.
5. If the agent started but preferences cannot be published, report **Started; parameters were not remembered**, preserve the actual thread profile and running agent, and name storage repair. After repair the operator may explicitly restart with those parameters again. A crash between registration and publication has the same durability limit; no automatic eventual-publication promise or new outbox is added.
6. A parked/disconnected recording is eligible only when the existing core classifier proves vacancy. A disconnected but live native session, an unknown process/session, an incomplete park/start or pending continuation continues to require recovery. Busy/dirty live sources refuse without mutation.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| `Preferences`, `PathPreference`, `Profile` | `cmd/internal/launchprefs/preferences.go` | new: one repository envelope, exact-path overrides and per-agent fallback, revision and preserved unknown fields |
| `Resolve`, `Remember`, `MergeLegacy` | `cmd/internal/launchprefs/resolve.go` | new: deterministic precedence, successful update and explicit legacy conflicts |
| `LaunchProfileInputs` / `ResolveLaunchProfile` | `cmd/internal/couchcore/launchprofile.go` | modified: compatibility facade over shared policy, not another algorithm |
| `SwitchSourceEvidence` | `cmd/internal/couchcore/switchagent.go` | new: tagged live-owned versus proven-vacant admission evidence, revision and generation |
| Explicit argument intent | `cmd/internal/couchmessage/protocol.go`, `cmd/internal/couchcmd/switch_agent_cli.go` | modified: distinguish absent from an explicit string array, including empty |
| Switch form state | `cmd/internal/couchtty/menu_switchagent.go`, `menu.go` | modified: editable input independent of preview success; accepted preview still gates effects |

One repository envelope has many paths and agents; each exact path has one last-agent choice and per-agent argv. Thread/incarnation profiles remain execution evidence and native-resume authority, not competing user-default stores (ARCH-DRY/PURPOSE). `launchprefs` imports neither launcher nor couchcore, preventing a dependency cycle. Supported-agent and fresh-selector validation stay with the existing launcher helpers.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| `Store.Load` / `Store.Remember` | `cmd/internal/launchprefs/store.go` | new | bounded read, repository lock and atomic replacement |
| Preference identity and legacy locators | launcher runtime / existing Couch verified workspace adapters | modified | verified Git common directory and physical path, existing legacy locators |
| Pair preference adapter | `cmd/internal/launcher/{runtime,osruntime,createflow,agent_defaults}.go` | modified | existing readiness seam and shared store |
| Couch preference adapter | `cmd/internal/couchcore/{couch,threadstore,threadstore_location,slotrecovery,slotmigration,recovery_preflight,startresolution}.go`, `cmd/internal/couchcmd/run.go` | modified | shared reads, registration publication, preview fingerprints and legacy import |
| Registration publisher | `cmd/internal/couchcore/{launch_existing,couch}.go` | modified | the two production StartRegistered promotion sites and existing warning transport |
| `ConfirmAgentSwitch` / remote selection | `cmd/internal/couchcmd/agent_switch_service.go`, existing hub target/slot operation helpers | modified | live generation/idle proof or canonical vacant target, hub ownership and clean Git |
| CLI/request/receipt transport | `couchcmd/{switch_agent_cli,slot_operations,run}.go`, `couchmessage` request validation | modified | literal argv, bounded payload and idempotent receipt identity |
| Form preview/effect dispatch | `couchtty/{menu_switchagent,menu,console_menu}.go` | modified | existing owned preview/effect queue and input decoder |

External doubles are the existing folder-backed launcher runtime and Couch environment/runner/process/session fakes. New store tests use actual private files and independent store instances. Git identity fixtures use a real repository and worktrees; no function-call mocks replace concurrent store or lifecycle behavior (ARCH-MOCK).

## Identity, migration and persistence

- Canonical key is verified common-Git identity, not the exact-folder RepoScope hash or a human label. A slot must prove its repository through existing workspace routing; a folder name alone never grants access to another repository's preferences. Non-Git standalone folders retain exact-directory isolation and have no cross-folder repository fallback.
- The shared store owns one identity-addressed envelope under Pair's data root and a stable lock file. Add canonical path derivation/classification through `artifactpath`, not ad hoc joins across readers. Both Pair and Couch receive the same data-root/identity inputs.
- On legacy discovery, adapters enumerate only identity-verified known sources: current/primary legacy agent defaults, retained Couch exact-path preferences and enrolled Git worktrees. Use existing ThreadStore routing/SlotLayout accessors for Couch paths. Do not introduce a raw fleet-wide scan or derive slot ownership from directory names.
- Import per-path values without collapsing paths. One unique legacy argv value for an agent can seed its repository fallback. Differing legacy values have no comparable successful timestamp: preserve exact overrides and expose an unresolved repository fallback requiring explicit accepted arguments. Do not use mtimes or revisions from unrelated files to choose a winner. Already confirmed canonical values outrank later-discovered legacy candidates; only missing legacy paths/agents are imported.
- Legacy import discovery is bounded and recorded by source identity/digest in the repository envelope so ordinary preview does not rescan the fleet. Both adapters use the same merge policy; a later Couch discovery can import missing old sources without overwriting confirmed canonical values. Original legacy bytes remain untouched during import. Once canonical state exists, legacy files are migration input only, never an independent preference authority.
- Retire current preference writes to `agent-default-*.json` and `.couch/preferences.json` after adapters move. Preserve per-tag configs, watcher updates, thread profiles and native session bindings as launch/resume records. Do not broadly rewrite their codecs or remove historical data in this issue.
- Strict known types, required fields, duplicate rejection and explicit-empty arrays survive. New envelopes and nested records retain unknown raw members using `strictjson.SplitObject`/`EncodeObject` rather than decode-and-drop (ARCH-EVOLVE). Existing legacy strict/tolerant decoder behavior is treated honestly; migration cannot invent unknown values already lost by an older writer.
- Suggested initial envelope limits: 4 MiB, 256 exact paths, at most the supported agent inventory per path and 4096 formatted argument bytes per profile. Enumerate at most 256 verified legacy preference sources per import. Refuse saturation without eviction and retain old bytes; report the file to inspect/archive. Reader and writer use the same limits.
- Per-repository locking covers read-modify-write and one atomic replacement of path plus fallback; it is released before returning. Thread-store registration retains its local journal atomicity, then releases that lock before preference publication. Never nest the two locks or hold a preference lock across process launch/registration. Publication errors use existing start/switch warning channels and combine with existing cleanup warnings.
- One bounded envelope replaces itself; no append-only preference history or launch outbox is added. Legacy sources remain existing compatibility residue until the existing repository-data lifecycle removes them. Canonical preference artifacts join the artifact inventory/retention model and remain while the repository's saved settings are retained; ordinary per-thread history pruning must not delete repository defaults (ARCH-FUNERAL).

## Source state and user-visible outcomes

Actions: edit, cancel, confirm switch/restart, inspect/recover, repair storage then explicitly restart.

| Internal state/failure | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Preview loading/refused, draft exists | editable parameters; confirmation unavailable or needs refreshed review | edit/cancel; refresh through confirmation after resolving stated refusal | cancel discards draft only |
| Live owned exact incarnation, idle and clean | Switch or Restart with new parameters available | confirm or cancel | confirmed operation starts a fresh conversation; prior session retained |
| Proven-vacant parked/session-gone source | same fresh-start action without trying to park a nonexistent wrapper | confirm or cancel | prior recording remains history; no running work to stop |
| Disconnected native session still live, busy, dirty, unknown or incomplete lifecycle | specific refusal; no effect | inspect/recover or commit/finish current work, then retry | none from refused action |
| Source replaced/reconnected, ownership moved, preference revision changed after preview | stale review refusal | reopen/review and confirm again | draft preserved where safe; no replacement is stopped |
| Source park incomplete | existing park recovery state; no target launch | inspect/recover exact source | partial park diagnostics; no new agent |
| Park complete, target claim/fork/registration fails | prior source remains parked/resumable, or existing explicitly unconfirmed launch state | inspect/recover; explicit retry only when safe | no automatic destruction of foreign/unknown processes |
| Target registered, preference write fails | Started; parameters were not remembered, exact running profile retained | repair storage, then explicit same-agent restart if desired | repository fallback may remain old; current conversation retained in history when restarted |
| Target registered and preference publication succeeds | started with accepted args; next slot sees fallback | normal use | previous session retained by existing history mechanism |

`SwitchSourceEvidence` binds the prepared thread revision and either exact live binding/session token or proved-vacant classification. Remote targeting resolves canonical repo:N and retained selected thread independently of wrapper connectivity, using existing hub target resolution. Connected caller authentication, hub-only authorization, no self-target, canonical queue key and repeat-receipt handling remain. Revalidate caller/owner, source mode/revision/generation, clean Git and profile fingerprint at queued preparation and immediately before first mutation under the existing lifecycle hold. A formerly live target becoming disconnected does not silently turn into vacant authority (ARCH-ORDER/SECURE).

## Arguments and form behavior

- `couch --switch-agent repo:N --agent NAME [--agent-args TEXT] --confirm` reuses `launcher.ParseLaunchParameters`; no shell execution, environment expansion or command substitution. Parse once, carry an optional typed argv array, and preserve absent versus explicit empty through request, queue and operation dispatch.
- Validate supported agent, formatted 4096-byte limit, protocol payload limit and `ValidateFreshAgentArgs` before mutation. Resume/session selectors remain invalid for this fresh-switch path.
- Receipt intent includes target, agent, argv presence and exact argv contents; reusing an ID with changed arguments is an id-conflict. Identical replay returns the original receipt before probing the replaced target.
- Permit same-agent preparation deliberately. Label confirmation **Restart with new parameters** and disclose fresh conversation. Different-agent action remains Switch. Both use the existing switch lifecycle and old-session capture; do not add a parallel restart engine.
- Typing/backspace/cursor/paste routing works even if preview has not arrived or failed. Preserve Unicode, spaces and quoting, explicit empty, byte limits, cancel purity and latest-generation preview rules. Editing invalidates accepted preview but does not depend on already having one. Late preview cannot overwrite edited text or dispatch a cancelled form. A first successful preview after an error is reviewable before any effect.

## Named test strategies

| Functions / seam | Strategy |
| --- | --- |
| `Resolve`, `Remember`, `MergeLegacy` | Pure table/property tests for independent agent/argv precedence, empty/absent distinction, no cross-agent inheritance, independent paths, unrelated-agent preservation and explicit legacy conflicts. Seed a resolver ignoring repository fallback and require failure. |
| Preference codec | Literal/fuzz invalid known types/duplicates/limits; older/newer raw unknown fields at envelope/path/agent nesting survive decode-update-encode. Exact/one-over bound leaves prior file unchanged. |
| `Store.Remember`, identity adapters | Independent instances race controlled publication barriers; require no lost path/agent updates and final fallback equals serialized publication order. Real Git primary/two worktrees share one store, symlink aliases converge, another repo and non-Git directories stay separate. |
| Legacy adapters / import | Successful old form config in one verified slot is discovered before another slot's CLI known-config lookup; conflicting candidates retain exact paths and demand explicit args for unresolved fallback. Count source reads before/after import; source bytes never change. |
| Registration publisher | Control failed fork, failed registration, successful registration, reconciliation, recovery and warm attach. No learning before success; failure after successful registration yields warning without process stop/rollback. Crash between commit/publication keeps exact executed profile and makes no eventual-repair promise. |
| `ConfirmAgentSwitch`, prepare/execute | Stateful real guard+dispatcher matrix: live/proven-vacant/unknown × same/different agent × absent/empty/nonempty args. Controlled replacement/reconnect/idle-to-busy/Git-dirty/hub-move/preference revision at prepare/queue/effect barriers: zero unauthorized effects and one exact launch on success. |
| CLI/protocol/receipts | Table/fuzz parser, malformed quoting/NUL/limits/resume selectors, optional empty array round-trip, real socket flow and receipt conflict/replay with changed or unchanged argument intent. |
| `reduceSwitchAgentKey` / preview / console | Feed raw terminal bytes through console input with held/refused preview, edit quoted Unicode text, cancel, inject late/duplicate previews, same-agent Restart label and exact final argv. Keep reducer immutability/cursor tests. |
| Actual public flow | Existing disposable Couch/runner harness: form-configure slot A → CLI switch slot B without prior config → same-agent explicit-args restart → parked recording switch. Preserve prior session and assert fresh registration plus canonical remembered args. Stage a build for operator echo:2/echo:3 smoke only after hub authorizes :0 and selects safe idle targets. |

Growth is preference bytes B, path count P and discovered legacy sources L: one bounded read/decode/write per publication O(B+P), direct keyed resolution O(1), one import enumeration O(L), no per-entry full-tree rescans. Count decodes/writes/source reads at N and 2N, require linear operations and zero repeated legacy discovery on an unchanged imported source set. Form editing is bounded by the existing 4096 bytes and issues no preference IO per keystroke (ARCH-CONSTRAINTS/PURE).

## Execution and review boundaries

### M1 — Shared preference authority

**Files:** new `cmd/internal/launchprefs/{preferences,resolve,codec,store}.go` and colocated tests; adapters enumerated above; artifact-source/reader inventories; `atlas/` preference map/index.

- [ ] Write pure resolver/update/legacy/codec red tests, observe failures, implement the minimum shared types/policy preserving existing agent/argv semantics.
- [ ] Add real-file concurrent store and real-Git identity red tests; implement bounded atomic repository storage and identity adapters.
- [ ] Add cross-client migration and success/failure-publication red tests. Move every enumerated current-default reader/writer, keep history/resume configuration separate, and implement explicit warning behavior at both successful-registration promotion sites.
- [ ] Run `go test ./cmd/internal/launchprefs ./cmd/internal/launcher -count=1` and targeted Couch preference/registration/slot migration tests, plus counted growth and negative-control tests. Existing recovery/resume tests must show no default learning.
- [ ] Audit every new production artifact path and changed Go reader; refresh only audited inventories. Update atlas and Log, commit, push; run required recorded affected check and `sdlc milestone-close --issue 728 --milestone M1 --verified '<actual evidence>'`. Resolve mandatory findings before M2.

### M2 — Form, CLI and safe switch/restart

**Files:** `couchcore/{switchagent,actionableinventory,ops,operationdispatch}.go` as needed; `couchcmd/{agent_switch_service,switch_agent_cli,slot_operations,run}.go`; typed request/receipt codec in `couchmessage`; `couchtty/{menu_switchagent,menu,console_menu}.go`; colocated tests, README and embedded Couch skill/help.

- [ ] Write raw-console held/error-preview edit and same-agent red tests; implement editable draft plus honest Restart confirmation using the existing parser/preview generation rules.
- [ ] Write live/vacant service-guard and lifecycle race red tests; implement typed source evidence and canonical parked target resolution while preserving all unknown/foreign-process refusal tests. Do not broadly weaken connected/live safeguards.
- [ ] Write explicit-argv CLI/protocol/receipt red tests; implement literal argv transport and id-conflict protection through the same queue/core operation.
- [ ] Run `go test ./cmd/internal/couchtty -run 'SwitchAgent' -count=1`, `go test ./cmd/internal/couchcmd -run 'SwitchAgent|SwitchGuard|HubPreparedSwitch' -count=1`, relevant couchmessage protocol tests and couchcore switch/parked-producer race tests; then execute the real public harness above. Update fixed scanner lists and per-file hashes only after audit.
- [ ] Update help/embedded skill/README/atlas with precedence, same-agent fresh-conversation semantics, explicit empty and visible warning/refusal/recovery. Run `make test-affected-inventory`; push final code and `sdlc local-check run --name affected`.
- [ ] Ask hub to authorize `sdlc stage --build` and choose idle operator smoke targets. Record form typing, shared defaults, same-agent explicit-parameter restart and parked-source results; preserve existing native sessions for recovery. Do not switch another live slot without that authorization.
- [ ] Close M2 through its mandatory review, then `sdlc close --issue 728 --verified '<recorded checks + real acceptance>'`, `sdlc pr` and on SHIP `sdlc merge --yes`. Report landing to hub; #743 remains separately parked pending #509.

## Architecture check

ARCH-DRY: one preference authority/codec/resolver, one switch lifecycle. ARCH-PURE: policy/codecs separate from store/process/terminal IO. ARCH-PURPOSE: all four reports plus Pair/Couch consumers and cross-slot fallback covered. ARCH-MOCK: stateful file/process/session doubles and actual public acceptance. ARCH-CONSTRAINTS: bounded files/arguments/imports and counted growth. ARCH-SECURE: verified repository ownership, literal argv, exact live identities and unknown refusals. ARCH-ORDER: typed live/vacant preparation, stale guard rechecks, learning after registration with explicit publication failure. ARCH-FUNERAL: replaced bounded envelope, stable lock, existing history retention, no outbox. ARCH-EVOLVE: additive raw-preserving format; legacy source bytes survive import.

## Approval requested from hub

Approve the shared authority and bounded compatibility migration; same-agent fresh-conversation semantics; and explicit started-with-warning preference publication instead of a cross-root journal. This last item is a deliberate durability tradeoff: a crash/write failure after successful registration can leave fallback old, while actual launched arguments remain in thread history. After approval, run change-code's plan-quality gate; derive estimate only when requested by that gate. No code or live slot switch before approval.

## Revisions

### 2026-10-20 23:43 — Advisory proposal audit: migration, preview and finalization

This revision governs the initial publication/migration wording above. One read-only advisory audit identified three necessary refinements; no implementation has begun and hub approval is still required.

**Incremental legacy conflicts.** The repository fallback records whether it was imported or operator-confirmed, plus bounded source identity/digest provenance. An imported fallback is provisional: discovering a differing legacy value invalidates it and exposes a conflict, in either discovery order. An operator-confirmed fallback is authoritative and is not replaced by later legacy discovery. Conflicting legacy values for the same physical path/agent also remain unresolved until explicit accepted input; preserve all conflicting source bytes/provenance instead of picking an arbitrary winner. If last-agent legacy choices conflict, require an explicit agent instead of guessing. Extend `MergeLegacy` and adapter tests to both discovery orders, duplicate/equal candidates, same-path conflict, explicitly confirmed empty fallback and late legacy sources.

**Read-only previews.** Add `cmd/internal/couchcore/threadstore_preview.go` / `PreviewPathLaunchPreference` to the consumer sweep and tests. `Store.Load` on a preview path performs bounded read-only resolution, including a pure in-memory legacy merge where canonical migration is absent; it never initializes a preference directory/lock, writes migration markers, enrolls retention or replays a journal. Persist migration provenance only inside an authorized successful-configuration publication. Existing imported-state source digests prevent repeated discovery after publication; before that first publication, bounded read-only discovery may repeat. Fingerprints include the relevant canonical revision and legacy source digests so acceptance detects changed inputs. Assert byte-identical filesystem trees before/after opening, editing and cancelling the form, and for failed/stale preview/confirmation. Count source reads at N/2N separately for pre-publication previews and already-imported operation paths; do not claim a zero-read cache that does not exist.

**Final launch adoption and warnings.** Successful `StartRegistered` is necessary but not sufficient: `launch_existing.go` can still fail actor registry persistence and clean up the helper afterward. The shared publication helper runs only after the successful launch's last fallible finalization that can trigger rollback, including registry persistence/actor adoption. The two registration promotion sites in launch_existing.go and couch.go reconciliation must use the same success predicate; no preference publication on subsequent registry/adoption failure. Keep thread registration's local journal intact and never nest its lock with the repository preference lock.

Pair's existing `defaultReady` handling currently treats default-persistence error as a launch error; move that specific error onto the explicit started-with-warning path after successful child adoption. Couch StartResult's Warning() path exists, but SwitchAgentResult currently has a Warning **field**, while receipt completion consumes a Warning() interface. Plan the result/codec adaptation explicitly: preserve the public warning JSON meaning, combine preference and cleanup warnings once, surface it through console and CLI, and finish a remote receipt as succeeded-with-warning. Do not infer interface conformance from a same-named field. Extend real adapter/receipt tests to registry failure (old defaults unchanged), preference publication failure (registered child stays adopted, zero rollback/stop), successful receipt with visible warning, Pair exit/result success with diagnostic, and combined cleanup/publication warnings. A crash after successful adoption but before publication retains the disclosed best-effort durability limit and exact historical invocation; no new outbox is implied.
