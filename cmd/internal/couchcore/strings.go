package couchcore

import "strings"

func joinArgs(args []string) string { return strings.Join(args, " ") }

func trimTrailingNewline(s string) string { return strings.TrimSpace(s) }

// sanitizeKey turns a folded worktree path into a single filesystem-safe
// filename, so a description sidecar is one flat file per tree.
func sanitizeKey(key string) string {
	return strings.NewReplacer("/", "_", ":", "_", " ", "_").Replace(strings.TrimPrefix(key, "/"))
}
