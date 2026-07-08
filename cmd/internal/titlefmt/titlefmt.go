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
