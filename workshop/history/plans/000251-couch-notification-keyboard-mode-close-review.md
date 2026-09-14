# Boundary Review — pair#251 (whole-issue close)

| field | value |
|-------|-------|
| issue | 251 — Keep Ctrl+Return notification jumps working across Couch thread switches |
| repo | pair |
| issue file | workshop/issues/000251-couch-notification-keyboard-mode.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6a9ca2bffa05ae5de98e77d4adddaed15b9376dc..816ceddcd34cb9de02bb9a7727e61a76f909b9b8 |
| command | sdlc close --issue 251 |
| reviewer | codex |
| timestamp | 2026-09-14T09:46:45-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The keyboard repair matches the revised Spec/Plan, and the affected-package tests, focused race tests, and paint-gate check pass. I found no blocking runtime defect. One inexpensive documentation correction remains: README still describes the old keyboard ownership and accepted encodings.

1. **Strengths**
   - `keyboard.go:12` shares a small, pure suffix policy across live output and takeover.
   - `console_keyboard_test.go:44` exercises terminal-generated input through notification jumping and acknowledgement.
   - `keyboard_test.go:49` uses a blocked Host write to verify serialization; cleanup tests cover both buffers and reject later output.
   - The additive flag operation matches the [Kitty protocol](https://sw.kovidgoyal.net/kitty/keyboard-protocol/#progressive-enhancement).

2. **Critical findings**

   None.

3. **Important findings**

   **README update missing for changed keyboard behavior — [README.md:404](/Users/xianxu/workspace/pair/README.md:404).** README attributes protocol enablement to Zellij and says Ctrl+Return recognizes **only** `CSI 13;5u`. Production now maintains disambiguation itself and recognizes explicit press/repeat forms (`keys.go:181`). Update this paragraph to describe Couch ownership, supported event forms, and the unsupported-terminal fallback. Atlas already documents these correctly. **ARCH-PURPOSE**: operator documentation must accompany the delivered behavior.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:
   - `go test ./cmd/internal/couchtty ./cmd/internal/hostty ./cmd/internal/ptychild ./cmd/internal/artifactpath`
   - Focused keyboard/mouse tests under `-race -count=1`
   - `tests/paint-gate-consumers-test.sh`
   - Pinned-range `git diff --check`

   Coverage includes fragmentation, oversized strings, reset/pop, background isolation, flag preservation, explicit events, and teardown. I did not independently repeat the recorded live smoke or baseline red run.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared control constant and policy helper.
   - **ARCH-PURE — pass:** deterministic byte policy separated from Host IO.
   - **ARCH-PURPOSE — flag:** README remains inconsistent with delivered behavior.
   - **ARCH-MOCK — pass:** stateful terminal double consumes production output through the Host seam.
   - **ARCH-CONSTRAINTS — pass:** bounded per-chunk work; no added workers or retained production buffers.
   - **ARCH-SECURE — pass:** framing preserved; diagnostic fields bounded and quoted.
   - **ARCH-ORDER — pass:** scanner/write transactions and terminal release are serialized; deterministic barriers exercise ordering.
   - **ARCH-FUNERAL — pass:** terminal ownership ends explicitly; existing opt-in diagnostic capture retains its documented cleanup policy.

7. **Plan revision recommendations**

   Append a `## Revisions` entry adding README synchronization to the documentation deliverable. No implementation redesign is needed.

```findings
findings:
  - id: new
    severity: Important
    family: public-behavior-doc-parity
    title: |
      README still describes the previous Ctrl+Return protocol contract
    detail: |
      README.md:404-408 attributes keyboard enablement to Zellij and says only CSI 13;5u is recognized, while Console now maintains disambiguation and keys.go accepts explicit press/repeat forms. Update the operator documentation with Couch ownership, supported events, and the unsupported-terminal fallback; atlas alone does not satisfy the README gate (ARCH-PURPOSE).
```

---

## Re-review — 2026-09-14T09:49:45-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 251 — Keep Ctrl+Return notification jumps working across Couch thread switches |
| repo | pair |
| issue file | workshop/issues/000251-couch-notification-keyboard-mode.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6a9ca2bffa05ae5de98e77d4adddaed15b9376dc..3ecee04dca027c98aa40a03b0b62803ff4cc33e5 |
| command | sdlc close --issue 251 |
| reviewer | codex |
| timestamp | 2026-09-14T09:49:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range matches the revised Spec and Plan. BR-1 is addressed by the README correction, supported by the implementation and passing behavioral tests. No new blocking findings. The range also includes documented #207 diagnostic work; its recovery milestone remains separate.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:404-410 now documents Couch ownership, implicit/explicit press and repeat, release exclusion, and the unsupported-terminal fallback. This matches console.go:593,1152,1236 and keys.go:178-182. The correction is prose-only; existing physical-key and event-encoding tests pass.
```

1. **Strengths**

   - `keyboard.go:11` provides one pure suffix policy, preserving payload bytes and deferring insertion across incomplete framing.
   - `console.go:1030` serializes cleanup, resets both relevant buffers, and prevents subsequent terminal writes.
   - `console_keyboard_test.go:41` derives physical Ctrl+Return from emitted terminal state, then verifies notification selection, acknowledgement, and ordinary Return forwarding.
   - Mouse diagnostics distinguish scanner beliefs from accepted writes and test partial writes, errors, and attribution during blocked output.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Tests for `couchcmd`, `couchtty`, `hostty`, and `ptychild`.
   - Focused race tests covering keyboard, notification shortcuts, interception, and mouse tracing.
   - Paint-gate consumer checks and pinned-range `git diff --check`.

   Coverage includes fragmented and oversized controls, replay, background isolation, preserved flags, bounded stack growth, output ordering, and shutdown. Full-suite and physical-terminal acceptance are recorded in the tracker; I did not independently repeat those. The plan explicitly records that the expanded live protocol probe was not performed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared control constant, suffix helper, and existing framing authority.
   - **ARCH-PURE — pass:** Byte transformation and formatting remain directly testable without IO; Console owns integration.
   - **ARCH-PURPOSE — pass:** Startup, live output, takeovers, and cleanup implement the stated shortcut requirement.
   - **ARCH-MOCK — pass:** Stateful Host double consumes production writes. Additive flags and independent screen stacks agree with the [Kitty protocol](https://sw.kovidgoyal.net/kitty/keyboard-protocol/#progressive-enhancement).
   - **ARCH-CONSTRAINTS — pass:** Bounded suffix overhead; no new production queue, timer, or worker.
   - **ARCH-SECURE — pass:** Existing framing handles arbitrary PTY bytes; diagnostic fields are bounded and quoted.
   - **ARCH-ORDER — pass:** Scanner/write transactions share a mutex; barrier tests exercise contention and release suppression.
   - **ARCH-FUNERAL — pass:** Keyboard changes create no durable artifacts. Existing opt-in diagnostic captures have documented ownership, cost, closure, and operator removal.

   Keep #224’s typed output interface separate; the current mutex establishes the required ordering.

7. **Plan revision recommendations:** None. Existing revisions explain the serialization prerequisite, cleanup changes, and narrowed live acceptance scope.
