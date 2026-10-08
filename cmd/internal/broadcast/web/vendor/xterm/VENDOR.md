# Vendored @xterm/xterm

- Package: `@xterm/xterm` 6.0.0 (MIT, see `LICENSE`)
- Tarball: https://registry.npmjs.org/@xterm/xterm/-/xterm-6.0.0.tgz
- Integrity (npm `dist.integrity`, verified on download 2026-10-07):
  `sha512-TQwDdQGtwwDt+2cgKDLn0IRaSxYu1tSUjgKarSDkUM0ZNiSRXFpjxEsvc/Zgc5kq5omJ+V0a8/kIM2WD3sMOYg==`
- Files copied unmodified from the tarball:
  - `package/lib/xterm.js` → `xterm.js`
  - `package/css/xterm.css` → `xterm.css`
  - `package/LICENSE` → `LICENSE`
- SHA-256 of the copies:

```
14903579ff54664cd72f8e8699e6961a6272c21863ec1c3b118cdc8af5d4a972  xterm.js
854a7c0fb70e8b1a083c16797ab827299fb18744f5ad34f227b48337e33293c6  xterm.css
```

Same version as `@xterm/headless` in `tests/terminal-oracle`, so the oracle
that checks `terminal.Render` output checks the parser viewers run. Served
from the binary by `cmd/internal/broadcast` (#395); never loaded from a CDN.

Notes: 6.0.0 (2025-12-22) implements synchronized output (DECSET 2026), which
every broadcast frame is bracketed in, so viewers paint only whole frames.
5.5.0 ignored it, and a large write could paint half-way (#395 M5 smoke).

To update: `npm pack @xterm/xterm@<v>` in a scratch directory, compare its
sha512 with `npm view @xterm/xterm@<v> dist.integrity`, extract into an empty
directory, copy the three files, and update this file and the oracle's pin
together.

## @xterm/addon-unicode11 0.9.0

- License: MIT (`LICENSE-addon-unicode11`).
- Tarball: https://registry.npmjs.org/@xterm/addon-unicode11/-/addon-unicode11-0.9.0.tgz
- Integrity (verified on download 2026-10-08):
  `sha512-FxDnYcyuXhNl+XSqGZL/t0U9eiNb/q3EWT5rYkQT/zuig8Gz/VagnQANKHdDWFM2lTMk9ly0EFQxxxtZUoRetw==`
- `package/lib/addon-unicode11.js` → `addon-unicode11.js`, unmodified:

```
72353b5178e1a7382716df1cfedf8ab070eea655d38995bb9f4f284fe56e2f2b  addon-unicode11.js
```

Why: xterm.js defaults to Unicode 6 widths, which count most emoji (`👆`, `✅`,
`🤖`, …) as one column while Couch counts two, so a viewer misplaced the rest
of a row after one (#412). Unicode 11 widths agreed with Couch on 61 of 65
sample glyphs; the misses are emoji with a variation selector (`❤️`, `⚠️`:
Couch 2, viewer 1) and joined or skin-tone sequences (`👨‍💻`, `👍🏽`: Couch 2,
viewer 4). `@xterm/addon-unicode-graphemes` 0.4.0 handled those but counted
plain emoji as one column, so it was worse overall. The oracle pins the same
add-on.
