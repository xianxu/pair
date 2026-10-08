package broadcast

import "embed"

// webFS is the viewer page and its vendored renderer, served from the binary
// so the page never loads anything from another origin.
//
//go:embed web/index.html web/viewer.js web/viewer.css web/vendor/xterm/xterm.js web/vendor/xterm/xterm.css web/vendor/xterm/addon-unicode11.js web/vendor/fonts/*.woff2
var webFS embed.FS
