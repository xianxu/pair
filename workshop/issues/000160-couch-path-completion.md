---
id: 000160
status: codecomplete
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours: 3.60
started: 2026-09-01T12:02:48-07:00
actual_hours: N/A
---

# Couch start path tab completion

## Problem

The Couch “start thread” form accepts a filesystem path as unassisted text. An
operator must already know and type the exact directory, which makes navigating
to a nearby or unfamiliar working directory unnecessarily slow and
error-prone. `Tab` currently moves focus from the path field to the agent field,
so completion also needs an explicit field-navigation model.

## Spec

### Interaction

- While the path field is focused, `Tab` completes filesystem directories. It
  never offers regular files.
- Completion resolves the directory portion of the typed value and matches the
  final path segment. Absolute and relative paths retain their original form in
  the editable value; completion does not silently replace a relative path with
  an absolute one.
- Relative paths are interpreted from Couch's process working directory, the
  same base used by the existing start flow. Empty input lists that directory
  and inserts a relative candidate such as `src/`. An exact `.` or `..` becomes
  `./` or `../`; a following `Tab` then lists its children. A value already
  ending in a path separator lists that directory's children. Repeated
  separators are retained in the editable prefix, while the OS resolves them
  for listing. `~` has no special expansion because the start flow is not a
  shell; it is matched as a literal filename.
- One matching directory completes immediately. Multiple matches open a
  bounded menu beneath the path field, with the first match selected.
- While the completion menu is open, `Up` and `Down` move its selection and
  `Enter` accepts the selected directory. Accepting a directory writes the
  candidate into the path field with a trailing path separator so the operator
  can continue navigating. Immediate single-match completion uses the same
  trailing separator. `Tab` advances through candidates only after the menu is
  visible; the `Tab` that requested the result leaves the first candidate
  selected.
- When no completion menu is open, `Up` and `Down` move between the path and
  agent fields. Agent choice continues to use `Left` and `Right`.
- `Escape` closes an open completion menu without leaving the start form. A
  second `Escape` retains the existing start-form back behavior.
- Editing the path or moving to the agent field closes any visible completion
  menu. `Enter` retains its existing preview/start behavior whenever completion
  is not open.
- Directory names are ordered lexically. Hidden directories are offered only
  when the segment being completed begins with `.`. Symlinks that resolve to
  directories count as directories, and accepting one preserves the typed
  symlink spelling rather than substituting its resolved target.
- Ordering compares the displayed entry names bytewise, case-sensitively. This
  is deterministic across supported platforms and does not depend on the
  filesystem's enumeration order.

### Architecture and data flow

The existing `MenuState`/`ReduceMenu` reducer remains the sole transition
authority. Completion state records the request identity, candidates, and
selected candidate on the start frame. A request identity is the start frame's
immutable instance plus a monotonically increasing completion generation. A
`Tab` key transition emits a typed directory-completion effect containing that
identity and the unmodified path text. This extends the existing reducer/effect
pattern rather than creating a second input path (`ARCH-DRY`).

The Console effect shell performs the filesystem read asynchronously and sends
a completion-result event back to the reducer. Filesystem access is injected
behind a narrow directory-listing seam so production uses the OS filesystem and
tests use a deterministic stateful fake; selection, filtering, ordering, and
path reconstruction remain pure reducer/helper behavior (`ARCH-PURE`,
`ARCH-MOCK`). The effect and result both carry the complete request identity.
Editing the path, changing fields, or leaving the form advances/clears the
frame's accepted completion generation and immediately clears candidates and
any completion-owned notice before emitting any later request. A result may
mutate candidates, path, selection, or notices only when both its frame instance
and generation exactly equal the visible start frame's current request; every
other result is inert, including a result delivered to a later start frame that
happens to reuse the same numeric generation.

The directory-listing seam is batched rather than an API that materializes a
whole directory. Production opens one directory and reads at most 128 entries
per batch, classifying directory symlinks at the IO boundary. A pure bounded
accumulator consumes those facts and retains the lexically smallest 200 matching
names plus an overflow bit. It scans all entries to make lexical truncation
deterministic, but memory remains bounded to the current 128-entry batch and 200
retained candidates. The stateful fake implements the same batched contract.

Repeated `Tab` for the same path while its request is pending coalesces into the
existing request. If input changes and a new `Tab` arrives while an older
filesystem scan is still running, the shell retains only the newest pending
request behind the one active scan; another newer request replaces that pending
request. Completion work therefore has one active and at most one queued query,
and the reducer identity rule makes results from superseded work harmless.

Completion is advisory only. It does not grant start authorization or replace
the existing preview/token validation. The completed path still passes through
the ordinary start preview and canonical resolution flow (`ARCH-PURPOSE`).

### Errors and operating envelope

An absent or unreadable parent directory produces a local error notice and
leaves the typed path unchanged. No matches produces an informational “no
matching directories” notice and no menu. Cancellation or stale completion
results do not overwrite a newer notice or path. Completion notices carry the
same frame-instance/generation owner as their request. Only an exact current
result may publish success, error, no-match, or truncation state; editing,
changing fields, or leaving the frame clears a notice owned by the invalidated
request without disturbing notices owned by other menu work.

Completion is a keystroke-driven UI path (`ARCH-CONSTRAINTS`). Filesystem reads
must not block reducer input or painting. The shell runs at most one filesystem
listing and retains at most one latest-wins pending query; results expose at
most 200 matching directories, and the rendered menu is clipped to the
available terminal rows. If more than 200
directories match, the UI says the result is truncated; the operator can type a
longer prefix and request completion again. CPU, memory, and concurrent work are
therefore bounded by one directory scan and 200 retained candidates per active
start frame. A huge directory costs one O(N) sequential scan, O(N log 200)
bounded-selection CPU in the worst case, at most 128 enumerated entries plus 200
candidate names in memory, and one metadata lookup per symlink needed for
directory classification. Network-mounted filesystem latency is tolerated asynchronously;
newer input makes its eventual result stale rather than blocking the UI. The
single worker can remain occupied by a slow OS read, but it cannot multiply
unbounded work or prevent input, painting, cancellation, or form exit.

### Testing

- Pure reducer tests cover the focus-key model, immediate single completion,
  multiple-candidate selection/cycling/acceptance, escape behavior, edits that
  invalidate results, and preservation of the existing Enter-to-start path.
- Pure path-completion tests cover relative and absolute inputs, lexical order,
  directory-only filtering, hidden-directory rules, directory symlinks,
  trailing separators, no matches, batched accumulation, deterministic
  top-200 truncation, and the memory-sized retained bound.
- Console integration tests use a stateful fake directory lister to prove the
  UI stays responsive, errors remain local, repeated requests coalesce, pending
  work is latest-wins and bounded, and stale/out-of-order results cannot mutate
  current state or notices. Reducer cases explicitly cover results after form
  exit and a later frame reusing the same generation with a different frame
  instance.
- Rendering tests cover bounded candidate rows, selection, truncation text, and
  narrow/short terminals. Existing menu lifecycle and performance suites remain
  green.

## Done when

- `Tab` in Couch’s start path field completes directories through a bounded,
  visible selection menu without offering regular files.
- `Up`/`Down` provide the approved field navigation when completion is closed
  and candidate navigation when it is open; existing start and agent selection
  behavior remains intact.
- Filesystem enumeration is asynchronous, injected, and protected against
  stale results; errors never destroy the operator’s typed path.
- Automated reducer, completion, shell-integration, rendering, and regression
  tests prove the specified behavior.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.00 impl=0.08
item: greenfield-go-module design=0.30 impl=0.32
item: smaller-go-module design=0.06 impl=0.16
item: tui-screen design=0.40 impl=0.40
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.05 impl=0.05
item: milestone-review design=0.05 impl=0.14
design-buffer: 0.15
total: 3.60
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only. The calibration source is marked
stale, so the estimate is provisional. The approved plan earns the 15% design
buffer; implementation values are already scaled to 40% of the v2/v2.1 table,
and the existing Couch/Go architecture keeps familiarity at 1.0.

## Plan

- [x] Build the pure path query, bounded accumulator, and shared latest-wins scheduler.
- [x] Add completion identity, key transitions, invalidation, notices, and bounded rendering to the menu reducer.
- [x] Wire batched filesystem enumeration through the Console with a stateful fake and bounded worker schedule.
- [x] Exercise the real input loop, update README/Atlas, run full verification, and close through the SDLC gate.

## Log

### 2026-09-01
- 2026-09-01: closed — go test -race ./cmd/internal/couchtty -run ^(TestConsoleCompletion|TestOSDirectoryBatchReader|TestFakeDirectoryBatchReader)$ -count=1; go test ./cmd/internal/couchtty -count=1; go test ./... -count=1; git diff --check; live Console run-loop fake-filesystem completion coverage passes; review verdict: SHIP

Claimed the issue and entered planning. The operator approved a directories-only
completion menu: `Tab` completes/cycles paths, `Up`/`Down` navigate fields when
closed and candidates when open, and `Left`/`Right` retain agent selection.

The first fresh-context spec review found ambiguous path bases and async
ownership. The spec now defines literal path semantics, exact frame/generation
identity, owned notices, and a one-active/one-latest-pending filesystem queue.

The second review caught whole-directory materialization beneath the result
cap. The filesystem contract now enumerates in bounded batches and retains only
a bounded lexical top set.

The approved spec was translated into the durable implementation plan at
`workshop/plans/000160-couch-path-completion-plan.md`; no milestone tags are
used because this is one atomic review boundary.

The first `change-code` plan-quality round raised PQ-1
(`executable-test-strategy`): the plan was over-specified with prose fixtures
and diff procedure. Tasks 1–6 now name each tested function and its adversarial
input/mechanical guard while leaving executable cases to test code.

The second plan-quality round found compact case inventories still violated the
same PQ-1 class. The plan now applies one uniform rule to every risky surface:
named function, one adversarial input class, and one mechanical guard only.

The third plan-quality round disposed PQ-1 as addressed. The reconciled 3.60h
estimate was then derived with estimate-logic-v3.1 from the accepted scope; the
calibration source reports stale, so the value is provisional.

Implementation kept completion advisory behind the existing preview grant.
The pure query/accumulator caps retained names at 200, the Console reads 128
filesystem entries per batch behind an injected seam, and the generalized
latest-wins scheduler admits one active plus one newest pending request
(`ARCH-DRY`, `ARCH-PURE`, `ARCH-MOCK`, `ARCH-CONSTRAINTS`). Exact
frame/generation matching rejects late results after edits or form lifetimes.

Verification passed: `go test -race ./cmd/internal/couchtty -run
'^(TestConsoleCompletion|TestOSDirectoryBatchReader)' -count=1`, `go test
./cmd/internal/couchtty -count=1`, `go test ./... -count=1`, and `git diff
--check`. The full-suite worktree prerequisite was generated with the declared
runtime-bundle generator; the exhaustive artifact-source inventory was updated
for the two new production files.

The first close review returned FIX-THEN-SHIP with BR-1
(`filesystem-terminal-error-propagation`) and BR-2
(`external-double-contract-fidelity`). The OS seam now injects a close-capable
cursor and joins terminal close failures; the stateful fake now chunks by the
requested bound, records batch sizes, propagates configured errors, and drives
a Console-level local-error assertion. Added the generalized prevention rule to
`workshop/lessons.md` (`ARCH-MOCK`, `ARCH-CONSTRAINTS`).

Close-review round two disposed BR-2 and kept BR-1 open: broad cancellation
filtering erased a sibling close failure from `errors.Join`. The worker now
removes only cancellation leaves and preserves all other terminal errors; a
Console-level combined cancellation/close regression is green under `-race`.

## Revisions

### 2026-09-01 — close first spec-review ambiguities

Defined empty/relative/dot/separator/tilde behavior and deterministic ordering;
made the first versus subsequent `Tab` behavior explicit; and added exact
request/notice ownership plus bounded latest-wins filesystem concurrency.

### 2026-09-01 — bound huge-directory enumeration

Replaced whole-directory materialization with 128-entry batched enumeration and
a pure bounded lexical top-200 accumulator. Clarified that invalidation clears
completion state immediately and added both halves of request identity to the
regression contract.

### 2026-09-01 — add durable implementation plan

Replaced the placeholder plan row with four checkable delivery groups and added
the detailed TDD sequence, core-concept inventory, operating envelope, and SDLC
close handoff in the canonical plan artifact.

### 2026-09-01 — address plan-quality PQ-1

Compressed the entire implementation plan from enumerated fixture/diff prose to
named-function test strategies, observable outcomes, architectural ownership,
and exact RED/GREEN commands.

### 2026-09-01 — apply PQ-1 as a class-wide rule

Removed the remaining parenthetical case lists and procedural mechanics across
Tasks 1–6; each test step now follows the gate's exact strategy-line contract.

### 2026-09-01 — derive post-gate estimate

Added the estimate-logic-v3.1 Method A primitive block and reconciled
`estimate_hours` only after the plan-quality gate cleared, as required.
