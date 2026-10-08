# Boundary Review — pair#395 (milestone M4)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 568d2bf3d48852836f199ecffb84cef65673ca9b..82d799fc630c3e75c194dee750da227cb4a959e3 |
| command | sdlc milestone-close --issue 395 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-10-07T21:34:18-07:00 |
| verdict | SHIP |

## Review

The review is done; writing up the verdict.

```verdict
verdict: SHIP
confidence: high
```

M4 does what its revised sketch says. Couch now asks the terminal for its 16 ANSI colours (OSC 4) as well as foreground and background (OSC 10/11). Replies are parsed strictly, kept under `c.mu`, and sent to each new viewer as `event: theme` before its first frame. The viewer accepts only `#rrggbb` values. JetBrains Mono 2.304 (OFL licence) is bundled, listed explicitly, and served only under the link's token. A plan Revision dated 2026-10-07 records that bundling the font replaced the `COUCH_BROADCAST_FONT_FILE` idea. All four font files match the SHA-256 values in `VENDOR.md`.

Test runs at the window head:
- The targeted tests pass: `broadcast` (including `TestViewerNode`), `couchtty -run 'Broadcast|OSC4|Palette'`, and `terminal -run TestInput`.
- The other `couchtty` and `terminal` failures are `operation not permitted` from the sandbox (pty and `/tmp`), not from this diff.
- The `artifactpath` failure is about `wrapcmd` and `nvim/review` files. `theme.go` is registered correctly, so it is not caused by this window.

Nothing blocks SHIP. The findings below are all Minor.

1. **What was done well**
   - The server checks the palette at both ends: `parseOSC4Reply` (`console_palette.go:103`) rejects bad digits, more than 4 hex digits per component, indices outside 0–15, and non-`rgb:` values. The viewer's `xtermTheme` (`viewer.js:35`) checks again. `theme.test.mjs` shows CSS/URL injection attempts end up as `{}`.
   - The font route allows only the listed files. `TestServerServesFontsUnderToken` returns 404 for a wrong token and for `OFL.txt`.
   - OSC 4 replies are classified as replies, so they never reach the child. `TestOSC4ReplyFeedsBroadcastThemeNotChild` covers this, plus a new row in `input_test.go` that also tests every split point.
   - The theme is read per viewer through a function guarded by the console mutex. A late joiner or a reconnecting viewer gets the theme without any new hub state.
   - The font vendoring is reproducible: source URL, zip hash, per-file hashes, the licence file, and a rule that only redistributable fonts may be added.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **Stale M4 line in the issue (plan-code-drift):** the issue's `## Plan` M4 row (issue file, line 179) still says `COUCH_BROADCAST_FONT_FILE` served via `@font-face`. That idea was replaced by the bundled font. This is the 6th finding in the `plan-code-drift` family. The rule that covers all of them: when a plan Revision replaces a mechanism, the issue's `## Plan` row that names it is rewritten in the same commit.
   - **Second colour parser (ARCH-DRY):** `parseOSC4Reply` writes its own `rgb:` parser. OSC 10/11 go through `ansi.XParseColor`. The strict parser is defensible (`XParseColor` turns bad hex into 0), but short components round differently. For example `rgb:f/8/0` gives `#ff8800` here, while `XParseColor` uses its own shift. Background and ANSI colours could differ by one step. Either note the difference or use one strict parser for all three. `broadcast.Hex` also repeats `toRGBA`'s `>>8`.
   - **Decoder seam not pinned (ARCH-MOCK):** the new OSC 4 row in `input_test.go:99` leaves `event` as nil. Only `Reply=true` is pinned, not that the decoder outputs `uv.UnknownOscEvent` with the full `ESC ] 4;…` prefix, which `parseOSC4Reply` needs. I checked ultraviolet's `decoder.go:859`, and the prefix is there today. If an ultraviolet upgrade changes it, themes quietly stop arriving. The fix is to set `event: uv.UnknownOscEvent(tc.raw)` on that row.
   - **8-bit terminators:** the 8-bit ST (`0x9c`) and the C1 OSC introducer (`0x9d`) aren't handled. That's rare, and the result is only a missing colour.
   - **Font wait before connecting:** `start()` waits up to 3 s for the font before it opens the event stream, so a slow font load delays the first frame. This is accepted and the atlas documents it.

5. **Test coverage**
   - Covered: the OSC 4 parse table, the query covering all 16 colours, the theme arriving before the first frame, no theme event when none is set, and the end-to-end path (reply event → console → session → viewer, in `TestBroadcastSendsOperatorTheme`).
   - Also covered: theme mapping and rejection in node, and the theme event reaching the renderer.
   - The new `TestBroadcastLateHandoverAfterAbandonIsNotAdopted` tests the ordering behind M3's BR-11 directly; it's a welcome deterministic check.
   - The visual check was done by the operator and is recorded in the Log.

6. **Architecture, per principle**
   - **ARCH-DRY:** flag (Minor, the second colour parser above).
   - **ARCH-PURE:** pass. `parseOSC4Reply`, `Hex` and `xtermTheme` are pure; the IO is a small `Theme` function.
   - **ARCH-PURPOSE:** pass. Colours and font both reach viewers, and the change of approach is recorded.
   - **ARCH-MOCK:** flag (Minor, the decoder seam above).
   - **ARCH-CONSTRAINTS:** pass. 16 extra query sequences once at startup, about 380 KB of embedded fonts chosen by the operator, and a bounded 3 s wait for the font.
   - **ARCH-SECURE:** pass. Input is checked at both ends, fonts are served from a fixed list behind the token, and the CSP already allows `font-src 'self'`.
   - **ARCH-ORDER:** pass. The palette is set once at startup and only read per viewer; a reply arriving after a viewer joins misses only that viewer's theme, and that's harmless.
   - **ARCH-FUNERAL:** pass. Nothing durable is created: the palette lives in memory and the fonts are inside the binary.

7. **Suggested plan Revision:** add one entry rewriting the issue's M4 `## Plan` row to "the operator's palette (OSC 10/11/4) as `event: theme`; JetBrains Mono 2.304 bundled under `/<token>/fonts/`", in place of `COUCH_BROADCAST_FONT_FILE`.

```findings
findings:
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Issue Plan M4 row still names COUCH_BROADCAST_FONT_FILE after the bundled-font revision
    detail: |
      This is the 6th finding in family plan-code-drift. Rule: a plan Revision that replaces a mechanism also rewrites the issue's Plan row naming it, in the same commit. The issue file at line 179 still promises the env-var font file.
  - id: new
    severity: Minor
    family: duplicate-color-parser
    title: |
      parseOSC4Reply writes its own rgb: parser while OSC 10/11 use ansi.XParseColor
    detail: |
      ARCH-DRY. The strict parser is defensible, but short components round differently from XParseColor's shift, so background and ANSI colours can disagree by one step. broadcast.Hex also repeats toRGBA. Use one strict parser for both, or say why there are two.
  - id: new
    severity: Minor
    family: plan-test-coverage-gap
    title: |
      The input_test OSC 4 row pins only Reply, not the UnknownOscEvent with the full prefix that capturePalette needs
    detail: |
      This is the 3rd finding in family plan-test-coverage-gap. Rule: when a feature consumes an event from an external decoder, at least one test feeds the real decoder's output to the consumer instead of a hand-built event. Fix: set event uv.UnknownOscEvent(tc.raw) on that row (input_test.go:99).
```
