package launcher

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// The `pair rename <old> <new>` offline subcommand + the inside-session rename
// re-entry (#99 M5b, ported from bin/pair-shell 307-546). Moves every tag-scoped
// sidecar from <old> to <new>. Enumeration is EXACT-name (never a glob — `rename
// brain new` must not touch `*-brain-2-*` files, shell 315-319). The plan build is
// pure over an injected `exists` predicate; only the mv + journal are effects.

// renameAgents is the hardcoded agent set whose per-(tag,agent) sidecars rename
// carries (shell 408). A new agent must be added here (and nowhere else — the zip
// design below needs it in one enumerator only).
var renameAgents = []string{"claude", "codex", "agy"}

// renamePathsFor enumerates every candidate sidecar path for a tag, in a stable
// order (shell rename_paths_for, 396-417). The order is identical for any tag, so
// zip(renamePathsFor(old), renamePathsFor(new)) yields the (src,dst) pairing
// directly — no base-name substitution needed (ARCH-PURE, drops shell 445-486).
func renamePathsFor(tag, dataDir string) []string {
	var out []string
	for _, fam := range []string{
		"outer-tty", "pair-wrap-pid", "title-pid",
		"agent", "agent-pid", "agent-output", "agent-picks",
		"layout-mode", "queue", "quote", "image-capture",
	} {
		out = append(out, filepath.Join(dataDir, fam+"-"+tag))
	}
	out = append(out,
		filepath.Join(dataDir, "image-capture-"+tag+".done"),
		filepath.Join(dataDir, "draft-"+tag+".md"),
		filepath.Join(dataDir, "log-"+tag+".md"),
		filepath.Join(dataDir, "nvim-pid-"+tag+"-draft"),
		filepath.Join(dataDir, "nvim-pid-"+tag+"-scrollback"),
	)
	for _, a := range renameAgents {
		out = append(out,
			filepath.Join(dataDir, "config-"+tag+"-"+a+".json"),
			filepath.Join(dataDir, "pane-"+tag+"-"+a+".json"),
			filepath.Join(dataDir, "scrollback-"+tag+"-"+a+".ansi"),
			filepath.Join(dataDir, "scrollback-"+tag+"-"+a+".raw"),
			filepath.Join(dataDir, "scrollback-"+tag+"-"+a+".viewport"),
			filepath.Join(dataDir, "scrollback-"+tag+"-"+a+".events.jsonl"),
			filepath.Join(dataDir, "draft-"+tag+"-"+a+".md"),
		)
	}
	return out
}

// renamePair is one src→dst move in the rename plan.
type renamePair struct{ src, dst string }

// validateRenameTags normalizes both tags and applies rename's own gates: charset
// (NormalizeTag), ≤256 length (shell 359-364), and old!=new (shell 365-368).
func validateRenameTags(oldRaw, newRaw string) (old, new string, err error) {
	if old, err = NormalizeTag(oldRaw); err != nil {
		return "", "", fmt.Errorf("invalid tag: %w", err)
	}
	if new, err = NormalizeTag(newRaw); err != nil {
		return "", "", fmt.Errorf("invalid tag: %w", err)
	}
	for _, t := range []string{old, new} {
		if len(t) > 256 {
			return "", "", fmt.Errorf("tag '%s' is too long (max 256)", t)
		}
	}
	if old == new {
		return "", "", fmt.Errorf("old and new tag are the same ('%s')", old)
	}
	return old, new, nil
}

// renamePlan builds the (src,dst) move list for old→new, pure over the injected
// `exists` predicate (shell 419-501). Refuses if the new tag is occupied (any of
// its sidecars already exist — a broader guard that subsumes the shell's per-pair
// dst-exists check) or if no source files exist for old.
func renamePlan(old, new, dataDir string, exists func(string) bool) ([]renamePair, error) {
	newPaths := renamePathsFor(new, dataDir)
	for _, p := range newPaths {
		if exists(p) {
			return nil, fmt.Errorf("tag '%s' is occupied — '%s' exists", new, p)
		}
	}
	oldPaths := renamePathsFor(old, dataDir)
	var pairs []renamePair
	for i, src := range oldPaths {
		if exists(src) {
			pairs = append(pairs, renamePair{src: src, dst: newPaths[i]})
		}
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no files found for tag '%s' in %s", old, dataDir)
	}
	return pairs, nil
}

// renameJournal serializes the plan as `src\tdst` lines — written before the mv
// loop so a crash leaves a forensic breadcrumb + drives the reverse-rollback
// (shell 513-514/535-539).
func renameJournal(pairs []renamePair) string {
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(p.src)
		b.WriteByte('\t')
		b.WriteString(p.dst)
		b.WriteByte('\n')
	}
	return b.String()
}

// sessionTracked reports whether a session named `name` exists in ANY state
// (attached / detached / EXITED) — rename's offline gate blocks on a resurrectable
// row too, so it is membership over Sessions(), NOT SessionBlocksReuse (shell
// 380-390 + the resurrectable-contract comment 821-823).
func sessionTracked(sessions []Session, name string) bool {
	for _, s := range sessions {
		if s.Name == name {
			return true
		}
	}
	return false
}

func sessionTrackedForTag(sessions []Session, index SessionNameIndex, scopeKey, tag string) bool {
	for _, entry := range index.Entries {
		if entry.Tag != tag {
			continue
		}
		if scopeKey != "" && entry.ScopeKey != scopeKey {
			continue
		}
		if sessionTracked(sessions, entry.SessionName) {
			return true
		}
	}
	return sessionTracked(sessions, "pair-"+tag)
}

// runRename drives `pair rename [--restart-check] <old> <new>` (shell 307-546):
// validate the tags, gate on tracked sessions, build the pure move plan, then
// (unless --restart-check) journal + mv with reverse-rollback on failure. Returns
// the exit code; the offline subcommand never launches zellij.
func runRename(rt Runtime, args LaunchArgs, dataDir string, stdout, stderr io.Writer) int {
	return runRenameScoped(rt, args, dataDir, "", stdout, stderr)
}

func runRenameScoped(rt Runtime, args LaunchArgs, dataDir, scopeKey string, stdout, stderr io.Writer) int {
	old, newTag, err := validateRenameTags(args.RenameOld, args.RenameNew)
	if err != nil {
		fmt.Fprintf(stderr, "pair rename: %v\n", err)
		return 1
	}
	if _, ok := rt.FileSize(dataDir); !ok {
		fmt.Fprintf(stderr, "pair rename: data dir not found: %s\n", dataDir)
		return 1
	}

	// Session gate (shell 378-391): empty on zellij-absent/query-error, matching
	// the shell's `|| true` (gate skipped). --restart-check skips the live-old
	// gate (the real mv runs post-kill from the restart re-entry).
	sessions, _ := rt.Sessions()
	index, _ := rt.ReadSessionNameIndex()
	if !args.RenameCheckOnly && sessionTrackedForTag(sessions, index, scopeKey, old) {
		fmt.Fprintf(stderr, "pair rename: session 'pair-%s' is still tracked by zellij.\n", old)
		fmt.Fprintf(stderr, "             Quit it first (Alt+x), or use the in-session rename.\n")
		return 1
	}
	if sessionTrackedForTag(sessions, index, scopeKey, newTag) {
		fmt.Fprintf(stderr, "pair rename: session 'pair-%s' already exists in zellij.\n", newTag)
		return 1
	}

	pairs, err := renamePlan(old, newTag, dataDir, func(p string) bool { _, ok := rt.FileSize(p); return ok })
	if err != nil {
		fmt.Fprintf(stderr, "pair rename: %v\n", err)
		return 1
	}

	if args.RenameCheckOnly {
		fmt.Fprintf(stdout, "pair rename: ok (%d file(s) would move from '%s' to '%s')\n", len(pairs), old, newTag)
		return 0
	}

	journal := filepath.Join(dataDir, ".rename-"+old+"-to-"+newTag+".journal")
	_ = rt.WriteAtomic(journal, renameJournal(pairs))
	fmt.Fprintf(stdout, "pair rename: %d file(s) %s → %s\n", len(pairs), old, newTag)
	done := 0
	for _, p := range pairs {
		if err := rt.Rename(p.src, p.dst); err != nil {
			fmt.Fprintf(stderr, "pair rename: mv failed: %s → %s: %v\n", p.src, p.dst, err)
			fmt.Fprintf(stderr, "pair rename: rolling back %d completed rename(s)...\n", done)
			for j := done - 1; j >= 0; j-- {
				_ = rt.Rename(pairs[j].dst, pairs[j].src) // reverse the completed moves
			}
			return 1 // keep the journal — diagnostic (shell 540)
		}
		done++
	}
	rt.Remove(journal)
	fmt.Fprintf(stdout, "pair rename: ok\n")
	return 0
}
