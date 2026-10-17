---
collaboration-mode: discussion
---

# Background work display implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Execute the session-warm protocol and display changes with superpowers-executing-plans; bounded read-only investigations may use subagents. Steps use checkbox syntax.

**Goal:** Keep a slot bright while its agent reports running background shells, while its free composer continues receiving messages.

**Architecture:** Extend the existing wrapper session with negotiated display-only background evidence. Classify the wrapper's already-owned terminal snapshot on its existing settle callback; retain evidence only within the admitted session and reuse #625's activity projection.

**Tech Stack:** Go, existing terminal emulator, strict versioned Unix session protocol, pure registry reducer, existing Console activity pass.

## Approval decisions

1. Approve the new nullable `BackgroundWork` field and hello-v4 negotiation below.
2. Approve initial native evidence coverage: Claude's captured shell footer. Codex/Qoder currently have no positive background-job capture or detector; return unknown there, preserve existing turn fading, and document this gap rather than infer job syntax. If those agents must be covered in this issue, gather native evidence and revise this plan before coding.
3. Recommended full flow for the cross-process protocol addition (likely over 100 added production lines); request TL approval to select it explicitly. No estimate until plan-quality passes.
4. Confirm live rollout route. The earlier no-pair:0-staging direction was for #625; this issue needs either authorized staging or an explicit post-landing live deferral. Do not touch :0 without that decision.

## Evidence and alternatives

`SessionFrame`, `SlotLiveness` and the detached Console snapshot have no background-work field. `handleTail` is an on-demand endpoint call, so polling it adds IO. `Settled` grants admission and must remain true with a free composer after a turn ends. Recorded Claude 2.1.296 `shell-footer.raw` and `idle-suggestion.raw` show an empty composer and a native `1 shell` footer with `↓ to manage`; body text saying “1 shell still running” is not authoritative.

Recommend one display field over on-demand tail polling (new work/IO for detached slots) or process/CPU inference (queued suites may be silent and CPU-free; child ownership differs). No new timer, filesystem source, process scan, endpoint, or persisted cache. The session connection itself owns evidence lifetime; no extra lease or expiry timer is needed.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| Background-work observation: unknown, known none, known active | `cmd/internal/wrapcmd/peer_composer.go` | new |
| SessionFrame hello-v4 and optional BackgroundWork | `cmd/internal/couchmessage/session_protocol.go` | modified |
| Registry session background evidence and LastWorkAt transition | `cmd/internal/couchmessage/registry.go` | modified |

The pure detector consumes one complete current terminal snapshot, returns a nullable boolean, and reuses `ruledBoxComposerBounds` with `claudeComposerSpec`. Count only a complete native shell counter in dot-separated footer fields below the closing composer rule, with matching native management chrome. Require a bounded nonnegative decimal count and correct singular/plural field. Inspect no body, draft, history, assistant completion prose or arbitrary substring. Recognized zero/no-shell native footer means false; incomplete, unsupported or unqualified footer means unknown. Do not infer Codex/Qoder syntax.

The registry remains the single reducer for session work evidence. Background evidence and settled admission are independent: all combinations are legal because an open turn and background shell can coexist, and a closed turn may have a running shell. Extend the existing transition rather than introduce a parallel clock owner.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Existing settle callback and generation fence | `cmd/internal/wrapcmd/peer_settle.go`, `peer_runtime.go`, `peer_delivery.go` | modified | current terminal model and existing timer |
| SessionClient/server transport | `cmd/internal/couchmessage/session_transport.go` | modified | existing authenticated connection and coalescing |
| Admitted wrapper activity | `cmd/internal/couchcmd/message_service.go` | modified | existing atomic SlotLiveness snapshot |
| Console activity tests | `cmd/internal/couchtty/console_activity_test.go` | modified | existing host, activity pass, injected clock |

Use recorded VT replay/stateful terminal model, real Unix session fixtures and the existing waitRig/Console fixtures. No new external binary dependency or new subprocess fixture is needed. Any necessary Pair subprocess must use `testpair.Environment`.

## Protocol, ordering and lifetime

- New client tries hello-v4, v3, v2, hello in that order; only a confirmed pre-ack close permits downgrade. Explicit refusal, malformed ack or timeout does not silently downgrade. Existing behavior otherwise remains unchanged.
- Only v4 activity/submit frames may carry BackgroundWork. Reject it on hello/ack and on older sessions. V3 keeps Picker, V2 keeps Settled, legacy stays unknown. Strict decode and size limit remain enforced.
- SessionClient copies nullable values under its existing mutex, coalesces changes and resends the current evidence after reconnect. No frame on unchanged evidence and no new heartbeat. The field is independent from Observation.LastActivity, submission counts, Picker and Settled.
- Classify after terminal Feed, on the existing settle callback, outside the delivery mutex. Capture/recheck the existing settle generation around the probe; stale callbacks cannot publish. The callback must publish background changes even when Settled remains true (today it returns early in that case). Pending output or incomplete terminal state produces unknown, clearing a cached positive.
- On the same admitted session: active makes display current; active→known-none records completion at broker receipt time, monotonically; unknown removes the active hint without inventing completion; unchanged known-none refresh does not renew work. Existing #625 turn transitions/seed remain intact.
- Binding scope/tag/agent must match the row. Disconnect, displacement and replacement drop the old session's display hint. A replacement with identical binding receives no inherited hint. Rejected/dormant sessions do not brighten anything.
- Consumer brightness is current if known-unsettled OR known-background-active. It never modifies Settled or admission. When both finish, use LastWorkAt and normal aging.

## Operating envelope and principles

Existing 3s settle debounce plus 1s transport throttle and 60s Console pass bound normal display convergence (existing budgets, not new timers). Work is at most one scan of the bounded current viewport per settled output episode, no transcript scans and no paint/input-path IO. Boolean evidence is O(1) per bounded registry session. Native footer accuracy is an explicit assumption: it reports native background-job state until redrawn. Unknown input stops positive renewal. No durable artifacts, keys or credentials are created.

ARCH-DRY: reuse composer geometry, settle generation, strict protocol fallback and #625 clock. ARCH-PURE: keep footer classification and registry transitions pure. ARCH-PURPOSE: test brightness plus successful message admission, disclose unsupported agents. ARCH-MOCK: recorded stateful VT/real transport fixtures and installed live conformance. ARCH-CONSTRAINTS: existing cadence and viewport/frame bounds. ARCH-SECURE: exact admitted incarnation and strict negotiated fields; transcript/body text grants no evidence. ARCH-ORDER: generation-fenced classification, ordered session frames and explicit transitions above. ARCH-FUNERAL: evidence dies with the existing connection; no new durable residue.

## Atomic implementation tasks (one close review)

### Task 1 — native display evidence and timer publication

Files: `cmd/internal/wrapcmd/peer_composer.go`, `peer_settle.go`, `peer_runtime.go`, `peer_delivery.go`; tests `peer_background_test.go`, `peer_settle_test.go`, existing recording sink tests.

- [ ] `backgroundWorkState(agent, terminalSnapshot) *bool` (new pure detector): recorded94×39 native screens seed malformed-footer/geometry fuzzing; moving a purported counter outside qualified footer geometry must never produce positive work. Run `go test ./cmd/internal/wrapcmd -run 'Test.*Background|TestRecordedIdleScreensReadDeliverable' -count=1 -timeout 2m`; require behavioral RED before implementation.
- [ ] Implement the pure detector and wire a separate display probe into the existing settle callback. Deterministically hold a probe while output changes to prove stale generation rejection; verify true→false/unknown changes publish even with Settled continuously true. Preserve existing idle/admission tests.
- [ ] Run focused GREEN and race tests, then checkpoint the complete evidence slice.

### Task 2 — negotiated transport, admitted registry and fade clock

Files: `cmd/internal/couchmessage/session_protocol.go`, `session_transport.go`, `registry.go`; tests their colocated protocol/transport/registry files; `cmd/internal/couchcmd/message_service.go`, `idle_activity_test.go`; `cmd/internal/couchtty/console_activity_test.go`.

- [ ] `SessionFrame.Validate`, `DecodeSessionFrame`, `SessionClient.BackgroundWork` and `SessionClient.session`: use the real socket fixtures with strict older-schema broker doubles and interrupted handshakes; assert negotiated field ownership and latest-state delivery across reconnection, without silently downgrading uncertain outcomes. Require behavioral RED before implementation.
- [ ] `Registry.Advance` and `messageService.WrapperActivity`: table-driven event sequences with an injected broker clock independently assert incarnation isolation, monotonic completion and no renewal from idle/unknown evidence; disconnected/detached/replaced identities must obey the existing admission boundary. Require behavioral RED before implementation.
- [ ] Console activity pass: reuse the real Console/host fixture with an injected clock and old-file baseline; positive settled background work must defeat fade without manufacturing new file activity, and loss/completion of evidence must restore ordinary aging. Keep a successful free-composer delivery integration as the admission oracle.
- [ ] Implement the new field through the production path and registry transition. Keep message admission unchanged; integration test successfully reserves/commits/submits a peer message through a free composer whose background flag remains active.
- [ ] Run `go test ./cmd/internal/couchmessage ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/wrapcmd -count=1 -timeout 5m` and focused `-race` tests. Measure changed-source artifact inventory if any new production file becomes necessary. Commit the working slice.

### Task 3 — documentation and acceptance

Files: `README.md`, `atlas/couch.md`, issue Log; no project-file edits (TL owns it).

- [ ] Describe independent fade evidence, compatibility, supported native footer and unsupported-agent gap in README/atlas. Record red/green and concurrency outcomes and any approved scope revisions.
- [ ] Push candidate and run `sdlc local-check run --name suite` to completion; keep queue/run time separate. Ordinary `sdlc close` owns the fresh review; fix blockers before PR/merge. Integrate main and land with normal `sdlc merge --yes` gates.
- [ ] Follow the TL-approved live route: existing old-file-activity Claude slot with ended turn, native background suite running, unfocused tab and switcher bright within existing pass; successfully deliver a peer request while the suite remains running. Use actual old-file evidence, not touched mtimes or a newly born pane as the only positive control. Compare idle old row, then completion and normal aging. Record exact installed build and message receipt. Do not stage/restart :0 without TL authorization; an approved post-landing live check remains explicitly pending until performed.

## Revisions

2026-10-16: initial plan proposal; TL approval is pending. No implementation performed.

### 2026-10-16 — TL approval a807af4c

All four decisions resolved: design approved, native detection Claude only (Codex/Qoder stay unknown), full flow explicitly selected, installed live check deferred until the restart after landing, no staging in pair:0. Earlier questions above are preserved as the design record, superseded by this decision. Additional live Codex counter evidence is documented in the issue with the capture/qualification work needed before adding that source. Proceed to the normal full-flow change-code gate.

### 2026-10-16 — PQ-1 test-strategy refinement

Plan-quality accepted the design but required function-level testing strategies. Replaced the case inventories with named detector, protocol, client, registry and activity surfaces plus seeded fuzzing, controlled generation interleaving, real transport and clock-driven integration strategies. The `peerDelivery.settleFired` callback is verified with the existing fakeTimers scheduler; `proxy.backgroundWorkNow` is verified through the stateful terminal Feed/Snapshot seam. No state, field, scope, cadence or rollout change.
