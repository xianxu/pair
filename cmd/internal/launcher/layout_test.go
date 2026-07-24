package launcher

import (
	"strings"
	"testing"
)

func TestParseLayoutMode(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want LayoutMode
		ok   bool
	}{
		{"layout2", Layout2, true},
		{"layout3\n", Layout3, true},
		{"", "", false},
		{"small", "", false},
	} {
		got, ok := ParseLayoutMode(tc.raw)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("ParseLayoutMode(%q) = (%q, %v), want (%q, %v)", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestResolveLayout(t *testing.T) {
	for _, tc := range []struct {
		name     string
		request  LayoutRequest
		recorded LayoutMode
		valid    bool
		want     LayoutResolution
	}{
		{
			name: "unrecorded defaults to layout2",
			want: LayoutResolution{Mode: Layout2},
		},
		{
			name:     "implicit reuses layout3",
			recorded: Layout3,
			valid:    true,
			want:     LayoutResolution{Mode: Layout3},
		},
		{
			name:     "explicit layout2 conflicts with recorded layout3",
			request:  LayoutRequest{Mode: Layout2, Explicit: true},
			recorded: Layout3,
			valid:    true,
			want:     LayoutResolution{Mode: Layout2, Conflict: true},
		},
		{
			name:     "explicit layout3 conflicts with recorded layout2",
			request:  LayoutRequest{Mode: Layout3, Explicit: true},
			recorded: Layout2,
			valid:    true,
			want:     LayoutResolution{Mode: Layout3, Conflict: true},
		},
		{
			name:     "same explicit mode is not a conflict",
			request:  LayoutRequest{Mode: Layout3, Explicit: true},
			recorded: Layout3,
			valid:    true,
			want:     LayoutResolution{Mode: Layout3},
		},
		{
			name:    "invalid record defaults to layout2",
			request: LayoutRequest{},
			want:    LayoutResolution{Mode: Layout2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveLayout(tc.request, tc.recorded, tc.valid); got != tc.want {
				t.Fatalf("ResolveLayout() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestLayoutAssetBasename(t *testing.T) {
	if got := LayoutAssetBasename(Layout2); got != "main-2.kdl" {
		t.Fatalf("layout2 asset = %q", got)
	}
	if got := LayoutAssetBasename(Layout3); got != "main-3.kdl" {
		t.Fatalf("layout3 asset = %q", got)
	}
}

func TestUsageTextDocumentsSelectableLayouts(t *testing.T) {
	usage := UsageText()
	for _, want := range []string{"--layout2", "--layout3", "defaults to layout2", "recorded layout", "before `--`"} {
		if !strings.Contains(usage, want) {
			t.Fatalf("UsageText missing %q:\n%s", want, usage)
		}
	}
}
