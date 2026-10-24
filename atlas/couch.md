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

### Tab styles (#796, #779)

`couch --tab-style collapse` is the launch default; `--tab-style multiline`
selects the previous one/two-row packer. `couchtty.TabStyle` and
`DefaultTabStyle` in `tab_style.go` provide the shared parser and Console
configuration; the choice is presentation policy, separate from `Couch.Layout`.
The CLI applies it before child adoption and initial geometry.

Collapse uses one bottom row. `status_collapse.go` prepares candidates from the
existing presentation's hub/group identities: quiet slots and quiet inactive
hubs fold into per-group counts only when at least three members can fold;
one or two remain individual chips. Candidates exclude known parked/non-live
rows; pending placeholders stay explicit. Anchors use the launching hub rather
than the focused tab's owner. Counts mean hidden live members. Active and
attention chips stay explicit
when space permits. Counts carry actor cardinality, so later scarcity overflow never
double-counts folded actors. The shared measured planner and drawing pass in
`status_layout.go` and `reserve.go` retain active/attention chips first. At
impossible widths the overflow marker includes `!` when hidden actors need
attention (`+N !` when it fits); the bell still opens the full notification list (bell).
A tiny host prioritizes a clipped, clickable active span over fixed controls.

`couchcore.CompactSlotGlyph` projects the existing closed `SlotGlyph` grammar
to one cell: dirty, diverged, ahead/behind, issue branch, otherwise neutral `·`.
The collapse pass retains one marker per exact hidden live member in presentation
slot order on frame-local `StatusActor.FoldedStateStrip`. After base packing,
`decorateFoldedCounts` measures the winning cells once and admits whole strips
left-to-right into spare columns. A non-fitting group stays plain `+N`; later
smaller strips can still fit. No detail displaces actors, counts, controls or
attention. Global mixed-group overflow has no strip, and multiline is unchanged.
The shared drawing pass applies existing glyph colour/owner shading and extends
the same count hit span through the separator and final marker. Markers convey
checkout evidence, never agent activity; all detail dies with the frame.

A complete drawn group count opens the root switcher scoped to its exact owner
hub and repository group, using the same presentation membership before text
filtering. Scope remains through inventory refresh and nested action return;
removed membership produces an empty scope rather than a broader list. Escape
clears a text filter first, then the root scope. Opening the ordinary switcher
clears both. Counts and actor chips publish row/column hit spans in the same
confirmed chrome revision. Shared `textwidth` measurement and fitting use the
same grapheme metric as physical terminal cells, so the whole count, including
every digit, is clickable even after emoji aliases. Clipped or absent counts
have no action, and stale
or stalled-paint geometry cannot open a different group.

Explicit multiline preserves #779's measured packing across at most two rows.
Groups stay together unless one original group exceeds a full line; a continued
row starts with the full repository label. Chrome/cast/capture controls remain
at the start of row one, the attention badge at its right edge, and transient
notices use free space on the final row. Beyond two rows, retained actors keep
their original order after priority selection; the final overflow counts hidden
actors and flags hidden attention. Labels are sanitized/measured once, with
O(B + A log A) bounded planning shared by both styles.

Both styles share `ChromeStatusText`, which projects transient and capability
notices to at most40 display cells, including an ellipsis, after terminal
sanitization. Feed and diagnostic records retain full text; switcher Ctrl+Y
copies held retries, selected failures, current capability diagnostics, then the
last feed notice. Producers retain complete upstream wording; common wording
cleanup is tracked separately in #802.

Collapse keeps child height constant across changes in actor content, focus and
attention, removing #810's content-driven one/two-row resize trigger. Physical
host-size changes still resize PTYs. Multiline's extra row reduces pane height
only while used. Both styles preserve the existing broadcast, privacy and
annotation guards: if `LIVE ⏸` cannot be shown completely in its own style,
broadcast frame admission fails closed. See [terminal ownership](terminal.md)
for admitted versus confirmed geometry and [broadcast](broadcast.md) for trusted
chrome rows.

### Slot scratch (#708)

[Slot scratch](couch-scratch.md) maps optional TestScratch temp settings, the
24-hour interior cleaner, stable socket namespaces and reboot-only metadata
routing. Existing panes inherit changes only after relaunch.

### Per-slot switcher jobs (#576)

`couchtty/menu_operations.go` owns pure target admission, exact attempt matching,
current landing intent and one retained result per listed row. Busy selection
waits on its existing job; newer navigation fences final focus and old forms
never reappear. Missing targets keep their label/identity and diagnostic in the
notice; selected row failures use the existing Ctrl+Y copy surface.

The existing worker count is split in `operation_queue.go`: lifecycle workers
own effects and adoption; the final worker owns currently hosted navigation.
The shared16-position admission bound reserves15 lifecycle positions and1
navigation position. If hosted endpoint proof disappears, the same pending
request transfers once to lifecycle or reports overload; navigation never
resumes or adopts. Remote calls, continuation and startup use lifecycle workers.

Running start reservations keep their original identity while inventory materializes: pure aliases use the exact numbered checkout or the primary scope derived by launcher from the accepted worktree. Address-replacing ordinary actions reserve that primary scope too; other ordinary actions stay address-specific. Admission, busy selection and linear row projection use these aliases; final outcomes still bind to exact returned identity.

`console_operation_adoption.go` prepares final outcomes off Run. Installation
reserves identity under the console mutex, performs resize/registration outside
it, and revalidates lifetime, endpoint and geometry before committing. Rejected
registration is retired without closing the caller-owned child. Run performs
final lifecycle observation, reduction, navigation and remote acknowledgement,
with one completion owner and shutdown settlement. Existing core lifecycle and
park nonce reservations remain authoritative; UI navigation never cancels them.

### Load-independent lifecycle test evidence (#659)

The recovery ownership fixtures use `SlotCatalogFake`; a separate catalog-timeout
regression checks refusal without changing retained state. Park client-growth
semantics inject queries into a test-only readiness helper while fake-CLI tests
keep the native query contract. Clipboard key/fact tests exercise the unchanged
product timeout and fallback. Activity byte-count invariants configure the existing
probe-budget seam, with production deadline tests kept separately.

Affected failures include best-effort `testwait.HostState()` load information.
Native open-file conformance skips only unavailable/incomplete lsof observations;
run `PAIR_REQUIRE_LSOF_CONFORMANCE=1 go test ./cmd/internal/diagnosticlog -count=1 -run 'Test(OpenFileInspectionConformance|UnregisteredChildOpenFileBlocksLegacyExpiry)$'`
to require that native evidence. An answered wrong holder or identity always fails.

### Slot continuity and capture provenance (#528)

The [slot-continuity target](../workshop/targets/slot-continuity.md) defines
three outcomes: **Continue**, **Resume from disk**, and **Start fresh**.
Zellij's positive non-EXITED listing authorizes Continue/attach, including
sessions launched by older Pair binaries without policy files. A failed
listing never authorizes attachment. Observations get one bounded retry;
uncertainty is disclosed with a recovery action rather than a pending-only row.
Explicit recovery now uses the shared prepared path in #556 M1: an attributed
running owner requires confirmation of its exact process generations before any
stop. The person and authenticated owning hub use the same confirmation; unknown
or foreign remnants are left untouched and Fresh allocates a new terminal identity.
An unlisted start claim ceases to be Starting after five minutes or proven owner
death. Native recovery fences the exact old nonce before making a new claim;
Fresh archives the source and allocates a new tag. Age never authorizes killing
the Couch supervisor. Agent activity never demotes a positively listed session
from Continue.

A direct Proven or Fresh recovery sets `RecoveryObservationRequest.SkipCandidates`
(pair#726). The reader still builds `Source`, the screen, the capture, `Agent`,
`Generation`, the branch and the issue, and fences the capture identity. It
skips only the native transcript scan, which walks the agent's whole native
store under the hub effect gate and is consumed only by an inferred (keyed)
choice. When it offers the confirmed Resume from disk, `--resume-candidates`
says an incomplete scan does not feed that choice: it comes from the
established binding.

Public Continue attaches through `couchcore/warm_attach.go`. After attachment,
it may replace only a proven-dead client-helper incarnation with its exact
locally owned helper, rechecking the old PID/start identity at the write gate.
Bindings, parks and history remain intact. Open claims or uncertain/live old
owners leave the record untouched: a borrowed attachment shows the reason.
Wrapper and agent PIDs never become the client-helper incarnation. Its client
is tracked in the registry and an in-memory handle map owned by this Couch.
Detach, ordinary leave and failed-attach cleanup end only that exact helper.
A persisted registry entry alone is never proof that this Couch owns its handle.
Public completion and Pair’s final attach handoff also use positive listing,
not lifecycle owner metadata. Explicit Park/relaunch/switch may adopt the exact
locally owned helper through existing transactions after their admission checks;
foreign or unresolved lifecycle owners remain protected. Explicit continuation
recovery keeps its separately authorized adoption path.

Every command-line hub is an explicit launch/attach request, including the cwd
hub for bare `couch`. A positively listed hub uses Continue; a stopped hub uses
the normal proven-resume/fresh startup path. Queued extra hubs follow the same
rule. Non-hub slots use preview reads and background attachment of every
Continue row, with bounded parallelism. Startup never launches or resumes their
agents, completes interrupted starts/parks, or replays slot-store journals.
The same guarded dead-client-helper replacement applies to these attachments;
conversation bindings, parks and open claims remain untouched. An unhealthy slot never blocks Couch or
other slots. Explicit per-slot actions own recovery writes. Zellij listing's
own connection-refused stale-socket deletion is an accepted native side effect.

Pair retains its canonical socket-keyed attach lease and disables layout
serialization for newly created sessions. Attach is a write, never a probe;
it is invoked only after a positive live listing. The lease serializes Pair
callers but cannot prevent server death between listing and attach. For legacy
saved layouts, that resurrection race is accepted and measured on scratch
sessions; no policy-file authority or custom probe attempts to eliminate it.
Stable lock inodes occupy 256 stripes in session-locks-v2 and are retained.

Startup attachments and explicit Continue use the same single bounded retry.
Background attachment does not block the console or steal focus. Every failure
keeps its Continue outcome and shows a reason; activating its selected row
retries the warm attachment (if its row is not selected, select it first);
a later inventory observation may independently change the outcome.
Authenticated wrappers reconnect and join the broker independently of panes;
retained thread/workspace, positive terminal listing, readiness and exact process
proofs remain the message authority. An unfocused live slot can send and receive.

Automatic broker admission and use-time thread/agent/tail authority use the
shared context-bounded preview getter, including backend-routing reads. Pending
store journals remain untouched until an explicit lifecycle action; an authority
read cannot enroll retention or recover a transaction merely because a wrapper
reconnects. Explicit lifecycle getters retain their recovery behavior.

Warm attachment preflight, inventory and optional dead-client repair use the
same contextual preview readers. An absent warm-only session refuses before
lifecycle bookkeeping. The launched Pair helper skips proof migration and
orphan-editor/index housekeeping for a selected terminal; title-agent inference
reads bounded local metadata without opening native transcripts. Draft opens,
poller-PID cleanup and quit/restart marker admission reject blocking special
files. These checks do not change explicit cold launch or recovery authority.

A broken message channel does not authorize restarting a functioning agent.
Older wrappers with an unlinked endpoint remain usable by terminal, with
“messaging unavailable until next relaunch.” New-wrapper endpoint repair is a
separate follow-up (#557). Resume/Fresh execution and confirmation by the person
or hub share #556's prepared recovery path, described below. Couch's resume and
recovery launches refuse only if another positively live slot in Couch's
registry records the same conversation ("conversation X is in use by repo:N",
#597). No launch takes native admission (#595): the conversation lock below is
the only other guard, and its refusal is the same "in use by" naming the holder.
A Fresh reboot stops everything the slot owns (#607): to whatever terminal
attribution proves, it adds the wrapper Couch's message registry names (exact pid
and start identity), that wrapper's descendants from one process-table snapshot,
and the slot's own zellij session by its recorded name unless another live slot's
record names it. All of it is listed in the reboot confirmation; the reboot
refuses only when an exit cannot be confirmed.

Graceful Couch detach can retire a proven-dead helper before the next Couch
starts. A listed-live terminal with no retained incarnation then attaches as
borrowed: the registry and local handle own the client. Narrow dead-client
repair replaces an existing exact identity only; it never invents a durable
incarnation, agent PID or wrapper PID from this empty state.

New tty captures start with an events witness naming their file identities,
capture generation and producing launch ordinal/nonce. Archive metadata v2
preserves that witness; v1 remains readable. Certification additionally needs
a native binding for the same launch ordinal. A capture witness alone is not
conversation proof, and historical files are never retroactively certified.
Terminal startup budget (#773): `startupbudget` owns the absolute deadline and
its consumed exec representation. `launchTrackedThread` chooses it before the
blocked helper starts; public borrowed-client attachment in `warm_attach.go`
selects it once before preparation and the existing helper retry loop too.
Both transport the same absolute value and consume its remaining time, while
public attachment preserves its separate short listing-query window and the
borrowed terminal's lifecycle. Helper acknowledgement, pane birth, exact readiness and
registration consume that same window. Explicit caller deadlines are preserved;
no-deadline calls reuse the two-minute ceiling. Runtime handoffs carry startup,
short observation and interactive contexts separately. Native precreation and
postbirth queries retain their five-second observation bounds, capped by startup;
the initial ten-second birth probe remains diagnostic. Admission joins observers
and its failed client before releasing the terminal lease; startup expiry after
admission cannot end the interactive client. Final exact recovery observations
retain their separate bounded reconciliation contract and never retry launch.

Existing operation progress flows through context reporters into the console or
the exact remote receipt's pure transition function. Optional immutable
`OperationProgress` gives stage/start time; repeated observations preserve its
age and terminal/retired request updates refuse. The console's pure menu
transition separately retains stage-start time for both row and progress-notice
rendering; operations without stage reports keep their operation-age display.
The CLI's existing 15-second
status cadence displays stage and elapsed time. Launch failure diagnostics name
the consuming stage and its elapsed duration. This additive ephemeral receipt
field deploys with the coordinated supervisor/CLI rebuild; it is no persisted
format migration and does not promise old strict clients can read new fields.

Implementation: launcher/session_handoff*.go, launcher/session_lease.go,
couchcore/startup.go, couchcore/sessionevidence.go and
artifactpath/capture_witness.go.

[DOM submission](webcast-submission.md) (#483) sends the cast tab's open notes
from the draft's Shift+Ctrl+Alt+Return chord. Couch owns the durable JSON/PNG
bundle and acknowledgment; the existing authored-draft transaction delivers
its path to the pinned agent. Its local submission RPCs do not use peer-message
delivery or consume the peer allowance.

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
record and retained history. Repository arguments live in the global Couch namespace,
shared by every slot of that repository. Each backend uses the existing
lock, revision checks and recoverable journal. The supervisor lease still belongs
to the selected global namespace beneath the account lease.

### Failed-start session cleanup (#712)

After an acknowledged helper dies, owned-session cleanup shares one five-second
budget across owner observation, revalidation, retries and native deletion.
Persistent refusal returns SessionCleanupUnconfirmed. Marking its exact binding,
start nonce and incarnation unknown has a separate five-second persistence
context covering metadata reads, revision retries and the journal write. If even
that cannot commit, the original occupied claim stays unchanged, and history and
the receipt report both cleanup and unknown-state persistence uncertainty. No
destructive rollback or replacement launch follows unproved absence.

For unconfirmed session cleanup, the maximum deadline budget after helper death
is **20 seconds**: five for
session cleanup + five for disposition persistence + the existing ten-second
diagnostic-publication cap. Ordinary operating-system syscall/scheduling latency
is outside cooperative context guarantees. Exact helper termination keeps its
handle until reaped and precedes this envelope; an unreapable helper is never
abandoned to make the deadline. There is no post-expiry session metadata reread.

History uses the existing phase launch and outcome unproven; the receipt and
operator line name cleanup uncertainty and retain the diagnostic nonce. Inspect
with couch --show, then use the existing continue, resume from disk or start fresh
recovery outcome when fresh evidence permits it. No new recovery state is added.

### Failed recovery launch disposition (#714)

An acknowledged recovery launch gets one final registration look before it is
classified. If registration is still unconfirmed, Couch retains the per-thread
operation hold while checking the exact recorded helper and proposed terminal.
Only a reaped direct child, an absent owned process group and an absent exact
terminal release the failed creating incarnation. `ReconcileStart` owns the pure
evidence decision; `StartFailedConfirmed` applies its nonce/revision-fenced
transition and retains the original selection, park and thread for immediate
retry. It does not signal, delete a session or start a replacement.

Live/unknown group or terminal evidence, changed metadata and failed disposition
persistence all remain **Launch unconfirmed**; inspect the exact attempt before
another recovery action. The final disposition has one five-second context for
observation and metadata admission/write, followed by the existing optional
ten-second diagnostic-publication cap. Lock contention may refuse immediately.
The successful-launch path adds no disposition probes. See
`couchcore/recovery_launch_disposition.go` and `recovery_launch_failure_test.go`.

### Lifecycle failure evidence (#531, #683)

`launchfailure.go` owns `LifecycleFailure` and the legacy `LaunchFailure` aliases,
independently of ThreadRecord. `lifecyclefailure.go` centralizes display matching
of exact dispatched addresses/references and optional canonical slot paths; these
are diagnostic evidence, never lifecycle binding authority. Records retain the operation
nonce, dispatched target, optional thread address, slot path, park/launch nonce, phase, outcome and
bounded exact process identities with executable kinds. Holders never collect argv.
Generic failures have no inferred phase. Launch fields retain their existing
16 KiB raw output tail, 4 KiB per error and separate pre/post-cleanup exit codes.
Unresolved native pane queries also retain typed `zellij_*` evidence on that
same operation: exact raw stdout/stderr prefixes (4 KiB each), original counts
and truncation flags, nullable native exit, bounded argv/executable and version.
Version observation uses the failed query's executable and remaining proof
deadline. Concise notices/receipts keep raw bytes in the failure record, and
successful operations perform no version query or failure write.

Zellij0.45.1 can falsely miss a live session during its internal socket lookup.
`launcher/zellij_read_recovery.go` recognizes two native signatures only for
its existing idempotent read allowlist: exact missing-target stderr plus a
validated plain/colored listing containing that live target, or exact exit1
no-active-session output while the target socket is present. Reissues share
the original deadline and three-attempt cap; the successful reply still passes
strict parsing, independent owner agreement and exact-generation fencing.
Socket presence permits another read and never grants ownership. Remove these
compatibility signatures when the supported native release fixes lookup/IPC
routing and the private native controls are requalified (#761, #748).

`launchfailure_store.go` atomically publishes one JSON array at
`<namespace>/launch-diagnostics/failures.json` (0600), under its private file lock.
Retention trims to 64 entries, then evicts oldest whole entries until at most
1 MiB remains; it refuses a newest entry that cannot fit alone. The local
`lifecyclefailure_codec.go` preserves unknown raw values, including nested address
and holder/process members, through enrichment and rewrite. Known types, required
identity/time fields and recursive duplicate keys remain strict; legacy rows need
no new fields. No format version or global strictjson policy changes. Since #695
the codec sits on the shared `strictjson` object helpers (below).

`operationattempt_record.go` owns `OperationAttempt.Finish`, the sole terminal
recorder. Synchronous dispatch finishes its attempt directly. Queued actions keep
one owned attempt through preparation, guards, effect, adoption and readiness,
then the shared terminal adapter finishes it. Adoption, readiness, optional storage
and remote callbacks run on the existing bounded, joined workers. Run only reduces
the completed result; Stop cancels the worker publication context. Pre-enqueue
refusals stay immediate messages and are not logged. Nested calls borrow that attempt.
Launch-capture helpers return evidence without publishing separately. Typed error
chains retain errors.Is/As and a receipt carries `FailureNonce`, matching the
stored operation nonce after the receipt expires. #695 supplies teardown details
through `LifecycleFailureDetails`; this evidence does not authorize teardown or
finalize a park.

The recorder appends failure evidence once and performs no diagnostics IO on
success. It owns no coordinator, admission order, target-binding requirement or
write-time supersession. Missing cached paths still retain exact dispatched
references; preparation can add the actual dispatched thread address without
resolving or authorizing it for diagnostics. Unreadable records remain recoverable.
Optional nonce/publication failures preserve the original action result with a
warning rather than inviting another external effect.

Diagnostic address composition uses `completeFailureAddress`: it copies one
complete scope/tag pair atomically, falling back to the captured launch address
when outer dispatch evidence is partial. It never combines components from two
targets. Unresolved calls retain their reference/path with an empty address, so
strict record validation stays in force. The concise operator message also
retains an outer recorder's save-failed warning alongside the launch exit.

Pure `LastLifecycleSuccessAt(record)` is the sole success-time definition: verified
park time and a committed live incarnation's start time. LastActiveAt is excluded
because retirement may represent death. Row and show hide a historical failure
only when this existing record evidence is strictly later than its completion.
Missing/equal timestamps retain it, including a late failure after an earlier
success. #695 extends this pure seam with its persisted proven-teardown event.

`lifecyclefailure_projection.go` caches one bounded store snapshot by file identity,
size and mtime. Each inventory refresh projects the latest matching historical failure in
O(history + rows), without altering row state or action guards. A malformed or
unreadable store stays visibly unavailable. `couch --show` uses the same evidence,
including failures for an empty slot and legacy exact-address records. The existing
`LaunchFailures` result field is retained; `couchcmd/launchfailure.go` sanitizes
child text line by line.

Storage admission uses cancellable nonblocking flock attempts, with no background
waiter. Couch supplies an independent ten-second diagnostic admission context,
including after cancellation. The limit bounds waiting for a lease, not filesystem
IO after admission. Once admitted, atomic publication completes. Cleanup is not
postponed for diagnostic IO. A crash before terminal publication can lose that
attempt's in-memory evidence; older published history remains durable. Private
service conformance and a disposable-slot display check after an operator Couch
restart provide separate acceptance; this feature does not stage :0.

### Lifecycle primitives (#695)

**Additive records.** `strictjson/object.go` is the one strict-known,
preserve-unknown codec: `SplitObject`, `EncodeObject`, `AppendMembers`, `CloneRaw`
and `MergeRaw`. It refuses duplicate keys at any depth, and it writes preserved
values verbatim. The failure log and the thread record use it. The thread record
(`threadrecord.Record.Unknown`) carries top-level members it does not own through
every rewrite. `ThreadRecord.MarshalJSON` is the persisted encoding, so every
writer keeps them wherever it marshals. An older couch CLI running a direct-store
operation therefore cannot erase a newer field. Nested objects stay strict, and
there is no schema bump. Decode checks the binding's types only, so one bad or
newer-build binding refuses a resume (`resume-transcript-invalid`) and never hides
the thread.

New retained-raw editor endpoint state (#758) uses additive `RejectDuplicateKeysPreserveNumbers`, `SplitObjectPreserveNumbers` and `EncodeObjectPreserveNumbers`. These retain valid numeric tokens outside float64 range while checking syntax, duplicate keys and owned nullability. The caller decodes its exact owned fields into typed values; opaque child members stay raw. Existing strict readers keep their historical numeric acceptance. See [editor endpoint ownership](editor-endpoints.md).

**Transcript binding.** `ThreadRecord.Transcript` (`threadrecord/transcript.go`)
stores the agent's exact conversation. It keeps the native ID plus the root name
and relative path that the ledger binding's authorization proof names. A native ID
alone does not locate the file: Claude munges the cwd, and Codex adds a dated
folder. Resume, relaunch, recovery, startup and preview all go through
`Couch.bindTranscript` (`couchcore/transcript_bind.go`):

- It reads the tag's ledger by its known name with `ReadCurrentLaunchAt`, never a
  listing. A newer established root adopts, which covers pair's own context reset.
- It stats the bound file once. A missing file refuses with
  `resume-transcript-missing` and is never searched for.
- An unreadable or missing ledger keeps a located stored binding. The row note for
  that case belongs to pair#684's row notes.
- Only when no file is known does the old listing resolver answer, once:
  `bindByResolver` handles a session before its first turn, a legacy record, or a
  proofless binding row.

The resume start claim stores the result (`StartEvent.Transcript`) in its own
revision-checked write. `sessioninventory.RootTranscriptID` and `NativeRootNames`
are the pure path parsers and root names it validates with. The counted tests run
the production binder over `sessioninventorytest.FakeRuntime`: 1 ledger read,
1 stat, 0 listings.

**Lifecycle address.** `couchcore/lifecycleaddress.go`'s `LifecycleAddress` is a
slot's physical root, built by `NewSlotAddress`. A `:0` primary passes a zero
`SlotIdentity`. `Key` is `slot:<root>`, the same function as #683's failure slot
key. It has one kind: the operator chose one team per repository, so a team is an
ordinary `:0` slot.

**Lifetime lease (M2).** `lifetimelease` holds a shared flock at
`<namespace>/leases/<digest>.lease`, one per thread tree. The key is
`Key() + scope/tag`, per thread rather than per slot, because a store from before
one-thread-per-repository (pair#363) can hold sibling threads at one path.
- Couch prepares the lease and passes it in `COUCH_LIFETIME_LEASE`.
- The launcher's pair, `pair wrap` and session-watch each hold it. The wrapper
  strips it from the agent's env, so nothing the agent runs can hold the slot.
- Plain pair without Couch opens nothing.
- The wrapper also publishes the agent's controlling tty in `agent-ready`, read
  from the kernel process table (`procutil.Process.TTY`).
- The lease file is created by the launch and removed after any proof.
  Retirements with no proof leave theirs to `lifetimelease.SweepFree` at Couch
  start, which unlinks only free files. `Hold` re-checks that it locked the file
  its path names.

**Teardown (M2).** `couchcore/teardown.go`.
- `Teardown(thread)` quiesces the session and stops the registered wrapper. It
  then sweeps the lease holders and the processes on the agent's tty, by exact
  identity, with SIGTERM then SIGKILL, at most 3 rounds, within a 10s budget.
- It proves the tree down: the lease is free, the published agent (by exact
  identity) is dead, nothing is on the tty, and the session is absent twice. #692's listing omission is unproven, never absence.
- It then folds the park or start claim the dead owners left
  (`clearLifecycleDebris`).
- The tty is trusted only while the tree is up (the agent is listed or the lease
  is held), because a dead agent's device may be reused.
- `ProveDown` proves without stopping or folding.
- Failures are `ErrTeardownUnproven`, with a phase and holders, through #683's
  `LifecycleFailureDetails`.
- The agent's descendants that left its tty (setsid'd runners) are `LeftRunning`.
  They are never killed. They are registered in `<namespace>/orphans.json`
  (`LeftRunningRegistry`, tolerant and preserving, 256 cap). `couch --resources`
  shows them, and `couch --reap-orphans --confirm` ends them by exact identity.
  The resource sampler prunes vanished identities.

**Retire authorities (M2).** `couchcore/retire.go`. Every store retirement
(`archiveThread`, `ReplaceThreadExpected`, `replaceSlotCurrent`) takes a sealed
`RetireAuthority` and refuses without one:
- `TeardownProof`: that thread, that lifecycle key, at the proof's revision.
- `AdoptionAuthority`: survivor adoption, with every other owner proven dead.
- `UnreadableRecordEscape`: undecodable bytes, exactly those.
- `FreshLeavesRemnants`: a confirmed Fresh whose source could not be proven down.
  It stops only what the operator confirmed and starts beside the remnants. It is
  the only authority that may replace over an open park or claim.

Archive tears down after its quiesce. Reboot's archive-only escape and a fresh
slot start use `ProveDown`. A record retired by proof is archived with
`proven_down_at`, which `LastLifecycleSuccessAt` counts, so an emptied slot's
`--show` hides the unproven attempt before it.

Claude native resume validates an exact session through bounded UUID-leaf
probes under the authorized project roots, preserving duplicate and root-role
checks without walking unrelated history. Missing scrollback is separate from
native transcript identity; see [exact Claude root lookup](session-identity.md#exact-claude-root-lookup).

`LaunchOutputHandle` reuses the PTY child's existing ring (combined stdout/stderr).
ExecRunner forwards stderr through a bounded tee; stdin/stdout retain their
routing, but stderr is now a pipe, so isatty(2) is false. After direct child exit,
a one-second drain budget prevents descendants holding stderr from indefinitely
blocking reaping; subsequent descendant writes can receive EPIPE. Exit sampling
only reads Wait after Exited closes. The fake models direct-child completion
separately from process-group liveness. Unobserved exit stays unknown; an exit
first observed after cleanup is not attributed to a startup crash. Abrupt kill
before the recorder runs and output beyond the retained tail remain limits.

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
Socket endpoint hashes remain derived from the selected namespace. With explicit
isolation, broker and wrapper sockets live in the physical `COUCH_ISOLATED_ROOT/m`
directory; listener startup can sweep only that owned parent. Namespace or parent
escapes and overlong socket addresses refuse before listener effects. With isolation
unset, the existing per-UID socket addresses and startup sweep are unchanged.
The [shared native fleet helper](couch-testing.md) owns this boundary for disposable
multi-slot acceptance tests (#744).


### Durable numbered slots (#306)

The thread action menu's **Add slot** entry (#313) opens the existing start form
with the family's starting directory projected into the primary checkout and agent focus. `menuRepositoryRoot`
uses the validated slot primary root or scope-matched ordinary root; unknown
roots do not offer the shortcut. The same StartCreate preview/fingerprint and
repository admission checks apply as when entering a path manually, so Add slot
also fills the lowest free number (`:0` included) before adding one, and the
form lists the repository's parked threads when it adds one (#332).
Both entry points get their selectable agents from `launcher.AgentInventory()`
when `NewMenuState` is constructed (#529). Until an arrow makes a choice explicit,
the preview resolver owns the default; explicit choices remain generation-fenced
through asynchronous previews and the start commit.

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
(except the reserved bare `hub`, below)
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

Unset admission timestamps are omitted from wrapper receipts for older supervisors.
Wrapper commit requests omit this broker-only field entirely, while the broker
keeps its original timestamp in sender receipts, allowing older wrappers to receive.
Wrapper reply decoding remains strict; incompatible fields identify the wrapper slot
and advise restarting Couch. A failed commit reply never triggers another commit;
possible input effects remain uncertain. Broker release has its own delivery event,
so cancellation does not falsely report operator keystrokes (#594).

The automatic-input lease consumes the post-submit Empty observation that
confirmed a delivery; publishing its receipt does not create a new render fence
(#552). Turn-only confirmation and uncertain writes retain fresh-empty protection.
Codex startup/status field labels are case-insensitive, so `/status` transcript
output cannot misclassify its empty composer as unresolved startup.

An unexpected lease wait has a 30-second tripwire measured only during eligible
idle blocked attempts. Human drafts, pickers and input holds do not accrue that
time. Exceptional release requires a prior Submitted outcome, a complete Empty
snapshot and all input guards clear. Unknown composers and uncertain writes
remain blocked. Receipt details name missing-render, unrecognized-composer or
incomplete-frame evidence. Root regressions run with recovery disabled.

The wrapper's one bounded reporter records `automatic-input-stall` with a styled
screen capped at 40 rows/16 KiB in existing rotated wrap diagnostics. Its internal
`automatic-input-stall` RPC gets one two-second attempt. The supervisor validates
the current incarnation, then issues a console notice, an owning-hub message and
a desktop alert through the same presenter as `--notify`; hub-origin events use
the console notice instead of self-messaging. This does not grant general worker
notification authority. Full-binding occurrence IDs and a monotonic per-wrapper
sequence prevent repeated effects, including older duplicates after newer events.
Suppression high-water marks survive same-wrapper reconnect; one record per slot
is bounded by MaxActors, replaced by a verified new incarnation, and collected at
supervisor shutdown. Saturation refuses visibly instead of evicting duplicate
evidence. No offline queue, restart replay or uncertain-write retry is added.
Reporter work joins before diagnostic teardown. See `wrapcmd/automatic_input.go`,
`automatic_input_report.go` and `couchcmd/automatic_input_stall.go`.


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

`couch --notify [--title T] MESSAGE` is a light, hub-only operation. The message
service checks the live registered caller and its connected workspace against
the supervisor's hub root, then calls Console.NotifyDesktop through the shared
presenter writer. Other callers get advice to send to the hub instead. There is
no notification queue, receipt store or automatic retry on an uncertain write.
Supervisor notifications use ST-terminated OSC 777 directly on the console host
writer (stdout), outside the child OSC suppression path. A successful CLI reply
says `Notification written; desktop delivery not confirmed.` This confirms the
write only; the terminal protocol supplies no desktop-display acknowledgement.
For Ghostty the `desktop-notifications` setting must be true (default), macOS
Notifications permission and an alert style must allow Ghostty, and Focus must
permit the alert. Focused versus unfocused presentation needs operator observation;
Couch neither changes those settings nor retries to infer them.
Existing CLI messaging verbs omit notification-only fields for older supervisors.
The CLI distinguishes an explicit unsupported-notify decode refusal from missing
messaging registration and uncertain replies; only the latter leaves the effect
unconfirmed. Missing registration and unsupported commands provide restart advice,
never an automatic resend. The legacy socket regression lives in
`cmd/internal/couchcmd/messages_compat_test.go`.
Child OSC9/777/99 notifications are swallowed from all panes; only per-pane
counts enter the existing COUCH_TRACE diagnostics as `notify-suppressed`, and
ordinary BEL is unchanged. Each aggregate names the pane endpoint and cumulative
count; a fresh attachment starts a new counter. No notification text is logged.

**Notification list (bell) (pair#673, ephemeral lifecycle pair#733).** Ctrl+Return, Ctrl+Alt+a or the
status row's 🔔N badge opens the top-right panel. Enter/click opens and removes
an item, `d` dismisses, and the always-visible Clear all row or `c` empties it.
Three sources report facts through the message service: hub notifications,
blocked prompts (`publishBlocked`), and wait-watcher question episodes.
`couchcore.AttentionList` owns the pure transitions: removing a live condition
suppresses it until that kind/thread leaves its source set; a later occurrence
is fresh. Only notification events expire at 12 hours, on the existing activity
tick and fact/panel admission. Current live conditions do not expire. Restart
starts with no remembered events or suppressions and re-derives live state.
The legacy `NAMESPACE/attention/list.json` is neither read nor written; no saver
worker remains. Count and in-memory payload byte bounds still evict oldest
notifications first. The badge stays red while a visible prompt is outstanding;
the panel is operator-only. The hub's Needs-you section is the durable record.
See `couchcore/attention_list.go`, `attention_store.go` (byte-size encoding only),
`couchcmd/attention.go`, and `couchtty/console_attention.go`.

`couch --actors`, `--send-to repo[:N]|hub [--agent NAME] --message TEXT|--message-file PATH` and `--message-status ID`
use bounded Unix stream RPC, without constructing another mutable supervisor.
The send's `AdmissionTimeout` starts before sender authority checks and is
inherited by recipient admission, within `SendTransportTimeout`'s existing
response margin. Refusal bookkeeping reuses admission's families snapshot;
it performs no further store reads after admission expires (#634).
Targeted registry events inherit that same context as well as service lifetime;
a stalled registry loop cannot extend the send past its admission budget.
`--message-file` (pair#486) reads the body from a file at send time, so an agent writes a message
containing shell syntax with its editor and the call stays a bare `couch …` its sandbox exempts.
`--actors` and `--message-status` support JSON.
`Broker.StatusContext` resolves an exact visible ID before considering unique
prefixes in the caller's retained sends, received messages and refusals. It
uses the same locked visibility and waiting-line projection as exact reads;
ambiguous prefixes list sorted visible full IDs, while misses identify an
unknown lookup. Full-ID wrapper recovery after restart remains available;
prefix resolution is limited to receipts the broker currently retains (#635).
One delivery at a time per
actor, a FIFO behind it (pair#492: `ActorState.queue`, at most `MaxQueued` = 32),
the inbound breaker, and bounded terminal receipt retention limit the runtime.
Accepted pre-paste mail has no wait horizon while its exact receiver is connected.
An exact send to a busy actor, or one
whose wrapper refuses the reservation, is queued (`Enqueue`); the actor's worker
promotes the head (`Broker.promote`: Observe, Reserve, `Promote` with a fresh
commit deadline) after each delivery, retrying every `PromoteRetry` while the wrapper
is busy. A departing sender or recipient cancels messages not yet committed
with an actionable detail. A handed commit retains the conservative
`indeterminate` disconnect outcome: its last pre-paste observation cannot prove
that no input followed. Family selection requires fresh resting
branch and quiet-wrapper observations; exact sends allow occupied-slot
coordination. Quietness is never acceptance of repository work.
Receipt reads include the standing composer wait and the complete receiver-line
position (pending pre-paste head = 1). Active receipts survive retention sweeps;
terminal receipts receive a fresh one-hour retention period. A hung but connected
receiver therefore retains mail until it returns or disconnects, with no silent
expiry. Separate receiver workers remain independent.
Promotion failure cleanup retains that actor's reservation while its bounded
`Release` RPC runs outside the shared broker mutex. It clears only the same
attempt's reservation after reacquiring the mutex; another receiver's admission
and receipt reads can proceed during cleanup. Disconnect/replacement cancels the
old worker without transferring its cleanup to the replacement endpoint.

The bare target `hub` (pair#463, `couchmessage.HubTarget`) is resolved by the
message service, not the broker: `messageService.hubRecipient` rewrites it to
the hub's exact slot via `hubBinding` at send time, ahead of repository-name
and prefix resolution (a repository named `hub` stays reachable as `hub:N`),
and refuses when Couch has no hub, the hub isn't connected, or the sender is
the hub.

**The breaker is a leaky bucket (#433).** Each actor's pure `ActorState` holds
a burst (`InboundAllowance` 8) that regains one message per `RefillInterval`
(3 min), so a ping-pong empties it while reports at least that far apart never
do. A genuine
operator submission refills it, and so does a self-addressed message: the
operator's parley session-sync submit runs `couch --send-to` inside the slot's
own session, so the broker resolves the sender to the recipient. The slot whose
worktree Couch started from is the TL hub (`messageAuthority.hub`) and registers
with `HubAllowance` 64. `--actors` prints `allowance=remaining/capacity`.

**Registration follows lifecycle events, not polling (#365).** Each wrapper
holds one long-lived session on the namespace's `registry` socket
(`couchmessage.SessionClient` / `SessionServer`). The hello carries the binding,
and the kernel's peer PID must equal `Binding.PID`. The connection closing *is*
the death or exec event: Go sockets are close-on-exec, so a SIGUSR2 re-exec
reconnects as a new session even when its binding is byte-identical. A live
wrapper reconnects after Couch restarts without an attached or focused client.
The message service does not subscribe to Console pane events; attachment and
messaging have independent owners. No startup enumeration or reattach sweep is
needed for registration.

One pure `couchmessage.Registry` decides which bindings may receive:
- **Admission:** the wrapper hello immediately requests admission, including
  when Couch has no pane for its thread. `messageService` executes the registry's
  effects: one full authority check per wrapper admission, then broker
  `Register`/`Disconnect`.
- **Ordering:** the registry owns same-slot displacement, the newest session
  wins, and late admission results for a closed or displaced session are
  discarded. Local pane replacement or detachment does not change membership.
- **Failed checks:** retry on a bounded ladder (0.5–8 s), then the session goes
  dormant until something touches it: a submit, a reconnect, or a
  send that targets its slot.
- **Exact recipient:** the newest admitted session wins. A request already
  committed to the old incarnation may still finish there, or end
  `Indeterminate`; it is never redirected to the new one.

A full check (`messageAuthority.live`) verifies the wrapper's exact process
identity, workspace, launch nonce and ready agent, plus its positively listed
terminal. Observe, Reserve, Deliver and a sending caller use
`messageAuthority.current`: `messageThreadResolver` reads the retained working
path and performs the native session listing for that exact terminal, while the
PID file, ready file and session index must still name this wrapper, nonce and
session. These use-time reads can spawn the native listing command; they never
attach, launch, stop, or mutate a retained record. Admission additionally runs
the full ready-agent/process ownership probes. None runs on an idle timer.

The wrapper pushes observation changes coalesced to at most one frame a second,
submissions at once, and nothing while idle. `--actors` reads that memory. An
admitted idle actor stays known. The resting branch comes from the Console's
slot-git cache (`MessageSlotGit`), and only family admission runs git, per
candidate per send. The pre-#365 1 s heartbeat, 1 s reconcile and 10 s
verification window are gone. Wrappers from older binaries still send
`register` and are answered `unsupported` at no cost; their slots receive again
after a relaunch.

**Settled and build identity (#421).** A wrapper also reports whether a restart
now would interrupt nothing (`Settled`) and which executable it runs (`Build`:
content sha256, vcs revision and dirty bit, via
`couchmessage.BuildIdentityOfFile`).
- **What Settled means:** no turn is open (an atomic mirror of the
  notification lifecycle), no overlay is up (a fresh session's first message
  is its launch prompt since #522, so the turn it opens covers its briefing),
  the composer reads empty through the
  recognizer peer delivery uses, and nothing is pending.
- **When it is checked:** only after `wrapcmd.SettleInterval` (3s) without
  output or input (`peer_settle.go`), so a streaming agent pays nothing. Any
  activity unsettles the wrapper at once.
- **Negotiated per connection, because every session frame decodes strictly:**
  - A wrapper with a known build sends its newest hello (binding plus `Build`).
    The client tries v4, v3, v2, then plain hello, each downgrade only after an
    old broker's ack-less close; refusal, malformed ack and timeout do not downgrade.
  - Only hello-v2 and newer sessions may put `Settled` on activity and submit frames.
    A legacy session that does is ended as malformed.
  - `hello-v3` (pair#465) is hello-v2 that may also put `Picker` on those
    frames. A v2 session claiming
    `Picker` is ended as malformed, so a new wrapper never breaks a Couch that
    hasn't been restarted yet.
  - `hello-v4` (pair#632) adds optional `BackgroundWork` on activity/submit;
    nil is unknown, false known none, true native background work. Older sessions
    claiming it are ended as malformed; downgraded clients omit the field.
- **`Picker` (pair#465):** whether an input-holding picker is up: the
  wrapper's `pickerActive` latch (closed by Enter or by the composer coming
  back, pair#464). Unlike `Settled`, it is not gated on quiet. `setPickerLocked`
  reports each change under `overlayMu` the moment the latch flips, because a
  picker's own repaints (Claude Code's auto-deny countdown) keep output
  flowing. `SlotLiveness.Picker` holds it, and nil means unknown.
- **Field placement:** neither `Binding` (the actor key) nor `Observation`
  (which also crosses the delivery endpoint) carries the new fields.
- **Reading it:** the registry keeps the newest admitted incarnation's claims,
  and `messageService.SlotLiveness(slot)` reads them lock-free. A nil `Build`
  or `Settled` means unknown, never idle.

**What delivery does and does not promise (#365).** Messaging promises
at-most-once input to the wrapper's PTY, not exactly-once task execution. The
remaining uncertainty is stated, never papered over:
- **Wrapper receipt.** A `submitted` receipt says the envelope was pasted,
  submitted, and the submission was seen to land (below). It says nothing
  about whether the agent read or acted on it. Check the transcript or the
  issue state for that.
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
- **Not admitted (#618).** A send that never reaches `Admit` leaves a `refused`
  receipt naming why (a failed ownership check, or "admission timed out" when
  the checks outrun `AdmissionTimeout`). Admission is the only way into an
  inbox, so a refusal is definite: the send answers `refused` with the reason,
  and `--message-status` reads it for 10 minutes. Refusals are capped at 64
  (oldest evicted), never count against `MaxReceipts`, and a resend with the
  same ID is a fresh attempt. A send's exchange budget on both ends of the socket
  is `SendTransportTimeout` (admission plus a second), so the verdict arrives.
- **Liveness behind the authority (#618).** The use-time ownership check runs
  at every admission and every queued-message retry (`PromoteRetry`, 1 s). Its
  terminal listing reads `launcher.ListingPoller`: one supervisor loop reads one
  validated full `zellij list-sessions --no-formatting` reply every second (#711),
  deriving names and exited state from that same observation. A snapshot older than 5 s
  refuses as "zellij liveness is stale". The thread record, PID file, ready
  file and session index are still read per check. Recovery and relaunch
  decisions keep their direct zellij reads.
  Malformed refreshes preserve the last good timestamp instead of renewing authority.
  An unrelated session starting between two native calls cannot create a disagreement:
  there is no paired short listing. Direct inventory readers use the same full parser.
- **Input during admission (#711).** A recipient whose broker actor and both live
  registrations are unchanged can become busy after its wrapper accepts a reservation.
  Admission captures the sender's original broker actor under lock and retains it through
  routing, reservation and queue fallback; same-binding reconnect is a new lifetime.
  Operator input invalidating that observation uses the existing exact-target queue;
  disconnect or replacement still refuses stale work. Refusals identify caller lookup /
  current validation or recipient lookup / observe / reserve, and the CLI never infers
  that restarting Couch is appropriate from a generic unavailable reply. Unknown or
  malformed evidence grants no new authority, and uncertain sends retain receipt-first
  recovery. The three causal regressions and 8/16-session query budgets live in
  `broker_input_invalidation_test.go`, `zellij_listing_churn_test.go`, and
  `message_phase_test.go`.
- **Receive state in `--list` (#618).** `couch --list` asks the running
  supervisor once (the unauthenticated `refusals` op, 250 ms, best effort) for
  each slot's latest unsuperseded refusal and prints it under the row as
  `can't receive: <reason>` with the one command that clears it. Ownership and
  registration sources carry `couchmessage.RelaunchRequiredError`; the broker
  records a safe `{reason,class}` projection and the list chooses relaunch from
  that class (#633), never diagnostic phrases. Timeout/cancellation and unknown
  classes choose resend. The op contains no sender binding or body. Mixed-version
  response-shape mismatch omits these best-effort annotations, leaving the ordinary
  list intact.
- **Queued hub batches (#648).** Verified Claude hubs prepare the head
  without input, then the existing wrapper input writer offers readiness at its
  ordinary safe-composer decision. The broker freezes the current capped FIFO
  prefix under its mutex, including reports arriving during the held head's
  wait. One finalization enables one combined paste/submit; the writer rechecks
  input safety before pasting. Complete unchanged envelopes are separated by
  `\n\n---\n\n`. `COUCH_PEER_BATCH_MAX_CHARS` defaults to 16,384 Unicode code
  points including headers/separators and accepts 1–32,768; invalid values refuse
  startup. Over-cap single messages proceed alone, with remaining reports kept
  for the next composer. Actual encoded requests retain the existing frame cap.
  One confirmation updates all original receipts atomically; each member remains
  queryable and recoverable from the wrapper's bounded 64-ID recent store.
  Before freeze, cancellation removes only that queued sender's mail; after
  freeze, members share one cancellation job/outcome. Uncertain writes never
  replay. Ordinary receivers, Codex and Qoder retain single-message delivery; an old
  wrapper's explicit pre-commit unknown-operation refusal also selects the legacy
  path, while transport/decode errors never trigger fallback. See
  `couchmessage/batch.go`, `broker_batch.go`, `endpoint_batch.go`,
  `wrapcmd/peer_batch.go` and `TestPeerBatchRuntimeSocketFIFOAndReceiptRecovery`.
  Tall Codex singleton recognition (#652) shares viewport-bounded prompt/footer
  geometry with ordinary Return. Beyond the local 20-row domain, a complete
  visible styled prompt, clear continuation edges and the existing trailing
  native footer signature are required. Clipped prompts, partial frames and
  unrecognized footer wording remain unknown. Owned Codex 0.163.0 PTY checks
  qualified 33 visible rows at 100x60, exact text retention after Return added a
  newline, and a separate explicit submission. Codex batching remains disabled;
  operator Return/submit smoke is scheduled for the hub's next restart. See
  `wrapcmd/composer_recognizers.go`, `codex_tall_composer_test.go` and the opt-in
  `TestPeerLiveConformance` paste-tall scenario.
- **Peer turns in the Pair log (#618, #648).** When a delivery ends `Submitted`,
  pair-wrap appends the exact envelope it pasted to the thread's Pair log
  through the managed locked writer (`pairlog.PersistPeerTurn`, append ID
  `couch-peer-<delivery ID>`). The session watcher correlates it like any draft
  turn, so a slot driven only by peer messages confirms its binding and can be
  relaunched. The draft pane's history and `!!` skip `couch-peer-` entries.
  A batch is one exact combined turn, using its head ID for idempotency, so
  transcript correlation hashes the same input the agent received. One launch
  records at most 8 turns and 256 KiB of aggregate envelope payload: a binding
  needs one completed round after its launch, and entries follow the Pair log.
- **Relaunch rides out a slow quit (#618).** `--relaunch` waits up to
  `RelaunchParkWait` (60 s) for the agent's quit inside its operation before
  it offers recover/abandon; a plain park keeps its shorter wait.
- **Retries.** The CLI mints a fresh ID per send and tells the sender not to
  retry an uncertain one. A reused ID that reaches a wrapper that remembers it
  adopts the recorded outcome and is not pasted again.

Each wrapper endpoint conditionally reserves its observed input generation,
then accepts one delivery commit. The broker polls outcome receipts; it never
retries PTY input after uncertainty. Pair's input owner arbitrates ordinary
typing, image admission and automatic paste/submit.

**Delivery: submit like the draft pane, confirm after (pair#427).** Shared
confirmation reads composer/turn evidence without projecting the body. Exact
text projections broke for #418's long Claude pastes and boot-time rendering.
Qoder's bounded submit-key retry below has a narrower ownership check: the
complete pending composer must still match Couch's known envelope.
- **Before the paste** the gates are unchanged. No other automatic input is in
  flight, no operator keystroke arrived in the last second, bracketed paste is
  on, and a recognized composer reads empty. Unknown or occupied composers wait.
  Receipt Detail names incomplete frames, hidden cursors, or unrecognized
  geometry/input without recording screen contents. The terminal resize owner
  preserves existing evidence on unchanged synchronized geometry; an actual
  resize or retry after an aborted resize still requires fresh cursor evidence
  (`peer_resize_test.go`, pair#735). A running Claude shell footer alone does
  not block its free composer. Supervision-log health is tracked in pair#736.
- **A busy agent is read between frames and pasted mid-turn (pair#438).**
  A streaming agent nearly always has output in flight, so waiting for none
  starved deliveries for whole turns. Claude Code frames each repaint as a
  synchronized update (DECSET 2026; Codex's captured screens do too), and the
  terminal model tracks it, forgetting it on a reset
  (`terminalModel.FrameComplete`). Output in flight waits only when the model is
  mid-frame or the stream never framed one. The policy is to paste mid-turn:
  Claude Code (measured; Codex unmeasured) queues the message behind its turn ("Press up to edit queued
  messages"), and the composer clears. That is the ordinary confirm evidence,
  so a queued message ends `submitted`.
- **Operator input means keystrokes (pair#428).** The wrapper classifies its
  own stdin (`terminalReplies.operatorKeystrokes`), never the agent's screen.
  Text, control and editing keys, kitty/legacy key encodings and an operator's
  bracketed paste count. Focus events, mouse reports (SGR, urxvt, X10) and
  terminal reports do not, whether solicited or not. Those reports are
  DA/DECRPM/DSR, color-scheme `?997;N n`, kitty flags, window `t`, and OSC/DCS
  replies. A CPR (`CSI r;c R`) is a report only when solicited, because
  modified F3 looks the same. Those bytes still reach the agent unchanged. A
  lone ESC resolves as a key at the 30ms flush, so it no longer stalls a
  delivery. Other panes can't block: each has its own PTY. A draft send reaches
  this stdin only as focus events and its paste.
- **The composer must be in view (pair#428).** Before pasting, the wrapper
  aligns zellij's scrolled viewport (`dump-screen -p terminal_$ZELLIJ_PANE_ID`)
  with its own model's scrollback and screen (`peer_viewport.go`). It pastes
  only while the cursor row is in view, so a small scroll doesn't block. A
  delivery never yanks the view from scrollback the operator is reading: it
  waits and goes when they scroll back. One reading lasts 500ms and is taken
  off the dispatch locks. When the wrapper can't judge (no zellij, a failed
  dump, no alignment), the composer counts as in view.
- **A draft send waits for a delivery in flight (pair#428).** Between the
  robot's paste and its confirmation, an operator chunk that opens a bracketed
  paste is held, along with everything after it. The hold lasts at most
  `PeerDraftHold` (1s). The held input is then replayed with its original
  spacing, so the draft's paste, settle and Alt+Enter keep their rhythm. Both
  messages submit, in order. Past the cap the paste counts as operator input.
  In the other direction, a delivery already waits for a draft send through the
  1s settle and the occupied composer, and it never pastes while held input is still queued. A typed keystroke never starts a hold. Any input that arrives during a hold, keystrokes included, queues behind the held paste to keep order.
- **Paste, fixed delay, submit:** after the bracketed paste the wrapper waits
  `wrapcmd.PeerSubmitDelay` (150ms; the draft pane uses 100ms), then sends the
  submit key. Past the paste, the poll runs every 50ms instead of every second.
- **Confirm after the fact, agent-agnostically:** a delivery ends `submitted`
  when the composer was seen occupied after the paste and now reads empty, or
  when a turn opens that was not open at the submit. Otherwise it ends
  `indeterminate`, with a detail that starts `uncertain:` and states the
  evidence. Examples: "composer still holds text; no turn started", or
  "composer empty but never showed the paste".
- **Stops after the paste:** operator input, an image, a dialog, the deadline
  or exit ends the delivery `indeterminate` (`uncertain: … after the
  paste/submit`). It never ends `expired` or `cancelled` while text may still
  sit in the composer.
- **The input deadline starts at paste.** `Message.Deadline` bounds the initial
  reservation/commit exchange; promotion refreshes it, and each endpoint RPC
  remains bounded. After commit, accepted mail waits for receiver readiness
  without a pre-paste deadline in the wrapper, broker model or status poll.
  From the paste, the wrapper bounds submit and confirmation by
  `DeliveryTimeout` (30s). A post-paste expiry remains indeterminate and names
  the standing waits with durations; it never replays the body.
- **Operator holds retain mail (pair#629).** A draft, scrollback or picker
  remains `held` with its reason; unknown composer, paste mode or booting waits
  remain `queued`. No hold-credit clock or 30-minute cap can discard the mail.
  Pre-paste waits keep only the standing reason; duration history belongs to
  the bounded post-paste window, so changing picker titles cannot grow a
  history throughout an indefinite wait.
  A picker detail is `blocked by a picker: <title>`, where the title
  comes from `couchmessage.ReadPicker` over the wrapper's own tail and is read
  once per picker. The check runs ahead of the transient waits, so a
  repainting picker keeps its reason. The message is delivered when the picker
  closes and the composer is safe.
  While held the wrapper's receipt is `held` with the reason;
  `RemoteEndpoint.Deliver` hands each interim receipt to the broker
  (`DeliveryProgress`), so the sender's `couch` status reads `held` too.
  Queued and held status polls run at the existing one-second hold cadence;
  active input confirmation uses the short cadence. This bounds idle wait IO
  without extending or retrying any input operation.

Receiver profiles exist for Claude Code, Codex CLI and Qoder CLI at any installed version
(`peerReceiverAgents`; the exact-version allowlist was removed in #360 after
auto-updates silently dropped slots). Fixtures under `wrapcmd/testdata/peer/`
and `TestPeerLiveConformance` were captured on Claude Code 2.1.286 and Codex
CLI 0.159.2; per-version evidence from daily use is #368. Every captured paste
shape must read as an occupied composer
(`TestPeerComposerCapturedPastesReadOccupied`). Qoder's profile shares the Return recognizer's ruled-box geometry,
reads input from its indented origin, and distinguishes its captured grey
placeholder from authored text using the painted cursor and matching hint style.
Qoder idle (1.1.71) and folder trust (1.1.66) fixtures are frozen from pair:4;
tests cover registration, live-tail transport, settling, trust refusal and one
paste/submit through the real broker seam. Native Qoder submit and reply, live peek, actor registration and idle settling
were confirmed by the hub in the authorized #550 pair:4 smoke. Every launcher agent has an explicit receiver policy,
checked against AgentInventory; Agy, Muse and Grok remain disabled pending
captured peer-input support. Human Couch acceptance remains a separate step.
The shipped skill's receiver sentence is pinned to that policy by
`TestSkillNamesReceiverProfiles` (#558), so an inventory change fails a test
instead of leaving `couch --skill` stale.

Claude and Codex single-line suggested prompts use shared ANSI faint styling
and the cursor at the input origin, independent of wording or RGB color.
Delivery reads the current composer, not a sticky human-draft flag. Buffered
input/output blocks inspection; newly forwarded human input gets one second to
settle before inspecting the screen. Visible draft text still blocks delivery,
while an erased draft can become eligible without submitting it. A polling timer exists only while a message is pending (1s queued, 50ms
past the paste); events also wake checks. After the horizon, the broker allows
two seconds for read-only receipt collection; this never extends the wrapper's
deadlines. Waiting
receipts report the blocking guard and retain that reason on expiry. Senders
query `--message-status` for final outcomes; admission returns before delivery.
Peer delivery, the one automatic writer since #522 retired the orientation
paste, holds automatic-input ownership across the entire paste/submit/confirm
transaction. The next delivery waits for a fresh, empty composer after that
transaction terminates, including cancellation.

Qoder additionally requires a complete post-submit rendered empty composer to
confirm (#569); an unrelated turn flag cannot confirm while text remains.
Within the existing 30-second delivery window it can attempt CR at most three
times, at least 500 ms apart, without repeating the body. A retry requires the
complete composer to equal the expected envelope's word-wrapped layout and no
admitted human input since the paste. Unknown, changed, clipped or unsupported
renderings decline retry; the native pending fixture qualifies the ordinary
wrapped restart notice.

Unconfirmed Qoder input enters explicit operator recovery at terminal receipt
publication. One authenticated tripwire alerts its owning hub; exact pending
envelopes get the instruction to press Alt+Enter or clear the composer, while
unverifiable input keeps uncertainty wording. Later deliveries refuse promptly
without pasting. Only a later complete empty render releases this quarantine,
observed through normal output wakes even without another delivery. There is
no idle polling, automatic deletion or submit after the old deadline.
`wrapcmd/peer_qoder_submission.go` owns the Qoder evidence and quarantine adapter;
the shared delivery and automatic-input reducers own its effects and lifetimes.


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

Every public verb has a relative cost, `light`, `seconds` or `heavy`, in one
table (`couchcmd/cost.go`, pair#490, from timings in that issue's Log).
`couch --help` renders it at the end of each verb's line through `verbLine`,
which panics on an untagged verb; the skill's "What each verb costs" section
lists the same tiers, and `cost_test.go` keeps both equal to the table.

#### Delayed peer refusal capture (#791)

The wrapper records one `PEER-wait` diagnostic when the active delivery head has
remained refused before paste for 15 seconds, checked by its existing queued
poll. Later refusal polls or reason changes do not repeat the capture; paste,
terminal receipt, release or head replacement ends that diagnostic lifetime.
Queued followers have no separate refused-screen observation. The record uses
the existing per-slot wrap-events JSONL from `artifactpath.Paths.WrapEvents()`;
no per-delivery file or extra polling worker is added.

The record preserves the exact refusal detail and its cursor, geometry, styled
cells and frame/synchronization evidence. Composer refusals use the snapshot
that made the decision; an earlier non-screen guard captures a coherent screen
at that refusal and identifies its generation. `evidence.last_resize` records
the latest admitted resize attempt/completion times, requested columns/rows,
whether geometry changed, and prepared/committed/aborted outcome. It is copied
under the same terminal mutex as the screen and generation. A same-size retry
after an abort can require fresh cursor evidence while reporting changed=false.
Absent last_resize means this wrapper has observed no admitted resize. The whole encoded record is at
most 64 KiB, including its envelope, with explicit screen truncation/omission
metadata when the full screen cannot fit. Screen contents may be private: they
remain local diagnostics and are never automatically uploaded publicly. The
capture adds no message body or argv beyond the observed screen.

Encoding and writer enqueue happen after admission/mailbox locks are released.
The proxy's existing buffered diagnostic owner handles flush, close, rotation
and [retention/GC](storage-retention.md). `PAIR_WRAP_EVENTS=0`, an unavailable
writer or a failed capture can lose evidence, but cannot alter delivery safety,
receipt state or recognition. Recognition remains unchanged. The controlled
resize-then-silence replay demonstrates a hidden cursor can hold delivery until
a repaint; #810 plans recovery through redraw and re-observation. Existing
recordings lack precise resize timestamps, so they cannot establish historical
post-resize silence or keystroke causation.

### Lifecycle system notices (#574)

`couchtty/console_lifecycle.go` observes final successful operation results after
Console adoption, successful park results carrying `VerifiedPark`, and confirmed
needs-relaunch inventory transitions. It never infers agent exit from an attachment
ending, EOF or an orphaned session.

**Native agent exit (#657)** has one source: the wrapper that owns the child.
- **Wrapper:** after `group.Wait()` reaps a child that exited on its own, and before its registry session closes, the wrapper (`wrapcmd/native_exit_report.go`) sends one broker RPC, `native-exit`, carrying `{LaunchOrdinal, AgentIdentity, ExitCode}`, bounded at 2 s.
  - Its shape follows `automatic-input-stall`.
  - A SIGUSR2 restart or any wrapper-initiated stop (the `stoppedByWrapper` flag) sends none.
- **Couch:** `couchcmd/native_exit.go` admits the report only from the authenticated current wrapper (`switchCaller`) when:
  - the ID matches `NativeExitID(binding, ordinal)`;
  - the ordinal and identity match the ready record (`authority.child`, file-only), which fences the child before a restart, since the binding survives that exec;
  - the session is still connected;
  - the same (binding, launch) hasn't already been reported.
- **Notice:** Couch then calls `Console.ObserveNativeExit`, which emits an `exited` fact with `Mode` "code N" in the Console's observation sequence.
- **Not covered:** agents without a registry session (agy, muse, grok) produce no exit notice.
- **Compatibility:** an older Couch answers `invalid-request` and the wrapper ignores it.

Completion carries cached
slot/branch metadata and known initiation. The existing `ReattachDone` transition
closes a startup cohort; no debounce or additional native discovery is used.

`couchcore/lifecycle_notice.go` owns the pure bounded collector and its publication
transitions. It keeps at most 128 current facts, coalesces unsent changes, freezes
immutable numbered parts within the existing body limit and retires each part
from its receipt. `couchcmd/lifecycle_notices.go` routes through existing hub
ownership and the message broker/#648 batching. A fact whose slot identity isn't
cached yet (ordinary target or none) waits in a bounded `unresolved` set: 128 paths,
newest per path, oldest evicted at the cap, cleared at restart drain and on ownership
change. It resolves on the next refresh once the wrapper's verified workspace is
cached. This matters because a launch is reported before the wrapper connects
(#693). An ordinary fact expires after `lifecycleOrdinaryWait` (2 min), since its row
may not be a slot at all. Restart drain reports only the untargeted facts still
waiting, as lacking verified slot metadata. Its callbacks only reduce bounded
memory; owned admission workers and bounded receipt-memory checks stop/join with
the service. Busy/absent hubs retain current facts; input uncertainty never permits
replay. This is a Slots projection, not an audit log. `COUCH_LIFECYCLE_NOTICES`
accepts `on` (default) or `off`; there is no additional ownership/mute store.

The confirmed `executeHubMove` producer drops the old unadmitted projection and
queues one check-Slots line for the new owner. Admitted receipts finish at their
original hub without replay. The existing cancellable hub effect gate serializes
bounded admission against moves; the ownership mutex is not held across IO.
Projection generations fence jobs extracted before the drop. Restart collection
survives the drop, and only genuinely new lifecycle facts follow the new owner.

Hub slot-operation admission (`handleHubSlotOperation`, pair#726) is a cheap
gate on warm sources, inside its own 1.5 s budget, below the 2 s exchange. The
single deadline spans initial authentication, target resolution, queue-name lookup
and enqueue, including move and operation-status requests. The caller comes
from its connected workspace (`connectedWorkspace`), and the target
from `hubTargetMemo`. Admission never falls back to a synchronous fresh caller
workspace or target resolver: native cancellation can itself outlast the exchange.
A missing caller proof says to reconnect; a missing memo says it is unavailable.
A pending memo flight returns wire code `warming`, no operation receipt and a
positive `RetryAfterMillis` (500 ms). The CLI repeats only that typed nothing-queued
refusal, keeping its request ID and intent, for at most eight attempts inside a
30-second total admission budget. Each exchange and sleep is capped by the
remaining budget, and progress says it is waiting for the catalog to warm.
Transport errors, uncertain replies and all other refusals never trigger admission
resends. Queued work and its fresh proofs retain their own queue context.
The memo maps a target reference to its `HubSlot`.
One single-flight discovery per repository fills all its verified numbered
slots under unambiguous names and aliases. Memo reads and queue-key selection
use the fresh resolver's shared exact-sibling-directory precedence rule before
enrollment lookup, including prewarmed aliases. The service captures the
supervisor's fleet context once at startup; if it cannot establish that context,
it retains per-reference resolution rather than guessing from enrollment. Startup prewarms the distinct hub
repositories without delaying service startup; cold concurrent requests join
the same flight. Missing slots, primary slots and unresolved repository names
fall back to the existing per-reference resolver. Both kinds of flight share
the eight-resolution cap, run under the service lifetime and join at `Close`.
Entries are used for 10 minutes and every fresh target resolution refreshes its
entry; memo writes sweep expired entries and cap them at 256. Admission waits
only within its budget. Otherwise admission returns the typed warming refusal above, and any overrun names its phase. A failed
resolution answers with its own error for 30 s before retrying. Move-slot
admission (`handleHubMove`) names its slots from the same memo, within the same
budget. The queued move then re-resolves each slot fresh and refuses if any
differs, because those slots are the move's payload. A warm
admission spawns no `sdlc` or `git`. The queued guard still re-proves authority
on fresh discovery in prepare and at the effect gate, so a stale memo admits at
most a job that refuses before any effect. `OSSlotCatalog.Discover` verifies
slot rows concurrently, at most eight at a time. It keeps each row's
replacement fence and fails the whole inventory on a fatal row. That cut an
8-slot discovery from 3 s to 1 s.

`couchmessage.Message.Origin` is immutable: peer is omitted/zero, system is created
only through internal server admission. External requests cannot carry origin.
Envelopes, batch matching, deduplication and canonical receipts preserve it.
Unknown system receipts returned only by a wrapper cannot be imported as server
provenance after restart/eviction; retained promotion needs the matching original
server-admitted queue head. Copied headings remain peer body text. No wrapper PID
authentication or new terminal frame is introduced. The hub verifies the canonical
receipt and updates Slots without replying; source-owned TL help is ariadne#433.
Worker startup availability also uses the internal system producer.

### Unanswered-report rules (#530)

`couchcmd/supervision_model.go` owns volatile report episodes and the per-lifetime
set of slot addresses already offered a startup resend prompt. The coordinator
in `supervision.go` uses the existing 15-second wait-watcher tick. It does not
change attach/resume or registry lifecycle transitions.

Authenticated CLI send attempts to `hub` or its exact live address start an
episode, including hub admission failures visible to this supervisor. A small
leading-word completion hint keeps a landing report eligible after return to a
resting branch. A confirmed hub CLI delivery initiated after the report, or an
increase in the broker's accepted operator submission counter, clears it.
Queued/uncertain replies and supervisor-generated prompts do not clear it.

Known, settled workers quiet for 30 seconds with no incoming message pending get
a hub nudge at 15 minutes and one Couch operator notice at 30 minutes. Ordinary
waiting notices coalesce with the nudge; picker/prompt notices keep their existing
behavior. The same `wait-notice-off` file mutes report reminders. Operator
escalation does not require the hub still to be connected and never generates a
desktop alert. Each notice names one action: inspect with `couch --peek`.

When the hub becomes available, each connected worker address gets one normal
Send attempt asking it to reconsider its last unanswered report. The ordinary
busy-recipient queue is unchanged. Attempts are marked before sending, so an
uncertain result or reconnect cannot regenerate the prompt. Original reports
are never replayed or retained across Couch restarts. The same tick reads terminal
broker receipts, reports startup cancellation/expiry/uncertainty once, and drops
the completed send metadata. There is no separate capped report history. Immutable system origin
keeps generated notices out of session-file obligations; matching prose from a real agent
is not exempt. All episode/attempt metadata dies with the supervisor; disconnected
bindings lose their report episodes. The separate fact log below does not drive
these rules. Parley's sole file-ping watcher is parley.nvim#321.

### Supervision fact log (#543)

The message service records best-effort JSONL at `<namespace>/supervision.jsonl`.
`supervision_facts.go` owns its v1 schema; `supervision_factlog.go` owns one bounded
writer, and `supervision_fact_events.go` adapts existing observations. There is no
new watcher, candidate tracking, or online labeling.

`couch --resources [--json]` adds request-time `supervision_log` health (#736 M1).
The existing fact owner captures bounded sanitized backend errors and acknowledged
write counts; `health()` copies owner memory under its mutex, never reading the
file or waiting on the sink. Pure `factLogLifecycle.advance` distinguishes opening,
working, unavailable, failed-write and closed. A later independent successful row
restores working; the failed row is never retried and its last error remains
historical. Closed absorbs late IO completions. Counters report offers, complete
sink acknowledgments and existing drops, not power-loss durability or a drained
queue. The fact file's default location is `~/.local/share/pair/couch/supervision.jsonl`.
Retained `couch --message-status ID --json` receipts, rather than this fact stream,
explain per-message waiting decisions. The 2026-10-21 readout exposed an open
failure: persisted device16777232 versus observed16777229 with identical inode,
size and nanosecond mtime. #736 closes around health instrumentation; #774 owns
this measured repair and the hub-coordinated fresh-write acceptance.

`diagnosticlog/checkpoint_identity.go` classifies persisted generations as exact,
device-only or mismatched. Under the existing log flock, a current checkpoint
without a pending intent is durably rebound before append admission; the old
reservation is untouched. Preview uses the same persisted comparison read-only.
Collection carries preview's first observed generation privately through strict
revalidation, then may rebind retained metadata before a new deletion intent;
current rows and all already-pending transaction recovery keep strict identity.
A device change between observations within one operation is refused. Inode,
size, nanosecond-time and uncertain-tail mismatches never authorize a reset,
truncation or new append. Errors identify the mismatched component and recovery
stage. There is no new format, retry loop, payload reread or durable artifact.
A rebind adds one bounded existing atomic metadata publication per touched
current/retained generation; normal append and page bounds are unchanged.

The optional health field requires the rebuilt CLI with the new supervisor;
older strict resource decoders reject it. A new CLI accepts an older server's
absent field and renders health unavailable. No negotiation, retry, persisted
format migration or decoder relaxation is involved; other operations' responses
and admission decisions are unchanged. `resources_projection.go` retains this
bounded value when trimming detail rows; both projection and cloning isolate it
from the cached process sample.

Rows record:

- `hub_delivery`: endpoint completion to the hub, wording hint (landed,
  waiting-for-operator, progress, unknown), generated flag and confirmed,
  uncertain or cancelled outcome. Queue-only failures have no delivery row.
- `session_write`: one authenticated best-effort report after a guarded write.
  It carries the owner captured under that write guard; the caller must match
  the owner. Cross-owner writes, missing launch identity, old/unavailable Couch,
  and lost report calls leave gaps in this coverage. No file paths are sent.
- `operator_submit`: accepted submission counter advances (a jump is one
  observation of a total, not invented events). Stale counters are not events.
- `slot_state`: changed connected/settled/picker observations, including removal
  and replacement. Unknown booleans are null; no recovery transition changes.
- `snapshot`: once a minute on the existing tick, bounded current actor facts,
  including hub/empty inventory, cached resting state and quiet age.

Every row has schema version, supervisor generation, increasing sequence,
monotonic elapsed milliseconds, UTC observation time and cumulative dropped-row
count. Slot, incarnation, sender and message IDs are lifetime-salted opaque hashes.
No message prose, paths, nonces or transcripts are recorded. Different lifetimes
cannot be joined by identity. Hints are text classifications, not proof of intent.

Producers never perform disk IO: they offer bounded rows to one 256-entry writer
buffer. Rows exceed neither 32 KiB nor 128 snapshot actors. Full buffers, open/write
failure and shutdown drop rows without retry; later sequence gaps expose missing
rows. Logging failure cannot fail delivery. The CLI's one write-report call is
limited to 200 ms and cannot invalidate the file write. A missing log means
unavailable evidence, never zero activity.

The shared diagnostic writer registers the stable path with Pair GC and
persists a 64 MiB current-generation cap. Whole rows refuse when proven
rotation cannot proceed. Daily and size rotation preserve every closed
supervision generation behind `KeepDiagnosticsUntilBackup`, shared with
wrap-events, adapt and fullscreen diagnostics. Count and age eviction remain
disabled pending ops `history-backup` piece 6 (archive before evict); total
history growth is intentionally unbounded. Pre-existing oversized evidence
is preserved. The writer and buffered tail end with the supervisor; historical
generations remain on disk. See [storage retention](storage-retention.md).

`python3 scripts/supervision-labels/labels.py LOG...` derives ten-minute labels
from selected local segments. See its [skill](../scripts/supervision-labels/SKILL.md)
for the input limits and interpretation. True means a write was observed for the
hub subject; false requires contiguous same-generation coverage strictly beyond
the inclusive ten-minute endpoint; missing, corrupt or dropped evidence gives null. False is only “no
write observed,” not neglect: failed telemetry calls before admission cannot be
reconstructed. The script writes JSONL to stdout and changes no runtime behavior.

### Waiting notices to the hub (#451)

A slot that waits for someone sends nothing, so delivery robustness can't
report it. `waitWatcher` (`couchcmd/wait_notice.go`) ticks every 15s
(`waitTick`) on the message service and tells the hub instead.
- **Inputs are memory:** the broker listing (`LastActivity`, `Pending`, the
  slot-git resting view) and the registry's `Settled` (#421). A slot quiet past
  the threshold (`COUCH_WAIT_NOTICE_AFTER`, at least 1m, default 10m; any
  other value refuses Couch's start) with a
  Settled claim costs one tail read from its own wrapper (the peek path); a
  nil Settled (pre-hello-v2 wrapper) is unknown and never waiting.
- **Waiting (`classifyWait`):** unsettled (a dialog or a draft: byte-quiet
  that long with something open), or settled with the
  last prose line ending `?`, or settled off its resting branch with no message
  in flight. Settled on the resting branch after a normal ending is not.
  `lastProseLine` reads above the composer prompt (`❯`/`›`) and skips chrome
  paragraphs (`✻` summary, `※` recap, all-faint runs, separators);
  `dialogTitle` is `couchmessage.ReadPicker`'s title. Couch judges; Pair
  still only renders (tail.go).
- **Behind a picker (pair#465):** a slot whose wrapper reports `Picker` waits
  `behind a picker` after `pickerNoticeAfter` (1m), not the general threshold.
  `pickerSummary` quotes `(kind) detail — title [options]`. The title and
  detail are bounded at 200 bytes each, and the options share one line bounded
  at 160 bytes. Its wait runs from `SlotLiveness.PickerSince`, the
  wrapper's `LastActivity` on the frame that opened it, because a picker that
  repaints (the auto-deny countdown) is never quiet. Episodes stay keyed by
  `LastActivity`, so the general path never repeats it, and a repaint re-reads
  the tail but finds the same wait (#468).
- **`couchmessage.ReadPicker` (`picker.go`):** the one reader of a picker's
  text, shared by this notice and the wrapper's held receipt. It anchors on
  the bottom-most selected option (`❯ N.`), so numbered prose never counts.
  Options are the block around it (past a rule when an option follows), and
  the title is the line above the first option, or the paragraph above it.
  Kind comes from the wording only: `Do you want to…`/`Would you like to…` is
  a permission prompt (with `Detail`, e.g. `Bash command: <command>` from
  between its dashed rules), an `Enter to select` footer is a question, and
  anything else is plain `picker`.
- **Episode:** one judged wait, the same reason and summary (#468). A new
  `LastActivity` re-reads the tail, but output that leaves the same wait on
  screen is the same episode: an idle agent still prints (Claude Code's update
  check redraws its footer every 30 min; an idle auto-compaction). A turn, the
  operator's input or a branch change ends it by changing the wait or ending
  it. One notice per episode once admitted, a failed admission retries next
  tick. Not-waiting verdicts are cached per (activity, settled, resting,
  pending, picker), so an idle slot costs one tail per output burst. The map is pruned
  to the listing each tick and dies with the service; a Couch restart
  re-announces a slot still waiting, once.
- **Delivery:** `Broker.Send` from the waiting slot's binding to the hub's
  exact slot (`messageService.hubBinding`, the connected binding in
  `messageAuthority.hub`; the registry's newest incarnation of that slot),
  so the hub's breaker, delivery and ledger apply
  unchanged. The body puts `slot repo:N waiting Mm <reason>` first (pair#211);
  #789 caps the chrome row at 40 cells and #802 shortened the built-in notices.
- **Not watched:** the hub itself (its operator is the one it would tell;
  the strict form of #441's self-addressed rule), and slots listed in
  `<store>/wait-notice-off` (one `repo:N` per line, `#` comments; the
  operator's file, Couch only reads it; if unreadable it mutes nothing and
  the notice says so). No hub connected, no notices.

### Session-sync files and the delivery ledger (#437, #439)

`couch --session-sync read|write|diff|status FILE` is the one mutation path
for a `type: session-sync` file (parley.nvim#313's turn-based operator/agent
file). Package `sessionsync` owns the rules; `couchcmd/session_sync.go` is the
CLI. It runs in-process against the store and needs no supervisor.
- **Agentic documents (#573):** a namespace registry admits plain markdown
  without changing frontmatter. Stable document identities preserve central
  shadow/state through metadata-only rename; register/lookup/list/forget are
  documented in the [Parley caller contract](couch-agentic-documents.md).
  Explicit couch-owner wins; plain owner metadata is not routing authority.
  Registration persists until forget, which durably exports current metadata
  before handing a typed session file back to its legacy route. Historical
  pathname intervals retain pre-rename owed submits and never expire by age.
  Operator locks older than 30 minutes still refuse writes, with age/ping
  advice. The registry is bounded and refuses growth without eviction.
- **Owner (#443):** the frontmatter's `couch-owner:` names the owning slot
  (parley.nvim#316); a bare `owner:` is a legacy alias read only when
  `couch-owner:` is absent. `ParseOwner` reads the frontmatter line by line,
  not as YAML, so a `process-instruction: [copy-edit] …` line is harmless.
- **Pessimistic for the human, optimistic for the agent (#439):** parley's
  `<file>.lock` (holder `operator`) refuses every write (exit 3); a legacy
  `holder: agent` lock counts as none, and Couch never writes or deletes the
  lock. The agent holds no lock: `read --section|--frontmatter|--all` prints a hash (sha256,
  16 hex) of the section body as stored (or the whole file), and `write
  --if-hash H` replaces it only if it still hashes to H, else refuses `stale`
  (exit 6, current hash in `Refusal.Hash` / JSON `Hash`). A write without
  `--if-hash` refuses. Edits in other sections never block. Every legacy-file verb holds
  `flock(LOCK_EX)` on `.<file>.flock` from read to last write, so two couch
  writers can't both pass the check. Registered-file operations and registry
  changes share the namespace's persistent registry flock; route selection
  occurs under it, including during a legacy handoff. Parley takes no flock (its lock fences
  it). `release` is a retired no-op (one release, then removed). Sections match
  whole `## HEADING` lines only (`section()` in `file.go`, shared by read and
  replace). Every file a write touches is published by temp+fsync+rename, so a
  reader outside the flock (parley reloading a clean buffer) never sees a torn
  file (pair#445): the file itself (`replaceShared`, through a symlink to its
  target, keeping its mode; parley and nvim follow the path's mtime), the
  shadow (`.<file>.shadow`) and the state sidecar.
- **Shadow (#448):** the file as the agent last saw it; `diff` shows the file
  against it. A section write updates only that section (and the restamp) in
  the shadow, so the operator's edits elsewhere stay in `diff` until the agent
  takes them in. `write --all` and `read --all` set it to the whole file. A
  shadow lacking the written section (the operator added it) stays unchanged.
  The first shadow written by `read --all` pins a sidecar-less file's horizon
  in the sidecar, since the horizon falls back to the shadow's mtime.
- **Frontmatter scope (#643):** `read --frontmatter` and `write --frontmatter`
  project only the closed leading `---` block (`FrontmatterBody` and
  `ReplaceFrontmatter` in `sessionsync/file.go`). The hash includes the delimiters
  and excludes the closing line ending. The original closing line ending and all
  body bytes, including Updated, survive a write exactly. BODYFILE is a complete
  block with optional trailing whitespace and no body payload. Both operations
  advance only the frontmatter in the shadow; a first baseline contains only the
  block, leaving the body unread. A previous plain-document baseline keeps its
  complete body when frontmatter is later added. Malformed current/shadow blocks
  refuse instead of acknowledging the body. Existing locks, owner/type validation,
  folds, registry limits and per-file atomic publication apply. The document,
  shadow and final state publish sequentially; after an IO error, reread the
  target, inspect diff/status and recompute before retrying any still-owed folds.
- **One edit path: `apply` and the round (#661):** `apply EDITS.json FILE --if-hash H` (H is the whole-file hash) takes the review pane's records `{old, occurrence, new, explain}`, as `[...]` or `{"records": [...]}`.
  - `ResolveEdits` (`sessionsync/edits.go`) resolves every record against the file, all or nothing: an empty, not-found or overlapping record refuses the batch as `edits-invalid`, listing each one (`Refusal.Invalid`, JSON `Invalid`), and nothing is written.
  - The review pane's Lua resolver (`reconstruct.resolve`) runs the same fixtures, in `testdata/edit-records`.
  - Apply resolves after the lock, hash and owed checks, then writes through the same prepared write as `write`.
  - **The round:** every agent write that changes the file (`write --all|--section|--frontmatter` and `apply`) replaces the round sidecar.
    - Where it lives: `.<file>.round`, or `state/<ID>/round.json` for a registered document. It is carried by register and forget, and kept by rename.
    - What it holds: `{base_hash, result_hash, at, presentation, spans}`, derived by `DeriveRound` (`round.go`) from a capped line diff (`linediff.go`) narrowed to the changed bytes.
    - Positions are 0-based [line, byte column] in the result.
    - The stamp is excluded: the diff baseline is restamped exactly when the write restamps.
    - `explain` attaches by base-range overlap (`attachReasons`).
    - A no-op write keeps the previous round.
    - **No stale round:** the old sidecar is removed before the document moves, so a failed round write (a warning) leaves none.
  - `round FILE --json` answers `{Code, Error, Round, Cleared}` (the parley.nvim#383 contract). `Cleared` is any submit of the file after `at` in the raw ledger, folded submits included, through `submitNames`, which `OwedForDocument` also uses.
  - **Propose:** under the document's `presentation: propose`, a non-frontmatter write keeps the base text and writes the body's diff as markers (`ProposeDocument`, `propose.go`), one per changed line. Unpaired blank lines are not marked.
    - **The engine:** `BuildEdit`/`contentStep` (`round.go`) is shared with `pair review apply`, so both paths behave identically.
    - **Applied directly** (`directUnderPropose`):
      - records wholly inside the frontmatter, which is never proposed;
      - a side-doc answer (`answersSideDoc`): a ready operator marker (`readyMarkerIn`, `marker.go`, the markers.lua grammar) replaced by a new `→ [proposal|discussion](PATH)` link;
      - the fold a link's `🤖[fold in]` asks for, which keeps the link with ` (folded DATE)` (S3).
    - **Refused:** a record crossing the frontmatter's end, or a body write changing the frontmatter (`propose-unencodable`).
    - **Operator-owned keys:** `presentation` and `collaboration-mode` refuse any agent change on every path, `write --frontmatter` included (`operator-owned`).
    - **Conformance:** `testdata/edit-records/conformance` holds ariadne#437's universal-review examples byte for byte. The file, shadow, hash and round all follow the written marker bytes.
    - A hunk over 3 lines refuses with `propose-too-large`.
    - A line the marker grammar can't carry refuses with `propose-unencodable`.
    - Both refusals name the one fallback, a side-doc proposal.
- **Ledger:** the message service's `messageEndpoint.Deliver`
  (`message_deliveries.go`) appends one row per delivery that may have
  reached the agent: submitted, indeterminate, an endpoint error, or a
  non-terminal answer. It skips explicit cancel and expiry, and the owner's
  own `session-sync ping: ` (pair#441): a ping whose sender binding equals the
  recipient, the breaker's identity rule (#433), so it cannot seed the next
  ping. A peer sending the same text is still recorded. The row goes to
  `<store>/session-sync/<repo>_<N>.jsonl` and holds the ID, the sender slot,
  the time and the body's first line (a submit's path line is kept whole).
  `couchmessage` is untouched.
- **Ledger bound:** a file past 256 KiB is rewritten keeping at most 512 rows
  younger than 14 days, within half the size bound.
- **Rename attribution (#573):** immutable broker admission time becomes the
  row's AddressedAt for pathname interval matching; older rows fall back to At.
  At remains delivery completion time for horizon, status and retention. A
  queued old-path submit completed after rename/reuse still belongs to the
  original document, rather than the replacement at its former pathname.
- **Missing data refuses:** `recording.json` marks a recording Couch, and
  `status` refuses `not-recording` without it.
  - An append failure writes a per-slot `<slot>.gap` record. A restart does not
    clear it. It expires 14 days after the last failure, and the next append
    deletes it.
  - Any verb on a file whose horizon reaches past a gap refuses `delivery-gap`
    until `--accept-gap`. The sidecar's `GapAccepted` records that per file.
  - So "nothing owed" is never a blind spot.
  - The slot is validated by couch's own grammar
    (`couchcore.ParseWorkspaceReference`) and query-escaped into the filename.
- **Owed:** rows for the file's owner slot (`couch-owner:`) from the sidecar's `Since`
  (`.<file>.sync`; without a sidecar, the shadow's mtime), minus folded IDs,
  minus submits naming another file (the owner's own pings never reach the
  ledger; see Ledger). A body `submitted: <abs path>` is a
  submit, owed only by the document it names. A plain delivery is owed only by
  the owner's session file (`type: session-sync`), never by its other agentic
  documents, so each delivery is owed and pinged once (#599). Only
  `write --fold ID` clears a delivery or a submit, never a plain write. The sidecar's
  `Folded` is pruned to IDs still in the ledger.
- **`status --json`:** the Couch envelope
  `{"Code","Error","Status":{File,Owner,Holder,LastWrite,Owed[{ID,Kind,From,At}]}}`,
  agreed with parley.nvim#314; `Holder` is `operator` or `""`. `read --json`
  answers `{"Code","Error","Read":{File,Section,Hash,Holder,Body}}`. `Owed` is never null, and a refusal exits
  non-zero with a `Code` other than `ok`.

### Stale-section reminders (#630)

The existing 15-second wait tick offers work to one serialized
`sessionStaleWatcher`; it never waits for an external command. For registered
`type: session-sync` documents owned by a verified hub, the worker runs
`tl-lead status check FILE` at most once per 30 minutes per registration while
the admitted wrapper is known settled, outside a picker and has no pending mail.
Unknown or busy owners and operator-held files are skipped without consuming
that allowance, so release can be noticed on the next tick. It checks current
registration, owner, lock and wrapper generation again before sending.

Only tl-lead owns section freshness budgets. A stale result becomes one ordinary
hub-to-self `session-sync ping: ` message; fresh results remain silent. Neither
inspection nor the check writes the document or advances its unread baseline.
Registered-only metadata inspection selects its route under the registry lock;
forgotten files never enter the legacy reader.
Checks have a five-second deadline and bounded output; their owned process group ends on
completion or cancellation. Missing/failing tl-lead or unknown output sends nothing and
logs one bounded diagnostic per continuous failure episode. Uncertain sends are
not retried within the allowance. Cooldowns are in memory, so restarting Couch
can produce one extra reminder. Service shutdown collects the worker.

### Stale-section CLI conformance (#630)

The stale-section reminder's real CLI contract is opt-in. Before landing changes
to the reminder worker, Couch's session-sync CLI, tl-lead status-check output or
the supported tl-lead binary, build Couch once and run:

```sh
make bin/couch
PAIR_LIVE_SESSION_STALE=1 \
  PAIR_LIVE_COUCH_BIN="$PWD/bin/couch" \
  PAIR_LIVE_TL_LEAD_BIN=/absolute/path/to/tl-lead \
  go test ./cmd/internal/couchcmd -run '^TestSessionStaleLiveConformance$' -count=1 -v
```

The fixture adopts a disposable isolated namespace, creates a Git-backed status
file and admits a disposable wrapper through the real message service. It runs
the real tl-lead and Couch commands and receives the reminder through the real
endpoint transport with a controlled composer. It checks half-hour repeats,
fresh silence, operator-lock release and unchanged document/shadow/state bytes.
It builds no binaries inside tests and never contacts an installed hub. Native
installed-hub acceptance remains the TL's check after a Couch restart.

### Recover plan after a restart (#367)

Fleet `staged` holds (ariadne#344, pair#539) preserve measured git facts. A
confirmed healthy staged host checkout is reported as `staged`, held without
a recovery action. It remains detached for reboot safety; ordinary detached
checkouts, dirt, operations and unavailable evidence keep their existing guards.

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

**Relaunching a live idle slot (pair#421).** `couch --relaunch repo:N
--confirm [--same-binary] [--force-unknown]` does from another agent's shell
what Alt+n does in the console. It parks the slot and cold-resumes it on the
current binary, keeping the conversation, and rides the slot-operation path
below. The verb is in the one list (`couchcore.IsSlotOperation`) that the
protocol, the socket and admission all read.

**Admission.** Relaunch acts on LIVE rows, which `ActorActions` never offers
anything for. So `PrepareSlotOperation` hands it to `prepareLiveRestart`:
`DecideLiveRestart` over the facts a `LiveRestartProbe` gathers on the console
queue (`couchcmd.liveRestartProbe`). The checks run in order, so the receipt
names the first thing to fix:

| Code | When |
|------|------|
| `not-live` | Nothing is running in the slot; use `--resume`. |
| `busy-unknown` | No wrapper session, or one that predates Settled. `--force-unknown` overrides, and the receipt notes it. |
| `busy` | The slot's wrapper reports it is not Settled. |
| `dirty` | `git status --porcelain` on the slot's checkout is non-empty or unreadable. |
| `stale-binary` | `pair` on Couch's PATH has the same content hash as the slot's running executable. |
| `unavailable` | This Couch cannot probe live slots. |

**Freshness.** The stale-binary refusal names the fix: `make build in
<checkout>`, where the checkout is bin/pair's parent and pair:0 in practice.
It applies under `PAIR_DEV` too: a Couch-launched Pair never runs
`dev_rebuild` (pair#489), so a relaunch sweep builds once, by hand, and each
relaunch is judged on the binary on disk. `--same-binary` overrides it. When the binary is newer but
behind its checkout's HEAD, or was built from a dirty tree, the note says so
without refusing.

**Reload-context (pair#421 M3).** `couch --reload-context repo:N --confirm
[--force-unknown]` does what Shift+Alt+N (`pair agent restart`) does: a fresh
agent conversation in the same Pair process. It does not pick up a new binary;
use relaunch for that.
- **Admission:** the same rule as relaunch, minus freshness.
- **Effect:** `Couch.ReloadContext` re-checks liveness under the thread's
  hold, then calls `LiveRestartProbe.RestartConversation`. That signals the
  wrapper the broker holds for the thread, never a pid file. It first
  re-checks `ProcOps.Identity(Binding.PID) == Binding.Start`, then sends
  SIGUSR2.
- **Confirmation:** a SIGUSR2 re-exec keeps the binding byte-identical, so
  success means a new session token (`SlotLiveness.Session`) within 20s.
  Otherwise the receipt is `failed` with code `unconfirmed`, saying to peek
  before retrying. A finish cannot be "unknown".
- **Admission note:** it reaches the receipt on success and on failure
  (`withAdmissionNote`).

**Prepare (pair#434).** `couch --prepare repo:N --confirm [--force-unknown]
[--json]` brings an idle slot up for new work. The TL runs it before
dispatching an issue. Four steps, each stopping the run when it does not succeed
(`couchcore.RunPrepare`, one `PrepareReport` row per step):
- **admit:** an admission-only slot operation `prepare` through the running
  Couch. It applies reload-context's rule plus the resting branch
  (`main-slotN`, `main` for `:0`; refusal `not-resting`), changes nothing, and
  hands back the checkout in the receipt's `Path`.
- **refresh:** `weave refresh` in that checkout, run in the CLI process,
  streaming to stderr. Weave fast-forwards the resting branch and the slot's
  substrates, then compiles. It never runs on the console's serial operation
  queue, which a refresh would hold for minutes.
- **recheck:** the checkout must still be clean. A refresh that rewrote a
  tracked file stops here as `refresh-dirty`, naming the paths. The known case
  is compile's `.gitattributes` union-merge block (ariadne#320) in a repo that
  has not committed it to main. Couch never commits it.
- **reload:** a confirmed `reload-context`, which re-admits at its own effect,
  so a turn that started during the refresh refuses it.

The report ends with the checkout's HEAD against `origin/main` and the commit of
each declared substrate (`layergraph.DeclaredSubstrates`), which is what the
slot's `sdlc` and `weave` build from. Re-running converges: the refresh is
idempotent, and nothing is journaled.

**Restarts return when ready (pair#427).** A `--relaunch` or
`--reload-context` receipt reports `succeeded` only once a NEW wrapper
session for the thread reports Settled. Settled means booted, its turns
(including a launch prompt's) finished, and idle, so a message sent right after it is pasted into a ready
composer.
- **Before the effect:** admission's prepare step records the session to
  replace (`liveRestartProbe.readinessAfter`).
- **After a successful operation:** `awaitReadiness` waits off the console
  queue, so other slots' operations keep running. `AwaitReady` polls
  liveness, bounded by 2 minutes.
- **On expiry:** the receipt is `failed` with code `unready`. The restart
  took; restarting again would interrupt it, so peek the slot. A wrapper
  that predates Settled reports (no `Build`) gives `unready` at once.
- **CLI wait:** the poll budget is 5 minutes, which covers the operation plus
  the readiness wait.
- **Scope:** the console's own Alt+n is not routed through slot operations
  and does not wait.

**Admission to effect (pair#421).** The console queue runs between
admission and effect, so both verbs re-check the busy guard at the effect.
Admission records its evidence in the implicit `require-settled` argument:
`known` (the wrapper said settled) or `forced` (`--force-unknown` admitted an
unknown).
- **At the effect** (`LiveRestartProbe.ConfirmNotBusy` before relaunch's
  park, and inside `RestartConversation` before reload's signal):
  - a wrapper that now says it is NOT settled is refused (`busy`);
  - unknown (session gone, or no claim) passes only when admission was
    `forced`, so a disconnect after a known-settled admission is refused
    (`busy-unknown`).
- **Scope:** the console's own Alt+n carries no `require-settled` and is
  ungated, because the operator is looking at the slot.
- **Rollout workflow** for agents: `couch --skill`, section "Rolling out a new
  Pair binary to idle slots".

**Arguments and the receipt.** The admitted call addresses the row by thread
(`repo-scope` and `tag`), the dialect the relaunch dispatch reads. The probe
finds the slot's wrapper the same way, by thread rather than by the
`repo:N` spelling (`messageService.LivenessForThread`). The admission note
reaches the receipt's `Warning` through `couchcmd.notedResult`.

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
   listed-terminal and launch identity rather than messaging registration
   (operator decision after the smoke test): the thread named by the request's
   scope and tag retains a working path and has a positively listed terminal
   (`authority.thread`), and the launch its record names
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

### The hub leads its section (pair#491)

Bare Couch’s hub is the slot it started from (`hubWorktree`, pair#463). Each hub is marked
by its leading position, reverse-video tab chip (pair#521), and underlined switcher row (pair#730).
- **Tab bar:** the hub is drawn first. A primary hub's own slots follow it in
  numeric order, then the remaining repositories alphabetically, each with its
  slots: `ops  :1  ariadne  :1  :2  blog  pair  :1  :2` (pair#732). A numbered
  hub keeps its standalone chip group.
  `[…]` still marks the active chip. Hub label, glyphs and brackets reverse
  their existing colours; red/attention/idle precedence stays the same. Gaps
  and folded `+N` counts retain their owner's inactive styling.
- **Switcher:** the hub is the first line. Every other row, and the message
  lines under it, sits one indent level (2 spaces) deeper. The hub main row
  is underlined; only the selected row uses reverse video. A selected hub keeps
  underline and adds bold beside the cursor arrow, preserving red
  for a blocked hub. Owner focus does not dim switcher rows (#840); parked
  age treatment and existing activity/status cues remain. The tab bar retains
  inactive-owner dimming. Attention detail lines keep their usual styling.
- **Data path:** couchcmd hands the root to `Console.SetHub` at startup. It is
  reduced as `MenuEventHub` into `MenuState.Hub`, which re-sorts the installed
  inventory (`reorderMenuInventory`). `PresentThreads(rows, git, hub)` puts the
  hub row first, flags it `Hub`, promotes its repository group and adds the
  indent (`hubSection`). That one order serves the switcher, the tab bar and
  clicks. Ownership sections retain command-line hub order.
- **One chip per slot (#607):** a slot's chip is its current inventory row, as
  in the switcher. A pane still open on a replaced address (a reboot's old
  thread, until its client exits) gets no chip.
- **Several hubs (#546):** explicit directories define ordered hubs. Their
  `.hub` claims and ephemeral CLI moves feed `HubOwnership`; unclaimed slots
  belong to the first hub. `Console.SetHubOwnership` supplies one section per
  hub, with a ` │ ` separator and focus-relative fading. Exact claims override
  repository claims; no assignment is stored on thread records.
- **Narrow widths:** `fitChips` (`reserve.go`, pure) lays out the actor cells.
  - It first collapses each repository's numbered slots into a `+N` count,
    groups giving way from the right.
  - If a *kept* chip (hub, active, red or belled) still lies past the edge,
    whole groups fold, from the right, into one trailing `+N` until it is in
    view. A kept chip whose group leader folded shows its full label.
  - Otherwise the row clips as before, so a cut-short plain tab stays clickable.
  - A `+N` cell has no chip span: clicking it does nothing.

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

### Hosted title ownership (#649)

Hosted wrappers skip title-child allocation through the shared Couch-hosted
predicate. The sole updater lives in `couchcmd/titles.go`, called from
`Console.advanceActivity` on the existing single-flight activity schedule.
Accepted inventory changes request a pass; the existing minute tick advances
idle heat. No separate timer, process, session listing or client query is added.

`couchcore/title_target.go` projects private display authority from the current
thread binding, successful session observation and newest typed launch. A
pre-effect local check revalidates session/start nonce/agent/ordinal; unknown
inventory cancels obsolete work. Target metadata is excluded from row JSON.
One `sessioninventory.TitleProjection.PollContext` per invocation supplies both
native activity and usage. The worker snapshots admitted live wrapper work once
and merges it with prompt/pane/native activity before title and fading decisions,
so both consumers use the same latest activity. Warm unchanged projections decode
no native history. Agy database growth invalidates admission and keeps the last
valid display until the next scheduled poll fully revalidates its database and
transcript (#656); true mutations still stop the source.

The optional pass has a two-second deadline and rotates its first target under
load. Commands use the existing joined process-group runner; failures leave
idempotent assignments unconfirmed for the next pass. The shared
`titlepoller.AdvanceTitleUpdate` records only confirmed successful titles and
retires replaced targets. cmux keeps scoped owner metadata, heat thresholds and
foreign-live-owner deferral using the inventory's existing host listing;
unresolved ownership defers. Standalone wrappers retain their pipe-leased title
poller and cadence. Older already-running wrappers keep their pollers until
restarted; rollout measurements name that coverage.

Tests: `couchcmd/titles_test.go` exercises the production native projection and
command seam, zero-work warm passes, bounded cancellation and an isolated
1,000-tick heap/cache regression with generation turnover. The console tests
exercise shared activity scheduling and cancellation on inventory replacement.

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

**Live work (#625).** The same background activity pass also reads the admitted
wrapper's existing `SlotLiveness.Settled` snapshot through
`messageService.WrapperActivity`; this projection adds no IO probe or timer.
Known-unsettled stays current even during a silent peer-driven turn/command
wait. The registry timestamps work reports and the busy-to-settled transition
with supervisor receipt time, monotonically within that session. Settled
refreshes do not advance `LastWorkAt`. The first settled report seeds it from
that authenticated wrapper's `LastActivity`, capped at receipt time; a Couch
reconnect does not itself invent recent work. Replacement/disconnect drops
the prior session's evidence. Scope, exact tag and agent must match. Legacy/unknown
wrappers contribute nothing, retaining the file-only rule. `Settled` is the
same broad restart-safety signal used by relaunch: an occupied composer,
picker or recent IO can also hold it false. The Console merges the newest
wrapper/file evidence for both views; positive wrapper work still counts when
the file probe fails, preserving any newer previous observation on that error.

**Background work (#632).** `SlotLiveness.BackgroundWork` is independent display
evidence on hello-v4. The wrapper's existing settle callback reads one complete
current terminal snapshot for both admission and display, using the existing
generation fence; source activity invalidates the display hint, and an older
in-flight probe cannot publish over a newer source. Display changes publish even
when `Settled` stays true. `backgroundWorkState` reuses the Claude composer bounds
and counts only a bounded native shell-counter field with management chrome below
the closing rule. Qualified captures include Claude 2.1.296 and the 2.1.301
PR-link variant (94×39): canonical positive-decimal `PR #N` is accepted only
as the first optional field, alongside a complete shell counter and management
chrome. PR-only absence remains unknown. The shared row projection preserves
erased blank cells while omitting wide-glyph continuation cells. Body prose,
drafts, incomplete synchronized paint, missing or
unqualified footer and unsupported agents grant no positive hint. Recognized
no-shell/zero state is false; loss of qualified evidence is unknown. This is a
native-UI observation, not a process/CPU inference or transcript scan.

The registry owns the nullable hint per admitted connection. Active background
work keeps the existing activity pass current; active→known-none records completion
using monotonic supervisor receipt time. Unknown clears the hint without inventing
completion, and idle refreshes do not renew it. Disconnect/displacement/replacement
discard it. Neither the field nor its clock changes `Settled`, message admission or
restart admission. Current native coverage is Claude Code; Codex/Qoder remain
unknown until their own native captures and geometry/negative/lifetime tests qualify
a source. The issue records the live Codex counter evidence and remaining
qualification work. No new timer, endpoint, polling request or persisted cache.

**Cost at rest (#507).** A pass probes every live thread (every 60 s, and on a
switch or a landed inventory). The probe's runtime
(`threadactivity.NewOSRuntime`) keeps one `sessioninventory.OwnerLaunchCache`
for Couch's lifetime, so a thread's multi-megabyte ledger is decoded again only
when its fingerprint changes (a launch or a binding). Likewise the continuation
watch, which asks every hosted thread every 500 ms, reuses a thread's status
while its record file is the one it read (`Couch.ContinuationRequests`,
`recordStamp`; records are replaced by rename). Measured on the operator's store,
22 live threads: a warm activity pass 0.34 s of CPU (was 8.6 s), a warm
continuation scan 27 µs (was ~100 ms).

**Unanswered Zellij reads (#562).** A successful CLI exit with empty or
whitespace-only stdout is no reply, not an empty inventory, detached client set,
or owner evidence. `launcher.ZellijSource` reports `ErrZellijNoReply`; callers
keep uncertainty and perform no automatic retry. The explicit exit1 diagnostic
`No active zellij sessions found.` remains positive empty-inventory evidence.
A new query is a new observation, and malformed nonempty pane JSON still fails.
Name-length assignments retain an unanswered probe separately from genuine
validator refusals, so they cannot learn a monotone length bound from uncertainty.
Cancellation is checked before parsing partial output. Nonzero name probes accept
only exact post-parse absence diagnostics: Zellij validates the --session length
before action dispatch; arbitrary failure diagnostics remain uncertain.
Native server identity and effect-time generation checks remain required; this
boundary cannot detect a nonempty discovery result that silently omitted a server.

**Unchanged evidence reuse (#559).** The long-lived collision checker owns a
`launcher.SessionNameIndexCache`, shared by `PairSessionName` and
`resolveScopedBindings`; message authority uses the same reader. It retains the
latest strict-decoded binding per owner for each legacy/scoped file generation,
so repeated observations do not decode or traverse historical index rows.
Live process, PID, readiness and effect-time ownership checks stay fresh.

`threadactivity.NewOSRuntime` also owns one `sessioninventory.ActivityCache`.
Its compact timestamp result is reusable only while the exact current ledger,
catalog presence/generation and scanner-observed native file/listing metadata
still match. A changed input runs the original `QuerySessionContext` and
`ActivityForSession`; resume identity alone never authorizes activity. Native
listings, when used by a proofless query, remain dependencies so new candidates
invalidate it. An unreadable disposable catalog disables caching and retains
the original proof/native fallback. The existing `OwnerLaunchCache` now shares
current and previous-established evidence with resume queries too. The previous
launch retains only resume identity/status; the current launch drops unused
baselines and binding history while preserving its scanner authorization proof.
Actual incomplete observations invalidate reusable evidence and fence older
loads; a canceled waiter cannot invalidate an independent loader.

Caches use file identity, mutation/ctime, size and timestamps; loads verify
stable metadata before publication, serialize only the same key, and retry
movement once. Each cache keeps at most 128 ready entries with LRU eviction;
activity/index slots include their in-flight entries in that bound. Cache
saturation runs an uncached safe query. They retain no descriptors or durable
cache files and die with their caller. Cold or genuinely changing evidence can
still require full scanner work; #508 owns the large durable rows and #157 the
full-inventory CLI query.

The release net now enters the actual consumers (#204):
`TestWorkbenchPerformanceIdleUITick` and
`TestWorkbenchPerformanceMessageUnchangedEvidence` use 20 scoped owners with
30,000,000-byte valid ledgers and independently count read/decoder-input bytes.
The UI test applies real scheduled results to the menu reducer; each unchanged
pass reads/decodes zero ledger/catalog/native bodies. Every message request must
reach a submitted receipt and its endpoint, with zero unchanged index/history
body work. Small fresh security reads remain active. Cold and owner-mutation
controls prevent disconnected counters from passing; #559's smaller component
tests retain invalidation/security checks. See [workbench performance](workbench-performance.md)
for normal-suite commands, strict selected-attach scan counts, other-path scan
ratchets and the separate 20-slot/60-second CPU acceptance smoke. Counts do not
claim whole-supervisor CPU behavior after rollout.

The shared chunked JSONL reader searches only appended bytes after a retained
newline-free prefix (#626). Counted-search regressions pin linear delimiter
work across long records and chunk boundaries; the 20-owner UI fixture keeps
its existing deadlines and logs per-probe and generation timing to distinguish
cold IO/decoding from missing progress. Cache authority and invalidation rules
are unchanged.

### Slots behind a picker draw red (pair#465)

A live thread whose wrapper reports `Picker` draws in the terminal's red
(`blockedSGR`, `\x1b[31m`; not the broadcast's LIVE red) in both views, and on
the active chip too. Precedence: placeholder > red > bell/attention > idle fading.
- **Data path:** after every registry step, `messageService.publishBlocked`
  derives the set of `ThreadAddress`es from liveness. When the set changes, it
  hands it to `Console.SetBlockedThreads` under a mutex shared with
  `SetBlockedView`, so a stale set never overtakes a newer one.
- **Console side:** the setter never blocks; it is a latest-wins channel of
  one. `Console.Run` reduces `MenuEventBlocked` into `MenuState.Blocked` and
  repaints. `statusModelLocked` fills `StatusActor.Blocked`, and the switcher
  reads `MenuState.Blocked`.
- **Timing:** red appears within the session's one-second frame pacing and
  clears the same way.

The persisted data is `threadactivity.Latest`, shared with the title poller's
heat ramp. It is the newest of the bound agent transcript's
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

The switcher’s `Ctrl+Y` copies the latest retained `Feed.Messages()` body outside
the console lock (`console_menu.go`, #604). The panel accepts canonical Kitty
Ctrl+Y as well as legacy 0x19 (#646); release events never trigger a copy.
`copy_key.go` classifies only decoded Ctrl+Y press/release events before product
routing and reports `clipboard_key` via the existing bounded supervision writer.
Its three content-free fields are encoding class, switcher focus and release;
all other input is ignored. `clipboard_copy` remains the later operation outcome.
No raw key stream, notice text or clipboard content enters either diagnostic.
The 2026-10-17 read-only probe proved the old CSI-u decoder dropped enhanced
Ctrl+Y; the historical keystroke bytes were not captured. Local consoles first use clipcmd's
existing `pbcopy` / `wl-copy` / `xclip` tool cascade, with a one-second command
budget and bounded pipe cleanup. Successful tool exit confirms local copying.
Explicit SSH environments skip the host clipboard. Missing, failed or timed-out
tools fall back to `Presenter.Copy` through the serialized parent writer; its
OSC 52 request has no delivery acknowledgment, so feedback says **unconfirmed**.
Status-row clipping and expiry do not remove the diagnostic. Copy feedback stays
in `MenuNotice`, so repeating the action does not copy its own confirmation.

A read-only investigation for #604 found Ghostty's installed default permits
`clipboard-write` and no local permission override; Zellij documents OSC 52 write
support. The current Couch parent chain was Couch → login shell → login →
Ghostty, with no outer multiplexer. The layer losing the original request was
not established without a live clipboard probe. Local clipboard copying bypasses these delivery
dependencies rather than assuming either terminal blocks OSC 52. See
[Ghostty's clipboard permission](https://ghostty.org/docs/config/reference#clipboard-write)
and [Zellij's clipboard support](https://zellij.dev/documentation/compatibility.html).
The approved #604 diagnostic scope ends at this read-only investigation; the
original loss layer remains unknown. The clipboard smoke follows the next
natural Couch restart, with further diagnosis only if it fails. Successful
local copying bypasses OSC 52; its unresolved delivery uncertainty primarily
concerns the SSH fallback.

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
additional storage or polling is added. Under #650 pair's per-turn slug refresh
is off by default when Couch hosts the thread (pair's wrapper gates the
`pair-slug` spawn on `launcher.CouchHostedEnv`, and the `slugcmd.Run` body
gates the external Claude Stop-hook entry the same way; `PAIR_SLUG=on`
re-enables both), so a hosted row carries a slug only while that knob is on —
the display path is unchanged.

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
`--archived`. `peek` (pair#362) projects as `--peek repo:N[:M…][,repo:N…]
[--lines N] [--json] [--transcripts]`, a read-only look at other slots. Its lines come first from
the slot's **live tail** (pair#425), else from the plain-text render of the
thread's terminal recording (`scrollbackcmd.RenderOwnedLines`, under the same
retention lease `pair scrollback render` takes); `source` says which, and a
failed live read is named in `unavailable`. With `--transcripts` it also returns
the Pair sent-prompt log and native transcript paths from the switcher's
`OSSwitchContextResolver`; by default it resolves none (pair#429), and the
recording fallback takes the agent from the thread record (`RecordAgent`).
Transcripts are paths, never parsed, and every unreadable source is named in
`unavailable`.

**Fast path (pair#429).** Building a Couch and resolving a `repo:N` to its thread
costs about 1.5s each. So when every reference is a slot and `--transcripts` is
absent, the CLI (`couchcmd.fastPeek`) builds no Couch and asks the running
Couch's broker for each slot's tail by name, concurrently: `tail` {`Target`
repo:N, lines}. The broker resolves the slot as a send's exact target resolves
(`couchmessage.ResolveTailSlot`: alias or unique prefix, against the connected
bindings' `Slot`), and answers with the tail plus `TailThread` {slot, tag,
agent, working path}, which it reads from the thread record as the typed peek does
(`RecordAgent`), so both paths give the same JSON. The fast path answers only when every slot does. Otherwise (no Couch, a
Couch older than the by-slot form, a reference that isn't a slot, or a slot
that can't be read) the typed `peek` operation runs the whole request and names
each failure.

The **live tail** is not a new store. Each claude/codex wrapper already keeps its
agent pane in a vt emulator (`wrapcmd.terminalModel`: screen + 10k-line
scrollback); `terminalModel.Tail(n)` renders its last rows on request. The path:
peek's `SlotTail` → identity-free broker op `tail` {thread scope, tag, lines}, or {slot, lines} on the fast path
(like `broadcast-status`: the store dir is the access control, the answer is
read-only) → `messageService.handleTail` picks the connected wrapper bound to that
thread → wrapper endpoint op `tail` → `peerDelivery.tailProbe`. Bounds: 200 lines
per request (more is refused, and the recording answers instead), 128 KiB per
answer (oldest lines dropped, counted in `truncated`).
**Markup** (generic terminal bookkeeping, never agent knowledge): `‹dim›…‹/dim›`
faint runs, `‹rev›…‹/rev›` reverse video, `‹cursor›` before a visible cursor's
cell, a literal `‹` doubled to `‹‹`, and a `cursor` summary (`ROW,COL SHAPE [steady]`, `hidden at ROW,COL`, or
`outside the tail`; rows count the returned lines). Pair renders and never
classifies: whether a slot is busy is the caller's judgement, and the couch skill
tells callers so. **Multi-slot:** `,` separates groups and `repo:1:2:3` expands to
three slots (`couchcore.ExpandPeekReferences`, at most 32); one reference keeps the single
`PeekResult`, several return `PeekSnapshot{slots}` read concurrently, and a slot
that does not resolve becomes a section whose `unavailable` says why.
Any operation that declares a `json` flag prints its result as JSON.

The hosted-agent hook `publish-description` projects only through hidden
`couch --internal publish-description <text>`, which pair's draft calls for a `!`
tag line (#337), a `!!` describe line (#358), and a bare `!` clear line (#357),
which publishes an empty summary. A recipient agent also runs it from its shell
after claiming dispatched work, publishing `repo#N <title>` so the sender sees
which slot holds the issue (#419; the `couch --skill` receiving step); the op
prints one confirmation line. `prepare-start`, `start`,
`attach`, `switch`, `park`, `resume`, `relaunch`, `prepare-switch-agent`,
`leave`, `stop`, `alias` and `reboot` are TUI/in-process
operations. `switch-agent` also has the confirmed hub-only remote form described
below. #363 removed `open-slot`, `fresh-slot`, `name`, `describe`, `archive`,
`recover-thread` and `recover-checkpoint` from the declarations: resume and
reboot replace them, and their couchcore internals (`OpenSlot`,
`StartFreshSlot`, `Couch.ArchiveThread`, `RecoverThread`, `ApplyThreadMetadata`)
stay as what resume and reboot compose. The stored name and description fields
stay on the record, written by nothing in the switcher, and are neither displayed
nor matched: `ThreadReferenceFields` is `{Address, Label, WorkingPath, Summary}`,
store resolution leaves `Label` and `Summary` empty (tag and path only), and the
switcher passes what a row shows.

### Agent keys and model selection (#544)

Typed process/input operations `keys` and `set-model` execute through the
existing live-owner console queue and effect gate. Their remote forms are:

```sh
couch --keys repo:N STEP... [--json]
couch --set-model repo:N --model MODEL [--effort LEVEL] [--scope session|default] --confirm [--json]
```

`couchcmd/agent_drive_cli.go` parses the bounded request;
`agent_drive_messages.go` sends admission once and polls `operation-status`;
`agent_drive_service.go` authenticates a verified owning hub, resolves a
different exact slot and rechecks ownership and caller/wrapper incarnation at
the effect gate. The wrapper must positively report settled state and a fresh
empty agent composer. A Neovim draft is independent. Busy/unknown state, an
existing picker, buffered paste or an occupied agent composer refuses. An old
wrapper without the capability asks the caller to relaunch on the new build.
The exception is a validated recovery-only key sequence: one or more `esc` or
`clear-line` steps, with no other steps or model request. A fresh typed
`Recoverable` projection admits a positively recognized occupied composer while
retaining the same caller, owner, incarnation and input-exclusion guards. It
may acquire a leftover awaiting-empty lease, never an active peer/drive lease.
Generic keys and model selection retain the settled empty-composer requirement.
A second exception (pair#691) is exactly one `esc` on a slash-command panel the
hub opened, such as Claude's `/usage`. The panel replaces the composer, so the
wrapper's `Dismissable` projection needs positive evidence of its own: the
current viewport (never scrollback) ends in the panel's Esc footer, with no
composer prompt and no numbered choice (`agentdrive/panel.go`). Pickers and
permission prompts number their choices, so Esc there stays refused. Every
other key, `clear-line` included, refuses while a panel shows. The dismissal is
confirmed only when an empty composer returns. A panel still showing at the
drive deadline reports unconfirmed with its tail, and nothing retries. A panel
frame repainted after the write proves nothing, because a usage panel repaints
when its data loads. One rule, `agentdrive.Request.Admits`, decides admission
for the service, its pre-input guard, the wrapper and the reducer. The panel
is the rows below the viewport's last full-width rule, since earlier prompt
rows stay visible above it. A panel taller than the screen has scrolled its rule
away and fills the whole viewport. The footer wording comes from the Claude
2.1.304 captures in `wrapcmd/testdata/drive/claude/2.1.304/` (short and tall
panels, plus real working-turn screens that must refuse). Claude reports
progress as working for as long as its panel is open, so the wrapper's turn flag
stays set. For the lone `esc` only, the panel proof outranks that flag:
`Dismissable` has no turn gate, and a dismissal drive does not stop on an active
turn (`DriveState.panelOwnsInput`). It still confirms only on an idle composer,
which follows Claude's progress-stopped report at dismissal. Every other drive
still stops on an active turn. Any other request refused
while a panel shows says so and names the lone `esc` (`agentdrive.PanelRefusal`).
For a non-recovery key, the service learns about the panel through one read-only
observation, made only on the not-settled refusal path (pair#716).

`agentdrive.Steps` is the one table of caller steps (pair#716). `Validate`
accepts a named key only if it is a row. `couch --keys --help` renders the rows,
the limit constants, the admission rules and the panel round trip, and
`couch --help` names them inline. A test holds the skill's step sentence to the
same table. Since pair#717 each row also carries its bytes, which `InputBytes`
reads. These are classic xterm encodings, the family the drive already sends to
Claude and Codex. The families `f1-f12`, `ctrl-LETTER` and `alt-LETTER` are
generated rows that share a group, printed once. `RefusedSteps` names chords
that are never a plain step, each with its reason: `ctrl-c` (use `couch
--interrupt`), `ctrl-d`, `ctrl-z`, the flow-control `ctrl-s` and `ctrl-q`, and
the aliases `ctrl-i`, `ctrl-m` and `ctrl-u`. The aliases are refused because a
second name for the same byte would split the admission rules.

`couch --interrupt repo:N --confirm` (pair#717) is the third drive request
shape, beside keys and model (`Request.Interrupt`). It is never a key step, so
no sequence can carry Ctrl+C by accident. Each shape defines its admitting proof
once (`Request.shapeProof`), and start admission (`Admits`), the pre-write
recheck (`WriteAdmits`) and the turn gate (`DriveState.TurnTolerant`) all derive
from it. The interrupt's proof is `Interruptible`: positive evidence of work,
never the turn flag alone, because a bare submission or an open Claude panel
also sets that flag. Claude uses the lifecycle reducer's `Working()` predicate
(open, not completed, activity seen, no stop grace), a recognized composer and
its own bottom-viewport interrupt hint. Codex uses the current coherent native
work status above its qualified composer: `RecognizeCodexWorking` recognizes
`Working ` and `Waiting for ` status prefixes plus `esc to interrupt`, with
either captured spinner bullet. Counters, truncated summaries and following
command-detail rows do not supply authority and are ignored. The same recognizer feeds lifecycle observation and interrupt admission;
Codex admission does not depend on cached notification flags. Interrupt observations
carry the last operator-input value of the mailbox activity sequence, so normal
work redraws do not revoke a queued interrupt. Observation, wrapper admission
and the writer use the same sequence selector; typing, Esc, image input and
submit advance it. Other drive shapes retain the all-activity sequence. Reader
output not yet parsed blocks interrupt proof until the current frame is known. Status position
is relative to the composer because modern Codex can leave most of the viewport
blank below it. MCP startup, quoted status text and partial redraws do not prove
work; partial redraws also do not publish a stop transition (#742). It also needs a complete frame, no picker, and the wrapper owning
input. It may take a residual
automatic-input wait, as recovery does, but never an active owner. `Admits`
checks the interrupt first and exclusively, so an idle agent never gets
Ctrl+C. The drive writes one internal `interrupt` key (`\x03`) and is done
only after a fresh complete idle composer proves the turn ended. Unknown or
picker screens cannot confirm it. It is unconfirmed at the deadline and never resent. A second
interrupt refuses because no turn is working. `couchmessage.IsDriveOp` names
the three drive ops wherever the protocol, service, CLI and console dispatch
them.

`couchmessage/drive_endpoint.go` provides `drive-observe`, `drive-start`,
`drive-status` and idempotent `drive-cancel` over the existing authenticated
endpoint. The RPC handler admits a bounded job; it never writes the PTY.
`wrapcmd/agent_drive.go` runs effects in the existing single stdin writer and
shares automatic-input ownership with peer delivery. Ordinary input cancels
the drive before entering that writer. Turn activity, owner/session changes,
shutdown and the absolute deadline stop input. Admission/status RPC closure
and ordinary CLI exit do not own the operation lifetime. Existing
`ttyio.Writer.WriteContext` bounds a blocked PTY write; context-aware
`couchmessage.Call` interrupts socket I/O and joins its cancellation callback.
No second writer, transport or durable model-setting field is introduced.

`agentdrive/{keys,recipe,screen,model}.go` owns pure validation, request intent,
versioned screen parsing and the reducer. `tailmarkup/markup.go` is the shared
styled-tail decoder, including literal escaped `‹‹`. Model navigation matches
unique observed labels, rather than assumed row numbers. Each screen-derived
write carries the atomic terminal Feed generation that supplied its evidence.
Immediately before input the writer requires a fresh complete proof: the same
generation, or newly parsed picker evidence identical in phase, selection,
labels, model, effort and footer. A benign repaint can change generation while
preserving that proof; changed recognized picker evidence refuses the cached
action. Unknown or incomplete screens wait without keys within the absolute
budget, allowing an owned `/model` transient to settle. Initial commands and raw
steps likewise wait for a complete frame within that deadline.
Success requires a new matching confirmation after apply, never old scrollback
or a pre-existing footer. Omitted effort follows the observed picker value.

Raw steps are the `agentdrive.Steps` rows: `text:…`, `enter`, `submit`, the
arrows, `tab`, `shift-tab`, `backspace`, `delete`, `home`, `end`, `pageup`,
`pagedown`, `esc`, `clear-line`, `wait:MS`, `f1-f12`, `ctrl-LETTER` and
`alt-LETTER`. Return translation stays shared with Pair: `enter` natively submits a positively
proved single-line slash command, inserts newline in ordinary composer text and
selects in a picker; `submit` means Alt+Return. Paths, leading user whitespace,
multiline input and incomplete text proof retain the existing newline mapping.
Known authored slash sequences wait for exact fresh echo within the existing
drive deadline. The final command proof and Return decision share one immutable
frame; changed ownership or command text vetoes the write. Unsupported raw edits
do not gain command prediction. Ordinary peer/prose sends and exact /model
cleanup authority remain separate. Draft-pane Enter is unchanged.
A lone ESC is flushed as one separate step. Raw success confirms acknowledged
input and the final styled observation, without claiming an arbitrary command's
semantic result. Printable UTF-8 text cannot hide terminal controls.
Codex Enter/submit waits at least the shared 150ms submission interval after an
acknowledged text write, avoiding its rapid-input paste suppression. This is a
pure effect deadline serviced by the same writer, within the absolute budget;
reserved model cleanup can preempt an un-emitted key.

Model scope defaults to `session`. Codex 0.162.1 has separate model, reasoning
and optional Advanced Reasoning phases. A unique explicit `›` marks the selected
row under a recognized complete heading/footer; reverse styling is compatible
but not required by current native captures. `s` applies only to this session,
while Enter on an effort applies defaults. Claude 2.1.299 combines model and
effort: `❯` selects a row, `✔` marks the current model, and left/right adjusts
effort. Its verified session action is `s`; default scope is refused before
input until a persistence recipe is verified. Session success reports the
observed model, effort and scope. Failed navigation backs out only through
owned recognized phases, with at most three separate ESC steps for Codex's
Advanced → reasoning → model → composer path. Recovery is claimed only after
a fresh empty composer; human cancellation never writes cleanup over new input.
Timeout cleanup may also clear this drive's acknowledged exact single-line
`/model` command with Ctrl+U, pinned to current composer evidence. Multiline or
changed text, uncertain write progress and possibly applied model selections
authorize no cleanup. Explicit recovery keys are a new operator-directed action,
not replay of the failed drive; inspect its receipt and slot first, then use
`couch --keys repo:N clear-line` to clear the current line.

Bounds: 64 raw steps, 8 KiB aggregate text, each explicit wait 0–1000 ms and
5 seconds total; 96 recipe navigation steps, 128 labels of at most 128 bytes,
200 observed lines and 32 KiB retained result tail. One drive can be active per
wrapper; competing automatic input refuses. The 30-second admission budget
includes queue wait. A model drive reserves min(3 seconds, one tenth of the budget remaining
at execution start) within that deadline for recognized recovery;
the CLI observes for at most 36 seconds. Couch retains at most 64 receipts for
five minutes; the wrapper retains at most eight completed drive receipts for
five minutes. Request IDs bind the complete keys/model/effort/scope intent.
Read-only status polling never replays a lost admission or uncertain input.
Inspect `couch --operation-status ID` and `couch --peek repo:N` before further
input when the outcome is unconfirmed. Shutdown removes wrapper jobs/receipts.

Recorded evidence lives in `wrapcmd/testdata/drive/{codex,claude}/<version>/`.
Run `make test-agent-drive-live` after changing the drive, picker parser or
input writer, and after upgrading Codex, before landing such changes. This
opt-in target enables both `TestAgentDriveLiveConformance` (session, disposable
default and unsupported-effort recovery) and
`TestAgentDriveVirtualTerminalLiveConformance` (actual Run/socket writer under
scratch and Pair virtual terminals). It requires installed Codex and an existing
local login; it drives no peer slot. Ordinary `make test` leaves native login
checks opt-in. Claude live fleet acceptance is owned by the hub.
Codex live conformance uses fresh disposable repositories, PTYs and copied
existing auth: session selection leaves scratch defaults unchanged; default
selection changes only that disposable config. Claude styled captures come
from hub-authorized live research, including session apply and ESC recovery.
No new Claude credential helper, Keychain reuse or shared-default experiment
is part of this change. Completed model-control chrome is recognized separately
from genuine questions and pickers so it does not open a conversational turn or
produce a false waiting notice.

**Resume and Fresh share prepared recovery (#556).**
`ResumeTargetWith` (`couchcore/resume_route.go`) gives a positively listed terminal
Continue priority and starts only an attach helper, including the background
warm-only pass. Lost pointers can adopt one proved existing terminal; they never
repair a checkout or start an agent. Existing pending/running #522 continuation
requests retain their exact executors. Actual cold native resume and the thin
`Reboot`/`ResumeCandidate` facades enter `executePreparedRecovery`
(`recovery_prepared.go`). Native resume carries no Prompt. Fresh allocates a new
tag and uses the normal #522 Prompt to point at previous-screen, branch and issue
evidence. The raw unreadable-primary archive escape stops and launches nothing. Recordless numbered-slot
Fresh uses this same recovery launch tail: its new record and creating start
claim commit a tagged Fresh intent together, then the producer rereads the
committed record and supplies the existing RecoveryAuthority. Pair validates the
exact helper, nonce, address, revision, agent and cwd before advancing that
address's reserved marker. It does not treat FreshRequired alone as authority to
establish an address (#747).

Preparation resolves the normal saved profile, preserves available screen and
durable selection, and obtains exact, expiring stop acceptance before retirement.
`--recovery-confirmation TOKEN` lets the owning hub accept the same offer as the
menu without repeating the target. A settled connected source uses
`PairLifecycle.ParkExpected`; a damaged source uses the immutable confirmed
terminal plan (`launcher/recovery_terminal_confirmed.go`). Retirement succeeds
only after every accepted process generation is independently proved gone.
`--force-unknown` is not retirement authority. Unproved/foreign owners remain
untouched; Fresh uses a distinct terminal identity and read-only checkout
readiness instead of reconciling setup beneath them. Normal workspace repair is
permitted only after absence is proved. Record replacement uses the existing
slot/global archive journal, preserving open lifecycle evidence in the archive.

Unconfirmed launch acknowledgment or registration retains the exact pending
claim, reobserves it once, and reports uncertainty. It never silently kills the
possible new agent or replaces it with Fresh. A StartClaim whose owner is dead
or which is older than five minutes offers explicit recovery rather than staying
Starting. Profile failure still leaves the existing agent untouched.

**The switcher's per-row actions are one pure table (#363 M2).**
`menuActionItems(row)` is `menuRowActions(menuRowFactsOf(row))`
(`couchtty/menu_actions.go`): `menuRowFacts` reduces a row to its kind (`:0`
primary or `:1+` slot) and phase (live, live with a failed / running / pending
continuation, resumable, unusable, unknown, busy) plus `ResumeOffered`,
`DirectoryMissing`, `AliasOffered` and `AddSlotOffered`, and `menuRowActions` is
the Spec table over them. Live rows get detach, relaunch, park and switch coding
agent (`:0` adds alias and add slot); rows that are not live get resume and
reboot, and a `:0` among them also gets add slot unless its directory is missing
(pair#402: a new slot needs the primary checkout, not `:0`'s agent). The
legacy action table supplies Relaunch for unknown/needs-relaunch rows and details
for genuinely in-flight work; the recovery frame adds explicit disk choices and
Fresh independently of scan/TTY availability. `menuRowAdviceOf`
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

Continuation has two internal operations (`pair#249`; #522 retired
`retry-continuation`, `dismiss-continuation` and `continuation-status`):

- `request-continuation`, invoked as `couch --internal request-continuation <absolute-path>`, durably accepts the hosted source's exact checkpoint. The inherited scope, tag, agent, session, launch ordinal, and expected digest bind publication to the writer's validated bytes and current source generation. This metadata operation can run in another worktree without becoming a second supervisor.
- `continue-thread`, declared as `couch --internal continue-thread` and run by the Console's watcher under the live owner, executes an accepted request, or continues one a previous Couch left running.

**Compaction without composer paste (#522).**
- **The launch prompt:** `executeContinuation` starts the new conversation fresh,
  with `ContinuationPrompt` as a one-time first message. It names Couch's private
  continuation copy, its NEXT ACTION, and `couch --peek <tag> --previous`.
  - It travels as `TrustedLaunchProfile.Prompt` (fresh profiles only), then
    `LaunchArgs.Prompt`, then `AgentCommand.Prompt` in `PAIR_AGENT_COMMAND`.
  - The wrapper adds it only to the exec argv (`launcher.PromptArgs`): a
    leading positional, `-i PROMPT` for agy. It goes first so that a saved
    flag taking several values can't swallow it.
  - No saved profile, ledger entry, config, `StartArgs.ExtraArgs` or the
    wrapper's in-pane fresh start ever carries it, so nothing replays it.
- **The coordinates:** after park, the compaction records #517's
  `PreviousSession` with its `Continuation` path.
- **Completion:** registration completes the request: `checkpoint.Advance`
  `Registered` moves Running to Complete. There is no delivery receipt to wait
  for.

**A request that doesn't finish leaves one line, never a blocking state.**
- **How it ends:** `failContinuation` calls `ThreadStore.MissContinuation`. That
  clears the request and writes `ThreadRecord.ContinuationMissed`
  (`checkpoint.Miss`: reason, continuation path, `TargetMayExist`) in one
  revision.
- **Where it shows:** the row reads `<state> · compaction didn't finish`. The
  focused row and one console notice give `Miss.Line`, which names one action:
  - compact again, if the old conversation runs;
  - resume, if it was parked;
  - relaunch, if the new conversation may be running.
- **What it blocks:** nothing. The next registered start or published request
  clears it.
- **A target that may still exist:** an incarnation recorded after the source's
  retirement. It is guarded by the thread's incarnations, not the continuation:
  `CommitStartClaim` refuses while one is recorded, and relaunch parks the exact
  one first.
- **Old records:** a request stored as `failed` before #522 decodes into the line
  (`fromPersistedThreadRecord`). The next write persists that.

**An in-flight request composes with its thread; it does not replace it**
(`pair#280`).
- **State text:** `<state> · continuation queued|continuing…`.
- **Actions:** an in-flight request owns a live row (no actions) until Couch
  finishes it. A row that is not live offers resume (which routes the request to
  `Continue` or `RecoverThread`) and reboot.
- **Refusals:** every refusal an in-flight request causes uses one wording,
  `checkpoint.InFlight`.
  - `continuationGuard` (relaunch, switch-agent, cold resume, start) and publish
    use it directly.
  - Each other check wraps its refusals once, in `withContinuationExits`:
    `RecoverThread`, `prepareAbsentContinuation`, `archiveContinuationVacant`
    and `validateContinuationWarm`.
  - `TestEveryRefusalARequestInFlightCausesSaysSo` drives one row per call site.

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
request. An empty record may be archived with its pending continuation
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

Acceptance is not completion, and registration is (#522). The fresh target's
launch prompt names the materialized copy, so the target that registers with the
request's exact attempt has its continuation. A failure leaves the thread's one
line (`ContinuationMissed`), not a request. The line blocks nothing; a target
that may still be running is guarded by the thread's incarnations and parked
exactly by relaunch. Whether the agent then acts on its first message is the
agent's: a prompt swallowed by a startup screen shows as an idle new
conversation, whose coordinates (`--peek --previous`) and continuation are
recorded.

The embedded snapshot remains authoritative if the original file is edited,
removed, or saved in a sibling worktree. It is retained through failure and
completion until superseded, and remains in the archived ThreadRecord. Archive
removes the derived materialized file. Active requests prevent unrelated cold
resume, agent switching, or relaunch from bypassing their ownership while the
compaction is in flight. Explicit archive may retain a pending request only after
proving its source/target unoccupied; it never marks an unfinished request
complete.
Hosted inner `pair restart` and address-changing rename routes refuse before
teardown; use Couch's tracked relaunch or name action. Standalone Pair retains
its outer restart-loop ownership and draft-seeding workflow.

`make test-couch-zellij-live` exercises continuation seed transport alongside
real park teardown. A deterministic pane under real Zellij reads the exact
materialized snapshot named by the launch profile's prompt and publishes its
readiness record. The production reader verifies session, agent, tag, attempt and
live PID; that registration completes the request, and deleting the session
invalidates its receipt. This fixture uses temporary stores and no paid agents.
It uses the stateful fake for source parking and the blocked launch helper. The
wrapper's exec of the prompt is pinned by launcher's
`TestCreateLayoutWrapperPreservesAgentCommand` and couchcmd's continuation
acceptance. A real agent acting on its first message remains an operator smoke
test. The conformance workflow runs on relevant changes
and weekly.

Live Zellij fixtures have two traps (pair#324):
- **Zellij's config dir.** On macOS, zellij resolves its config dir from HOME
  and ignores `XDG_CONFIG_HOME`. A fresh HOME, such as a CI runner, counts as a
  first run: zellij starts its default layout and silently ignores
  `--new-session-with-layout`. Any launch that passes that flag must pin
  `--config-dir`, as `LaunchSession` does.
- **Ownership evidence.** Session ownership needs live command evidence
  (#355), so a bare `ControlledZellij` session (just a shell) is never owned.
  - `ControlledZellij.AddOwnerWrapperPane` adds a pane whose running argv is
    `pair wrap --scrollback-log <raw>`.
  - The ownership probe parses each pane's argv. It refuses shell operators
    and `$`, and it reads JSON's quotes as shell quoting, which splits the
    argv wrongly. Stand-in agent panes therefore launch through a script that
    `exec`s the binary (`standInPaneScript`).
  - pair#432 tracks three cases excluded from `test-couch-zellij-live`.

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

Switch agent (`pair#184`, `pair#517`, `pair#728`) has a human form and a hub-only
remote API: `couch --switch-agent repo:N --agent NAME [--agent-args PARAMETERS]
[--interrupt-first] --confirm [--json]`. It selects one exact other slot; `self`,
family/range targets and the caller's own slot refuse. A connected source needs
current authenticated ownership, idle proof and clean Git. A retained parked
source is selected passively by `ResolveAgentSwitchTarget`, then admitted through
`PrepareAgentSwitch`'s existing classifier. Disconnect alone never proves vacancy.
Private `SwitchSourceEvidence` carries the admitted mode, revision, working path
and canonical repository identity; serialized previews cannot mint that evidence.
The service checks admission again at queued execution and immediately before park.
Retained-source admission shares one verified workspace observation across its
checks, with fresh alias/sibling/address selection, record, Git and hub checks.
Queued preparation and the effect guard resolve current workspace authority
again; the request-local observation never replaces those proofs.

Presence-aware `couchmessage.SwitchArguments` preserves absent versus explicit
empty argv. `PrepareKnownAgentSwitch` resolves Couch's canonical repository choice;
explicit argv uses `PrepareAgentSwitch`. Strict array decoding and the shared
literal/fresh-selector validators reject malformed input. Both early replay and
queued receipt admission bind presence, exact arguments and interrupt intent.
Accepted parameters publish only after final console adoption through the M1 owner.

`--interrupt-first` reuses the authenticated drive owner while `SwitchAgent` holds
its lifecycle lock: one total thirty-second phase, at most one interrupt, followed
by fresh idle proof and the ordinary park guard. The first ordinary or
cooperative live proof returns its verified wrapper connection token to the
operation owner; later guards require that same token, including local forms
whose initial source token is empty. Reconnection between phases refuses.
Unknown working proof sends nothing. Normal and cooperative paths retain
dirty-tree refusal. Force-confirmation cancellation copies the prepared preview
before dropping its offer, preserving retained render snapshots.

The form's **Force switch** requests a local proof offer, then one explicit risk
confirmation defaulting to Cancel. `AgentSwitchForceOffer`/`AgentSwitchForceConsent`
in `couchcore/switch_force.go` bind owner, exact accepted fingerprint and wrapper
token with one-shot consumption. `MenuEffect.TypedPayload` forwards it through the
existing console lifecycle queue and `OperationCall.TypedPayload`; operation args,
CLI and message requests contain no force permission. `couchcmd/switch_force.go`
waives only idle and cleanliness/status-display policy, rechecking record,
process, wrapper and canonical workspace/repository identity. `SlotGitStatus`'s
optional changed-file count comes from the same porcelain observation; malformed
or directory-summary counts remain unknown. Force preserves worktree files and
still uses the existing `ParkExpected` completion-plus-helper-death boundary.

The switch parks the exact outgoing incarnation and starts a fresh conversation
at the same thread address and working path. Its composer is empty: the switch
neither gathers native context nor delivers a prompt. The hub or operator
captures any needed context beforehand and decides what to send afterward.
Compaction and continuation retain their separate flows.

The outgoing Pair tty archive survives cleanup in `VerifiedPark`, using the exact
collision-safe artifact token. Retry recovery checks at most 32 prior completions,
newest first, and honors cancellation; unresolved history beyond that budget
fails explicitly before cleanup. Before target launch, `RecordPreviousSession`
retains only the verified park identity, source agent, capture token and capture
time. Registration may clear `VerifiedPark` without losing these tty coordinates.
A failed launch/retry keeps the original source; a different verified park
supersedes it, including one performed separately before the switch. Old records'
context fields remain readable and validated for compatibility, but are omitted
when rewritten (`threadrecord/previous_session.go`).

`couch --peek repo:N --previous [--lines N | --all]` reads that exact parked tty
recording and never substitutes the current incarnation. Missing capture evidence
is reported explicitly. `--previous --transcripts` refuses with guidance to
capture native coordinates before switching. Current `--transcripts` reads still
resolve the current native transcript and sent-prompt paths.

`--all` selects a complete retained terminal-history projection in the short-lived
CLI, not the Couch server. It conflicts with `--lines`; previous and full reads
accept one slot. Replay holds the existing capture retention lease and refuses
recordings above 64 MiB raw, 8 MiB events, 100,000 rendered rows, resize
dimensions outside 1–1000 or screen area above 262,144 cells. Retained history
is capped at 1,048,576 cells and 128 MiB logical bytes. Any capacity-driven
history loss refuses the read, even if a later terminal erase clears history;
checked emulator construction and resize report invalid geometry. Refusals
name the files. Full reads preserve ordinary terminal erase-history semantics;
a separately located native transcript can supply content erased from tty history.
Output names the recording files. Previous reads cost seconds and full replay is heavy.

Fresh registration requires a ready record matching the new launch nonce,
agent, session and process. The nonce travels in the trusted launch profile. Existing park/start/resume state owns failures:
a still-running source permits another switch attempt; a definitively failed
target leaves the old successful profile available for resume. A possibly
surviving target must be resolved through ordinary recovery before another cold
launch. Registration is the commit point for the new agent; later attachment
errors concern that committed target. The panel retains focus for a panel-origin
switch. There is no separate switch recovery record or delivery lifecycle.

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

`ctrl+return` toggles the existing **notification list (bell)** (`pair#811`), sharing
`couchkeys.ActionAttention`, `HitAttention` and `Console.toggleAttention` with
Ctrl+Alt+a and the bell. This applies from both actors and the switcher; the
underlying focus, active thread, switcher selection, paging acknowledgements and
previous-thread history do not change. A second press closes the overlay. An
empty list shows the existing “Nothing is waiting on you.” notice.

This replaces #221's direct newest-page jump and its switcher modified-Return
acceptance; `onNewestPageHotkey` and the separate newest-page action are removed.
Ordinary Enter still accepts a switcher row, while Enter or clicking an attention
item uses the existing notification arrival funnel. Ctrl+Space then Enter remains
the paging route and Ctrl+Backspace still returns to the previous thread.

The chord uses Kitty keyboard disambiguation (`couchkeys.CtrlReturnSequence`,
`\x1b[13;5u`); explicit press and repeat also toggle, while release does not.
The presenter owns one keyboard-protocol stack entry per screen it presents on
and restores both at release (`atlas/terminal.md`, #279). Child protocol
negotiation stays in the endpoint and determines child input encoding. Plain
Return still reaches the agent; on terminals without enhanced keys, click the
bell to open the notification list. Ctrl+Alt+a also requires enhanced encoding.
Paste content remains literal.
Physical-key tests cover child protocol changes and both main/alternate screens;
production-input regressions cover all three encodings from actor/switcher focus,
ordinary Enter, and both encodings of previous-thread navigation.

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

Inside the open switcher, a left press on a different row selects it without
switching; a press on the selected visible row activates it (#851). A double
click therefore selects then activates, with no timer. Chips inside the open
switcher follow the same rule; a hidden chip first reveals and selects its
row, while +N retains its group-opening action. Tab-bar chips outside the
switcher still activate immediately. Keyboard behavior is unchanged.

An activating click dispatches the SAME declared `switch` (or `resume`) operation Enter
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
`unsupported-agent`, `unknown`, `orphaned-server` (#399), and #214's named
binding failures `conversation-ambiguous`, `no-turn` and `unconfirmed`
(`IsBindingFailure` is the class). (`stale-incarnation` and `unrecorded-child` were
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

**One-gesture full quit** (#827). One confirmed switcher Alt+x owns the whole
park leave (`leave_park.go`):
- **Budget:** `LeaveParkBudget` (60s) is a scheduling budget shared by gate
  waits, queueing, drives and reconciliation. Synchronous durable writes still
  settle and are joined, so it is not a hard exit ceiling.
- **Drives:** each target gets at most `leaveParkMaxDrives` (3), under one thread
  hold. Each drive joins its exact admitted future (`parkFuture.AwaitJoined`,
  through the Leave-only `parkJoined`/`recoverJoined`); public Park/Recover keep
  the cancellable `Await`.
- **Retry policy:** `decideLeaveParkRetry` is pure. An open transaction retries
  through Recover under its own nonce. A fresh park needs `parkRolledBackError`
  (the controller's proof that nothing was published and the rollback committed)
  plus the same captured incarnation. Anything else is the operator's.
- **Report:** a failure returns `LeaveParkIncompleteError`, whose text is
  `FormatLeaveParkSummary`: every failed, not-started, skipped (unvouched
  incarnation) or unreadable thread once,
  in snapshot order, with its next action. The switcher's error notice and its
  copy key carry it whole, and Couch stays open.
- **Thresholds:** `parkCommitDeadline` is 10s (a measured 709ms worst commit at
  30 slots under load; `workshop/evidence/000827/`). The unpublished rollback
  gets the same budget. Relaunch's post-failure read keeps its own 1s
  `relaunchObservationTimeout`.
- **Store lock:** preview and routing reads take `store.lock` outside the
  retention coordinator. Coordinated foreground writes (`withUpdateLockContext`)
  therefore poll it within `storeReadLockWait` and then return
  `ErrThreadStoreBusy`. Before #827 they refused at the first EWOULDBLOCK, which
  is what left fan-out parks behind. Retention maintenance still yields at once.

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
diagnostic. The reattach pass displays it on the Continue row, a refused continuation is
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

**Registration waits on Pair, not a stopwatch (`pair#489`).** Every launch
(spawn, cold resume, fresh, warm) waits for registration under
`whileHelperAlive`: the wait ends at registration, when the helper's DIRECT
child exits (`Handle.Exited`; a dead launch fails at once: "Pair exited before
registering", after one 1 s final look so registration already written still
counts), or at `pairRegistrationTimeout`, a 2-minute hang ceiling. Startup scales with load
(at load ~200, launch-to-pane took 9-11 s and PAIR_DEV's no-op `make build`
9.6 s, so the old fixed 15 s failed five relaunches), so a slow start is waited
for, and adopted, instead of failed. Any failure takes the same post-ack cleanup
(the helper's group is ended, the thread rolls back to parked). The CLI prints
"still running" every 15 s while it polls the receipt, and asks again when a
status poll times out (a Couch busy launching can answer late); only the 5 min
poll budget gives up. Group liveness (`Alive`) is the wrong exit signal: a
Couch-launched Pair's title poller stays in its process group for up to 30 s.

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
- **Cold resume.** Its wait ends when the helper's direct child exits (about
  10 s in), not at a deadline (pair#489), then rolls back to `parked` as before.

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
- **Every other Continue thread attaches in the background** (`pair#206`,
  revised by `pair#528` M1). A start arms the reattach pass once its own thread has attached; a
  resume of one named thread does not. The pass is pure state in `MenuState`
  (`menu_reattach.go`), so the switcher's one transition authority orders it
  against every operator operation. Its specification is the transitions table
  in the #206 plan. The decisions:
  - **Seeded once**, from the first successful inventory: rows projected Continue from positive Zellij
    listing, excluding every requested hub, most recently active first.
    Unknown, exited and recovery rows are never seeded.
    The queue is never pruned or extended afterwards, because a failed refresh
    reads as "no sessions at all".
  - **Bounded parallelism**, as a `resume` with the implicit `warm-only` arg,
    using the existing lifecycle budget and operation queue.
    It re-proves the thread at its turn, and refuses one that stopped being
    warm, parked meanwhile or its session gone, before any effect. The pass
    never starts an agent. Every final failure marks the row `reattach failed:` with its code, or
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
can take (`primaryFreshStep`; explicit Fresh uses prepared recovery and any
required exact-stop acceptance). Known debris does not block --
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
helper without starting any workspace writer. The target starts at Couch's own
priority. #498's whole-group nice at launch was replaced by the slot renicer
(#502, below).

**Slot renicer (#502).** It keeps Couch, the pair wrapper, zellij and the agent at normal priority, and lowers only
what an agent spawns.
- **Each pass** (`couchcmd/renicer.go`, every `COUCH_RENICE_INTERVAL`, default 5 s; one goroutine on the message
  service's lifetime, so passes never overlap):
  1. reads the broker's connected bindings (`Broker.Connected`: the wrapper's pid and start identity);
  2. reads each slot's agent-pid file (`Paths.AgentPID()`, through the hardened pid-file reader);
  3. takes one `procutil.Table()` snapshot (one `sysctl kern.proc.all`, now with `Nice`);
  4. asks `couchcore.PlanRenice`.
- **The plan is pure.** A slot is confirmed only when the same snapshot shows the agent's parent is the wrapper
  and the wrapper's identity is the binding's. It never goes by a process name. Only confirmed slots lower
  anything: the agent's strict descendants below the level. The agent, the wrappers, and Couch and its ancestors
  are never lowered.
- **Applying it:** each process is identity-checked again (`procutil.Identity`), then
  `setpriority(PRIO_PROCESS, pid, COUCH_SLOT_NICE)`. A process that has exited (ESRCH) is skipped. A refusal
  (EPERM/EACCES) is counted and that pid is never asked again. `COUCH_SLOT_NICE=0` only reports (the agent's nice; nothing is counted as lowered).
- **Facts:** per binding (`couchmessage.NiceFacts`: confirmed, agent nice, lowered, at level, denied), read by
  `couch --actors` through `Broker.SetNiceView`. A confirmed agent found below Couch's own priority (its nice above
  Couch's, from the same snapshot) gets one notice: relaunch the slot to restore its priority (#541).
- **Lifetime:** facts, refusals and notices go with their binding, so nothing durable is created.
- **Cost:** 0.44 ms per pass over 1,425 processes (`BenchmarkReniceScan`), about 0.009% of a core at 5 s.

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
the Pair client the leader of one actor-owned process group. Its launch
auxiliaries inherit that group. The surviving native wrapper separately owns
its session-watcher and title-poller children: their own sessions survive client
detach, and wrapper shutdown stops and joins them (#715). Couch sends TERM and
then unconditional KILL to the client group, reaps the
client, and proves the group empty. On Darwin, a group signal probe returning
EPERM is ambiguous even after the client is reaped: Couch confirms membership
with `kern.proc.pgrp`. An empty successful snapshot proves absence; any member
(including zombies) or snapshot error retains ownership. Other platforms keep
the conservative EPERM interpretation. The remaining process class is the zellij
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
- a `.log` that starts with `couch-exit: <reason>` is a recorded exit
  (`pair#409`), written by `crashreport.RecordExit` before a deliberate exit:
  notice `previous couch exited: <reason>`, and the file is renamed `.crash` like a
  panic. Today the one writer is couchcmd. After `console.Run` returns, it
  records `terminal.ExitReason(console.TerminalFailure())`, which is set when a
  write to the outer terminal failed. Since #430 a write past
  `terminal.WriteTimeout` on its own budget stalls the presenter and does not
  end couch (see terminal.md). The stall wording below is left for a write
  bounded by a caller's deadline, which reads as
  `terminal stopped accepting output (a write waited up to 5s; wrote A of N bytes)`,
  and the notice names the file, since a panic during the exit lands there too.
  The reason comes from teardown's own classification (`Console.TerminalFailure`),
  so a shutdown that only cancelled a paint records nothing. Without it, this exit
  left nothing behind, because teardown's own `couch: terminal: …` line goes to
  the terminal that just stopped accepting output.

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

Couch selects an agent from the explicit request or the root actor's
`$PAIR_AGENT`; native resume keeps its retained thread profile. Startup arguments
come from explicit accepted argv (including empty), then that agent's record in
Couch's repository configuration. The repository key is the verified physical
Git common directory, so slots share per-agent arguments while independent
repositories remain isolated. `launcher.AgentInventory` supplies supported agents.

`repopreferences.go` owns pure resolution, folding and marked borrowing;
`repopreferences_codec.go` preserves unknown fields with strict known types.
The global ThreadStore document `threadstore/repo-preferences.json` is bounded to
4 MiB and 256 repositories, with 4096-byte formatted arguments per agent.
`repopreferences_store.go` serializes accepted updates under the namespace lock,
regardless of which backend holds the conversation. Missing own-repo settings may
yield a marked, deterministic same-agent suggestion from another confirmed
repository; CLI never implicitly accepts it. The active switch form renders the
own/borrowed/absent/conflicting state and donor repository before acceptance;
cancellation dispatches nothing. Explicit acceptance saves it locally after adoption.

`repopreferences_migration.go` reads bounded legacy Couch path-preferences and
enrolled slot `.couch/preferences.json` files, independently verifies Git identity
and containment, and folds them in memory. Equal legacy values agree; differing
values expose a conflict requiring explicit arguments. Confirmed values outrank
late legacy sources. Source identities, exact bytes and digests survive in the
canonical document; original files stay untouched and receive no new writes.
Committed sources are not read again. Preview never creates locks/directories,
replays journals, enrolls retention or reads Pair's per-folder defaults.

The resolved selector-free argv travels explicitly in the existing strict,
tag-bound `PAIR_COUCH_LAUNCH_PROFILE`. Its existing repo-default provenance marker
is retained for protocol compatibility, not a fallback into Pair's preference
store. Pair's standalone defaults and per-folder behavior remain independent.
`preference_boundary_test.go` rejects a Couch dependency on Pair preference readers.

The start claim records explicit `remember_configuration` intent independently
of fresh/resume shape. Registration promotes exact execution history and journals
thread + manifest; it writes no preferences. Only successful final console attach
calls `PublishAdoptedConfiguration`. Its shared ephemeral completion publishes
once and reaches StartResult, SwitchAgentResult and warning/receipt wrappers. A
publication failure keeps the adopted child and visibly says startup parameters
were not remembered. Failed attachment/registry adoption, automatic resume,
warm attachment and recovery never learn merely because they ran. There is no
durable retry/outbox after the disclosed adoption/publication crash gap.

Migration and writer admission share one five-second context. Coordinator waits
honor that context; busy nested-store attempts release the coordinator before
waiting, preserving concurrent merges within the remaining budget. Publication
runs on the existing lifecycle completion worker before operation/remote
completion, while Console.Run continues input and rendering. Initial attachment
runs before Run and queues any warning for first paint.

Same-agent **Restart with new parameters** pins the preview's exact native ID
through a private accepted intent, independently of edited selector-free argv.
`editedrestart.go` decides at most one cold resume and one disclosed fresh
fallback; `editedrestart_evidence.go` preserves exact missing versus changed or
unknown evidence. `editedrestart_launch.go` consumes each request before the
existing start-claim CAS and `launchTrackedThread`. A failed resume permits fresh
only after authoritative absence, the exact existing cleanup owner's rollback
receipt and fresh vacancy/admission. Replacement or prelaunch authorization
refusal stays terminal even if a later read reports absence. The receipt is
request-local; interrupted operations retain ordinary start reconciliation,
which never reconstructs permission for fallback. For an ID-only filename or
requested-resume binding, editedrestart_presence.go asks the same scoped artifact
owner for one exact-ID metadata observation (editedrestart_metadata.go). A unique
root becomes a located binding; complete absence is Missing, while partial/error
listings and duplicates refuse. Located bindings retain one stat and no listing;
ordinary resume policy and binder cost are unchanged. Fresh success visibly reports
`no previous conversation found; started fresh with the new parameters`.

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
every existing `path-preferences/` file stayed readable after that deletion.
The later #728 repository-argument migration retains those files as bounded
read-only inputs; they no longer own current defaults. The old `policy` object
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
**Wall-clock budgets (pair#436, pair#481, pair#514).** A default-lane test asserts its outcome, not a time. "Doesn't block" or
"doesn't wait" is a hang guard around the work while its dependency is held, and a cost that can be counted is
counted (an operation count, as the broadcast marks' stroke walk does). A real budget stays only where speed is the
test's subject. Go checks use `perfbudget.Check` (`cmd/internal/perfbudget`); the doctor shell check uses the same
`PAIR_PERF_BUDGETS=1` opt-in. Both log elapsed time on every run and enforce the ceiling only in that lane.
`make test-perf-budgets` runs each budget in isolation on a quiet host; its recipe is the canonical list.
`make test-perf-capture` always checks report shape and uses controlled clock/tool fixtures to verify the
capture's work shedding and probe priority. The production six-second budget remains cooperative: it sheds
later work, but does not preempt an already-running external collector. Most
couchcore tests are `t.Parallel()`; ones that set process env, chdir, mutate a
package global, signal real processes or measure wall time stay serial. Its `TestMain` defaults
`-test.parallel` to 3x GOMAXPROCS (the tests mostly wait on subprocesses); an explicit flag wins.
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

**Named binding failures** (`pair#214`). The evidence pass keeps the refusal
code (`ThreadEvidence.ParkedRefusal`), and the row names it whether or not a
receipt exists:
- ambiguous reads `conversation-ambiguous` (reboot);
- unbound reads `no-turn` (reboot);
- provisional, on an incomplete storage listing, reads `unconfirmed` (retry).

A resolver IO error leaves the proof unresolved (`unknown`), never a binding
verdict.

**An unturned fresh launch does not hide the conversation it replaced.** When
the newest launch is a chosen-id fresh start whose file a complete listing
proves absent (its agent never took a turn, e.g. an in-pane Shift+Alt+N
restart), `QueryResumeTargetContext` falls back to the nearest earlier
generation if it bound one root (`sessionledger.PreviousEstablished`).
- couch's proof and pair's resume read this query, so both continue that
  conversation.
- Pair's restart decisions read the per-launch target and are unchanged.

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
| reboot / Start fresh | explicit Fresh selection, version/config/checkout validation and any exact-stop acceptance | `Couch.Reboot` → shared prepared recovery | Fresh remains available independently of native scan/TTY evidence; unknown owners are left untouched |
| resume | `ResumableState` | `SelectResumableRoot` | `detached`, `parked` |

The switcher's **offer** is written separately, in the action table
(`menuRowActions`), and `couchtty`'s `TestActionOfferedImpliesPermitted`
compares the two over the table's derived domain. Resume on an unusable row is
admitted by `ResumeTarget`'s route rather than `ResumableState`, so the test
pairs it with that statement (resumable, or unusable). It is deliberately
not filtered through the predicate: a filter makes offered-implies-permitted true
by construction, and a guard that cannot fail is not a guard.

The guard's reach (relaunch, switch-agent, a cold start claim; `pair#280`) stays
falsifiable because the list's oracle is the guard's BEHAVIOUR, not another
predicate: `TestTheContinuationGuardsReachMatchesEveryRowAction` drives every
declared row action through the production dispatcher on a live thread holding a
request in flight (a failed one no longer exists, #522). A listed operation must be refused by the guard itself, and an
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

Recovery candidates are an explicit choice: `couch --resume-candidates repo:N --json` lists up to five native/proven choices. Capture-backed alternatives retain their measured tail matching; when the screen cannot support inference, native-only alternatives show **unconfirmed**, physical cwd, semantic native activity and native ID without a tail score. Screen failure reasons and readable capture-file activity remain separate from native-scan completeness. A missing launch association weakens provenance without making an otherwise complete scan incomplete.

`couch --resume repo:N --candidate KEY` selects the exact observed evidence. Native-only keys bind the fixed observation cutoff, source revision and ordered artifact fingerprints and expire after five minutes; refresh after expiry or changed evidence. Selection freezes the original native snapshot and verifies the exact selected artifact before preservation, retirement and claim admission. It persists bounded native evidence before source retirement; the launcher independently revalidates the artifact and physical cwd before composing native resume argv. No capture or established binding is manufactured. Plain resume continues to require an established binding; Fresh remains available with honest context-loss advice. Listing does not provision or write lifecycle state, and automatic inference stays disabled. The shared scan envelope remains 30 seconds, 10,000 metadata visits, 64 eligible roots and 256 MiB; failed screen reads and retry scans share that read budget. Screen replay separately preflights its shared 64 MiB raw and 8 MiB event limits before reserving replay bytes; rejected screens leave unread bytes available to native discovery. Readable owned raw-file activity remains descriptive even if companion events or archive authority fails.

Explicit multi-directory launches enqueue the typed `start-hub` operation on
the existing console operation queue. It reuses StartInteractive for each extra
hub while preserving the first hub’s focus and the one supervisor lifetime.

The typed `move-slot` operation commits an authenticated hub ownership move on
the same console queue; CLI receipts include the destination and persist mode.
The queue has concurrent workers. `dispatchHubOperation` serializes ownership moves
and process mutations at their effects; extra-hub admission is chained after each
prior adoption to preserve command-line order.

Multi-hub ownership (#546) lives in `couchcore.HubOwnership`: ordered active
hub definitions, bounded parsed `.hub` claims, and revisioned ephemeral moves.
Exact claims beat repository claims; equal specificity picks the first CLI hub
and warns; everything else falls back to main. Thread records carry no hub field.
`couchcmd.hubRuntime` publishes that one immutable snapshot to routing, mutation
guards, supervision and `Console.SetHubOwnership`. A move runs on the existing
operation queue with caller and revision rechecks inside that effect gate. Persistence uses bounded
no-follow reads and expected-content atomic replacements; incomplete writes retain
runtime ownership and give inspect/retry guidance. Claims are reread only for an
explicit persistent move. No file watcher or per-hub supervisor is added.

Bare startup bypasses this configuration path. Explicit extra hubs reuse
StartInteractive and the existing background adoption path. Per-row hub sections
and focus-relative fading use memory-only projection; the cast remains a singleton.
Switcher move controls are deferred to #548. Confirmed delivery facts cover every
active hub; recipient hub classification is retained before external delivery so
a disconnect during completion does not erase the evidence.

Recovery uses #522’s one-time launch Prompt: a fresh start receives only evidence
coordinates (previous-screen peek, branch, issue and exact retained capture paths),
never screen contents. Native resume receives no prompt. Launch notices report
startup, without delivery callbacks or pasted-input status.

Warm Continue attachments share the optional context-scoped launch timing
observer with native launches, but report their actual attach preparation,
admission, spawn, acknowledgement, live check, registry/repair and cleanup
steps. They report retry backoff when it occurs; they do not report cold claims,
native registration or promotion. Explicit fleet detach shares one lazy owner
observation scope. Only locally owned warm clients need the borrowed-helper
preview; managed detach retains its normal generation and session checks.

## Confirmed disk recovery (#556 M1)

`couch --resume-candidates repo:N --json` includes a proven native binding when
available, plus explicit ranked candidates with their evidence. There are at most
five disk choices total: a proven default plus four candidates, or the best
candidate plus four alternatives. Automatic inference stays off. Plain
`couch --resume repo:N` chooses proven resume; `--candidate KEY` chooses that exact
ranked evidence snapshot. `couch --reboot repo:N --confirm` requests Fresh, including
when transcript scans or previous screen evidence are incomplete or absent.

Before retirement the executor resolves the saved configuration, checks workspace
identity, preserves available screen evidence under its existing retention owner,
and durably records the selected intent. Native launch rechecks that selection at
the claim and Pair execution gates. Proven resume needs no screen; inferred resume
still requires its strict capture/candidate provenance. Fresh uses normal saved
path/default configuration when no conversation profile remains; agents cannot
supply recovery argv or edit defaults.

A request that would stop attributed running processes returns an exact stop set
and a short-lived confirmation command, `couch --recovery-confirmation TOKEN`.
That standalone command works for the owning authenticated hub without the menu.
The token is in memory, single-use, valid for two minutes, and bound to the
confirmer, source revision, selection, native identity and terminal attribution.
Changed ownership or generations require a new selection/confirmation. Restarting
Couch forgets unused confirmations. Menu and CLI dispatch the same executor.

The registered-wrapper fallback preserves pane ownership evidence (#797): it runs
any original terminal retirement first, quiesces the named session while the
additional wrapper tree is still live, then signals and joins that tree. Normal
park keeps its existing ordering. All scoped deletion callers pass the positively
observed exact server to `OSRuntime.DeleteOwnedSessionContext`; its quiescence
loop can confirm absence but cannot authorize a later server generation. A
name-only deleter or unknown/foreign ownership refuses. Failure diagnostics retain
the bounded native pane JSON before disposable cleanup removes its store.

Selected recovery intent lives in schema-4 thread/start records; schema-3 records
are migrated in memory without startup writes. Future schema versions refuse
mutation. Fresh archives the source through the existing store journal and always
allocates a new tag, preserving metadata and native files while avoiding namespace
reuse. Its ordinary #522 first-message Prompt points at the previous-screen peek,
exact retained paths when available, branch and issue. Native resume has no Prompt.
The previous-screen reader validates retained capture ownership and file identities;
orientation is display evidence, never native launch authority.

After an acknowledged or uncertain launch, loss of registration does not authorize
cleanup or another agent. One bounded exact-attempt reobservation can prove
success; otherwise the retained claim is reported as unconfirmed with inspection
coordinates. Pre-exec definite failures retain ordinary rollback behavior.
Implementation owners: `couchcore/recovery_prepared.go`,
`recovery_confirmation.go`, `recovery_start_fence.go`, `launch_existing.go`,
`launcher/recovery_terminal.go`, and `threadrecord/recovery.go`.

The queue's `couch --internal recovery-confirmation --token TOKEN` operation
shares the same token owner as the standalone public CLI; it is transport plumbing,
not a second confirmation mechanism. The token argument is never stop authority
without the bound source/actor/generation rechecks.

`Reboot` is a facade over prepared Fresh, including ordinary saved configuration
for an explicitly chosen agent. Its earlier independent park/stop/launch pipeline
is removed. The existing corrupt-primary/no-usable-path archive escape remains:
exact bytes move through the archive journal with expected-byte CAS, no process
is stopped, and the result says no fresh agent started. Structurally valid future
record versions refuse mutation. A numbered slot with no record can still start
Fresh beside unknown remnants using a distinct tag and read-only checkout
validation; its notice says workspace setup was not refreshed.


## Native conversation admission (#595)

The wrapper holds a nonblocking exclusive flock for each exact agent/conversation
ID it knows, from discovery until wrapper exit. The stable cache inode is
`~/.cache/pair/native-locks/<agent>-<sha256-of-exact-ID>.lock`, constructed by
artifactpath. Its bounded holder label is diagnostic only; the kernel decides
ownership. A contender refuses with the holder and Start fresh advice. Files are
never replaced or unlinked online; one permanent inode and ordinarily one
filesystem allocation unit remain per distinct conversation (APFS measurement:
6 metadata bytes, 4 KiB allocated). Reclamation requires operator-approved
quiescence of every cooperating wrapper.

The launcher still resolves supported latest selectors once and pins an exact
ID. Claude/Qoder latest ordering and configuration constraints are unchanged.
Codex indirect selectors preserve their current CLI behavior and display the
existing not-guarded notice until #571 supplies faithful resolution. Exact resume
IDs and known new-session destinations lock before native spawn; a fork locks its
known destination, not the source. The shared selector parser owns option/value,
delimiter and positional grammar. Saved startup defaults are not rewritten.

Unknown Fresh IDs remain unknown until the existing session watcher commits a
current-launch binding and sends a bounded private-pipe notification. The wrapper
validates the launch ordinal and locks that ID once; repeat reports are harmless.
A known-ID conflict or lock IO failure stops the wrapper's own group. Discovery
failure before delivery remains visibly unknown and does not interrupt the
watcher's config or lifecycle work. No native fd adapter, bootstrap or historical
ownership scan is involved.

The native child already starts in its own PTY session/group. Every controlled
wrapper exit joins TERM/grace/KILL cleanup of that directly created group before
reaping its leader and releasing conversation locks; fresh wrapper exec follows
cleanup. The guarantee ends with wrapper lifetime. SIGKILL can leave an orphan
engine after the kernel drops the wrapper lock: #583's reaper and #587's
unattributed report handle that accepted gap. Agents outside Pair are outside the
guard. Exact source stop-set attribution remains separate from admission.

Rollout needs no cutover gate (TL e93b109d): a resume whose lock another live
wrapper holds refuses at native start with "conversation X is in use by
<holder>; Start fresh", where the holder is the slot address ("pair:3"), else
its Pair tag, else "standalone". Wrappers started before this build hold no
lock until their slot is relaunched; that transitional gap closes as slots are
relaunched. There is no capability barrier, override or dual guard.

The forensic native-owner scan and reservation store are deleted in this build.
Their obsolete on-disk files are unused, not removed by startup; any cleanup is a
separate operator-approved quiescent operation. Shared strict JSON and ledger
validation remain for their other consumers.

Owners: `nativelock/`, `nativeid/` (the one conversation-ID rule), `artifactpath/native_lock.go`,
`wrapcmd/native_lock.go`, `wrapcmd/owned_group.go`,
`sessionwatch/binding_notice.go`,
`launcher/native_resume_selector*.go`, and Couch recovery/Relaunch. Real kernel
contention, process-group cleanup and private-pipe checks run in ordinary Go
package tests under the recorded suite on Darwin/Linux.

## Resource monitoring (#587)

`couch --resources [--json]` reads the running supervisor's cached resource view
through the broker socket. Queries perform no process scan, terminal query or
Git lookup. One joined supervisor worker samples at one-minute cadence; a
separate inventory worker authenticates roots at startup and on changed
thread/helper/lifecycle evidence. Routine unchanged menu refreshes do not
repeat ownership probes.

Ownership starts with an authenticated terminal server `(PID, start identity)`
from `ScopedThreadArtifactCollisionChecker.NamedPairSessionContext`, published
in `ResourceRootCache` before being consumed by the sampler. Probe revisions
are assigned before I/O, so a late old result cannot replace a newer proof.
Cached console rows include draft-only, detached, parked and never-started
slots; hosted helper/client roots and admitted wrapper bindings supplement
server descendants. Missing workspace or terminal evidence remains explicit
unknown coverage. A slot's branch or quiet output alone never proves idle:
idle requires the matching wrapper's settled state and no pending broker work
or picker.

`couchcore/resource*.go` owns pure generation-fenced attribution, deltas,
episodes and bounded history; `couchcmd/resources_inventory.go` owns root
refresh; `couchcmd/resources.go` owns sampling and storage; `resources_alert.go`
owns advisory broker delivery. `procutil/resource*.go` provides the
native reader. Darwin with cgo supports cumulative process CPU, current RSS and
physical disk-write bytes. Other builds report unsupported counters. PID reuse,
changed parentage, zombies, partial reads and counter regression cannot become
valid rate samples. The first observation establishes a baseline; CPU 100%
means one core. Observed cumulative totals omit unsampled lifetimes.

Remembered process generations preserve original slot ownership after
reparenting; confirmed root loss or thread replacement can expose proven
orphans. A process first seen after orphaning has no invented slot owner.
A detached command keeps its slot through its inherited environment (#606):
`DetachedSlotRoots` makes a launchd child whose `COUCH_THREAD_SCOPE`/`TAG`
names a live slot's thread a root of that slot, so a backgrounded or
abandoned tool call (a recorded `sdlc local-check` suite, its test binaries, a
leaked helper) counts toward the slot instead of raising orphan alerts. macOS
withholds a platform binary's environment (`/bin/zsh`), so a silent launchd
child's direct children answer instead. Zellij servers, agents and embedded
nvim are never claimed this way; a relaunched thread's leftovers stay subject
to the orphan and unattributed rules. Each process generation's environment is
read once — two native calls per read, at most one read per table row on the first
pass and afterwards only for generations the memo has not seen (#644) — and only
the two thread keys are kept; the reads show as `environment_calls` in the sampler
cost.
Known process kinds parented by PID 1 above the RSS floor are shown as
**unattributed (unproven)**, never acted on. Only candidate classification may
inspect argv; full command lines are not retained or displayed. Sampling can
miss a process born and exited, or born and orphaned, between observations.
The monitor itself sends messages and does not kill, restart or renice
anything; the one destructive verb is the operator's, and it is confirmed.

### Unattributed embedded nvim (#583)

An untagged `nvim --embed` at PID 1 — what nvim's own TUI/embed fork leaves
behind when its owner dies — is a leak the tagged reap cannot see: no sidecar
names it and no argv carries a tag. The sweep classifies it from evidence the
sample already holds (PID 1, the `nvim` short name, an exact `--embed` argv
field, an age at or above `COUCH_RESOURCE_ORPHAN_NVIM_AGE`) plus one new probe,
its cwd, read through `proc_pidinfo` only for rows that cleared those gates. A
cwd inside a managed checkout is positive evidence that pair spawned it, and
containment narrows the report, so an incomplete slot inventory reports less
rather than more. The managed roots are the **inventory** slots'
`WorktreeRoot` (`ManagedRoots`), which is also this sweep's coverage limit: an
orphan whose slot has left the console's slot list entirely has no root left to
match and is not named, so a worktree that outlives its slot escapes the sweep
until some inventoried slot's root contains it again. Rows appear under
`Orphan nvim (untagged; reap needs --confirm):` in `couch --resources` and as
`orphan_nvim` in `--json`, and leave the generic unattributed section — one
process under two headings reads as two findings. The persisted ring keeps them (`ResourceFrame.OrphanNvim`), so the
24 h history still records an offender the live report suppressed. One hub alert
fires per episode on the existing `resource_episode` seam after `OrphanSamples`
consecutive passes, addressed by the managed checkout the sweep matched, because
an orphan has no slot of its own to resolve a hub from.

`couch --reap-orphans [--confirm] [--json]` previews these rows, registered
`LeftRunning` entries and proven previous-incarnation processes (below). It
takes no slot. Without confirmation it reads the monitor's cached report and
signals nothing. A confirmed request obtains a fresh classification from the
sampler worker. The unattributed nvim and registered teardown families retain
`launcher.TreeReaper`'s tree cleanup (#399: SIGTERM, 3 s, SIGKILL, 2 s), with
descendants deepest first and process identities checked before signalling.
The whole nvim tree goes because killing only the editor reparents its language
servers to launchd and manufactures the next leak. Confirmation is enforced by
`couchmessage.ValidateRequest`; a hand-written unconfirmed request cannot reach
the destructive classification. Wire omissions are announced as `Coverage:`
lines on stderr, never treated as a complete census.

Owners: `couchcore/resource_orphan.go` (the pure classification),
`couchcmd/resources.go` (the gated probe and the sweep seam),
`couchcmd/reap_orphans.go` (the verb), `procutil/identity_age.go` and
`procutil/resource_darwin.go` (the two native reads the gate pays for).
`couchcmd/reap_orphans_live_test.go`, under `PAIR_LIVE_ORPHAN_REAP=1`, conforms
the composition against the real host: a real reparented embed at PID 1 inside a
managed checkout is named, an embed with a live parent in the same checkout under
the same age gate is not, and the confirmed reap ends it.

### Proven previous-incarnation processes (#792)

The same confirmed command can clean up a live slot's proven former processes
while preserving its current incarnation. Eligibility comes from the resource
sampler's positive orphan provenance: exact PID/start identity, original root,
thread and verified slot workspace. Names, PID existence or a matching thread
alone grant no authority. A bound caller must be the current authenticated hub
that owns the target slot; workers and other hubs are refused. There is no
special fleet-wide steward role. An entirely unbound local CLI uses the existing
UID-private owner-only namespace socket as the operator boundary. Partial or
stale credentials cannot fall back to that boundary.

The CLI freezes the admitted set and drives TERM, the shared 3 s wait, KILL for
proven survivors, and the shared 2 s wait. Every signal is a separate short
supervisor request. The existing sampler worker supplies a fresh shared
`collectOwnership` proof, including detached roots, before the effect. Unknown
detached evidence, incomplete inventory or changed ownership refuses the effect;
current ownership always vetoes remembered orphan provenance. Under the hub read
lock the supervisor checks the current caller and owning hub, rechecks the
nonempty process start identity and sends one signal to one PID. Native
check-to-signal identity checks do not provide an atomic PID handle.

This group never expands descendants, even below an eligible old root. Targets
that overlap a registered teardown entry are handled once through the exact-set
path; a refusal cannot fall back to tree cleanup. Overlaps are removed before
lossy projection. Legacy nvim/teardown tree admission also has a process-lifetime
fence: any incomplete table/inventory, selection overflow, ambiguous identity or
unknown ownership sets one worker-owned sticky flag. Subsequent healthy scans
cannot recover forgotten provenance and never clear it. Legacy admission then
refuses with `provenance incomplete; use the exact proven reap or restart Couch`.
Only a fresh process resets it. The flag is never persisted; no new registry or
process-wide lease is introduced. Every legacy TERM/KILL is a separate
`reap-legacy-signal` exchange with the exact root, target and closed family.
The worker returns the same current collection's full proof and legacy
classification, without retaining another table or signal permit. The pure
legacy decision requires current ancestry and excludes protected ownership.
The supervisor holds the hub read lock and the sticky gate's read lock through
its final flag check, root/target identity rereads and one signal; the worker
sets the flag under that gate's write lock. This revokes outstanding plans,
including KILL escalation. The CLI freezes and deduplicates descendants once,
then uses the exact executor and supervisor callbacks. It never signals legacy
processes locally. A surviving child whose root ended or whose ancestry changed
is refused or partial; remembered ancestry grants no permission.

The confirmed response uses `provenance-incomplete` with exact proven rows and
no actionable legacy rows, so older CLIs fail closed. The current CLI still
executes independently guarded exact targets, reports `legacy_refusal` in JSON
and returns nonzero for the unavailable legacy path. The sampler is constructed
once by the Couch service owner. Its worker starts after the existing inventory
seed attempt completes through one service-owned, cancellation-aware barrier;
requests remain unavailable until then. Healthy seed permits ordinary admission;
a failed seed sets the sticky refusal. Shared collection sets the sticky refusal using the pure
`ResourceOwnershipComplete` decision;
cache projection and persistence contain no copy of that authoritative flag.

Text and JSON preserve one final outcome per admitted target: `reaped` follows
positive absence after an acknowledged signal; `skipped` means its original
generation disappeared or changed, without claiming the replacement ended;
`refused` means confirmation, authority or current proof did not permit the
effect; `failed` includes signal errors, uncertain replies, cancellation and
survival past the final wait. Any refusal or failure gives a nonzero exit and
retains partial progress. An uncertain TERM reply permits neither automatic
resend nor KILL escalation. Inspect resources before a new explicit confirmation.
Coverage and displayed labels retain the existing output sanitization. The
2-minute command and 2-second exchange bounds remain unchanged.

Owners: `couchcore/resource_reap.go` (exact target and pure proof decision),
`couchcmd/resources.go` (worker-owned ownership collection),
`couchcmd/reap_proven.go` (exact supervisor authority and guarded effect),
`couchcore/resource_reap_legacy.go` (current legacy root/ancestry decision),
`couchcmd/reap_legacy_signal.go` (legacy effect gate),
`couchcmd/reap_legacy.go` (frozen forest and supervisor callbacks),
`couchcmd/reap_orphans.go` (CLI composition), and `launcher/exact_reap*.go`
(invocation-local transition model and effect interpreter). Public-command
conformance uses a disposable Couch with Claude and Codex; it does not require
staging a build on an operator slot.

### Resource policy and cost

| Setting | Default | Meaning |
| --- | --- | --- |
| `COUCH_RESOURCE_CADENCE` | `1m` | Sample interval, 1 second through 1 hour |
| `COUCH_RESOURCE_WRITE_BYTES_PER_SECOND` | `2097152` | Sustained disk-write threshold |
| `COUCH_RESOURCE_WRITE_DURATION` | `10m` | Required sustained-write interval |
| `COUCH_RESOURCE_PROCESS_RSS_BYTES` | `8589934592` | Per-process RSS threshold |
| `COUCH_RESOURCE_IDLE_CPU_PERCENT` | `90` | Idle-slot CPU threshold, percent of one core |
| `COUCH_RESOURCE_IDLE_CPU_DURATION` | `15m` | Required idle-CPU interval |
| `COUCH_RESOURCE_ORPHAN_RSS_BYTES` | `268435456` | Proven-orphan RSS threshold; two samples required |
| `COUCH_RESOURCE_ORPHAN_NVIM_AGE` | `6h` | How old an untagged embedded nvim must be before the sweep names it; a fresh one may still be mid-launch |

Set policy before starting Couch. Unknown or malformed resource settings refuse
startup. Unknown measurements interrupt accumulation rather than clearing an
already admitted incident. Alerts resolve the current owning hub and use normal
broker delivery. Episodes live only in memory; retries within a run reuse the
same effect ID. Restart begins fresh observations and may repeat an advisory
alert; sustained thresholds accumulate their configured interval again.
Unproven candidates remain report-only.

The displayed cost records attempted native calls, sampler-thread CPU and wall
time. At most three table size/read attempts plus two self-CPU reads cost eight
native calls; successful usage with identity fencing adds two per process, as does
a successful environment, argv or cwd read, and a read that fails before its fence
adds one. Sampler-thread CPU excludes concurrent GC and other goroutines.
The cost conformance (`TestResourceSamplerNativeCost`, env-gated on
`PAIR_RESOURCE_COST=1`) gates work, and steady CPU as a ratio to a same-run
baseline, rather than absolute CPU (#644). Absolute CPU cannot give one verdict on
an idle host and on one running the rest of the fleet:
measured across a wide load range, a pass's native call counts barely moved while
its CPU moved by tens of percent, which flipped a fixed budget on unchanged code.
What the conformance asserts instead is load-invariant. Every pass's native calls
must equal the sum of its per-category counters plus the two self-CPU reads, so no
native work goes unaccounted; no category may read a row more than once, at two
native calls per read over at most the table's rows; and each pass writes the
near-cap history exactly once. The first pass — #606's one-time environment scan,
the admission burst #587's budget deliberately excluded — is bounded by that work
alone, and its CPU is reported rather than asserted. Steady passes are gated
against a baseline taken in the same run: at most twice the thread CPU of the
history write the pass itself performs. Every pass reports the host state it was
measured in (table rows and load average) and every failure names it, so a red run
distinguishes a busy host from a sampler regression without a second
investigation. The measurements behind each constant are recorded in #644. The
product requirement is unchanged: well under 1% of one core. Current-pass final CPU
is published in the cached view after storage completes; its history frame cannot
include the completed cost of writing itself.

History retains aggregate slot frames and orphan/unattributed summaries for at
most 24 hours, 1,440 frames and 4 MiB. Only history is persisted, best-effort, with one fixed replacement stage:
at most 8 MiB including the current file and stage. See [resource snapshot retention](storage-retention.md#resource-monitor-snapshots-587).

The cached response is a byte-budgeted projection (`resources_projection.go`)
within the broker's frame envelope. It preserves all bounded slot aggregates,
shortens long presentation labels, and reports omitted detail counts separately
for processes, orphans, unproven candidates, episodes and diagnostics. Projection
cost is included in sampling. Incomplete
startup or capped inventories cannot prove owner disappearance.

History read failures begin a fresh in-memory ring with a visible continuity
warning. Write failures do not roll back that ring or gate alert delivery.
Unsafe paths remain protected by the existing file checks. State, provenance,
rate baselines and alert episodes end with the supervisor; pre-existing orphans
remain unproven after restart unless fresh ownership evidence establishes them.


### Hosted terminal ingress diagnostics (#622 M1)

`ResourceView.HostedIngest` is an optional, read-only request-time section beside
the cached process sample. `ptychild.Child.ingest` adds len(p) once to the child's
atomic counter at entry, including later failed parsing/publication; it retains
no payload and adds no clock or worker. `Child.IngestBytes` reads atomically.
`couchtty.Console.HostedIngestSnapshot` copies ordered hosted panes, including
hidden ones, under the existing console mutex without acquiring the endpoint
parser lock. Reaped children disappear; a replacement has a fresh endpoint ID.
The section retains at most ResourceMaxSlots (256) clients and counts omissions.

The resources handler adds supervisor PID/start time and projects the fresh
section through the shared encoded-byte budget in resources_projection.go.
The JSON `hosted_ingest` object names `sampled_at`, `server_pid`,
`server_started_at`, `clients` and `omitted`; each row has `endpoint_id`, `thread`,
`process` (PID/identity), and `ingest_bytes`. A missing section is unavailable;
a supported empty snapshot remains present. Server and endpoint identities must
match between samples before computing bytes/s; disappeared or omitted clients
cannot count as zero. This timestamp is independent of the minute-cached
ResourceView.SampledAt. Query-time snapshot/projection cost is outside the
periodic sampler's reported cost. No hosted counters enter persisted history.

Measurement procedure: preserve two complete JSON observations across natural
60-second idle and busy intervals, bracket with cumulative server CPU plus a
5-second native sample, and record host load, concurrent suites, visible console
and wrapper states. Report identity-matched byte rates separately from total
server CPU. Profiles identify hot paths, not exact per-client CPU attribution.
M1 changes no parser policy; optimization awaits the measured comparison.

### Eager ASCII processing (#622 M2)

Hosted endpoints keep parsing output eagerly whether visible or hidden. In
`third_party/vt/utf8.go`, `handlePrint` skips general grapheme segmentation only
between two single printable ASCII characters, retaining the provisional cluster
and normal `handleGrapheme` transition for subsequent Unicode amendment. In
`terminal/frame.go`, `validateCell` skips segmentation for a single printable
ASCII byte of width one after its existing content/link/width checks. Queries,
effects, modes, geometry, history and publication retain the same authority and
ordering; there is no lazy replay, extra buffer or visibility policy.

Qualification uses differential slow references plus complete Child ingest,
publication and frame validation benchmarks. Fixed-work ABBA child CPU measured
13.7–15.6% lower ASCII redraw cost and 2.8–6.3% lower mixed redraw cost at the
approved quiet-load criterion. These figures are fixture results, not live
server savings. TL-authorized landing precedes the next coordinated Couch
restart; live server CPU/sample and console/peek/waiting acceptance remain
pending until that check.

### Interrupted park recovery (#651)

`couch --recover-park repo:N --confirm [--json]` resolves the current interrupted
park through the authenticated owning-hub operation queue and receipt protocol.
The switcher offers the same confirmed **Recover park** action only while a
thread has an open park. Both carry the exact repository scope, tag and park
nonce; queued effects refuse a replacement transaction or changed ownership.
A repeated request uses its original receipt; recovering an already settled park
has no effect. Unknown receipt status calls for inspection, not an automatic resend.

CLI recovery selects its exact address and park nonce from the console's last
successful cached inventory before receipt admission (#681). The authenticated caller's admission-verified workspace cache and canonical target slot supply the initial hub ownership check, without external caller/target workspace resolution. A missing caller cache refuses with refresh-and-retry advice. Current-world caller, ownership and target checks remain on the owned queue and at dispatch. Unknown, missing
or ambiguous cached selection refuses with advice to refresh the switcher
inventory, inspect the row and retry. Full workspace/fleet inspection runs only
on the existing operation queue after the receipt is allocated; slow preparation
therefore remains inspectable instead of exhausting the two-second admission
exchange. Queued preparation and dispatch compare against the captured identity;
a replacement park is refused rather than selected later. Repository aliases
share one canonical pending key and one bounded name lookup on new admission.

Recovery never starts a replacement agent. A verified completion finalizes the
park; positively dead exact owners with positively absent terminals may retire
interrupted bookkeeping while preserving incomplete-cleanup diagnostics. Missing
or failed completion is never reported as verified cleanup, and uncertain or
foreign ownership refuses. The result names the available Continue, Resume from
disk or Start fresh action after normal recovery evidence is checked. Conversation
identity survives; missing scrollback bytes cannot be reconstructed.

Public declaration/admission lives in `couchcore/ops.go` and `slot_operation.go`;
execution and receipt wiring in `couchcmd/console_slot_operations.go` and
`slot_operations.go`; row authority and confirmation in `couchtty/menu_actions.go`
and `menu.go`. The recovery decision/controller and journaled store mutation live
in `couchcore/park_recovery.go` and `threadstore.go`.

An open transaction's row shows the shared unfinished-transaction/Recover park
advice, including on a still-live row; retained continuation status stays appended.
With an absent terminal and live or unknown exact owner, explicit recovery
refuses before entering cleanup reconciliation. A live-owner refusal says the
transaction is still finishing and names the same recovery action to retry after inspection.

Recovery carries its worker context through lifecycle lock admission, protocol
observation/publication/cleanup, and checked threadstore mutations. Contended
locks cannot retain an abandoned recovery worker; existing synchronous filesystem
work and owned temporary cleanup finish before its reservation is released.
Continuation, switch, detach, leave and direct recovery refusals reuse the same
bounded exact-target advice without changing their ownership guards.

Inactive-owner tab chips and per-owner collapse counts share `inactiveOwnerStyle`
in `idle_shade.go`: quiet content blends at least50% toward the known terminal
background (stronger idle fade stays stronger), quantized for256-color hosts.
Without a palette, the existing faint rendition remains; NO_COLOR disables the
owner fade. Attention, themed red and placeholder cues keep their precedence.
Global overflow has no single owner and keeps its normal style. The shared
renderer applies this to collapse and multiline without changing hit spans.
