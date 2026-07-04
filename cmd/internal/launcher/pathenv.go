package launcher

import (
	"os"
	"path/filepath"
	"strings"
)

// prependBinToPath puts <pairHome>/bin at the front of PATH, idempotently. The
// launcher calls this once at entry so zellij and everything it execs by bare
// name — pair-wrap (the agent pane), copy_command "copy-on-select", Run
// "pair-help"/"pair-scrollback-open", and the nvim viewers' helpers — resolve
// from the resolved asset root's bin/. The retired shell bin/pair did this
// prepend; the Go launcher that replaced it (#99 M5c) dropped it, so a copied or
// Homebrew install (whose bin/ isn't already on the user's PATH) couldn't launch
// (#95). Empty PATH yields just the bin dir; an already-leading bin dir is left
// unchanged (dev shells / re-launch).
func prependBinToPath(pairHome, path string) string {
	binDir := filepath.Join(pairHome, "bin")
	sep := string(os.PathListSeparator)
	if path == "" {
		return binDir
	}
	if path == binDir || strings.HasPrefix(path, binDir+sep) {
		return path
	}
	return binDir + sep + path
}
