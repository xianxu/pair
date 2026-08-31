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

**Delta:** the ordinary inventory now contains only proven live and verified
parked threads; other durable records remain available to diagnostics/recovery
without leaking implementation states into the switcher. Enter switches a live
TTY or resumes verified park, while the application contract remains
suspend/resume only. Shared opaque start-resolution tokens bind preview to
launch. Last-good snapshots, generation-gated async work, frame-specific Tab
behavior, and explicit 100-row latency/resource bounds make every hot-path and
failure transition deterministic (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE,
ARCH-CONSTRAINTS).

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

- [ ] M1 — Build and integrate authoritative actionable inventory plus bounded,
      token-bound start preparation/admission contracts.
- [ ] M2 — Build the pure hierarchical menu reducer, renderer, key semantics,
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
