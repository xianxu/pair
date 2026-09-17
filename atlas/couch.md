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

The absolute physical `COUCH_STORE_DIR` is one durable namespace. One Couch
supervisor owns it through a non-inherited advisory lease; another supervisor
refuses with verified PID/process-start identity. `couchcore.ThreadStore` is
the mutable authority for composite thread records, using one global store
lock, revision-checked record updates, and a recoverable write-ahead journal
for membership or multi-record changes.

`registry.json` remains as a transitional live-handle cache for the shipped
console. It is not a metadata or display authority. The one-time journal import
of its actors into ThreadStore went with `pair#170` M4: every store that needed
it was cut over years of commits ago, and the manifest keys that recorded the
cutover survive only as decode tombstones. CLI diagnostics read the raw
one-row-per-composite-thread inventory; the ordinary switcher reads the
actionable projection described below.

That raw `ThreadInventory` remains the diagnostic/recovery view: persisted
incarnation states are shown even when Couch cannot prove a usable terminal.
M1 exposes `ActionableThreadInventory`, a pure fail-closed projection over the
same snapshot plus exact owner observations. It emits only `live` when one
durable live PID/start identity exactly matches one observed TTY owner, or
`parked` when verified park exists with no active park transaction, reservation,
or incarnation. Contradictory and undecodable records stay available to
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
operation identities (`rename` presents the shared `name` operation). Inputs
are byte-bounded at 1 KiB for filters/names and 4 KiB for paths/descriptions.
The input seam decodes horizontal arrows in both CSI and application-mode SS3,
so the start form's agent selector is reachable in either terminal mode. Root
rows clip variable label/path text around a protected state/age/bell suffix at
the 40-column minimum. Generated key traces keep stack depth, UTF-8 ownership,
and effects bounded
(ARCH-DRY, ARCH-PURE, ARCH-CONSTRAINTS).

Both CLI resolution and in-memory menu filtering derive from
`ClassifyThreadReferenceFields`/`MatchThreadReferenceFields`: exact opaque tags
win set-wide over case-insensitive name/path containment, with no store read on
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
`--archived`; the hosted-agent hook `publish-description` projects only through
hidden `couch --internal publish-description <text>`. `prepare-start`, `start`,
`attach`, `switch`, `park`, `resume`, `relaunch`, `prepare-switch-agent`,
`switch-agent`, `leave`, `stop`, `name`,
`describe`, `archive`, `recover-thread` and `recover-checkpoint` are TUI/in-process operations. `orientation-status` is
an internal owner operation for one launch attempt.

Continuation has four internal operations (`pair#249`):

- `request-continuation`, invoked as `couch --internal request-continuation <absolute-path>`, durably accepts the hosted source's exact checkpoint. The inherited scope, tag, agent, session, launch ordinal, and expected digest bind publication to the writer's validated bytes and current source generation. This metadata operation can run in another worktree without becoming a second supervisor.
- `continue-thread`, invoked in process through `couch --internal continue-thread`'s declared operation, executes or reconciles an accepted request under the live owner.
- `retry-continuation`, exposed in the switcher's thread actions, reconciles a retained failure. After Couch has exited, `couch --internal retry-continuation <tag>` in the thread's repository acquires the normal singleton lease and opens a Console for recovery. It refuses a competing owner.
- `continuation-status`, represented by `couch --internal continuation-status`, reconciles the exact launch attempt's orientation receipt under the live owner. The Console supplies the address, request ID, and attempt through the typed operation arguments.

### Stale-thread recovery

`recover-thread` reobserves the selected address and prefers warm attachment to
an exactly owned detached session. `recover-checkpoint` accepts one absolute
`path` (4096-byte limit), reads the bounded checkpoint, and starts a new
conversation through the existing continuation executor. Both use the Console
operation queue; internal CLI recovery and `archive` acquire the same namespace
supervisor lease. The CLI resolves their repository scope from the caller.

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

Archive uses the same reconciliation, then checks occupancy before quiescing.
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

`relaunch` (`pair#182`) is detailed under **Exit, detach, and terminal
lifecycle**; the one thing worth knowing at this level is that its commonest
refusal is not a fault. A cold resume needs `--resume <native-id>`, and that
name comes from the ledger's `binding` row, which is written only once the agent
completes a turn. A thread started minutes ago and never used therefore has no
proof of WHICH conversation to resume -- the ordinary state of a fresh session,
and the state relaunch meets most, because relaunching is something you do to a
session you just started. It says so rather than guessing.

`archive` is the operator's "delete", and it is COMPLETE: it stops the thread's
zellij session first (`Artifacts.Quiesce` -> `zellij delete-session --force`,
polled until the session is verifiably gone), then removes the thread from the
working set and KEEPS its record, moving `threadstore/records/<scope>/<tag>.json` to
`threadstore/archive/<scope>/<tag>.json` and dropping the address from the
manifest in one journal entry, so a crash cannot leave a record in both sets or
neither. Restoring is that move reversed plus a manifest re-add -- `Snapshot`
walks the manifest, so a restored file the manifest does not list stays
invisible. It refuses a live/unknown helper or an open start/park transaction: archiving a record couch is
hosting would leave the console owning a thread the store no longer lists. Exact helper-death proof permits
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
selects claude, codex, agy or muse, then edits parameters loaded from the shared
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
resizes, the control constants). See [Terminal ownership](terminal.md).

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
(`placeholderSGR`). The thread starting now carries the spinner, from the
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
(a branch beside `hotkeyByte`, since it is not an escape sequence) and the Kitty
`\x1b[127;5u` (an ordinary `knownSequences` row). In legacy encoding `0x08` is
`^H`, so ctrl-h is taken from the child too -- deliberate, and harmless under
the Kitty protocol zellij pushes. `panelkeys.go` computed a `modified` flag and
then ignored it for backspace, so the CSI-u form decoded as a plain backspace;
that is fixed as defence in depth, since the interceptor claims both encodings
before the panel sees them but forwards paste content verbatim.

`ctrl+return` **answers the newest page** (`pair#221`). From an actor it lands on
`attention.NewestActor()` -- the thread `ctrl-space` would have opened the
switcher on -- with no switcher in between. In code it is `ctrl+backspace`'s
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

The chord uses Kitty keyboard disambiguation (`newestPageSequence`,
`\x1b[13;5u`); explicit press and repeat forms also jump, while release does
not. The presenter owns its keyboard-protocol stack entry and restores it at release.
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
Enter switches a proven live row or resumes an exact verified-park row, and
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
and retains the form plus local error. Park, leave, rename, and description use
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
row plus one per pending attention message, so `RenderMenuView` returns
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
`unsupported-agent`, `unknown`. (`stale-incarnation` and `unrecorded-child` were
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
views, named rather than hidden. Ephemeral console targets bind only to durable proven-live rows,
so a stale child handle cannot turn an inactive row's Enter into switch. If
Park removes the final actor while the switcher owns focus, the console remains
available for the refreshed resumable row. Lifecycle shortcuts are panel-only
(#245): Alt+x opens the typed `leave` confirmation in its park disposition;
Alt+d performs the detach sweep without confirmation. Individual thread actions
remain in the switcher. While an actor is displayed, those raw chords reach
Zellij and the receiving pane. No inner-pane focus cache or key-time query exists.
The agent consumes only Shift+Alt+T/Left/Right; Couch consumes its three navigation
chords. `Interceptor` frames candidates, then Console routes the preceding bytes
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
through the `PairLifecycleController`'s bounded, capacity-one worker. Startup
recovery, Park, Retry, Recover, Abandon, and Leave all enter that same boundary;
same-address/same-nonce overlap shares one future, while other work overloads
without lifecycle effects.

**Alt+n / Ctrl+Alt+n relaunch the highlighted switcher row** (`pair#182`,
`pair#245`). Couch replaces the helper with the current binary and keeps the
conversation. While a Pair pane is displayed these chords pass inward: the agent
receives input; other panes retain Pair's existing in-process reload. There is
no whole-Couch relaunch; leave the switcher, rebuild and run Couch again.

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
inventory, resume execution and its post-claim recheck (`pair#248`). None of
these warm paths resolves or requires a native conversation binding. The
inventory passes only candidates (no incarnation, no verified park, a usable
saved profile and working path), which bounds
*whether* the zellij snapshot runs -- a couch with nothing detachable pays
nothing -- and, since `pair#228`, its fan-out too: two `list-sessions` runs plus
one `action list-clients` per *candidate* session, not per session on the host.
Before that it asked every live pair session, **measured at 1.49 s** on a
13-live-session host (2026-09-02). `list-clients` is the expensive call, about
250 ms against a real detached session, so the whole reattach path now asks it
of two sessions: the detached proof and its re-proof. `pair resume`'s launcher
and couch's registration poll read liveness only (`SessionLive`). See
`cmd/probes/reattachcost` for before and after.

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
  is the list's one home. Each filters before it reads -- to the cwd, or to rows
  whose layout differs -- so `startupAsks` resolves exactly that union and leaves every other candidate
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

Resume accepts verified park **or proved detachment**. A detached thread has no
verified park because nothing was torn down; its authority is the surviving
session. Both `DecideResume` (Enter's gate) and
`ProjectActionableThreads` (the switcher's list) carry that second authority --
widening only one would list a row whose Enter fails, or hide a row that would
have worked. A third gate used to sit between them, `ReconcileResumeAdmission`,
which re-checked fleet capacity before relaunching; it went with admission in
`pair#170` M4. The detached branch is checked BEFORE the `ParkHistory`
tombstone scan, which refuses on any tombstoned entry with no break: a thread
once abandoned mid-park and later detached would otherwise be permanently
unreattachable. The occupied-incarnation refusal is unchanged, because detach
retires the incarnation and the record passes on its own merits.
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

Interactive `couch [<repo>]` startup resolves the requested repository scope and
physical working path, then applies `SelectResumableRoot` to the same
proof-bearing actionable inventory used by the switcher. It RANKS: detached
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
class that most needs it. But archiving one never stops its session: the guard
that proves a thread is not live needs a decoded record, so quiescing would kill
an agent on the strength of a record couch just failed to read. The archive
returns `ArchiveResult.Warning()` saying so.

Both projections take one `ThreadProjectionInput` (records + evidence +
unreadable). The three used to travel separately with the unreadable set as a
trailing variadic, which meant omitting it compiled cleanly and silently
restored "some records get no row" -- the regression the total projection exists
to prevent. One value makes the omission named and visible at each construction site;
`FromSnapshot` is the form that cannot forget.

`PathHoldsUsableThread` is the other half: **one thread per repository path**,
enforced at the single site every creation entry funnels through
(`spawnResolved`), refusing a start where a live, detached or parked row already
holds the path. Known debris does not block -- a path whose only rows are
unusable-but-classified must stay startable. An UNREADABLE record is the
deliberate exception, and it accepts the hazard this sentence used to warn
against ("one corrupt record locks its repo out permanently"): couch cannot tell
which path such a record holds, so it blocks the scope rather than risk a second
thread over live work. The lockout is bounded by naming the record's file in the
refusal, and by the switcher reached from another repository -- which is the
recovery path, and is stated in the refusal because an unstated escape is no
escape. In total version skew every record is unreadable and no repository
starts; the file path is then the only honest next step. The
TREE is not the bound; two threads in one tree at different subdirectories
remain legal. There is deliberately no opt-in flag: `StartArgs.SameTree` looks
like one and is documented as inert legacy serialization, so reading it would
resurrect a dead field as policy.

A row's label is `threadLabel`: the human name, else the working directory's
last segment, else the tag. `DisambiguateLabels` appends the tag's tail to
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
SIGHUP all revoke child-enabled mouse/focus/paste/synchronized-output/extended-
keyboard modes, reset the scrolling region, clear the reserved row, leave
alternate screen, restore/show the cursor, restore raw mode, stop host event
sources, close the blocking input seam, and join console workers before
returning. This explicit reset is required because restoring termios does not
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

Path preferences are strict revisioned records below
`threadstore/path-preferences/`, addressed by a digest of normalized repository
identity plus canonical physical path while retaining both values in the
record for validation. The resolved profile travels to Pair as a strict
tag-bound `PAIR_COUCH_LAUNCH_PROFILE`. `PAIR_USE_REPO_DEFAULT=1` accompanies it
only for matching repo-default provenance; path provenance supplies one
authoritative empty value. `ExecRunner` overlays supplied child keys after
removing inherited duplicates, so stale launch policy cannot cross the process
boundary. Pair consumes both keys before launch and does not persist
Couch-resolved argv back as a new repository default.

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

### `SessionObservation` — three values, not a boolean

`couchcore/sessionevidence.go`. `SessionUnresolved` is the **zero value**, so an
observation nobody populated fails closed: if absence were the zero value, a
gather branch that silently stopped running would assert "no session" for every
thread it skipped — the anonymous refusals `#181` removed. `session-gone` is
archive-eligible, which is what makes the distinction load-bearing.

There is deliberately no fourth "held elsewhere" value. The refresh never counts
clients — `list-clients` costs ~250 ms per live session (`#228`) — and the
reattach path re-observes attach state before committing. **Optimistic inventory,
strict action:** the expensive question is asked for the one thread the operator
pressed Enter on, so cost is proportional to what you *do*, not what you *have*.
`DetachedSessions` remains the action path's authority, guarded by
`RequireAttachState`.

### One class, four sites

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

### `ThreadBusy` has exactly one producer

A `ThreadStartClaim` — couch's record of its **own** in-flight operation, not a
claim about an external process. Without it, the window between claiming a start
and the launcher acquiring a pid would classify `session-gone`, an
archive-eligible reason, for a thread starting normally.

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
- **parked thread** — a durable thread with an exact verified resume handle and
  no occupied incarnation.

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
