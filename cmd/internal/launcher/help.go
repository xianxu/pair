package launcher

// UsageText is `pair --help` / `pair help` — a CONCISE synopsis of the launch
// verbs (#99 M5c, replacing the retired bin/pair-shell's help block). The full
// in-session keybindings / behavior notes live on Alt+h (bin/pair-help), not here,
// so this stays a short usage rather than a transcription that would drift.
func UsageText() string {
	return `pair — Neovim-backed input field for any TUI coding agent.

USAGE
  pair                          claude in the default session
  pair <agent>                  e.g. pair codex / pair agy
  pair resume <tag>             attach pair-<tag> if it exists, else create it
                                (agent inferred from saved state)
  pair continue [slug] [agent]  resume from a continuation doc; bare lists them
  pair [<agent>] -- <args>      forward args to the agent on create
  pair list | ls                list pair-* sessions and their attach state
  pair rename <old> <new>       rename every tag-scoped file from <old> to <new>
  pair -h | --help              this message

Use ` + "`--`" + ` to separate pair's args from the agent's. When creating a
session you're prompted for a name; ` + "`resume <tag>`" + ` skips the prompt.
In-session keybindings are on Alt+h.
`
}
