# Vendored fonts

## JetBrains Mono 2.304

- License: SIL Open Font License 1.1 (`OFL.txt`), which allows bundling and
  redistribution with software.
- Source: https://github.com/JetBrains/JetBrainsMono/releases/download/v2.304/JetBrainsMono-2.304.zip
  - SHA-256 of the zip: `6f6376c6ed2960ea8a963cd7387ec9d76e3f629125bc33d1fdcd7eb7012f7bbf`.
    GitHub publishes no digest for it; its size (5622857 bytes) matched the
    release API's asset size on download, 2026-10-07.
- Files copied unmodified from `fonts/webfonts/` and the zip root:

```
c503cc5ec5f8b2c7666b7ecda1adf44bd45f2e6579b2eba0fc292150416588a2  JetBrainsMono-Bold.woff2
3a013466c0eee979fb9d42c2d7a8887cd3645dc8b897cfc5b71781cf982efc5a  JetBrainsMono-BoldItalic.woff2
cb6a1b246318ed3885d7dffa14a2609297fe80e9b8e500bea33b52fa312a36a4  JetBrainsMono-Italic.woff2
a9cb1cd82332b23a47e3a1239d25d13c86d16c4220695e34b243effa999f45f2  JetBrainsMono-Regular.woff2
```

Why this font: Ghostty draws JetBrains Mono, its built-in default, whenever the
configured `font-family` isn't installed. The operator's config names
DejaVuSansM Nerd Font Mono, which isn't installed, so JetBrains Mono is what
the operator's terminal actually shows. The broadcast viewer
(`cmd/internal/broadcast`, #395) serves these files same-origin, under the
link's token, and uses them with `@font-face`; viewers install nothing.
Nerd Font icons, which Ghostty draws from its built-in symbol font, fall back to
whatever the viewer's browser has.

Only fonts whose licence allows redistribution may be added here. To update:
download the release zip, compare its size with the release API, extract into
an empty directory, copy the four webfont files and the licence, and update
this file.

## Noto Sans Symbols 2 2.008, subset (`NotoSansSymbols2-Couch.woff`)

- License: SIL Open Font License 1.1 (`OFL-NotoSansSymbols2.txt`, copied from
  the release zip); no Reserved Font Name, so a modified subset may keep its
  name.
- Source: https://github.com/notofonts/symbols/releases/download/NotoSansSymbols2-v2.008/NotoSansSymbols2-v2.008.zip
  - SHA-256 of the zip: `346c930bbe8eb946701a05c54e9c11a2094dee1d93c387bf1771c0a3e335688f`
    (2331441 bytes, matching the release API's asset size on download,
    2026-10-08; GitHub publishes no digest).
  - Input: `NotoSansSymbols2/unhinted/ttf/NotoSansSymbols2-Regular.ttf`,
    SHA-256 `c4a0a80f0041ce4be81e2478faad22776d23edb98ae3f0d19bd37044820ecf9d`
    (`symbols.py` refuses any other file).
- Output, built by `symbols.py build` (reproducible):

```
2afe0898aa8b2fbfe6c10cac6573515191bb53474cafe6a0ca7cffea1edd5a4a  NotoSansSymbols2-Couch.woff
```

Why (#415): JetBrains Mono lacks some symbols agents draw, among them `⏺`
`⏸` `✳` `✔` and the braille spinner frames. iPad Safari falls back per
character and draws several of them as Apple Color Emoji. The viewer lists
this font as "Couch Symbols" after JetBrains Mono, with a `unicode-range` of
exactly `symbols.txt`, so the browser fetches it only when one of those
symbols is on screen.

The subset is **modified**: each glyph is moved into JetBrains Mono's 600-unit
cell, centred by its own advance box, and scaled down only if it would
overflow, so every symbol keeps the one column Couch counts. It is WOFF 1.0,
not WOFF2, so the Go tests can read its cmap and advances with the standard
library (`symbols_test.go`); at 55 glyphs it is under 4 KB.

`symbols.txt` is derived, not hand-picked. `symbols.py scan` reads recorded
agent output and Couch's own sources, keeps symbols and punctuation that
JetBrains Mono lacks and that are not wide (wide ones are emoji by default, as
in the operator's terminal), and adds those Noto Sans Symbols 2 covers to the
list. Scans only add; remove a code point by hand. It reports the symbols the
font lacks (2026-10-08: `※ ⅓ ⅔ ↳ ↵ ⇄ ⇣ ⎿ ⚙ ⟹ ⧈ ⧉`), which fall back to the
viewer's system fonts. To regenerate, with `pip install fonttools brotli`:

```
symbols.py scan  NOTO_TTF ~/.local/share/pair/**/scrollback-*.raw $(git ls-files 'cmd/*.go')
symbols.py build NOTO_TTF   # then paste the printed unicode-range into viewer.css
```

`go test ./cmd/internal/broadcast` fails until the font, the list and the
stylesheet's `unicode-range` agree.
