package launcher

import (
	"os"
	"path/filepath"
	"strings"
)

// prependBinToPath puts <pairHome>/bin AND the running executable's dir at the
// front of PATH (exeDir first), idempotently. The launcher calls this once at
// entry so zellij and everything it execs by bare name resolves the right
// binaries:
//   - <pairHome>/bin holds the shell shims (pair-help, pair-notify) and, in the
//     dev/source layout, `pair` itself;
//   - exeDir (the running executable's dir) is where the installed `pair` lives
//     in the copied/Homebrew layout (pair is never in its own $PAIR_HOME/bin
//     runtime bundle, #104 M3), so this is what makes `pair <sub>` resolve there —
//     every former helper is now reached as a `pair` subcommand.
//
// The retired shell bin/pair did the bin prepend; the Go launcher that replaced
// it (#99 M5c) dropped it, so a copied/Homebrew install (whose bin/ isn't on the
// user's PATH) couldn't launch (#95). It is fully idempotent (re-launch / in-
// session restart re-runs this): the two front dirs are deduped against each
// other and against the existing PATH entries, so PATH never grows on restart.
func prependBinToPath(pairHome, exeDir, path string) string {
	binDir := filepath.Join(pairHome, "bin")
	sep := string(os.PathListSeparator)

	// front: exeDir then binDir, empties and dups dropped.
	var front []string
	for _, d := range []string{exeDir, binDir} {
		if d != "" && !containsStr(front, d) {
			front = append(front, d)
		}
	}
	// out: front dirs, then existing PATH entries minus the fronted ones (dedup)
	// and empties.
	out := append([]string(nil), front...)
	if path != "" {
		for _, p := range strings.Split(path, sep) {
			if p != "" && !containsStr(out, p) {
				out = append(out, p)
			}
		}
	}
	return strings.Join(out, sep)
}

func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
