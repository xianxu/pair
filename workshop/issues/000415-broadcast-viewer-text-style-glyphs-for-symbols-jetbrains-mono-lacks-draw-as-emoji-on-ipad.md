---
id: 000415
status: open
deps: [pair#412]
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: '5bad09ec0ab0437f2fa8a7d703630c8940f97031' # card fields mirrored from issue-cards; edit via sdlc
---

# Broadcast viewer: text-style glyphs for symbols JetBrains Mono lacks (⏸ ⏺ draw as emoji on iPad)

## Problem

On an iPad (Safari), the broadcast viewer draws `⏸` (U+23F8) and `⏺` (U+23FA)
as Apple emoji, a white glyph on a grey tile, where the operator's terminal
draws plain text symbols. The packed JetBrains Mono 2.304 doesn't contain these
code points (checked against its `cmap`: 1,363 code points, no U+23F8 or
U+23FA), so Safari falls back per character, and for symbols that also exist as
emoji it picks Apple Color Emoji. Found in the #412 smoke on 2026-10-08.

## Spec

- Pack a small, redistributable symbol font (for example Noto Sans Symbols 2,
  OFL) that has text-style glyphs for these symbols. Check its `cmap` first.
  Serve it same-origin like JetBrains Mono, with an `@font-face`
  `unicode-range` limited to the symbols it supplies, so the browser fetches it
  only when such a symbol is on screen and all other text stays JetBrains Mono.
- List it in the viewer's font stack after JetBrains Mono and before any emoji
  fallback.
- Cover at least the symbols Couch and common agent output use: `⏸ ⏺ ⏵ ⏹ ⏳
  ⏱ ↩ ✔ ✓ ✗ ⚠` (exact set decided from the font's coverage and a scan of
  typical Claude Code output). Keep each glyph's width as Couch counts it.
- Optionally, also send the text-presentation selector (U+FE0E) for these
  symbols in the viewer. Measure whether it changes anything on iOS before
  keeping it.

## Done when

- On an iPad (manual smoke), `LIVE ⏸` and Claude Code's `⏺` bullets draw as
  plain text symbols in the viewer, matching the operator's terminal.
- The font file is vendored with source, licence and checksum (`VENDOR.md`),
  served only under the token, and loaded only for its `unicode-range`; a
  server test covers the route, and the page lint still allows no other
  origin.
- A width test shows these symbols keep the column width Couch counts.

## Plan

- [ ] Pick the font and check its cmap; vendor a subset or the whole file
- [ ] `@font-face` with `unicode-range`, route, tests; iPad smoke

## Log

### 2026-10-08

- Filed from #412's smoke, kept out of #412 to avoid scope creep.
