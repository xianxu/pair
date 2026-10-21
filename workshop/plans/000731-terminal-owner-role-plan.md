---
collaboration-mode: discussion
---

# Terminal owner role implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Follow the approved plan through SDLC's plan-quality and close review boundaries.

**Goal:** A wrapper that execs a native shell loses its live Pair-terminal role while healthy Pair terminals retain shortcuts.

**Architecture:** Keep the existing append-only registration format and liveness answer. An identity-fenced native argv read can confirm a Pair wrapper or confirm a different command; unavailable inspection retains the existing live registration and records why. Make this filtered registration mandatory for the right-terminal role in the routing projection; startup templates and titles remain layout hints, never sufficient live routing authority.

**Tech stack:** Go, existing procutil.Source native reader seam, workbenchshortcut role projection, installed Zellij 0.45.1 and isolated PTY fixtures.

## Evidence and scope

The private native capture is in the issue's side folder, metadata-capture.json. A controlled executable named pair term used the production registry and role functions, then gated syscall.Exec into /bin/sh. PID31490 and its birth time survived; terminal_command remained the original pair term template, title remained terminal, and pane_command changed to sh -i. LiveIDs and both role projections still admitted RightTerminal. The shell accepted a command. Exact session cleanup returned0; the private namespace had no sessions and all recorded owned processes were gone. This measures native metadata behavior, not production recovery fault handling.

The termcmd cached-draft fast path trusts registry membership without a pane listing. A pane_command-only veto misses it. All registry readers are enumerated below, including Couch's direct registry consumer. No new process-table scan, Zellij settings resolver, registry rewrite, lease artifact or background poller is proposed.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| Terminal owner argv predicate | cmd/internal/workbenchshortcut/terminal_owner.go | new |
| Terminal owner observation outcome | cmd/internal/workbenchshortcut/terminal_owner.go | new |
| Routing role projection (RoleForPaneWith) | cmd/internal/workbenchshortcut/shortcut.go | modified |

The argv predicate accepts the supported Pair executable basename and the actual terminal-wrapper invocation, rejects shells, substring matches and shortcut/help invocations. Its exact accepted grammar must be checked against termcmd.RunWithRuntime before implementation. It receives typed argv, not a space-joined command string (ARCH-PURE, ARCH-SECURE).

RoleForPaneWith preserves plugin/agent/draft/review precedence but grants RightTerminal only to an ID retained by the live-registration projection. Missing registration never inherits authority from a title or startup template. An unreadable owner keeps the previous liveness-based registration policy, explicitly distinct from a confirmed different command. RoleForPane remains the existing metadata classifier for layout shape and left-pane discovery; its right-terminal result is explicitly a hint. Audit every direct call: launcher.ClassifyLiveLayout describes shape; draft/clip/layout callers select left panes; Couch already refuses an unregistered right-terminal hint.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Native terminal owner observation | cmd/internal/workbenchshortcut/terminal_owner.go | new | injected procutil.Source Identity/Command reads |
| Registry owner filter (LiveIDs and env adapter) | cmd/internal/workbenchshortcut/shortcut.go | modified | existing append-only registry and supplied owner predicate |
| Runtime owner wiring | termcmd/run.go; layoutcmd/layoutcmd.go; clipcmd/runtime.go; couchcmd/shortcut_focus.go | modified | the shared owner observation |
| Owner inspection diagnostics | cmd/internal/workbenchshortcut/terminal_owner.go | new | existing scoped AdaptLog and diagnosticlog retention |
| Native exec conformance | cmd/internal/workbenchshortcut/terminal_owner_live_test.go | new | private Zellij/PTY, controlled wrapper and shell |

Use Source.Identity before and after Source.Command. The typed outcome separates confirmed wrapper, confirmed different command, and inspection unavailable, with bounded reason codes. A confirmed changed identity rejects the old observation; an unavailable identity/argv read does not prove a mismatch and retains today's liveness answer. This is a point-in-time observation, not a permanent lease. Reuse the existing native seam; the stateful fake changes a process from wrapper to shell under the same PID/birth, and separately models disappearance, replacement and unreadability. Table must never be called. Do not log argv or environments.

Aggregate unavailable reasons once per registry read and append at most one bounded event to the existing scoped adaptation log. Reuse diagnosticlog's managed retention and explicit scoped root/tag (including Couch's target), rather than ambient host state or terminal stderr. Logging is best effort: its failure never changes role or shell admission. The event contains reason categories/counts, not native argv/error strings; no new artifact family, background worker or unbounded history is introduced.

LiveIDs receives an ownership predicate instead of a liveness-only predicate, caches its result once per PID within one read, and retains its existing ordering/deduplication behavior. Centralize native construction so the four production runtime paths cannot retain kill-only semantics (ARCH-DRY). Reader tests use portable private registry files; unknown records remain untouched because no rewrite occurs (ARCH-EVOLVE).

## Alternatives

1. Retire a row before exec: requires a stable lock shared by append and atomic rewrite, unknown-line preservation, and an answer when bookkeeping fails. It still cannot make the shell depend on a successful write, so this alone does not fulfill the purpose.
2. A held lifetime lease: provides stronger producer evidence, but the current two-field protocol carries no lease capability. Adding one requires a new artifact/protocol, cleanup and legacy-reader policy. That is a larger change than reusing the native per-PID owner seam for this role bug.
3. **Recommended:** strengthen the existing registry observation and require that ownership in the routing projection. It covers old registrations as well as new ones without migration. Native observation already exists; no new durable resource is needed (ARCH-FUNERAL).

## Operating envelope and states

This is shortcut admission and optional focus observation, not a batch service. N is the registry's existing row count; U is its distinct valid PIDs. Read once, O(N) parsing, at most U command reads and 2U identity reads, plus the existing per-PID liveness check; no subprocesses, full process enumeration or additional Zellij query. At most one existing managed diagnostic event is appended per read when inspection was unavailable; its bounded category counts do not grow with argv size. The existing registry growth policy is unchanged. A counted fake test at N and2N, including duplicate rows and malformed records, guards the cost. Existing admission contexts and native IO semantics remain; no hard wall-clock latency guarantee is invented (ARCH-CONSTRAINTS).

Actions: use the working shell; retry an observation; explicitly relaunch a Pair terminal.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|--------------------------|---------------|-----------------|---------------|
| Registered native owner is a live Pair wrapper | Existing Pair terminal | Use existing shortcuts | None |
| Owner execed a shell, exited, or changed identity | No confirmed Pair-terminal role | Explicitly relaunch a Pair terminal if Pair shortcuts are wanted | No automatic loss; shell remains usable |
| Live registered PID's identity/argv inspection is unavailable | Existing Pair-terminal role retained; reason recorded in the adaptation log | Retry observation after process inspection recovers | Ownership can remain stale while inspection is unavailable; accepted compatibility tradeoff |
| Registration unavailable/failed | Existing registration warning; no confirmed Pair-terminal role | Explicitly relaunch a Pair terminal after storage recovers | No automatic loss; no shell is withheld |

Covered: native exec with unchanged PID; stale templates/titles; missing-command split panes; registry-read and native-read failures with their distinct policies; all role consumers and cached fast paths. Not covered: hostile same-user process/registry forgery, atomic authorization across an arbitrary later exec, or redesigning registry storage/compaction. A fallback shell whose process inspection is unavailable can retain the old registry answer; the hub explicitly accepts that compatibility limit so sandbox restrictions do not break healthy panes. This adds no authoritative state between observations: source state is observed afresh per registry read; the fake controls exec/replacement ordering within a read and separate reads (ARCH-ORDER).

## Tasks (one close boundary)

- [x] Add failing pure argv/role tests and stateful owner/registry tests. Cases: same PID/birth wrapper→shell denies; unavailable command/identity reads retain the old liveness answer with a bounded diagnostic reason; identity replacement during argv read; stale title/startup command; registered split with missing native metadata; foreign pane, plugin and other Pair roles; duplicate and unknown records. Add N/2N counters with zero Table calls and at most one diagnostic event. Run focused tests and preserve the initial failure.
- [x] Implement the shared predicate and native owner adapter, then wire termcmd, layoutcmd, clipcmd and Couch. Update RoleForPaneWith's right-terminal authority rule and comments. Exercise actual termcmd cached-draft routing and Couch focus projection: shell transition removes membership, prevents the fast path and prevents Pair terminal effects. Healthy owner remains functional; registration failure stays visible and shell fallback remains usable. Add a focused repository guard for the mechanical kill-only registry-reader mistake, citing #731; do not alter upstream lint/gate configuration.
- [x] Add isolated native exec conformance using the same registry/owner/role path: preserve PID, capture before/after startup command/title/pane_command, verify shell input, active-wrapper role and post-exec no role. Include a second active registered pane and unknown registry lines; the same file stays unchanged across reads. Reuse the existing owned PTY/session cleanup patterns, private HOME/socket/config/data/runtime roots, installed binary and no operator session. Keep the controlled-native evidence distinct from unit coverage of production recovery and bookkeeping failures. Qualify layout, clipboard, terminal and Couch consumers plus race tests; no operator UI or :0 staging without hub approval.
- [x] Update atlas/README only where the role/recovery contract is exposed, audit changed test readers using the new preflight, run targeted repolint/artifactpath checks, push and run the recorded affected/full check required by current gates. SDLC close owns the fresh review; enter normal merge on SHIP and report the verified landing. Required live acceptance is the disposable native conformance unless the hub requests an additional operator check.

## Risky function strategies and guards

| Function/seam | Adversarial strategy | Independent guard |
|---------------|----------------------|-------------------|
| Terminal argv predicate | Generated argv/path/extra-flag variants | Exact wrapper accepted; shells and substring/helper commands rejected |
| Native owner observation | Stateful exec/reuse/error during command read | Confirmed mismatch rejects; unreadability retains today's answer with reason; no Table call |
| Owner diagnostic adapter | Private managed log with unavailable categories and failed output | At most one bounded event; no argv/environment leakage; failure cannot alter classification |
| LiveIDs | Duplicate/malformed/future rows with N/2N live and dead PID sets | Foreign/unknown bytes unchanged; once-per-PID proof; stable ID order |
| RoleForPaneWith | Cross-product of role metadata, plugin flag and membership | Stale Pair-looking metadata alone never grants RightTerminal |
| focusedWorkbenchPanesContext and shortcutFocusProbe | Production admission before/after fake exec | No cached fast path or right-terminal effect after loss; other roles preserved |
| Native exec fixture | Real foreground process handoff and shell command acknowledgement | PID unchanged, command changed, role removed, second owner preserved, exact cleanup |

## Approval

Hub approved d495d55d-e94e-4033-8b07-6095ee2c618a with the unavailable-inspection compatibility policy below. Proceed to change-code. If owner argv grammar, cost or consumer audit requires a broader mechanism, stop and revise this plan rather than silently widening it.

## Revisions

### 2026-10-20 — approved unavailable-inspection compatibility

Reason: the hub has observed unavailable process inspection inside sandboxed contexts and Zellij servers. Denying every unavailable read would silently break healthy terminals. Delta: preserve three typed observation outcomes; confirmed non-wrapper argv denies, while unavailable identity/argv keeps the existing liveness registry answer and records a bounded reason through existing scoped managed diagnostics. Pin both policies, diagnostic failure and counts in tests. This supersedes the proposed fail-closed unavailable-read behavior, retained in the initial plan commit c4b3b584a. No persisted registry format change, lock, lease or new resolver is added.

### 2026-10-20 — implementation wiring and reader audit

The implementation names the native registry adapter LiveOwnerIDs in terminal_owner.go. LiveIDs remains the storage/predicate seam with per-PID caching. Three runtimes inherit the owner check through LiveTerminalPaneIDsFromEnv, so their runtime source files need no duplicate edits; Couch has an explicit procutil.Source injection seam. The accepted owner invocation is the canonical pair/pair-go term command generated by the launcher; shortcut/help helpers are excluded. Tests consume private registries, owned executable fixtures and literal pane data; the existing term source-scanner and shortcut source-scanner retain their fixed preflight coverage. No additional repository input is introduced by the new owner fixtures. The plan remains one close boundary.
