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
`attach`, `switch`, `park`, `resume`, `relaunch`, `leave`, `stop`, `name`,
`describe` and `archive` are TUI/in-process operations.

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
invisible. It refuses a LIVE or mid-park thread: archiving a record couch is
hosting would leave the console owning a thread the store no longer lists, which
is the stale-incarnation shape by construction. Every other state goes,
including every unusable reason, because the operator decides a thread is
finished.

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
couch starts `pair resume <tag> --layout2` inside a child pty and owns the
operator tty until the console exits. Verified by operator smoke; the
alternative (couch absorbing zellij's role) was considered and rejected because
the agent child is never spawned by Go — zellij spawns it from a KDL layout, and
`entrypoint.ValidRootMarkers` *defines* a valid pair install as having those
layouts.

**Couch launch IS the console (`pair#146` M2).** It allocates a pty per child,
puts the operator's terminal in raw mode, and routes bytes -- so it no longer
hands the child its own stdio and blocks. The mechanism is shared with `pair term`
rather than written twice: `cmd/internal/ptychild` (a child on a pty, its
bounded replay ring, the #127 query deny-list, one scanner over its output) and
`cmd/internal/hostty` (the operator's terminal: size, raw mode, coalesced
resizes, the control constants). See `atlas/architecture.md`, "The terminal
plumbing is shared with couch".

Public launch requires terminal stdin and stdout before store, lease, or
actor work. The stdio runner remains an injected domain seam and live
conformance target, but is no longer selected by public argv.

**The pty is a CAPABILITY on a handle, not a second Runner signature.**
`Runner.Start` is unchanged; a handle from `PtyRunner` additionally satisfies
`TerminalHandle`. `ExecRunner`'s does not, and a test asserts that -- a
capability check no runner can fail is vacuous. `Terminal()` returns the
concrete `*ptychild.Child` rather than an interface, because `FakeRunner`'s
double IS one, so a test takes the branch production takes.

## The reserved row: a reservation, not compositing

The child is given a terminal one row shorter and the host's scrolling region is
pinned above the last row (DECSTBM). The child is never told, so this is a
resize rather than compositing.

Three things that design has to survive, each learned the expensive way:

- **Scrolling.** What DECSTBM is for. A child scrolling at the bottom of its own
  screen scrolls inside the region.
- **Erasing.** DECSTBM does *not* cover it. Every full-screen app clears the
  display on startup and that takes the row with it while the region stays
  intact -- which is why the signal is `Screen.TakeRowDirty` (erase, margin
  reset, RIS, alt-screen transition) rather than anything named for the region.
  The console repaints on it.
- **Not corrupting the child.** A pty read boundary falls wherever the kernel
  puts it, so a paint written between two chunks can land inside one of the
  child's escape sequences. Two rules keep that impossible rather than unlikely:
  **`Console.Run` is the only goroutine that writes to the host** (resizes and
  hotkeys are events it drains, not writers), and every console-originated write
  goes through a gate that defers while the CHILD's stream is mid-sequence,
  paying the debt on the next chunk that ends on a boundary.

  Both halves were learned by getting them wrong. Asking the *child* whether it
  was mid-sequence answered about a later chunk, because ptychild's pump feeds
  its scanner before the console has drained the earlier one -- so the tracking
  belongs to the stream the console WRITES. And feeding the console's own
  escapes into that scanner let it frame our bytes together with the child's
  partial and report "safe" precisely when it was not, so the scanner is fed
  child bytes only.

Verified against a real terminal emulator (`vtscreen_test.go`) and against a
real pty child (`console_live_test.go`, `PAIR_LIVE_COUCH=1`), and confirmed by
operator smoke on the full Ghostty -> couch -> pair -> zellij -> claude stack
2026-08-23.

## Navigation

`ctrl-space` is intercepted before the child sees it. It arrives in TWO
encodings and both are recognised: the legacy `0x00`, and CSI-u
`\x1b[32;5u` under the Kitty keyboard protocol, which zellij enables -- so the
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

**The projection is TOTAL** (`pair#181`): every record in the manifest becomes a
row, and `ClassifyThread` returns a state plus, when the row cannot be acted on,
a `ThreadReason` from one closed vocabulary -- `binding-lost`,
`stale-incarnation`, `unrecorded-child`, `session-gone`, `never-started`,
`invalid`, `path-missing`, `profile-missing`, `unsupported-agent`, `unknown`.
Failing closed is unchanged -- an unproved row is not actionable and startup
never selects it -- but it is expressed as a state rather than as absence. The
IO shell (`gatherThreadEvidence`) resolves evidence and decides nothing;
`ThreadEvidence` carries a `ProofStatus` per question, so "we asked and the
answer was no" and "we could not ask" are different answers. Without that
distinction one failed zellij query would assert `session-gone` on every
detached row, and `session-gone` is a reason retirement acts on. `couch --list`
and `--show` classify through the same function over the same evidence, with
OS-derived liveness in place of the console's pty proof, so one store cannot
produce two stories. Ambiguous and legacy-unverified records now appear in both
views, named rather than hidden. Ephemeral console targets bind only to durable proven-live rows,
so a stale child handle cannot turn an inactive row's Enter into switch. If
Park removes the final actor while the switcher owns focus, the console remains
available for the refreshed resumable row. The two lifecycle chords read as a
2x2: the KEY chooses the disposition (Alt+x parks, Alt+d detaches) and the
SURFACE chooses the scope (an actor means that thread, the switcher means every
live thread and then leaving). Alt+x on the switcher therefore opens the typed
`leave` confirmation in its park disposition, parks every live thread
sequentially, and closes the console only after durable success and exact
Pair-child death; Alt+d there does the same sweep with detach and no
confirmation. Confirmation rides the disposition, not the scope.
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

**Alt+d is Couch's own detach** (`pair#170`), intercepted like Alt+x and for the
same reason: un-intercepted, Pair's `PairConfirmDetach` runs `zellij action
detach` from inside the session, leaving Couch with a dead child and a stale live
incarnation that the fail-closed projection hides -- the operator's safest
gesture would make the thread disappear. Detach is park's WARM counterpart:
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

**Detached is a derived actionable state, not a persisted one.** `launcher`
already classifies a live zellij session with zero clients as `SessionDetached`,
and `pair resume` already reattaches onto one, so `ProjectDetachedSessions`
consumes that rather than teaching Couch a second way to ask. It fails closed
both ways -- two addresses claiming one session name, or two zellij rows sharing
one name -- and the projector's detached branch requires ZERO incarnations, which
is what keeps a crashed Couch's stale `IncarnationLive` from masquerading as a
clean detach. `DetachedSessions` takes **candidates** rather than returning the
whole set, because the session-name index is per repo scope. Each candidate
carries the resume proof its caller already resolved (agent + native id) and the
observation carries it back, so `detachedResumeProofMatches` — the pure twin of
`parkedResumeProofMatches` — enforces it in the projector rather than trusting
the shell. The inventory passes only candidates (no incarnation, no verified
park, a saved profile, an established binding), which bounds
*whether* the zellij snapshot runs -- a couch with nothing detachable pays
nothing. It does not bound the snapshot's own cost: that is two `list-sessions`
runs plus one `action list-clients` per non-exited session **on the host** --
**measured at 1.49 s** on a 13-live-session host (~100 ms per session, serially),
2026-09-02.

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
  a detach candidate the normal case. `pair#172` parallelizes the per-session
  queries, which are independent; the candidate filter decides only *whether*
  the snapshot runs.

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
returns an `UnreadableArchiveWarning` saying so.

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

Two neighbouring states are deliberately never SELECTED, though both are now
listed. A session **attached elsewhere** yields no detached observation, so
couch cannot steal it. A **stale `IncarnationLive` from a crashed couch** shows
as `unusable/stale-incarnation` and is not selectable, so startup creates a new
thread; automatic reconciliation is the gap `pair#171` owns, and archive is the
manual out.

**The native-binding gate no longer hides a row, and no longer applies to the
warm path at all** (`pair#181`). It once did both: `ActionableThreadInventoryContext`
dropped every candidate whose binding was not one exact established root, and
`DecideResume` demanded that binding for detached threads too. The reasoning was
sound and the conclusion was wrong. Startup has NO fallback by design
(`pair#167`), so a Resume refusal stops `couch` rather than starting something
else, and the invariant that makes that safe is *a row the inventory offers is
one resume can take* — but the gate enforced it by making the row VANISH, and
the native session id it demanded is the COLD path's proof, which a warm
reattach never consumes.

The atlas recorded the fork before it was taken: "The alternative — list it,
refuse its `Enter` with the diagnostic, and gate only startup selection — was
not weighed when the gate was written; it is the fork to revisit if an operator
hits this." The operator hit it. That alternative is what `pair#181` built: the
row is listed as `binding lost — repairable`, `Enter` explains, and
`SelectResumableRoot` never offers it because only proven states rank. A
detached thread whose binding degrades is now visible with its reason instead of
being a live agent nothing mentions.

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

## Spawning: `pair resume <opaque-tag> --layout2`

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
`COUCH_THREAD_TAG`, and launches as `pair resume <opaque-tag> --layout2`.

`COUCH_INPUT_TRACE=<path>` (`pair#182`) is the one env var couch reads for
ITSELF rather than passing down: it appends every operator keystroke couch
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
Except for matching the scope/tag to establish Pair's reserved address claim,
Pair treats these Couch-owned values as opaque pass-through context for the
hosted child: it does not resolve Couch names or paths and never reads or
mutates Couch's manifest or records.
Distinct starts at one path therefore use distinct Pair
sessions and artifacts. Layout stays pinned to layout2: couch owns terminal
switching, so layout3's third pane is the layer couch replaces.

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

## Liveness is recomputed, never stored

Because Couch owns the console, diagnostic flags run in a **second process**
with no `Handle`. So `ActorRecord` persists `{PID, Identity}` where `Identity`
is `procutil`'s kernel start token, and a reader recompares it: a recycled PID
reports not-live because the token differs.

Within one process, `ExecRunner` reaps its children in a background goroutine
and liveness is a closed channel — **not** `kill -0`, which succeeds for a
zombie and would report an exited-but-unreaped child as running.

## The bounded mailbox

`Enqueue` (`mailbox.go`) is a pure function: collapse by kind, drop the oldest
non-control entry over capacity, never drop control. `couchtty/notice.go` uses
it for the exit/bell feed.

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
