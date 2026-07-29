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
