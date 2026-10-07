# Vendored @xterm/xterm

- Package: `@xterm/xterm` 5.5.0 (MIT, see `LICENSE`)
- Tarball: https://registry.npmjs.org/@xterm/xterm/-/xterm-5.5.0.tgz
- Integrity (npm `dist.integrity`, verified on download 2026-10-07):
  `sha512-hqJHYaQb5OptNunnyAnkHyM8aCjZ1MEIDTQu1iIbbTD/xops91NB5yq1ZK/dC2JDbVWtF23zUtl9JE2NqwT87A==`
- Files copied unmodified from the tarball:
  - `package/lib/xterm.js` → `xterm.js`
  - `package/css/xterm.css` → `xterm.css`
  - `package/LICENSE` → `LICENSE`
- SHA-256 of the copies:

```
1f991ac3b4b283ebf96e60ae23a00a52765dd3a2e46fa6fdda9f1aab032f7495  xterm.js
ba8e6985669488981ccf40c0cefe3aba80722cb6c92de7ad628b0bd717faf2b6  xterm.css
```

Same version as `@xterm/headless` in `tests/terminal-oracle`, so the oracle
that checks `terminal.Render` output checks the parser viewers run. Served
from the binary by `cmd/internal/broadcast` (#395); never loaded from a CDN.

Notes: 5.5.0 has no synchronized-output (DECSET 2026) support; the broadcast
sends only finished frames, each in one `term.write`.

To update: `npm pack @xterm/xterm@<v>` in a scratch directory, compare its
sha512 with `npm view @xterm/xterm@<v> dist.integrity`, extract into an empty
directory, copy the three files, and update this file and the oracle's pin
together.
