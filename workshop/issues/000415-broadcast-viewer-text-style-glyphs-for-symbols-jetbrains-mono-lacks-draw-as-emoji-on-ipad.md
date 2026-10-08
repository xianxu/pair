---
id: 000415
status: working
deps: [pair#412]
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: 'f89573f4925b0cdc02001040dbf9bf57b04f4bfb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-08T12:59:16-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "5d3cdc5f", done: "752969f7"}
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
- **Coverage is a set, not two characters.** Agents hosted in Couch draw
  with many symbols for their ASCII-art UI (bullets, spinners, arrows,
  checkmarks, box and block elements). Expect fewer than 100 code points.
  Derive the set reproducibly: scan recorded agent output (Pair's raw
  scrollback recordings) for code points JetBrains Mono lacks, union them with
  the symbols Couch draws itself (`⏸ ⏺ ⏵ ⏹ ⏳ ⏱ ↩ ✔ ✓ ✗ ⚠` and so on), and
  keep those the chosen font covers. Check the result into the repository as a
  list, with the script that produced it, so it can be regenerated as agents
  change.
- Subset the font to that list (fewer than 100 glyphs, a few KB as WOFF2) and
  set `unicode-range` to exactly the list. Keep each glyph's width as Couch
  counts it.
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
- The code-point list and the script that derives it are checked in; the
  vendored subset covers exactly that list (a test compares the font's cmap
  with the list).

## Plan

- [x] Pick the font and check its cmap: Noto Sans Symbols 2 v2.008 (55 of 67
      wanted code points; Math, Symbols, DejaVu Sans Mono cover fewer)
- [x] `symbols.py scan|build`, `symbols.txt`, subset WOFF with cell advances,
      VENDOR.md
- [x] "Couch Symbols" `@font-face` + `unicode-range`, viewer font stack, route
- [x] Tests: route + unlisted-file 404s, cmap == list == CSS range, Couch
      width 1 and 600-unit advance per symbol
- [ ] iPad smoke (operator)

## Log

### 2026-10-08

- Filed from #412's smoke, kept out of #412 to avoid scope creep.
- Operator: agents in Couch will use more such characters for ASCII art;
  expect fewer than 100 code points. The spec now plans for a derived set
  (scan of recorded agent output, plus Couch's own symbols) and a subset
  font.
- Scan (66 recordings, 1.7 GB, plus non-test Go sources): 67 non-wide
  symbol/punctuation code points JetBrains Mono lacks; braille spinner frames
  dominate. Noto Sans Symbols 2 covers 55 → `symbols.txt`. Uncovered, left to
  system fallback: `※ ⅓ ⅔ ↳ ↵ ⇄ ⇣ ⎿ ⚙ ⟹ ⧈ ⧉`. Of those, only `⚙` has an emoji form.
- Wide (Emoji_Presentation) characters such as `⏳ ✅ ❌` are excluded on
  purpose: the operator's terminal draws them as emoji too.
- U+FE0E (optional in Spec) not done: once the font supplies the glyph, the
  selector has nothing left to fix, and measuring it on iOS needs the device.
- Verification: `go test ./cmd/internal/broadcast` green; mutation (drop
  U+2733 from symbols.txt) fails `TestSymbolFontCoversExactlyTheList`. The
  build is reproducible (two runs, same SHA-256). Broadcast's dependents
  (`cmd/couch`, `couchcmd`, `couchtty`) pass unsandboxed except
  `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees`, which fails
  on this agent session's inherited `PAIR_DATA_DIR` scope (unrelated).
  `make -k test`: `test-review` fails the same 7 checks on origin/main (pre-existing);
  `test-lua` and `test-changelog` pass with the env scrubbed;
  `test-pair-embedded-runtime` hits the same inherited-scope conflict.
- iPad smoke 1 (operator): no more emoji, but `⏺` swallowed the next column
  (`●The`) and `⏸` drew just outside LIVE's red background. Cause: xterm.js
  6's DOM renderer `WidthCache` measures a character on first draw and keeps
  `letter-spacing = cell − measured`. The unicode-range face was fetched by
  that same first draw, so the fallback was measured. Once the real one-cell
  glyph arrived, the stale negative spacing collapsed it to zero advance, and
  the next cell drew under it. Fix: `loadFont` also loads "Couch Symbols"
  (`SYMBOL_SAMPLE` ⏺) before the terminal opens. `TestViewerPreloadsSymbolFont`
  guards it. The atlas's "no preload needed" claim was wrong and is corrected.

## Revisions

- 2026-10-08: subset ships as **WOFF 1.0**, not WOFF2 (Spec said "a few KB as
  WOFF2"). Reason: the cmap/advance test reads the font with Go's stdlib
  (zlib); WOFF2 needs brotli, which isn't a dependency. Cost: ~3.9 KB instead
  of ~3 KB. Glyphs are **modified** (re-advanced to JetBrains Mono's 600-unit
  cell, scaled only on overflow) to honour "keep each glyph's width"; OFL
  allows it, Noto has no Reserved Font Name.
- 2026-10-08: the symbol font is **preloaded** at page start (4 KB) rather
  than fetched on first use. Done-when's "loaded only for its unicode-range"
  still holds in that only those code points draw from it, but the file is
  always fetched. Reason: xterm.js's per-character width measurement (see Log).
