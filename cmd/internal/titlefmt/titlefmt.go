package titlefmt

import "strings"

var emojiWords = map[string]string{
	"brain": "🧠",
	"book":  "📗",
	"pair":  "♋",
}

// EmojiTitle applies the personal cmux display convention to compound session
// titles while preserving a literal single-word repo/workspace title.
func EmojiTitle(title string) string {
	if !strings.Contains(title, "-") {
		return title
	}
	parts := strings.Split(title, "-")
	for i, part := range parts {
		if emoji, ok := emojiWords[part]; ok {
			parts[i] = emoji
		}
	}
	return strings.Join(parts, "-")
}

// TildeAbbrev shortens an absolute path by replacing $HOME with "~".
//
// This is THE cwd-shortening rule for every title surface (#129). Before it,
// `launcher.TildeAbbrev` and `titlepoller.abbrevCwd` were byte-identical copies;
// they were consolidated here *before* a third consumer was added, because two
// sites independently implementing one rule is exactly how the SGR-terminator
// divergence in #127 got made. `titlefmt` is the right home: it has no IO and
// was already imported by both packages.
func TildeAbbrev(path, home string) string {
	if home == "" {
		return path // defensive: the shell assumes $HOME is always set
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	return path
}

// PaneTitlePrefix builds the leading "<cwd> [<tag>] · " that every Pair-owned
// pane title carries, so the terminal tab title identifies the workbench rather
// than just the pane's role (#129).
//
// zellij composes the tab title as "<session name> | <focused pane title>", and
// the session-name half is a socket filename we cannot shape. This prefix owns
// the half we can. If zellij ever gains an option to drop its prefix
// (zellij-org/zellij#1495), this becomes the whole title unchanged — which is
// why the tag is included even though it currently duplicates the session name.
//
// Returns "" when there is nothing to say, so a caller never emits a bare
// separator with no content in front of it.
func PaneTitlePrefix(cwdDisplay, tag string) string {
	var b strings.Builder
	if cwdDisplay != "" {
		b.WriteString(cwdDisplay)
	}
	if tag != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString("[")
		b.WriteString(tag)
		b.WriteString("]")
	}
	if b.Len() == 0 {
		return ""
	}
	b.WriteString(" · ")
	return b.String()
}
