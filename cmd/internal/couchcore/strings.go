package couchcore

import "strings"

func joinArgs(args []string) string { return strings.Join(args, " ") }

func trimTrailingNewline(s string) string { return strings.TrimSpace(s) }
