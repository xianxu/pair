# Boundary Review — pair#842 (whole-issue close)

| field | value |
|-------|-------|
| issue | 842 — Cast broadcast bar: a ⏎ send button when sketches or annotations are pending |
| repo | pair |
| issue file | workshop/issues/000842-cast-broadcast-bar-a-send-button-when-sketches-or-annotations-are-pending.md |
| boundary | whole-issue close |
| milestone | — |
| window | aafc7c8c08f9d49b950b38265beecc9ed569b1b2..f1b91fa4cecfab3fac3731105d05fb93331e7faf |
| command | sdlc close --issue 842 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T00:25:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation follows the revised scope: Chrome feedback drives visibility independently of preview completion, and clicks reuse the existing submission action. Focused tests, the Neovim transaction suite, and inventory checks passed. Two Important findings remain: missing README documentation and an incomplete successor-endpoint test oracle. The required operator live-cast acceptance remains pending in the tracker.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-surface-documentation
    title: |
      README update appears missing for the Send button and wider shortcut availability
    detail: |
      README.md:1819 still describes Shift+Ctrl+Alt+Return as a draft action, while cmd/internal/workbenchshortcut/shortcut.go:183 makes it available across panes, including fullscreen right-terminal applications. README is unchanged in this range and does not describe the new Chrome status ⏎ control. Update the submission instructions with button visibility, shortcut availability, and the terminal-feedback limitation.
  - id: new
    severity: Important
    kind: defect
    family: endpoint-isolation-test-oracle
    title: |
      Pane-replacement test does not observe the successor endpoint
    detail: |
      cmd/internal/couchtty/console_send_test.go:74 creates a successor child, but lines 99–103 flush and count writes only on r.child, the original child. Accidental routing to the successor would therefore satisfy the zero-write assertion. Retain and clean up the successor, flush both endpoints, and assert that neither receives the chord after replacement (ARCH-ORDER).
```

1. **Strengths**

   - `webcast/board/submission.go:48` reuses selection eligibility and exact delivered-version masking rather than introducing another feedback authority.
   - `couchtty/console_send.go:30` observes the Board outside `Console.mu`, then revalidates cast identity before publishing the cached result.
   - `couchtty/console_send.go:61` validates the accepted pane, thread, endpoint, and cast before bounded FIFO admission.
   - Rendering tests sweep widths 0–100 and verify complete glyphs, exact spans, and non-overlap.

2. **Critical findings**

   None found.

3. **Important findings**

   The two findings above require a README correction and a stronger endpoint-isolation assertion. No production misrouting was observed.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Focused Go tests covering feedback projection, Send rendering/clicks, shortcuts, and wrapper/terminal routing.
   - `make test-submission-transaction`, including DOM submission tests.
   - `make test-affected-inventory`.
   - `git diff --check`.

   The full repository suite and operator live-cast acceptance were not independently performed during this review. No prior findings required disposition.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared binding generates the Lua mapping; payload construction remains in the existing transaction.
   - **ARCH-PURE — pass:** rendering is directly tested; Board observation and endpoint admission remain in integration code.
   - **ARCH-PURPOSE — pass:** delivers the explicitly revised Chrome eligibility scope; terminal limitations are documented in atlas.
   - **ARCH-MOCK — pass:** reuses stateful browser, recording, and child-endpoint seams.
   - **ARCH-CONSTRAINTS — pass:** painting reads a cached scalar; candidate observations are coalesced; input admission uses the bounded FIFO.
   - **ARCH-SECURE — pass:** no new credential or persisted-input boundary; stale identities are checked before admission.
   - **ARCH-ORDER — flag:** production identity checks are sound on inspection, but the successor-endpoint test oracle is incomplete.
   - **ARCH-FUNERAL — pass:** watcher subscriptions have deferred cleanup; no new durable artifact family.
   - **ARCH-EVOLVE — pass:** no persisted application format changes.

7. **Plan revision recommendations**

   No behavioral revision is needed. Refresh the superseded Core concepts tables into a current greppable table with explicit PURE/INTEGRATION kinds and the final Board/Cast/watcher paths. Keep live acceptance explicitly pending until observed.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8c21a1b4-831f-412c-82e0-35a6f73b024d","repository":"github.com/xianxu/pair","issue_id":"000842","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0543ab481621ccf4","reviewed_head":"f1b91fa4cecfab3fac3731105d05fb93331e7faf","slot":"pair:7","requested_at":"2026-10-24T07:22:03.004821Z"},"phase":"finished","pid":76236,"started_at":"2026-10-24T07:22:03.30821Z","finished_at":"2026-10-24T07:25:21.029921Z","session_id":"01a16cc9-c0f8-7e91-b7f8-567c085db44d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1m52.283111666s
head: "f1b91fa4cecfab3fac3731105d05fb93331e7faf"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000415

---

## Re-review — 2026-10-24T00:36:02-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 842 — Cast broadcast bar: a ⏎ send button when sketches or annotations are pending |
| repo | pair |
| issue file | workshop/issues/000842-cast-broadcast-bar-a-send-button-when-sketches-or-annotations-are-pending.md |
| boundary | whole-issue close |
| milestone | — |
| window | aafc7c8c08f9d49b950b38265beecc9ed569b1b2..27f192b8930fe672722112a425c97ca21a5d774e |
| command | sdlc close --issue 842 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T00:36:02-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are addressed, and no new defects were found. The implementation matches the revised Chrome-only eligibility scope and reuses the existing submission transaction. Focused tests and the Neovim transaction suite passed. The tracker explicitly leaves operator live-cast acceptance pending; this review does not establish that acceptance.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:1819–1830 now documents availability from any Pair pane, fullscreen right-terminal reservation, Chrome Send visibility, and the terminal-feedback limitation. These match workbenchshortcut/shortcut.go:183 and couchtty/console_presentation.go:17.
  - id: BR-2
    disposition: addressed
    note: |
      console_send_test.go:61–118 retains and cleans up the successor, flushes both endpoints, and checks both sinks. In a disposable copy, deliberately routing to the current successor made the replacement test fail at line 114: “successor received 1 send chord(s).”
```

1. **Strengths**

   - `console_send.go:59–75` validates the accepted frame’s pane, thread, endpoint, and cast before admitting input, and reports refusal.
   - `webcast/board/submission.go:48` reuses existing selection logic and exact delivered-version masking.
   - `reserve_send_test.go` checks independent spans, Unicode widths, clipping, and absence of invisible hit targets.
   - README and atlas now explain the implemented scope consistently.

2. **Critical findings**

   None.

3. **Important findings**

   None remaining.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused tests passed across couchtty, webcast/board, webcast, workbenchshortcut, keyhelp, wrapcmd, and termcmd.
   - `make test-submission-transaction` passed, including the real Neovim transaction failure matrix and DOM submission tests.
   - The deliberate wrong-destination mutation independently qualified BR-2’s corrected oracle.
   - `git diff --check` passed; the reviewed checkout remains unchanged.
   - The full repository suite and operator live-cast acceptance were not independently performed.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** one declared chord and existing PairSendDOM transaction own submission.
   - **ARCH-PURE — pass:** rendering and shortcut decisions remain pure; Board observation and endpoint admission stay at integration boundaries.
   - **ARCH-PURPOSE — pass:** delivers the explicitly revised eligibility scope without introducing unsupported terminal submission.
   - **ARCH-MOCK — pass:** tests use existing stateful child/endpoint and transaction seams; no new external dependency.
   - **ARCH-CONSTRAINTS — pass:** paint reads cached availability; candidate observation occurs outside the Console lock; input admission uses the bounded FIFO.
   - **ARCH-SECURE — pass:** the button emits a fixed internal chord and adds no new untrusted-data parser or credential handling.
   - **ARCH-ORDER — pass:** accepted-frame identities guard admission; successor isolation now has a qualified regression oracle.
   - **ARCH-FUNERAL — pass:** candidate subscriptions are released with the existing tracked watcher; no new durable artifacts.
   - **ARCH-EVOLVE — pass:** no persisted format changes.

7. **Plan revision recommendations**

   None required. The final entity table matches the implementation, and existing revisions explicitly supersede the preview-based design. Keep live acceptance pending until observed.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a6349f19-b63e-4fc1-a6e1-83da4da91abe","repository":"github.com/xianxu/pair","issue_id":"000842","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4a19e5a9d227be68","reviewed_head":"27f192b8930fe672722112a425c97ca21a5d774e","slot":"pair:7","requested_at":"2026-10-24T07:33:26.244811Z"},"phase":"finished","pid":26864,"started_at":"2026-10-24T07:33:26.381459Z","finished_at":"2026-10-24T07:35:51.171603Z","session_id":"01a16cd4-2c6f-71e2-a0b3-b2c59bc7c9df","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.6223995s
head: "27f192b8930fe672722112a425c97ca21a5d774e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
