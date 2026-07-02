package launcher

import "testing"

func TestFormatAge(t *testing.T) {
	const day = 86400
	cases := []struct {
		now, then int64
		want      string
	}{
		{1000 * day, 1000 * day, "today"},
		{1000 * day, 1000*day - 3600, "today"}, // <1 day
		{1000 * day, 999 * day, "yesterday"},   // exactly 1 day
		{1000 * day, 995 * day, "5d ago"},
	}
	for _, c := range cases {
		if got := FormatAge(c.now, c.then); got != c.want {
			t.Errorf("FormatAge(%d,%d) = %q, want %q", c.now, c.then, got, c.want)
		}
	}
}

func TestAgeColorBuckets(t *testing.T) {
	cases := []struct {
		days int
		want string
	}{
		{0, "\033[38;5;250m"},
		{1, "\033[38;5;245m"},
		{3, "\033[38;5;242m"},
		{6, "\033[38;5;240m"},
		{30, "\033[38;5;238m"},
	}
	for _, c := range cases {
		if got := AgeColor(c.days); got != c.want {
			t.Errorf("AgeColor(%d) = %q, want %q", c.days, got, c.want)
		}
	}
}

// TildeAbbrev must only abbreviate on a real path boundary — a sibling like
// /home-other must NOT become ~-other.
func TestTildeAbbrev(t *testing.T) {
	cases := []struct {
		cwd, home, want string
	}{
		{"/home/x", "/home/x", "~"},
		{"/home/x/repo", "/home/x", "~/repo"},
		{"/home/x-other", "/home/x", "/home/x-other"}, // sibling not mangled
		{"/tmp/z", "/home/x", "/tmp/z"},
		{"/tmp/z", "", "/tmp/z"}, // no home → unchanged
	}
	for _, c := range cases {
		if got := TildeAbbrev(c.cwd, c.home); got != c.want {
			t.Errorf("TildeAbbrev(%q,%q) = %q, want %q", c.cwd, c.home, got, c.want)
		}
	}
}

func TestPaneTitle(t *testing.T) {
	if got := PaneTitle("claude", "/home/x/repo", "/home/x"); got != "claude [~/repo]" {
		t.Errorf("PaneTitle = %q", got)
	}
}

func TestEmojiTitle(t *testing.T) {
	if got := EmojiTitle("pair-brain-book"); got != "♋-🧠-📗" {
		t.Errorf("EmojiTitle = %q", got)
	}
	if got := EmojiTitle("plain"); got != "plain" {
		t.Errorf("EmojiTitle passthrough = %q", got)
	}
}
