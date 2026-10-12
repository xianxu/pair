package broadcast

import "embed"

// webFS is the viewer page and its vendored renderer, served from the binary
// so the page never loads anything from another origin.
//
//go:embed web/index.html web/viewer.js web/notes.js web/collab.js web/viewer.css web/vendor/xterm/xterm.js web/vendor/xterm/xterm.css web/vendor/xterm/addon-unicode11.js web/vendor/fonts/*.woff2 web/vendor/fonts/*.woff
var webFS embed.FS

// CollabJS is the viewers' shared collaboration module (web/collab.js:
// pointer colors, the swatch rail, the sticky-note dialog), which the web
// cast serves too (#470).
func CollabJS() []byte {
	b, err := webFS.ReadFile("web/collab.js")
	if err != nil {
		panic(err) // embedded at build time
	}
	return b
}
