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
