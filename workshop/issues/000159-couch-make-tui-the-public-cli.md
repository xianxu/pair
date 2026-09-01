---
id: 000159
status: working
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours: 2.41
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

The public argv grammar is closed:

| argv | result |
|---|---|
| `couch` | launch the TUI at `.` |
| `couch <path>` | launch the TUI at that path |
| `couch -- <path>` | launch a path whose spelling begins with `-` |
| `couch --list` | print the global durable inventory |
| `couch --show <ref>` | print one thread, resolving repository scope only from process CWD |
| `couch --help` / `couch -h` | print public help without runtime initialization |
| anything else | exit 2 with an argv error; do not initialize policy, storage, or actors |

Every public flag form is exact: it cannot be combined with a path or another
flag, repeated, or placed after a positional path. `--help` is valid only as
the sole token. `--` accepts exactly one following non-empty path. An empty
positional path fails. `--internal=<operation>`, a missing internal operation,
and `--` within internal arguments fail rather than choosing another mode.
Relative paths resolve from CWD; symlinks, repository subdirectories,
non-repository directories, and existing non-directory paths retain the
existing canonical Couch/Git resolution behavior and errors. A word matching a
removed operation, such as `start`, is only a path and receives the same
validation.

`couch --help` documents only `couch [path]`, `--list`, `--show`, and help. It
does not enumerate lifecycle actions or protocol endpoints. Park, resume,
leave, switch, stop, name, and describe are TUI actions only; their existing
typed operations remain the single in-process authority used by the TUI
(`ARCH-DRY`, `ARCH-PURPOSE`).

The one operation that genuinely crosses a process boundary remains reachable
through the hidden machine protocol `couch --internal publish-description
<text>`.
This is the existing `couch` executable, not a new `couch-internal` binary.
Normal help and user documentation omit `--internal`; the hosted agent
description hook migrates from `couch publish-description <text>` to this exact
form. The description may be the explicit empty argv value to clear it, and
scope/tag still come only from `$COUCH_THREAD_SCOPE` and `$COUCH_THREAD_TAG`.
Internal resolution first checks a deliberate whitelist containing only
`publish-description`, then derives its schema and execution from the typed
Couch operation registry. Adding a registry operation never exposes it through
argv automatically (`ARCH-DRY`, `ARCH-PURPOSE`).

Every typed operation has one allowed presentation:

| operation | allowed presentation |
|---|---|
| `list` | public `--list` diagnostic |
| `show` | public `--show <ref>` diagnostic |
| `publish-description` | hidden process protocol only |
| `prepare-start`, `start`, `attach`, `switch`, `park`, `resume`, `leave`, `stop`, `name`, `describe` | TUI/in-process dispatch only; never argv-reachable |

Current documentation and tests for old `start`, `list`, `show`, `resume`, and
lifecycle command forms migrate to the table above rather than remaining as
aliases. Root launch no longer accepts `--agent` or `--no-console`: agent choice
belongs to Couch/Pair defaults and the TUI's start-thread flow, while a bare or
path launch without terminal stdin/stdout exits nonzero before policy lookup or
actor creation. Installed-command smoke runs under a real PTY.

Argument classification is a small pure parser returning one closed result:
`Launch{Path}`, `List`, `Show{Ref}`, `Internal{Operation, Args}`, `Help`, or
`ParseError`. Internal arguments are passed to the existing typed binder only
after the whitelist boundary is selected. IO, namespace acquisition, policy
lookup, admission, and console construction stay in the current runtime shell
(`ARCH-PURE`).

This is a startup/UI interaction. Parsing is pure, allocation-bounded by the
small argv vector, linear in argv bytes, and adds no filesystem scan or external
process. The change must not move policy lookup, thread inventory work, or actor
launch ahead of the existing TUI startup point. Verification uses the injected
runtime and existing stateful Couch fakes, plus an installed-command smoke for
bare invocation (`ARCH-MOCK`, `ARCH-CONSTRAINTS`).

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

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The work extends familiar Go command and TUI
seams; no novel stack or missing-library adjustment applies.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: smaller-go-module design=0.06 impl=0.16
item: cross-cutting-refactor design=0.10 impl=0.16
item: cross-cutting-refactor design=0.10 impl=0.16
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.10 impl=0.08
item: milestone-review design=0.04 impl=0.12
design-buffer: 0.15
total: 2.41
```

## Plan

- [x] Write and approve a durable implementation plan.
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

The durable implementation plan passed fresh-context review after five rounds.
The final plan puts the installed bare-command PTY smoke under RED before code,
classifies every typed operation in the registry, migrates TUI start to its
accepted token only, makes terminal/process teardown explicit and bounded, and
enforces a repository-wide current-source shadow sweep for obsolete argv
(`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

The first SDLC plan-quality round raised two Important findings. The plan now
expresses verification as named-function adversarial strategies rather than
prose test inventories/diff choreography, and states explicit non-goals with
rationale. This keeps the plan executable without duplicating the tests or code
it will produce (`ARCH-PURE`, `ARCH-PURPOSE`).

Implementation now projects the typed registry into four explicit homes: TUI,
public list, public show, and hidden process protocol. A repository-wide current
source guard rejects obsolete command-shaped Couch instructions across README,
atlas, active #153 guidance, probes, and production Go; historical milestone
records remain intact. The installed PTY smoke proves bare launch reaches the
fake Pair process, while the piped form proves terminal refusal happens before
policy or actor effects (`ARCH-PURPOSE`, `ARCH-DRY`, `ARCH-CONSTRAINTS`).

## Revisions

### 2026-09-01T09:18:00-07:00 — close the argv and operation projections

**Reason:** fresh-context spec review found that the first draft did not
enumerate the internal whitelist, current caller migration, removed launch
options, parser precedence, path behavior, or non-terminal startup.

**Delta:** the spec now gives closed argv and operation-presentation tables;
whitelists only `publish-description` for hidden process invocation; removes
`--agent` and `--no-console`; makes non-terminal launch fail before effects;
defines path/flag precedence and canonical-resolution ownership; and names the
pure parser's finite result variants. These enumerations make the central
public/internal projection exhaustively testable (`ARCH-PURPOSE`, `ARCH-PURE`,
`ARCH-CONSTRAINTS`).
