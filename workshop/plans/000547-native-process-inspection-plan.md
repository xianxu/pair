# Native process inspection implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use test-first steps below.

**Goal:** Remove production ps subprocesses, preserving launcher and cleanup decisions under macOS seatbelt.

**Architecture:** Keep procutil.Table as a cheap native metadata snapshot. Add native executable/argv inspection and a context-aware enriched snapshot which retains per-process inspection errors and checks start identities. Consumers use typed observations rather than reconstructing ps output. Unsupported OS readers explicitly return an unsupported error; legacy string helpers retain their empty-on-error contract.

**Tech Stack:** Go, x/sys/unix sysctl on Darwin, procfs on Linux.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Process (PID, PPID, Identity, Nice, short kernel Name) | cmd/internal/procutil/procutil.go | modified |
| CommandInfo (executable and argv), Darwin argv decoder, Linux stat/cmdline decoders | cmd/internal/procutil/command.go, table_parse.go | new |
| Typed server projection | cmd/internal/launcher/session_servers.go | modified |
| Process ancestry projection | cmd/internal/launcher/osruntime.go | modified |

Process rows belong to one table read. CommandInfo never treats argv[0] as executable identity: argv can be rewritten. Kernel Name is explicitly short and can only prefilter candidates conservatively. Darwin procargs2 includes the executable path independently; Linux uses /proc/PID/exe. The existing ParseServerProcesses text API remains for existing callers/tests but delegates per-command matching to the same pure projection. Direct unit tests cover malformed native buffers, embedded spaces/empty argv, environment exclusion, PID cycles and exact server selection.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Native command readers | cmd/internal/procutil/command_darwin.go, command_linux.go, command_other.go | new | kern.procargs2 or procfs |
| Table readers | cmd/internal/procutil/table_darwin.go, table_linux.go, table_other.go | modified | native metadata |
| Snapshot/context and per-row inspection | cmd/internal/procutil/command.go | new | Table plus argv reader and identity recheck |
| Launcher host observations | cmd/internal/launcher/osruntime.go, session_servers.go, session_quiescence.go | modified | procutil |
| Runtime inventory | cmd/internal/diagnosticlog/proof.go | modified | procutil plus existing environment reader |

The snapshot starts with Table, then reads commands serially and rechecks each row's start identity before accepting executable/argv. Gone, unreadable and changed rows retain an error, never fabricated argv. Context cancellation stops enrichment and returns the context error. Callers that need complete zellij evidence refuse if a potential zellij row cannot be inspected; unrelated inaccessible rows do not make every observation fail. Short kernel names are conservative prefilters (including truncation prefixes); successful native executable/argv still decides membership. Orphan-nvim discovery remains best effort. Diagnostic inventory checks full native executable names and retains StrictIdentity fencing across environment inspection.

Tests use a stateful process-reader fake (table and command/identity reads can remove/recycle rows between calls) behind the same snapshot helper used in production; Linux procfs fixtures model stat, exe symlink and cmdline. Real own-process/helper-child probes compare native metadata, executable and arguments, on macOS under seatbelt and Linux CI. No actual unrelated process is signaled.

## Constraints and principles

- ARCH-DRY/PURPOSE: enumerate all production Go ps calls including wrappers: five launcher sites, procutil Command/ProcessChildren/other Identity, diagnostic inventory. A repository AST/source guard prevents their return. Reuse existing server matching and procutil identity, not parallel OS logic in callers.
- ARCH-PURE: native buffer decoding, server selection and ancestry stay pure; syscall/file reads stay in platform adapters. Add new production paths to artifactpath manifest.
- ARCH-MOCK: snapshot fake controls disappearance and PID reuse; Linux fixture tests drive the real filesystem adapter. Existing launcher stateful worlds remain unchanged. Real helper-child conformance runs on supported OSes.
- ARCH-CONSTRAINTS: startup/refresh/maintenance path, O(process count), one metadata table read and at most one argv read plus identity recheck per inspected process, serial (no fan-out). Cheap Table consumers do not pay argv costs. Command input bound 2 MiB (existing Darwin environment bound); Linux uses limited reads. Context checked before reads and between rows. Native syscalls cannot be interrupted mid-call; no detached workers. No persistent cache.
- ARCH-SECURE: treat kernel/procfs buffers as externally mutable; bound sizes/counts and reject truncation. Do not retain/export environment in command results or log argv. Missing inspection is uncertainty, not authority to delete/signal. Unsupported native readers return an explicit unsupported error; legacy Identity/Command preserve empty-on-error API.
- ARCH-ORDER: snapshot holds no durable state between decisions; process exit/PID reuse during enrichment returns unavailable for that row. Destructive consumers retain final identity/argv fences. Cancellation aborts observation without work continuing. Controlled fake sequences reproduce exit, reuse and cancellation.
- ARCH-FUNERAL: no new persistent artifacts, subprocesses, cache or background workers; snapshots die with their caller. Test children are killed and waited by cleanup.

## Tasks

- [ ] Add failing tests for command decoding and native self/child inspection; remove ps-unavailable skips on supported platforms. Add pure Linux stat fixtures and short-name coverage. Run targeted tests and record red.
- [ ] Implement native command readers and enriched snapshot with per-row errors, cancellation and identity fencing. Convert Command, ProcessChildren and unsupported Identity. Run pure/fake tests, real Darwin probes, Linux cross-build (Linux native tests in CI).
- [ ] Add failing production no-ps guard and launcher projection/ancestry regressions. Migrate all five launcher sites to typed procutil observations; preserve server identities for reachable and orphaned rows when integrating #528. Run targeted launcher/procutil tests.
- [ ] Migrate diagnostic runtime inventory with full executable names and StrictIdentity checks. Replace ps PATH fixtures with an explicit existing/new inventory injection seam in diagnostic/gccmd tests, preserving maintenance cancellation and isolation from real unrelated processes. Run both package suites.
- [ ] Update artifact ownership inventory and atlas architecture description. Merge fresh main, resolve overlap preserving both purposes, then run targeted race tests, make -k build and full make -k test in background.
- [ ] Build process test binaries and run them under sandbox-exec with (version 1)(allow default); require native process probes and launcher process tests to pass, without ps skips. Record actual commands/results and commit. Close with SDLC fresh-context review, fix findings, then PR/merge and report real hashes to hub.

## Verification commands

- go test ./cmd/internal/procutil ./cmd/internal/launcher ./cmd/internal/diagnosticlog ./cmd/internal/gccmd -run '<focused changed tests>' -count=1
- go test -race ./cmd/internal/procutil ./cmd/internal/launcher -run '<focused process tests>' -count=1
- GOOS=linux GOARCH=amd64 go test -c -o /tmp/pair547-procutil-linux.test ./cmd/internal/procutil
- go test -c -o /tmp/pair547-procutil.test ./cmd/internal/procutil; sandbox-exec -p '(version 1)(allow default)' /tmp/pair547-procutil.test -test.run '<process probes>' -test.v (run as separate commands)
- make -k build; make -k test (separate calls, full suite background)

## Coordination

#528 has unlanded edits at 5bb368c363fc971b944c6601c785e033c7c8cf5c. Do not stack on that branch. Merge main regularly; preserve its immutable server generation authorization and reachable identity enrichment when it lands. Hub approved the class-wide sweep and full flow, including explicit unsupported fallback, receipt f7aa9f0b-b462-4513-81a2-1ff946803f80.
