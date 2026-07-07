package entrypoint

import "strings"

// nonBusybox names have their own entrypoint semantics and must never be
// treated as a helper alias: `pair` is the launcher, `pair-go` the launch
// handoff, `pair-dev`/`pair-help`/`pair-notify` are shell shims, and
// `pair-scribe` folds to `pair scribe` (its ~/.zshrc caller is updated, not
// symlinked).
var nonBusybox = map[string]bool{
	"pair":        true,
	"pair-go":     true,
	"pair-dev":    true,
	"pair-scribe": true,
	"pair-help":   true,
	"pair-notify": true,
}

// busyboxSubcommand maps an invoked program's base name to the flat pair
// subcommand it should run when pair is invoked under a helper's name (busybox
// style, via a symlink). It strips the `pair-` prefix and validates against
// `valid` (dispatcher.DispatchNames()), so an arbitrary name on PATH never
// resolves. Only flat subcommands are reachable this way — the single surviving
// need is the external `pair-slug` Stop-hook symlink; the nested families
// (review/scrollback/changelog/clip) are reached as `pair <group> <leaf>` by
// rewritten callers, never by a busybox name.
func busyboxSubcommand(base string, valid []string) (string, bool) {
	if nonBusybox[base] {
		return "", false
	}
	sub := strings.TrimPrefix(base, "pair-")
	for _, v := range valid {
		if v == sub {
			return sub, true
		}
	}
	return "", false
}
