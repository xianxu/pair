# Boundary Review — pair#416 (whole-issue close)

| field | value |
|-------|-------|
| issue | 416 — Scrolled-back pane swallows keystrokes: zellij 0.45 scroll_mode_sync enters an unbound Scroll mode |
| repo | pair |
| issue file | workshop/issues/000416-scrolled-back-pane-swallows-keystrokes-zellij-0-45-scroll-mode-sync-enters-an-unbound-scroll-mode.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d5e6cf3c616da6dfe5256d36a10e7c95d020c57..1a63116cc981f1c387248aad979c9c2c12c08531 |
| command | sdlc close --issue 416 |
| reviewer | claude |
| timestamp | 2026-10-08T18:03:53-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This change is small and it closes the bug. `zellij/config.kdl` now sets `scroll_mode_sync false` and explains why in a comment. A new test, `TestConfigDisablesScrollModeSync`, checks that the setting is present in both the source config and the bundle mirror, and it ignores commented-out lines. The duplicated parsing from the frame-style test was moved into one shared helper. The bundle mirror doesn't appear in the diff because `.gitignore:126` ignores it and it is regenerated. The regenerated copy does contain the new line (line 44), and both config tests pass. `zellij/config.kdl` is the only zellij config file in the repo; the other `.kdl` files are layouts and probes. Nothing blocks shipping. The operator smoke test in the "Done when" list still has to be done by hand, and that happens outside this review.

1. **Strengths**
   - `configSettings` (`cmd/internal/runtimebundle/paneframestyle_test.go:58`) pulls the parse that used to be inline into one helper. Both tests now use the same parser and the same rule that comments don't count (ARCH-DRY: pass).
   - The pinned value can't be satisfied by a comment. The comment block right above the setting names the key itself, so this matters, and the test handles it correctly.
   - The comment at `zellij/config.kdl:38-43` gives the cause and how the fix works (Write sends ClearScroll first), which the Spec's source-level diagnosis supports.
   - The fix is just a config setting with no runtime code. 0.44 compatibility follows the precedent already recorded for `pane_frame_style` at config.kdl:60-61.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The three `## Plan` checkboxes are still unticked, even though the first two are delivered. Tick them at close.
   - Nothing automated checks that a key actually snaps the pane back. The test only checks the config line, so the live zellij behavior relies on the operator smoke test. That's acceptable for a config-only fix.

5. **Test coverage**
   - The new test fails if the setting is removed from either path, so it catches a regression. It runs without IO beyond reading a fixture file (ARCH-PURE: pass).
   - The "Done when" clause about ignoring commented mentions is met by the shared helper, which the earlier test already relied on.

6. **Architecture**
   - **ARCH-DRY:** pass, because the shared helper replaces the copy.
   - **ARCH-PURE:** pass; there is no logic tied to IO.
   - **ARCH-PURPOSE:** pass. The one config file and its mirror are both covered, and no keybind workaround was deferred.
   - If pair ever adds Scroll-mode bindings, this setting needs another look.

7. **Plan revisions:** none.

```findings
findings:
  - id: new
    severity: Minor
    family: plan-checkbox-state-tracks-delivery
    title: |
      Plan checkboxes for the delivered config line and regression test remain unticked
    detail: |
      The first two Plan items are in the 7d5e6cf3..1a63116c range but are still [ ]. Tick them at close. The operator smoke test is still pending.
```
