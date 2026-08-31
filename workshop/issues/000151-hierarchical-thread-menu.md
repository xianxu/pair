---
id: 000151
status: working
deps: [149, 152]
github_issue:
created: 2026-08-24
updated: 2026-08-30
estimate_hours: 7.60
started: 2026-08-30T15:58:54-07:00
---

# couch: hierarchical work-thread menu

## Problem

`#146` proves deterministic tty switching, but its flat transitional panel has
no discoverable home for secondary actions after the unintuitive `:` command
namespace is removed. Adding actor-specific actions there would encode the
wrong identity: actors are live harness incarnations, while `#149` makes the
durable work thread the thing an operator names, describes, parks, resumes, and
returns to.

## Spec

**A filtered hierarchical menu over actionable work threads.** The root list is
one row per durable work thread from `#149` that the shared core projection can
prove is either live or verified parked. Rows lead with the human name when
present and the opaque tag fallback when not, then show path, live/parked state,
notification state, and—when parked—relative last-active age. A live row says
`live` rather than presenting a historical age. Multiple threads at one path
therefore remain visibly distinct.

The core—not the terminal UI—projects stored lifecycle detail into this user
model. Its pure input is the durable records plus a set of ephemeral live-TTY
observations keyed by exact thread and incarnation identity. Only the live
Couch owner can publish such an observation, after registering the pane and
verifying its child; persisted incarnation state alone never proves a live row.
A thread is live only when one current observation matches its durable
incarnation; it is parked only when it has an exact verified resume handle and
no occupied incarnation.
Transitional, ambiguous, corrupt, abandoned, legacy-unverified, or otherwise
unsupported records remain durable but are omitted from the ordinary switcher;
their diagnostics belong in notices/logs and a future recovery surface. Start
and park retain their initiating UI context while in progress, and publish a row
or state change only after reaching verified live or parked. The projection is
a shared core function: the terminal supplies its owner observations, and any
future CLI/advisor consumer must obtain equivalent owner evidence or omit live
rows rather than promote persisted state. A separate explicitly diagnostic raw
inventory retains every valid durable record for recovery tooling; it is never
the ordinary switcher source (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

Notification is an ephemeral join over locally hosted TTYs, not persisted
thread lifecycle. Bells from any inactive local incarnation mark its thread;
switching to that thread clears the mark. Historical or remotely unobserved
threads have no inferred notification state.

Every menu level is a frame with its own stable item identities, filter text,
and selected identity. Printable input filters the current list. Filtering keeps
the selected item when it remains visible, otherwise selects the first match;
zero matches means no selection. Up/Down move within the filtered rows. In the
thread list, Tab pushes the selected thread's action frame. Tab is a quiet no-op
in action and confirmation lists and in rename/describe text input; in the start
form it moves between path and agent. Escape pops one nested frame, and the
parent frame returns with its filter and selection unchanged. Enter invokes the
selected item's primary action. With no selection, Enter and thread-list Tab
report `no selection` and do not change levels.

Escape at the root follows the transitional panel's familiar two-step rule: a
non-empty filter is cleared while retaining the selected thread when possible;
with an empty filter, Escape returns to the active thread through the forced
clear-and-replay switch path. If no live thread can receive focus, the root menu
stays visible and reports why. Escape from a nested list always returns one
level even when that child has a filter; the parent frame is restored exactly.

At the thread level, Enter switches to a live TTY already owned by Couch or
resumes a parked thread. This switch is Couch's existing presentation behavior,
not a generalized application attach/detach contract. This issue requires only
Pair's durable suspend/resume contract; live-only applications, persistence of
live processes across Couch exit, and richer workspace-pane composition are out
of scope.

Tab opens that thread's secondary actions. The first actions are `park` for a
live thread, `rename`, and `describe`; a parked thread offers `resume` instead
of `park`. Actions capture the durable thread ID when the submenu opens and
revalidate it at dispatch. A thread that disappears produces a notice and pops
back to the refreshed root list; it can never redirect an action to whichever
row moved into the old screen position. Dispatch also revalidates action
applicability against current live/parked state. If the state changed, no stale
action runs; the menu returns to the refreshed root list with that thread
selected when it remains actionable and reports the change. If the target no
longer belongs in the actionable projection, the menu pops to the preserved
root frame, applies its ordinary stable-selection fallback among the remaining
rows, and reports the target's label and diagnostic location; it does not
delete the hidden durable record.

Every async completion and successful inventory refresh reconciles the stack
against the new actionable projection before applying ordinary success/failure
restoration. Starting at the root, each thread-bound action, confirmation, or
rename/describe frame survives only while its captured identity remains
actionable and that action remains applicable; the first invalid frame and all
its descendants are discarded through the hidden-target transition. A typed
rename/describe draft in such an invalidated frame is not reassigned or
dispatchable and is discarded with an explicit notice. This safety transition
takes precedence over the normal promise that a failed operation preserves its
input. The global start form is not thread-bound, so it remains open; only its
saved originating stack is reconciled before Escape or success restores it.

`park` always enters a confirmation frame. Its rows are `cancel` (selected by
default) and `park <thread label>`; Escape or Enter on cancel returns to the
action frame, while Enter on the explicit park row dispatches. Success returns
to the root list with the same thread selected and marked parked. Failure leaves
the action frame intact and reports through the notice feed only while the
thread still projects as live. If the failed attempt leaves it in a hidden
unresolved lifecycle, the action frame is no longer valid and follows the
hidden-target root transition above.

`rename` and `describe` open text-input frames. Printable keys append and
Backspace edits. Escape cancels back to the unchanged action frame. Enter sends
the exact text through couch's shared operation surface; validation remains the
operation's responsibility. Failure keeps the input open with its text intact
and reports a notice. Success refreshes thread data and returns to the action
frame. No terminal UI action gets a private dispatch path unavailable to the
advisor (ARCH-DRY, ARCH-PURE).

Ctrl-Space is the global start action while any menu-list frame is visible: it
opens a two-field start form without discarding the menu stack. The path field
accepts `.` when empty. The agent field is a selector populated from Pair's
shared agent inventory. Tab moves between form fields; Left/Right changes the
agent while that selector is active; Enter submits the form. The initial agent
is the path's last successfully used agent, or the root actor's agent when the
path has no history. Its parameters are that agent's last successful exact
arguments at the path, falling back to Pair's repository default. The form
shows the resolved agent and whether arguments came from path history or the
repository default.

Editing the path marks its preference preview stale. Leaving the path field or
submitting canonicalizes it and re-resolves the preview. Until the operator
changes the agent selector, the agent follows the newly resolved path default;
once Left/Right explicitly changes it, that agent choice remains sticky across
later path edits. Either an agent change or a path re-resolution recomputes the
arguments for the final `{canonical path, selected agent}` pair and immediately
updates the displayed source. Submission performs the same resolution once
more and refuses if the preview no longer matches, so stale asynchronous input
cannot launch with another path's arguments.

Resolution uses the shared operation surface and returns the resolved agent,
exact arguments and source plus an opaque token covering canonical path,
selected agent, relevant preference/default revision, and that exact launch
profile. Each form edit advances a generation. At most one preview request runs,
with at most one pending slot that is always replaced by the newest generation;
an obsolete request is cancelled when the resolver supports cancellation, and
otherwise its result is discarded before the one coalesced latest request runs.
Thus continued typing cannot accumulate work or leave the newest preview
unrequested. Submit carries the current token to the live owner, which either
revalidates and launches that exact accepted resolution or refuses before
effects while preserving the form. The UI never reproduces preference
resolution or performs a check separate from the launch it authorizes
(ARCH-DRY, ARCH-PURE).

Enter while the current generation is still resolving does not start a second
resolution path. It arms one submit for that generation and immediately shows
`resolving`; the accepted latest token continues to submit automatically only
if the generation is unchanged. Any edit or Escape cancels the armed submit,
and resolution failure preserves the form with its diagnostic.

Escape restores the prior stack without updating preferences; Enter returns to
the refreshed root list with the new thread selected only after start succeeds.
Ctrl-Space is a no-op while any text input is already active. A failed start
keeps the path, selected agent, originating menu stack, filters, and selections
intact and reports through the notice feed. Only successful incarnation
registration updates #149's path preference.

The current selection uses reverse video plus `▸`, so it remains visible without
color. A child menu renders to the right of its parent row when space permits
and below the parent list in a narrow terminal. Live threads render normally;
parked age adds a relative text label and one of three progressively dimmer ANSI
grays: less than 24 hours, 1–7 days, and more than 7 days. Terminals without
256-color support retain the state and age text, so gray is never the sole cue.

The menu transition reducer, filtering, age-band choice, and wide/narrow layout
decision are pure. Rendering and operation dispatch are thin injected edges.
Thread summaries, resolution, and operations come from their existing shared
sources rather than being restated in the terminal package (ARCH-PURE,
ARCH-PURPOSE).

Filtering, navigation, and rendering use only the current in-memory inventory;
ordinary keystrokes never read the thread store or resolve references. An
inventory refresh runs asynchronously with at most one in flight. Failure
preserves the complete last-good inventory, menu stack, filters, selections,
and input text and reports a non-blocking notice; before the first successful
read, the menu shows inventory unavailable rather than an empty list. After an
operation succeeds, a failed refresh never causes redispatch: its returned
thread projection updates the affected row when available, otherwise the
last-good view remains visibly refresh-pending until a later refresh succeeds.

**Operating envelope (ARCH-CONSTRAINTS).** Couch's switcher is a keystroke-
critical primary UI. The operator chose 100 rows as the supported workload and
a MacBook Pro M2 Max as the target environment. The 16 ms computation budget is
the domain-informed 60 Hz frame budget; the operator's requirement that the
primary switcher feel immediate
sets 50 ms for opening and 100 ms for lifecycle progress feedback. Opening from
the in-memory inventory produces the first frame within 50 ms; navigation,
filtering, selection, render computation, and applying a completed refresh each
complete within 16 ms for 100 rows. Inventory I/O never gates opening or input.
Start, park, and resume may take longer, but confirmation or progress feedback
appears within 100 ms and lifecycle work never blocks the key-reading loop.
There is at most one inventory refresh, plus one running and one coalesced-latest
start preview; generations discard late results rather than accumulating work.
Memory is linear in the current inventory, and frames retain identities and
text rather than inventory copies. Filter/name input is bounded at 1 KiB and
path/description at 4 KiB; longer persisted display values are clipped safely,
never rewritten. At 40x10 cells the single-column menu remains operable; below
that it asks for a resize rather than emitting a malformed interface.

The target measurement uses a release build on the M2 Max on AC power with Low
Power Mode off and a fixed 100-row inventory fixture. After 20 unrecorded warmup
iterations, it records 200 iterations of open, filter/navigation/render, refresh
apply, and blocked-lifecycle progress-feedback paths and requires each path's
p95 to meet its 50/16/16/100 ms budget in three consecutive runs. One run is an
idle baseline; two run beside a repository-owned deterministic load fixture of
four CPU workers hashing fixed in-memory buffers, representing bounded
development co-tenancy without disk or network variance. The evidence records
hardware and OS version plus every run's p50, p95, and maximum. The performance
fixture renders at 120x40 cells with rows named `thread-000`
through `thread-099`; its committed script measures opening, each byte of
filtering to `thread-09`, twenty alternating Down/Up moves, one
selection-preserving completed refresh, and feedback from one injected blocked
lifecycle operation. Portable automated tests guard the same 100-row fixture's
bounded operations and allocations rather than asserting target-specific wall
time. Full-console tests cover bounds, late completions, and minimum size.

## Revisions

### 2026-08-24 — root exit, stale state, and failed start are total transitions

**Reason:** spec review found three missing edges in the menu state machine and
one inconsistent recency sentence.

**Delta:** root Escape now clears filter then returns to the active thread;
nested Escape pops exactly one level. Dispatch revalidates both durable identity
and current action applicability. Failed start preserves its input and the full
originating stack. Only parked rows show historical age/grayscale; live rows say
`live`. These rules make every reported edge deterministic without adding a
second operation or state source (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-25 — start form selects and remembers the agent

**Reason:** the operator wants Ctrl-Space start to choose among available LLM
harnesses while making the common case require no repeated choice.

**Delta:** start is a path-plus-agent form. It defaults from #149's successful
path history, then the root actor, and reuses parameters only for the same agent
at that path. The form exposes the argument source and preserves all state on
failure or Escape; preference changes occur only after successful registration.

### 2026-08-30 — keep the switcher actionable, suspend-only, and non-blocking

**Reason:** operator review separated Couch's role as a TTY switcher from a
future generalized attach/detach or workspace host, rejected internal recovery
states in the primary UI, and classified the switcher as a keystroke-critical
primary surface. Fresh-context spec review also found stale start-preview and
refresh-failure transitions that lacked a shared authoritative contract.

**Delta:** the delivered ordinary inventory will contain only proven live and
verified parked threads; other durable records remain available to
diagnostics/recovery without leaking implementation states into the switcher.
Enter switches a live TTY or resumes verified park, while the application
contract remains suspend/resume only. Shared opaque start-resolution tokens
bind preview to launch. Last-good snapshots, generation-gated async work,
frame-specific Tab behavior, and explicit 100-row latency/resource bounds make
every hot-path and failure transition deterministic (ARCH-DRY, ARCH-PURE,
ARCH-PURPOSE, ARCH-CONSTRAINTS).

### 2026-08-30 — make evidence and hidden-target transitions explicit

**Reason:** fresh-context review found that the actionable projection did not
name where live-TTY proof comes from, and that a park failure could invalidate
its own action frame by moving the target into a deliberately hidden recovery
state. It also found incomplete preview coalescing, terminal-level Tab coverage,
and measurement provenance.

**Delta:** the pure projection now joins records with exact live-owner
observations, while raw diagnostic inventory remains a separate recovery
surface. Hidden targets deterministically pop invalid frames without deletion.
Start preview has one running request plus one coalesced latest generation.
Latency budgets name their operator/domain basis and M2 Max co-tenanted target;
terminal tests must send both supported Tab encodings (ARCH-DRY, ARCH-PURE,
ARCH-PURPOSE, ARCH-CONSTRAINTS).

### 2026-08-30 — make latency verification replayable

**Reason:** the second fresh-context review found that "ordinary development
load" did not define reproducible evidence for the accepted latency budgets.

**Delta:** target verification now fixes hardware/power conditions, a 100-row
fixture, warmup and sample counts, p95 thresholds across three runs, and a
repository-owned four-worker deterministic co-tenancy load. Pending start
submission also uses the existing generation rather than creating a parallel
resolution path (ARCH-CONSTRAINTS, ARCH-PURE).

### 2026-08-30 — reconcile retained frames before restoration

**Reason:** the third fresh-context review found that unconditional input/stack
restoration conflicted with hiding a captured thread that becomes
non-actionable during async work.

**Delta:** projection reconciliation now precedes restoration. Invalid
thread-bound frames and drafts are discarded with notice and cannot dispatch;
the global start form survives with a reconciled origin stack. The target
benchmark also pins terminal dimensions, row identities, and scripted events
so its evidence is mechanically replayable (ARCH-PURE, ARCH-PURPOSE,
ARCH-CONSTRAINTS).

### 2026-08-30 — split implementation at three genuine review boundaries

**Reason:** implementation planning exposed three independently testable risk
surfaces: lifecycle/start authority, pure menu behavior, and terminal/async
integration.

**Delta:** the issue plan now marks M1/M2/M3 boundaries matching those surfaces,
so each planned `sdlc milestone-close` has a real issue row and fresh-context
review rather than treating the whole multi-file change as one atomic pass.

### 2026-08-30 — complete the M2 behavioral classes

**Reason:** the M2 boundary review found that the pure reducer and renderer
covered the main transitions but under-delivered seven enumerable behavior
classes: shared operation projection, accepted-preview authority, operation
origin correlation, frame applicability, list filtering, hierarchical
geometry, and hidden-target diagnostics.

**Delta:** M2 verification now covers optional versus explicit start-agent
semantics and accepted resolution provenance; every operation origin including
root resume; frame validity independently from filtered selection; filtering
for every list-frame kind; child geometry anchored beside the selected parent
row or below the parent list; one grant per accepted generation; and preserved
target label plus diagnostic location when a durable thread leaves the
actionable projection (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-CONSTRAINTS).

### 2026-08-30 — correlate outcomes without result-generated identity

**Reason:** the second M2 boundary review found that start failure correlation
required the created address that only a successful start can produce, and that
switch dispatch committed bell clearing before focus succeeded.

**Delta:** operation correlation enumerates every declared operation across
success/failure and present/missing result addresses: existing-thread
operations require their captured request address, while start failures may
complete without a created address and successful starts require one. UI state
changes that assert an operation succeeded, including bell clearing, commit
only on a correlated successful completion (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-30 — prove input reachability and protect row semantics

**Reason:** the third M2 boundary review traced reducer and renderer behavior
through their real seams and found horizontal agent selection unreachable from
terminal input, long rows able to erase state/age/bell cues at 40 columns, and
the bell outcome test selecting the active rather than an inactive live row.

**Delta:** every reducer key must be reachable through all terminal modes Pair
accepts, with legacy CSI and application-mode SS3 horizontal arrows tested at
every read split and then driven through the reducer. Root rows reserve a
protected state/age/notification suffix and clip only variable label/path text
within the remaining width. Bell outcome tests use a distinct inactive live
target (ARCH-PURPOSE, ARCH-CONSTRAINTS).

### 2026-08-30 — scope preview identity across form lifetimes

**Reason:** the fourth M2 boundary review found that closing and reopening the
start form reset its generation to one, allowing the scheduler and reducer to
mistake an old form's pending work and completion for the new form's authority.
It also found that the plan's Core concepts tables described final M3 changes
as already present at M2.

**Delta:** preview identities are monotonic for the entire `MenuState`, so edits
and new form lifetimes share one nonzero sequence; the reducer-plus-scheduler
Escape/reopen trace rejects the old token before admitting the new request. All
Core concepts and integration rows distinguish planned change, delivery
milestone, and current M2 status (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-30 — correlate results to dispatch attempts, not targets

**Reason:** the sixth M2 boundary review found that operation/address pairs
identify targets but not attempts: a delayed duplicate from completed attempt A
could retire a newer identical attempt B.

**Delta:** every menu-lifetime dispatch receives one monotonic nonzero attempt
identity carried by `MenuEffect`, `MenuOperationOrigin`, and `MenuEvent`; only
an exact attempt may accept inventory, clear in-flight state, or apply an
outcome. One exhaustive stale-A-after-B table covers switch, resume, park,
name, describe, and start across success/failure result-address shapes, with
fail-safe refusal on identity exhaustion (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-30 — distinguish frame instances from structural positions

**Reason:** the seventh M2 boundary review found that frame kind/depth can alias
a newly reopened confirmation or draft after the originating frame is removed,
letting the old completion mutate the replacement.

**Delta:** every frame receives one monotonic menu-lifetime instance identity;
operation origins capture it alongside attempt/target identity. Frame-local
restoration requires the exact instance, while global successful
park/resume/start restoration remains explicit. The operation × outcome ×
replaced-frame table covers all six operations and fail-safe frame-identity
exhaustion (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-30 — transform origin stacks without owning later overlays

**Reason:** the eighth M2 boundary review found that exact origin identity still
did not preserve an unrelated global start overlay legally opened after
dispatch; stack slicing removed both the owned origin and the later overlay.

**Delta:** completion restoration transforms only its captured originating
stack, then reattaches surviving global start overlays by instance identity
unless target-invalid reconciliation has already removed them. A real-navigation
operation × outcome sweep covers switch, resume, park, name, describe, and
start with post-dispatch overlay creation (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-31 — render one left-anchored surface with breadcrumb ancestry

**Reason:** live smoke showed that rendering the global `start thread` form as
a right-hand child falsely presented it as subordinate to the selected thread,
while full ancestor columns consumed most of the primary switcher surface. The
operator selected the breadcrumb/single-surface treatment after comparing it
with collapsed ancestor rails and dimmed ancestor columns.

**Delta:** `threads` and `start thread` are independent level-zero surfaces,
both anchored at the terminal's left edge. A nested action, confirmation, or
text surface replaces its parent and renders a subdued breadcrumb such as
`threads › <thread> › actions`; it never retains a full parent column. Left and
Escape return one hierarchy level, while Right follows the same forward path
as Tab/Enter for the current list. The reducer remains the hierarchy authority
and the renderer derives breadcrumb labels from its frame stack
(`ARCH-DRY`, `ARCH-PURE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — make single-surface rendering and horizontal keys total

**Reason:** fresh-context review found that the approved visual revision did
not explicitly supersede the old wide/right-of-parent and narrow/below-parent
contract, and its generic Left/Right language conflicted with the start form's
agent selector. It also left breadcrumb components and reconciliation order
implicit.

**Delta:** this revision supersedes every earlier requirement for simultaneous
parent/child bodies, child rectangles, `MenuLayoutWide`, or `MenuLayoutNarrow`.
At every supported size the renderer emits exactly one active body at column
zero; no ancestor body or reserved ancestor column remains. Terminal width now
changes only clipping and viewport capacity. At 40x10 the current surface stays
operable; below 40x10 only the resize request renders.

Horizontal hierarchy keys are frame-specific. On `threads`, Right is exactly
Tab and opens actions; Enter still switches/resumes; Left is exactly root Escape,
including clearing a non-empty filter before returning to the active thread. In
action and confirmation lists, Right is Enter on the selected row, Tab remains
a quiet no-op, and Left is Escape. In rename/describe, Left cancels like Escape,
Right and Tab are no-ops, and Enter submits. `start thread` retains its existing
form contract rather than hierarchy aliases: Tab changes fields, Left/Right
change the agent only while the agent field is active, Enter submits, and Escape
restores the reconciled originating stack. Each alias inherits the corresponding
no-selection behavior.

Breadcrumbs are a pure projection of the reconciled thread-bound visible stack.
Root and global start render only their own level-zero titles, `threads` and
`start thread`; a start form never displays its retained origin as ancestry.
Thread-bound components are `threads`, the current actionable thread display
label, `actions`, and the active leaf label when present: `park`, `rename`, or
`describe`. Reconciliation precedes breadcrumb derivation, so an invalidated or
discarded target never remains visible. The breadcrumb is sanitized, clipped to
one terminal row without wrapping, and does not reduce the active body below its
operable viewport (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

**Superseding acceptance:** at both 120x40 and 40x10, root, global start opened
from root/action/confirmation, actions, park confirmation, rename, and describe
render one left-anchored active body with exact breadcrumb components, visible
selection/form focus, and no parent body; below 40x10 only the resize request
renders. Pure reconciliation and raw-host-loop tables cover the complete
frame-specific Left/Right/Tab/Enter matrix through CSI/SS3 arrows and HT/CSI-u
Tab. Invalidating or renaming a nested target, including beneath an open global
start, removes stale ancestry while preserving only the reconciled origin that
Escape may restore. This acceptance supersedes the Done-when bullet requiring
wide and narrow parent/child layouts.

### 2026-08-31 — include direct leave confirmation in breadcrumb projection

**Reason:** follow-up review found that the exhaustive surface list omitted the
Alt+x leave confirmation, whose reducer stack is root-plus-confirmation rather
than root-actions-confirmation.

**Delta:** thread-bound leaf labels include `park`, `rename`, `describe`, and
`leave couch`. Alt+x leave renders
`threads › <current actionable thread> › leave couch` directly from its
root-plus-confirmation stack and never invents an `actions` component. The
single-surface renderer and horizontal-key acceptance includes this leave
confirmation at 120x40, 40x10, and below minimum: Left/Escape backs out,
Right=Enter on the selected row, Tab is a no-op, and Ctrl-Space opens a
level-zero global start whose Escape restores the reconciled leave origin
(`ARCH-DRY`, `ARCH-PURPOSE`).

### 2026-08-31 — expose local messages, text cursors, and resume landing

**Reason:** operator smoke found that start failures were silent on non-root
surfaces, editable fields did not own the terminal cursor, and a parked-thread
resume could attach without visibly landing—or be refused by the exited pane
whose queued exit had not yet retired from Console routing. The operator chose
a message banner immediately below the breadcrumb over a fixed footer or inline
message.

**Delta:** every switcher surface renders the same optional one-line local
message banner immediately below its breadcrumb and before its controls. The
banner is omitted when empty; appearance may shift the controls down by one row
as explicitly chosen. Operation, preview, inventory, reconciliation, and
validation failures set an error-level message rendered as `error: <text>`;
progress/informational notices use the same location without the error prefix.
The banner is sanitized and clipped without wrapping, remains visible at 40x10,
and never writes into the separate agent-pane notification/status row. Agent-pane
error routing remains a later slice and will reuse that existing notification
area rather than adding another region.

Pure menu rendering now returns both body text and one optional bounded cursor
intent. The cursor is visible at the end of the active editable value for the
start path field, rename/describe input, and any visible non-empty root/action/
confirmation typeahead filter. It is hidden for selections, confirmations, the
start agent selector, empty implicit filters, unavailable/resize screens, and
all other non-text surfaces. Cursor coordinates derive from the final rendered
and clipped lines after the optional banner, using terminal cell width for
Unicode; Console only translates the intent into host move/show/hide sequences.
Leaving the switcher restores the selected agent pane's replayed cursor state,
and teardown still unconditionally shows the shell cursor (`ARCH-DRY`,
`ARCH-PURE`, `ARCH-CONSTRAINTS`).

A successful parked-thread resume is one presentation transaction: resume the
exact durable address, attach the returned exact terminal handle, then clear/
replay and focus that handle. New start remains intentionally different and
returns to the switcher with its row selected. A terminal pane whose child is
already exited but whose queued exit event has not yet retired may not block a
same-thread attach and is never a switch candidate; a still-live duplicate
continues to refuse. Exact handle identity, not thread-address map iteration,
selects the resumed pane. Attach failure still aborts only the exact newly
started handle before the reducer receives failure (`ARCH-PURE`, `ARCH-PURPOSE`).

**Acceptance:** raw-host tests cover every frame with and without messages,
including failed `../pair` start, and assert that the banner survives repaint at
120x40 and 40x10 without leaking into the status row. Pure render tests cover
cursor row/column after banner insertion, clipping, and wide Unicode for every
editable class plus hidden-cursor non-text classes. Stateful Console tests cover
resume with a normal retired pane and with a done-but-not-yet-retired pane,
requiring exact attach-before-replay, focused resumed handle, no abort on success,
and exact abort with preserved switcher/error banner on failure.

## Done when

- Enter switches to/resumes the selected work thread; thread-list Tab enters
  its action menu;
  Escape restores the exact parent filter and selection.
- Only proven live and exact verified-parked threads appear in the ordinary
  switcher; undecodable records are never mislabeled or discarded.
- Park requires explicit confirmation and preserves the durable work thread.
- Rename and description operate on the selected durable thread, survive a
  harness restart, and expose operation failures without losing typed text.
- Multiple threads at one path are distinct rows; parked threads remain listed
  with textual state/age and progressively dimmer age bands.
- Ctrl-Space opens start from every list level without destroying the menu stack.
- The start form selects any declared Pair agent, defaults to the path's last
  successful agent or the root actor, and uses that agent's path arguments or
  repository default without crossing arguments between agents.
- Start submits the exact current shared resolution token; out-of-order preview
  completions and preference changes before submit cannot launch stale inputs.
- Enter during resolution arms at most one generation-bound submit; editing or
  cancelling prevents the late token from launching.
- Stale targets and zero-match lists never dispatch against a different row.
- A target that becomes non-actionable invalidates its nested frames and returns
  to the preserved root without deleting or mislabeling the durable record.
- Refresh and async completion reconcile every retained/originating stack before
  restoring it; a global start form survives, while invalid target-bound drafts
  are explicitly discarded and can never reach dispatch.
- Inventory failure preserves the last-good UI, and post-success refresh failure
  never repeats a mutation.
- Legacy HT (`0x09`) and Kitty CSI-u Tab (`CSI 9u`) drive the same frame-specific
  Tab behavior through the real console decoder.
- Opening and every ordinary keystroke meet the 100-row interaction budget
  without synchronous I/O or unbounded asynchronous work.
- Three target-machine measurement runs meet the specified p95 budgets under
  both baseline and deterministic four-worker co-tenancy fixtures.
- Wide and narrow terminal layouts are readable, with selection visible without
  relying on color.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.
The thorough approved plan earns the ×0.2 design discount and 15% design buffer;
implementation values are already scaled to 40% per v3.1. Existing Go terminal,
runner, and fake seams cover the stack, so the library-availability check found
no novel greenfield dependency to discount further. The primitives map to M1's
inventory/grant/admission authority, M2's pure menu/renderer/decoder/scheduler,
M3's Console/runner/performance integration, two expected operator UX rounds,
atlas work, and the three real milestone reviews.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.20 impl=0.24
item: greenfield-go-module design=0.30 impl=0.32
item: cross-cutting-refactor design=0.10 impl=0.20
item: tui-screen design=0.40 impl=0.40
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.03 impl=0.12
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.40 impl=0.40
item: cross-cutting-refactor design=0.20 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.10 impl=0.08
item: milestone-review design=0.04 impl=0.12
item: milestone-review design=0.04 impl=0.12
item: milestone-review design=0.04 impl=0.12
item: ux-rename-iteration design=0.40 impl=0.08
item: ux-rename-iteration design=0.40 impl=0.08
design-buffer: 0.15
total: 7.60
```

## Plan

- [x] M1 — Build and integrate authoritative actionable inventory plus bounded,
      token-bound start preparation/admission contracts.
- [x] M2 — Build the pure hierarchical menu reducer, renderer, key semantics,
      reconciliation, and bounded preview scheduler.
- [ ] M3 — Integrate asynchronous inventory/actions/forms with Console, add
      full-console/performance evidence, and update the Couch atlas.

## Log

### 2026-08-24

Split from `#146` after operator smoke rejected the `:` command namespace and
design clarified that historical work threads—not live actors or worktrees—are
the durable menu rows. Depends on `#149` so the menu never invents a second
identity or metadata store.

### 2026-08-30 — session summary
- 2026-08-30: closed M1 — TDD staging contract went red before the five-surface sweep and green after it; go test -p 20 couchcore/couchcmd/couchtty/artifactpath, focused race tests, and git diff --check pass; review verdict: SHIP

Kept #151 narrowly focused on the hierarchical TTY switcher. Couch owns the
presentation switch among live PTYs while Pair exposes only durable
suspend/resume; generalized attach/detach and workspace composition are
deferred. Operator-approved spec deltas hide non-actionable recovery states,
bind start preview to execution, preserve last-good UI across refresh failure,
define Tab per frame, and require a non-blocking 100-row hot path.

### 2026-08-30 — plan-quality gate round 1

`sdlc change-code` blocked before implementation with `PQ-1` (the plan copied
test-case/procedural matrices instead of function-level strategies) and `PQ-2`
(the new runner cancellation fake lacked explicit real-runner conformance).
Revised the plan across all chunks and added an always-run shared cancellation
contract for FakeRunner, ExecRunner, and PtyRunner (ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-30 — M1 actionable inventory

Added the fail-closed actionable projection and its one-snapshot Couch wrapper.
Focused projection, wrapper, and raw-inventory tests pass with `-p 20`. The
broader `couchcore` run remains blocked by the known #152 plan-contract test,
which still opens the completed plan from `workshop/plans/`.

### 2026-08-30 — M1 bounded start grants

Added a mutex-owned, in-memory one-attempt grant table. The focused suite and
race detector pass with `-p 20`, covering entropy/collision limits, defensive
resolution ownership, atomic claim, pre-claim expiry, capacity, finish, and
owner-restart invalidation (ARCH-PURE, ARCH-CONSTRAINTS).

### 2026-08-30 — M1 prepared start authority

Factored direct and token-bound start through one pure, length-delimited
resolution fingerprint. Prepare has no allocation/fork effects; submit claims
once, revalidates policy/preference/default evidence, and passes accepted
candidate policy directly into admission. Focused legacy/new tests and the race
detector pass with `-p 20`; the full package reaches only the already-recorded
#152 artifact-path contract failure (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

### 2026-08-30 — M1 shared operations and CLI

Declared agent-facing `prepare-start` authority and implicit token-bound
`start`, then routed public `couch start` through both while keeping singleton
ownership and Console/PTy choice separate. The archived-plan contract failure
was fixed in a separate side-quest commit by sharing active/history artifact
lookup. Unfiltered `couchcore` and `couchcmd` tests pass with `-p 20`.

### 2026-08-30 — M1 boundary review round 1

The fresh review returned REWORK with four open families: the required-token
schema had not migrated the transitional Console consumer (`BR-1`), actionable
projection accepted structurally invalid records (`BR-2`), the exhaustive
production-source inventory omitted all three new files (`BR-3`), and the atlas
described M3 adoption as already current (`BR-4`). Swept every production start
dispatcher, validated records before projection with valid positive fixtures,
classified the full new-source set, corrected atlas staging prose, and
revised the later context plan to account for M1-delivered operation context.

### 2026-08-30 — M1 boundary review round 2

The second review mutation-verified `BR-1` through `BR-3`, but kept `BR-4`
open because the project milestone still described the future M3 consumer as
current. The transitional flat panel remains wired to raw `ThreadInventory` until M3; the atlas, project, issue, plan revision, and README now state that
same staged boundary, and one source/document contract makes M3 update them
together (ARCH-PURPOSE).

### 2026-08-30 — M2 total menu reducer

Extracted one store-free exact-over-fuzzy thread matcher for both operation
resolution and in-memory filtering, then built the immutable-by-copy menu
reducer through root, action, confirmation, bounded text/start, bell, refresh,
and operation-result traces. Generated key traces enforce structural/text
bounds and declared-operation-only effects; reordered refreshes preserve exact
identity, hidden targets discard descendants, and completions emit no effects
(ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-CONSTRAINTS).

### 2026-08-30 — M2 bounded menu renderer

Added pure contained geometry for single, wide, narrow, and below-minimum
layouts plus viewport rendering that keeps the selected row visible. Renderer
tests pin terminal-column clipping, control-sequence stripping, live versus
parked age text, three age bands, optional grayscale, 100-row viewports, and a
wide two-frame boundary whose separator previously exceeded the declared width
(ARCH-PURE, ARCH-CONSTRAINTS).

### 2026-08-30 — M2 semantic Tab framing

The shared terminal decoder now maps legacy HT and unmodified Kitty CSI-u Tab
to one semantic key across every input split, drops modified CSI-u Tab, and
reserves the zero key kind as unknown. The decoder fuzz corpus carries all
three forms so framing changes cannot leak escape bytes into filter text
(ARCH-DRY, ARCH-PURE).

### 2026-08-30 — M2 bounded preview scheduling

Added a pure one-running/one-latest preview scheduler: a newer generation
requests cancellation once, replaces only the pending slot, and cannot retire
running work until its matching terminal outcome arrives. Start-form reducer
events bind preview acceptance and one armed submit to the exact nonzero
generation; edits/Escape cancel submit authority, stale or duplicate results
cannot dispatch, and failure preserves form input (ARCH-PURE,
ARCH-CONSTRAINTS).

### 2026-08-30 — M2 pure-menu map ready for boundary

Mapped the shared matcher, total reducer/reconciliation order, bounded
wide/narrow renderer, semantic Tab framing, and one-running/one-latest preview
scheduler. The atlas explicitly keeps this core inert: the current Console
continues to present the flat compatibility panel until M3 supplies exact pane
observations and bounded workers (ARCH-PURPOSE).

### 2026-08-30 — M2 boundary review round 1

The fresh review returned REWORK with seven blocking behavior families. The
test-first sweep now preserves optional start-agent semantics and accepted
agent/argv provenance, maps operation IDs to UI labels, reuses one accepted
preview generation, correlates every completion to its captured origin before
accepting returned inventory, retains applicable zero-match frames, filters
confirmation rows by their displayed label, anchors child geometry to parent
rows/lists, and carries the prior human label plus composite diagnostic address
through hidden-target transitions. Focused regressions went red at every named
gap and the complete `couchtty` package is green (`ARCH-DRY`, `ARCH-PURE`,
`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-30 — M2 boundary review round 2

The review mutation-verified all seven prior fixes, then exposed two remaining
outcome classes: start failure could not correlate without a success-produced
address, and switch dispatch cleared its bell before focus succeeded. The
operation/outcome/address table and switch success/failure regressions went red
against the reviewed code and green after correlation stopped requiring
result-generated identity and bell clearing moved to successful completion
(`ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-31 — smoke: preserve one short completed round and classify park exits

Operator smoke produced a complete Claude `hello` → assistant round, but the
parked thread disappeared. Exact artifacts showed the submitted Pair log and
the matching scanner-authorized Claude transcript; the ledger remained launch-
only because `QualifyTurnSequence` still required a 32-byte/five-word single
turn. Changed the single-turn rule to accept one globally unique completed
round regardless of content length, while retaining the stronger threshold and
paired-turn matcher once multiple turns exist. Pure matching plus watcher-level
tests now cover the short round (`ARCH-PURPOSE`).

The same smoke left `[actor] exited (0)` stuck after a successful park because
Console classified every child exit as an unexpected control notice. Console
now correlates the immutable in-flight Park origin and, when completion wins
the channel race, retains the exact expected child handle until its exit is
consumed. Both event orders suppress only the expected shutdown; ordinary exits
remain notices. Focused race tests and the full sessioninventory, sessionwatch,
couchcore, and couchcmd race suites pass at `-p 20`. The full couchtty suite
still reaches the pre-existing flat-panel compatibility failures (including its
nil legacy `PanelModel` assertion) after the hierarchical UI migration
(`ARCH-PURE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — correction: Claude queued input was the two-round blocker

A second operator smoke falsified the short-turn-only diagnosis above. The
new Brain thread had two submitted Pair turns and complete Claude work, yet its
ledger was still launch-only. Replaying those exact artifacts through the
production scanner showed that Claude encodes input submitted while busy as
`queue-operation {operation:"enqueue", content:...}` and later emits a matching
`remove`; the normalizer ignored both. Pair therefore saw two turns while the
native matcher saw only the first, preventing both the short-single and paired-
turn paths. The normalizer now emits one operator event for a non-empty
`enqueue`, ignores `remove`, and fails closed on unknown queue operations. A
stateful watcher regression reproduces the macOS no-generation-token append,
two Pair turns, queued Claude operator, progress, and exact proof-bearing ledger
binding. The real saved smoke artifacts replay to one qualified round with this
change. Full sessioninventory/sessionwatch race suites pass at `-p 20`; rebuilt
`bin/pair` and `bin/couch` carry the fix (`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — smoke: established proof survived natural transcript growth

The next smoke established a proof-bearing binding correctly, then still hid
the row after park. The binding proof named the exact Claude root at 47 KiB;
normal agent work grew that same stable file to 531 KiB. On the operator's APFS
volume `st_gen` is unavailable, so exact-query advancement rejected the append
and downgraded the already established root. This made normal transcript growth
act like identity revocation.

For a proof-named target with the same stable file identity, no truncation, no
available-generation disagreement, and monotonic growth, exact query now falls
back once to full validation of that one file—never the corpus—checks the same
agent/root/schema, and publishes the newer authorized snapshot. Validated
publications with both generation tokens absent may advance monotonically;
stale/lower cursors, crossed parser cursors, disputes, stable-ID replacement,
truncation, and generation mismatch remain fail-closed. The second query reuses
the published snapshot with zero body reads. The exact failed owner
`couch-ab578f9435b532ba` was revalidated to Claude root `67761ab3…`, restoring
its actionable resume authority. Full sessioninventory/sessionwatch race suites
pass at `-p 20`; binaries rebuilt (`ARCH-PURE`, `ARCH-PURPOSE`,
`ARCH-CONSTRAINTS`).

### 2026-08-30 — M2 boundary review round 3

The review confirmed BR-12 and the bell implementation, but required the bell
test to use an actually inactive live row and found two production-seam gaps:
horizontal agent selection had no CSI/SS3 decoder path, and long rows could
clip required lifecycle/bell cues. Focused seam and minimum-width regressions
went red, then green after shared four-direction arrow decoding and protected
semantic suffix layout (`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-30 — M2 boundary review round 4

The review disposed the three production-seam findings, then found preview
identity reused across start-form lifetimes and final M3 entities mislabeled as
current in the plan. The composed reducer/scheduler Escape/reopen trace went
red with two generation-one forms and green after menu-lifetime monotonic
identity; both Core concepts tables now state delivery and current M2 status
for every row (`ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-30 — M2 boundary review round 5

The review mutation-verified preview lifetime identity and confirmed every plan
row matched the tree, but kept BR-17 open because that staging truth was not
executable. A new contract enumerates all 16 Core concepts/integration rows,
checks exact delivery/current status against present and absent files, and
proves an in-memory future-M3-as-current mutation fails (`ARCH-PURPOSE`).

### 2026-08-30 — M2 boundary review round 6

The review disposed BR-17, then found target-level correlation unable to reject
a delayed duplicate after an identical successor dispatch. The exhaustive
six-operation stale-A-after-B regression went red without attempt identity and
green after each effect/origin/result carried one monotonic menu-lifetime
attempt; exhaustion refuses work (`ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-30 — M2 boundary review round 7

The review disposed BR-18, then showed that kind/depth could alias a reopened
frame at the same structural position. The six-operation replacement-frame
table went red for park failure and name/describe success, then green after
origins captured monotonic frame-instance identity; global successful
park/resume/start behavior stays explicit and identity exhaustion refuses
navigation (`ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-31 — M2 boundary review round 8
- 2026-08-31: closed M2 — Focused operation/outcome overlay regression passes; bounded couchcore/couchcmd/couchtty/artifactpath suite passes with -p 20; targeted couchcore+couchtty race suite passes with -p 20; issue and project validation pass; git diff --check passes. BR-20 now preserves unrelated later global start overlays by frame-instance identity.; review verdict: SHIP

The review disposed BR-19, then found that successful global restoration and
park-failure restoration still discarded a global start overlay opened after
dispatch. The six-operation success/failure navigation sweep went red for
resume success, park failure/success, and start success, then green after
restoration became a captured-prefix transform that retains unrelated later
start overlays by frame instance (`ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-31 — M3 Task 10 bounded actionable refresh

Added the pure one-running/one-dirty refresh schedule and a single-flight
Console controller over a context-bearing actionable-inventory provider.
Console snapshots exact hosted PID/start identities under its mutex, performs
store projection outside it, and applies terminal results only through the
menu reducer. Barrier tests prove open/filter repaint do not wait for blocked
inventory I/O and teardown cancels and joins it within 250 ms; failures retain
last-good state and initial unavailable remains distinct from successful empty
(`ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — M3 Task 11 typed queue and lifecycle context slice

The existing bounded sequential operation queue now returns the typed result
and exact request key once, restores overload-refused admission for retry, and
coalesces duplicate progress without redispatch across all menu/leave
operations. Owner dispatch carries its supplied context through park normal,
retry, recover, abandon, leave, and resume admission; direct `Resume` remains
an explicit background convenience wrapper. Exhaustive canceled-context and
targeted race tests pin the full branch set (`ARCH-PURE`, `ARCH-PURPOSE`,
`ARCH-CONSTRAINTS`).

### 2026-08-31 — M3 Task 11 blocked-launch cancellation

The blocked runner seam now accepts the operation context, refuses an already
canceled launch before helper effects, and carries that context through both
new starts and verified-park resumes. Tracked launch checks cancellation on
both sides of acknowledgement: pre-ack helpers are canceled/reaped before
rollback, while post-ack targets use the existing exact quiesce/reconcile
authority. Registration deadlines derive from the caller context. One shared
Fake/Exec/Pty trace plus deterministic new-start and resume tests cover
`blocked → canceled → reaped` and `acknowledged → canceled → reaped`, including
verified-park restoration (`ARCH-MOCK`, `ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — M3 Task 11 Console action lifetime

Every synchronous or queued Console action now receives a child of the one
Console lifetime context, including the prepare/start/attach chain. Stop
cancels that parent before teardown joins the bounded action worker. Blocking
lifecycle and metadata dispatch tests prove cancellation is observed, the
dispatcher runs exactly once, and `Console.Run` returns within 250 ms; targeted
race and full Couch TTY/command suites pass (`ARCH-PURPOSE`,
`ARCH-CONSTRAINTS`).

### 2026-08-31 — M3 Task 11 start-preview controller

Console now composes reducer preview effects with the pure one-running/one-
latest-pending scheduler. Each `prepare-start` runs under a Console child
context; replacement cancels by exact generation, terminal completion alone
retires the running slot, and typed `PreparedStart` values re-enter
`ReduceMenu`. An armed Enter dispatches the accepted token exactly once through
the shared operation queue. Controller tests cover replacement, Stop
cancellation/join, stale-safe result application, and pending submit; targeted
race and full TTY/command suites pass (`ARCH-PURE`, `ARCH-PURPOSE`,
`ARCH-CONSTRAINTS`).

### 2026-08-31 — M3 Task 11 exact started-actor abort

`Couch.AbortStarted` now owns the failure half after a successful start is
transferred but terminal attachment does not commit. It refuses nil,
unregistered, or record/handle identity mismatches without effects; an exact
match reuses the existing post-ack handle and Pair-session quiesce/reconcile
authority, then removes and persists the transitional registry record. Tests
prove mismatches cannot kill or quiesce another actor and exact cleanup leaves
the durable incarnation fail-closed. Targeted race and full core suites pass
(`ARCH-DRY`, `ARCH-PURPOSE`).

### 2026-08-31 — M3 Task 11 transactional terminal attach

Terminal attachment now commits routing state and its exit watcher as one
mutex-owned transaction. It refuses stopped Consoles, exited children,
duplicate thread/handle identities, and record/handle process mismatches
without changing active/root/order state. Teardown crosses the same mutex
before its final worker wait, closing the Add/Wait race. The composition root
now invokes `AbortStarted` whenever owner-local attach refuses, so exact child,
Pair session, durable state, and registry cleanup finish before failure returns.
Concurrent Stop, dead-terminal rollback, terminal restoration, and wired abort
tests pass under the race detector and full TTY/command suites
(`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — M3 Task 11 semantic operation completion

Menu-originated operations now carry the reducer's exact operation, attempt,
address, and frame identity through the sequential queue. Typed start/resume
results complete terminal attach first; attach refusal completes exact abort
cleanup first; only then does one correlated terminal event re-enter
`ReduceMenu`. Stale or compatibility completions cannot impersonate a newer
attempt, while legacy Alt+x/flat-panel work retains its existing completion
path until Task 12 removes it. Semantic failure and attach-before-success tests,
the focused restoration matrix, targeted race suite, and full core/TTY/command
packages pass (`ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-31 — make local messages, cursor ownership, and resume completion total

Fresh review found that the first live-smoke revision named the three missing
behaviors without making their state spaces exhaustive. This revision is the
normative Task 12 contract and supersedes the earlier summary where they differ.

#### Switcher message contract

`MenuNotice` is one typed value (`level`, `text`), owned by `MenuState` and
rendered by the one active switcher surface. The renderer sanitizes it to one
terminal row. Errors render as `error: <text>`; progress/info renders without a
prefix. A handled semantic event clears the prior notice unless the event
installs one below; a stale completion rejected by the reducer preserves it.

| Producer | Level | Exact text or pattern | Surface after reconciliation |
|---|---|---|---|
| Console initialization; refresh begins before first inventory | info | `thread inventory unavailable` | current surface |
| inventory provider failure | error | `thread inventory unavailable: <provider error>` | current surface |
| Enter/Tab with no root row; Enter with no action/confirmation row | error | `no selection` | originating surface |
| root Escape without a live active thread | error | `no live thread can receive focus` | root |
| invalid park-hotkey operation | error | `park action is unavailable` | current surface |
| park-hotkey target absent or not live | error | `active thread is no longer actionable` | root after reconciliation |
| selected target disappears | error | `thread <label> (<scope>/<tag>) is no longer actionable` | root, or retained level-zero start |
| stale action/confirmation target or action | error | `thread action is no longer applicable` | root, or retained level-zero start |
| stale rename/describe target | error | `thread input is no longer applicable` | root, or retained level-zero start |
| invalid frame kind | error | `menu frame is no longer valid` | root, or retained level-zero start |
| start submit begins preview resolution | progress | `resolving` | start |
| preview sequence cannot advance | error | `start preview identity exhausted` | start |
| preview dispatcher/provider failure | error | `<returned error>`, fallback `start preview failed` | start |
| operation sequence cannot advance | error | `operation attempt identity exhausted` | originating surface |
| frame sequence cannot advance | error | `menu frame identity exhausted` | originating surface |
| queue refusal, dispatcher failure, attach refusal, abort failure, or correlated operation failure | error | `<returned error>`, fallback `<operation> failed` | restored/reconciled origin |

The optional banner is immediately below the breadcrumb and before the blank
separator and controls. It is absent, not blank, when empty, so later rows move
down exactly one only while it is visible. Root, actions, park, direct leave,
rename, describe, and start opened from root/actions/confirmation share this
placement. At 120×40 and 40×10 it remains bounded with one active body and no
ancestor body. Below 40×10 the renderer emits only the resize request, hides
cursor and banner, retains the typed notice, and reveals it after a supported
resize. Switcher notices never enter the actor-pane feed; later actor-pane
errors reuse its existing notification row (`ARCH-DRY`, `ARCH-PURPOSE`).

#### Cursor contract

The pure render result contains body bytes plus optional 1-based terminal-cell
cursor intent. Intent refers to the final clipped body after banner insertion,
never bytes/code points. A clipped logical end clamps to the final rendered
field cell. Width reuses the terminal-width authority: ASCII=1, combining=0,
double-width=2.

| Active surface/edit state | Empty | Non-empty or clipped | Banner effect |
|---|---|---|---|
| root filter | hidden | after rendered `filter: …` | row +1 |
| actions filter | hidden | after rendered `filter: …` | row +1 |
| park confirmation filter | hidden | after rendered `filter: …` | row +1 |
| direct-leave confirmation filter | hidden | after rendered `filter: …` | row +1 |
| rename input | after `> ` | rendered/clamped field end | row +1 |
| describe input | after `> ` | rendered/clamped field end | row +1 |
| start with path selected | after `▸ path  ` | rendered/clamped path end | row +1 |
| start with agent selected | hidden | hidden | none |
| selection-only, unavailable, or below-minimum resize | hidden | hidden | none |

The table applies independently at 120×40 and 40×10; below minimum always
hides. Tests locate the exact rendered field row and assert its cell end for
ASCII, combining acute, and double-width input, clipped and unclipped. Host
order is: hide inherited actor cursor; clear/paint switcher; restore Couch
status; then `MoveTo`+`ShowCursor`, or `HideCursor`. Actor return clears/replays
before transferring cursor ownership. Every teardown emits `ShowCursor`
(`ARCH-PURE`, `ARCH-CONSTRAINTS`).

#### Start/resume completion contract

1. Resume success: typed result → attach exact handle → correlated reducer
   success/clear in-flight → force-switch exact handle → clear/replay →
   status/focus. No abort or switcher repaint.
2. Start success: typed result → attach exact handle → correlated reducer
   success/select returned row → refresh → switcher repaint. No force-switch.
3. Attach failure: refusal → synchronous exact-actor abort → correlated reducer
   failure → captured-origin restoration/reconciliation → local error-banner
   paint. No success, focus, or replay; attach and abort errors remain visible.

Address admission and address switching ignore panes whose child is `Done`; a
live same-address pane refuses before routing/order/watcher/reducer mutation. A
later queued exit for the old done handle removes only that handle and cannot
remove, activate, or redirect the new handle. Ordered fake-host/fake-runner
traces, not final-map assertions alone, enforce this contract
(`ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — make parked mean resumable and animate pending work

Live smoke found two verified shutdown records presented as `parked` even
though neither Pair session had completed one user/agent exchange. Their exact
owner ledgers contained no established native binding, so `resume` correctly
refused them after the switcher had incorrectly advertised the action.

The ordinary switcher now admits a verified-park row only when the existing
`NativeBindingResolver` proves the saved launch agent has one exact established
root with a non-empty native ID. Resolution happens in the already asynchronous
actionable-inventory provider and uses `QuerySession`'s exact ledger plus
proof-named, catalog-backed incremental validation; it never enters the
keystroke/render path or performs a whole-inventory scan. Unbound,
provisional, ambiguous, malformed, or unreadable parked records remain in
`couch list/show` diagnostics but are absent from the two-state switcher. Thus
`live` means exact hosted-process proof and `parked` means exact resume
authority; no third user-facing lifecycle leaks into the menu (`ARCH-DRY`,
`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

Every accepted asynchronous switcher operation immediately installs one local
progress banner: start preview `resolving…`; start `starting thread…`; resume
`resuming <label>…`; park `parking <label>…`; leave `leaving couch…`; name
`renaming <label>…`; describe `saving <label> description…`. The banner begins
before work is enqueued. Its one-cell spinner cycles `◐ ◓ ◑ ◒` every 100 ms
while progress is current, using one Console-owned timer only while needed.
Ticks are pure reducer events and cause no inventory, filesystem, or operation
work. Completion, refusal, Escape/teardown, or replacement with an info/error
notice stops animation; duplicate operation submission remains refused by the
existing exact in-flight identity. Input and navigation stay responsive while
work runs, and below-minimum rendering retains state without painting until the
terminal is supported again (`ARCH-PURE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — totalize resume proof, cancellation, and progress identity

Fresh review found that the preceding revision did not make resolver teardown,
progress replacement, or contradictory resume observations total. This
revision is normative and supersedes it where they differ.

`ParkedResumeObservation` is the immutable tuple `(address, saved agent,
non-empty native ID)`. A structurally verified parked record is actionable only
when its saved launch profile is valid and there is exactly one observation for
its address, whose agent equals that profile and whose native ID is non-empty.
Zero, duplicate, wrong-agent, empty-ID, or otherwise contradictory observations
omit the row. Live projection continues to use only exact hosted-process proof.
The same raw record remains visible through diagnostic `couch list/show` for
every omitted binding status or resolver failure.

The existing `NativeBindingResolver` becomes context-bearing. The actionable
provider passes its Console lifetime context into every parked-row resolution,
checks cancellation between rows, and stops without publishing a partial
snapshot. Exact query/incremental validation checks that context at each
operation boundary; production local regular-file syscalls themselves are not
interruptible, so no goroutine is detached to simulate cancellation. A
context-aware fake proves a blocked resolver observes Stop and the sole refresh
worker joins within 250 ms (`ARCH-PURE`, `ARCH-CONSTRAINTS`).

Progress owns an exact identity: preview generation or operation attempt.
`MenuNotice.Text` remains `resolving` for preview (preserving the prior typed
contract), while the renderer produces `◐ resolving…`; other typed texts omit
the ellipsis likewise and render as `◐ starting thread…`, `◐ resuming
<label>…`, `◐ parking <label>…`, `◐ leaving couch…`, `◐ renaming <label>…`, or
`◐ saving <label> description…`. Frames cycle `◐ ◓ ◑ ◒` without changing
terminal width.

| Event class | Current progress result |
|---|---|
| exact tick for current generation/attempt | advance one frame |
| stale tick or stale completion | preserve current progress and phase |
| filter, Up/Down, Tab/Right, nested overlay, inventory refresh, resize | preserve |
| preview edit/replacement | replace with the new preview generation at phase zero |
| Escape from a pending preview | cancel that generation and clear |
| Escape/focus loss during a dispatched operation | preserve; operation is not implicitly cancelled |
| exact success | clear, then perform the specified landing/restoration |
| exact refusal/failure | replace with the local error banner |
| explicit info/error for the same current work | replace and stop animation |
| unsupported-size render | retain without painting |
| teardown | discard state and stop/drain the timer |

The Console timer is armed only while the switcher is focused on current
progress. Stop/drain/rearm occurs on the Run owner; a queued tick carries its
old identity and is harmless after replacement. No timer goroutine, provider
call, filesystem call, or operation dispatch occurs on a tick
(`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — define notice precedence during pending work

Fresh review found one remaining cross-product: a different reducer event can
produce a required local notice while progress is current. `MenuNotice`
therefore also carries an optional preview-generation or operation-attempt
owner. There is one banner, never a hidden stack of notices.

| Incoming producer while progress is current | Banner after event | Timer | Later exact success |
|---|---|---|---|
| successful refresh, ordinary navigation, unrelated info | preserve progress | continues | clears only its owned progress |
| refresh-provider failure | replace with refresh error | stops | preserves unrelated error |
| reconciliation/target-disappeared error | replace with reconciliation error | stops | preserves unrelated error |
| validation/no-selection error | replace with validation error | stops | preserves unrelated error |
| frame/preview/operation identity exhaustion | replace with exhaustion error | stops | preserves unrelated error |
| exact failure/refusal for current work | replace with work-owned error | stops | N/A |
| exact success while owned progress remains | clear progress | stops | cleared |
| stale tick/completion/preview result | no change | no change | no change |
| newly accepted replacement preview/work | replace with new owned progress at phase zero | rearmed | clears only the new owned progress |

Errors are never suppressed and progress is not implicitly restored after an
unrelated error. A successful completion removes a notice only when that notice
is the progress owned by the same exact generation/attempt; it cannot erase a
newer or unrelated diagnostic. A matching failure is the latest result of the
requested work and replaces the current banner with its owned error
(`ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-31 — M3 Task 13 performance and operator evidence

The committed 100-row harness now covers open, filter, navigation, pure render,
refresh apply, and first lifecycle feedback. Portable allocation/bound tests
and five `BenchmarkMenu100` runs pass. The optimized target binary was built by
`go1.26.6 darwin/arm64` with SHA-256
`227d5590559606b5e5a9d04a7646dbf17b8db28ea244c713eb5ae764cdbe662d`.

On the Apple M2 Max, 20 warmups plus 200 samples per path passed in one baseline
and two trials beside exactly four joined CPU workers. The worst p95 across the
three trials was: open 69.334µs (50 ms budget), filter 128µs, navigation
192.458µs, pure render 57.334µs, refresh apply 213.75µs (each 16 ms budget),
and first progress feedback 328.959µs (100 ms budget). Boundaries were Console
input-to-repaint return, refresh-result-to-repaint return, and pure render
call-to-ANSI return as applicable (`ARCH-CONSTRAINTS`).

Operator smoke used an isolated `COUCH_STORE_DIR=$(mktemp -d)` store and
verified hierarchical navigation, start progress, local error banners, cursor
placement, clean-store park and exact resume after one and two exchanges,
renamed-thread persistence, Leave Couch parking all actors, and terminal-mode
restoration after exit. The final APFS no-generation-token failure was
reproduced with exact Claude root `67761ab3-d9ee-477d-b01c-d34b452159c1` and
fixed by one exact validated growth read followed by cached zero-content-read
queries. The operator then confirmed the parked row reappeared and separately
verified mouse movement no longer emitted escape bytes after Couch quit.

### 2026-08-31 — M3 boundary-review disposition

The first M3 boundary returned REWORK. The three finding families were swept
as classes: all seven operation results now have a declared projection policy;
mutating successes remain visibly `refresh pending` until a successful
actionable inventory arrives, including after provider failure. The obsolete
`PanelModel`, resolver/summary callbacks, prompt/controller state, source, and
tests are deleted, with executable absence checks. The target performance
protocol now enters through raw input/result channels on a running Console and
ends only on a correlated `FakeHost` frame (`ARCH-DRY`, `ARCH-PURE`,
`ARCH-PURPOSE`, `ARCH-MOCK`, `ARCH-CONSTRAINTS`).

The corrected M2 Max protocol passed 20 warmups and 200 samples for each of six
paths in a baseline and two trials beside exactly four joined CPU workers. The
worst observed p95 was 290.625µs (first feedback); every path remained below
291µs. The boundary for every value is semantic input/result through
`Console.Run` to the matching unique per-sample emitted frame. The optimized
`go1.26.6 darwin/arm64` test binary SHA-256 was
`006ec35437d410a5c0757a1ad17a6db811e05c602ca310a51faf6d560807e97b`.
