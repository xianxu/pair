---
id: 000160
status: working
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
started: 2026-09-01T12:02:48-07:00
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
- One matching directory completes immediately. Multiple matches open a
  bounded menu beneath the path field, with the first match selected.
- While the completion menu is open, `Up` and `Down` move its selection and
  `Enter` accepts the selected directory. Accepting a directory writes the
  candidate into the path field with a trailing path separator so the operator
  can continue navigating. `Tab` advances through the same candidates.
- When no completion menu is open, `Up` and `Down` move between the path and
  agent fields. Agent choice continues to use `Left` and `Right`.
- `Escape` closes an open completion menu without leaving the start form. A
  second `Escape` retains the existing start-form back behavior.
- Editing the path or moving to the agent field closes any visible completion
  menu. `Enter` retains its existing preview/start behavior whenever completion
  is not open.
- Directory names are ordered lexically. Hidden directories are offered only
  when the segment being completed begins with `.`. Symlinks that resolve to
  directories count as directories.

### Architecture and data flow

The existing `MenuState`/`ReduceMenu` reducer remains the sole transition
authority. Completion state records the request generation, candidates, and
selected candidate on the start frame. A `Tab` key transition emits a typed
directory-completion effect containing the unmodified path text and a unique
generation. This extends the existing reducer/effect pattern rather than
creating a second input path (`ARCH-DRY`).

The Console effect shell performs the filesystem read asynchronously and sends
a completion-result event back to the reducer. Filesystem access is injected
behind a narrow directory-listing seam so production uses the OS filesystem and
tests use a deterministic stateful fake; selection, filtering, ordering, and
path reconstruction remain pure reducer/helper behavior (`ARCH-PURE`,
`ARCH-MOCK`). Results whose generation or frame identity no longer matches the
visible start frame are discarded, including results arriving after edits,
field changes, or form exit.

Completion is advisory only. It does not grant start authorization or replace
the existing preview/token validation. The completed path still passes through
the ordinary start preview and canonical resolution flow (`ARCH-PURPOSE`).

### Errors and operating envelope

An absent or unreadable parent directory produces a local error notice and
leaves the typed path unchanged. No matches produces an informational “no
matching directories” notice and no menu. Cancellation or stale completion
results do not overwrite a newer notice or path.

Completion is a keystroke-driven UI path (`ARCH-CONSTRAINTS`). Filesystem reads
must not block reducer input or painting. The shell requests at most one listing
per explicit `Tab`, results expose at most 200 matching directories, and the
rendered menu is clipped to the available terminal rows. If more than 200
directories match, the UI says the result is truncated; the operator can type a
longer prefix and request completion again. CPU, memory, and concurrent work are
therefore bounded by one directory scan and 200 retained candidates per active
start frame. Network-mounted filesystem latency is tolerated asynchronously;
newer input makes its eventual result stale rather than blocking the UI.

### Testing

- Pure reducer tests cover the focus-key model, immediate single completion,
  multiple-candidate selection/cycling/acceptance, escape behavior, edits that
  invalidate results, and preservation of the existing Enter-to-start path.
- Pure path-completion tests cover relative and absolute inputs, lexical order,
  directory-only filtering, hidden-directory rules, directory symlinks,
  trailing separators, no matches, and the 200-result bound.
- Console integration tests use a stateful fake directory lister to prove the
  UI stays responsive, errors remain local, and stale/out-of-order results
  cannot mutate current state.
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

## Plan

- [ ] Write the implementation plan after the approved spec passes review.

## Log

### 2026-09-01

Claimed the issue and entered planning. The operator approved a directories-only
completion menu: `Tab` completes/cycles paths, `Up`/`Down` navigate fields when
closed and candidates when open, and `Left`/`Right` retain agent selection.
