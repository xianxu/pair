---
collaboration-mode: discussion
---

# Terminal owner role implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Follow the approved plan through SDLC's plan-quality and close review boundaries.

**Goal:** A wrapper that execs a native shell loses its live Pair-terminal role while healthy Pair terminals retain shortcuts.

**Architecture:** Keep the existing append-only registration format. Treat a registered PID as a live terminal owner only while an identity-fenced native argv read identifies the running Pair terminal command. Make registered ownership mandatory for the right-terminal role in the routing projection; startup templates and titles remain layout hints, never sufficient live routing authority.

**Tech stack:** Go, existing procutil.Source native reader seam, workbenchshortcut role projection, installed Zellij 0.45.1 and isolated PTY fixtures.

## Evidence and scope

The private native capture is in the issue's side folder, metadata-capture.json. A controlled executable named pair term used the production registry and role functions, then gated syscall.Exec into /bin/sh. PID31490 and its birth time survived; terminal_command remained the original pair term template, title remained terminal, and pane_command changed to sh -i. LiveIDs and both role projections still admitted RightTerminal. The shell accepted a command. Exact session cleanup returned0; the private namespace had no sessions and all recorded owned processes were gone. This measures native metadata behavior, not production recovery fault handling.

The termcmd cached-draft fast path trusts registry membership without a pane listing. A pane_command-only veto misses it. All registry readers are enumerated below, including Couch's direct registry consumer. No new process-table scan, Zellij settings resolver, registry rewrite, lease artifact or background poller is proposed.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| Terminal owner argv predicate | cmd/internal/workbenchshortcut/terminal_owner.go | new |
| Routing role projection (RoleForPaneWith) | cmd/internal/workbenchshortcut/shortcut.go | modified |

The argv predicate accepts the supported Pair executable basename and the actual terminal-wrapper invocation, rejects shells, substring matches and shortcut/help invocations. Its exact accepted grammar must be checked against termcmd.RunWithRuntime before implementation. It receives typed argv, not a space-joined command string (ARCH-PURE, ARCH-SECURE).

RoleForPaneWith preserves plugin/agent/draft/review precedence but grants RightTerminal only to an ID from confirmed live registrations. Missing registration or an unreadable owner never inherits authority from a title or startup template. RoleForPane remains the existing metadata classifier for layout shape and left-pane discovery; its right-terminal result is explicitly a hint. Audit every direct call: launcher.ClassifyLiveLayout describes shape; draft/clip/layout callers select left panes; Couch already refuses an unregistered right-terminal hint.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Native terminal owner observation | cmd/internal/workbenchshortcut/terminal_owner.go | new | injected procutil.Source Identity/Command reads |
| Registry owner filter (LiveIDs and env adapter) | cmd/internal/workbenchshortcut/shortcut.go | modified | existing append-only registry and supplied owner predicate |
| Runtime owner wiring | termcmd/run.go; layoutcmd/layoutcmd.go; clipcmd/runtime.go; couchcmd/shortcut_focus.go | modified | the shared owner observation |
| Native exec conformance | cmd/internal/workbenchshortcut/terminal_owner_live_test.go | new | private Zellij/PTY, controlled wrapper and shell |

Use Source.Identity before and after Source.Command; absent/changed identity or unreadable argv yields no ownership. This is a point-in-time observation, not a permanent lease. Reuse the existing native seam; the stateful fake changes a process from wrapper to shell under the same PID/birth, and separately models disappearance, replacement and unreadability. Table must never be called. Do not log argv or environments.

LiveIDs receives an ownership predicate instead of a liveness-only predicate, caches its result once per PID within one read, and retains its existing ordering/deduplication behavior. Centralize native construction so the four production runtime paths cannot retain kill-only semantics (ARCH-DRY). Reader tests use portable private registry files; unknown records remain untouched because no rewrite occurs (ARCH-EVOLVE).

## Alternatives

1. Retire a row before exec: requires a stable lock shared by append and atomic rewrite, unknown-line preservation, and an answer when bookkeeping fails. It still cannot make the shell depend on a successful write, so this alone does not fulfill the purpose.
2. A held lifetime lease: provides stronger producer evidence, but the current two-field protocol carries no lease capability. Adding one requires a new artifact/protocol, cleanup and legacy-reader policy. That is a larger change than reusing the native per-PID owner seam for this role bug.
3. **Recommended:** strengthen the existing registry observation and require that ownership in the routing projection. It covers old registrations as well as new ones without migration. Native observation already exists; no new durable resource is needed (ARCH-FUNERAL).

## Operating envelope and states

This is shortcut admission and optional focus observation, not a batch service. N is the registry's existing row count; U is its distinct valid PIDs. Read once, O(N) parsing, at most U command reads and 2U identity reads; no subprocesses, full process enumeration or additional Zellij query. The existing registry growth policy is unchanged. A counted fake test at N and2N, including duplicate rows and malformed records, guards the cost. Existing admission contexts and native IO semantics remain; no hard wall-clock latency guarantee is invented (ARCH-CONSTRAINTS).

Actions: use the working shell; retry an observation; explicitly relaunch a Pair terminal.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|--------------------------|---------------|-----------------|---------------|
| Registered native owner is a live Pair wrapper | Existing Pair terminal | Use existing shortcuts | None |
| Owner execed a shell, exited, changed identity, or argv cannot be read | No confirmed Pair-terminal role | Explicitly relaunch a Pair terminal if Pair shortcuts are wanted | No automatic loss; shell remains usable |
| Registration unavailable/failed | Existing registration warning; no confirmed Pair-terminal role | Explicitly relaunch a Pair terminal after storage recovers | No automatic loss; no shell is withheld |

Covered: native exec with unchanged PID; stale templates/titles; missing-command split panes; registry-read and native-read failures; all role consumers and cached fast paths. Not covered: hostile same-user process/registry forgery, atomic authorization across an arbitrary later exec, or redesigning registry storage/compaction. This adds no state between observations: source state is observed afresh per registry read; the fake controls exec/replacement ordering within a read and separate reads (ARCH-ORDER).

## Tasks (one close boundary)

- [ ] Add failing pure argv/role tests and stateful owner/registry tests. Cases: same PID/birth wrapper→shell; command read and identity read failures; identity replacement during argv read; stale title/startup command; registered split with missing native metadata; foreign pane, plugin and other Pair roles; duplicate and unknown records. Add N/2N counters with zero Table calls. Run focused tests and preserve the initial failure.
- [ ] Implement the shared predicate and native owner adapter, then wire termcmd, layoutcmd, clipcmd and Couch. Update RoleForPaneWith's right-terminal authority rule and comments. Exercise actual termcmd cached-draft routing and Couch focus projection: shell transition removes membership, prevents the fast path and prevents Pair terminal effects. Healthy owner remains functional; registration failure stays visible and shell fallback remains usable. Add a focused repository guard for the mechanical kill-only registry-reader mistake, citing #731; do not alter upstream lint/gate configuration.
- [ ] Add isolated native exec conformance using the same registry/owner/role path: preserve PID, capture before/after startup command/title/pane_command, verify shell input, active-wrapper role and post-exec no role. Include a second active registered pane and unknown registry lines; the same file stays unchanged across reads. Reuse the existing owned PTY/session cleanup patterns, private HOME/socket/config/data/runtime roots, installed binary and no operator session. Keep the controlled-native evidence distinct from unit coverage of production recovery and bookkeeping failures. Qualify layout, clipboard, terminal and Couch consumers plus race tests; no operator UI or :0 staging without hub approval.
- [ ] Update atlas/README only where the role/recovery contract is exposed, audit changed test readers using the new preflight, run targeted repolint/artifactpath checks, push and run the recorded affected/full check required by current gates. SDLC close owns the fresh review; enter normal merge on SHIP and report the verified landing. Required live acceptance is the disposable native conformance unless the hub requests an additional operator check.

## Risky function strategies and guards

| Function/seam | Adversarial strategy | Independent guard |
|---------------|----------------------|-------------------|
| Terminal argv predicate | Generated argv/path/extra-flag variants | Exact wrapper accepted; shells and substring/helper commands rejected |
| Native owner observation | Stateful exec/reuse/error during command read | Failed/changed evidence never returns an owner; no Table call |
| LiveIDs | Duplicate/malformed/future rows with N/2N live and dead PID sets | Foreign/unknown bytes unchanged; once-per-PID proof; stable ID order |
| RoleForPaneWith | Cross-product of role metadata, plugin flag and membership | Stale Pair-looking metadata alone never grants RightTerminal |
| focusedWorkbenchPanesContext and shortcutFocusProbe | Production admission before/after fake exec | No cached fast path or right-terminal effect after loss; other roles preserved |
| Native exec fixture | Real foreground process handoff and shell command acknowledgement | PID unchanged, command changed, role removed, second owner preserved, exact cleanup |

## Approval

Proposed after the native measurement; awaiting hub review before change-code. If owner argv grammar, cost or consumer audit requires a broader mechanism, stop and revise this plan rather than silently widening it.
