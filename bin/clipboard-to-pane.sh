#!/usr/bin/env bash
# Pull whatever is on the OS clipboard, reflow paragraph wraps,
# prefix every line with "> " (markdown quote), and inject into
# the pane below current focus (the nvim draft pane in pair's layout).
# A trailing blank line is added so the cursor lands ready for a reply.
#
# Triggered by Alt+n via zellij keybind. Focus is left in nvim — the user
# pulled a quote in to react to it inline.

set -euo pipefail

# OS-aware clipboard read
if command -v pbpaste >/dev/null 2>&1; then
    clip="$(pbpaste)"
elif command -v wl-paste >/dev/null 2>&1; then
    clip="$(wl-paste --no-newline)"
elif command -v xclip >/dev/null 2>&1; then
    clip="$(xclip -selection clipboard -o)"
else
    echo "clipboard-to-pane: no clipboard tool found (need pbpaste, wl-paste, or xclip)" >&2
    exit 1
fi

[ -z "$clip" ] && exit 0

# Reflow with par if available; otherwise pass through unchanged.
# par -w99999 collapses wrap-induced line breaks while preserving
# paragraph and list structure.
if command -v par >/dev/null 2>&1; then
    quoted="$(printf '%s' "$clip" | par -w99999 2>/dev/null | sed 's/^/> /')"
else
    quoted="$(printf '%s' "$clip" | sed 's/^/> /')"
fi

zellij action move-focus down
zellij action write-chars "$quoted"$'\n\n'
