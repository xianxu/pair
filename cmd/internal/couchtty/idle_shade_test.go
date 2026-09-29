package couchtty

import (
	"image/color"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestIdleLevelForBoundaries(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		last  time.Time
		known bool
		want  IdleLevel
	}{
		{"unknown", time.Time{}, false, IdleFresh},
		{"known but zero", time.Time{}, true, IdleFresh},
		{"future", now.Add(time.Minute), true, IdleFresh},
		{"just now", now, true, IdleFresh},
		{"23h59m59s", now.Add(-24*time.Hour + time.Second), true, IdleFresh},
		{"24h", now.Add(-24 * time.Hour), true, IdleDay},
		{"71h59m", now.Add(-72*time.Hour + time.Minute), true, IdleDay},
		{"72h", now.Add(-72 * time.Hour), true, IdleStale},
		{"30d", now.Add(-30 * 24 * time.Hour), true, IdleStale},
	} {
		if got := IdleLevelFor(now, tc.last, tc.known); got != tc.want {
			t.Errorf("%s: level = %d, want %d", tc.name, got, tc.want)
		}
	}
}

var (
	white = color.RGBA{0xff, 0xff, 0xff, 0xff}
	black = color.RGBA{0, 0, 0, 0xff}
)

func TestBlendMovesTowardTheBackground(t *testing.T) {
	if got := blend(white, black, 0); got != white {
		t.Fatalf("blend at 0 = %v, want the foreground", got)
	}
	if got := blend(white, black, 0.40); got != (color.RGBA{153, 153, 153, 0xff}) {
		t.Fatalf("blend(white, black, 0.40) = %v, want #999999", got)
	}
	if got := blend(amberRGB, white, 0.65); got != (color.RGBA{255, 241, 166, 0xff}) {
		t.Fatalf("blend(amber, white, 0.65) = %v, want #fff1a6", got)
	}
}

func TestQuantize256(t *testing.T) {
	for _, tc := range []struct {
		in   color.RGBA
		want int
	}{
		{color.RGBA{0xff, 0, 0, 0xff}, 196},
		{color.RGBA{0x80, 0x80, 0x80, 0xff}, 244},
		{color.RGBA{0, 0, 0, 0xff}, 16},
		{color.RGBA{0xff, 0xff, 0xff, 0xff}, 231},
		{color.RGBA{0, 0xff, 0, 0xff}, 46},
		{color.RGBA{0x5f, 0x87, 0xaf, 0xff}, 67},
	} {
		if got := quantize256(tc.in); got != tc.want {
			t.Errorf("quantize256(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

var trueColorSGR = regexp.MustCompile(`^\x1b\[38;2;(\d+);(\d+);(\d+)m$`)

func channelSum(t *testing.T, sgr string) int {
	t.Helper()
	m := trueColorSGR.FindStringSubmatch(sgr)
	if m == nil {
		t.Fatalf("not a truecolor SGR: %q", sgr)
	}
	sum := 0
	for _, part := range m[1:] {
		v, _ := strconv.Atoi(part)
		sum += v
	}
	return sum
}

func TestFadeStyleLevelZeroIsTodaysBytesInEveryPalette(t *testing.T) {
	for name, p := range map[string]Palette{
		"unknown":        {},
		"dark truecolor": {FG: white, BG: black, Known: true, TrueColor: true},
		"light 256":      {FG: black, BG: white, Known: true},
		"no color":       {FG: white, BG: black, Known: true, TrueColor: true, NoColor: true},
	} {
		if got := FadeStyle(p, IdleFresh, baseDefault); got != "" {
			t.Errorf("%s: level-0 label = %q, want no SGR", name, got)
		}
		if got := FadeStyle(p, IdleFresh, baseAmber); got != attentionSGR {
			t.Errorf("%s: level-0 amber = %q, want attentionSGR", name, got)
		}
	}
}

func TestFadeStyleRecedesTowardTheBackgroundOnBothThemes(t *testing.T) {
	dark := Palette{FG: white, BG: black, Known: true, TrueColor: true}
	light := Palette{FG: black, BG: white, Known: true, TrueColor: true}
	if got := FadeStyle(dark, IdleDay, baseDefault); got != "\x1b[38;2;153;153;153m" {
		t.Fatalf("dark level-1 label = %q", got)
	}
	for _, base := range []styleBase{baseDefault, baseAmber} {
		// Dark background: fading darkens. Light background: fading lightens.
		// Stated as ordering, independent of the weight table.
		if d1, d2 := channelSum(t, FadeStyle(dark, IdleDay, base)), channelSum(t, FadeStyle(dark, IdleStale, base)); !(d2 < d1) {
			t.Errorf("base %d dark: stale sum %d must be below day sum %d", base, d2, d1)
		}
		if l1, l2 := channelSum(t, FadeStyle(light, IdleDay, base)), channelSum(t, FadeStyle(light, IdleStale, base)); !(l2 > l1) {
			t.Errorf("base %d light: stale sum %d must exceed day sum %d", base, l2, l1)
		}
	}
	if l1 := channelSum(t, FadeStyle(light, IdleDay, baseDefault)); l1 == 0 {
		t.Fatal("light level-1 label did not move off the black foreground")
	}
}

func TestFadeStyleFallbacks(t *testing.T) {
	quantized := Palette{FG: white, BG: black, Known: true}
	if got, want := FadeStyle(quantized, IdleDay, baseDefault), "\x1b[38;5;"+strconv.Itoa(quantize256(blend(white, black, idleBlend[IdleDay])))+"m"; got != want {
		t.Fatalf("256-color level-1 label = %q, want %q", got, want)
	}
	unknown := Palette{TrueColor: true}
	for _, level := range []IdleLevel{IdleDay, IdleStale} {
		if got := FadeStyle(unknown, level, baseDefault); got != "\x1b[90m" {
			t.Errorf("unknown palette level %d label = %q, want SGR 90", level, got)
		}
		if got := FadeStyle(unknown, level, baseAmber); got != attentionSGR {
			t.Errorf("unknown palette level %d amber = %q, want attentionSGR", level, got)
		}
	}
	noColor := Palette{FG: white, BG: black, Known: true, TrueColor: true, NoColor: true}
	for _, level := range []IdleLevel{IdleDay, IdleStale} {
		if got := FadeStyle(noColor, level, baseDefault); got != "" {
			t.Errorf("NO_COLOR level %d label = %q, want today's bytes (none)", level, got)
		}
		if got := FadeStyle(noColor, level, baseAmber); got != attentionSGR {
			t.Errorf("NO_COLOR level %d amber = %q, want today's attentionSGR", level, got)
		}
	}
}
