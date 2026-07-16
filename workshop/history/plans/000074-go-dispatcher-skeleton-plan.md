# Pair Go Dispatcher Skeleton Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an opt-in Go dispatcher skeleton that establishes Pair's future primary CLI shape without changing the public `bin/pair` launcher.

**Architecture:** Create a pure `cmd/internal/dispatcher` package for argv parsing, help rendering, planned command metadata, and unsupported-command errors (`ARCH-PURE`). Add `cmd/pair-go/main.go` as a thin IO shell that calls the dispatcher and writes output/exit codes. Wire `pair-go` into the existing `GO_BINS` build pattern while leaving `bin/pair`, zellij layouts, and helper command names unchanged (`ARCH-DRY`, `ARCH-PURPOSE`).

**Tech Stack:** Go 1.26, standard library only, existing `Makefile.local` build conventions.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `CommandFamily` | `cmd/internal/dispatcher/dispatcher.go` | new |
| `Result` | `cmd/internal/dispatcher/dispatcher.go` | new |
| `Dispatch` | `cmd/internal/dispatcher/dispatcher.go` | new |
| `Help` | `cmd/internal/dispatcher/dispatcher.go` | new |

- **CommandFamily** — metadata for planned Pair command families that the skeleton can list without implementing.
  - **Relationships:** N:1 with `Help`; `Help` renders all families from one slice.
  - **DRY rationale:** One source for names/descriptions/status text instead of duplicating planned command lists between help and dispatch validation.
  - **Future extensions:** Add handler wiring when later issues port a family.

- **Result** — pure dispatch outcome: stdout, stderr, and process exit code.
  - **Relationships:** 1:1 with a dispatcher invocation; consumed by the IO shell.
  - **DRY rationale:** Keeps exit behavior testable without spawning a subprocess.
  - **Future extensions:** Can grow a handler enum or execution callback once subcommands become real.

- **Dispatch** — pure argv parser and router for `help`, `--help`, `version`, and unsupported commands.
  - **Relationships:** Owns one invocation's decision; reads `CommandFamily` metadata; returns `Result`.
  - **DRY rationale:** Centralizes command parsing before behavior begins migrating out of shell scripts.
  - **Future extensions:** Replace unsupported-command outcomes with real handlers one family at a time.

- **Help** — pure renderer for the dev CLI usage and planned command families.
  - **Relationships:** Reads `CommandFamily`; used by `Dispatch`.
  - **DRY rationale:** Help text and tests derive from the same command-family metadata.
  - **Future extensions:** Add richer command-specific help when handlers exist.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `pair-go main` | `cmd/pair-go/main.go` | new | `os.Args`, stdout/stderr, process exit |
| `pair-go build target` | `Makefile.local` | modified | `go build` into `bin/pair-go` |
| `architecture note` | `atlas/architecture.md` | modified | repo map documentation |

- **pair-go main** — minimal binary entrypoint: call `dispatcher.Dispatch(os.Args[1:])`, print streams, exit.
  - **Injected into:** No pure entity; it consumes the pure `Result`.
  - **Future extensions:** Later issues can replace result-only routing with handler execution behind the same package.

- **pair-go build target** — follow the existing `GO_BINS` and per-binary recipe convention so `make build` compiles the skeleton.
  - **Injected into:** Build/install flow only.
  - **Future extensions:** The public `pair` entrypoint switch happens in a later issue, not here.

- **architecture note** — record that `pair-go` is the opt-in dispatcher skeleton and not the public launcher.
  - **Injected into:** Atlas readers and future migration issues.
  - **Future extensions:** Update when the public entrypoint changes.

## Chunk 1: Dispatcher Skeleton

### Task 1: Add pure dispatch tests

**Files:**
- Create: `cmd/internal/dispatcher/dispatcher_test.go`
- Later create: `cmd/internal/dispatcher/dispatcher.go`

- [x] **Step 1: Write failing tests for help, version, and unsupported commands**

Cover:
- empty argv and `help` return exit `0` with usage and planned families.
- `--help` and `-h` mirror help.
- `version` returns deterministic dev metadata without pretending to be the shell launcher.
- a planned-but-unimplemented command like `wrap` returns exit `2` and an unsupported message.
- an unknown command returns exit `2` and suggests `pair-go help`.

- [x] **Step 2: Run the package test and verify it fails**

Run: `go test ./cmd/internal/dispatcher -count=1`

Expected: fail because `dispatcher.go` does not exist yet.

- [x] **Step 3: Implement the minimal dispatcher**

Create:
- `CommandFamily` with `Name`, `Summary`, and `Status`.
- `Families()` returning planned command families: `launch`, `wrap`, `slug`, `context`, `scrollback-render`, `changelog`, `continuation`, `scribe`.
- `Result` with `Stdout`, `Stderr`, `ExitCode`.
- `Dispatch(args []string) Result`.
- `Help(program string) string`.

- [x] **Step 4: Run dispatcher tests and verify they pass**

Run: `go test ./cmd/internal/dispatcher -count=1`

Expected: pass.

### Task 2: Add the opt-in binary wrapper and build target

**Files:**
- Create: `cmd/pair-go/main.go`
- Modify: `Makefile.local`

- [x] **Step 1: Write a failing wrapper smoke test if needed**

If the pure tests cover all behavior, keep `main.go` untested directly and rely on `go test ./...` plus build verification. Do not introduce subprocess tests unless pure coverage misses behavior.

- [x] **Step 2: Implement `cmd/pair-go/main.go`**

Main reads `os.Args[1:]`, delegates to `dispatcher.Dispatch`, writes stdout/stderr to the matching streams, and exits with the returned code.

- [x] **Step 3: Wire `pair-go` into `Makefile.local`**

Append `pair-go` to `GO_BINS`, add the `.PHONY` alias, and add:

```make
$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
	go build -o $@ ./cmd/pair-go
```

- [x] **Step 4: Build the new binary**

Run: `make pair-go`

Expected: `bin/pair-go` is created.

### Task 3: Document and verify non-disruption

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `workshop/issues/000074-go-dispatcher-skeleton.md`

- [x] **Step 1: Update atlas**

Add `bin/pair-go` to the piece list as the opt-in Go dispatcher skeleton and mention it in the packaging migration target as development-only for this issue.

- [x] **Step 2: Run focused verification**

Run:
- `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
- `make pair-go`
- `go test ./... -count=1`
- `bash tests/pair-continue-test.sh`

Expected: all pass. The shell launcher remains the public entrypoint.

- [x] **Step 3: Update issue checkboxes and log**

Mark completed plan and done-when boxes, then close through `sdlc close --issue 74 --verified '<evidence>'` with atlas evidence. This plan expects an atlas update, so prefer satisfying the atlas gate.
