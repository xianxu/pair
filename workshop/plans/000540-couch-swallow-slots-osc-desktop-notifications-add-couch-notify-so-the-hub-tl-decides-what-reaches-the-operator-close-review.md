# Boundary Review — pair#540 (whole-issue close)

| field | value |
|-------|-------|
| issue | 540 — couch: swallow slots' OSC desktop notifications; add couch --notify so the hub (TL) decides what reaches the operator |
| repo | pair |
| issue file | workshop/issues/000540-couch-swallow-slots-osc-desktop-notifications-add-couch-notify-so-the-hub-tl-decides-what-reaches-the-operator.md |
| boundary | whole-issue close |
| milestone | — |
| window | 407f578d0e1ee78b3b75aeacb26e1f726cec230b..70d3c74ea45cc212deea503934a7a0cac67d50ca |
| command | sdlc close --issue 540 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T20:09:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation correctly separates child suppression from hub-owned output, reuses existing authority checks, and updates user documentation. One correctness bug blocks shipping: a stalled presenter silently drops `--notify` yet reports success. The acceptance criteria also omit the added compatibility behavior. Console and CLI tests could not compile because this checkout lacks generated runtime assets.

Done-when: stale — it covers notification suppression/output, but not the legacy-supervisor compatibility and recovery diagnostics added in Revisions.

1. **Strengths**

   - Child suppression preserves ordinary BEL, title, clipboard, and hyperlink behavior; tests exercise split packets, duplicate delivery, and pane replacement.
   - Hub admission reuses registered caller authority and rechecks identity after workspace lookup (`notify_service.go:33`).
   - Lost-reply coverage executes the service through a private socket and verifies one notification without retry.
   - README and both relevant atlas pages document the new surface, limits, and uncertainty contract.

2. **Critical findings**

   - **Success reported for dropped notifications — ARCH-ORDER / ARCH-PURPOSE.** At [presenter.go:1057](/Users/xianxu/Library/Caches/sdlc/review-checkouts/3a4ac4df71ef6d86/cmd/internal/terminal/presenter.go:1057), `Notify` calls `writeUnlessStalled`. That helper returns `nil` without writing when `p.stalled` is true (`presenter.go:305–307`). Consequently, `handleNotify` returns `ok` and the CLI prints “Notification written” despite emitting nothing. The helper also hides its own write-timeout errors after entering stall recovery. Preserve presenter recovery, but return a non-success notification outcome for dropped or incomplete writes. Add regressions for an already-stalled presenter and a notification write that itself stalls; assert truthful outcomes and no replay after recovery.

3. **Important findings**

   - **Acceptance criteria omit expanded scope — ARCH-PURPOSE.** The issue’s [Done-when criterion](/Users/xianxu/Library/Caches/sdlc/review-checkouts/3a4ac4df71ef6d86/workshop/issues/000540-couch-swallow-slots-osc-desktop-notifications-add-couch-notify-so-the-hub-tl-decides-what-reaches-the-operator.md:36) remains notification-only. The plan’s Revisions at lines 225–243 add compatibility for existing messaging commands, missing-registration diagnostics, and unsupported-notify recovery. Add acceptance criteria for these behaviors, including single-exchange/no-retry guarantees.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed full packages: `notifyosc`, `terminal`, and `couchmessage`.
   - `git diff --check` passed.
   - Couch console/CLI verification failed during setup: `cmd/internal/runtimebundle/embed.go:8:41: pattern assets/runtime/files: no matching files found`. `Makefile.local:249` supplies the generation prerequisite; it was not executed under this read-only review.
   - Existing presenter degradation tests explicitly verify silent dropping through the shared helper. New notification tests cover cancellation and hard failures, but omit this recoverable-stall state.
   - No prior findings required disposition.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared parser, sanitizer, transport, authority, and writer.
   - **ARCH-PURE: pass** — pure parsing/encoding and protocol validation remain separate from integration seams; concept-table classifications match the implementation.
   - **ARCH-PURPOSE: flag** — false notification success and incomplete acceptance criteria, as above.
   - **ARCH-MOCK: pass on inspection** — stateful host/registry fixtures, private socket tests, and PTY/native conformance paths exist; the latter were not independently rerun.
   - **ARCH-CONSTRAINTS: pass on inspection** — explicit payload bounds, bounded transport/presenter machinery, and fixed per-pane counters.
   - **ARCH-SECURE: pass** — server-owned hub authority, sanitized output, and content-free diagnostics.
   - **ARCH-ORDER: flag** — dropped output is collapsed into confirmed success.
   - **ARCH-FUNERAL: pass** — counters expire with panes; diagnostics reuse existing bounded retention.

7. **Plan revision recommendations**

   Add a `## Revisions` entry distinguishing notification acknowledgment from best-effort child effects: stalled/dropped writes must return non-success while retaining recovery and no-replay behavior. Update Done-when to cover the approved mixed-version compatibility additions.

```findings
findings:
  - id: new
    severity: Critical
    family: effect-outcome-truthfulness
    title: |
      Notify reports success when the presenter drops its output during a stall
    detail: |
      cmd/internal/terminal/presenter.go:1057 delegates to writeUnlessStalled, which returns nil without writing when already stalled and also hides recoverable write-timeout errors. The broker consequently returns ok and the CLI claims the notification was written. Preserve presenter recovery but propagate a non-success notification outcome; add stalled-before-notify and stalls-during-notify regressions with no replay after recovery. ARCH-ORDER, ARCH-PURPOSE.
  - id: new
    severity: Important
    family: acceptance-scope-traceability
    title: |
      Done-when omits the approved mixed-version compatibility behavior
    detail: |
      workshop/issues/000540-couch-swallow-slots-osc-desktop-notifications-add-couch-notify-so-the-hub-tl-decides-what-reaches-the-operator.md:36 covers only notifications, while workshop/plans/000540-hub-notifications-plan.md:225 adds legacy messaging compatibility and recovery diagnostics. Add acceptance criteria for existing verbs against the old request contract, missing-registration and unsupported-notify guidance, and preservation of one-shot/no-retry behavior. ARCH-PURPOSE.
```

---

## Re-review — 2026-10-13T20:28:11-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 540 — couch: swallow slots' OSC desktop notifications; add couch --notify so the hub (TL) decides what reaches the operator |
| repo | pair |
| issue file | workshop/issues/000540-couch-swallow-slots-osc-desktop-notifications-add-couch-notify-so-the-hub-tl-decides-what-reaches-the-operator.md |
| boundary | whole-issue close |
| milestone | — |
| window | 407f578d0e1ee78b3b75aeacb26e1f726cec230b..0dcc138b6429a7ef620a27723e15e89420d94efe |
| command | sdlc close --issue 540 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T20:28:11-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the notification and compatibility contracts. Both prior findings are addressed, including independently verified red/green regression evidence for stalled writes. No new findings. The repository remains unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      presenter.go:318 preserves stall recovery while returning non-success to Notify. desktop_notify_test.go:148 covers both pre-existing and notification-induced stalls, recovery, and no replay. Routing Notify back through writeUnlessStalled in a temporary pinned-head copy made both regressions fail; restoring the fix made them pass.
  - id: BR-2
    disposition: addressed
    note: |
      Issue #540 Done-when lines 37–42 now require legacy actors/send/status compatibility, accurate restart guidance, single-exchange behavior, and truthful notification outcomes. These match the plan's mixed-version revision and messages_compat_test.go:34; the compatibility regression passed, and its frozen request shape matches the referenced pre-540 source.
```

1. **Strengths**
   - Acknowledged notification writes preserve errors without disrupting best-effort child effects (`cmd/internal/terminal/presenter.go:305–328`).
   - Suppression tests exercise every split position, all three protocols, duplicate delivery, counter lifecycle, and preservation of other terminal effects (`cmd/internal/couchtty/desktop_notify_test.go`).
   - Admission uses server-owned hub identity and rechecks caller authority before submission; real-socket tests cover lost acknowledgments without retries (`cmd/internal/couchcmd/notify_service_test.go`).
   - README, atlas, help, embedded skill, and artifact inventory cover the new surface.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused encoder, terminal, protocol, Couch CLI/console, and wrapper tests passed.
   - Initial integration compilation lacked generated runtime assets. Generating the normal bundle in a temporary pinned-head copy resolved this.
   - BR-1 mutation testing produced the expected failures in both stall cases.
   - Full-suite and opt-in native/desktop smoke results were not independently rerun. The temporary copy was removed.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Existing parser, sanitizer, authority checks, writer, and trace reused.
   - **ARCH-PURE — pass:** Pure validation/encoding remain directly testable; inventory classifications match the code.
   - **ARCH-PURPOSE — pass:** All child notification origins are suppressed; deliberate hub output remains separate.
   - **ARCH-MOCK — pass:** Stateful host/broker fixtures share production seams; native conformance fixtures cover the external path.
   - **ARCH-CONSTRAINTS — pass:** Payload bounds, existing deadlines/backpressure, and fixed-size counters constrain work.
   - **ARCH-SECURE — pass:** Closed request validation, sanitization, and live caller checks protect ingress.
   - **ARCH-ORDER — pass:** Existing presenter serialization owns writes; deterministic tests cover replacement, cancellation, stalls, and lost replies.
   - **ARCH-FUNERAL — pass:** Counters expire with panes; diagnostics use existing bounded retention; no new durable queue or receipt family.

7. **Plan revision recommendations:** None; existing revisions reconcile both prior findings.
