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
