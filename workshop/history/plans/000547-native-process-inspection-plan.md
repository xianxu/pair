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

- [x] Add failing tests for command decoding and native self/child inspection; remove ps-unavailable skips on supported platforms. Add pure Linux stat fixtures and short-name coverage. Run targeted tests and record red.
- [x] Implement native command readers and enriched snapshot with per-row errors, cancellation and identity fencing. Convert Command, ProcessChildren and unsupported Identity. Run pure/fake tests, real Darwin probes, Linux cross-build (Linux native tests in CI).
- [x] Add failing production no-ps guard and launcher projection/ancestry regressions. Migrate all five launcher sites to typed procutil observations; preserve server identities for reachable and orphaned rows when integrating #528. Run targeted launcher/procutil tests.
- [x] Migrate diagnostic runtime inventory with full executable names and StrictIdentity checks. Replace ps PATH fixtures with an explicit existing/new inventory injection seam in diagnostic/gccmd tests, preserving maintenance cancellation and isolation from real unrelated processes. Run both package suites.
- [x] Update artifact ownership inventory and atlas architecture description. Merge fresh main, resolve overlap preserving both purposes, then run targeted race tests, make -k build and full make -k test in background.
- [ ] Build process test binaries and run them under sandbox-exec with (version 1)(allow default); require native process probes and launcher process tests to pass, without ps skips. Record actual commands/results and commit. Close with SDLC fresh-context review, fix findings, then PR/merge and report real hashes to hub.

## Verification commands

- go test ./cmd/internal/procutil ./cmd/internal/launcher ./cmd/internal/diagnosticlog ./cmd/internal/gccmd -run '<focused changed tests>' -count=1
- go test -race ./cmd/internal/procutil ./cmd/internal/launcher -run '<focused process tests>' -count=1
- GOOS=linux GOARCH=amd64 go test -c -o /tmp/pair547-procutil-linux.test ./cmd/internal/procutil
- go test -c -o /tmp/pair547-procutil.test ./cmd/internal/procutil; sandbox-exec -p '(version 1)(allow default)' /tmp/pair547-procutil.test -test.run '<process probes>' -test.v (run as separate commands)
- make -k build; make -k test (separate calls, full suite background)

## Coordination

#528 has unlanded edits at 5bb368c363fc971b944c6601c785e033c7c8cf5c. Do not stack on that branch. Merge main regularly; preserve its immutable server generation authorization and reachable identity enrichment when it lands. Hub approved the class-wide sweep and full flow, including explicit unsupported fallback, receipt f7aa9f0b-b462-4513-81a2-1ff946803f80.

## Revisions

### 2026-10-14 — PQ-1 / PQ-2: executable interfaces and scope

The following replaces the unnamed test-topic paragraph and makes the implementation seams concrete. `CommandInfo` has `Executable string` and `Args []string`. `Observation` embeds `Process` and `CommandInfo`, with `Err error` for unavailable per-process evidence. `Snapshot(context.Context) ([]Observation, error)` enriches all rows (not only kernel-name matches); short names are used only to classify unreadable potential candidates. This preserves spoofed argv coverage in the existing live quiescence sentinel.

| Function | Mechanical verification strategy |
|---|---|
| `ParseDarwinProcArgs(raw) (CommandInfo, []byte, error)` | Bounded malformed-buffer fuzzing enforces argc/termination/size bounds; returned remainder starts at environment, never included in argv. Reuse in diagnosticlog's existing `parseDarwinEnvironment` to avoid a second envelope parser. |
| `parseLinuxStat(pid, raw) (Process, error)` | Portable pure fixtures and fuzzing enforce stat field offsets and parenthesized-name boundaries; production Table and Identity share the parser. |
| `parseLinuxCmdline(raw) ([]string, error)` | Bounded arbitrary-byte fixtures enforce NUL termination and preserve argument boundaries. |
| `readLinuxCommand(root, pid) (CommandInfo, error)` | Temporary procfs fixture uses stat/cmdline files and exe symlinks; exercises actual filesystem adapter on either development host. |
| `snapshot(ctx, reader)` / public `Snapshot(ctx)` | Stateful reader fake removes/recycles rows between table, command and identity reads, asserting unavailable observations cannot acquire another incarnation's command; cancellation prevents subsequent reads. Native own-process probe checks argv/executable/identity agreement on supported hosts. |
| `Command(pid)` / `ProcessChildren()` | Real child process conformance pins command availability and parent membership with ps absent from PATH; seatbelt run must pass, not skip. |
| `serverProcesses(observations)` | Direct typed fixtures assert existing exact server matching and fail-closed unreadable zellij candidates; identities derive from the same observation, never a second table. |
| `hasProcessAncestor(rows, pid, name)` | Pure graph fixtures assert exact short-name ancestry, missing-parent termination and cycle termination using one snapshot. |
| `OSInspection.Runtimes(ctx)` | Injected snapshot/environment/identity reader uses stateful inventory fixtures to pin long executable names and disappearance during environment inspection; canceled contexts cannot authorize cleanup. |
| `psExecViolations(root)` | Scan production Go AST call arguments for literal ps and /bin/ps (including wrappers); synthetic violating sources prove direct/context/absolute/wrapped calls are detected. |

Inventory injection: `OSInspection` accepts an optional process snapshot reader, environment reader and strict identity reader (zero value native). `gcruntime.NewWithInspection(root, inspection)` wires the existing Inspection interface to both legacy runtime proof and diagnostic writer proof; `New` delegates with OSInspection. `gccmd.Run` delegates to a private runner accepting the service constructor; acceptance tests inject a stateful Inspection instead of PATH-faking ps. These narrow composition seams preserve the real storage lifecycle tested by public apply.

Non-goals: no zellij authorization redesign (owned by #528), no new whitespace-bearing server matching behavior, no new process cache/daemon, no support for new operating systems. Existing string command consumers keep space-joined argv semantics. Native decoding preserves individual arguments internally. Linux native execution is required in available Linux CI; local verification includes portable Linux-reader fixtures and cross-compilation, explicitly distinguished from native execution if no Linux runtime is available.

### 2026-10-14 — implementation locations and safety discovery

Shared decoders live in command_parse.go (rather than the provisional table_parse.go); portable Linux IO lives in command_procfs.go. Diagnostic process-reader injection stays private to OSInspection, while the existing public Inspection interface is injected through NewWithInspection and the command's service constructor. Added positivePID's signed-32-bit limit after a red test proved a larger PID could truncate to self on Darwin; this is within native boundary validation scope. No other scope change.

### 2026-10-14 — Linux acceptance limitation

Hub explicitly accepted portable Linux adapter fixtures and cross-build, with native Linux execution untested: the repository has no Linux test runner. Do not start colima or create a runner for this issue. This supersedes the earlier native-Linux-CI verification expectation; Darwin native plus seatbelt remains required and has passed.

### 2026-10-14 — native ancestry test fixture

The full suite exposed an additional ps-dependent fixture in tests/pair-embedded-runtime-test.sh. Replace its synthetic ancestry with tests/without-ancestors.py, a bounded test-only reparented worker preserving command exit status. This tests the real native ancestry path without a production bypass. Other ps test stubs serve negative subprocess assertions, not launcher process inventory, and remain appropriate.

### 2026-10-14 — complete ancestry fixture sweep

The preceding fixture note omitted cmd/pair-go/main_test.go's public command acceptance helper. Its targeted run reproduced the same inherited-Zellij failure; it now shares the bounded reparented worker and uses native sidecar inspection. That supersedes the claim that only negative ps assertions remained after the shell fix.

### 2026-10-14 — live quiescence terminal-process evidence

Real Zellij quiescence under seatbelt exposed a metadata row persisting after process exit while kern.procargs2 returned EINVAL. EINVAL alone must remain unknown. Extend Process with the native terminal/zombie status from the same metadata read, and omit only confirmed terminated rows from command Snapshot; Table retains them for tree consumers. A controlled exited-but-unreaped child will confirm the cause and pin behavior, with fake native rows covering terminal versus unreadable live evidence. This refines the existing disappeared-process contract, not zellij authorization.

### 2026-10-14 — BR-1: test helper owns failure cleanup

The close reviewer reproduced a surviving grandchild after the reparented command timed out. The helper must terminate its private process group before returning on timeout or supervisor cancellation (SIGTERM/SIGINT), including startup races. Successful commands retain their existing sidecar lifetime semantics. Add controlled descendant regressions for timeout and cancellation plus status preservation; this is a test-fixture lifetime correction under ARCH-ORDER/ARCH-FUNERAL.

### 2026-10-14 — BR-2: synchronize terminal fixture metadata

The prepared review measured command EINVAL while Darwin P_stat was still 2. The test fixture must wait with a bounded deadline for native Table metadata to mark the child terminal, keeping it unreaped until cleanup; command failure alone cannot establish fixture readiness. Add deterministic ordering regression plus repetition/concurrent-load checks. This corrects test synchronization under ARCH-ORDER without changing production behavior.
