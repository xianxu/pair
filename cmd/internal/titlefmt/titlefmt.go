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
