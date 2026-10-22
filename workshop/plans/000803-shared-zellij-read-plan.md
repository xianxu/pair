---
collaboration-mode: discussion
---

# Shared Go Zellij reads implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Execute the extraction in the main session; SDLC owns the fresh boundary review.

**Goal:** Give every Go production Zellij query the same bounded compatibility read, beginning with layout's failing native admission.

**Architecture:** Extract #748's read transport and classification from launcher into independent `cmd/internal/zellijquery`. Retain launcher aliases/wrappers for its public evidence/error API and existing tests. Every Go query consumer uses that seam; mutations and deliberately raw diagnostic experiments retain their own execution paths.

**Tech Stack:** Go contexts, os/exec, existing joined zellijexec collector and strict zellijreply decoders. No dependency or persisted-format changes.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Read kind and retry classification | `cmd/internal/zellijquery/recovery.go` | modified (extracted from launcher) |
| Strict session reply validation | `cmd/internal/zellijquery/session_reply.go` | modified (moved from launcher with compatibility wrappers) |
| Native socket-root spelling | `cmd/internal/zellijquery/socket.go` | modified (moved from launcher with compatibility wrappers) |

One argv shape selects one expected reply kind. Only a complete validated
different kind qualifies as crossed; malformed same-kind payloads remain the
caller's strict refusal. The session validator is moved, not rewritten. Existing
#761 full-diagnostic, exact target and socket predicates remain one source.
This serves ARCH-DRY and ARCH-PURPOSE without a launcher/titlepoller cycle.

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| Query source and attempt observation | `cmd/internal/zellijquery/query.go` | modified (extracted) | zellijexec.Run, exec.Cmd, caller context |
| Socket presence | `cmd/internal/zellijquery/recovery.go` | modified (extracted) | exact Lstat; preserves native path spelling |
| Go consumer adapters | launcher/layoutcmd/clipcmd/couchcmd/opener/titlepoller/wrapcmd | modified | shared source, existing caller-specific strict parsing |

Source supports the existing injectable process runner, executable path and
context-scoped observer. Each returned attempt result retains stdout, stderr and
the original process/collector error separately from the read-classification
error. This is required by the session-name probe: direct native Clap/absence
diagnostics may prove a length bound, while joined collection failures cannot.
The ordinary launcher adapter retains its existing error wrapping and aliases.
Do not add a caller-customizable retry predicate or a second retry loop.

## Operating contract

- A logical read has at most three sequential attempts in one original 5s cap,
  preserving any earlier parent deadline. No backoff and no deadline refresh.
- Allowlist actual production read shapes: full unformatted session listing;
  explicit/implicit-session clients; JSON panes with supported command/state/
  geometry/tab flags; JSON current-tab info. Reject unknown/duplicate flags from
  retry eligibility. A non-allowlisted query is collected once.
- Empty/whitespace successful replies and validated crossed kinds retry.
  Keep #761's exact false-miss classification for explicit session reads.
  Arbitrary stderr, nonzero errors and collector uncertainty do not become
  generic recovery. Only #761's already-proven diagnostic exception retries.
- dump-screen may validly be empty: route it through shared bounded collection
  but do not classify/retry screen text. Failure-only version reads and raw
  reproduction/latency probes are diagnostic exclusions, not decision consumers.
- zellijexec.Run remains the one-attempt joined process-group collector.
  Cancellation stops attempts and joins the current subprocess before return.
- Confirm still requires two independently valid agreeing observations. Each
  observation may recover inside its own read; no mutation or comparison is retried.
- No durable state or background worker is created. Buffered outputs and attempt
  metadata die with the read. Existing bounded observer snapshots stay 64KiB per
  stream. This change adds no on-disk format, cache, ownership or cleanup policy.

Actions: invoke operation again, inspect native session.

| Failure / observation | Visible state | Recovery | Possible loss |
|---|---|---|---|
| Recoverable reply followed by valid reply | existing success | none needed | no decision from the discarded observation |
| Three unanswered/crossed replies | existing refusal / unavailable | invoke operation again | requested action has not been authorized |
| Malformed data, collector error or cancelled budget | existing refusal / unavailable | inspect native session | no new mutation authority |

The read holds no authority between calls. Native output is untrusted and strict
caller parsers remain mandatory (ARCH-SECURE). No absent/mismatched observation
can establish ownership. There is no lock that can make a native reply reliable;
the existing identity and independent-confirmation fences remain in the callers.

Cost: N logical reads make N attempts when healthy, at most 3N under the known
bug; two-observation confirmation makes at most 6N attempts. Parse/byte work is
linear in bytes per attempt. Count N=8/16 for healthy and recovery schedules;
timing is conformance evidence, not the complexity defense.

## Task 1: Reproduce and extract the shared owner

Files: new `cmd/internal/zellijquery/{query,recovery,session_reply,socket}.go`
and corresponding tests; adapt launcher `zellij.go`, `zellij_read_recovery.go`,
`session_reply.go`, `zellij_socket.go` to retain aliases/forwarders.

- [ ] Add a stateful executable-fixture regression through layout's actual OS
  reader: successful empty reply, valid pane listing, then independent valid
  confirmation. Run it before implementation and retain the expected failure.
- [ ] Extract existing process/evidence/classification code without changing
  session grammar or socket path semantics. Add supported implicit-session,
  geometry/tab and current-tab read shapes. Preserve existing public errors,
  observer fields and launcher tests through aliases/forwarders.
- [ ] Pin pure classification cases: all allowed/unknown argv, complete wrong
  kind, malformed same kind, exact diagnostics and unsafe socket spelling.
- [ ] Pin actual collector schedules: empty/crossed then success, three-attempt
  exhaustion, direct exit, joined collection failure, arbitrary stderr,
  cancellation between attempts, identical effective deadlines and one event
  per attempt. Reuse the shared Zellij reply serializer/catalog for new producers.
- [ ] Run `go test ./cmd/internal/zellijquery ./cmd/internal/launcher -run 'Zellij|SessionReply|SessionListing|Socket' -count=1` and commit the extraction.

## Task 2: Route every in-scope Go consumer

Files: layoutcmd/layoutcmd.go, clipcmd/runtime.go,
couchcmd/shortcut_focus.go, opener/runtime.go, titlepoller/runtime.go,
wrapcmd/peer_viewport.go, launcher/osruntime.go and their colocated tests.

- [ ] Replace direct query execution with the shared source. Preserve parent
  contexts, raw outputs/error diagnostics, last-known-state behavior and caller
  parsers; do not add independent retries to Confirm or ParsePanes.
- [ ] Preserve launcher name-probe interpretation of original direct-exit
  diagnostics, and refusal for empty/unknown/collector failures after recovery.
- [ ] Exercise layout panes and current-tab plus clip/focus/opener/liveness and
  name-probe adapters against stateful fixture schedules; retain existing native
  fixture authorities and explicit private environments.
- [ ] Count work at 8/16 observations, including two-confirmation success,
  exhaustion with zero effects, and cancellation with no later process spawn.
- [ ] Sweep all production Go exec calls; leave mutations and raw diagnostics
  excluded explicitly. Update Pair lint to reject new direct query bypasses
  outside the shared owner and qualified diagnostics; test positive/negative
  syntax cases and document syntax-analysis limits.

## Task 3: Qualify and land atomically

Files: atlas/zellij-reply-safety.md, atlas/go-lint-catalog.md,
artifactpath source inventory, fixed affected-reader certificates if required,
this plan and the issue Log.

- [ ] Map the new owner and remaining Lua follow-up explicitly in the atlas.
  Classify all new production files and register actual reader-test inputs.
- [ ] Run affected packages and races for the shared seam/adapters, full
  artifactpath, repolint, reader inventory, and shared Zellij fixture decoder
  preflights. No assertions/deadlines are weakened to get a pass.
- [ ] Run the existing disposable native terminal-recovery/layout conformance
  entrypoint against the installed binary and retain actual attempts/evidence.
  Never touch the live Couch fleet or :0; #743's hosted CI is the external check.
- [ ] Push via sdlc pr; run the recorded branch suite via local-check. Record
  all failures and classify them under normal policy, without blind retries.
- [ ] Close once via SDLC's mandatory fresh review; resolve blockers, then enter
  sdlc merge --yes and let it validate the integrated head. Report PR/merge pin
  to the hub. #759 operator acceptance can interrupt this work when called.

## Revisions

2026-10-22: TL a27b640f approved Go-only scope and full flow. Lua consumers are
split into an unclaimed dependent issue using a later thin CLI adapter. No Go
consumer is deferred merely to retain the initial quick-flow size assumption.
