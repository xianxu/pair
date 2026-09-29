# Boundary Review — pair#247 (whole-issue close)

| field | value |
|-------|-------|
| issue | 247 — Shade live Couch threads by idle time |
| repo | pair |
| issue file | workshop/issues/000247-couch-live-idle-shading.md |
| boundary | whole-issue close |
| milestone | — |
| window | 25233cdf7d09f502aa54a99cfcb446986e574dbf..9011130807b360cf82fa2bc9408f3f78b426d761 |
| command | sdlc close --issue 247 |
| reviewer | claude |
| timestamp | 2026-09-28T23:59:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

I reviewed round 5 of pair#247 (`25233cdf..90111308`). All three open Minor findings are fixed, and I found nothing new. The fixes are:
- **BR-2 (glyph colour):** glyph colour now comes from a typed base colour that feeds `FadeStyle`.
- **BR-3 (atlas):** the atlas now matches the code that is actually wired.
- **BR-4 (precedence comments):** both precedence sites now point at each other.

The whole feature is delivered as the Spec describes:
- a three-level idle classifier;
- fading that blends toward the terminal's reported background;
- one shared activity definition used by couch and the title poller;
- a background activity pass that only allows one run at a time;
- palette capture through the input replies.

Tests: `threadactivity`, `titlepoller` and `couchtty` pass at HEAD 90111308, except two couchtty tests I had to skip. `TestNotificationPTYConformance` and `TestCouchProductionSoak` fail inside the sandbox with "operation not permitted" when starting a pty child and creating a temp directory. That is the sandbox blocking them, not a problem in this diff. I did not run them outside the sandbox.

1. **Strengths**
   - `slotGlyphBase` (`reserve.go`) returns a `styleBase`, not an escape sequence, so glyph colour and fading come from one place. `FadeStyle(p, IdleFresh, baseAmber)` is exactly `attentionSGR`, which keeps a fresh row byte-for-byte unchanged. `TestRenderStatusRowFreshChipIsUnchanged` pins this across three palettes.
   - The slot-git merge became a shared generic `mergeObservations` instead of a copy, and `repaintVisible` gathers every background repaint into one helper (ARCH-DRY).
   - `IdleLevelFor`, `FadeStyle`, `blend` and `quantize256` are pure and tested without IO. `threadactivity.Latest` gets its IO through an injected `Runtime`, and its tests use a fake runtime (ARCH-PURE).
   - `ensureMenuLocked` keeps a palette reply that arrives before the menu is built. `capturePalette` ignores a malformed reply (nil colour), which leaves the palette unknown and the fade on its ANSI 90 fallback instead of an invented colour (ARCH-SECURE).
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
5. **Test coverage**
   - The tab bar's fading and its precedence rules are tested through `RenderStatusRow`.
   - The switcher is tested through the rendered frame, which is the BR-6 fix.
   - `Latest` has cases for each signal being the newest, and for a launch with no other signal.
6. **Architecture, marker by marker**
   - ARCH-DRY: pass. ARCH-PURE: pass. ARCH-PURPOSE: pass; both views read the same `MenuState.Activity` and `Palette`.
   - ARCH-MOCK: pass. The file-system and session IO is behind `threadactivity.Runtime`. The only real terminal interaction, the OSC 10/11 query, is exercised through input replies.
   - ARCH-CONSTRAINTS: pass. Each probe is bounded to 2 s and runs one at a time off the render path, and the measured 114 ms per thread is recorded.
   - ARCH-SECURE: pass.
   - ARCH-ORDER: pass. `AdvanceRefreshSchedule` uses generation checks to drop stale results, and a failed probe keeps the last value.
   - ARCH-FUNERAL: pass. Activity is in-memory only, and addresses a pass no longer probes are dropped.
7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      slotGlyphBase now returns a styleBase consumed by FadeStyle in both colorMenuGlyph and RenderStatusRow; a new glyph colour must add a base, it cannot silently become amber.
  - id: BR-3
    disposition: addressed
    note: |
      atlas/couch.md now says the tab bar reads Activity/Palette via statusModelLocked into StatusActor.Idle/StatusModel.Palette, matching console_presentation.go, and M2 wiring is live.
  - id: BR-4
    disposition: addressed
    note: |
      Cross-reference comments now sit in RenderStatusRow (reserve.go) and renderRootMenuFrame's live-row branch (menu_render.go), each naming the other.
```
