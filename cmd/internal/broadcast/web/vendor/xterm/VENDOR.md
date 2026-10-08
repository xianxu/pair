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
