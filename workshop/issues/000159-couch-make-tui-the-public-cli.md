---
id: 000159
status: working
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
started: 2026-09-01T09:03:30-07:00
---

# couch: make TUI the public CLI

## Problem

Couch is a terminal UI, but its executable currently presents every typed
operation as a peer CLI subcommand. Bare `couch` prints that implementation
inventory instead of opening the workspace, so the public interface describes
the internal state machine rather than the product the operator uses.

## Spec

Couch has one primary user mode: open the TUI. Bare `couch` opens Couch in the
current directory, and `couch <path>` opens it for the given repository or
subdirectory. This is the existing console-owning start flow; `start` is removed
as a user concept and is not retained as a compatibility alias.

Two read-only diagnostic exceptions remain as flags:

- `couch --list` prints the durable thread inventory.
- `couch --show <ref>` prints one current-repository thread by tag, path, or
  operator name.

`couch --help` documents only `couch [path]`, `--list`, `--show`, and help. It
does not enumerate lifecycle actions or protocol endpoints. Park, resume,
leave, switch, stop, name, and describe are TUI actions only; their existing
typed operations remain the single in-process authority used by the TUI
(`ARCH-DRY`, `ARCH-PURPOSE`).

The few operations that genuinely cross a process boundary remain reachable
through the hidden machine protocol `couch --internal <operation> [arguments]`.
This is the existing `couch` executable, not a new `couch-internal` binary.
Normal help and user documentation omit `--internal`; Pair-owned hooks and
launch/attach call sites use it explicitly. Internal resolution still derives
from the typed Couch operation registry, so the new CLI projection does not
create a parallel operation model (`ARCH-DRY`). Unknown internal operations,
malformed flags, missing values, nonexistent paths, and extra public positional
arguments fail nonzero with a local, actionable error.

Argument classification is a small pure parser: public flags are recognized
before path launch, `--` permits a path beginning with `-`, and internal
arguments are passed to the existing typed binder only after the hidden
boundary is selected. IO, namespace acquisition, policy lookup, admission, and
console construction stay in the current runtime shell (`ARCH-PURE`).

This is a startup/UI interaction. Parsing adds no filesystem scan or external
process and must be negligible relative to existing Couch startup; the change
must not move policy lookup, thread inventory work, or actor launch ahead of
the first TUI response. Verification uses the injected runtime and existing
stateful Couch fakes, plus an installed-command smoke for bare invocation
(`ARCH-MOCK`, `ARCH-CONSTRAINTS`).

## Done when

- Bare `couch` opens the TUI in the current directory.
- `couch <path>` opens the same TUI for that path without a `start` subcommand.
- `couch --list` and `couch --show <ref>` provide the two public diagnostics.
- Normal help exposes no lifecycle-action or machine-protocol subcommands.
- Required Pair/Couch subprocess integrations use the hidden `--internal`
  protocol, while TUI actions continue to use the typed in-process registry.
- A positional token such as `start` is only a path (and fails if that path does
  not exist); old lifecycle subcommands, malformed public arguments, and unknown
  internal operations do not silently select a different behavior.
- Automated tests cover parsing, help, public launch/diagnostics, every required
  internal call site, and the installed bare-command path.

## Plan

- [ ] Write and approve a durable implementation plan.
- [ ] Implement the public CLI projection and hidden protocol using TDD.
- [ ] Update every Pair/Couch caller, README, and atlas surface.
- [ ] Verify focused, installed-command, and bounded full-suite behavior.

## Log

### 2026-09-01

The operator selected a TUI-first surface: `couch [path]`, with `--list` and
`--show` as diagnostic flags. There is no `start` compatibility alias and no
separate `couch-internal` executable. Necessary cross-process operations live
behind an undocumented `--internal` flag; all user lifecycle actions remain in
the TUI.
