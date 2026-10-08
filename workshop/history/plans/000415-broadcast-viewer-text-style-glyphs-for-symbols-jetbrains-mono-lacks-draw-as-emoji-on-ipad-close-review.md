# Boundary Review — pair#415 (whole-issue close)

| field | value |
|-------|-------|
| issue | 415 — Broadcast viewer: text-style glyphs for symbols JetBrains Mono lacks (⏸ ⏺ draw as emoji on iPad) |
| repo | pair |
| issue file | workshop/issues/000415-broadcast-viewer-text-style-glyphs-for-symbols-jetbrains-mono-lacks-draw-as-emoji-on-ipad.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6f9a549ef3ec73ec4408e4cdba232265c559e8a3..6706f8e3f757f489429156b7c5a46448323e033a |
| command | sdlc close --issue 415 |
| reviewer | claude |
| timestamp | 2026-10-08T13:59:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The change does what the issue set out to do. A derived list (`symbols.txt`), a reproducible subset font, and a `unicode-range` all agree, and the tests enforce that agreement. The font is served only under the token, and every glyph is exactly one JetBrains Mono cell wide. I checked these myself at HEAD 6706f8e3: `go test ./cmd/internal/broadcast` passes, and the vendored WOFF's SHA-256 (`2afe0898…5a4a`) matches the value in `VENDOR.md`. The preload fix for xterm.js's per-character width cache is correct. The atlas and the Revisions entry both record it. Nothing blocks the close. The only gap is that two prose comments still say the font is fetched lazily, which stopped being true with the preload revision, and they should be swept.

**1. Strengths**
- `symbols_test.go:49-74`: one test compares the font's cmap, `symbols.txt` and the CSS `unicode-range` exactly. The Log records that removing a code point from the list makes it fail. Those three copies can't drift apart unnoticed.
- `symbols_test.go:91-107`: the width test checks two things for each symbol. Couch's `GraphemeWidth` must be 1, and the font advance must equal JetBrains Mono's cell. That is the invariant the iPad bug depends on.
- `symbols.py:35-38`: the script refuses any input font whose SHA-256 differs from the pinned one, and the build is reproducible. A vendored artifact with a pinned source, licence and checksum meets the Done-when.
- `theme_test.go:83-94`: tests cover the new route without the token and the unlisted vendor files (`symbols.py`, `symbols.txt`, the licence), which keeps the attack surface small.
- `TestViewerPreloadsSymbolFont` checks that `SYMBOL_SAMPLE` is in the list. A sample outside the list would silently load nothing.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- `viewer.css:30-32` ("fetched only when one is on screen") and `VENDOR.md` ("so the browser fetches it only when one of those symbols is on screen") contradict the preload in `viewer.js:189-190` and the corrected atlas text. Of the three places that make this claim, only the atlas was corrected.
- `parseWOFF` does no bounds checks. That is acceptable because it is test-only and reads a checksummed vendored file. If someone reuses it outside tests it needs bounds checks.
- `symbols.py scan` reads each recording into memory in full (the recordings total 1.7 GB). This is development-only, so it is acceptable, but streaming would be friendlier.
- U+FE0E was skipped with the reason given in the Log. The Spec marked it optional, so this is acceptable.

**5. Test coverage notes**
- Covered: the route and token, the cmap/list/CSS agreement, the width invariant, and the preload wiring.
- The rendering itself (xterm.js measuring and Safari's emoji fallback) can only be checked by eye. The operator's second iPad smoke covered it and passed. That is the right oracle here.

**6. Architecture (ARCH-\*)**
- **ARCH-DRY: pass.** The list has one source; the CSS range is generated from it and checked by the test.
- **ARCH-PURE: pass.** The change is static assets plus a small piece of glue in `loadFont`.
- **ARCH-PURPOSE: pass on the code; flagged on the docs.** All consumers (font, CSS, tests) derive from `symbols.txt`. The preload correction was swept into the atlas but not into the CSS comment or `VENDOR.md` (Minor above).
- **ARCH-MOCK: N/A.** The runtime adds no external dependency, and fontTools is a development-only build tool.
- **ARCH-CONSTRAINTS: pass.** It adds a 4 KB fetch at page start, and the existing 3 s timeout race bounds it.
- **ARCH-SECURE: pass.** The font is same-origin and served only under the token. Unlisted files return 404, which is tested. The input font is pinned by SHA-256.
- **ARCH-ORDER: pass.** The only ordering is font load before `term.open`. A single awaited `Promise.all` makes it explicit, and the fallback path is unchanged.
- **ARCH-FUNERAL: pass.** Nothing durable is created at runtime. The font is a vendored file embedded in the binary.

**7. Plan revision recommendations:** none. The two Revisions entries (WOFF 1.0 instead of WOFF2, and the preload) already match the code.

```findings
findings:
  - id: new
    severity: Minor
    family: superseded-claim-not-swept
    title: |
      viewer.css comment and VENDOR.md still say the symbol font is fetched only when a symbol is on screen
    detail: |
      The preload revision (viewer.js loadFont, SYMBOL_SAMPLE) always fetches the face at page start. The atlas was corrected, but viewer.css (the @font-face comment) and VENDOR.md ("Why (#415)" paragraph) still describe lazy fetching. Reword both to "preloaded; used only for its unicode-range".
```
